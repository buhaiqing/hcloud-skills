#!/usr/bin/env python3
"""Unit tests for scripts/check_safety_class_enum.py + safety_class runtime guard."""

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

import check_safety_class_enum as csce  # noqa: E402
import gcl_runner as gaw  # noqa: E402

REPO_ROOT = Path(__file__).resolve().parents[1]


def _valid_intent(safety_class: str) -> str:
    return json.dumps(
        {
            "operation": "list",
            "resource_scope": ["i-1"],
            "expected_state": "stable",
            "safety_class": safety_class,
        }
    )


class RuntimeGuardTests(unittest.TestCase):
    def test_runner_accepts_canonical_enum_values(self) -> None:
        for value in csce.EXPECTED_VALUES:
            with self.subTest(value=value):
                sanitized = gaw.sanitize_operation_intent(_valid_intent(value))
                self.assertIsInstance(sanitized, dict)
                self.assertEqual(sanitized["safety_class"], value)

    def test_runner_rejects_unknown_safety_class(self) -> None:
        with self.assertRaises(ValueError) as ctx:
            gaw.sanitize_operation_intent(_valid_intent("explosive"))
        self.assertIn("safety_class", str(ctx.exception))
        self.assertIn("explosive", str(ctx.exception))

    def test_runner_rejects_numeric_safety_class(self) -> None:
        with self.assertRaises(ValueError):
            gaw.sanitize_operation_intent(_valid_intent("42"))

    def test_runner_tolerates_missing_safety_class(self) -> None:
        payload = json.dumps({"operation": "list", "resource_scope": [], "expected_state": "stable"})
        sanitized = gaw.sanitize_operation_intent(payload)
        self.assertIsInstance(sanitized, dict)
        self.assertNotIn("safety_class", sanitized)


class SchemaGuardTests(unittest.TestCase):
    def test_schema_enum_matches_expected(self) -> None:
        ok, errors = csce.check_schema(REPO_ROOT)
        self.assertTrue(ok, errors)

    def test_schema_required_fields_present(self) -> None:
        schema = json.loads((REPO_ROOT / csce.SCHEMA_RELATIVE).read_text(encoding="utf-8"))
        required = schema["properties"]["operation_intent"]["required"]
        for key in ("operation", "resource_scope", "expected_state", "safety_class"):
            self.assertIn(key, required)


class CodeGuardTests(unittest.TestCase):
    def test_runner_constants_match_expected(self) -> None:
        ok, errors = csce.check_code()
        self.assertTrue(ok, errors)


class DocsGuardTests(unittest.TestCase):
    def test_documents_enumerate_values(self) -> None:
        ok, errors = csce.check_docs(REPO_ROOT)
        self.assertTrue(ok, errors)


class TracePersistenceTests(unittest.TestCase):
    def test_persisted_traces_with_invalid_safety_class_are_flagged(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            audit = root / "audit-results"
            audit.mkdir(parents=True)
            bad = audit / "gcl-trace-bad.json"
            bad.write_text(
                json.dumps(
                    {
                        "trace_schema_version": "v1",
                        "operation_intent": {"operation": "x", "safety_class": "explosive"},
                    }
                ),
                encoding="utf-8",
            )
            ok, errors = csce.check_traces_under_audit(root)
            self.assertFalse(ok)
            self.assertTrue(any("explosive" in e for e in errors))

    def test_clean_audit_dir_passes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "audit-results").mkdir()
            ok, errors = csce.check_traces_under_audit(root)
            self.assertTrue(ok, errors)


class CliAndRepoTests(unittest.TestCase):
    def test_repo_passes(self) -> None:
        report = csce.check_all(REPO_ROOT)
        self.assertTrue(report["ok"], report["errors"])

    def test_main_cli_repo(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["check_safety_class_enum.py", "--root", str(REPO_ROOT)]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = csce.main()
            self.assertEqual(rc, 0)
            self.assertIn("OK", stdout.getvalue())
        finally:
            sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
