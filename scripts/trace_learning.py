#!/usr/bin/env python3
"""Experience Learning Loop — aggregate GCL traces into failure knowledge base.

Reads audit-results/gcl-trace-*.json, extracts failure patterns, clusters and
deduplicates them, then writes back to each skill's assets/failure_patterns.json.

Usage:
  python3 scripts/trace_learning.py aggregate --skill huaweicloud-ecs-ops [--since-hours 168] [--dry-run]
  python3 scripts/trace_learning.py learn --skill huaweicloud-ecs-ops --trace <path>
  python3 scripts/trace_learning.py report --skill huaweicloud-ecs-ops
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

UTC = timezone.utc  # noqa: UP017

ROOT_DEFAULT = Path(__file__).resolve().parents[1]

# Categories aligned with gcl_runner._FAILURE_SIGNATURES + extended types
VALID_CATEGORIES = frozenset(
    {"cli_parameter", "runtime", "cross_skill", "permission", "resource_state", "network", "token_efficiency", "skill_generation"}
)


def _now_iso() -> str:
    return datetime.now(UTC).strftime("%Y-%m-%dT%H:%M:%SZ")


def load_failure_patterns(root: Path, skill: str) -> dict[str, Any]:
    """Load existing failure_patterns.json or return empty scaffold."""
    path = root / skill / "assets" / "failure_patterns.json"
    if path.exists():
        return json.loads(path.read_text(encoding="utf-8"))
    return {
        "$schema": "failure-patterns/v1",
        "skill_id": skill,
        "patterns": [],
        "meta": {"total_patterns": 0, "last_aggregation": None, "source_traces_analyzed": 0},
    }


def save_failure_patterns(root: Path, skill: str, data: dict[str, Any]) -> Path:
    """Write failure_patterns.json back to disk."""
    path = root / skill / "assets" / "failure_patterns.json"
    path.parent.mkdir(parents=True, exist_ok=True)
    data["meta"]["total_patterns"] = len(data["patterns"])
    data["meta"]["last_aggregation"] = _now_iso()
    path.write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    return path


def _extract_pattern_from_trace(trace: dict[str, Any]) -> dict[str, Any] | None:
    """Extract failure_pattern from a GCL trace's final block."""
    final = trace.get("final")
    if not final:
        return None
    fp = final.get("failure_pattern")
    if not fp:
        return None
    return fp


def _trace_timestamp(trace: dict[str, Any]) -> datetime | None:
    """Best-effort parse of trace creation time from iterations or final."""
    # Traces don't have a top-level timestamp; infer from file mtime externally.
    # Return None to signal "use file mtime".
    return None


def _make_pattern_id(skill: str, existing: list[dict[str, Any]]) -> str:
    """Generate next sequential pattern ID."""
    prefix = skill.replace("huaweicloud-", "").replace("-ops", "").upper()
    max_num = 0
    for p in existing:
        pid = p.get("id", "")
        m = re.search(r"(\d+)$", pid)
        if m:
            max_num = max(max_num, int(m.group(1)))
    return f"{prefix}-FP{max_num + 1:03d}"


def _signature_key(fp: dict[str, Any]) -> tuple[str, str, str]:
    """Dedup key: (category, error_code/error, command_pattern)."""
    category = fp.get("category", "unknown")
    error = fp.get("error", "")
    command = fp.get("command", "") or ""
    # Normalize command to first meaningful token
    cmd_key = command.split()[0] if command.split() else ""
    return (category, error, cmd_key)


def _merge_pattern(existing: dict[str, Any], new_fp: dict[str, Any], trace_file: str) -> None:
    """Merge a new failure observation into an existing pattern entry."""
    stats = existing.setdefault("stats", {})
    stats["occurrence_count"] = stats.get("occurrence_count", 0) + 1
    stats["last_seen"] = _now_iso()
    if not stats.get("first_seen"):
        stats["first_seen"] = _now_iso()
    learned = existing.setdefault("learned_from", [])
    if trace_file not in learned:
        learned.append(trace_file)
    # Update fix suggestion if new one is more specific
    new_fix = new_fp.get("fix", "")
    if new_fix and len(new_fix) > len(existing.get("fix", {}).get("action", "")):
        existing.setdefault("fix", {})["action"] = new_fix[:200]


