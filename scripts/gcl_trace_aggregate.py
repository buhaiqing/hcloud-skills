#!/usr/bin/env python3
"""Aggregate GCL trace files into a Huawei Cloud quality summary.

Reads `audit-results/gcl-trace-*.json` or explicit `--input` paths and emits
`audit-results/gcl-quality-summary-YYYYMMDD-HHMMSS.json`.

Usage:
  python3 scripts/gcl_trace_aggregate.py
  python3 scripts/gcl_trace_aggregate.py --input audit-results/gcl-trace-*.json
  python3 scripts/gcl_trace_aggregate.py --since-hours 24
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any

FINAL_STATUSES = ("PASS", "SAFETY_FAIL", "MAX_ITER")
RUBRIC_DIMS = ("correctness", "safety", "idempotency", "traceability", "spec_compliance")


def parse_trace(path: Path) -> dict[str, Any] | None:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (json.JSONDecodeError, OSError) as exc:
        print(f"WARN: skip {path}: {exc}", file=sys.stderr)
        return None
    if "skill" not in data or "final" not in data:
        print(f"WARN: skip {path}: missing skill/final", file=sys.stderr)
        return None
    return data


def last_scores(trace: dict[str, Any]) -> dict[str, float]:
    iterations = trace.get("iterations") or []
    if not iterations:
        return {}
    return dict(iterations[-1].get("critic", {}).get("scores") or {})


def aggregate(traces: list[dict[str, Any]]) -> dict[str, Any]:
    by_skill: dict[str, dict[str, Any]] = {}
    totals = {status: 0 for status in FINAL_STATUSES}
    totals["total_runs"] = len(traces)
    score_sums: dict[str, float] = {dim: 0.0 for dim in RUBRIC_DIMS}
    score_count = 0

    for trace in traces:
        skill = trace.get("skill", "unknown")
        status = (trace.get("final") or {}).get("status", "UNKNOWN")
        if status in totals:
            totals[status] += 1

        bucket = by_skill.setdefault(
            skill,
            {"total": 0, "PASS": 0, "SAFETY_FAIL": 0, "MAX_ITER": 0, "avg_iterations": 0.0},
        )
        bucket["total"] += 1
        if status in bucket:
            bucket[status] += 1
        iterations = len(trace.get("iterations") or [])
        bucket["avg_iterations"] = (
            (bucket["avg_iterations"] * (bucket["total"] - 1) + iterations) / bucket["total"]
        )

        scores = last_scores(trace)
        if scores:
            score_count += 1
            for dim in RUBRIC_DIMS:
                score_sums[dim] += float(scores.get(dim, 0))

    total_runs = int(totals["total_runs"])
    pass_rate = totals["PASS"] / total_runs if total_runs else 0.0
    avg_scores = {
        dim: round(score_sums[dim] / score_count, 3) if score_count else None
        for dim in RUBRIC_DIMS
    }

    return {
        "version": "1.0",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "cloud": "huaweicloud",
        "metric_namespace": "CUSTOM.GCL",
        "window": {"trace_count": total_runs},
        "totals": totals,
        "pass_rate": round(pass_rate, 4),
        "avg_rubric_scores": avg_scores,
        "by_skill": by_skill,
        "trace_files": [trace.get("_source_path") for trace in traces],
    }


def collect_paths(root: Path, inputs: list[str] | None, since_hours: int | None) -> list[Path]:
    if inputs:
        output: list[Path] = []
        for pattern in inputs:
            output.extend(sorted(root.glob(pattern) if "*" in pattern else [Path(pattern)]))
        return [path for path in output if path.is_file()]

    audit_dir = root / "audit-results"
    if not audit_dir.is_dir():
        return []
    paths = sorted(audit_dir.glob("gcl-trace-*.json"))
    if since_hours is None:
        return paths
    cutoff = datetime.now(timezone.utc) - timedelta(hours=since_hours)
    return [path for path in paths if datetime.fromtimestamp(path.stat().st_mtime, tz=timezone.utc) >= cutoff]


def persist_summary(root: Path, summary: dict[str, Any]) -> Path:
    out_dir = root / "audit-results"
    out_dir.mkdir(parents=True, exist_ok=True)
    ts = datetime.now(timezone.utc).strftime("%Y%m%d-%H%M%S")
    path = out_dir / f"gcl-quality-summary-{ts}.json"
    path.write_text(json.dumps(summary, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    return path


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--input", nargs="*", help="Trace file(s) or glob under --root")
    parser.add_argument("--since-hours", type=int, default=None, help="Only traces modified within N hours")
    args = parser.parse_args()

    paths = collect_paths(args.root, args.input, args.since_hours)
    if not paths:
        print("No gcl-trace files found.", file=sys.stderr)
        return 1

    traces: list[dict[str, Any]] = []
    for path in paths:
        trace = parse_trace(path)
        if trace:
            trace["_source_path"] = str(path.relative_to(args.root))
            traces.append(trace)

    if not traces:
        print("No valid traces parsed.", file=sys.stderr)
        return 1

    summary = aggregate(traces)
    out = persist_summary(args.root, summary)
    print(json.dumps({"summary_path": str(out), "pass_rate": summary["pass_rate"], "total_runs": summary["totals"]["total_runs"]}))
    return 0


if __name__ == "__main__":
    sys.exit(main())
