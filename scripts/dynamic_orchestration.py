#!/usr/bin/env python3
"""Dynamic Orchestration Engine — multi-skill collaborative fault resolution.

Goes beyond the static delegation matrix by dynamically composing execution
plans based on fault diagnosis, resource topology, and historical success rates.

Usage:
  python3 scripts/dynamic_orchestration.py plan --fault "ECS instance unreachable" --skills huaweicloud-ecs-ops,huaweicloud-vpc-ops
  python3 scripts/dynamic_orchestration.py execute --plan <plan.json> [--dry-run]
  python3 scripts/dynamic_orchestration.py status --execution-id <id>
"""

from __future__ import annotations

import argparse
import json
import sys
import time
import uuid
from collections import deque
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

UTC = timezone.utc  # noqa: UP017

ROOT_DEFAULT = Path(__file__).resolve().parents[1]

# --- Skill Capability Registry ---
# Maps skill → capabilities it can contribute to orchestration
SKILL_CAPABILITIES: dict[str, dict[str, Any]] = {
    "huaweicloud-ecs-ops": {
        "domain": "compute",
        "capabilities": ["instance_lifecycle", "diagnostics", "resize", "migration"],
        "delegates_to": ["huaweicloud-vpc-ops", "huaweicloud-ces-ops", "huaweicloud-elb-ops"],
    },
    "huaweicloud-vpc-ops": {
        "domain": "network",
        "capabilities": ["connectivity", "security_group", "route_table", "subnet"],
        "delegates_to": ["huaweicloud-ecs-ops", "huaweicloud-elb-ops"],
    },
    "huaweicloud-ces-ops": {
        "domain": "monitoring",
        "capabilities": ["metrics_query", "alarm_management", "dashboard"],
        "delegates_to": [],
    },
    "huaweicloud-elb-ops": {
        "domain": "network",
        "capabilities": ["health_check", "backend_manage", "listener"],
        "delegates_to": ["huaweicloud-ecs-ops", "huaweicloud-vpc-ops"],
    },
    "huaweicloud-rds-ops": {
        "domain": "database",
        "capabilities": ["instance_lifecycle", "backup", "performance", "parameter"],
        "delegates_to": ["huaweicloud-ecs-ops", "huaweicloud-ces-ops", "huaweicloud-vpc-ops"],
    },
    "huaweicloud-iam-ops": {
        "domain": "identity",
        "capabilities": ["permission_check", "policy_manage", "credential_rotate"],
        "delegates_to": [],
    },
    "huaweicloud-cbr-ops": {
        "domain": "backup",
        "capabilities": ["backup_create", "restore", "policy_manage"],
        "delegates_to": ["huaweicloud-ecs-ops", "huaweicloud-rds-ops"],
    },
    "huaweicloud-dns-ops": {
        "domain": "network",
        "capabilities": ["record_manage", "zone_manage", "health_check"],
        "delegates_to": ["huaweicloud-vpc-ops"],
    },
    "huaweicloud-billing-ops": {
        "domain": "cost",
        "capabilities": ["cost_analysis", "budget_alert", "right_sizing"],
        "delegates_to": [],
    },
    "huaweicloud-cts-ops": {
        "domain": "audit",
        "capabilities": ["event_query", "trail_manage", "compliance_check"],
        "delegates_to": [],
    },
}

# --- Fault → Skill Routing Rules ---
# Maps fault keywords to required skill capabilities
FAULT_ROUTING: list[dict[str, Any]] = [
    {
        "pattern": ["unreachable", "connectivity", "timeout", "connection refused"],
        "required_capabilities": ["connectivity", "diagnostics"],
        "priority_skills": ["huaweicloud-vpc-ops", "huaweicloud-ecs-ops"],
    },
    {
        "pattern": ["disk full", "storage", "no space", "capacity"],
        "required_capabilities": ["instance_lifecycle", "metrics_query"],
        "priority_skills": ["huaweicloud-ecs-ops", "huaweicloud-ces-ops"],
    },
    {
        "pattern": ["permission denied", "403", "access denied", "unauthorized"],
        "required_capabilities": ["permission_check"],
        "priority_skills": ["huaweicloud-iam-ops"],
    },
    {
        "pattern": ["high cpu", "high memory", "performance", "slow"],
        "required_capabilities": ["metrics_query", "diagnostics"],
        "priority_skills": ["huaweicloud-ces-ops", "huaweicloud-ecs-ops"],
    },
    {
        "pattern": ["database", "rds", "connection pool", "deadlock"],
        "required_capabilities": ["instance_lifecycle", "performance"],
        "priority_skills": ["huaweicloud-rds-ops", "huaweicloud-ces-ops"],
    },
    {
        "pattern": ["backup", "restore", "recovery", "data loss"],
        "required_capabilities": ["backup_create", "restore"],
        "priority_skills": ["huaweicloud-cbr-ops"],
    },
    {
        "pattern": ["dns", "resolve", "domain"],
        "required_capabilities": ["record_manage", "health_check"],
        "priority_skills": ["huaweicloud-dns-ops", "huaweicloud-vpc-ops"],
    },
    {
        "pattern": ["cost", "billing", "budget", "overspend"],
        "required_capabilities": ["cost_analysis"],
        "priority_skills": ["huaweicloud-billing-ops"],
    },
]