def _create_pattern_entry(
    fp: dict[str, Any], skill: str, existing: list[dict[str, Any]], trace_file: str
) -> dict[str, Any]:
    """Create a new pattern entry from a raw failure_pattern."""
    category = fp.get("category", "runtime")
    if category not in VALID_CATEGORIES:
        category = "runtime"
    return {
        "id": _make_pattern_id(skill, existing),
        "category": category,
        "signature": {
            "error_code": "",
            "error_message_regex": fp.get("error", ""),
            "command_pattern": (fp.get("command") or "").split()[0] if fp.get("command") else "",
        },
        "root_cause": fp.get("fix", "Unknown — pending analysis")[:200],
        "fix": {
            "strategy": "retry" if category in {"cli_parameter", "runtime"} else "halt",
            "action": fp.get("fix", "")[:200],
            "playbook_ref": None,
        },
        "prevention": "",
        "stats": {
            "occurrence_count": 1,
            "first_seen": _now_iso(),
            "last_seen": _now_iso(),
            "auto_fixed_count": 0,
            "escalated_count": 0,
            "success_rate": 0.0,
        },
        "learned_from": [trace_file],
    }


def scan_traces(root: Path, skill: str, since_hours: int | None) -> list[tuple[Path, dict[str, Any]]]:
    """Scan audit-results for GCL traces matching skill, filtered by age."""
    traces_dir = root / "audit-results"
    if not traces_dir.exists():
        return []
    results: list[tuple[Path, dict[str, Any]]] = []
    now = datetime.now(UTC)
    for f in sorted(traces_dir.glob("gcl-trace-*.json")):
        try:
            data = json.loads(f.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, OSError):
            continue
        if data.get("skill") != skill:
            continue
        if since_hours is not None:
            mtime = datetime.fromtimestamp(f.stat().st_mtime, tz=UTC)
            age_hours = (now - mtime).total_seconds() / 3600
            if age_hours > since_hours:
                continue
        results.append((f, data))
    return results


def cmd_aggregate(args: argparse.Namespace) -> int:
    """Aggregate all traces into failure_patterns.json."""
    root: Path = args.root
    skill: str = args.skill
    since_hours: int | None = args.since_hours
    dry_run: bool = args.dry_run

    traces = scan_traces(root, skill, since_hours)
    if not traces:
        print(f"No traces found for skill={skill}" + (f" within {since_hours}h" if since_hours else ""))
        return 0

    data = load_failure_patterns(root, skill)
    patterns = data["patterns"]
    existing_keys: dict[tuple[str, str, str], dict[str, Any]] = {}
    for p in patterns:
        sig = p.get("signature", {})
        key = (p.get("category", ""), sig.get("error_message_regex", ""), sig.get("command_pattern", ""))
        existing_keys[key] = p

    new_count = 0
    updated_count = 0
    skipped_count = 0

    for trace_path, trace in traces:
        fp = _extract_pattern_from_trace(trace)
        if not fp:
            skipped_count += 1
            continue
        key = _signature_key(fp)
        trace_name = trace_path.name
        if key in existing_keys:
            _merge_pattern(existing_keys[key], fp, trace_name)
            updated_count += 1
        else:
            entry = _create_pattern_entry(fp, skill, patterns, trace_name)
            patterns.append(entry)
            existing_keys[key] = entry
            new_count += 1

    data["meta"]["source_traces_analyzed"] = data["meta"].get("source_traces_analyzed", 0) + len(traces)

    print(f"Traces scanned: {len(traces)}")
    print(f"  New patterns: {new_count}")
    print(f"  Updated patterns: {updated_count}")
    print(f"  Skipped (no failure): {skipped_count}")

    if dry_run:
        print("\n[DRY-RUN] No changes written.")
        return 0

    if new_count > 0 or updated_count > 0:
        out_path = save_failure_patterns(root, skill, data)
        print(f"\nWritten: {out_path}")
    else:
        print("\nNo changes to write.")
    return 0


