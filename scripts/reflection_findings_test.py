"""Tests for scripts/reflection_findings.py."""

from __future__ import annotations

import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPTS = Path(__file__).resolve().parent
FIXTURE = SCRIPTS / "fixtures" / "reflection_findings_sample.jsonl"
CLI = SCRIPTS / "reflection_findings.py"


class SchemaValidateTest(unittest.TestCase):
    """Unit tests for the schema validator."""

    def test_valid_row_passes(self) -> None:
        from scripts.reflection_findings import validate
        row = {
            "ts": "2026-09-01T10:00:00Z",
            "round": 1,
            "finding_type": "missing-foo",
            "detail": "foo missing",
            "source_skill": "huaweicloud-ecs-ops",
            "rubric_item": None,
            "proposed_action": "add-rubric-item",
        }
        self.assertEqual(validate(row)["finding_type"], "missing-foo")

    def test_missing_field_rejected(self) -> None:
        from scripts.reflection_findings import SchemaError, validate
        bad = {"ts": "2026-09-01T10:00:00Z", "round": 1}
        with self.assertRaises(SchemaError):
            validate(bad)

    def test_bad_round_rejected(self) -> None:
        from scripts.reflection_findings import SchemaError, validate
        bad = {
            "ts": "2026-09-01T10:00:00Z",
            "round": 4,
            "finding_type": "x",
            "detail": "x",
            "source_skill": "huaweicloud-ecs-ops",
            "rubric_item": None,
            "proposed_action": "add-rubric-item",
        }
        with self.assertRaises(SchemaError):
            validate(bad)

    def test_bad_finding_type_rejected(self) -> None:
        from scripts.reflection_findings import SchemaError, validate
        bad = {
            "ts": "2026-09-01T10:00:00Z",
            "round": 1,
            "finding_type": "Not Kebab",
            "detail": "x",
            "source_skill": "huaweicloud-ecs-ops",
            "rubric_item": None,
            "proposed_action": "add-rubric-item",
        }
        with self.assertRaises(SchemaError):
            validate(bad)

    def test_bad_proposed_action_rejected(self) -> None:
        from scripts.reflection_findings import SchemaError, validate
        bad = {
            "ts": "2026-09-01T10:00:00Z",
            "round": 1,
            "finding_type": "missing-foo",
            "detail": "x",
            "source_skill": "huaweicloud-ecs-ops",
            "rubric_item": None,
            "proposed_action": "delete",
        }
        with self.assertRaises(SchemaError):
            validate(bad)


