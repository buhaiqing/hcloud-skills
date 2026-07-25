"""L4 Runtime Orchestrator — closed-loop fault handler.

Chains 4 L4 engines (topology, predictive, orchestration, trust) + GCL quality gate
+ trace learning into a single fault-handling pipeline:

    topology → orchestration → predictive → GCL gate → trust → learning

This is the runtime surface that turns the 4 independent CLIs into a closed loop.
Reuses engine functions via direct import (not subprocess) for efficiency.

Usage:
    runtime_orchestrator.py handle --fault "RDS connection timeout" \\
        --resource rds:instance --risk medium --skills huaweicloud-rds-ops --json
    runtime_orchestrator.py handle --fault "ECS high CPU" --resource ecs:instance --json
"""
from __future__ import annotations

import argparse
import json
import sys
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

# Direct imports of engine functions (cheaper than subprocess; lets us reuse state)
sys.path.insert(0, str(Path(__file__).resolve().parent))

import dynamic_orchestration as do
import predictive_ops as po
import progressive_trust as pt
import topology_graph as tg
from gcl_runner import (
    decide,
    load_failure_patterns,
    mask_secrets,
    match_pre_execution_risk,
    structural_critic,
)

UTC = timezone.utc  # noqa: UP017
ROOT = Path(__file__).resolve().parents[1]


def _now_iso() -> str:
    return datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


def _derive_resource_from_fault(fault: str) -> str:
    """Heuristic: extract resource type from fault description."""
    fault_l = fault.lower()
    for token in ("rds", "ecs", "elb", "vpc", "cce", "dcs", "gaussdb", "dms"):
        if token in fault_l:
            return f"{token}:instance"
    return "unknown:resource"


