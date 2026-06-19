#!/usr/bin/env python3
"""Unit tests for scripts/validate_gcl_alarm_plan_schema.py."""

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

import gcl_alarm_wire as gaw  # noqa: E402
import validate_gcl_alarm_plan_schema as vps  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]
SCHEMA = ROOT / "huaweicloud-ces-ops/assets/gcl-alarm-plan.schema.json"
FIXTURE = ROOT / "scripts/fixtures/gcl-alarm-plan-healthy.json"


def write_plan(root: Path, payload: dict, name: str = "gcl-alarm-plan-test.json") -> Path:
    audit = root / "audit-results"
    audit.mkdir(parents=True, exist_ok=True)
    path = audit / name
    path.write_text(json.dumps(payload), encoding="utf-8")
    return path


def healthy_plan() -> dict:
    return json.loads(FIXTURE.read_text(encoding="utf-8"))


def healthy_summary() -> dict:
    return {
        "cloud": "huaweicloud",
        "pass_rate": 0.95,
        "totals": {"PASS": 19, "MAX_ITER": 1, "SAFETY_FAIL": 0, "total_runs": 20},
    }


class SchemaValidationTests(unittest.TestCase):
    def test_fixture_validates_against_schema(self) -> None:
        self.assertEqual(vps.validate_plans(ROOT, SCHEMA, [FIXTURE], False)[0]["ok"], True)

    def test_alarm_wire_plan_validates_against_schema(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            summary_path = root / "audit-results" / "gcl-quality-summary-test.json"
            summary_path.parent.mkdir(parents=True, exist_ok=True)
            summary_path.write_text(json.dumps(healthy_summary()), encoding="utf-8")
            args = type(
                "Args", (), {"root": root, "summary": summary_path, "config": root / "missing.yaml", "write_plan": True}
            )()
            with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(gaw.cmd_plan(args), 0)
            persisted = sorted((root / "audit-results").glob("gcl-alarm-plan-*-plan.json"))
            self.assertEqual(len(persisted), 1)
            results = vps.validate_plans(root, SCHEMA, None, False)
            self.assertEqual(len(results), 1)
            self.assertTrue(results[0]["ok"], results[0]["errors"])

    def test_invalid_alarm_name_fails(self) -> None:
        plan = healthy_plan()
        plan["alarm_plan"][0]["name"] = "rogue-rule"
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_plan(root, plan)
            results = vps.validate_plans(root, SCHEMA, None, False)
            self.assertFalse(results[0]["ok"])
            self.assertTrue(any("name" in err for err in results[0]["errors"]))

    def test_invalid_severity_fails(self) -> None:
        plan = healthy_plan()
        plan["alarm_plan"][1]["severity"] = "FATAL"
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_plan(root, plan)
            results = vps.validate_plans(root, SCHEMA, None, False)
            self.assertFalse(results[0]["ok"])

    def test_missing_thresholds_fails(self) -> None:
        plan = healthy_plan()
        del plan["thresholds"]["max_iter_warn_count"]
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_plan(root, plan)
            results = vps.validate_plans(root, SCHEMA, None, False)
            self.assertFalse(results[0]["ok"])

    def test_additional_property_false_fails(self) -> None:
        plan = healthy_plan()
        plan["unknown"] = "nope"
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_plan(root, plan)
            results = vps.validate_plans(root, SCHEMA, None, False)
            self.assertFalse(results[0]["ok"])

    def test_bad_datetime_fails(self) -> None:
        plan = healthy_plan()
        plan["generated_at"] = "not-a-timestamp"
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_plan(root, plan)
            results = vps.validate_plans(root, SCHEMA, None, False)
            self.assertFalse(results[0]["ok"])

    def test_cloud_must_be_huaweicloud(self) -> None:
        plan = healthy_plan()
        plan["cloud"] = "aliyun"
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_plan(root, plan)
            results = vps.validate_plans(root, SCHEMA, None, False)
            self.assertFalse(results[0]["ok"])


class CliTests(unittest.TestCase):
    def test_main_allow_empty(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            old_argv = sys.argv
            try:
                sys.argv = ["validate_gcl_alarm_plan_schema.py", "--root", tmp, "--allow-empty"]
                with contextlib.redirect_stdout(io.StringIO()) as stdout:
                    rc = vps.main()
                self.assertEqual(rc, 0)
                self.assertIn("OK: no GCL alarm plan files found", stdout.getvalue())
            finally:
                sys.argv = old_argv

    def test_main_include_fixture_passes(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["validate_gcl_alarm_plan_schema.py", "--root", str(ROOT), "--include-fixture"]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = vps.main()
            self.assertEqual(rc, 0)
            self.assertIn(f"OK: {FIXTURE.relative_to(ROOT)}", stdout.getvalue())
        finally:
            sys.argv = old_argv

    def test_main_invalid_plan_returns_1(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            plan = healthy_plan()
            plan["alarm_plan"][0]["name"] = "rogue-rule"
            write_plan(root, plan)
            old_argv = sys.argv
            try:
                sys.argv = [
                    "validate_gcl_alarm_plan_schema.py",
                    "--root",
                    str(root),
                    "--schema",
                    str(SCHEMA),
                ]
                with contextlib.redirect_stdout(io.StringIO()):
                    rc = vps.main()
                self.assertEqual(rc, 1)
            finally:
                sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
