#!/usr/bin/env python3
"""Validate GCL quality summary JSON files against the repository schema."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

from json_schema_subset import load_json, json_type, resolve_under_root, type_matches, validate_file, validate_value

DEFAULT_SCHEMA = Path("huaweicloud-ces-ops/assets/gcl-quality-summary.schema.json")
DEFAULT_SUMMARY = Path("scripts/fixtures/gcl-quality-summary-healthy.json")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--schema", type=Path, default=DEFAULT_SCHEMA)
    parser.add_argument("summary", nargs="*", type=Path, default=[DEFAULT_SUMMARY])
    parser.add_argument("--json", action="store_true")
    return parser


def validate_summaries(root: Path, schema: Path, summaries: list[Path]) -> list[dict[str, Any]]:
    schema_path = resolve_under_root(root, schema)
    results: list[dict[str, Any]] = []
    for raw_summary in summaries:
        summary_path = resolve_under_root(root, raw_summary)
        errors = validate_file(summary_path, schema_path)
        results.append({"summary": str(raw_summary), "ok": not errors, "errors": errors})
    return results


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    results = validate_summaries(root, args.schema, args.summary)
    ok = all(result["ok"] for result in results)
    if args.json:
        print(json.dumps({"ok": ok, "schema": str(args.schema), "results": results}, indent=2, ensure_ascii=False))
    else:
        for result in results:
            status = "OK" if result["ok"] else "FAIL"
            print(f"{status}: {result['summary']}")
            for error in result["errors"]:
                print(f"  - {error}")
    return 0 if ok else 1


__all__ = [
    "load_json",
    "json_type",
    "type_matches",
    "validate_file",
    "validate_value",
    "validate_summaries",
]


if __name__ == "__main__":
    sys.exit(main())
