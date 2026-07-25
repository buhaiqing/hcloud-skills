#!/usr/bin/env python3
"""GCL Phase 3 — aggregate trace files into a quality summary.

Scans `audit-results/gcl-trace-*.json`, computes aggregate quality metrics
(pass rate, rubric averages, per-skill breakdown, FinOps/AIOps stats), and
writes `audit-results/gcl-quality-summary-YYYYMMDD-HHMMSS.json` consumed by
`gcl_alarm_wire.py` for CES SLO evaluation.

Usage:
  python3 scripts/gcl_trace_aggregate.py --since-hours 168
  python3 scripts/gcl_trace_aggregate.py --since-hours 24 --output -
  python3 scripts/gcl_trace_aggregate.py --all
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timezone

# `datetime.UTC` is a 3.11+ alias; the agent runtime is still on Python 3.10
# (see AGENTS.md §Python 3.10 Syntax Compatibility). Use `timezone.utc` and
# expose it under the name `UTC` so call sites stay short.
UTC = timezone.utc  # noqa: UP017
from pathlib import Path  # noqa: E402
from typing import Any  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]
AUDIT_DIR = ROOT / "audit-results"
TRACE_GLOB = "gcl-trace-*.json"

# Rubric dimensions tracked in GCL spec §3
RUBRIC_DIMENSIONS = ("correctness", "safety", "idempotency", "traceability", "spec_compliance")


def is_gcl_trace(data: dict[str, Any]) -> bool:
    """Return True if the JSON payload is a valid GCL trace (not an orchestrator decision)."""
    return "trace_schema_version" in data and "final" in data and "status" in data.get("final", {})


def parse_trace_timestamp(path: Path, data: dict[str, Any]) -> datetime | None:
    """Extract timestamp from trace content or filename for time-window filtering."""
    # v2+ traces have started_at
    started = data.get("started_at")
    if started:
        try:
            return datetime.fromisoformat(started.replace("Z", "+00:00"))
        except (ValueError, AttributeError):
            pass
    # Fallback: parse from filename pattern gcl-trace-YYYYMMDD-HHMMSS.json
    stem = path.stem  # e.g. gcl-trace-20260621-022734
    parts = stem.split("-", 2)  # ['gcl', 'trace', '20260621-022734']
    if len(parts) >= 3:
        ts_part = parts[2]
        for fmt in ("%Y%m%d-%H%M%S", "%Y%m%dT%H%M%S"):
            try:
                return datetime.strptime(ts_part, fmt).replace(tzinfo=UTC)
            except ValueError:
                continue
    return None


def collect_traces(audit_dir: Path, since_hours: float | None) -> list[tuple[Path, dict[str, Any]]]:
    """Load and filter GCL trace files from audit directory."""
    if not audit_dir.exists():
        return []
    cutoff: datetime | None = None
    if since_hours is not None:
        cutoff = datetime.now(UTC).replace(microsecond=0) - _hours_delta(since_hours)

    results: list[tuple[Path, dict[str, Any]]] = []
    for path in sorted(audit_dir.glob(TRACE_GLOB)):
        try:
            data = json.loads(path.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, OSError):
            continue
        if not is_gcl_trace(data):
            continue
        if cutoff is not None:
            ts = parse_trace_timestamp(path, data)
            if ts is not None and ts < cutoff:
                continue
        results.append((path, data))
    return results


def _hours_delta(hours: float) -> Any:
    """Create a timedelta from hours (avoid importing timedelta at top for style)."""
    from datetime import timedelta

    return timedelta(hours=hours)


def extract_final_scores(trace: dict[str, Any]) -> dict[str, float] | None:
    """Extract rubric scores from the final iteration's critic."""
    iterations = trace.get("iterations", [])
    if not iterations:
        return None
    final_iter = iterations[-1]
    scores = final_iter.get("critic", {}).get("scores")
    if scores and isinstance(scores, dict):
        return {k: float(v) for k, v in scores.items() if k in RUBRIC_DIMENSIONS}
    return None