def handle_fault(
    fault: str,
    resource: str | None,
    risk: str,
    skills: list[str] | None,
    trust_data: dict[str, Any] | None,
    metric_values: list[float] | None,
    metric_threshold: float | None,
    audit_root: Path,
) -> dict[str, Any]:
    """Run the full closed-loop pipeline for a fault."""
    fault_id = str(uuid.uuid4())
    started_at = _now_iso()
    resource = resource or _derive_resource_from_fault(fault)

    # Step 1 — Topology: blast radius for the affected resource
    graph = tg.build_graph_from_skills(ROOT)
    impact = graph.blast_radius(resource, max_depth=3)
    topology_result = {
        "origin": impact["origin"],
        "total_affected": impact["total_affected"],
        "max_depth_reached": impact["max_depth_reached"],
        "criticality_score": graph._compute_criticality(resource),  # noqa: SLF001
        "domains_impacted": impact.get("domains_impacted", []),
        "affected_resources": [r["resource_id"] for r in impact["affected_resources"][:5]],
    }

    # Step 2 — Orchestration: discover skills, build execution plan
    matched_skills = do.match_fault_skills(fault, skills)
    discovered = do.discover_transitive_skills([s["skill"] for s in matched_skills])
    strategy = do.select_strategy(len(discovered), has_dependency=len(discovered) > 1)
    execution_plan = do.build_execution_plan(fault=fault, skills=matched_skills, strategy=strategy)
    orchestration_result = {
        "primary_skills": [s["skill"] for s in matched_skills],
        "transitive_skills": discovered,
        "strategy": strategy,
        "plan_id": execution_plan["plan_id"],
        "step_count": len(execution_plan["steps"]),
        "max_total_timeout_seconds": execution_plan["max_total_timeout_seconds"],
    }

    # Step 3 — Predictive: optional trend detection on supplied metric series
    predictive_result: dict[str, Any] = {"trend": None, "breach": None, "evaluated": False}
    if metric_values and len(metric_values) >= 3:
        trend = po.detect_trend(metric_values)
        predictive_result["trend"] = trend
        if metric_threshold is not None:
            breach = po.predict_breach_time(metric_values, threshold=metric_threshold, interval_hours=1.0)
            predictive_result["breach"] = breach
        predictive_result["evaluated"] = True

    # Step 4 — GCL: structural critic on the planned steps (read-only safety gate)
    gcl_decisions: list[dict[str, Any]] = []
    overall_safety = True
    for step in execution_plan["steps"]:
        skill_short = step.get("skill_short") or step.get("skill", "").replace("huaweicloud-", "").replace("-ops", "")
        candidate_cmd = f"hcloud {skill_short} {step.get('action', 'unknown')}"
        generator_payload = {
            "command": candidate_cmd,
            "exit_code": 0,
            "result_excerpt": "dry-run",
            "skill": step.get("skill", "unknown"),
            "operation": step.get("action", "unknown"),
            "risk_level": risk,
            "iteration": 1,
        }
        scores = structural_critic(generator_payload).get("scores", {})
        # Augment with pre-execution risk match (knowledge-base look-ahead)
        pre_risk = None
        if "skill" in step:
            patterns = load_failure_patterns(ROOT, step["skill"])
            pre_risk = match_pre_execution_risk(candidate_cmd, patterns)
        critic_payload = {
            "scores": scores,
            "decision": decide(scores),
            "pre_execution_risk": pre_risk,
        }
        gcl_decisions.append({"step": step.get("step"), "gcl": critic_payload})
        if scores.get("safety", 1.0) == 0.0:
            overall_safety = False
    gcl_result = {
        "overall_safety": overall_safety,
        "decisions": gcl_decisions,
        "passed_steps": sum(1 for d in gcl_decisions if d["gcl"]["decision"] in ("PASS", "ACCEPT")),
    }

    # Step 5 — Trust: evaluate + update
    trust_history = trust_data.get("operations", []) if trust_data else []
    trust_eval = pt.evaluate_operation(
        trust_score=pt.compute_trust_score(trust_history),
        operation_risk=risk,
        operation_type=fault,
    )
    trust_result = {
        "trust_level": trust_eval["trust_level"],
        "composite_score": pt.compute_trust_score(trust_history)["score"],
        "auto_approve": trust_eval.get("auto_approved", False),
        "requires_human_approval": trust_eval.get("requires_confirmation", True),
    }

    # Step 6 — Learning: synthesize a trace record (no real execution in dry-run mode)
    trace = {
        "trace_id": fault_id,
        "skill": matched_skills[0]["skill"] if matched_skills else "unknown",
        "request": fault,
        "command": execution_plan["steps"][0]["action"] if execution_plan["steps"] else "",
        "started_at": started_at,
        "finished_at": _now_iso(),
        "status": "pass" if overall_safety else "fail",
        "exit_code": 0 if overall_safety else 1,
        "stdout": "",
        "stderr": "",
        "iteration": 1,
        "max_iterations": 1,
        "decision": "pass" if overall_safety else "halt",
        "resource_scope": {"resource_id": resource, "type": resource.split(":")[0]},
        "operation_intent": {"goal": fault, "risk_class": risk},
        "critic_scores": {"safety": 1.0 if overall_safety else 0.0, "correctness": 0.9, "idempotency": 0.85, "secops": 0.95, "finops": 0.8},
        "trust": trust_result,
        "topology": topology_result,
        "predictive": predictive_result,
    }
    # Persist trace (mask secrets before write)
    trace_path = audit_root / f"orchestrator-trace-{fault_id}.json"
    trace_path.write_text(json.dumps(mask_secrets(json.dumps(trace)), ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    learning_result = {
        "trace_persisted": str(trace_path),
        "patterns_matched": sum(1 for d in gcl_decisions if d["gcl"]["pre_execution_risk"]),
        "knowledge_base_skills_used": sorted({s["skill"] for s in matched_skills}),
    }

    return {
        "fault_id": fault_id,
        "started_at": started_at,
        "finished_at": _now_iso(),
        "fault_description": fault,
        "resource": resource,
        "risk_class": risk,
        "topology": topology_result,
        "orchestration": orchestration_result,
        "predictive": predictive_result,
        "gcl": gcl_result,
        "trust": trust_result,
        "learning": learning_result,
        "decision": "auto_proceed" if (overall_safety and trust_result["auto_approve"]) else "human_review_required",
    }


def cmd_handle(args: argparse.Namespace) -> int:
    audit_root = ROOT / "audit-results"
    audit_root.mkdir(parents=True, exist_ok=True)
    trust_data = None
    if args.trust_data:
        trust_data = json.loads(args.trust_data.read_text() if isinstance(args.trust_data, Path) else args.trust_data)
    result = handle_fault(
        fault=args.fault,
        resource=args.resource,
        risk=args.risk,
        skills=args.skills.split(",") if args.skills else None,
        trust_data=trust_data,
        metric_values=[float(x) for x in args.metric_values.split(",")] if args.metric_values else None,
        metric_threshold=args.metric_threshold,
        audit_root=audit_root,
    )
    if args.output:
        Path(args.output).write_text(json.dumps(result, indent=2, ensure_ascii=False), encoding="utf-8")
    print(json.dumps(result, indent=2, ensure_ascii=False))
    return 0


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description="L4 Runtime Orchestrator — closed-loop fault handler.")
    sub = p.add_subparsers(dest="cmd", required=True)

    h = sub.add_parser("handle", help="Handle a fault end-to-end through the L4 pipeline")
    h.add_argument("--fault", required=True, help="Fault description, e.g. 'RDS connection timeout'")
    h.add_argument("--resource", default=None, help="Affected resource id, e.g. 'rds:instance'. Auto-derived if omitted.")
    h.add_argument("--risk", choices=["low", "medium", "high", "critical"], default="medium")
    h.add_argument("--skills", default=None, help="Comma-separated primary skills, e.g. 'huaweicloud-rds-ops,huaweicloud-vpc-ops'")
    h.add_argument("--trust-data", default=None, help="JSON string or @path with trust history")
    h.add_argument("--metric-values", default=None, help="Comma-separated numeric series for predictive trend detection")
    h.add_argument("--metric-threshold", type=float, default=None, help="Threshold for breach-time prediction")
    h.add_argument("--json", action="store_true", help="Emit JSON (default for handle)")
    h.add_argument("--output", default=None, help="Write result to this path")
    h.set_defaults(func=cmd_handle)
    return p


def main() -> int:
    args = build_parser().parse_args()
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())