class ProposePureTest(unittest.TestCase):
    """Pure-function tests for propose()."""

    def test_repeat_cluster_is_proposed(self) -> None:
        from scripts.reflection_findings import load_jsonl, propose
        rows = load_jsonl(FIXTURE)
        out = propose(rows, min_hits=3, rubric_path=None)
        types = [p["finding_type"] for p in out]
        self.assertIn("missing-cli-fallback-note", types)
        proposal = next(p for p in out if p["finding_type"] == "missing-cli-fallback-note")
        self.assertEqual(proposal["hits"], 5)
        self.assertEqual(proposal["proposed_action"], "add-rubric-item")
        self.assertIn("first", proposal["evidence_ts"])
        self.assertIn("last", proposal["evidence_ts"])

    def test_singleton_not_proposed(self) -> None:
        from scripts.reflection_findings import load_jsonl, propose
        rows = load_jsonl(FIXTURE)
        out = propose(rows, min_hits=3, rubric_path=None)
        types = [p["finding_type"] for p in out]
        self.assertNotIn("error-table-too-wide", types)
        self.assertNotIn("hardcoded-region-in-yaml", types)

    def test_rubric_linked_finding_skipped(self) -> None:
        from scripts.reflection_findings import load_jsonl, propose
        rows = load_jsonl(FIXTURE)
        out = propose(rows, min_hits=3, rubric_path=None)
        for p in out:
            # any row carrying rubric_item must NOT seed a proposal
            self.assertNotEqual(p["finding_type"], "no-credential-mask-table")

    def test_rubric_filter_skips_covered_type(self) -> None:
        from scripts.reflection_findings import load_jsonl, propose
        rows = load_jsonl(FIXTURE)
        # (a) Rubric mentioning a long cluster token MUST drop the matching
        # finding — the most common cluster in FIXTURE is
        # `missing-cli-fallback-note`, whose long tokens include `fallback`.
        with tempfile.TemporaryDirectory() as td:
            rubric_covers = (
                "Section A: SecOps MUST include a CLI fallback note in every "
                "skill's troubleshooting section, so on-call agents know "
                "what to do when the primary CLI is unavailable."
            )
            covered_path = Path(td) / "rubric_covers.md"
            covered_path.write_text(rubric_covers, encoding="utf-8")
            out_covered = propose(rows, min_hits=3, rubric_path=covered_path)
            self.assertEqual(
                out_covered, [],
                f"covered cluster must be filtered out, got {out_covered}",
            )
            # (b) Rubric mentioning none of the cluster tokens MUST still propose.
            rubric_irrelevant = (
                "Section B: docs should use consistent heading levels and "
                "avoid stale screenshots."
            )
            open_path = Path(td) / "rubric_open.md"
            open_path.write_text(rubric_irrelevant, encoding="utf-8")
            out_open = propose(rows, min_hits=3, rubric_path=open_path)
            self.assertGreaterEqual(
                len(out_open), 1,
                "irrelevant rubric must NOT filter anything; expected at "
                f"least one proposal, got {out_open}",
            )

    def test_proposals_sorted_by_hits_desc(self) -> None:
        from scripts.reflection_findings import load_jsonl, propose
        rows = load_jsonl(FIXTURE)
        out = propose(rows, min_hits=2, rubric_path=None)
        hits = [p["hits"] for p in out]
        self.assertEqual(hits, sorted(hits, reverse=True))


class LoadJsonlTest(unittest.TestCase):
    """Tests for load_jsonl()."""

    def test_bad_jsonl_line_raises(self) -> None:
        from scripts.reflection_findings import SchemaError, load_jsonl
        with tempfile.NamedTemporaryFile("w", suffix=".jsonl", delete=False) as fh:
            fh.write('{"ts":"2026-09-01T10:00:00Z"}\n')
            fh.write("not json\n")
            path = Path(fh.name)
        try:
            with self.assertRaises(SchemaError):
                load_jsonl(path)
        finally:
            path.unlink()


