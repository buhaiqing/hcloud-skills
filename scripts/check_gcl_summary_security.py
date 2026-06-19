#!/usr/bin/env python3
"""Scan GCL quality summary files for credential leaks.

Quality summaries aggregate trace metadata and are produced by
`gcl_trace_aggregate.py`. They SHOULD be free of secrets even though traces
are already masked, because callers occasionally re-export them to dashboards
or BI tools. This gate mirrors the trace security check and uses the same
shared secret patterns.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

from gcl_security_scan import scan_payload

DEFAULT_GLOB = "audit-results/gcl-quality-summary-*.json"
DEFAULT_FIXTURE = "scripts/fixtures/gcl-quality-summary-healthy.json"


def collect_summary_paths(root: Path, inputs: list[Path] | None, latest: bool) -> list[Path]:
    if inputs:
        return [path for path in inputs if path.is_file()]
    paths = sorted(root.glob(DEFAULT_GLOB))
    if latest and paths:
        return [paths[-1]]
    return paths


def scan_summaries(root: Path, inputs: list[Path] | None, latest: bool) -> list[dict[str, Any]]:
    summary_paths = collect_summary_paths(root, inputs, latest)
    results: list[dict[str, Any]] = []
    for summary_path in summary_paths:
        try:
            payload = json.loads(summary_path.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, OSError) as exc:
            results.append({"summary": str(summary_path), "ok": False, "findings": [], "error": str(exc)})
            continue
        findings = scan_payload(payload)
        try:
            display = str(summary_path.relative_to(root))
        except ValueError:
            display = str(summary_path)
        results.append({"summary": display, "ok": not findings, "findings": findings})
    return results


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--latest", action="store_true", help="Scan only the latest summary when no explicit path is given")
    parser.add_argument("--allow-empty", action="store_true", help="Return success when no summary files exist")
    parser.add_argument("--include-fixture", action="store_true", help="Also scan the healthy fixture (used in CI smoke)")
    parser.add_argument("--json", action="store_true")
    parser.add_argument("summary", nargs="*", type=Path)
    return parser


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    inputs: list[Path] = [root / path if not path.is_absolute() else path for path in args.summary]
    if not inputs and args.include_fixture:
        inputs.append(root / DEFAULT_FIXTURE)
    results = scan_summaries(root, inputs or None, args.latest)
    if not results and args.allow_empty:
        if args.json:
            print(json.dumps({"ok": True, "results": []}, indent=2, ensure_ascii=False))
        else:
            print("OK: no GCL summary files found")
        return 0

    ok = bool(results) and all(result["ok"] for result in results)
    if args.json:
        print(json.dumps({"ok": ok, "results": results}, indent=2, ensure_ascii=False))
    else:
        if not results:
            print("FAIL: no GCL summary files found")
        for result in results:
            status = "OK" if result["ok"] else "FAIL"
            print(f"{status}: {result['summary']}")
            for finding in result.get("findings", []):
                print(f"  - {finding['field']}: matched {finding['pattern']}")
            if "error" in result:
                print(f"  - error: {result['error']}")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
