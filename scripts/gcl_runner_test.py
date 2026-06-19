#!/usr/bin/env python3
"""Unit tests for scripts/gcl_runner.py."""

from __future__ import annotations

import contextlib
import io
import json
import sys
import tempfile
import unittest
from pathlib import Path

_HERE = Path(__file__).resolve().parent
if str(_HERE) not in sys.path:
    sys.path.insert(0, str(_HERE))

import gcl_runner  # noqa: E402


def quiet_cmd_run(ns) -> int:
    with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
        return gcl_runner.cmd_run(ns)


class SecretMaskingTests(unittest.TestCase):
    def test_mask_hw_secret_key(self) -> None:
        out = gcl_runner.mask_secrets("HW_SECRET_ACCESS_KEY=supersecretvalue")
        self.assertIn("HW_SECRET_ACCESS_KEY=<masked>", out)
        self.assertNotIn("supersecretvalue", out)

    def test_mask_secret_access_key(self) -> None:
        out = gcl_runner.mask_secrets("SecretAccessKey=abcdef1234567890abcdef123456")
        self.assertIn("SecretAccessKey=<masked>", out)

    def test_has_credential_leak_sk(self) -> None:
        self.assertTrue(gcl_runner.has_credential_leak("SK=abcdefghijklmnopqrstuvwxyz123456"))
        self.assertFalse(gcl_runner.has_credential_leak("SK=short"))

    def test_no_leak_when_already_masked(self) -> None:
        self.assertFalse(gcl_runner.has_credential_leak("HW_SECRET_ACCESS_KEY=<masked>"))


class OperationIntentTests(unittest.TestCase):
    def test_sanitize_operation_intent_masks_sensitive_keys(self) -> None:
        raw = json.dumps({"operation": "delete", "resource_scope": ["i-123"], "sk": "abcdefghijklmnopqrstuvwxyz123456"})
        intent = gcl_runner.sanitize_operation_intent(raw)
        self.assertEqual(intent["operation"], "delete")
        self.assertEqual(intent["sk"], "<masked>")

    def test_sanitize_invalid_json_returns_summary(self) -> None:
        intent = gcl_runner.sanitize_operation_intent("delete with HW_SECRET_ACCESS_KEY=secret")
        self.assertIn("summary", intent)
        self.assertNotIn("secret", intent["summary"])


class RunCommandTests(unittest.TestCase):
    def test_success(self) -> None:
        result = gcl_runner.run_command('printf "hello"')
        self.assertEqual(result["exit_code"], 0)
        self.assertIn("hello", result["result_excerpt"])
        self.assertEqual(result["stdout_len"], len("hello"))

    def test_failure_exit_code(self) -> None:
        result = gcl_runner.run_command("exit 7")
        self.assertEqual(result["exit_code"], 7)

    def test_stderr_captured(self) -> None:
        result = gcl_runner.run_command('printf "err" 1>&2')
        self.assertEqual(result["exit_code"], 0)
        self.assertIn("err", result["result_excerpt"])

    def test_timeout(self) -> None:
        result = gcl_runner.run_command("sleep 5", timeout=1)
        self.assertEqual(result["exit_code"], -1)
        self.assertIn("TIMEOUT", result["result_excerpt"])

    def test_command_secret_masked(self) -> None:
        result = gcl_runner.run_command('printf "HW_SECRET_ACCESS_KEY=shouldnotappear"')
        self.assertIn("<masked>", result["command"])
        self.assertNotIn("shouldnotappear", result["command"])


class StructuralCriticTests(unittest.TestCase):
    def test_passes_clean_hcloud_run(self) -> None:
        gen = {"exit_code": 0, "result_excerpt": '{"Response":{"RequestId":"x"}}', "command": "hcloud ecs list-servers"}
        critic = gcl_runner.structural_critic(gen)
        self.assertEqual(critic["scores"]["correctness"], 1.0)
        self.assertEqual(critic["scores"]["safety"], 1.0)
        self.assertEqual(critic["scores"]["spec_compliance"], 1.0)
        self.assertFalse(critic["blocking"])

    def test_spec_compliance_partial_for_non_hcloud_smoke_command(self) -> None:
        gen = {"exit_code": 0, "result_excerpt": "ok", "command": "printf ok"}
        critic = gcl_runner.structural_critic(gen)
        self.assertEqual(critic["scores"]["spec_compliance"], 0.5)

    def test_fails_on_nonzero_exit(self) -> None:
        gen = {"exit_code": 1, "result_excerpt": "InvalidParameter", "command": "hcloud ecs list-servers"}
        critic = gcl_runner.structural_critic(gen)
        self.assertEqual(critic["scores"]["correctness"], 0.0)
        self.assertTrue(critic["blocking"])

    def test_fails_on_credential_leak(self) -> None:
        gen = {
            "exit_code": 0,
            "result_excerpt": "SecretAccessKey=abcdefghijklmnopqrstuvwxyz123456",
            "command": "hcloud ecs list-servers",
        }
        critic = gcl_runner.structural_critic(gen)
        self.assertEqual(critic["scores"]["safety"], 0.0)
        self.assertTrue(critic["blocking"])


