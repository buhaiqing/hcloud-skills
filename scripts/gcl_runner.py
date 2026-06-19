#!/usr/bin/env python3
"""GCL Orchestrator — Generator execution loop with external Critic injection.

Implements the Orchestrator role from the Huawei Cloud GCL spec. Generator runs
`hcloud`/shell commands; Critic scores MUST come from an isolated context via
`--critic-json` or stdin. This script never self-scores as Critic in production mode.

Usage:
  python3 scripts/gcl_runner.py run \
    --skill huaweicloud-ecs-ops \
    --request "List ECS instances read-only" \
    --command 'hcloud ecs list-servers --region cn-north-4' \
    [--max-iter 2] \
    [--critic-json path/to/critic.json]

  # Rule-based structural audit only (CI/local smoke; NOT production quality pass):
  python3 scripts/gcl_runner.py run ... --structural-critic-only

Trace output: `audit-results/gcl-trace-YYYYMMDD-HHMMSS.json`
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

SKILL_MAX_ITER: dict[str, int] = {
    "huaweicloud-ecs-ops": 2,
    "huaweicloud-iam-ops": 2,
    "huaweicloud-rds-ops": 2,
    "huaweicloud-gaussdb-ops": 2,
    "huaweicloud-dcs-ops": 2,
    "huaweicloud-dms-ops": 2,
    "huaweicloud-css-ops": 2,
    "huaweicloud-cce-ops": 2,
    "huaweicloud-cbr-ops": 2,
    "huaweicloud-vpc-ops": 2,
    "huaweicloud-obs-ops": 2,
    "huaweicloud-swr-ops": 2,
    "huaweicloud-functiongraph-ops": 2,
    "huaweicloud-waf-ops": 2,
    "huaweicloud-hss-ops": 2,
    "huaweicloud-elb-ops": 3,
    "huaweicloud-ces-ops": 3,
    "huaweicloud-lts-ops": 3,
    "huaweicloud-cts-ops": 3,
    "huaweicloud-billing-ops": 5,
    "huaweicloud-skill-generator": 3,
}

RUBRIC_THRESHOLDS: dict[str, float] = {
    "correctness": 0.5,
    "safety": 1.0,
    "idempotency": 0.5,
    "traceability": 0.5,
    "spec_compliance": 0.5,
}

SECRET_PATTERNS = [
    re.compile(r"HW_SECRET_ACCESS_KEY\s*=\s*[^\s\"']+", re.I),
    re.compile(r"SECRET_ACCESS_KEY\s*=\s*[^\s\"']+", re.I),
    re.compile(r"SecretAccessKey\s*[=:]\s*[^\s\"']+", re.I),
    re.compile(r"SK\s*[=:]\s*[A-Za-z0-9/+]{20,}", re.I),
]


def mask_secrets(text: str) -> str:
    out = text
    replacements = [
        (r"(HW_SECRET_ACCESS_KEY\s*=\s*)([^\s\"']+)", r"\1<masked>"),
        (r"(SECRET_ACCESS_KEY\s*=\s*)([^\s\"']+)", r"\1<masked>"),
        (r"(SecretAccessKey\s*[=:]\s*)([^\s\"']+)", r"\1<masked>"),
        (r"(SK\s*[=:]\s*)([A-Za-z0-9/+]{20,})", r"\1<masked>"),
    ]
    for pattern, replacement in replacements:
        out = re.sub(pattern, replacement, out, flags=re.I)
    return out


def has_credential_leak(text: str) -> bool:
    if "<masked>" in text:
        return False
    return any(pattern.search(text) for pattern in SECRET_PATTERNS)


def sanitize_operation_intent(raw: str | None) -> dict[str, Any] | None:
    if not raw:
        return None
    try:
        intent = json.loads(raw)
    except json.JSONDecodeError:
        return {"summary": mask_secrets(raw)[:500]}
    return _mask_json(intent)


def _mask_json(value: Any) -> Any:
    if isinstance(value, dict):
        masked: dict[str, Any] = {}
        for key, item in value.items():
            if re.search(r"secret|password|token|credential|ak|sk", key, re.I):
                masked[key] = "<masked>"
            else:
                masked[key] = _mask_json(item)
        return masked
    if isinstance(value, list):
        return [_mask_json(item) for item in value]
    if isinstance(value, str):
        return mask_secrets(value)
    return value


def run_command(command: str, timeout: int = 120) -> dict[str, Any]:
    try:
        proc = subprocess.run(
            command,
            shell=True,
            capture_output=True,
            text=True,
            timeout=timeout,
        )
        combined = (proc.stdout or "") + (proc.stderr or "")
        masked = mask_secrets(combined)
        excerpt = masked[:2000] + ("..." if len(masked) > 2000 else "")
        return {
            "command": mask_secrets(command),
            "exit_code": proc.returncode,
            "result_excerpt": excerpt,
            "stdout_len": len(proc.stdout or ""),
            "stderr_len": len(proc.stderr or ""),
        }
    except subprocess.TimeoutExpired:
        return {
            "command": mask_secrets(command),
            "exit_code": -1,
            "result_excerpt": f"TIMEOUT after {timeout}s",
            "stdout_len": 0,
            "stderr_len": 0,
        }


def structural_critic(generator: dict[str, Any]) -> dict[str, Any]:
    scores: dict[str, float] = {}
    suggestions: list[str] = []

    exit_code = generator.get("exit_code", -1)
    excerpt = generator.get("result_excerpt", "")
    command = generator.get("command", "")

    scores["correctness"] = 1.0 if exit_code == 0 else 0.0
    if exit_code != 0:
        suggestions.append(f"Generator exit_code={exit_code}; fix command, credentials, or endpoint")

    leak = has_credential_leak(excerpt) or has_credential_leak(command)
    scores["safety"] = 0.0 if leak else 1.0
    if leak:
        suggestions.append("Credential leak in trace — mask HW_SECRET_ACCESS_KEY and re-run")

    scores["idempotency"] = 0.5
    scores["traceability"] = 1.0 if command and excerpt else 0.5
    if not excerpt:
        suggestions.append("Empty generator output — capture stdout/stderr in trace")

    scores["spec_compliance"] = 1.0 if exit_code == 0 else 0.0
    if exit_code == 0 and "hcloud" not in command and "go run" not in command.lower():
        scores["spec_compliance"] = 0.5

    return {
        "scores": scores,
        "suggestions": suggestions[:3],
        "blocking": scores["safety"] == 0.0 or scores["correctness"] == 0.0,
        "_mode": "structural-only",
    }


def load_critic(path: Path | None, stdin: bool) -> dict[str, Any] | None:
    if path:
        return json.loads(path.read_text(encoding="utf-8"))
    if stdin and not sys.stdin.isatty():
        return json.loads(sys.stdin.read())
    return None


def validate_critic_payload(critic: dict[str, Any]) -> list[str]:
    errors: list[str] = []
    scores = critic.get("scores")
    if not isinstance(scores, dict):
        return ["critic.scores must be object"]
    for dim in RUBRIC_THRESHOLDS:
        if dim not in scores:
            errors.append(f"critic.scores missing '{dim}'")
        elif scores[dim] not in (0, 0.5, 1, 0.0, 1.0):
            errors.append(f"critic.scores.{dim} must be 0, 0.5, or 1")
    if "suggestions" not in critic:
        errors.append("critic.suggestions required")
    if "blocking" not in critic:
        errors.append("critic.blocking required")
    return errors


def decide(scores: dict[str, float]) -> str:
    if scores.get("safety", 1) == 0:
        return "SAFETY_FAIL"
    for dim, threshold in RUBRIC_THRESHOLDS.items():
        if scores.get(dim, 0) < threshold:
            return "RETRY"
    return "PASS"


_FAILURE_SIGNATURES: list[tuple[str, re.Pattern[str]]] = [
    ("cli_parameter", re.compile(r"InvalidParameter|MissingParameter|APIGW\.|APIG\.", re.I)),
    ("runtime", re.compile(r"TIMEOUT|RequestLimitExceeded|InternalError|ConnectionError|Throttling", re.I)),
    ("cross_skill", re.compile(r"delegate-to|not found in target skill|cross-skill", re.I)),
    ("token_efficiency", re.compile(r"token budget|exceeds.*token|too long|truncated", re.I)),
    ("skill_generation", re.compile(r"frontmatter missing|missing rubric|broken link", re.I)),
]


def extract_failure_pattern(
    skill: str,
    command: str,
    generator: dict[str, Any],
    critic: dict[str, Any],
) -> dict[str, Any] | None:
    corpus_parts = [
        command or "",
        generator.get("result_excerpt", "") or "",
        *(critic.get("suggestions") or []),
    ]
    corpus = "\n".join(corpus_parts)
    for category, pattern in _FAILURE_SIGNATURES:
        match = pattern.search(corpus)
        if not match:
            continue
        fix = (critic.get("suggestions") or ["Investigate failure pattern and add fix"])[0]
        return {
            "category": category,
            "skill": skill,
            "command": mask_secrets(command[:200]) if command else None,
            "error": match.group(0),
            "fix": fix[:200],
            "count": 1,
            "reusable": category in {"cli_parameter", "runtime"},
        }
    return None


def persist_trace(root: Path, trace: dict[str, Any]) -> Path:
    out_dir = root / "audit-results"
    out_dir.mkdir(parents=True, exist_ok=True)
    ts = datetime.now(UTC).strftime("%Y%m%d-%H%M%S")
    path = out_dir / f"gcl-trace-{ts}.json"
    path.write_text(json.dumps(trace, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    return path


def cmd_run(args: argparse.Namespace) -> int:
    root = args.root
    max_iter = args.max_iter or SKILL_MAX_ITER.get(args.skill, 3)
    trace: dict[str, Any] = {
        "trace_schema_version": "v1",
        "skill": args.skill,
        "request": mask_secrets(args.request),
        "operation_intent": sanitize_operation_intent(args.operation_intent),
        "rubric_version": "v1",
        "masked_fields": ["request", "operation_intent", "generator.command", "generator.result_excerpt"],
        "iterations": [],
    }

    critic_feedback = ""
    command = args.command

    for iteration in range(1, max_iter + 1):
        generator = run_command(command, timeout=args.timeout)
        generator["args"] = {"iter": iteration, "critic_feedback": critic_feedback or None}

        if args.structural_critic_only:
            critic = structural_critic(generator)
        else:
            critic = load_critic(args.critic_json, args.critic_stdin)
            if critic is None:
                print(
                    "ERROR: No Critic payload. Pass --critic-json, pipe JSON to stdin, "
                    "or use --structural-critic-only for CI/local smoke tests.",
                    file=sys.stderr,
                )
                return 2
            errors = validate_critic_payload(critic)
            if errors:
                print("ERROR: Invalid critic JSON: " + "; ".join(errors), file=sys.stderr)
                return 2

        decision = decide(critic["scores"])
        trace["iterations"].append(
            {
                "iter": iteration,
                "generator": generator,
                "critic": {
                    "scores": critic["scores"],
                    "suggestions": critic.get("suggestions", []),
                    "blocking": critic.get("blocking", False),
                },
                "decision": decision,
            }
        )

        if decision == "SAFETY_FAIL":
            trace["final"] = {
                "status": "SAFETY_FAIL",
                "iter": iteration,
                "output": None,
                "failure_pattern": extract_failure_pattern(args.skill, command, generator, critic),
            }
            path = persist_trace(root, trace)
            print(f"SAFETY_FAIL — trace: {path}", file=sys.stderr)
            return 3

        if decision == "PASS":
            trace["final"] = {
                "status": "PASS",
                "iter": iteration,
                "output": generator.get("result_excerpt", ""),
                "failure_pattern": None,
            }
            path = persist_trace(root, trace)
            print(f"PASS (iter {iteration}) — trace: {path}")
            return 0

        critic_feedback = "; ".join(critic.get("suggestions", [])[:3])

    last_iteration = trace["iterations"][-1]
    trace["final"] = {
        "status": "MAX_ITER",
        "iter": max_iter,
        "output": last_iteration["generator"].get("result_excerpt", ""),
        "unresolved": [
            dim
            for dim, threshold in RUBRIC_THRESHOLDS.items()
            if last_iteration["critic"]["scores"].get(dim, 0) < threshold
        ],
        "failure_pattern": extract_failure_pattern(
            args.skill,
            command,
            last_iteration["generator"],
            last_iteration["critic"],
        ),
    }
    path = persist_trace(root, trace)
    print(f"MAX_ITER — trace: {path}", file=sys.stderr)
    return 1


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="cmd", required=True)

    run = subparsers.add_parser("run", help="Execute GCL loop")
    run.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    run.add_argument("--skill", required=True, help="Skill id, e.g. huaweicloud-ecs-ops")
    run.add_argument("--request", required=True, help="Sanitized user request stored in trace")
    run.add_argument(
        "--operation-intent",
        default=None,
        help="Sanitized operation_intent JSON; omit raw user wording and secrets",
    )
    run.add_argument("--command", required=True, help="Shell command for Generator")
    run.add_argument("--max-iter", type=int, default=None)
    run.add_argument("--timeout", type=int, default=120)
    run.add_argument("--critic-json", type=Path, default=None, help="External Critic JSON file")
    run.add_argument("--critic-stdin", action="store_true", help="Read Critic JSON from stdin")
    run.add_argument(
        "--structural-critic-only",
        action="store_true",
        help="Use rule-based structural critic (CI/local smoke only; not production mutations)",
    )
    run.set_defaults(func=cmd_run)
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
