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
import os
import platform
import re
import subprocess
import sys
import time
import uuid
from datetime import datetime, timezone

# `datetime.UTC` is a 3.11+ alias; the agent runtime is still on Python 3.10
# (see AGENTS.md §Python 3.10 Syntax Compatibility). Use `timezone.utc` and
# expose it under the name `UTC` so call sites stay short.
UTC = timezone.utc  # noqa: UP017
from pathlib import Path  # noqa: E402
from typing import Any  # noqa: E402

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


def _now_iso() -> str:
    return datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


def resolve_skill_version(root: Path, skill: str) -> str:
    """Extract metadata.version from SKILL.md YAML frontmatter (best-effort)."""
    skill_md = root / skill / "SKILL.md"
    if not skill_md.exists():
        return "unknown"
    try:
        text = skill_md.read_text(encoding="utf-8")
        # Lightweight regex parse — avoids PyYAML dependency
        m = re.search(r'^\s*version:\s*["\']?([^"\'\n]+)["\']?', text, re.M)
        return m.group(1).strip() if m else "unknown"
    except OSError:
        return "unknown"


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
    sanitized = _mask_json(intent)
    if isinstance(sanitized, dict) and "resource_scope" in sanitized:
        sanitized["resource_scope"] = _mask_resource_scope(sanitized["resource_scope"])
    _enforce_safety_class_enum(sanitized)
    return sanitized


SAFETY_CLASS_VALUES: tuple[str, ...] = ("read-only", "mutating", "destructive")


# Resource identifier patterns that MUST be masked before they land in a trace.
# Each pattern preserves the type prefix (the segment before the first ``-``)
# and replaces everything that follows with ``***``. The Critic can still see
# the *kind* of resource without seeing the live identifier.
_RESOURCE_ID_PREFIX = re.compile(r"^(?P<type>[A-Za-z][A-Za-z0-9]*?)(?P<rest>-.*)?$")
_UUID_PATTERN = re.compile(r"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")
_ARN_PATTERN = re.compile(r"^(?P<prefix>acs:[A-Za-z0-9-]+:[A-Za-z0-9-]+:[A-Za-z0-9-]+:[A-Za-z0-9-]+/)(?P<id>.+)$")
_ALREADY_MASKED = re.compile(r"[*]{3,}|<masked>")
_MASK = "***"


def mask_resource_id(value: str) -> str:
    """Return a masked representation of a single resource identifier.

    * Already-masked values pass through unchanged (idempotent).
    * ARNs (``acs:...``) get only the trailing ID replaced with ``***``.
    * UUIDs become a plain ``***``.
    * Bare single-character inputs fall back to ``***`` (no type prefix).
    * Anything else is normalized to ``<type>-***`` where ``<type>`` is the
      alphabetic prefix (the substring before the first ``-``). This is
      intentionally strict: raw identifiers never appear in a trace.
    """

    if not isinstance(value, str):
        return _MASK
    if _ALREADY_MASKED.search(value):
        return value
    arn = _ARN_PATTERN.match(value)
    if arn:
        return f"{arn.group('prefix')}{_MASK}"
    if _UUID_PATTERN.match(value):
        return _MASK
    match = _RESOURCE_ID_PREFIX.match(value)
    if not match or not match.group("type"):
        return _MASK
    return f"{match.group('type')}-{_MASK}"


def _enforce_safety_class_enum(intent: dict[str, Any]) -> None:
    """Fail fast when the operation_intent carries a non-canonical safety_class.

    The GCL spec defines ``safety_class`` as an enum (``read-only``,
    ``mutating``, ``destructive``). Promoting an unknown class is a contract
    break that prevents the Critic from applying the correct safety gates.
    The runner never silently downgrades: an invalid value aborts the loop with
    a precise error, keeping production GCL fail-closed.
    """

    if not isinstance(intent, dict):
        return
    value = intent.get("safety_class")
    if value is None or value in SAFETY_CLASS_VALUES:
        return
    raise ValueError(
        f"operation_intent.safety_class={value!r} is not one of {SAFETY_CLASS_VALUES}; "
        "see docs/gcl-spec.md §operation_intent."
    )


