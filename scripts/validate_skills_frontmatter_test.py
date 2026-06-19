#!/usr/bin/env python3
"""Unit tests for scripts/validate_skills_frontmatter.py."""

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

import validate_skills_frontmatter as vsf  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]


def write_skill(root: Path, name: str, body: str) -> Path:
    skill_dir = root / name
    skill_dir.mkdir(parents=True)
    path = skill_dir / "SKILL.md"
    path.write_text(body, encoding="utf-8")
    return path


MINIMAL_FRONTMATTER = """---
name: {name}
description: Test skill
license: MIT
compatibility: Test runtime
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-06-19"
  cli_applicability: "dual-path"
  gcl:
    required: true
---

# body
"""


class RepoValidationTests(unittest.TestCase):
    def test_all_repo_skills_validate(self) -> None:
        report = vsf.validate_all(ROOT)
        self.assertTrue(report["ok"], report["errors"])
        self.assertGreaterEqual(report["count"], 20)


class ValidationTests(unittest.TestCase):
    def test_missing_frontmatter_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            path = write_skill(root, "huaweicloud-ecs-ops", "# no frontmatter\n")
            errors = vsf.validate_skill(path)
            self.assertTrue(any("missing YAML frontmatter" in error for error in errors))

    def test_name_directory_mismatch_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            path = write_skill(
                root,
                "huaweicloud-ecs-ops",
                MINIMAL_FRONTMATTER.format(name="huaweicloud-rds-ops"),
            )
            errors = vsf.validate_skill(path)
            self.assertTrue(any("does not match directory" in error for error in errors))

    def test_missing_cli_applicability_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            body = MINIMAL_FRONTMATTER.format(name="huaweicloud-ecs-ops").replace(
                '  cli_applicability: "dual-path"\n', ""
            )
            path = write_skill(root, "huaweicloud-ecs-ops", body)
            errors = vsf.validate_skill(path)
            self.assertTrue(any("missing metadata.cli_applicability" in error for error in errors))

    def test_billing_exempt_from_cli_applicability(self) -> None:
        billing = ROOT / "huaweicloud-billing-ops" / "SKILL.md"
        errors = vsf.validate_skill(billing)
        self.assertFalse(any("cli_applicability" in error for error in errors))


class CliTests(unittest.TestCase):
    def test_main_repo_passes(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["validate_skills_frontmatter.py", "--root", str(ROOT)]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = vsf.main()
            self.assertEqual(rc, 0)
            self.assertIn("OK:", stdout.getvalue())
        finally:
            sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
