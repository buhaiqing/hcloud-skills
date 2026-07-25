#!/usr/bin/env python3
"""Run the local validation suite for hcloud-skills GCL gates."""

from __future__ import annotations

import argparse
import shlex
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class Step:
    name: str
    argv: tuple[str, ...]


def build_steps(root: Path | None = None) -> list[Step]:
    if root is None:
        root = Path(__file__).resolve().parents[1]
    skillcheck = str(root / "skillcheck" / "bin" / "skillcheck")
    return [
        # B-class checks (now in skillcheck CLI)
        Step(
            "GCL Tier-A conformance",
            (skillcheck, "validate", "gcl-conformance", "--root", str(root)),
        ),
        Step(
            "Generator GCL contract",
            (skillcheck, "validate", "generator-contract", "--root", str(root)),
        ),
        Step(
            "safety_class enum contract",
            (skillcheck, "validate", "safety-class", "--root", str(root)),
        ),
        Step(
            "resource_scope PII contract",
            (skillcheck, "validate", "resource-scope", "--root", str(root)),
        ),
        Step(
            "gcl_quality wiring contract",
            (skillcheck, "validate", "alarm-wire-contract", "--root", str(root)),
        ),
        Step(
            "audit-results gitignore guard",
            (skillcheck, "check", "audit-results", "--root", str(root)),
        ),
        Step(
            "skill_generator drift guard",
            ("python3", str(root / "scripts/check_skill_generator_drift.py"), "check"),
        ),
        # Runtime GCL components (skillcheck gcl subcommands)
        Step(
            "GCL runner smoke test",
            (
                skillcheck,
                "gcl",
                "run",
                "--root",
                str(root / "huaweicloud-billing-ops"),
                "--quiet",
            ),
        ),
        Step(
            "GCL trace aggregate",
            (
                "python3",
                str(root / "scripts/gcl_trace_aggregate.py"),
                "--all",
                "--output",
                "-",
            ),
        ),
        Step(
            "GCL alarm wire plan",
            (
                skillcheck,
                "gcl",
                "alarm-wire",
                "plan",
                "--summary",
                str(root / "scripts/fixtures/gcl-quality-summary-healthy.json"),
                "--write-plan",
            ),
        ),
        # Markdown link integrity
        Step(
            "Markdown local links",
            ("python3", str(root / "scripts/check_markdown_links.py")),
        ),
        # SKILL.md frontmatter structural lint (catches dangling/orphan YAML lines)
        Step(
            "SKILL.md frontmatter lint",
            ("python3", str(root / "scripts/check_skill_frontmatter.py"), "--root", str(root)),
        ),
        # Python unit tests (scripts/*_test.py) — hermetic, run unconditionally
        Step(
            "Python unit tests (scripts/*_test.py)",
            (
                "python3",
                "-m",
                "unittest",
                "discover",
                "-s",
                str(root / "scripts"),
                "-p",
                "*_test.py",
                "-v",
            ),
        ),
        # skillcheck equivalence test
        Step(
            "skillcheck smoke test (embedded fixtures)",
            ("python3", str(root / "skillcheck/testdata/equivalence_test.py")),
        ),
    ]


def run_step(root: Path, step: Step) -> int:
    print(f"\n==> {step.name}")
    print("$ " + shlex.join(step.argv))
    proc = subprocess.run(step.argv, cwd=root)
    return proc.returncode


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root", type=Path, default=Path(__file__).resolve().parents[1]
    )
    parser.add_argument(
        "--list", action="store_true", help="Print commands without running them"
    )
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    root = args.root.resolve()
    steps = build_steps(root)

    if args.list:
        for step in steps:
            print(f"{step.name}: {shlex.join(step.argv)}")
        return 0

    for step in steps:
        rc = run_step(root, step)
        if rc != 0:
            print(f"\nFAILED: {step.name} exited with {rc}", file=sys.stderr)
            return rc

    print("\nOK: local validation suite passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