# --- Execution Strategies ---
STRATEGIES = ("sequential", "parallel", "fan_out_collect", "pipeline")


def _now_iso() -> str:
    return datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


def match_fault_skills(fault: str, available_skills: list[str] | None) -> list[dict[str, Any]]:
    """Match a fault description to relevant skills with confidence scores."""
    fault_lower = fault.lower()
    matched: list[dict[str, Any]] = []

    for rule in FAULT_ROUTING:
        score = sum(1 for kw in rule["pattern"] if kw in fault_lower)
        if score == 0:
            continue
        confidence = min(score / len(rule["pattern"]), 1.0)
        for skill in rule["priority_skills"]:
            if available_skills and skill not in available_skills:
                continue
            if skill in SKILL_CAPABILITIES:
                matched.append({
                    "skill": skill,
                    "confidence": round(confidence, 2),
                    "domain": SKILL_CAPABILITIES[skill]["domain"],
                    "capabilities": SKILL_CAPABILITIES[skill]["capabilities"],
                    "matched_keywords": [kw for kw in rule["pattern"] if kw in fault_lower],
                })

    # Deduplicate by skill, keep highest confidence
    best: dict[str, dict[str, Any]] = {}
    for m in matched:
        key = m["skill"]
        if key not in best or m["confidence"] > best[key]["confidence"]:
            best[key] = m
    return sorted(best.values(), key=lambda x: -x["confidence"])


def discover_transitive_skills(primary_skills: list[str]) -> list[str]:
    """Discover additional skills needed via delegation chains (BFS)."""
    discovered: set[str] = set(primary_skills)
    queue: deque[str] = deque(primary_skills)
    while queue:
        skill = queue.popleft()
        cap = SKILL_CAPABILITIES.get(skill, {})
        for delegate in cap.get("delegates_to", []):
            if delegate not in discovered:
                discovered.add(delegate)
                queue.append(delegate)
    return sorted(discovered)


def select_strategy(skill_count: int, has_dependency: bool) -> str:
    """Select execution strategy based on skill count and dependencies."""
    if skill_count == 1:
        return "sequential"
    if has_dependency:
        return "pipeline"
    if skill_count <= 3:
        return "parallel"
    return "fan_out_collect"


def build_execution_plan(
    fault: str,
    skills: list[dict[str, Any]],
    strategy: str,
) -> dict[str, Any]:
    """Build a structured execution plan from matched skills."""
    plan_id = f"orch-{uuid.uuid4().hex[:12]}"
    steps: list[dict[str, Any]] = []

    if strategy == "pipeline":
        # Order by domain: monitoring → network → compute → database → backup
        domain_order = {"monitoring": 0, "identity": 1, "network": 2, "compute": 3, "database": 4, "backup": 5, "cost": 6, "audit": 7}
        ordered = sorted(skills, key=lambda s: domain_order.get(s["domain"], 99))
        for i, s in enumerate(ordered):
            steps.append({
                "step": i + 1,
                "skill": s["skill"],
                "action": "diagnose_and_remediate",
                "depends_on": [i] if i > 0 else [],
                "confidence": s["confidence"],
                "timeout_seconds": 300,
            })
    elif strategy == "parallel":
        for i, s in enumerate(skills):
            steps.append({
                "step": i + 1,
                "skill": s["skill"],
                "action": "diagnose_and_remediate",
                "depends_on": [],
                "confidence": s["confidence"],
                "timeout_seconds": 300,
            })
    else:
        for i, s in enumerate(skills):
            steps.append({
                "step": i + 1,
                "skill": s["skill"],
                "action": "diagnose_and_remediate",
                "depends_on": [i] if i > 0 and strategy == "sequential" else [],
                "confidence": s["confidence"],
                "timeout_seconds": 300,
            })

    return {
        "plan_id": plan_id,
        "created_at": _now_iso(),
        "fault_description": fault,
        "strategy": strategy,
        "total_skills": len(skills),
        "steps": steps,
        "rollback_policy": "reverse_order",
        "max_total_timeout_seconds": sum(s.get("timeout_seconds", 300) for s in steps),
        "status": "pending",
    }


