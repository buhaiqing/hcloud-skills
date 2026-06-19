#!/usr/bin/env python3
"""Validate GCL trace JSON files against the repository trace schema."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

from json_schema_subset import resolve_under_root, validate_file

DEFAULT_SCHEMA = Path("huaweicloud-ces-ops/assets/gcl-trace.schema.json")
DEFAULT_GLOB = "audit-results/gcl-trace-*.json"


def collect_trace_paths(root: Path, inputs: list[Path] | None, latest: bool) -> list[Path]:
    if inputs:
        paths = [resolve_under_root(root, path) for path in inputs]
        return [path for path in paths if path.is_file()]
    paths = sorted(root.glob(DEFAULT_GLOB))
    if latest and paths:
        return [paths[-1]]
    return paths


def validate_traces(root: Path, schema: Path, traces: list[Path] | None, latest: bool) -> list[dict[str, Any]]:
    schema_path = resolve_under_root(root, schema)
    trace_paths = collect_trace_paths(root, traces, latest)
    results: list[dict[str, Any]] = []
    for trace_path in trace_paths:
        errors = validate_file(trace_path, schema_path)
        try:
            display = str(trace_path.relative_to(root))
        except ValueError:
            display = str(trace_path)
        results.append({"trace": display, "ok": not errors, "errors": errors})
    return results


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--schema", type=Path, default=DEFAULT_SCHEMA)
    parser.add_argument(
        "--latest", action="store_true", help="Validate only latest trace when no explicit trace is given"
    )
    parser.add_argument("--allow-empty", action="store_true", help="Return success when no trace files exist")
    parser.add_argument("--json", action="store_true")
    parser.add_argument("trace", nargs="*", type=Path)
    return parser


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    results = validate_traces(root, args.schema, args.trace or None, args.latest)
    if not results and args.allow_empty:
        if args.json:
            print(json.dumps({"ok": True, "schema": str(args.schema), "results": []}, indent=2, ensure_ascii=False))
        else:
            print("OK: no GCL trace files found")
        return 0

    ok = bool(results) and all(result["ok"] for result in results)
    if args.json:
        print(json.dumps({"ok": ok, "schema": str(args.schema), "results": results}, indent=2, ensure_ascii=False))
    else:
        if not results:
            print("FAIL: no GCL trace files found")
        for result in results:
            status = "OK" if result["ok"] else "FAIL"
            print(f"{status}: {result['trace']}")
            for error in result["errors"]:
                print(f"  - {error}")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
