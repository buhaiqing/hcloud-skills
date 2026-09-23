"""Unit tests for rule_compliance_probe.py.

Covers:
  - Three-bucket classification on the fixture (verifiable/manual/vague)
  - has_example detection
  - vague rule is NOT classified as verifiable
  - Bad --rules path exits with code 1
  - --exec-check exit code 2 on a command that fails
"""
from __future__ import annotations

import io
import json
import subprocess
import sys
import tempfile
import unittest
from contextlib import redirect_stderr, redirect_stdout
from pathlib import Path

HERE = Path(__file__).resolve().parent
FIXTURE = HERE / "fixtures" / "rule_probe_sample.md"
PROBE = HERE / "rule_compliance_probe.py"

sys.path.insert(0, str(HERE))
from rule_compliance_probe import (
    analyze,
)
from rule_compliance_probe import (
    main as probe_main,
)


class FixtureClassification(unittest.TestCase):
    def test_fixture_classifies_three_buckets(self):
        report = analyze(FIXTURE)
        by_title = {r["title"]: r for r in report["rules"]}

        for tid in ("R1", "R2", "R3", "R4", "R11"):
            self.assertEqual(by_title[tid]["machine_checkable"], "verifiable",
                             f"{tid} should be verifiable")

        for tid in ("R5", "R6", "R7"):
            self.assertEqual(by_title[tid]["machine_checkable"], "manual",
                             f"{tid} should be manual")

        for tid in ("R8", "R9", "R10"):
            self.assertEqual(by_title[tid]["machine_checkable"], "vague",
                             f"{tid} should be vague")

        # FIX-B1 regression: bare-slash and CJK prose must NOT be verifiable.
        for tid in ("R12", "R13"):
            self.assertNotEqual(by_title[tid]["machine_checkable"], "verifiable",
                                f"{tid} must NOT be verifiable")

    def test_vague_not_misjudged(self):
        report = analyze(FIXTURE)
        by_title = {r["title"]: r for r in report["rules"]}
        self.assertNotEqual(by_title["R8"]["machine_checkable"], "verifiable")
        self.assertNotEqual(by_title["R10"]["machine_checkable"], "verifiable")

    def test_bare_slash_text_not_path_anchor(self):
        """FIX-B1: prose containing `spec/` or `任何涉及代码/` must not trigger
        PATH_ANCHOR. R13 is the canonical negative fixture."""
        report = analyze(FIXTURE)
        by_title = {r["title"]: r for r in report["rules"]}
        self.assertFalse(by_title["R13"]["anchors"]["has_path"],
                         "R13 prose with bare slash + CJK must not be a path anchor")

    def test_verifiable_rate_is_fraction_in_range(self):
        report = analyze(FIXTURE)
        self.assertEqual(report["total_rules"], 13)
        rate = report["verifiable_rate"]
        self.assertGreaterEqual(rate, 0.0)
        self.assertLessEqual(rate, 1.0)

    def test_has_example_detection(self):
        report = analyze(FIXTURE)
        by_title = {r["title"]: r for r in report["rules"]}
        # R11 contains a fenced bash block → example present
        self.assertTrue(by_title["R11"]["has_example"])
        # R8 has neither example nor code
        self.assertFalse(by_title["R8"]["has_example"])

    def test_verification_cmd_extracted(self):
        report = analyze(FIXTURE)
        by_title = {r["title"]: r for r in report["rules"]}
        cmd = by_title["R11"]["anchors"]["verification_cmd"]
        self.assertIn("echo", cmd)
        self.assertIn("wc -c", cmd)


class BadArguments(unittest.TestCase):
    def test_missing_rules_file_exits_1(self):
        captured_err = io.StringIO()
        with redirect_stdout(io.StringIO()), redirect_stderr(captured_err):
            rc = probe_main(["--rules", "/nonexistent/path.md"])
        self.assertEqual(rc, 1)
        self.assertIn("not found", captured_err.getvalue())

    def test_help_flag_exits_0(self):
        # argparse -h exits via SystemExit(0); main shouldn't be reached.
        # FIX-B5: suppress argparse's stderr/stdout chatter so it doesn't
        # leak into the unittest output stream.
        with redirect_stdout(io.StringIO()), redirect_stderr(io.StringIO()), \
                self.assertRaises(SystemExit) as cm:
            probe_main(["--help"])
        self.assertEqual(cm.exception.code, 0)


