#!/usr/bin/env python3
"""Scan GCL CES alarm plan files for credential leaks.

The alarm plan file mirrors the same secret-leak concerns as the trace and
quality summary artifacts: a mistake that writes an `Authorization` header
or AK/SK into a `description` field would silently broadcast it via the
CI artifact. This gate reuses the shared scanner so all three artifacts
are held to one bar.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

from gcl_alarm_wire import PLAN_GLOB
from gcl_security_scan import scan_payload

DEFAULT_FIXTURE = "scripts/fixtures/gcl-alarm-plan-healthy.json"


def collect_plan_paths(root: Path, inputs: list[Path] | None, latest: bool) -> list[Path]:
    if inputs:
        return [path for path in inputs if path.is_file()]
    paths = sorted(root.glob(PLAN_GLOB))
    if latest and paths:
        return [paths[-1]]
    return paths


def scan_plans(root: Path, inputs: list[Path] | None, latest: bool) -> list[dict[str, Any]]:
    plan_paths = collect_plan_paths(root, inputs, latest)
    results: list[dict[str, Any]] = []
    for plan_path in plan_paths:
        try:
            payload = json.loads(plan_path.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, OSError) as exc:
            results.append({"plan": str(plan_path), "ok": False, "findings": [], "error": str(exc)})
            continue
        findings = scan_payload(payload)
        try:
            display = str(plan_path.relative_to(root))
        except ValueError:
            display = str(plan_path)
        results.append({"plan": display, "ok": not findings, "findings": findings})
    return results


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
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
    results = scan_plans(root, inputs or None, args.latest)
    if not results and args.allow_empty:
        if args.json:
            print(json.dumps({"ok": True, "results": []}, indent=2, ensure_ascii=False))
        else:
            print("OK: no GCL alarm plan files found")
        return 0

    ok = bool(results) and all(result["ok"] for result in results)
    if args.json:
        print(json.dumps({"ok": ok, "results": results}, indent=2, ensure_ascii=False))
    else:
        if not results:
            print("FAIL: no GCL alarm plan files found")
        for result in results:
            status = "OK" if result["ok"] else "FAIL"
            print(f"{status}: {result['plan']}")
            for finding in result.get("findings", []):
                print(f"  - {finding['field']}: matched {finding['pattern']}")
            if "error" in result:
                print(f"  - error: {result['error']}")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