def _mask_resource_scope(value: Any) -> Any:
    if isinstance(value, list):
        return [mask_resource_id(item) if isinstance(item, str) else _MASK for item in value]
    if isinstance(value, str):
        return mask_resource_id(value)
    return _MASK


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
    t0 = time.monotonic()
    try:
        proc = subprocess.run(
            command,
            shell=True,
            capture_output=True,
            text=True,
            timeout=timeout,
        )
        elapsed_ms = int((time.monotonic() - t0) * 1000)
        combined = (proc.stdout or "") + (proc.stderr or "")
        masked = mask_secrets(combined)
        excerpt = masked[:2000] + ("..." if len(masked) > 2000 else "")
        return {
            "command": mask_secrets(command),
            "exit_code": proc.returncode,
            "result_excerpt": excerpt,
            "stdout_len": len(proc.stdout or ""),
            "stderr_len": len(proc.stderr or ""),
            "duration_ms": elapsed_ms,
        }
    except subprocess.TimeoutExpired:
        elapsed_ms = int((time.monotonic() - t0) * 1000)
        return {
            "command": mask_secrets(command),
            "exit_code": -1,
            "result_excerpt": f"TIMEOUT after {timeout}s",
            "stdout_len": 0,
            "stderr_len": 0,
            "duration_ms": elapsed_ms,
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


def load_failure_patterns(root: Path, skill: str) -> list[dict[str, Any]]:
    """Load failure_patterns.json for a skill (empty list if absent)."""
    path = root / skill / "assets" / "failure_patterns.json"
    if not path.exists():
        return []
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
        return data.get("patterns", [])
    except (json.JSONDecodeError, OSError):
        return []


def match_pre_execution_risk(command: str, patterns: list[dict[str, Any]]) -> dict[str, Any] | None:
    """Check if command matches a known failure pattern signature."""
    for p in patterns:
        sig = p.get("signature", {})
        cmd_pattern = sig.get("command_pattern", "")
        if cmd_pattern and cmd_pattern in command:
            return p
        err_regex = sig.get("error_message_regex", "")
        if err_regex and re.search(err_regex, command, re.I):
            return p
    return None


def compute_ops_efficiency(trace: dict[str, Any]) -> dict[str, Any]:
    """Derive AIOps efficiency metrics from completed iterations."""
    iterations = trace.get("iterations", [])
    total_iters = len(iterations)
    retry_count = max(0, total_iters - 1)

    # Wasted time = duration of all non-final iterations (retries that didn't pass)
    wasted_time_ms = 0
    first_success_iter = None
    total_api_calls = 0
    for it in iterations:
        gen = it.get("generator", {})
        dur = gen.get("duration_ms", 0)
        total_api_calls += 1
        if it.get("decision") == "PASS":
            first_success_iter = it.get("iter")
            break
        wasted_time_ms += dur

    final_status = (trace.get("final") or {}).get("status", "UNKNOWN")
    automation_level = "full" if total_iters == 1 and final_status == "PASS" else "assisted"

    return {
        "retry_count": retry_count,
        "wasted_time_ms": wasted_time_ms,
        "first_success_iter": first_success_iter,
        "total_api_calls": total_api_calls,
        "automation_level": automation_level,
        "total_duration_ms": trace.get("duration_ms", 0),
    }


def compute_cost_attribution(trace: dict[str, Any]) -> dict[str, Any]:
    """Derive FinOps cost attribution from iterations + token_usage."""
    iterations = trace.get("iterations", [])
    cloud_api_calls = sum(1 for it in iterations if it.get("generator", {}).get("exit_code") is not None)

    token_usage = trace.get("token_usage", {})
    ai_cost_usd = token_usage.get("estimated_cost_usd", 0.0)

    # Resource cost from context (if provided)
    resource_ctx = trace.get("resource_context", {})
    resource_hourly_cost = resource_ctx.get("monthly_cost_usd", 0.0) / 720.0 if resource_ctx.get("monthly_cost_usd") else 0.0
    duration_hours = trace.get("duration_ms", 0) / 3_600_000.0
    resource_cost_usd = round(resource_hourly_cost * duration_hours, 8)

    total_cost_usd = round(ai_cost_usd + resource_cost_usd, 6)

    return {
        "cloud_api_calls": cloud_api_calls,
        "ai_cost_usd": ai_cost_usd,
        "resource_cost_usd": resource_cost_usd,
        "total_cost_usd": total_cost_usd,
        "cost_per_api_call_usd": round(ai_cost_usd / max(cloud_api_calls, 1), 6),
    }


def enhance_token_usage(trace: dict[str, Any]) -> None:
    """Add retry_waste and per-iteration cost to token_usage (mutates in place)."""
    token_usage = trace.get("token_usage")
    if not token_usage:
        return
    iterations = trace.get("iterations", [])
    total_iters = len(iterations)
    total_tokens = token_usage.get("total_tokens", 0)
    est_cost = token_usage.get("estimated_cost_usd", 0.0)

    if total_iters > 1:
        # Approximate retry waste: tokens attributed to failed iterations
        retry_waste_tokens = int(total_tokens * (total_iters - 1) / total_iters)
        token_usage["retry_waste_tokens"] = retry_waste_tokens
        token_usage["retry_waste_cost_usd"] = round(est_cost * (total_iters - 1) / total_iters, 6)
    else:
        token_usage["retry_waste_tokens"] = 0
        token_usage["retry_waste_cost_usd"] = 0.0

    token_usage["cost_per_iteration_usd"] = round(est_cost / max(total_iters, 1), 6)


CONTEXT_INJECT_KEYS = ("resource_context", "incident", "slo_context", "change_impact", "anomaly_baseline")


def _finalize_finops_aiops(trace: dict[str, Any]) -> None:
    """Compute and inject FinOps + AIOps derived fields before trace persistence."""
    enhance_token_usage(trace)
    trace["ops_efficiency"] = compute_ops_efficiency(trace)
    trace["cost_attribution"] = compute_cost_attribution(trace)


def persist_trace(root: Path, trace: dict[str, Any]) -> Path:
    out_dir = root / "audit-results"
    # Mode 0700 (owner-only) is required by `check_audit_results_guard`; a
    # default umask would create the dir as 0755 and fail the guard on the
    # first runtime call.
    out_dir.mkdir(parents=True, exist_ok=True, mode=0o700)
    ts = datetime.now(UTC).strftime("%Y%m%d-%H%M%S")
    path = out_dir / f"gcl-trace-{ts}.json"
    path.write_text(json.dumps(trace, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    return path


def cmd_run(args: argparse.Namespace) -> int:
    root = args.root
    max_iter = args.max_iter or SKILL_MAX_ITER.get(args.skill, 3)
    try:
        operation_intent = sanitize_operation_intent(args.operation_intent)
    except ValueError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2
    trace_id = f"gcl-{datetime.now(UTC).strftime('%Y%m%d-%H%M%S')}-{uuid.uuid4().hex[:8]}"
    started_at = _now_iso()
    t0_wall = time.monotonic()

    trace: dict[str, Any] = {
        "trace_schema_version": "v3",
        "trace_id": trace_id,
        "skill": args.skill,
        "skill_version": resolve_skill_version(root, args.skill),
        "request": mask_secrets(args.request),
        "operation_intent": operation_intent,
        "rubric_version": "v1",
        "masked_fields": ["request", "operation_intent", "generator.command", "generator.result_excerpt"],
        "started_at": started_at,
        "environment": {
            "runner_version": "2.0.0",
            "python_version": platform.python_version(),
            "platform": platform.system(),
            "ci": bool(os.environ.get("CI") or os.environ.get("GITHUB_ACTIONS")),
        },
        "iterations": [],
    }

    # Token usage injection (for cost analysis)
    if args.token_json:
        try:
            token_data = json.loads(Path(args.token_json).read_text(encoding="utf-8"))
            trace["token_usage"] = token_data
        except (json.JSONDecodeError, OSError) as exc:
            print(f"WARN: --token-json unreadable: {exc}", file=sys.stderr)

    # FinOps + AIOps runtime context injection
    if args.context_json:
        try:
            ctx = json.loads(Path(args.context_json).read_text(encoding="utf-8"))
            for key in CONTEXT_INJECT_KEYS:
                if key in ctx:
                    trace[key] = ctx[key]
        except (json.JSONDecodeError, OSError) as exc:
            print(f"WARN: --context-json unreadable: {exc}", file=sys.stderr)

    command = args.command

    # Pre-execution risk check against failure knowledge base
    patterns = load_failure_patterns(root, args.skill)
    risk = match_pre_execution_risk(command, patterns)
    if risk:
        trace["pre_execution_risk"] = {
            "pattern_id": risk.get("id", ""),
            "category": risk.get("category", ""),
            "known_fix": risk.get("fix", {}).get("action", ""),
            "historical_success_rate": risk.get("stats", {}).get("success_rate", 0.0),
        }

    critic_feedback = ""

    for iteration in range(1, max_iter + 1):
        generator = run_command(command, timeout=args.timeout)
        generator["args"] = {"iter": iteration, "critic_feedback": critic_feedback or None}

        if args.structural_critic_only:
            print(
                "WARN: --structural-critic-only is for CI/local smoke tests only; "
                "production GCL MUST use an externally supplied isolated Critic "
                "(--critic-json or stdin). See docs/gcl-spec.md.",
                file=sys.stderr,
            )
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
            trace["finished_at"] = _now_iso()
            trace["duration_ms"] = int((time.monotonic() - t0_wall) * 1000)
            trace["final"] = {
                "status": "SAFETY_FAIL",
                "iter": iteration,
                "output": None,
                "failure_pattern": extract_failure_pattern(args.skill, command, generator, critic),
            }
            _finalize_finops_aiops(trace)
            path = persist_trace(root, trace)
            print(f"SAFETY_FAIL — trace: {path}", file=sys.stderr)
            return 3

        if decision == "PASS":
            trace["finished_at"] = _now_iso()
            trace["duration_ms"] = int((time.monotonic() - t0_wall) * 1000)
            trace["final"] = {
                "status": "PASS",
                "iter": iteration,
                "output": generator.get("result_excerpt", ""),
                "failure_pattern": None,
            }
            _finalize_finops_aiops(trace)
            path = persist_trace(root, trace)
            print(f"PASS (iter {iteration}) — trace: {path}")
            return 0

        critic_feedback = "; ".join(critic.get("suggestions", [])[:3])

    last_iteration = trace["iterations"][-1]
    trace["finished_at"] = _now_iso()
    trace["duration_ms"] = int((time.monotonic() - t0_wall) * 1000)
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
    _finalize_finops_aiops(trace)
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
    run.add_argument(
        "--token-json",
        default=None,
        help="Path to JSON file with token usage data for cost analysis",
    )
    run.add_argument(
        "--context-json",
        default=None,
        help="Path to JSON with FinOps/AIOps runtime context (resource_context, incident, slo_context, change_impact, anomaly_baseline)",
    )
    run.set_defaults(func=cmd_run)
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
