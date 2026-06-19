#!/usr/bin/env python3
"""Unit tests for scripts/check_gcl_summary_security.py."""

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

import check_gcl_summary_security as css  # noqa: E402
import gcl_trace_aggregate as gta  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]
FIXTURE = ROOT / "scripts/fixtures/gcl-quality-summary-healthy.json"


def healthy_summary() -> dict:
    return json.loads(FIXTURE.read_text(encoding="utf-8"))


def write_summary(root: Path, payload: dict, name: str = "gcl-quality-summary-test.json") -> Path:
    audit = root / "audit-results"
    audit.mkdir(parents=True, exist_ok=True)
    path = audit / name
    path.write_text(json.dumps(payload), encoding="utf-8")
    return path


class ScanTests(unittest.TestCase):
    def test_clean_fixture_passes(self) -> None:
        self.assertEqual(css.scan_payload(healthy_summary()), [])

    def test_aggregator_output_passes(self) -> None:
        trace = {
            "skill": "huaweicloud-ecs-ops",
            "iterations": [
                {
                    "critic": {
                        "scores": {
                            "correctness": 1,
                            "safety": 1,
                            "idempotency": 0.5,
                            "traceability": 1,
                            "spec_compliance": 1,
                        }
                    }
                }
            ],
            "final": {"status": "PASS"},
        }
        summary = gta.aggregate([trace])
        self.assertEqual(css.scan_payload(summary), [])

    def test_secret_in_trace_files_detected(self) -> None:
        summary = healthy_summary()
        summary["trace_files"] = ["audit-results/gcl-trace-x.json (HW_SECRET_ACCESS_KEY=secretvalue)"]
        findings = css.scan_payload(summary)
        self.assertTrue(any("HW_SECRET_ACCESS_KEY" in finding["pattern"] for finding in findings))

    def test_already_masked_field_passes(self) -> None:
        summary = healthy_summary()
        summary["trace_files"] = ["HW_SECRET_ACCESS_KEY=<masked>"]
        self.assertEqual(css.scan_payload(summary), [])

    def test_authorization_header_detected(self) -> None:
        summary = healthy_summary()
        summary["by_skill"]["huaweicloud-billing-ops"]["note"] = "Authorization: Bearer abcdefghijklmnopqrstuvwxyz1234567890"
        findings = css.scan_payload(summary)
        self.assertTrue(any("authorization_header" in finding["pattern"] for finding in findings))


class CliTests(unittest.TestCase):
    def test_main_allow_empty(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            old_argv = sys.argv
            try:
                sys.argv = ["check_gcl_summary_security.py", "--root", tmp, "--allow-empty"]
                with contextlib.redirect_stdout(io.StringIO()) as stdout:
                    rc = css.main()
                self.assertEqual(rc, 0)
                self.assertIn("OK: no GCL summary files found", stdout.getvalue())
            finally:
                sys.argv = old_argv

    def test_main_include_fixture_passes(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["check_gcl_summary_security.py", "--root", str(ROOT), "--include-fixture"]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = css.main()
            self.assertEqual(rc, 0)
            self.assertIn(f"OK: {FIXTURE.relative_to(ROOT)}", stdout.getvalue())
        finally:
            sys.argv = old_argv

    def test_main_detects_leak(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            bad = healthy_summary()
            bad["trace_files"] = ["HW_SECRET_ACCESS_KEY=supersecretvalue"]
            write_summary(root, bad)
            old_argv = sys.argv
            try:
                sys.argv = ["check_gcl_summary_security.py", "--root", str(root)]
                with contextlib.redirect_stdout(io.StringIO()) as stdout:
                    rc = css.main()
                self.assertEqual(rc, 1)
                self.assertIn("FAIL: audit-results/gcl-quality-summary-test.json", stdout.getvalue())
            finally:
                sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
