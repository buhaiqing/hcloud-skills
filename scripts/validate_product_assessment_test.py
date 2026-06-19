#!/usr/bin/env python3
"""Unit tests for scripts/validate_product_assessment.py."""

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

import validate_product_assessment as vpa  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]


class RepoValidationTests(unittest.TestCase):
    def test_all_gcl_skills_validate(self) -> None:
        report = vpa.validate_all(ROOT)
        self.assertTrue(report["ok"], "\n".join(report["errors"]))
        self.assertEqual(report["files_checked"], len(vpa.PRODUCT_BY_SKILL))


class AssessmentValidationTests(unittest.TestCase):
    def test_example_payload_validates(self) -> None:
        payload = vpa.example_product_assessment("huaweicloud-ecs-ops", "ecs")
        errors = vpa.validate_assessment(payload, "test")
        self.assertEqual(errors, [])

    def test_invalid_finding_id_fails(self) -> None:
        payload = vpa.example_product_assessment("huaweicloud-ecs-ops", "ecs")
        payload["pillars"]["reliability"]["findings"] = [
            {
                "id": "bad-id",
                "severity": "High",
                "confidence": "HIGH",
                "title": "t",
                "evidence": "e",
                "recommendation": "r",
                "effort": "quick",
            }
        ]
        errors = vpa.validate_assessment(payload, "test")
        self.assertTrue(any("invalid" in error for error in errors))

    def test_unmasked_secret_in_trace_fails(self) -> None:
        payload = vpa.example_product_assessment("huaweicloud-ecs-ops", "ecs")
        payload["trace"]["commands"] = ["HW_SECRET_ACCESS_KEY=leaked"]
        errors = vpa.validate_assessment(payload, "test")
        self.assertTrue(any("unmasked secret" in error for error in errors))


class CliTests(unittest.TestCase):
    def test_main_repo_passes(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["validate_product_assessment.py", "--root", str(ROOT)]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = vpa.main()
            self.assertEqual(rc, 0)
            self.assertIn("OK:", stdout.getvalue())
        finally:
            sys.argv = old_argv

    def test_missing_contract_section_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            skill = root / "huaweicloud-ecs-ops" / "references"
            skill.mkdir(parents=True)
            (skill / "well-architected-assessment.md").write_text("# no contract\n", encoding="utf-8")
            errors = vpa.validate_file(skill / "well-architected-assessment.md", root, required=True)
            self.assertTrue(any("Worker Output Contract" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
