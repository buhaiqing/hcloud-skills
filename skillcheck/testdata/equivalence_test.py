#!/usr/bin/env python3
"""Smoke test: skillcheck Go binary self-checks against embedded fixtures.

Python source scripts have been deleted (migrated to Go). This test validates
that skillcheck's core functionality works correctly using embedded fixtures
(--self-check) and deterministic inputs.

This does NOT test against the live repo — the repo may have pre-existing issues
that skillcheck correctly detects (those are validated by `make self-check`).

Exit code 0 = all checks pass; non-zero = at least one failure.
"""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SKILLCHECK = ROOT / "skillcheck" / "bin" / "skillcheck"
EMBED_FIXTURES = ROOT / "skillcheck" / "internal" / "embed" / "fixtures"


def skillcheck_binary() -> str:
    """Resolve skillcheck binary path, building if needed."""
    if not SKILLCHECK.exists():
        subprocess.run(
            ["go", "build", "-C", str(ROOT / "skillcheck"), "-trimpath", "-o", "bin/skillcheck", "."],
            check=True,
        )
    return str(SKILLCHECK)


def run_skillcheck(*args: str, binary: str) -> subprocess.CompletedProcess:
    """Run skillcheck with the given arguments."""
    return subprocess.run(
        [binary, *args],
        capture_output=True, text=True,
    )


# ---------------------------------------------------------------------------
# Test cases: (name, skillcheck_args, expect_success)
# ---------------------------------------------------------------------------
SMOKE_TESTS = [
    # Schema validation against embedded fixtures
    ("validate schema trace (fixture)",
     ["validate", "schema", "trace", "--file", str(EMBED_FIXTURES / "gcl-trace-healthy.json")],
     True),

    ("validate schema summary (fixture)",
     ["validate", "schema", "summary", "--file", str(EMBED_FIXTURES / "gcl-quality-summary-healthy.json")],
     True),

    ("validate schema alarm-plan (fixture)",
     ["validate", "schema", "alarm-plan", "--file", str(EMBED_FIXTURES / "gcl-alarm-plan-healthy.json")],
     True),

    # Self-check: secret scan against embedded fixtures
    ("scan secret trace --self-check",
     ["scan", "secret", "trace", "--self-check"],
     True),

    ("scan secret summary --self-check",
     ["scan", "secret", "summary", "--self-check"],
     True),

    ("scan secret alarm-plan --self-check",
     ["scan", "secret", "alarm-plan", "--self-check"],
     True),

    # --help should always work
    ("--help",
     ["--help"],
     True),
]


def main() -> int:
    parser = argparse.ArgumentParser(description="Smoke test: skillcheck binary")
    parser.add_argument("--binary", default=None, help="Path to skillcheck binary")
    args = parser.parse_args()

    binary = args.binary or skillcheck_binary()
    total_failures: list[str] = []
    total_tests = 0

    print(f"Using skillcheck binary: {binary}")
    print()

    for name, go_args, expect_success in SMOKE_TESTS:
        total_tests += 1
        go = run_skillcheck(*go_args, binary=binary)

        if expect_success and go.returncode != 0:
            total_failures.append(
                f"  {name}: expected success but got exit={go.returncode}\n"
                f"    stderr: {go.stderr[:300]}"
            )
            print(f"[FAIL] {name}")
        elif not expect_success and go.returncode == 0:
            total_failures.append(
                f"  {name}: expected failure but got exit=0"
            )
            print(f"[FAIL] {name}")
        else:
            print(f"[OK]   {name}")

    print()
    if total_failures:
        print(f"=== {len(total_failures)} failure(s) ===")
        for f in total_failures:
            print(f)
        return 1
    else:
        print(f"All {total_tests} smoke tests passed.")
        return 0


if __name__ == "__main__":
    sys.exit(main())