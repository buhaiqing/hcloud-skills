#!/usr/bin/env python3
"""Smoke test: skillcheck Go binary self-checks against embedded fixtures and the live repo.

Python source scripts have been deleted (migrated to Go). This test validates
that skillcheck works correctly by running it against:
  1. Embedded fixtures (--self-check, schema validation)
  2. The live repository (frontmatter, product-assessment, etc.)

Exit code 0 = all checks pass; non-zero = at least one failure.
"""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SKILLCHECK = ROOT / "skillcheck" / "bin" / "skillcheck"
FIXTURES = ROOT / "scripts" / "fixtures"


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
# Test cases: (name, skillcheck_args)
# ---------------------------------------------------------------------------
SMOKE_TESTS = [
    # Schema validation against embedded fixtures
    ("validate schema trace (fixture)",
     ["validate", "schema", "trace", "--file", str(FIXTURES / "gcl-trace-healthy.json")]),

    ("validate schema summary (fixture)",
     ["validate", "schema", "summary", "--file", str(FIXTURES / "gcl-quality-summary-healthy.json")]),

    ("validate schema alarm-plan (fixture)",
     ["validate", "schema", "alarm-plan", "--file", str(FIXTURES / "gcl-alarm-plan-healthy.json")]),

    # Live repo checks
    ("validate eval-queries",
     ["validate", "eval-queries", "--root", str(ROOT)]),

    ("validate frontmatter",
     ["validate", "frontmatter", "--root", str(ROOT)]),

    ("validate product-assessment",
     ["validate", "product-assessment", "--root", str(ROOT)]),

    ("check example-config",
     ["check", "example-config", "--root", str(ROOT)]),

    ("check advanced-coverage",
     ["check", "advanced-coverage", "--root", str(ROOT)]),

    # Self-check: secret scan against embedded fixtures
    ("scan secret trace --self-check",
     ["scan", "secret", "trace", "--self-check"]),

    ("scan secret summary --self-check",
     ["scan", "secret", "summary", "--self-check"]),

    ("scan secret alarm-plan --self-check",
     ["scan", "secret", "alarm-plan", "--self-check"]),

    # Total entry point
    ("validate (total entry)",
     ["validate", "--root", str(ROOT)]),
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

    for name, go_args in SMOKE_TESTS:
        total_tests += 1
        go = run_skillcheck(*go_args, binary=binary)

        if go.returncode != 0:
            total_failures.append(
                f"  {name}: exit={go.returncode}\n"
                f"    stderr: {go.stderr[:300]}"
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