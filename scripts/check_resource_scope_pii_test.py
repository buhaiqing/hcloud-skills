#!/usr/bin/env python3
"""Unit tests for scripts/check_resource_scope_pii.py and the runtime PII mask."""

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

import re as _re

import check_resource_scope_pii as crs  # noqa: E402
import gcl_runner as gaw  # noqa: E402

REPO_ROOT = Path(__file__).resolve().parents[1]


class MaskResourceIdTests(unittest.TestCase):
    def test_aws_style_ids_preserve_prefix(self) -> None:
        self.assertEqual(gaw.mask_resource_id("i-abc123def456"), "i-***")
        self.assertEqual(gaw.mask_resource_id("sg-0f8c9a1b2c3d"), "sg-***")
        self.assertEqual(gaw.mask_resource_id("vpc-prod-01"), "vpc-***")
        self.assertEqual(gaw.mask_resource_id("subnet-1234abcd"), "subnet-***")

    def test_huawei_style_ids_preserve_prefix(self) -> None:
        self.assertEqual(gaw.mask_resource_id("rds-mysql-prod-01"), "rds-***")
        self.assertEqual(gaw.mask_resource_id("elb-abcdef"), "elb-***")

    def test_arn_masks_trailing_id_only(self) -> None:
        self.assertEqual(
            gaw.mask_resource_id("acs:rds:cn-north-4:12345:instance/i-abc123"),
            "acs:rds:cn-north-4:12345:instance/***",
        )

    def test_uuid_becomes_plain_mask(self) -> None:
        self.assertEqual(
            gaw.mask_resource_id("12345678-90ab-cdef-1234-567890abcdef"),
            "***",
        )

    def test_already_masked_passes_through(self) -> None:
        self.assertEqual(gaw.mask_resource_id("***"), "***")
        self.assertEqual(gaw.mask_resource_id("i-***"), "i-***")
        self.assertEqual(gaw.mask_resource_id("<masked>"), "<masked>")

    def test_unknown_shape_falls_back_to_plain_mask(self) -> None:
        # Empty string has no type prefix and falls back to "***"
        self.assertEqual(gaw.mask_resource_id(""), "***")
        # non-string inputs become "***"
        self.assertEqual(gaw.mask_resource_id(123), "***")
        self.assertEqual(gaw.mask_resource_id(None), "***")

    def test_non_string_input(self) -> None:
        self.assertEqual(gaw.mask_resource_id(123), "***")
        self.assertEqual(gaw.mask_resource_id(None), "***")


class SanitizerIntegrationTests(unittest.TestCase):
    def test_resource_scope_is_masked_in_sanitizer(self) -> None:
        sanitized = gaw.sanitize_operation_intent(
            json.dumps(
                {
                    "operation": "delete",
                    "resource_scope": ["i-abc123", "sg-0f8c9a1b"],
                    "expected_state": "gone",
                    "safety_class": "destructive",
                }
            )
        )
        self.assertEqual(sanitized["resource_scope"], ["i-***", "sg-***"])

    def test_resource_scope_string_is_masked(self) -> None:
        sanitized = gaw.sanitize_operation_intent(
            json.dumps(
                {
                    "operation": "list",
                    "resource_scope": "i-abc123",
                    "expected_state": "no-op",
                    "safety_class": "read-only",
                }
            )
        )
        self.assertEqual(sanitized["resource_scope"], "i-***")

    def test_already_masked_resource_scope_preserved(self) -> None:
        sanitized = gaw.sanitize_operation_intent(
            json.dumps(
                {
                    "operation": "list",
                    "resource_scope": ["i-***", "sg-***"],
                    "expected_state": "no-op",
                    "safety_class": "read-only",
                }
            )
        )
        self.assertEqual(sanitized["resource_scope"], ["i-***", "sg-***"])


class SchemaGuardTests(unittest.TestCase):
    def test_repo_schema_passes(self) -> None:
        ok, errors = crs.check_schema(REPO_ROOT)
        self.assertTrue(ok, errors)

    def test_schema_rejects_raw_id_pattern(self) -> None:
        schema = json.loads((REPO_ROOT / crs.SCHEMA_RELATIVE).read_text(encoding="utf-8"))
        items = schema["properties"]["operation_intent"]["properties"]["resource_scope"]["items"]
        raw_id = "i-abc123def456"
        for p in items["anyOf"]:
            self.assertIsNone(_re.fullmatch(p["pattern"], raw_id))


class CodeGuardTests(unittest.TestCase):
    def test_repo_code_passes(self) -> None:
        ok, errors = crs.check_code()
        self.assertTrue(ok, errors)


class RunnerMaskedFieldsTests(unittest.TestCase):
    def test_runner_includes_operation_intent_in_masked_fields(self) -> None:
        ok, errors = crs.check_runner_masked_fields()
        self.assertTrue(ok, errors)


class DocsGuardTests(unittest.TestCase):
    def test_documents_cover_masking(self) -> None:
        ok, errors = crs.check_docs(REPO_ROOT)
        self.assertTrue(ok, errors)


class TracePersistenceTests(unittest.TestCase):
    def test_persisted_trace_with_raw_id_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            audit = root / "audit-results"
            audit.mkdir(parents=True)
            bad = audit / "gcl-trace-bad.json"
            bad.write_text(
                json.dumps(
                    {
                        "operation_intent": {
                            "operation": "delete",
                            "resource_scope": ["i-abc123def456"],
                            "safety_class": "destructive",
                        }
                    }
                ),
                encoding="utf-8",
            )
            ok, errors = crs.check_traces_under_audit(root)
            self.assertFalse(ok)
            self.assertTrue(any("i-abc123def456" in e for e in errors))

    def test_persisted_trace_with_masked_id_passes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            audit = root / "audit-results"
            audit.mkdir(parents=True)
            good = audit / "gcl-trace-good.json"
            good.write_text(
                json.dumps(
                    {
                        "operation_intent": {
                            "operation": "delete",
                            "resource_scope": ["i-***"],
                            "safety_class": "destructive",
                        }
                    }
                ),
                encoding="utf-8",
            )
            ok, errors = crs.check_traces_under_audit(root)
            self.assertTrue(ok, errors)

    def test_empty_audit_dir_passes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            ok, errors = crs.check_traces_under_audit(Path(tmp))
            self.assertTrue(ok, errors)


class CliAndRepoTests(unittest.TestCase):
    def test_repo_passes(self) -> None:
        report = crs.check_all(REPO_ROOT)
        self.assertTrue(report["ok"], report["errors"])

    def test_main_cli_repo(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["check_resource_scope_pii.py", "--root", str(REPO_ROOT)]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = crs.main()
            self.assertEqual(rc, 0)
            self.assertIn("OK", stdout.getvalue())
        finally:
            sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
