#!/usr/bin/env python3
"""Guard the audit-results/ gitignore contract.

`audit-results/` holds GCL traces, quality summaries, and alarm plans. The
runtime scripts persist sensitive operator data there (sanitized but still
operational telemetry). This script verifies:

1. `.gitignore` contains the four required patterns:
   - `audit-results/` and `**/audit-results/`
   - `gcl-trace-*.json` and `**/gcl-trace-*.json`
   - `gcl-quality-summary-*.json` and `**/gcl-quality-summary-*.json`
   - `gcl-alarm-plan-*.json` and `**/gcl-alarm-plan-*.json`
2. `audit-results/` directory exists at the repo root with `mode 0700`
   (owner-only) so traces don't leak via multi-user runners.
3. Any tracked file under `audit-results/` is reported as a violation that
   MUST be removed from git history (informational; we never auto-rewrite
   history).
4. `docs/gcl-spec.md` documents the audit persistence policy so future
   contributors do not bypass the gate.

Run from repo root or pass `--root`.
"""

from __future__ import annotations

import argparse
import json
import re
import stat
import subprocess
import sys
from pathlib import Path
from typing import Any

GITIGNORE_REQUIRED_PATTERNS: tuple[str, ...] = (
    r"^audit-results/?\s*$",
    r"^\*\*/audit-results/?\s*$",
    r"^gcl-trace-\*\.json\s*$",
    r"^\*\*/gcl-trace-\*\.json\s*$",
    r"^gcl-quality-summary-\*\.json\s*$",
    r"^\*\*/gcl-quality-summary-\*\.json\s*$",
    r"^gcl-alarm-plan-\*\.json\s*$",
    r"^\*\*/gcl-alarm-plan-\*\.json\s*$",
)
GCL_DOC_REQUIRED_FRAGMENTS: tuple[str, ...] = (
    "audit-results/",
    "GCL",
    "gitignore",
)


def read_gitignore(root: Path) -> list[str]:
    path = root / ".gitignore"
    if not path.is_file():
        return []
    return list(path.read_text(encoding="utf-8").splitlines())


def check_gitignore(root: Path) -> tuple[bool, list[str]]:
    lines = read_gitignore(root)
    errors: list[str] = []
    if not lines:
        errors.append(".gitignore missing")
        return False, errors
    for pattern in GITIGNORE_REQUIRED_PATTERNS:
        if not any(re.match(pattern, line) for line in lines):
            errors.append(f".gitignore missing pattern: {pattern!r}")
    return not errors, errors


def check_directory(root: Path) -> tuple[bool, list[str]]:
    errors: list[str] = []
    audit_dir = root / "audit-results"
    if not audit_dir.is_dir():
        errors.append(f"{audit_dir}: missing (GCL runtime scripts will create it on demand)")
        return False, errors
    mode = stat.S_IMODE(audit_dir.stat().st_mode)
    if mode & 0o077:
        errors.append(f"{audit_dir}: mode {oct(mode)} is too permissive; GCL traces should be owner-only (chmod 700)")
    return not errors, errors


def check_tracked_files(root: Path) -> tuple[bool, list[str]]:
    errors: list[str] = []
    proc = subprocess.run(
        ["git", "ls-files", "audit-results/"],
        cwd=root,
        check=False,
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        # Not a git checkout — nothing to check.
        return True, []
    tracked = [line.strip() for line in proc.stdout.splitlines() if line.strip()]
    if tracked:
        errors.append(
            f"audit-results/ contains {len(tracked)} tracked file(s); remove from git history (first 3: {tracked[:3]})"
        )
    return not errors, errors


def check_documents(root: Path) -> tuple[bool, list[str]]:
    errors: list[str] = []
    doc = root / "docs" / "gcl-spec.md"
    if not doc.is_file():
        errors.append(f"{doc}: missing GCL spec doc — audit persistence contract undocumented")
        return False, errors
    text = doc.read_text(encoding="utf-8")
    for fragment in GCL_DOC_REQUIRED_FRAGMENTS:
        if fragment.lower() not in text.lower():
            errors.append(f"{doc}: missing fragment {fragment!r}")
    return not errors, errors


def check_all(root: Path) -> dict[str, Any]:
    sections = {
        "gitignore": check_gitignore(root),
        "directory": check_directory(root),
        "tracked_files": check_tracked_files(root),
        "documents": check_documents(root),
    }
    all_errors: list[str] = []
    for name, (_ok, errors) in sections.items():
        for error in errors:
            all_errors.append(f"{name}: {error}")
    return {
        "ok": not all_errors,
        "sections": {name: {"ok": ok, "errors": errors} for name, (ok, errors) in sections.items()},
        "errors": all_errors,
    }


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--json", action="store_true")
    return parser


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    report = check_all(root)

    if args.json:
        print(json.dumps(report, indent=2, ensure_ascii=False))
    else:
        for name, section in report["sections"].items():
            status = "OK" if section["ok"] else "FAIL"
            print(f"{status}: {name}")
            for error in section["errors"]:
                print(f"  - {error}")
        if report["ok"]:
            print("\n[audit-results guard] OK")
        else:
            print(f"\n[audit-results guard] FAIL: {len(report['errors'])} issue(s)")

    return 0 if report["ok"] else 1


__all__ = [
    "GITIGNORE_REQUIRED_PATTERNS",
    "check_all",
    "check_directory",
    "check_documents",
    "check_gitignore",
    "check_tracked_files",
]


if __name__ == "__main__":
    sys.exit(main())
