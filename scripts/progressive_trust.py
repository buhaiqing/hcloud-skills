#!/usr/bin/env python3
"""Progressive Trust — trust scoring and automatic confirmation level adjustment.

Reduces unnecessary human confirmations by computing trust scores based on
historical success rates, time decay, risk levels, and operator experience.

Usage:
  python3 scripts/progressive_trust.py score --skill huaweicloud-ecs-ops --operation "resize-instance"
  python3 scripts/progressive_trust.py evaluate --trust-data <trust.json> --operation <op.json>
  python3 scripts/progressive_trust.py report --root . [--skill huaweicloud-ecs-ops]
  python3 scripts/progressive_trust.py update --root . --skill huaweicloud-ecs-ops --operation "resize" --outcome success
"""

from __future__ import annotations

import argparse
import json
import math
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

UTC = timezone.utc  # noqa: UP017

ROOT_DEFAULT = Path(__file__).resolve().parents[1]

# --- Trust Level Definitions ---
TRUST_LEVELS: dict[str, dict[str, Any]] = {
    "L0_new": {
        "min_score": 0.0,
        "confirmation": "always",
        "description": "New operation, no history — always require human confirmation",
        "max_auto_risk": "none",
    },
    "L1_provisional": {
        "min_score": 0.3,
        "confirmation": "high_risk_only",
        "description": "Some success history — confirm only high/critical risk operations",
        "max_auto_risk": "low",
    },
    "L2_established": {
        "min_score": 0.6,
        "confirmation": "critical_only",
        "description": "Good track record — confirm only critical risk operations",
        "max_auto_risk": "medium",
    },
    "L3_trusted": {
        "min_score": 0.8,
        "confirmation": "never",
        "description": "Excellent track record — auto-execute all except destructive",
        "max_auto_risk": "high",
    },
    "L4_autonomous": {
        "min_score": 0.95,
        "confirmation": "never",
        "description": "Proven autonomous capability — full auto including destructive with rollback",
        "max_auto_risk": "critical",
    },
}

# Risk level ordering
RISK_ORDER = {"none": 0, "low": 1, "medium": 2, "high": 3, "critical": 4}

# --- Trust Score Components ---
# Weights for trust score computation
WEIGHTS = {
    "success_rate": 0.35,       # Historical success rate
    "consistency": 0.20,        # Variance in outcomes (lower = better)
    "recency": 0.20,            # Time decay (recent successes count more)
    "complexity_mastery": 0.15, # Success on complex operations
    "error_recovery": 0.10,     # Ability to recover from errors
}

# Half-life for time decay (days)
RECENCY_HALF_LIFE_DAYS = 30.0


def _now_iso() -> str:
    return datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


def _parse_iso(ts: str) -> datetime | None:
    """Best-effort ISO timestamp parse."""
    for fmt in ("%Y-%m-%dT%H:%M:%SZ", "%Y-%m-%dT%H:%M:%S%z", "%Y-%m-%d"):
        try:
            return datetime.strptime(ts, fmt).replace(tzinfo=UTC)
        except ValueError:
            continue
    return None


def compute_success_rate(history: list[dict[str, Any]]) -> float:
    """Compute simple success rate from operation history."""
    if not history:
        return 0.0
    successes = sum(1 for h in history if h.get("outcome") == "success")
    return successes / len(history)


def compute_consistency(history: list[dict[str, Any]]) -> float:
    """Compute consistency score (1.0 = perfectly consistent outcomes)."""
    if len(history) < 3:
        return 0.5  # Neutral for insufficient data
    outcomes = [1.0 if h.get("outcome") == "success" else 0.0 for h in history]
    mean = sum(outcomes) / len(outcomes)
    variance = sum((o - mean) ** 2 for o in outcomes) / len(outcomes)
    # Lower variance = higher consistency
    return max(0.0, 1.0 - math.sqrt(variance))