def aggregate(traces: list[tuple[Path, dict[str, Any]]]) -> dict[str, Any]:
    """Compute the quality summary from a list of (path, trace_data) tuples."""
    totals: dict[str, int] = {"PASS": 0, "SAFETY_FAIL": 0, "MAX_ITER": 0, "total_runs": 0}
    rubric_sums: dict[str, float] = {dim: 0.0 for dim in RUBRIC_DIMENSIONS}
    rubric_counts: dict[str, int] = {dim: 0 for dim in RUBRIC_DIMENSIONS}
    by_skill: dict[str, dict[str, Any]] = {}
    trace_files: list[str] = []

    # v3 FinOps/AIOps aggregation
    total_cost_usd = 0.0
    total_ai_cost_usd = 0.0
    total_retry_waste_tokens = 0
    cost_count = 0
    token_count = 0
    ops_retry_counts: list[int] = []
    ops_wasted_time: list[int] = []
    automation_levels: dict[str, int] = {}
    total_duration_ms_list: list[int] = []

    for path, trace in traces:
        totals["total_runs"] += 1
        status = trace.get("final", {}).get("status", "UNKNOWN")
        if status in totals:
            totals[status] += 1

        trace_files.append(path.name)

        # Rubric scores
        scores = extract_final_scores(trace)
        if scores:
            for dim in RUBRIC_DIMENSIONS:
                if dim in scores:
                    rubric_sums[dim] += scores[dim]
                    rubric_counts[dim] += 1

        # Per-skill breakdown
        skill = trace.get("skill", "unknown")
        if skill not in by_skill:
            by_skill[skill] = {"total": 0, "PASS": 0, "SAFETY_FAIL": 0, "MAX_ITER": 0, "iter_sum": 0}
        by_skill[skill]["total"] += 1
        if status in by_skill[skill]:
            by_skill[skill][status] += 1
        iter_count = len(trace.get("iterations", []))
        by_skill[skill]["iter_sum"] += max(iter_count, 1)

        # v3: cost_attribution
        cost_attr = trace.get("cost_attribution")
        if cost_attr and isinstance(cost_attr, dict):
            total_cost_usd += float(cost_attr.get("total_cost_usd", 0))
            total_ai_cost_usd += float(cost_attr.get("ai_cost_usd", 0))
            cost_count += 1

        # v3: token_usage
        token_usage = trace.get("token_usage")
        if token_usage and isinstance(token_usage, dict):
            total_retry_waste_tokens += int(token_usage.get("retry_waste_tokens", 0))
            token_count += 1

        # v3: ops_efficiency
        ops_eff = trace.get("ops_efficiency")
        if ops_eff and isinstance(ops_eff, dict):
            ops_retry_counts.append(int(ops_eff.get("retry_count", 0)))
            ops_wasted_time.append(int(ops_eff.get("wasted_time_ms", 0)))
            level = ops_eff.get("automation_level", "unknown")
            automation_levels[level] = automation_levels.get(level, 0) + 1
            dur = ops_eff.get("total_duration_ms")
            if dur is not None:
                total_duration_ms_list.append(int(dur))

    # Compute averages
    total_runs = totals["total_runs"]
    pass_rate = totals["PASS"] / total_runs if total_runs > 0 else 0.0
    avg_rubric: dict[str, float] = {}
    for dim in RUBRIC_DIMENSIONS:
        avg_rubric[dim] = round(rubric_sums[dim] / rubric_counts[dim], 4) if rubric_counts[dim] > 0 else 0.0

    # Finalize by_skill (compute avg_iterations)
    for skill_data in by_skill.values():
        total = skill_data["total"]
        skill_data["avg_iterations"] = round(skill_data.pop("iter_sum") / total, 2) if total > 0 else 0.0

    summary: dict[str, Any] = {
        "version": "1.0",
        "generated_at": datetime.now(UTC).isoformat(),
        "cloud": "huaweicloud",
        "metric_namespace": "CUSTOM.GCL",
        "window": {"trace_count": total_runs},
        "totals": totals,
        "pass_rate": round(pass_rate, 4),
        "avg_rubric_scores": avg_rubric,
        "by_skill": by_skill,
        "trace_files": trace_files,
    }

    # v3 FinOps/AIOps aggregate (only if data present)
    finops_agg: dict[str, Any] = {}
    if cost_count > 0:
        finops_agg["total_cost_usd"] = round(total_cost_usd, 6)
        finops_agg["total_ai_cost_usd"] = round(total_ai_cost_usd, 6)
        finops_agg["avg_cost_per_run_usd"] = round(total_cost_usd / cost_count, 6)
        finops_agg["runs_with_cost_data"] = cost_count
    if token_count > 0:
        finops_agg["total_retry_waste_tokens"] = total_retry_waste_tokens
        finops_agg["runs_with_token_data"] = token_count
    if finops_agg:
        summary["finops_aggregate"] = finops_agg

    aiops_agg: dict[str, Any] = {}
    if ops_retry_counts:
        aiops_agg["avg_retry_count"] = round(sum(ops_retry_counts) / len(ops_retry_counts), 2)
        aiops_agg["total_wasted_time_ms"] = sum(ops_wasted_time)
        aiops_agg["automation_level_distribution"] = automation_levels
        full_auto = automation_levels.get("full", 0)
        aiops_agg["full_automation_rate"] = round(full_auto / len(ops_retry_counts), 4)
    if total_duration_ms_list:
        aiops_agg["avg_duration_ms"] = round(sum(total_duration_ms_list) / len(total_duration_ms_list), 1)
        aiops_agg["p95_duration_ms"] = _percentile(total_duration_ms_list, 95)
    if aiops_agg:
        summary["aiops_aggregate"] = aiops_agg

    return summary


