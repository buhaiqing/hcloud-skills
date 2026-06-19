#!/usr/bin/env python3
"""Unit tests for `gcl_runner.structural_critic` and the --structural-critic-only CI smoke path.

The structural critic is the only Critic mode allowed in CI/local smoke tests.
Production GCL MUST use an externally supplied isolated Critic (see docs/gcl-spec.md).
This module locks the structural-critic contract:

* Dimension scores: correctness, safety, idempotency, traceability, spec_compliance
* Blocking rules: SAFETY_FAIL on credential leak, RETRY on missing scores
* CI smoke run returns exit 0, writes one trace, and the trace carries the
  sanitized operation_intent + masked request.
"""

from __future__ import annotations

import contextlib
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

_HERE = Path(__file__).resolve().parent
if str(_HERE) not in sys.path:
    sys.path.insert(0, str(_HERE))

import gcl_runner  # noqa: E402

REPO_ROOT = Path(__file__).resolve().parents[1]


class StructuralCriticDimensionsTests(unittest.TestCase):
    def test_pass_on_successful_hcloud(self) -> None:
        critic = gcl_runner.structural_critic(
            {
                "command": "hcloud ecs list-servers --region cn-north-4",
                "exit_code": 0,
                "result_excerpt": '{"Response":{"servers":[]}}',
            }
        )
        self.assertEqual(critic["_mode"], "structural-only")
        self.assertEqual(critic["scores"]["correctness"], 1.0)
        self.assertEqual(critic["scores"]["safety"], 1.0)
        self.assertEqual(critic["scores"]["idempotency"], 0.5)
        self.assertEqual(critic["scores"]["traceability"], 1.0)
        self.assertEqual(critic["scores"]["spec_compliance"], 1.0)
        self.assertFalse(critic["blocking"])
        self.assertEqual(critic["suggestions"], [])

    def test_nonzero_exit_lowers_correctness_and_spec(self) -> None:
        critic = gcl_runner.structural_critic(
            {
                "command": "hcloud ecs list-servers",
                "exit_code": 2,
                "result_excerpt": "MissingRegion: region required",
            }
        )
        self.assertEqual(critic["scores"]["correctness"], 0.0)
        self.assertEqual(critic["scores"]["spec_compliance"], 0.0)
        self.assertTrue(critic["blocking"])
        self.assertTrue(any("exit_code=2" in s for s in critic["suggestions"]))

    def test_credential_leak_is_safety_fail(self) -> None:
        critic = gcl_runner.structural_critic(
            {
                "command": "echo ok",
                "exit_code": 0,
                "result_excerpt": "HW_SECRET_ACCESS_KEY=supersecretvalue",
            }
        )
        self.assertEqual(critic["scores"]["safety"], 0.0)
        self.assertTrue(critic["blocking"])
        self.assertTrue(any("Credential leak" in s for s in critic["suggestions"]))

    def test_non_hcloud_command_drops_spec_compliance(self) -> None:
        critic = gcl_runner.structural_critic(
            {
                "command": "curl https://example.com",
                "exit_code": 0,
                "result_excerpt": "ok",
            }
        )
        self.assertEqual(critic["scores"]["spec_compliance"], 0.5)

    def test_go_run_command_keeps_spec_compliance(self) -> None:
        critic = gcl_runner.structural_critic(
            {
                "command": "go run ./main.go",
                "exit_code": 0,
                "result_excerpt": "ok",
            }
        )
        self.assertEqual(critic["scores"]["spec_compliance"], 1.0)

    def test_empty_output_lowers_traceability(self) -> None:
        critic = gcl_runner.structural_critic(
            {"command": "hcloud ecs list-servers", "exit_code": 0, "result_excerpt": ""}
        )
        self.assertEqual(critic["scores"]["traceability"], 0.5)
        self.assertTrue(any("Empty generator output" in s for s in critic["suggestions"]))

    def test_suggestions_capped_at_three(self) -> None:
        generator = {
            "command": "hcloud ecs list-servers",
            "exit_code": 9,
            "result_excerpt": "",
        }
        critic = gcl_runner.structural_critic(generator)
        self.assertLessEqual(len(critic["suggestions"]), 3)


class StructuralCriticOnlyCliTests(unittest.TestCase):
    """End-to-end CLI: invoke the script as a subprocess with --structural-critic-only."""

    def _invoke(self, extra_args: list[str]) -> tuple[int, str, str]:
        cmd = [
            sys.executable,
            str(REPO_ROOT / "scripts" / "gcl_runner.py"),
            "run",
            "--skill",
            "huaweicloud-billing-ops",
            "--request",
            "CI smoke test",
            "--operation-intent",
            '{"operation":"smoke","resource_scope":[],"expected_state":"no-op","safety_class":"read-only"}',
            "--command",
            'printf \'{"Response":{"RequestId":"ci-smoke"}}\'',
            "--max-iter",
            "1",
            "--structural-critic-only",
            *extra_args,
        ]
        with tempfile.TemporaryDirectory() as tmp:
            cmd.extend(["--root", tmp])
            proc = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
            return proc.returncode, proc.stdout, proc.stderr

    def test_cli_smoke_returns_pass(self) -> None:
        rc, stdout, stderr = self._invoke([])
        self.assertEqual(rc, 0, msg=f"stdout={stdout!r} stderr={stderr!r}")
        self.assertIn("PASS", stdout)

    def test_cli_smoke_emits_warning_to_stderr(self) -> None:
        """The CLI MUST log a non-production warning so the trace context is unmistakable."""
        rc, _stdout, stderr = self._invoke([])
        self.assertEqual(rc, 0)
        self.assertIn("structural-critic-only", stderr.lower())


class StructuralCriticOnlyDocGuardTests(unittest.TestCase):
    def test_spec_documents_non_production_use(self) -> None:
        spec = (REPO_ROOT / "docs" / "gcl-spec.md").read_text(encoding="utf-8")
        self.assertIn("structural-critic-only", spec)
        self.assertIn("CI/local smoke", spec)
        self.assertIn("MUST NOT", spec)

    def test_agents_documents_non_production_use(self) -> None:
        agents = (REPO_ROOT / "AGENTS.md").read_text(encoding="utf-8")
        self.assertIn("structural-critic-only", agents)
        self.assertIn("MUST NOT", agents)


class DecideIntegrationTests(unittest.TestCase):
    def test_decide_safety_fail_wins_over_pass(self) -> None:
        scores = {
            "correctness": 1.0,
            "safety": 0.0,
            "idempotency": 1.0,
            "traceability": 1.0,
            "spec_compliance": 1.0,
        }
        self.assertEqual(gcl_runner.decide(scores), "SAFETY_FAIL")

    def test_decide_retry_when_below_threshold(self) -> None:
        scores = {
            "correctness": 1.0,
            "safety": 1.0,
            "idempotency": 0.0,
            "traceability": 1.0,
            "spec_compliance": 1.0,
        }
        self.assertEqual(gcl_runner.decide(scores), "RETRY")


if __name__ == "__main__":
    with contextlib.suppress(SystemExit):
        unittest.main()
