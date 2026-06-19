#!/usr/bin/env python3
"""Unit tests for scripts/check_audit_results_guard.py."""

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

import check_audit_results_guard as cag  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]


class RepoValidationTests(unittest.TestCase):
    def test_repo_passes(self) -> None:
        report = cag.check_all(ROOT)
        self.assertTrue(report["ok"], report["errors"])


class GitignoreTests(unittest.TestCase):
    def _write_gitignore(self, root: Path, lines: list[str]) -> Path:
        path = root / ".gitignore"
        path.write_text("\n".join(lines) + "\n", encoding="utf-8")
        return path

    def test_missing_pattern_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._write_gitignore(root, ["audit-results/"])
            ok, errors = cag.check_gitignore(root)
            self.assertFalse(ok)
            self.assertTrue(any("gcl-trace" in e for e in errors))

    def test_all_patterns_pass(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._write_gitignore(
                root,
                [
                    "audit-results/",
                    "**/audit-results/",
                    "gcl-trace-*.json",
                    "**/gcl-trace-*.json",
                    "gcl-quality-summary-*.json",
                    "**/gcl-quality-summary-*.json",
                    "gcl-alarm-plan-*.json",
                    "**/gcl-alarm-plan-*.json",
                ],
            )
            ok, errors = cag.check_gitignore(root)
            self.assertTrue(ok, errors)


class DirectoryTests(unittest.TestCase):
    def test_missing_dir_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            ok, errors = cag.check_directory(root)
            self.assertFalse(ok)
            self.assertTrue(any("missing" in e for e in errors))

    def test_permissive_mode_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            audit = root / "audit-results"
            audit.mkdir()
            audit.chmod(0o755)
            ok, errors = cag.check_directory(root)
            self.assertFalse(ok)
            self.assertTrue(any("too permissive" in e for e in errors))


class CliTests(unittest.TestCase):
    def test_main_repo_passes(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["check_audit_results_guard.py", "--root", str(ROOT)]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = cag.main()
            self.assertEqual(rc, 0)
            self.assertIn("[audit-results guard] OK", stdout.getvalue())
        finally:
            sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
