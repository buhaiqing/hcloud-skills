#!/usr/bin/env python3
"""Unit tests for scripts/check_advanced_coverage.py."""

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

import check_advanced_coverage as cac  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]


class RepoValidationTests(unittest.TestCase):
    def test_tier_a_skills_have_advanced(self) -> None:
        report = cac.validate_all(ROOT)
        self.assertEqual(report["skills_checked"], 20)
        self.assertTrue(report["ok"], "\n".join(report["errors"]))


class SkillValidationTests(unittest.TestCase):
    def _write_skill(self, root: Path, name: str, *, with_advanced: bool, marker_text: str | None) -> Path:
        skill = root / name / "references"
        skill.mkdir(parents=True)
        (skill / "core-concepts.md").write_text("# stub\n", encoding="utf-8")
        if with_advanced:
            advanced = skill / "advanced"
            advanced.mkdir()
            (advanced / "aiops-patterns.md").write_text(
                "# aiops\nThis is Security-Sensitive content.\n" if marker_text is None else marker_text,
                encoding="utf-8",
            )
        return skill

    def test_missing_advanced_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._write_skill(root, "huaweicloud-ecs-ops", with_advanced=False, marker_text=None)
            report = cac.validate_skill(root, "huaweicloud-ecs-ops")
            self.assertFalse(report["ok"])
            self.assertTrue(any("missing references/advanced" in e for e in report["errors"]))

    def test_with_advanced_passes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._write_skill(root, "huaweicloud-ecs-ops", with_advanced=True, marker_text=None)
            report = cac.validate_skill(root, "huaweicloud-ecs-ops")
            self.assertTrue(report["ok"])

    def test_marker_counting(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self._write_skill(
                root,
                "huaweicloud-ecs-ops",
                with_advanced=True,
                marker_text="⚠ High risk — Security-Sensitive (高危) operation\n",
            )
            report = cac.validate_skill(root, "huaweicloud-ecs-ops")
            self.assertGreaterEqual(report["security_marker_count"], 3)


class CliTests(unittest.TestCase):
    def test_main_repo_passes(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["check_advanced_coverage.py", "--root", str(ROOT)]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = cac.main()
            self.assertEqual(rc, 0)
            self.assertIn("Checked", stdout.getvalue())
        finally:
            sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
