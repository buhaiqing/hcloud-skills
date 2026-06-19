#!/usr/bin/env python3
"""Scan GCL trace files for credential leaks."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

from gcl_security_scan import scan_payload

DEFAULT_GLOB = "audit-results/gcl-trace-*.json"


def collect_trace_paths(root: Path, inputs: list[Path] | None, latest: bool) -> list[Path]:
    if inputs:
        return [path for path in inputs if path.is_file()]
    paths = sorted(root.glob(DEFAULT_GLOB))
    if latest and paths:
        return [paths[-1]]
    return paths


def scan_traces(root: Path, inputs: list[Path] | None, latest: bool) -> list[dict[str, Any]]:
    trace_paths = collect_trace_paths(root, inputs, latest)
    results: list[dict[str, Any]] = []
    for trace_path in trace_paths:
        try:
            payload = json.loads(trace_path.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, OSError) as exc:
            results.append({"trace": str(trace_path), "ok": False, "findings": [], "error": str(exc)})
            continue
        findings = scan_payload(payload)
        try:
            display = str(trace_path.relative_to(root))
        except ValueError:
            display = str(trace_path)
        results.append({"trace": display, "ok": not findings, "findings": findings})
    return results


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument(
        "--latest", action="store_true", help="Scan only the latest trace when no explicit trace is given"
    )
    parser.add_argument("--allow-empty", action="store_true", help="Return success when no trace files exist")
    parser.add_argument("--json", action="store_true")
    parser.add_argument("trace", nargs="*", type=Path)
    return parser


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    inputs = [root / path if not path.is_absolute() else path for path in args.trace]
    results = scan_traces(root, inputs or None, args.latest)
    if not results and args.allow_empty:
        if args.json:
            print(json.dumps({"ok": True, "results": []}, indent=2, ensure_ascii=False))
        else:
            print("OK: no GCL trace files found")
        return 0

    ok = bool(results) and all(result["ok"] for result in results)
    if args.json:
        print(json.dumps({"ok": ok, "results": results}, indent=2, ensure_ascii=False))
    else:
        if not results:
            print("FAIL: no GCL trace files found")
        for result in results:
            status = "OK" if result["ok"] else "FAIL"
            print(f"{status}: {result['trace']}")
            for finding in result.get("findings", []):
                print(f"  - {finding['field']}: matched {finding['pattern']}")
            if "error" in result:
                print(f"  - error: {result['error']}")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
