#!/usr/bin/env python3
"""Unit tests for scripts/validate_eval_queries_schema.py."""

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

import validate_eval_queries_schema as veq  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]
SCHEMA = ROOT / "huaweicloud-skill-generator/assets/eval-queries.schema.json"


class RepoValidationTests(unittest.TestCase):
    def test_all_repo_eval_queries_validate(self) -> None:
        results = veq.validate_all(ROOT, SCHEMA, veq.discover_eval_files(ROOT))
        failures = [result for result in results if not result["ok"]]
        if failures:
            messages = "\n".join(f"{item['file']}: {item['errors']}" for item in failures)
            self.fail(f"eval_queries schema failures:\n{messages}")
        self.assertGreaterEqual(len(results), 20)


class FormatDetectionTests(unittest.TestCase):
    def test_activate_array_requires_both_polarities(self) -> None:
        schema = veq.load_eval_schema(SCHEMA)
        data = [{"query": "create ecs", "should_activate": True, "reason": "yes"}]
        errors = veq.validate_eval_document(data, schema, skill_name="huaweicloud-ecs-ops", path="test.json")
        self.assertTrue(any("should_activate=false" in error for error in errors))

    def test_match_object_skill_mismatch(self) -> None:
        schema = veq.load_eval_schema(SCHEMA)
        data = {
            "skill": "huaweicloud-ecs-ops",
            "should_match": [{"query": "q1", "reason": "r1"}],
            "should_not_match": [{"query": "q2", "reason": "r2"}],
        }
        errors = veq.validate_eval_document(data, schema, skill_name="huaweicloud-ces-ops", path="test.json")
        self.assertTrue(any("skill:" in error for error in errors))

    def test_structured_object_duplicate_ids(self) -> None:
        schema = veq.load_eval_schema(SCHEMA)
        entry = {
            "id": "E1",
            "category": "should_trigger",
            "query": "create user",
            "expected_skill": "huaweicloud-iam-ops",
            "pass_condition": "activates",
            "priority": "P0",
        }
        data = {
            "skill_name": "huaweicloud-iam-ops",
            "evaluation_queries": [
                entry,
                {**entry, "category": "should_not_trigger", "query": "create ecs"},
            ],
        }
        errors = veq.validate_eval_document(data, schema, skill_name="huaweicloud-iam-ops", path="test.json")
        self.assertTrue(any("duplicate id" in error for error in errors))


class CliTests(unittest.TestCase):
    def test_main_repo_passes(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["validate_eval_queries_schema.py", "--root", str(ROOT)]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = veq.main()
            self.assertEqual(rc, 0)
            self.assertIn("Checked", stdout.getvalue())
        finally:
            sys.argv = old_argv

    def test_main_invalid_file_returns_1(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            schema_dir = root / "huaweicloud-skill-generator/assets"
            skill_dir = root / "huaweicloud-ecs-ops/assets"
            schema_dir.mkdir(parents=True)
            skill_dir.mkdir(parents=True)
            (schema_dir / "eval-queries.schema.json").write_text(
                SCHEMA.read_text(encoding="utf-8"),
                encoding="utf-8",
            )
            (skill_dir / "eval_queries.json").write_text(
                json.dumps([{"query": "only positive", "should_activate": True}]),
                encoding="utf-8",
            )

            old_argv = sys.argv
            try:
                sys.argv = [
                    "validate_eval_queries_schema.py",
                    "--root",
                    str(root),
                    "huaweicloud-ecs-ops/assets/eval_queries.json",
                ]
                with contextlib.redirect_stdout(io.StringIO()) as stdout:
                    rc = veq.main()
                self.assertEqual(rc, 1)
                self.assertIn("FAIL:", stdout.getvalue())
            finally:
                sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
