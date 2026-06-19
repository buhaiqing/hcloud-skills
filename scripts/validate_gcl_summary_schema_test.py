#!/usr/bin/env python3
"""Unit tests for scripts/validate_gcl_summary_schema.py."""

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

import validate_gcl_summary_schema as vgs  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]
SCHEMA = ROOT / "huaweicloud-ces-ops/assets/gcl-quality-summary.schema.json"
FIXTURE = ROOT / "scripts/fixtures/gcl-quality-summary-healthy.json"


def healthy_summary() -> dict:
    return json.loads(FIXTURE.read_text(encoding="utf-8"))


class TypeTests(unittest.TestCase):
    def test_json_type(self) -> None:
        self.assertEqual(vgs.json_type(None), "null")
        self.assertEqual(vgs.json_type(True), "boolean")
        self.assertEqual(vgs.json_type(1), "integer")
        self.assertEqual(vgs.json_type(1.5), "number")
        self.assertEqual(vgs.json_type("x"), "string")
        self.assertEqual(vgs.json_type([]), "array")
        self.assertEqual(vgs.json_type({}), "object")

    def test_integer_matches_number(self) -> None:
        self.assertTrue(vgs.type_matches(1, "number"))
        self.assertTrue(vgs.type_matches(None, ["number", "null"]))
        self.assertFalse(vgs.type_matches(True, "integer"))


class SchemaValidationTests(unittest.TestCase):
    def test_fixture_validates_against_schema(self) -> None:
        errors = vgs.validate_file(FIXTURE, SCHEMA)
        self.assertEqual(errors, [])

    def test_aggregator_output_validates_against_schema(self) -> None:
        import gcl_trace_aggregate as gta

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
            "_source_path": "audit-results/gcl-trace-test.json",
        }
        summary = gta.aggregate([trace])
        schema = vgs.load_json(SCHEMA)
        self.assertEqual(vgs.validate_value(summary, schema), [])

    def test_missing_required_property_fails(self) -> None:
        summary = healthy_summary()
        del summary["cloud"]
        schema = vgs.load_json(SCHEMA)
        errors = vgs.validate_value(summary, schema)
        self.assertTrue(any("missing required property 'cloud'" in error for error in errors))

    def test_const_mismatch_fails(self) -> None:
        summary = healthy_summary()
        summary["cloud"] = "qcloud"
        schema = vgs.load_json(SCHEMA)
        errors = vgs.validate_value(summary, schema)
        self.assertTrue(any("expected const 'huaweicloud'" in error for error in errors))

    def test_number_range_fails(self) -> None:
        summary = healthy_summary()
        summary["pass_rate"] = 1.1
        schema = vgs.load_json(SCHEMA)
        errors = vgs.validate_value(summary, schema)
        self.assertTrue(any("maximum 1" in error for error in errors))

    def test_additional_property_false_fails(self) -> None:
        summary = healthy_summary()
        summary["totals"]["UNKNOWN"] = 1
        schema = vgs.load_json(SCHEMA)
        errors = vgs.validate_value(summary, schema)
        self.assertTrue(any("additional property 'UNKNOWN'" in error for error in errors))

    def test_bad_datetime_fails(self) -> None:
        summary = healthy_summary()
        summary["generated_at"] = "not-a-date"
        schema = vgs.load_json(SCHEMA)
        errors = vgs.validate_value(summary, schema)
        self.assertTrue(any("date-time" in error for error in errors))


class CliTests(unittest.TestCase):
    def test_main_default_fixture(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["validate_gcl_summary_schema.py", "--root", str(ROOT)]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = vgs.main()
            self.assertEqual(rc, 0)
            self.assertIn("OK: scripts/fixtures/gcl-quality-summary-healthy.json", stdout.getvalue())
        finally:
            sys.argv = old_argv

    def test_main_invalid_summary_returns_1(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            schema_dir = root / "huaweicloud-ces-ops/assets"
            fixture_dir = root / "scripts/fixtures"
            schema_dir.mkdir(parents=True)
            fixture_dir.mkdir(parents=True)
            (schema_dir / "gcl-quality-summary.schema.json").write_text(SCHEMA.read_text(encoding="utf-8"), encoding="utf-8")
            bad = healthy_summary()
            bad["pass_rate"] = -0.1
            (fixture_dir / "bad.json").write_text(json.dumps(bad), encoding="utf-8")

            old_argv = sys.argv
            try:
                sys.argv = ["validate_gcl_summary_schema.py", "--root", str(root), "scripts/fixtures/bad.json"]
                with contextlib.redirect_stdout(io.StringIO()) as stdout:
                    rc = vgs.main()
                self.assertEqual(rc, 1)
                self.assertIn("FAIL: scripts/fixtures/bad.json", stdout.getvalue())
            finally:
                sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
