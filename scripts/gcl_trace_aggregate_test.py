#!/usr/bin/env python3
"""Unit tests for scripts/gcl_trace_aggregate.py."""

from __future__ import annotations

import contextlib
import io
import json
import os
import sys
import tempfile
import time
import unittest
from pathlib import Path

_HERE = Path(__file__).resolve().parent
if str(_HERE) not in sys.path:
    sys.path.insert(0, str(_HERE))

import gcl_trace_aggregate as gta  # noqa: E402

SCORES = {
    "correctness": 1,
    "safety": 1,
    "idempotency": 0.5,
    "traceability": 1,
    "spec_compliance": 1,
}


def trace(skill: str, status: str, iterations: int = 1) -> dict:
    return {
        "skill": skill,
        "iterations": [{"iter": idx + 1, "critic": {"scores": SCORES}} for idx in range(iterations)],
        "final": {"status": status, "iter": iterations, "output": "..."},
    }


def write_trace(root: Path, name: str, payload: dict | str) -> Path:
    audit = root / "audit-results"
    audit.mkdir(parents=True, exist_ok=True)
    path = audit / name
    if isinstance(payload, str):
        path.write_text(payload, encoding="utf-8")
    else:
        path.write_text(json.dumps(payload), encoding="utf-8")
    return path


def quiet_main(argv: list[str]) -> int:
    old_argv = sys.argv
    try:
        sys.argv = argv
        with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            return gta.main()
    finally:
        sys.argv = old_argv


class ParseTests(unittest.TestCase):
    def test_invalid_json_returns_none(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = write_trace(Path(tmp), "gcl-trace-bad.json", "{not-json")
            with contextlib.redirect_stderr(io.StringIO()):
                self.assertIsNone(gta.parse_trace(path))

    def test_missing_final_returns_none(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = write_trace(Path(tmp), "gcl-trace-missing.json", {"skill": "huaweicloud-ecs-ops"})
            with contextlib.redirect_stderr(io.StringIO()):
                self.assertIsNone(gta.parse_trace(path))


class AggregateTests(unittest.TestCase):
    def test_status_counts_and_pass_rate(self) -> None:
        traces = [
            trace("huaweicloud-ecs-ops", "PASS", 1),
            trace("huaweicloud-ecs-ops", "SAFETY_FAIL", 2),
            trace("huaweicloud-rds-ops", "MAX_ITER", 3),
            trace("huaweicloud-rds-ops", "PASS", 1),
        ]
        summary = gta.aggregate(traces)
        self.assertEqual(summary["cloud"], "huaweicloud")
        self.assertEqual(summary["metric_namespace"], "CUSTOM.GCL")
        self.assertEqual(summary["totals"]["PASS"], 2)
        self.assertEqual(summary["totals"]["SAFETY_FAIL"], 1)
        self.assertEqual(summary["totals"]["MAX_ITER"], 1)
        self.assertEqual(summary["pass_rate"], 0.5)
        self.assertEqual(summary["by_skill"]["huaweicloud-ecs-ops"]["total"], 2)
        self.assertEqual(summary["by_skill"]["huaweicloud-rds-ops"]["MAX_ITER"], 1)
        self.assertEqual(summary["avg_rubric_scores"]["idempotency"], 0.5)

    def test_empty_traces_have_zero_pass_rate(self) -> None:
        summary = gta.aggregate([])
        self.assertEqual(summary["totals"]["total_runs"], 0)
        self.assertEqual(summary["pass_rate"], 0.0)


class CollectPathTests(unittest.TestCase):
    def test_no_trace_returns_empty(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            self.assertEqual(gta.collect_paths(Path(tmp), None, None), [])

    def test_since_hours_filters_old_files(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            recent = write_trace(root, "gcl-trace-recent.json", trace("huaweicloud-ecs-ops", "PASS"))
            old = write_trace(root, "gcl-trace-old.json", trace("huaweicloud-ecs-ops", "PASS"))
            old_time = time.time() - 72 * 3600
            os.utime(old, (old_time, old_time))

            paths = gta.collect_paths(root, None, since_hours=24)
            self.assertEqual(paths, [recent])

    def test_input_glob(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            first = write_trace(root, "gcl-trace-a.json", trace("huaweicloud-ecs-ops", "PASS"))
            second = write_trace(root, "gcl-trace-b.json", trace("huaweicloud-rds-ops", "MAX_ITER"))
            paths = gta.collect_paths(root, ["audit-results/gcl-trace-*.json"], None)
            self.assertEqual(paths, [first, second])


class MainTests(unittest.TestCase):
    def test_main_no_trace_returns_1(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            self.assertEqual(quiet_main(["gcl_trace_aggregate.py", "--root", tmp]), 1)

    def test_main_skips_invalid_json_and_persists_summary(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            write_trace(root, "gcl-trace-good.json", trace("huaweicloud-ecs-ops", "PASS"))
            write_trace(root, "gcl-trace-bad.json", "{bad")
            self.assertEqual(quiet_main(["gcl_trace_aggregate.py", "--root", tmp]), 0)
            summaries = sorted((root / "audit-results").glob("gcl-quality-summary-*.json"))
            self.assertEqual(len(summaries), 1)
            data = json.loads(summaries[0].read_text(encoding="utf-8"))
            self.assertEqual(data["totals"]["PASS"], 1)


class AuditResultsDirModeTests(unittest.TestCase):
    """`persist_summary` must create `audit-results/` at mode 0700 so the
    `check_audit_results_guard` gate passes. Regression: the dir was
    created with default umask (0755) on first run, which the guard
    hard-fails (see gcl_runner_test.AuditResultsDirModeTests)."""

    def test_persist_summary_creates_audit_results_at_0700(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            gta.persist_summary(root, {"totals": {"PASS": 0}})
            audit = root / "audit-results"
            self.assertTrue(audit.is_dir())
            import stat

            mode = stat.S_IMODE(audit.stat().st_mode)
            self.assertEqual(mode, 0o700, f"expected 0700, got {oct(mode)}")


if __name__ == "__main__":
    unittest.main()
