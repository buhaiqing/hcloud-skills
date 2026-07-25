"""Critic v1 — rule-based 5-dimension quality scorer.

Output schema MUST match gcl_runner.validate_critic_payload:
  - scores.{correctness, safety, idempotency, traceability, spec_compliance} in {0, 0.5, 1}
  - suggestions: list[str]
  - blocking: bool

CLI:
  critic_v1.py --generator <gen.json>            # pretty print
  critic_v1.py --generator <gen.json> --emit-critic-json  # write to critic.json
  critic_v1.py --generator <gen.json> --emit-critic-json --critic-out path/to/critic.json

Used by:
  - production GCL: gcl_runner.py run --critic-json path/to/critic.json
  - runtime_orchestrator.py: integrates via gcl_runner.structural_critic + this module

Rules (5 dimensions):
- Safety: destructive verb without --dry-run => 0; else 1
- Correctness: command present + contains "hcloud"/"go run" => 1; else 0.5
- Idempotency: read verbs (list/show/describe/get) => 1; write verbs (create/delete/update) => 0.5
- Traceability: result_excerpt non-empty => 1; else 0
- SecOps/Spec compliance: no credential leak + uses hcloud SDK pattern => 1

FinOps cost estimation is surfaced as a *suggestion* (not a score dim) since the rubric
doesn't include a FinOps dimension — operational teams should track cost separately.
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parent))
from gcl_runner import has_credential_leak  # noqa: E402

# Verb classification for idempotency scoring.
READ_VERBS = {"list", "show", "describe", "get", "fetch", "listquotas", "showquota"}
WRITE_VERBS = {"create", "delete", "terminate", "remove", "update", "modify", "attach", "detach", "reboot", "restart", "stop", "start"}
DESTRUCTIVE_VERBS = {"delete", "terminate", "remove", "drop", "purge"}

# Cost estimation: rough USD/hour per product+operation class. Real cost would come from BSS.
COST_TABLE: dict[tuple[str, str], float] = {
    ("ecs", "create-server"): 0.05,
    ("ecs", "delete-server"): 0.0,
    ("rds", "create-instance"): 0.12,
    ("rds", "delete-instance"): 0.0,
    ("rds", "enlarge-volume"): 0.01,
    ("elb", "create-loadbalancer"): 0.025,
    ("elb", "delete-loadbalancer"): 0.0,
    ("vpc", "create-vpc"): 0.001,
    ("cce", "create-cluster"): 0.5,
    ("cce", "create-node"): 0.08,
}

DESTRUCTIVE_RE = re.compile(r"\b(" + "|".join(sorted(DESTRUCTIVE_VERBS)) + r")\b", re.IGNORECASE)
DRY_RUN_RE = re.compile(r"--dry-run|-dry-run", re.IGNORECASE)
HCLOUD_RE = re.compile(r"\bhcloud\b", re.IGNORECASE)
GO_RUN_RE = re.compile(r"\bgo run\b", re.IGNORECASE)


def _extract_skill_short(command: str) -> str:
    """Best-effort extract skill short name from `hcloud <skill> ...` command."""
    m = re.search(r"\bhcloud\s+([a-z0-9_-]+)", command, re.IGNORECASE)
    return m.group(1).lower() if m else ""


def _extract_operation(command: str) -> str:
    """Best-effort extract operation verb from command tokens."""
    m = re.search(r"\bhcloud\s+[a-z0-9_-]+\s+([a-zA-Z][\w-]*)", command)
    return m.group(1).lower() if m else ""


def score_safety(command: str) -> tuple[float, list[str]]:
    """Safety = 0 if destructive verb without --dry-run, else 1."""
    suggestions: list[str] = []
    if DESTRUCTIVE_RE.search(command) and not DRY_RUN_RE.search(command):
        suggestions.append("Destructive command without --dry-run; require explicit confirmation or dry-run gate")
        return 0.0, suggestions
    return 1.0, suggestions


def score_correctness(command: str) -> tuple[float, list[str]]:
    """Correctness = 1 if well-formed hcloud/go-run command, else 0.5."""
    suggestions: list[str] = []
    if not command.strip():
        return 0.0, ["Empty command"]
    if HCLOUD_RE.search(command) or GO_RUN_RE.search(command):
        return 1.0, suggestions
    suggestions.append("Command does not match hcloud or `go run` pattern; verify spelling")
    return 0.5, suggestions


def score_idempotency(command: str) -> float:
    """Idempotency = 1 for read verbs, 0.5 for write verbs, 0.5 otherwise."""
    op = _extract_operation(command)
    if op in READ_VERBS:
        return 1.0
    if op in WRITE_VERBS:
        return 0.5
    return 0.5


def score_traceability(generator: dict[str, Any]) -> tuple[float, list[str]]:
    """Traceability = 1 if excerpt and command are both non-empty."""
    suggestions: list[str] = []
    excerpt = generator.get("result_excerpt", "")
    command = generator.get("command", "")
    if command and excerpt:
        return 1.0, suggestions
    suggestions.append("Missing result_excerpt or command; capture stdout/stderr in trace")
    return 0.0, suggestions


def score_spec_compliance(command: str, generator: dict[str, Any]) -> tuple[float, list[str]]:
    """Spec compliance = 1 if exit_code==0 + (hcloud | go run) + no credential leak, else 0.5."""
    suggestions: list[str] = []
    exit_code = generator.get("exit_code", -1)
    excerpt = generator.get("result_excerpt", "")
    leak = has_credential_leak(excerpt) or has_credential_leak(command)
    if leak:
        suggestions.append("Credential leak detected; mask HW_SECRET_ACCESS_KEY and re-run")
        return 0.0, suggestions
    if exit_code == 0 and (HCLOUD_RE.search(command) or GO_RUN_RE.search(command)):
        return 1.0, suggestions
    if exit_code != 0:
        suggestions.append(f"Exit code {exit_code}; verify endpoint/credentials")
    suggestions.append("Spec mismatch: ensure command matches skill's CLI/SDK pattern")
    return 0.5, suggestions


def estimate_cost(command: str) -> tuple[float, str]:
    """Estimate FinOps cost (USD) per call. Returns (cost, skill_short)."""
    skill_short = _extract_skill_short(command)
    op = _extract_operation(command)
    cost = COST_TABLE.get((skill_short, op), 0.001)  # default: negligible
    return cost, skill_short


def score(generator: dict[str, Any]) -> dict[str, Any]:
    """Run all 5 rules and return a critic-compatible payload."""
    command = generator.get("command", "")

    safety, s_safety = score_safety(command)
    correctness, s_correctness = score_correctness(command)
    idempotency = score_idempotency(command)
    traceability, s_traceability = score_traceability(generator)
    spec_compliance, s_spec = score_spec_compliance(command, generator)

    scores = {
        "correctness": correctness,
        "safety": safety,
        "idempotency": idempotency,
        "traceability": traceability,
        "spec_compliance": spec_compliance,
    }

    blocking = safety == 0.0 or correctness == 0.0
    suggestions = (s_safety + s_correctness + s_traceability + s_spec)[:5]

    # FinOps / SecOps surfaced as suggestions (not score dims)
    cost, skill_short = estimate_cost(command)
    if cost >= 0.1:
        suggestions.append(f"FinOps: estimated cost ${cost:.3f}/call (skill={skill_short}); consider cheaper alternative or batch")
    if safety == 1.0 and DESTRUCTIVE_RE.search(command):
        suggestions.append("SecOps: destructive command gated behind --dry-run; ensure human approval before live run")

    return {
        "scores": scores,
        "suggestions": suggestions,
        "blocking": blocking,
        "_mode": "critic-v1",
        "finops_estimate_usd": cost,
    }


def cmd_score(args: argparse.Namespace) -> int:
    payload = json.loads(Path(args.generator).read_text(encoding="utf-8")) if Path(args.generator).exists() else json.loads(args.generator)
    critic = score(payload)
    if args.emit_critic_json:
        out = Path(args.critic_out)
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(json.dumps(critic, indent=2, ensure_ascii=False), encoding="utf-8")
    print(json.dumps(critic, indent=2, ensure_ascii=False))
    return 0


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description="Critic v1 — rule-based 5-dimension quality scorer.")
    p.add_argument("--generator", required=True, help="Path to generator trace JSON, or inline JSON string")
    p.add_argument("--emit-critic-json", action="store_true", help="Write critic output to file")
    p.add_argument("--critic-out", default="critic.json", help="Output path when --emit-critic-json is set")
    p.set_defaults(func=cmd_score)
    return p


def main() -> int:
    args = build_parser().parse_args()
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())