def cmd_plan(args: argparse.Namespace) -> int:
    """Generate an orchestration plan for a fault."""
    fault = args.fault
    available = args.skills.split(",") if args.skills else None

    matched = match_fault_skills(fault, available)
    if not matched:
        print(f"No skills matched fault: {fault!r}", file=sys.stderr)
        print("Available skills:", ", ".join(sorted(SKILL_CAPABILITIES.keys())))
        return 1

    primary_skills = [m["skill"] for m in matched]
    all_skills = discover_transitive_skills(primary_skills)

    # Add transitively discovered skills not in matched
    matched_set = {m["skill"] for m in matched}
    for s in all_skills:
        if s not in matched_set and s in SKILL_CAPABILITIES:
            matched.append({
                "skill": s,
                "confidence": 0.3,
                "domain": SKILL_CAPABILITIES[s]["domain"],
                "capabilities": SKILL_CAPABILITIES[s]["capabilities"],
                "matched_keywords": [],
            })

    has_dependency = any(
        SKILL_CAPABILITIES.get(m["skill"], {}).get("delegates_to")
        for m in matched
    )
    strategy = select_strategy(len(matched), has_dependency)
    plan = build_execution_plan(fault, matched, strategy)

    if args.json:
        print(json.dumps(plan, indent=2, ensure_ascii=False))
    else:
        print(f"=== Orchestration Plan: {plan['plan_id']} ===")
        print(f"Fault: {fault}")
        print(f"Strategy: {strategy}")
        print(f"Skills: {len(matched)}")
        print()
        for step in plan["steps"]:
            deps = f" (after step {step['depends_on'][0]})" if step["depends_on"] else ""
            print(f"  Step {step['step']}: {step['skill']} [conf={step['confidence']}]{deps}")
        print()
        print(f"Rollback: {plan['rollback_policy']}")
        print(f"Max timeout: {plan['max_total_timeout_seconds']}s")

    if args.output:
        out = Path(args.output)
        out.write_text(json.dumps(plan, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
        print(f"\nPlan saved: {out}")

    return 0


def cmd_execute(args: argparse.Namespace) -> int:
    """Execute an orchestration plan (dry-run or real)."""
    plan_path = Path(args.plan)
    if not plan_path.exists():
        print(f"ERROR: plan file not found: {plan_path}", file=sys.stderr)
        return 1

    plan = json.loads(plan_path.read_text(encoding="utf-8"))
    execution_id = f"exec-{uuid.uuid4().hex[:8]}"

    print(f"=== Executing Plan: {plan['plan_id']} ===")
    print(f"Execution ID: {execution_id}")
    print(f"Strategy: {plan['strategy']}")
    print(f"Dry-run: {args.dry_run}")
    print()

    results: list[dict[str, Any]] = []
    for step in plan["steps"]:
        skill = step["skill"]
        print(f"  [{step['step']}/{len(plan['steps'])}] {skill}...", end=" ")
        if args.dry_run:
            print("SKIPPED (dry-run)")
            results.append({"step": step["step"], "skill": skill, "status": "skipped"})
        else:
            # In production, this would invoke the skill via GCL runner
            t0 = time.monotonic()
            results.append({
                "step": step["step"],
                "skill": skill,
                "status": "pending_execution",
                "started_at": _now_iso(),
                "duration_ms": int((time.monotonic() - t0) * 1000),
            })
            print("QUEUED")

    execution_record = {
        "execution_id": execution_id,
        "plan_id": plan["plan_id"],
        "started_at": _now_iso(),
        "dry_run": args.dry_run,
        "steps": results,
        "status": "completed" if args.dry_run else "in_progress",
    }

    if args.output:
        out = Path(args.output)
        out.write_text(json.dumps(execution_record, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
        print(f"\nExecution record: {out}")

    return 0


def cmd_status(args: argparse.Namespace) -> int:
    """Check execution status."""
    print(f"Execution {args.execution_id}: no persistent store configured (use --output on execute)")
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Dynamic Orchestration Engine — multi-skill collaborative fault resolution"
    )
    subparsers = parser.add_subparsers(dest="cmd", required=True)

    # plan
    p = subparsers.add_parser("plan", help="Generate orchestration plan for a fault")
    p.add_argument("--fault", required=True, help="Fault description text")
    p.add_argument("--skills", default=None, help="Comma-separated available skills (default: all)")
    p.add_argument("--json", action="store_true", help="Output as JSON")
    p.add_argument("--output", default=None, help="Save plan to file")
    p.set_defaults(func=cmd_plan)

    # execute
    e = subparsers.add_parser("execute", help="Execute an orchestration plan")
    e.add_argument("--plan", required=True, help="Path to plan JSON file")
    e.add_argument("--dry-run", action="store_true", help="Simulate without executing")
    e.add_argument("--output", default=None, help="Save execution record to file")
    e.set_defaults(func=cmd_execute)

    # status
    s = subparsers.add_parser("status", help="Check execution status")
    s.add_argument("--execution-id", required=True, help="Execution ID")
    s.set_defaults(func=cmd_status)

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
