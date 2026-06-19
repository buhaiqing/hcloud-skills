#!/usr/bin/env python3
"""Unit tests for scripts/check_gcl_trace_security.py."""

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

import check_gcl_trace_security as cts  # noqa: E402
import gcl_runner  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]


def clean_trace() -> dict:
    return {
        "request": "CI smoke test",
        "iterations": [
            {
                "generator": {
                    "command": "printf ok",
                    "result_excerpt": "ok",
                },
                "critic": {
                    "scores": {
                        "correctness": 1,
                        "safety": 1,
                        "idempotency": 0.5,
                        "traceability": 1,
                        "spec_compliance": 1,
                    },
                },
            }
        ],
        "final": {"status": "PASS", "iter": 1, "output": "ok", "failure_pattern": None},
    }


def write_trace(root: Path, payload: dict, name: str = "gcl-trace-test.json") -> Path:
    audit = root / "audit-results"
    audit.mkdir(parents=True, exist_ok=True)
    path = audit / name
    path.write_text(json.dumps(payload), encoding="utf-8")
    return path


def find_finding(results, field: str) -> dict | None:
    for result in results:
        for finding in result.get("findings", []):
            if finding["field"] == field:
                return finding
    return None


class ScanTests(unittest.TestCase):
    def test_clean_trace_passes(self) -> None:
        self.assertEqual(cts.scan_payload(clean_trace()), [])

    def test_already_masked_value_passes(self) -> None:
        trace = clean_trace()
        trace["request"] = "HW_SECRET_ACCESS_KEY=<masked>"
        self.assertEqual(cts.scan_payload(trace), [])

    def test_hw_secret_key_leak_detected(self) -> None:
        trace = clean_trace()
        trace["request"] = "run with HW_SECRET_ACCESS_KEY=supersecretvalue"
        findings = cts.scan_payload(trace)
        self.assertTrue(any("HW_SECRET_ACCESS_KEY" in finding["pattern"] for finding in findings))

    def test_bearer_token_leak_detected(self) -> None:
        trace = clean_trace()
        trace["iterations"][0]["generator"]["result_excerpt"] = (
            "Authorization: Bearer abcdefghijklmnopqrstuvwxyz1234567890"
        )
        findings = cts.scan_payload(trace)
        self.assertTrue(any("extra:" in finding["pattern"] for finding in findings))

    def test_private_key_leak_detected(self) -> None:
        trace = clean_trace()
        trace["iterations"][0]["generator"]["result_excerpt"] = "-----BEGIN RSA PRIVATE KEY-----"
        findings = cts.scan_payload(trace)
        self.assertTrue(any("private_key_block" in finding["pattern"] for finding in findings))

    def test_password_leak_detected(self) -> None:
        trace = clean_trace()
        trace["request"] = "credentials: password=TopSecret1234"
        findings = cts.scan_payload(trace)
        self.assertTrue(any("password_assignment" in finding["pattern"] for finding in findings))


class RunnerIntegrationTests(unittest.TestCase):
    def test_runner_output_passes_scan(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            args = gcl_runner.build_parser().parse_args(
                [
                    "run",
                    "--root",
                    str(root),
                    "--skill",
                    "huaweicloud-ecs-ops",
                    "--request",
                    "CI smoke test",
                    "--operation-intent",
                    '{"operation":"smoke","resource_scope":[],"expected_state":"no-op","safety_class":"read-only"}',
                    "--command",
                    "printf ok",
                    "--max-iter",
                    "1",
                    "--structural-critic-only",
                ]
            )
            with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(gcl_runner.cmd_run(args), 0)
            trace_path = next((root / "audit-results").glob("gcl-trace-*.json"))
            results = cts.scan_traces(root, [trace_path], latest=False)
            self.assertTrue(results[0]["ok"], results[0])


class CliTests(unittest.TestCase):
    def test_main_allow_empty(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            old_argv = sys.argv
            try:
                sys.argv = ["check_gcl_trace_security.py", "--root", tmp, "--allow-empty"]
                with contextlib.redirect_stdout(io.StringIO()) as stdout:
                    rc = cts.main()
                self.assertEqual(rc, 0)
                self.assertIn("OK: no GCL trace files found", stdout.getvalue())
            finally:
                sys.argv = old_argv

    def test_main_detects_leak(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            trace = clean_trace()
            trace["request"] = "with HW_SECRET_ACCESS_KEY=supersecretvalue"
            write_trace(root, trace)
            old_argv = sys.argv
            try:
                sys.argv = ["check_gcl_trace_security.py", "--root", str(root)]
                with contextlib.redirect_stdout(io.StringIO()) as stdout:
                    rc = cts.main()
                self.assertEqual(rc, 1)
                self.assertIn("FAIL: audit-results/gcl-trace-test.json", stdout.getvalue())
            finally:
                sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