class ExecCheck(unittest.TestCase):
    def test_exec_check_skipped_unsafe_blocks_pipes(self):
        # Build an inline doc with an unsafe bash block (uses |) and confirm
        # the probe routes it through `skipped_unsafe`.
        with tempfile.TemporaryDirectory() as td:
            tmp_doc = Path(td) / "unsafe_doc.md"
            tmp_doc.write_text(
                "- **RX**: Run this:\n\n  ```bash\n  echo foo | grep foo\n  ```\n",
                encoding="utf-8",
            )
            buf = io.StringIO()
            with redirect_stdout(buf):
                rc = probe_main(["--rules", str(tmp_doc), "--json", "--exec-check"])
            self.assertEqual(rc, 0)
            data = json.loads(buf.getvalue())
            rx = next(r for r in data["rules"] if r["title"] == "RX")
            er = rx["exec_result"]
            self.assertTrue(er.get("skipped_unsafe"))

    def test_exec_check_failed_command_exits_2(self):
        # grep for a needle that doesn't exist in the chosen file → returncode 1
        with tempfile.TemporaryDirectory() as td:
            tmp_doc = Path(td) / "fail_doc.md"
            tmp_doc.write_text(
                "- **RY**: smoke:\n\n  ```bash\n  grep XYZTHISNOTEXIST _no_such_file_\n  ```\n",
                encoding="utf-8",
            )
            with redirect_stdout(io.StringIO()), redirect_stderr(io.StringIO()):
                rc = probe_main(["--rules", str(tmp_doc), "--exec-check"])
            self.assertEqual(rc, 2, "non-zero command must yield exit code 2")

    def test_exec_check_skips_unsafe_git_subcommand(self):
        """FIX-B4: `git reset --hard`, `git commit`, `git push`, etc. must be
        rejected via skipped_unsafe, not executed."""
        with tempfile.TemporaryDirectory() as td:
            tmp_doc = Path(td) / "git_doc.md"
            tmp_doc.write_text(
                "- **RG**: reset:\n\n  ```bash\n  git reset --hard\n  ```\n",
                encoding="utf-8",
            )
            buf = io.StringIO()
            with redirect_stdout(buf):
                rc = probe_main(["--rules", str(tmp_doc), "--json", "--exec-check"])
            self.assertEqual(rc, 0)
            data = json.loads(buf.getvalue())
            rg = next(r for r in data["rules"] if r["title"] == "RG")
            er = rg["exec_result"]
            self.assertTrue(er.get("skipped_unsafe"),
                            f"git reset --hard must be skipped: {er}")
            self.assertIn("git subcommand", er.get("reason", ""))


class ExecVerbCJKTest(unittest.TestCase):
    """FIX-D2: CJK imperative verbs (必须/应该/禁止) classify as manual.

    Prior bug: the verb regex wrapped the CJK verbs in `\b...\b`, which is
    impossible in a continuous CJK stream and produced a dead pattern.
    A rule carrying 必须 then fell through to vague instead of manual.
    """

    def test_chinese_imperative_verb_classifies_manual(self):
        with tempfile.TemporaryDirectory() as td:
            doc = Path(td) / "cjk_doc.md"
            # A rule whose only signal is the CJK imperative 必须 must land
            # in the manual bucket. We deliberately avoid backticked commands
            # / paths / regexes / thresholds — those would short-circuit to
            # verifiable ahead of the verb check.
            doc.write_text(
                "- **CJK-MUST**: 任何规则文件都必须经过人工评审，"
                "未经评审不得合入主分支。\n",
                encoding="utf-8",
            )
            buf = io.StringIO()
            with redirect_stdout(buf):
                rc = probe_main(["--rules", str(doc), "--json"])
            self.assertEqual(rc, 0)
            data = json.loads(buf.getvalue())
            rule = data["rules"][0]
            self.assertEqual(
                rule["machine_checkable"], "manual",
                f"CJK 必须 must classify as manual, got "
                f"{rule['machine_checkable']}",
            )


class EndToEndCLI(unittest.TestCase):
    def test_cli_json_on_fixture(self):
        completed = subprocess.run(
            [sys.executable, str(PROBE), "--rules", str(FIXTURE), "--json"],
            capture_output=True, text=True, check=False,
        )
        self.assertEqual(completed.returncode, 0, completed.stderr)
        data = json.loads(completed.stdout)
        self.assertEqual(data["total_rules"], 13)
        self.assertIn("verifiable_rate", data)


if __name__ == "__main__":
    unittest.main(verbosity=2)