def compute_recency(history: list[dict[str, Any]]) -> float:
    """Compute recency-weighted score using exponential decay."""
    if not history:
        return 0.0
    now = datetime.now(UTC)
    weighted_sum = 0.0
    weight_total = 0.0

    for h in history:
        ts = h.get("timestamp", "")
        dt = _parse_iso(ts) if ts else None
        if dt is None:
            age_days = RECENCY_HALF_LIFE_DAYS  # Default age if no timestamp
        else:
            age_days = max(0, (now - dt).total_seconds() / 86400)

        # Exponential decay weight
        weight = math.exp(-0.693 * age_days / RECENCY_HALF_LIFE_DAYS)
        outcome_score = 1.0 if h.get("outcome") == "success" else 0.0
        weighted_sum += weight * outcome_score
        weight_total += weight

    return weighted_sum / max(weight_total, 1e-10)


def compute_complexity_mastery(history: list[dict[str, Any]]) -> float:
    """Score based on success rate for complex/high-risk operations."""
    complex_ops = [h for h in history if h.get("risk_level") in ("high", "critical")]
    if not complex_ops:
        # No complex ops attempted — neutral score
        return 0.5
    successes = sum(1 for h in complex_ops if h.get("outcome") == "success")
    return successes / len(complex_ops)


def compute_error_recovery(history: list[dict[str, Any]]) -> float:
    """Score based on successful recovery after initial failures."""
    recoveries = 0
    opportunities = 0
    for i, h in enumerate(history):
        if h.get("outcome") == "failure" and h.get("had_retry"):
            opportunities += 1
            # Check if next operation succeeded
            if i + 1 < len(history) and history[i + 1].get("outcome") == "success":
                recoveries += 1
    if opportunities == 0:
        return 0.7  # No failures to recover from — slightly positive
    return recoveries / opportunities


def compute_trust_score(history: list[dict[str, Any]]) -> dict[str, Any]:
    """Compute composite trust score from operation history."""
    components = {
        "success_rate": compute_success_rate(history),
        "consistency": compute_consistency(history),
        "recency": compute_recency(history),
        "complexity_mastery": compute_complexity_mastery(history),
        "error_recovery": compute_error_recovery(history),
    }

    weighted_score = sum(
        components[key] * WEIGHTS[key] for key in WEIGHTS
    )
    # Clamp to [0, 1]
    score = round(max(0.0, min(1.0, weighted_score)), 4)

    # Determine trust level
    level = "L0_new"
    for lvl_name, lvl_def in sorted(TRUST_LEVELS.items(), key=lambda x: -x[1]["min_score"]):
        if score >= lvl_def["min_score"]:
            level = lvl_name
            break

    return {
        "score": score,
        "level": level,
        "level_description": TRUST_LEVELS[level]["description"],
        "confirmation_policy": TRUST_LEVELS[level]["confirmation"],
        "max_auto_risk": TRUST_LEVELS[level]["max_auto_risk"],
        "components": {k: round(v, 4) for k, v in components.items()},
        "history_size": len(history),
        "computed_at": _now_iso(),
    }


def evaluate_operation(
    trust_score: dict[str, Any],
    operation_risk: str,
    operation_type: str,
) -> dict[str, Any]:
    """Evaluate whether an operation needs human confirmation."""
    max_auto = trust_score["max_auto_risk"]
    risk_val = RISK_ORDER.get(operation_risk, 4)
    max_auto_val = RISK_ORDER.get(max_auto, 0)

    auto_approved = risk_val <= max_auto_val
    reason = (
        f"Risk '{operation_risk}' ≤ max auto '{max_auto}' at trust level {trust_score['level']}"
        if auto_approved
        else f"Risk '{operation_risk}' > max auto '{max_auto}' — requires confirmation"
    )

    # Additional safety overrides
    override = None
    if operation_type in ("delete", "terminate", "destroy") and trust_score["level"] != "L4_autonomous":
        auto_approved = False
        override = "Destructive operation requires L4_autonomous trust level"
    if operation_risk == "critical" and trust_score["score"] < 0.95:
        auto_approved = False
        override = "Critical risk requires score ≥ 0.95"

    return {
        "operation_type": operation_type,
        "operation_risk": operation_risk,
        "trust_level": trust_score["level"],
        "trust_score": trust_score["score"],
        "auto_approved": auto_approved,
        "reason": override or reason,
        "requires_confirmation": not auto_approved,
        "evaluated_at": _now_iso(),
    }


