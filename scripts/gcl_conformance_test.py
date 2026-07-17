#!/usr/bin/env python3
"""Unit tests for scripts/check_gcl_conformance.py."""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

_HERE = Path(__file__).resolve().parent
if str(_HERE) not in sys.path:
    sys.path.insert(0, str(_HERE))

import check_gcl_conformance as gclc  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]


class SkillListTests(unittest.TestCase):
    def test_20_huaweicloud_skills(self) -> None:
        expected = {
            "huaweicloud-billing-ops",
            "huaweicloud-cbr-ops",
            "huaweicloud-cdn-ops",
            "huaweicloud-cce-ops",
            "huaweicloud-ces-ops",
            "huaweicloud-css-ops",
            "huaweicloud-cts-ops",
            "huaweicloud-dcs-ops",
            "huaweicloud-dms-ops",
            "huaweicloud-dns-ops",
            "huaweicloud-ecs-ops",
            "huaweicloud-eip-ops",
            "huaweicloud-elb-ops",
            "huaweicloud-functiongraph-ops",
            "huaweicloud-gaussdb-ops",
            "huaweicloud-hss-ops",
            "huaweicloud-kms-ops",
            "huaweicloud-iam-ops",
            "huaweicloud-lts-ops",
            "huaweicloud-obs-ops",
            "huaweicloud-rds-ops",
            "huaweicloud-swr-ops",
            "huaweicloud-vpc-ops",
            "huaweicloud-waf-ops",
        }
        self.assertEqual(gclc.GCL_SKILLS, expected)
        self.assertEqual(len(gclc.GCL_SKILLS), 24)


class CounterTests(unittest.TestCase):
    def test_full_coverage_returns_target(self) -> None:
        text = "\n".join(f"## {number}. Title" for number in range(1, 9))
        self.assertEqual(gclc._count_numbered_sections(text, 8), 8)

    def test_missing_section_returns_zero(self) -> None:
        text = "\n".join(f"## {number}. Title" for number in range(1, 8))
        self.assertEqual(gclc._count_numbered_sections(text, 8), 0)

    def test_non_sequential_returns_zero(self) -> None:
        text = "## 1. X\n## 3. Y\n## 4. Z\n## 5. W\n## 6. V\n## 7. U\n## 8. T"
        self.assertEqual(gclc._count_numbered_sections(text, 8), 0)


class PlaceholderTests(unittest.TestCase):
    def test_bare_placeholder_detected_outside_fence(self) -> None:
        self.assertTrue(gclc._has_bare_placeholders("Use {bad.placeholder}"))

    def test_bare_placeholder_ignored_inside_fence(self) -> None:
        self.assertFalse(gclc._has_bare_placeholders("```json\n{bad}\n```"))

    def test_double_brace_placeholder_allowed(self) -> None:
        self.assertFalse(gclc._has_bare_placeholders("Use {{output.operation_intent}}"))


class CheckSkillTests(unittest.TestCase):
    def test_check_skill_returns_expected_keys(self) -> None:
        report = gclc.check_skill(ROOT, "huaweicloud-ecs-ops")
        expected = {
            "skill",
            "rubric_sections",
            "prompt_sections",
            "has_quality_gate",
            "prompt_has_operation_intent",
            "prompt_has_no_bare_placeholders",
            "rubric_ok",
            "prompt_ok",
            "skill_ok",
            "ok",
        }
        self.assertEqual(set(report.keys()), expected)

    def test_check_skill_conformant(self) -> None:
        report = gclc.check_skill(ROOT, "huaweicloud-ecs-ops")
        self.assertEqual(report["rubric_sections"], 8)
        self.assertEqual(report["prompt_sections"], 7)
        self.assertTrue(report["has_quality_gate"])
        self.assertTrue(report["prompt_has_operation_intent"])
        self.assertTrue(report["prompt_has_no_bare_placeholders"])
        self.assertTrue(report["ok"])

    def test_check_skill_missing_files(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            fake_root = Path(tmp)
            (fake_root / "huaweicloud-fake-ops").mkdir()
            (fake_root / "huaweicloud-fake-ops" / "SKILL.md").write_text("# Fake\n", encoding="utf-8")
            report = gclc.check_skill(fake_root, "huaweicloud-fake-ops")
            self.assertFalse(report["has_quality_gate"])
            self.assertEqual(report["rubric_sections"], 0)
            self.assertEqual(report["prompt_sections"], 0)
            self.assertFalse(report["ok"])


class CheckAllTests(unittest.TestCase):
    def test_check_all_23_sorted(self) -> None:
        result = gclc.check_all(ROOT)
        self.assertEqual(len(result), 24)
        skills = [report["skill"] for report in result]
        self.assertEqual(skills, sorted(skills))


class ConformanceTests(unittest.TestCase):
    def test_all_23_pass(self) -> None:
        result = gclc.check_all(ROOT)
        failing = [report["skill"] for report in result if not report["ok"]]
        self.assertEqual(failing, [], f"Expected all 24 skills to conform; failing: {failing}")

    def test_rubric_section_count(self) -> None:
        for report in gclc.check_all(ROOT):
            with self.subTest(skill=report["skill"]):
                self.assertEqual(report["rubric_sections"], 8)

    def test_prompt_section_count(self) -> None:
        for report in gclc.check_all(ROOT):
            with self.subTest(skill=report["skill"]):
                self.assertEqual(report["prompt_sections"], 7)

    def test_quality_gate_present(self) -> None:
        for report in gclc.check_all(ROOT):
            with self.subTest(skill=report["skill"]):
                self.assertTrue(report["has_quality_gate"])


if __name__ == "__main__":
    unittest.main()