def _percentile(values: list[int], pct: int) -> int:
    """Compute the given percentile from a sorted list of ints."""
    if not values:
        return 0
    sorted_vals = sorted(values)
    idx = int(len(sorted_vals) * pct / 100)
    idx = min(idx, len(sorted_vals) - 1)
    return sorted_vals[idx]


def write_summary(summary: dict[str, Any], output: str | None, audit_dir: Path) -> Path | None:
    """Write the summary JSON. If output is '-', print to stdout only."""
    text = json.dumps(summary, indent=2, ensure_ascii=False)
    if output == "-":
        print(text)
        return None
    if output:
        out_path = Path(output)
    else:
        audit_dir.mkdir(parents=True, exist_ok=True, mode=0o700)
        stamp = datetime.now(UTC).strftime("%Y%m%d-%H%M%S")
        out_path = audit_dir / f"gcl-quality-summary-{stamp}.json"
    out_path.write_text(text + "\n", encoding="utf-8")
    return out_path


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Aggregate GCL trace files into a quality summary for SLO evaluation.",
    )
    parser.add_argument(
        "--since-hours",
        type=float,
        default=None,
        help="Only include traces from the last N hours (default: all).",
    )
    parser.add_argument(
        "--all",
        action="store_true",
        dest="include_all",
        help="Include all traces regardless of age (overrides --since-hours).",
    )
    parser.add_argument(
        "--output",
        default=None,
        help="Output path. Use '-' for stdout only. Default: auto-timestamped file in audit-results/.",
    )
    parser.add_argument(
        "--audit-dir",
        type=Path,
        default=None,
        help="Override audit-results directory (default: <repo-root>/audit-results).",
    )
    args = parser.parse_args(argv)

    audit_dir = args.audit_dir or AUDIT_DIR
    since_hours = None if args.include_all else args.since_hours

    traces = collect_traces(audit_dir, since_hours)
    if not traces:
        print("WARNING: No GCL trace files found.", file=sys.stderr)
        # Still produce an empty summary for downstream compatibility
        summary = aggregate([])
    else:
        summary = aggregate(traces)
        print(f"Aggregated {len(traces)} trace(s).", file=sys.stderr)

    out_path = write_summary(summary, args.output, audit_dir)
    if out_path:
        print(f"[aggregate] wrote {out_path}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