def load_trust_data(root: Path, skill: str) -> dict[str, Any]:
    """Load trust history for a skill."""
    path = root / skill / "assets" / "trust_history.json"
    if path.exists():
        return json.loads(path.read_text(encoding="utf-8"))
    return {
        "schema": "trust-history/v1",
        "skill_id": skill,
        "operations": {},
        "meta": {"created_at": _now_iso(), "total_evaluations": 0},
    }


def save_trust_data(root: Path, skill: str, data: dict[str, Any]) -> Path:
    """Save trust history."""
    path = root / skill / "assets" / "trust_history.json"
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    return path


def cmd_score(args: argparse.Namespace) -> int:
    """Compute trust score for a skill+operation."""
    root: Path = args.root
    skill = args.skill
    operation = args.operation

    data = load_trust_data(root, skill)
    history = data.get("operations", {}).get(operation, [])

    score = compute_trust_score(history)

    if args.json:
        print(json.dumps(score, indent=2, ensure_ascii=False))
    else:
        print(f"=== Trust Score: {skill} / {operation} ===")
        print(f"Score: {score['score']} ({score['level']})")
        print(f"Policy: {score['confirmation_policy']}")
        print(f"Max auto risk: {score['max_auto_risk']}")
        print(f"History: {score['history_size']} operations")
        print()
        print("Components:")
        for k, v in score["components"].items():
            bar = "█" * int(v * 20) + "░" * (20 - int(v * 20))
            print(f"  {k:20s} {bar} {v:.3f}")

    return 0


def cmd_evaluate(args: argparse.Namespace) -> int:
    """Evaluate if an operation needs confirmation.

    Loads trust history from --trust-data if provided, otherwise defaults to
    `<root>/<skill>/assets/trust_history.json`. After evaluation, records the
    outcome back to the source file so trust scores accumulate across sessions.
    """
    if args.trust_data:
        trust_path = Path(args.trust_data)
        if not trust_path.exists():
            print(f"ERROR: trust data not found: {trust_path}", file=sys.stderr)
            return 1
        trust_data = json.loads(trust_path.read_text(encoding="utf-8"))
        history = trust_data.get("history", [])
        persist_back = False  # external file: caller manages writes
    else:
        if not args.skill:
            print("ERROR: --skill is required when --trust-data is not provided", file=sys.stderr)
            return 1
        trust_path = args.root / args.skill / "assets" / "trust_history.json"
        data = load_trust_data(args.root, args.skill)
        # Normalize: trust_history.json stores {operations: {op: [...]}}; evaluate wants flat history
        all_ops: list[dict[str, Any]] = []
        for op_history in data.get("operations", {}).values():
            all_ops.extend(op_history)
        history = all_ops
        trust_data = {"history": history}
        persist_back = True

    score = compute_trust_score(history)

    op_risk = args.risk
    op_type = args.operation_type

    result = evaluate_operation(score, op_risk, op_type)

    # Persist back to default location so trust accumulates across sessions
    if persist_back and not args.trust_data:
        ops_dict = data.setdefault("operations", {})
        op_history_list = ops_dict.setdefault(op_type, [])
        # Append a single neutral observation so the score reflects activity
        op_history_list.append({
            "operation": op_type,
            "outcome": "evaluated",
            "risk": op_risk,
            "ts": _now_iso(),
            "trust_level": result["trust_level"],
        })
        data["meta"]["total_evaluations"] = data["meta"].get("total_evaluations", 0) + 1
        save_trust_data(args.root, args.skill, data)
        result["persisted_to"] = str(trust_path)

    if args.json:
        print(json.dumps(result, indent=2, ensure_ascii=False))
    else:
        status = "AUTO-APPROVED" if result["auto_approved"] else "REQUIRES CONFIRMATION"
        print(f"=== Evaluation: {op_type} (risk={op_risk}) ===")
        print(f"Decision: {status}")
        print(f"Reason: {result['reason']}")
        print(f"Trust level: {result['trust_level']} (score={result['trust_score']})")
        if result.get("persisted_to"):
            print(f"Persisted: {result['persisted_to']}")

    return 0


def cmd_state_path(args: argparse.Namespace) -> int:
    """Print default trust-state file path for the given skill."""
    path = args.root / args.skill / "assets" / "trust_history.json"
    print(str(path))
    return 0


