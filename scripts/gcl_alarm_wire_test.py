#!/usr/bin/env python3
"""Unit tests for scripts/gcl_alarm_wire.py."""

from __future__ import annotations

import argparse
import contextlib
import io
import json
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

_HERE = Path(__file__).resolve().parent
if str(_HERE) not in sys.path:
    sys.path.insert(0, str(_HERE))

import gcl_alarm_wire as gaw  # noqa: E402


def write_summary(root: Path, payload: dict) -> Path:
    audit = root / "audit-results"
    audit.mkdir(parents=True, exist_ok=True)
    path = audit / "gcl-quality-summary-test.json"
    path.write_text(json.dumps(payload), encoding="utf-8")
    return path


def healthy_summary() -> dict:
    return {
        "cloud": "huaweicloud",
        "pass_rate": 0.95,
        "totals": {"PASS": 19, "MAX_ITER": 1, "SAFETY_FAIL": 0, "total_runs": 20},
    }


def quiet_call(func, args: argparse.Namespace) -> int:
    with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
        return func(args)


class ThresholdTests(unittest.TestCase):
    def test_load_default_thresholds_when_config_missing(self) -> None:
        thresholds = gaw.load_thresholds_from_config(Path("/tmp/non-existent-gcl-config.yaml"))
        self.assertEqual(thresholds, gaw.DEFAULT_THRESHOLDS)

    def test_load_thresholds_from_config_block(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            config = Path(tmp) / "example-config.yaml"
            config.write_text(
                "gcl_quality: &gcl_quality\n"
                "  pass_rate_warn: 0.90\n"
                "  pass_rate_critical: 0.75\n"
                "  max_iter_warn_count: 4\n"
                "  safety_fail_alert: false\n",
                encoding="utf-8",
            )
            thresholds = gaw.load_thresholds_from_config(config)
            self.assertEqual(thresholds["pass_rate_warn"], 0.90)
            self.assertEqual(thresholds["pass_rate_critical"], 0.75)
            self.assertEqual(thresholds["max_iter_warn_count"], 4.0)
            self.assertFalse(thresholds["safety_fail_alert"])


class EvaluateTests(unittest.TestCase):
    def test_healthy_summary_ok(self) -> None:
        evaluation = gaw.evaluate(healthy_summary(), gaw.DEFAULT_THRESHOLDS)
        self.assertTrue(evaluation["ok"])
        self.assertEqual(evaluation["breaches"], [])

    def test_safety_fail_is_critical(self) -> None:
        summary = {"pass_rate": 0.99, "totals": {"PASS": 9, "MAX_ITER": 0, "SAFETY_FAIL": 1, "total_runs": 10}}
        evaluation = gaw.evaluate(summary, gaw.DEFAULT_THRESHOLDS)
        self.assertFalse(evaluation["ok"])
        self.assertTrue(any(breach["metric"] == "safety_fail_count" for breach in evaluation["breaches"]))

    def test_pass_rate_warning_is_non_critical(self) -> None:
        summary = {"pass_rate": 0.80, "totals": {"PASS": 8, "MAX_ITER": 0, "SAFETY_FAIL": 0, "total_runs": 10}}
        evaluation = gaw.evaluate(summary, gaw.DEFAULT_THRESHOLDS)
        self.assertTrue(evaluation["ok"])
        self.assertEqual(evaluation["breaches"][0]["severity"], "WARN")


class PlanTests(unittest.TestCase):
    def test_no_summary_returns_2(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            args = argparse.Namespace(root=Path(tmp), summary=None, config=Path(tmp) / "missing.yaml", write_plan=False)
            self.assertEqual(quiet_call(gaw.cmd_plan, args), 2)

    def test_plan_does_not_call_subprocess_run(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            summary = write_summary(root, healthy_summary())
            args = argparse.Namespace(root=root, summary=summary, config=root / "missing.yaml", write_plan=False)
            with patch.object(gaw.subprocess, "run") as run:
                self.assertEqual(quiet_call(gaw.cmd_plan, args), 0)
                run.assert_not_called()

    def test_slo_breach_returns_1(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            summary = write_summary(
                root,
                {
                    "pass_rate": 0.60,
                    "totals": {"PASS": 6, "MAX_ITER": 0, "SAFETY_FAIL": 1, "total_runs": 10},
                },
            )
            args = argparse.Namespace(root=root, summary=summary, config=root / "missing.yaml", write_plan=False)
            self.assertEqual(quiet_call(gaw.cmd_plan, args), 1)

    def test_plan_write_plan_persists_artifact(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            summary = write_summary(root, healthy_summary())
            args = argparse.Namespace(root=root, summary=summary, config=root / "missing.yaml", write_plan=True)
            self.assertEqual(quiet_call(gaw.cmd_plan, args), 0)
            persisted = sorted((root / "audit-results").glob("gcl-alarm-plan-*-plan.json"))
            self.assertEqual(len(persisted), 1)
            payload = json.loads(persisted[0].read_text(encoding="utf-8"))
            self.assertEqual(payload["cloud"], "huaweicloud")
            self.assertEqual(payload["metric_namespace"], "CUSTOM.GCL")
            self.assertIn("generated_at", payload)
            self.assertEqual(len(payload["alarm_plan"]), 4)


class ApplyTests(unittest.TestCase):
    def test_apply_dry_run_does_not_call_subprocess_run(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            summary = write_summary(root, healthy_summary())
            args = argparse.Namespace(root=root, summary=summary, config=root / "missing.yaml", dry_run=True)
            with patch.object(gaw.subprocess, "run") as run:
                self.assertEqual(quiet_call(gaw.cmd_apply, args), 0)
                run.assert_not_called()

    def test_apply_dry_run_persists_plan(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            summary = write_summary(root, healthy_summary())
            args = argparse.Namespace(root=root, summary=summary, config=root / "missing.yaml", dry_run=True)
            quiet_call(gaw.cmd_apply, args)
            persisted = sorted((root / "audit-results").glob("gcl-alarm-plan-*-dry-run.json"))
            self.assertEqual(len(persisted), 1)
            payload = json.loads(persisted[0].read_text(encoding="utf-8"))
            self.assertTrue(payload["dry_run"])
            self.assertEqual(payload["metric_namespace"], "CUSTOM.GCL")

    def test_apply_uses_hcloud_ces_command(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            summary = write_summary(root, healthy_summary())
            args = argparse.Namespace(root=root, summary=summary, config=root / "missing.yaml", dry_run=False)
            completed = gaw.subprocess.CompletedProcess(args=[], returncode=0, stdout="ok", stderr="")
            with patch.object(gaw.subprocess, "run", return_value=completed) as run:
                self.assertEqual(quiet_call(gaw.cmd_apply, args), 0)
                first_call = run.call_args_list[0].args[0]
                self.assertEqual(first_call[:3], ["hcloud", "ces", "create-alarm-rule"])


if __name__ == "__main__":
    unittest.main()
