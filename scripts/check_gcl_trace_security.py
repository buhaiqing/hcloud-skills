#!/usr/bin/env python3
"""Scan GCL trace files for credential leaks.

The repository treats every GCL trace as a high-blast-radius artifact: trace
files are committed to `audit-results/` (in `.gitignore` on real runs, but
sometimes distributed) and inspected by humans during incident triage. Any
leaked secret (HW AK/SK, API key, Bearer token, password) immediately fails
this gate, mirroring the GCL safety-fail contract for destructive ops.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

from gcl_runner import SECRET_PATTERNS

DEFAULT_GLOB = "audit-results/gcl-trace-*.json"

SCANNED_FIELDS = {
    "request",
    "command",
    "result_excerpt",
    "operation",
    "user_request",
    "summary",
    "final_state",
    "raw_response",
}

EXTRA_PATTERNS: tuple[tuple[str, re.Pattern[str]], ...] = (
    ("bearer_token", re.compile(r"Bearer\s+[A-Za-z0-9._\-]{20,}", re.I)),
    ("authorization_header", re.compile(r"Authorization\s*[:=]\s*['\"]?[^\s'\"]+", re.I)),
    ("private_key_block", re.compile(r"-----BEGIN (?:RSA |EC |DSA |OPENSSH |PGP )?PRIVATE KEY-----")),
    ("password_assignment", re.compile(r"(?i)password\s*[:=]\s*['\"]?[^'\"\s]{6,}")),
    ("api_key_assignment", re.compile(r"(?i)(?:api[_-]?key|secret[_-]?key)\s*[:=]\s*['\"]?[A-Za-z0-9._\-/+=]{16,}")),
)

LONG_TOKEN_PATTERN = re.compile(r"\b[A-Za-z0-9._\-/+=]{40,}\b")
ALLOWED_TOKENS = {
    "<masked>",
    "gcl-trace",
    "gcl-quality-summary",
    "audit-results",
    "huaweicloud",
    "huaweicloud-sdk-go-v3",
}


def collect_trace_paths(root: Path, inputs: list[Path] | None, latest: bool) -> list[Path]:
    if inputs:
        return [path for path in inputs if path.is_file()]
    paths = sorted(root.glob(DEFAULT_GLOB))
    if latest and paths:
        return [paths[-1]]
    return paths


def _is_scanned_text(value: str, field: str) -> bool:
    return field in SCANNED_FIELDS or value and len(value) <= 200_000


def _strings_in(value: Any, prefix: str = "") -> list[tuple[str, str]]:
    out: list[tuple[str, str]] = []
    if isinstance(value, dict):
        for key, item in value.items():
            child = f"{prefix}.{key}" if prefix else str(key)
            if isinstance(item, str):
                out.append((child, item))
            else:
                out.extend(_strings_in(item, child))
    elif isinstance(value, list):
        for index, item in enumerate(value):
            child = f"{prefix}[{index}]"
            if isinstance(item, str):
                out.append((child, item))
            else:
                out.extend(_strings_in(item, child))
    return out


def _scan_text(text: str) -> list[str]:
    findings: list[str] = []
    if "<masked>" in text:
        return findings
    for pattern in SECRET_PATTERNS:
        if pattern.search(text):
            findings.append(pattern.pattern)
            continue
    for label, pattern in EXTRA_PATTERNS:
        if pattern.search(text):
            findings.append(f"extra:{label}")
    return findings


def scan_trace(payload: dict[str, Any]) -> list[dict[str, str]]:
    findings: list[dict[str, str]] = []
    for field, value in _strings_in(payload):
        if not _is_scanned_text(value, field):
            continue
        matched = _scan_text(value)
        for match in matched:
            findings.append({"field": field, "pattern": match})
    return findings


def scan_traces(root: Path, inputs: list[Path] | None, latest: bool) -> list[dict[str, Any]]:
    trace_paths = collect_trace_paths(root, inputs, latest)
    results: list[dict[str, Any]] = []
    for trace_path in trace_paths:
        try:
            payload = json.loads(trace_path.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, OSError) as exc:
            results.append({"trace": str(trace_path), "ok": False, "findings": [], "error": str(exc)})
            continue
        findings = scan_trace(payload)
        try:
            display = str(trace_path.relative_to(root))
        except ValueError:
            display = str(trace_path)
        results.append({"trace": display, "ok": not findings, "findings": findings})
    return results


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--latest", action="store_true", help="Scan only the latest trace when no explicit trace is given")
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