def cmd_update(args: argparse.Namespace) -> int:
    """Record an operation outcome to update trust history."""
    root: Path = args.root
    skill = args.skill
    operation = args.operation
    outcome = args.outcome

    data = load_trust_data(root, skill)
    ops = data.setdefault("operations", {})
    history = ops.setdefault(operation, [])

    entry: dict[str, Any] = {
        "outcome": outcome,
        "timestamp": _now_iso(),
        "risk_level": args.risk,
        "had_retry": args.retry,
    }
    history.append(entry)
    data["meta"]["total_evaluations"] = data["meta"].get("total_evaluations", 0) + 1

    # Keep only last 100 entries per operation
    if len(history) > 100:
        ops[operation] = history[-100:]

    if args.dry_run:
        score = compute_trust_score(history)
        print(f"[DRY-RUN] New score would be: {score['score']} ({score['level']})")
        return 0

    out_path = save_trust_data(root, skill, data)
    score = compute_trust_score(history)
    print(f"Recorded: {operation} → {outcome}")
    print(f"New trust score: {score['score']} ({score['level']})")
    print(f"Saved: {out_path}")
    return 0


def cmd_report(args: argparse.Namespace) -> int:
    """Generate trust report for skills."""
    root: Path = args.root
    skills = [args.skill] if args.skill else [
        d.name for d in root.iterdir()
        if d.is_dir() and d.name.startswith("huaweicloud-") and d.name.endswith("-ops")
    ]

    print("=== Progressive Trust Report ===")
    for skill in sorted(skills):
        data = load_trust_data(root, skill)
        ops = data.get("operations", {})
        if not ops:
            continue
        print(f"\n  {skill}:")
        for op_name, history in sorted(ops.items()):
            score = compute_trust_score(history)
            print(f"    {op_name}: {score['score']:.3f} ({score['level']}) [{len(history)} ops]")

    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Progressive Trust — trust scoring and confirmation level adjustment"
    )
    subparsers = parser.add_subparsers(dest="cmd", required=True)

    # score
    s = subparsers.add_parser("score", help="Compute trust score")
    s.add_argument("--root", type=Path, default=ROOT_DEFAULT, help="Repo root")
    s.add_argument("--skill", required=True, help="Skill id")
    s.add_argument("--operation", required=True, help="Operation name")
    s.add_argument("--json", action="store_true", help="Output as JSON")
    s.set_defaults(func=cmd_score)

    # evaluate
    e = subparsers.add_parser("evaluate", help="Evaluate operation confirmation need")
    e.add_argument("--trust-data", required=False, help="Path to trust history JSON (defaults to <root>/<skill>/assets/trust_history.json)")
    e.add_argument("--root", type=Path, default=ROOT_DEFAULT, help="Repo root (used when --trust-data omitted)")
    e.add_argument("--skill", default=None, help="Skill id (used when --trust-data omitted)")
    e.add_argument("--operation-type", required=True, help="Operation type (e.g., resize, delete)")
    e.add_argument("--risk", required=True, choices=["low", "medium", "high", "critical"])
    e.add_argument("--json", action="store_true", help="Output as JSON")
    e.set_defaults(func=cmd_evaluate)

    # state-path — print default trust history file location
    sp = subparsers.add_parser("state-path", help="Print default trust-state file path")
    sp.add_argument("--root", type=Path, default=ROOT_DEFAULT, help="Repo root")
    sp.add_argument("--skill", required=True, help="Skill id")
    sp.set_defaults(func=cmd_state_path)

    # update
    u = subparsers.add_parser("update", help="Record operation outcome")
    u.add_argument("--root", type=Path, default=ROOT_DEFAULT, help="Repo root")
    u.add_argument("--skill", required=True, help="Skill id")
    u.add_argument("--operation", required=True, help="Operation name")
    u.add_argument("--outcome", required=True, choices=["success", "failure"])
    u.add_argument("--risk", default="low", choices=["low", "medium", "high", "critical"])
    u.add_argument("--retry", action="store_true", help="Mark as had retry")
    u.add_argument("--dry-run", action="store_true", help="Show score without saving")
    u.set_defaults(func=cmd_update)

    # report
    r = subparsers.add_parser("report", help="Generate trust report")
    r.add_argument("--root", type=Path, default=ROOT_DEFAULT, help="Repo root")
    r.add_argument("--skill", default=None, help="Specific skill")
    r.set_defaults(func=cmd_report)

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
