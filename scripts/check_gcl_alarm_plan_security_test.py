#!/usr/bin/env python3
"""Unit tests for scripts/check_gcl_alarm_plan_security.py."""

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

import check_gcl_alarm_plan_security as cas  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]
FIXTURE = ROOT / "scripts/fixtures/gcl-alarm-plan-healthy.json"


def write_plan(root: Path, payload: dict, name: str = "gcl-alarm-plan-test.json") -> Path:
    audit = root / "audit-results"
    audit.mkdir(parents=True, exist_ok=True)
    path = audit / name
    path.write_text(json.dumps(payload), encoding="utf-8")
    return path


def healthy_plan() -> dict:
    return json.loads(FIXTURE.read_text(encoding="utf-8"))


class ScanTests(unittest.TestCase):
    def test_clean_fixture_passes(self) -> None:
        self.assertEqual(cas.scan_payload(healthy_plan()), [])

    def test_authorization_header_in_description_detected(self) -> None:
        plan = healthy_plan()
        plan["alarm_plan"][0]["description"] = "Authorization: Bearer abcdefghijklmnopqrstuvwxyz1234567890"
        findings = cas.scan_payload(plan)
        self.assertTrue(any("authorization_header" in finding["pattern"] for finding in findings))

    def test_hw_secret_key_in_threshold_detected(self) -> None:
        plan = healthy_plan()
        plan["alarm_plan"][0]["metric_name"] = "gcl_overall_pass_rate (HW_SECRET_ACCESS_KEY=verysecretvalue)"
        findings = cas.scan_payload(plan)
        self.assertTrue(any("HW_SECRET_ACCESS_KEY" in finding["pattern"] for finding in findings))

    def test_masked_description_passes(self) -> None:
        plan = healthy_plan()
        plan["alarm_plan"][0]["description"] = "HW_SECRET_ACCESS_KEY=<masked>"
        self.assertEqual(cas.scan_payload(plan), [])


class CliTests(unittest.TestCase):
    def test_main_allow_empty(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            old_argv = sys.argv
            try:
                sys.argv = ["check_gcl_alarm_plan_security.py", "--root", tmp, "--allow-empty"]
                with contextlib.redirect_stdout(io.StringIO()) as stdout:
                    rc = cas.main()
                self.assertEqual(rc, 0)
                self.assertIn("OK: no GCL alarm plan files found", stdout.getvalue())
            finally:
                sys.argv = old_argv

    def test_main_include_fixture_passes(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["check_gcl_alarm_plan_security.py", "--root", str(ROOT), "--include-fixture"]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = cas.main()
            self.assertEqual(rc, 0)
            self.assertIn(f"OK: {FIXTURE.relative_to(ROOT)}", stdout.getvalue())
        finally:
            sys.argv = old_argv

    def test_main_detects_leak(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            bad = healthy_plan()
            bad["alarm_plan"][0]["description"] = "Authorization: Bearer abcdefghijklmnopqrstuvwxyz1234567890"
            write_plan(root, bad)
            old_argv = sys.argv
            try:
                sys.argv = ["check_gcl_alarm_plan_security.py", "--root", str(root)]
                with contextlib.redirect_stdout(io.StringIO()):
                    rc = cas.main()
                self.assertEqual(rc, 1)
            finally:
                sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()