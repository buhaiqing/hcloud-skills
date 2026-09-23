"""Reflection findings CLI: append structured findings, propose rubric items.

Schema (frozen, one JSON object per line):
    {"ts": "RFC3339", "round": 1, "finding_type": "kebab-case-id",
     "detail": "one-line", "source_skill": "huaweicloud-*-ops or repo",
     "rubric_item": "matched rubric item id or null",
     "proposed_action": "add-rubric-item|reword|drop"}

A finding_type that appears >= --min-hits times AND has rubric_item == null
qualifies for a rubric auto-proposal. Existing rubric items are filtered out
when --rubric is supplied.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from collections import Counter
from datetime import datetime
from pathlib import Path

REQUIRED_FIELDS = (
    "ts",
    "round",
    "finding_type",
    "detail",
    "source_skill",
    "rubric_item",
    "proposed_action",
)
ALLOWED_ACTIONS = {"add-rubric-item", "reword", "drop"}
Kebab_RE = re.compile(r"^[a-z][a-z0-9]*(-[a-z0-9]+)*$")


class SchemaError(ValueError):
    """Raised when a finding row violates the frozen schema."""


def validate(row: object) -> dict:
    """Validate a parsed JSON object against the findings schema.

    Raises SchemaError on missing fields, wrong types, or bad enums.
    """
    if not isinstance(row, dict):
        raise SchemaError(f"row must be a JSON object, got {type(row).__name__}")
    missing = [f for f in REQUIRED_FIELDS if f not in row]
    if missing:
        raise SchemaError(f"missing fields: {missing}")
    if not isinstance(row["ts"], str):
        raise SchemaError(f"ts must be string, got {type(row['ts']).__name__}")
    try:
        datetime.fromisoformat(row["ts"].replace("Z", "+00:00"))
    except ValueError as e:
        raise SchemaError(f"ts not RFC3339: {row['ts']!r} ({e})") from e
    if not isinstance(row["round"], int) or row["round"] not in (1, 2, 3):
        raise SchemaError(f"round must be 1, 2, or 3; got {row['round']!r}")
    ft = row["finding_type"]
    if not isinstance(ft, str) or not Kebab_RE.match(ft):
        raise SchemaError(
            f"finding_type must be kebab-case id, got {ft!r}"
        )
    if not isinstance(row["detail"], str) or not row["detail"].strip():
        raise SchemaError("detail must be non-empty string")
    if not isinstance(row["source_skill"], str) or not row["source_skill"].strip():
        raise SchemaError("source_skill must be non-empty string")
    if row["rubric_item"] is not None and not isinstance(row["rubric_item"], str):
        raise SchemaError("rubric_item must be string or null")
    if row["proposed_action"] not in ALLOWED_ACTIONS:
        raise SchemaError(
            f"proposed_action must be one of {sorted(ALLOWED_ACTIONS)}; "
            f"got {row['proposed_action']!r}"
        )
    return row


def load_jsonl(path: Path) -> list[dict]:
    """Load and validate every JSONL row; raises SchemaError on first bad row."""
    rows: list[dict] = []
    with path.open("r", encoding="utf-8") as fh:
        for lineno, raw in enumerate(fh, 1):
            line = raw.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError as e:
                raise SchemaError(f"line {lineno}: invalid JSON ({e})") from e
            try:
                rows.append(validate(obj))
            except SchemaError as e:
                raise SchemaError(f"line {lineno}: {e}") from e
    return rows


def append_finding(path: Path, finding_json: str) -> int:
    """Validate one finding and append as a single JSONL line. Exit code: 0 ok, 2 bad."""
    try:
        obj = json.loads(finding_json)
    except json.JSONDecodeError as e:
        print(f"schema error: --finding is not valid JSON: {e}", file=sys.stderr)
        return 2
    try:
        row = validate(obj)
    except SchemaError as e:
        print(f"schema error: {e}", file=sys.stderr)
        return 2
    # FIX-B3: rebuild the dict in REQUIRED_FIELDS order, dropping any extra
    # keys so the on-disk JSONL line is canonical regardless of caller input.
    extra = set(row.keys()) - set(REQUIRED_FIELDS)
    canonical = {k: row[k] for k in REQUIRED_FIELDS}
    if extra:
        print(
            f"warning: dropped extra keys on append: {sorted(extra)}",
            file=sys.stderr,
        )
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("a", encoding="utf-8") as fh:
        fh.write(json.dumps(canonical, ensure_ascii=False) + "\n")
    print(f"appended: {canonical['finding_type']} (round={canonical['round']})", file=sys.stderr)
    return 0


def _detail_for(finding_type: str, rows: list[dict]) -> str:
    """Return the most common detail string for a finding_type."""
    counts = Counter(r["detail"] for r in rows if r["finding_type"] == finding_type)
    return counts.most_common(1)[0][0]


def _proposal_action(rows: list[dict]) -> str:
    """Pick the most common proposed_action across the cluster; default add."""
    counts = Counter(r["proposed_action"] for r in rows)
    return counts.most_common(1)[0][0]


def _in_rubric(finding_type: str, rubric_text: str) -> bool:
    """Treat finding_type already covered if *any* of its long kebab tokens
    (>=4 chars) appear in rubric, or if finding_type itself is a substring of
    rubric text. Short tokens (<4 chars, e.g. "yam", "mas", "not") are
    intentionally ignored — they collide too easily with prose.
    """
    haystack = rubric_text.lower()
    ft_lower = finding_type.lower()
    if ft_lower in haystack:
        return True
    # Split on `-` so `no-credential-mask-table` → ['no','credential','mask','table'].
    tokens = [t for t in ft_lower.split("-") if t]
    long_tokens = [t for t in tokens if len(t) >= 4]
    if not long_tokens:
        return False
    return any(tok in haystack for tok in long_tokens)


def propose(
    rows: list[dict],
    min_hits: int,
    rubric_path: Path | None,
) -> list[dict]:
    """Aggregate finding_type counts and emit rubric-add proposals."""
    rubric_text = rubric_path.read_text(encoding="utf-8") if rubric_path else ""
    by_type: dict[str, list[dict]] = {}
    for r in rows:
        if r["rubric_item"] is not None:
            continue
        by_type.setdefault(r["finding_type"], []).append(r)
    proposals: list[dict] = []
    for finding_type, group in by_type.items():
        if len(group) < min_hits:
            continue
        if rubric_path is not None and _in_rubric(finding_type, rubric_text):
            continue
        timestamps = sorted(r["ts"] for r in group)
        proposals.append({
            "finding_type": finding_type,
            "hits": len(group),
            "evidence_ts": {"first": timestamps[0], "last": timestamps[-1]},
            "proposal": _detail_for(finding_type, rows),
            "proposed_action": _proposal_action(group),
        })
    proposals.sort(key=lambda p: (-p["hits"], p["finding_type"]))
    return proposals


def build_parser() -> argparse.ArgumentParser:
    """Build the argparse CLI."""
    p = argparse.ArgumentParser(
        prog="reflection_findings",
        description="Append reflection findings and propose rubric items.",
    )
    sub = p.add_subparsers(dest="cmd", required=True)

    a = sub.add_parser("append", help="append a single finding row")
    a.add_argument("--file", required=True, type=Path, help="JSONL findings path")
    a.add_argument("--finding", required=True, help="one finding as JSON string")

    pr = sub.add_parser("propose", help="aggregate findings into rubric proposals")
    pr.add_argument("--file", required=True, type=Path, help="JSONL findings path")
    pr.add_argument(
        "--min-hits",
        type=int,
        default=3,
        help="minimum occurrences to propose a new rubric item (default 3)",
    )
    pr.add_argument(
        "--rubric",
        type=Path,
        default=None,
        help="optional rubric file; finding_types already covered are skipped",
    )
    return p


def main(argv: list[str] | None = None) -> int:
    """CLI entry point. Returns process exit code."""
    args = build_parser().parse_args(argv)
    if args.cmd == "append":
        return append_finding(args.file, args.finding)
    if args.cmd == "propose":
        try:
            rows = load_jsonl(args.file)
        except (SchemaError, FileNotFoundError) as e:
            print(f"error: {e}", file=sys.stderr)
            return 2
        proposals = propose(rows, args.min_hits, args.rubric)
        json.dump(proposals, sys.stdout, ensure_ascii=False, indent=2)
        sys.stdout.write("\n")
        return 0
    return 2


if __name__ == "__main__":
    raise SystemExit(main())