def cmd_learn(args: argparse.Namespace) -> int:
    """Learn from a single trace file."""
    root: Path = args.root
    skill: str = args.skill
    trace_path = Path(args.trace)
    if not trace_path.is_absolute():
        trace_path = root / trace_path

    if not trace_path.exists():
        print(f"ERROR: trace file not found: {trace_path}", file=sys.stderr)
        return 1

    trace = json.loads(trace_path.read_text(encoding="utf-8"))
    if trace.get("skill") != skill:
        print(f"WARN: trace skill={trace.get('skill')} != --skill {skill}; proceeding anyway", file=sys.stderr)

    fp = _extract_pattern_from_trace(trace)
    if not fp:
        print("No failure_pattern in trace (likely a PASS). Nothing to learn.")
        return 0

    data = load_failure_patterns(root, skill)
    patterns = data["patterns"]
    key = _signature_key(fp)

    # Check existing
    for p in patterns:
        sig = p.get("signature", {})
        pkey = (p.get("category", ""), sig.get("error_message_regex", ""), sig.get("command_pattern", ""))
        if pkey == key:
            _merge_pattern(p, fp, trace_path.name)
            print(f"Updated existing pattern: {p['id']}")
            break
    else:
        entry = _create_pattern_entry(fp, skill, patterns, trace_path.name)
        patterns.append(entry)
        print(f"Created new pattern: {entry['id']} ({entry['category']})")

    data["meta"]["source_traces_analyzed"] = data["meta"].get("source_traces_analyzed", 0) + 1

    if args.dry_run:
        print("[DRY-RUN] No changes written.")
        return 0

    out_path = save_failure_patterns(root, skill, data)
    print(f"Written: {out_path}")
    return 0


def cmd_report(args: argparse.Namespace) -> int:
    """Print knowledge base summary."""
    root: Path = args.root
    skill: str = args.skill
    data = load_failure_patterns(root, skill)
    patterns = data["patterns"]
    meta = data["meta"]

    print(f"=== Failure Knowledge Base: {skill} ===")
    print(f"Total patterns: {len(patterns)}")
    print(f"Last aggregation: {meta.get('last_aggregation', 'never')}")
    print(f"Source traces analyzed: {meta.get('source_traces_analyzed', 0)}")
    print()

    # By category
    by_cat: dict[str, int] = {}
    for p in patterns:
        cat = p.get("category", "unknown")
        by_cat[cat] = by_cat.get(cat, 0) + 1
    print("By category:")
    for cat, count in sorted(by_cat.items(), key=lambda x: -x[1]):
        print(f"  {cat}: {count}")
    print()

    # Top patterns by occurrence
    observed = [p for p in patterns if p.get("stats", {}).get("occurrence_count", 0) > 0]
    if observed:
        print("Top patterns (by occurrence):")
        for p in sorted(observed, key=lambda x: -x["stats"]["occurrence_count"])[:5]:
            stats = p["stats"]
            print(
                f"  {p['id']} [{p['category']}] "
                f"count={stats['occurrence_count']} "
                f"success_rate={stats.get('success_rate', 0):.0%} "
                f"last={stats.get('last_seen', '?')}"
            )
    else:
        print("No patterns observed yet (all seed data, 0 occurrences).")

    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Experience Learning Loop — GCL trace to failure knowledge base"
    )
    subparsers = parser.add_subparsers(dest="cmd", required=True)

    # aggregate
    agg = subparsers.add_parser("aggregate", help="Aggregate traces into failure_patterns.json")
    agg.add_argument("--root", type=Path, default=ROOT_DEFAULT, help="Repo root")
    agg.add_argument("--skill", required=True, help="Skill id, e.g. huaweicloud-ecs-ops")
    agg.add_argument("--since-hours", type=int, default=None, help="Only traces newer than N hours")
    agg.add_argument("--dry-run", action="store_true", help="Print results without writing")
    agg.set_defaults(func=cmd_aggregate)

    # learn
    learn = subparsers.add_parser("learn", help="Learn from a single trace file")
    learn.add_argument("--root", type=Path, default=ROOT_DEFAULT, help="Repo root")
    learn.add_argument("--skill", required=True, help="Skill id")
    learn.add_argument("--trace", required=True, help="Path to gcl-trace-*.json")
    learn.add_argument("--dry-run", action="store_true", help="Print without writing")
    learn.set_defaults(func=cmd_learn)

    # report
    rpt = subparsers.add_parser("report", help="Print knowledge base summary")
    rpt.add_argument("--root", type=Path, default=ROOT_DEFAULT, help="Repo root")
    rpt.add_argument("--skill", required=True, help="Skill id")
    rpt.set_defaults(func=cmd_report)

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
