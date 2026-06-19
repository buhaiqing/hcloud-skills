#!/usr/bin/env python3
"""Validate GCL CES alarm plan JSON files against the repository schema.

The alarm plan is rendered by `scripts/gcl_alarm_wire.py` (`plan` /
`apply --dry-run`) and persisted to `audit-results/gcl-alarm-plan-*.json`.
This validator enforces the structure that downstream CES automation
(manual `apply`, future GitOps) depends on, and reuses the subset JSON
Schema validator shared with the trace/summary schema checks.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

from gcl_alarm_wire import PLAN_GLOB
from json_schema_subset import resolve_under_root, validate_file

DEFAULT_SCHEMA = Path("huaweicloud-ces-ops/assets/gcl-alarm-plan.schema.json")
DEFAULT_FIXTURE = Path("scripts/fixtures/gcl-alarm-plan-healthy.json")


def collect_plan_paths(root: Path, inputs: list[Path] | None, latest: bool) -> list[Path]:
    if inputs:
        return [path for path in inputs if path.is_file()]
    paths = sorted(root.glob(PLAN_GLOB))
    if latest and paths:
        return [paths[-1]]
    return paths


def validate_plans(root: Path, schema: Path, inputs: list[Path] | None, latest: bool) -> list[dict[str, Any]]:
    schema_path = resolve_under_root(root, schema)
    plan_paths = collect_plan_paths(root, inputs, latest)
    results: list[dict[str, Any]] = []
    for raw_path in plan_paths:
        plan_path = resolve_under_root(root, raw_path)
        errors = validate_file(plan_path, schema_path)
        try:
            display = str(plan_path.relative_to(root))
        except ValueError:
            display = str(plan_path)
        results.append({"plan": display, "ok": not errors, "errors": errors})
    return results


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--schema", type=Path, default=DEFAULT_SCHEMA)
    parser.add_argument("--latest", action="store_true")
    parser.add_argument("--allow-empty", action="store_true")
    parser.add_argument("--include-fixture", action="store_true")
    parser.add_argument("--json", action="store_true")
    parser.add_argument("plan", nargs="*", type=Path)
    return parser


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    inputs: list[Path] = [root / path if not path.is_absolute() else path for path in args.plan]
    if not inputs and args.include_fixture:
        inputs.append(root / DEFAULT_FIXTURE)
    results = validate_plans(root, args.schema, inputs or None, args.latest)
    if not results and args.allow_empty:
        if args.json:
            print(json.dumps({"ok": True, "results": []}, indent=2, ensure_ascii=False))
        else:
            print("OK: no GCL alarm plan files found")
        return 0

    ok = bool(results) and all(result["ok"] for result in results)
    if args.json:
        print(json.dumps({"ok": ok, "schema": str(args.schema), "results": results}, indent=2, ensure_ascii=False))
    else:
        if not results:
            print("FAIL: no GCL alarm plan files found")
        for result in results:
            status = "OK" if result["ok"] else "FAIL"
            print(f"{status}: {result['plan']}")
            for error in result["errors"]:
                print(f"  - {error}")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