class CriticPayloadTests(unittest.TestCase):
    def ok_payload(self) -> dict:
        return {
            "scores": {"correctness": 1, "safety": 1, "idempotency": 1, "traceability": 1, "spec_compliance": 1},
            "suggestions": [],
            "blocking": False,
        }

    def test_load_from_file(self) -> None:
        with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as handle:
            json.dump(self.ok_payload(), handle)
            path = Path(handle.name)
        try:
            critic = gcl_runner.load_critic(path, stdin=False)
            self.assertEqual(critic["scores"]["correctness"], 1)
        finally:
            path.unlink()

    def test_validate_valid_payload(self) -> None:
        self.assertEqual(gcl_runner.validate_critic_payload(self.ok_payload()), [])

    def test_validate_missing_dimension(self) -> None:
        payload = self.ok_payload()
        del payload["scores"]["safety"]
        errors = gcl_runner.validate_critic_payload(payload)
        self.assertTrue(any("safety" in err for err in errors))

    def test_validate_invalid_score(self) -> None:
        payload = self.ok_payload()
        payload["scores"]["correctness"] = 0.7
        errors = gcl_runner.validate_critic_payload(payload)
        self.assertTrue(any("correctness" in err for err in errors))


class DecideTests(unittest.TestCase):
    def test_pass(self) -> None:
        scores = {"correctness": 1, "safety": 1, "idempotency": 1, "traceability": 1, "spec_compliance": 1}
        self.assertEqual(gcl_runner.decide(scores), "PASS")

    def test_retry(self) -> None:
        scores = {"correctness": 1, "safety": 1, "idempotency": 0, "traceability": 1, "spec_compliance": 1}
        self.assertEqual(gcl_runner.decide(scores), "RETRY")

    def test_safety_fail_overrides(self) -> None:
        scores = {"correctness": 1, "safety": 0, "idempotency": 1, "traceability": 1, "spec_compliance": 1}
        self.assertEqual(gcl_runner.decide(scores), "SAFETY_FAIL")


class CmdRunEndToEndTests(unittest.TestCase):
    def run_with(self, critic_payload: dict | None, structural: bool = False, max_iter: int = 2) -> tuple[int, Path]:
        root = Path(tempfile.mkdtemp())
        args = [
            "run",
            "--root", str(root),
            "--skill", "huaweicloud-ecs-ops",
            "--request", "CI smoke test",
            "--operation-intent", '{"operation":"smoke","resource_scope":[],"expected_state":"no-op","safety_class":"read-only"}',
            "--command", 'printf "{\\\"Response\\\":{\\\"RequestId\\\":\\\"ci-smoke\\\"}}"',
            "--max-iter", str(max_iter),
        ]
        if structural:
            args.append("--structural-critic-only")
        if critic_payload is not None:
            critic_file = root / "critic.json"
            critic_file.write_text(json.dumps(critic_payload), encoding="utf-8")
            args.extend(["--critic-json", str(critic_file)])
        ns = gcl_runner.build_parser().parse_args(args)
        return quiet_cmd_run(ns), root

    def test_structural_pass(self) -> None:
        rc, root = self.run_with(None, structural=True, max_iter=1)
        self.assertEqual(rc, 0)
        trace_files = list((root / "audit-results").glob("gcl-trace-*.json"))
        self.assertEqual(len(trace_files), 1)
        data = json.loads(trace_files[0].read_text(encoding="utf-8"))
        self.assertEqual(data["final"]["status"], "PASS")
        self.assertEqual(data["operation_intent"]["operation"], "smoke")

    def test_external_critic_safety_fail(self) -> None:
        critic = {
            "scores": {"correctness": 1, "safety": 0, "idempotency": 1, "traceability": 1, "spec_compliance": 1},
            "suggestions": ["mask secrets"],
            "blocking": True,
        }
        rc, _root = self.run_with(critic)
        self.assertEqual(rc, 3)

    def test_external_critic_max_iter(self) -> None:
        critic = {
            "scores": {"correctness": 1, "safety": 1, "idempotency": 0, "traceability": 1, "spec_compliance": 1},
            "suggestions": ["add idempotency token"],
            "blocking": True,
        }
        rc, root = self.run_with(critic, max_iter=2)
        self.assertEqual(rc, 1)
        data = json.loads(next((root / "audit-results").glob("gcl-trace-*.json")).read_text(encoding="utf-8"))
        self.assertEqual(data["final"]["status"], "MAX_ITER")
        self.assertIn("idempotency", data["final"]["unresolved"])


if __name__ == "__main__":
    unittest.main()