class AppendSubprocessTest(unittest.TestCase):
    """End-to-end: append a finding, then propose reads it back."""

    def test_append_then_propose(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            jf = Path(td) / "findings.jsonl"
            row = {
                "ts": "2026-09-08T12:00:00Z",
                "round": 2,
                "finding_type": "missing-cli-fallback-note",
                "detail": "another missing fallback",
                "source_skill": "huaweicloud-sfs-ops",
                "rubric_item": None,
                "proposed_action": "add-rubric-item",
            }
            for i in range(3):
                payload = dict(row, ts=f"2026-09-{8 + i:02d}T12:00:00Z",
                               source_skill=f"huaweicloud-sfs-ops-{i}")
                res = subprocess.run(
                    [sys.executable, str(CLI), "append", "--file", str(jf),
                     "--finding", json.dumps(payload)],
                    capture_output=True, text=True, check=False,
                )
                self.assertEqual(res.returncode, 0, msg=res.stderr)
            res2 = subprocess.run(
                [sys.executable, str(CLI), "propose", "--file", str(jf),
                 "--min-hits", "3"],
                capture_output=True, text=True, check=False,
            )
            self.assertEqual(res2.returncode, 0, msg=res2.stderr)
            proposals = json.loads(res2.stdout)
            self.assertTrue(any(
                p["finding_type"] == "missing-cli-fallback-note" and p["hits"] >= 3
                for p in proposals
            ))

    def test_append_rejects_bad_row(self) -> None:
        with tempfile.TemporaryDirectory() as td:
            jf = Path(td) / "findings.jsonl"
            res = subprocess.run(
                [sys.executable, str(CLI), "append", "--file", str(jf),
                 "--finding", '{"round": 1}'],
                capture_output=True, text=True, check=False,
            )
            self.assertEqual(res.returncode, 2)
            self.assertIn("schema error", res.stderr)


class ProposeSubprocessTest(unittest.TestCase):
    """End-to-end propose against the shipped fixture."""

    def test_propose_against_fixture(self) -> None:
        res = subprocess.run(
            [sys.executable, str(CLI), "propose",
             "--file", str(FIXTURE), "--min-hits", "3"],
            capture_output=True, text=True, check=False,
        )
        self.assertEqual(res.returncode, 0, msg=res.stderr)
        proposals = json.loads(res.stdout)
        types = {p["finding_type"] for p in proposals}
        self.assertIn("missing-cli-fallback-note", types)
        # singletons must NOT appear
        self.assertNotIn("hardcoded-region-in-yaml", types)
        self.assertNotIn("error-table-too-wide", types)


class AppendCanonicalizationTest(unittest.TestCase):
    """FIX-B3-1: append must write only REQUIRED_FIELDS, dropping extras."""

    def test_append_drops_extra_keys(self) -> None:
        from scripts.reflection_findings import REQUIRED_FIELDS
        with tempfile.TemporaryDirectory() as td:
            jf = Path(td) / "findings.jsonl"
            payload = json.dumps({
                "ts": "2026-09-08T12:00:00Z",
                "round": 1,
                "finding_type": "missing-foo-bar",
                "detail": "foo bar missing",
                "source_skill": "huaweicloud-cce-ops",
                "rubric_item": None,
                "proposed_action": "add-rubric-item",
                "extra_field_1": "should-be-dropped",
                "extra_field_2": 42,
            })
            res = subprocess.run(
                [sys.executable, str(CLI), "append", "--file", str(jf),
                 "--finding", payload],
                capture_output=True, text=True, check=False,
            )
            self.assertEqual(res.returncode, 0, msg=res.stderr)
            # warning must be emitted
            self.assertIn("dropped extra keys", res.stderr)
            # on-disk line must have only 7 fields, in REQUIRED_FIELDS order
            written = jf.read_text().strip().splitlines()
            self.assertEqual(len(written), 1)
            obj = json.loads(written[0])
            self.assertEqual(set(obj.keys()), set(REQUIRED_FIELDS))
            self.assertEqual(list(obj.keys()), list(REQUIRED_FIELDS))


class InRubricTest(unittest.TestCase):
    """FIX-B3-2: _in_rubric now matches long tokens (>=4) OR substring."""

    def test_long_token_match_skips_proposal(self) -> None:
        from scripts.reflection_findings import _in_rubric
        # Rubric mentions "credential" — finding "no-credential-mask-table"
        # shares the long token "credential" → should be considered covered.
        rubric = "Section X: SecOps MUST include credential masking rules."
        self.assertTrue(_in_rubric("no-credential-mask-table", rubric))

    def test_unrelated_finding_not_covered(self) -> None:
        from scripts.reflection_findings import _in_rubric
        rubric = "Section X: SecOps MUST include credential masking rules."
        self.assertFalse(_in_rubric("hardcoded-region-in-yaml", rubric))

    def test_short_token_alone_does_not_cover(self) -> None:
        from scripts.reflection_findings import _in_rubric
        # Threshold is >=4 chars; a finding whose only tokens are <4 chars
        # (here `ab`, `cd`) is not covered even if the rubric mentions them.
        rubric = "Use ab/cd codes for X."
        self.assertFalse(_in_rubric("ab-cd", rubric))


if __name__ == "__main__":
    unittest.main()