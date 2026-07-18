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


def build_steps(python: str = sys.executable) -> list[Step]:
    return [
        Step("Ruff Python lint", ("bash", "scripts/run_ruff.sh", ".")),
        Step("Python 3.10 syntax compat", (python, "scripts/check_py310_compat.py")),
        Step("Validate Markdown local links", (python, "scripts/check_markdown_links.py")),
        Step("references/ deep link health", (python, "scripts/check_references_link_health.py")),
        Step("eval_queries.json schema", (python, "scripts/validate_eval_queries_schema.py")),
        Step("SKILL.md frontmatter", (python, "scripts/validate_skills_frontmatter.py")),
        Step("Well-Architected worker JSON", (python, "scripts/validate_product_assessment.py")),
        Step("example-config.yaml anchors/placeholders", (python, "scripts/check_example_config.py", "--warn-only")),
        Step("references/advanced coverage (TE-7)", (python, "scripts/check_advanced_coverage.py")),
        Step("audit-results gitignore guard", (python, "scripts/check_audit_results_guard.py")),
        Step("gcl_quality wiring contract", (python, "scripts/check_gcl_alarm_wire_contract.py")),
        Step(
            "GCL structural-critic CI smoke",
            (
                python,
                "-m",
                "unittest",
                "discover",
                "-s",
                "scripts",
                "-p",
                "gcl_structural_critic_test.py",
                "-v",
            ),
        ),
        Step("safety_class enum contract", (python, "scripts/check_safety_class_enum.py")),
        Step("skill_generator drift guard", (python, "scripts/check_skill_generator_drift.py", "check")),
        Step("resource_scope PII contract", (python, "scripts/check_resource_scope_pii.py")),
        Step("Generator GCL contract", (python, "scripts/check_generator_contract.py")),
        Step(
            "GCL runner smoke test",
            (
                python,
                "scripts/gcl_runner.py",
                "run",
                "--skill",
                "huaweicloud-billing-ops",
                "--request",
                "CI smoke test",
                "--operation-intent",
                '{"operation":"smoke","resource_scope":[],"expected_state":"no-op","safety_class":"read-only"}',
                "--command",
                'printf "{\\"Response\\":{\\"RequestId\\":\\"ci-smoke\\"}}"',
                "--max-iter",
                "1",
                "--structural-critic-only",
            ),
        ),
        Step("GCL trace schema", (python, "scripts/validate_gcl_trace_schema.py", "--latest")),
        Step("GCL trace security", (python, "scripts/check_gcl_trace_security.py", "--latest")),
        Step("GCL trace aggregate", (python, "scripts/gcl_trace_aggregate.py", "--since-hours", "168")),
        Step("GCL quality summary schema", (python, "scripts/validate_gcl_summary_schema.py")),
        Step("GCL quality summary security", (python, "scripts/check_gcl_summary_security.py", "--include-fixture")),
        Step(
            "GCL alarm wire plan",
            (
                python,
                "scripts/gcl_alarm_wire.py",
                "plan",
                "--summary",
                "scripts/fixtures/gcl-quality-summary-healthy.json",
                "--write-plan",
            ),
        ),
        Step("GCL alarm plan schema", (python, "scripts/validate_gcl_alarm_plan_schema.py", "--include-fixture")),
        Step("GCL alarm plan security", (python, "scripts/check_gcl_alarm_plan_security.py", "--include-fixture")),
        Step("GCL Tier-A conformance", (python, "scripts/check_gcl_conformance.py")),
        Step("skillcheck equivalence (Python vs Go)", (python, "skillcheck/testdata/equivalence_test.py")),
    ]


def run_step(root: Path, step: Step) -> int:
    print(f"\n==> {step.name}")
    print("$ " + shlex.join(step.argv))
    proc = subprocess.run(step.argv, cwd=root)
    return proc.returncode


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--list", action="store_true", help="Print commands without running them")
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    root = args.root.resolve()
    steps = build_steps()

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
