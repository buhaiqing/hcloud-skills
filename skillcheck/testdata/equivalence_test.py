#!/usr/bin/env python3
"""Equivalence test: Python scripts vs skillcheck Go binary, same input → same exit code + same failure items.

Run from repo root:
  python3 skillcheck/testdata/equivalence_test.py [--binary path/to/skillcheck]

Skips tests whose Python counterpart requires runtime state (GCL traces, audit-results).
Tests only the A-class subset that is purely file-system based and hermetic.

Exit code 0 = all equivalence checks pass; non-zero = at least one mismatch.
"""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SKILLCHECK = ROOT / "skillcheck" / "bin" / "skillcheck"
SCRIPTS = ROOT / "scripts"
FIXTURES = SCRIPTS / "fixtures"
SCHEMAS = ROOT / "skillcheck" / "internal" / "embed" / "schemas"


def skillcheck_binary() -> str:
    """Resolve skillcheck binary path, building if needed."""
    if not SKILLCHECK.exists():
        subprocess.run(
            ["go", "build", "-C", str(ROOT / "skillcheck"), "-trimpath", "-o", "bin/skillcheck", "."],
            check=True,
        )
    return str(SKILLCHECK)


def run_python(script: str, *args: str) -> subprocess.CompletedProcess:
    """Run a Python script from scripts/ and return the result."""
    return subprocess.run(
        [sys.executable, str(SCRIPTS / script), *args],
        capture_output=True, text=True,
    )


def run_skillcheck(*args: str, binary: str) -> subprocess.CompletedProcess:
    """Run skillcheck with the given arguments."""
    return subprocess.run(
        [binary, *args],
        capture_output=True, text=True,
    )


def check_exit_code(py: subprocess.CompletedProcess, go: subprocess.CompletedProcess, name: str) -> list[str]:
    """Compare exit codes.

    Rule: Python fail → skillcheck must also fail (no false negatives).
          Python pass → skillcheck may pass or fail (Go may be stricter).
    """
    failures = []
    py_exit = py.returncode
    go_exit = go.returncode
    py_failed = py_exit != 0
    go_failed = go_exit != 0

    if py_failed and not go_failed:
        failures.append(
            f"  {name}: false negative — Python failed (exit={py_exit}) but skillcheck passed\n"
            f"    Python stderr: {py.stderr[:300]}"
        )
    elif not py_failed and go_failed:
        # Go is stricter — note it but don't fail; this is acceptable behavior.
        # The equivalence test only guarantees Python-level coverage.
        pass

    return failures


def check_failure_items(py: subprocess.CompletedProcess, go: subprocess.CompletedProcess, name: str) -> list[str]:
    """Compare failure items (lines starting with [FAIL])."""
    failures = []
    py_fails = {line.strip() for line in py.stdout.splitlines() if line.startswith("[FAIL]")}
    go_fails = {line.strip() for line in go.stdout.splitlines() if line.startswith("[FAIL]")}
    if py_fails != go_fails:
        only_py = py_fails - go_fails
        only_go = go_fails - py_fails
        parts = [f"  {name}: failure items mismatch"]
        if only_py:
            parts.append(f"    only in Python: {sorted(only_py)}")
        if only_go:
            parts.append(f"    only in skillcheck: {sorted(only_go)}")
        failures.append("\n".join(parts))
    return failures


# ---------------------------------------------------------------------------
# Test cases: (name, python_script, python_args, skillcheck_args)
# ---------------------------------------------------------------------------
EQUIVALENCE_TESTS = [
    # schema validation — each kind against its own healthy fixture
    ("validate schema trace",
     "validate_gcl_trace_schema.py", ["--file", str(FIXTURES / "gcl-trace-healthy.json")],
     ["validate", "schema", "trace", "--file", str(FIXTURES / "gcl-trace-healthy.json")]),

    ("validate schema summary",
     "validate_gcl_summary_schema.py", [str(FIXTURES / "gcl-quality-summary-healthy.json")],
     ["validate", "schema", "summary", "--file", str(FIXTURES / "gcl-quality-summary-healthy.json")]),

    ("validate schema alarm-plan",
     "validate_gcl_alarm_plan_schema.py", ["--include-fixture"],
     ["validate", "schema", "alarm-plan", "--file", str(FIXTURES / "gcl-alarm-plan-healthy.json")]),

    ("validate eval-queries",
     "validate_eval_queries_schema.py", [],
     ["validate", "eval-queries", "--root", str(ROOT)]),

    # frontmatter — validate SKILL.md files in repo
    ("validate frontmatter",
     "validate_skills_frontmatter.py", [],
     ["validate", "frontmatter", "--root", str(ROOT)]),

    # product-assessment
    ("validate product-assessment",
     "validate_product_assessment.py", [],
     ["validate", "product-assessment", "--root", str(ROOT)]),

    # check example-config
    ("check example-config",
     "check_example_config.py", ["--warn-only"],
     ["check", "example-config", "--root", str(ROOT)]),

    # check advanced-coverage
    ("check advanced-coverage",
     "check_advanced_coverage.py", [],
     ["check", "advanced-coverage", "--root", str(ROOT)]),

    # secret scan — self-check against embedded fixtures
    ("scan secret trace --self-check",
     "check_gcl_trace_security.py", ["--latest"],
     ["scan", "secret", "trace", "--self-check"]),

    ("scan secret summary --self-check",
     "check_gcl_summary_security.py", ["--include-fixture"],
     ["scan", "secret", "summary", "--self-check"]),

    ("scan secret alarm-plan --self-check",
     "check_gcl_alarm_plan_security.py", ["--include-fixture"],
     ["scan", "secret", "alarm-plan", "--self-check"]),
]


def main() -> int:
    parser = argparse.ArgumentParser(description="Equivalence test: Python vs skillcheck")
    parser.add_argument("--binary", default=None, help="Path to skillcheck binary")
    args = parser.parse_args()

    binary = args.binary or skillcheck_binary()
    total_failures: list[str] = []
    total_tests = 0

    print(f"Using skillcheck binary: {binary}")
    print(f"Python: {sys.executable}")
    print()

    for name, py_script, py_args, go_args in EQUIVALENCE_TESTS:
        total_tests += 1
        py = run_python(py_script, *py_args)
        go = run_skillcheck(*go_args, binary=binary)

        failures = check_exit_code(py, go, name)
        if not failures:
            failures = check_failure_items(py, go, name)

        if failures:
            total_failures.extend(failures)
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
        print(f"All {total_tests} equivalence tests passed.")
        return 0


if __name__ == "__main__":
    sys.exit(main())
