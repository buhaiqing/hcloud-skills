#!/usr/bin/env python3
"""Unit tests for scripts/check_example_config.py."""

from __future__ import annotations

import contextlib
import io
import sys
import tempfile
import unittest
from pathlib import Path

_HERE = Path(__file__).resolve().parent
if str(_HERE) not in sys.path:
    sys.path.insert(0, str(_HERE))

import check_example_config as cec  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]


class RepoValidationTests(unittest.TestCase):
    def test_all_gcl_skills_have_example_config(self) -> None:
        report = cec.validate_all(ROOT)
        self.assertEqual(len(report["reports"]), 20)
        self.assertTrue(report["ok"], "\n".join(report["errors"]))


class ValidationTests(unittest.TestCase):
    def test_missing_file_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            report = cec.validate_file(root / "missing.yaml", root)
            self.assertFalse(report["ok"])
            self.assertTrue(any("missing" in error for error in report["errors"]))

    def test_anchors_detected(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            path = root / "ok.yaml"
            path.write_text(
                "_base: &base\n  period: 300\nalarm:\n  - <<: *base\n    name: a\n",
                encoding="utf-8",
            )
            report = cec.validate_file(path, root)
            self.assertTrue(report["ok"])
            self.assertIn("base", report["anchors_defined"])

    def test_undefined_alias_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            path = root / "bad.yaml"
            path.write_text("alarm:\n  - <<: *missing_anchor\n    name: a\n", encoding="utf-8")
            report = cec.validate_file(path, root)
            self.assertFalse(report["ok"])
            self.assertTrue(any("missing_anchor" in error for error in report["errors"]))

    def test_plaintext_secret_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            path = root / "leak.yaml"
            path.write_text("credentials:\n  sk: ABCDEFGHIJKLMNOP\n", encoding="utf-8")
            report = cec.validate_file(path, root)
            self.assertFalse(report["ok"])
            self.assertTrue(any("plaintext secret" in error for error in report["errors"]))

    def test_repeated_keys_warn(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            path = root / "dup.yaml"
            yaml_lines = (
                "items:\n"
                "  - name: a\n    period: 300\n    threshold: 90\n"
                "  - name: b\n    period: 300\n    threshold: 80\n"
                "  - name: c\n    period: 300\n    threshold: 70\n"
            )
            path.write_text(yaml_lines, encoding="utf-8")
            report = cec.validate_file(path, root)
            self.assertTrue(report["ok"])
            self.assertTrue(any("repeated 3+ times" in w for w in report["warnings"]))


class CliTests(unittest.TestCase):
    def test_main_repo_passes(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["check_example_config.py", "--root", str(ROOT), "--warn-only"]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = cec.main()
            self.assertEqual(rc, 0)
            self.assertIn("Checked", stdout.getvalue())
        finally:
            sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
