#!/usr/bin/env python3
"""Unit tests for scripts/check_generator_contract.py."""

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

import check_generator_contract as cgc  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]


def write_minimal_contract(root: Path, *, include_quality_gate: bool = True) -> None:
    generator_dir = root / "huaweicloud-skill-generator"
    references = generator_dir / "references"
    references.mkdir(parents=True)
    quality_gate = "## Quality Gate (GCL)\noperation_intent\n" if include_quality_gate else ""
    (references / "huaweicloud-skill-template.md").write_text(
        "  gcl:\n"
        "    required: true\n"
        "    default_max_iter: 2\n"
        "    rubric_version: \"v1\"\n"
        "    trace_path: \"audit-results/gcl-trace-YYYYMMDD-HHMMSS.json\"\n"
        f"{quality_gate}"
        "references/rubric.md\n"
        "references/prompt-templates.md\n"
        "gcl-prompt-backbone.md\n",
        encoding="utf-8",
    )
    (generator_dir / "SKILL.md").write_text(
        "references/gcl-prompt-backbone.md\n"
        "`references/rubric.md`\n"
        "`references/prompt-templates.md`\n"
        "`metadata.gcl`\n",
        encoding="utf-8",
    )
    (references / "gcl-prompt-backbone.md").write_text(
        "## 1. Generator prompt template\n"
        "PRIMARY: hcloud\n"
        "huaweicloud-sdk-go-v3\n"
        "## 2. Critic prompt template\n"
        "{{output.operation_intent}}\n"
        "Do NOT consider the original user request\n"
        "read-only\n"
        "## 3. Orchestrator prompt template\n"
        "audit-results/gcl-trace-YYYYMMDD-HHMMSS.json\n",
        encoding="utf-8",
    )


class ContractTests(unittest.TestCase):
    def test_repository_contract_passes(self) -> None:
        report = cgc.check_contract(ROOT)
        self.assertTrue(report["ok"], report["failures"])
        self.assertEqual(report["summary"]["failing"], 0)

    def test_missing_quality_gate_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_minimal_contract(root, include_quality_gate=False)
            report = cgc.check_contract(root)
            self.assertFalse(report["ok"])
            self.assertTrue(any(failure["item"] == "quality_gate_heading" for failure in report["failures"]))

    def test_missing_file_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "huaweicloud-skill-generator" / "references").mkdir(parents=True)
            report = cgc.check_contract(root)
            self.assertFalse(report["ok"])
            self.assertTrue(any(failure["item"] == "file_exists" for failure in report["failures"]))

    def test_bare_placeholder_detection(self) -> None:
        self.assertTrue(cgc.has_bare_placeholders("Use {raw}"))
        self.assertFalse(cgc.has_bare_placeholders("Use {{output.operation_intent}}"))
        self.assertFalse(cgc.has_bare_placeholders("Use {{env.HW_REGION_ID}}"))
        self.assertFalse(cgc.has_bare_placeholders("https://example/${OS}-${ARCH}.tar.gz"))

    def test_format_human(self) -> None:
        report = {"summary": {"passing": 1, "total": 2}, "failures": [{"scope": "template", "item": "x", "reason": "missing"}]}
        text = cgc.format_human(report)
        self.assertIn("Generator GCL contract", text)
        self.assertIn("FAIL template.x", text)

    def test_main_json_mode(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["check_generator_contract.py", "--root", str(ROOT), "--json"]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = cgc.main()
            self.assertEqual(rc, 0)
            self.assertIn('"ok": true', stdout.getvalue())
        finally:
            sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
