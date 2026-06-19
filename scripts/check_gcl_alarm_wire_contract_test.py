#!/usr/bin/env python3
"""Unit tests for scripts/check_gcl_alarm_wire_contract.py."""

from __future__ import annotations

import contextlib
import io
import json
import sys
import tempfile
import unittest
from pathlib import Path

_HERE = Path(__file__).resolve().parent
if str(_HERE) not in sys.path:
    sys.path.insert(0, str(_HERE))

import check_gcl_alarm_wire_contract as caw  # noqa: E402
import gcl_alarm_wire as gaw  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]


def _write_canonical_config(root: Path, **overrides: float) -> Path:
    config = root / "huaweicloud-ces-ops/assets/example-config.yaml"
    config.parent.mkdir(parents=True, exist_ok=True)
    defaults = gaw.DEFAULT_THRESHOLDS
    body_lines = [
        "gcl_quality: &gcl_quality",
        f"  pass_rate_warn: {overrides.get('pass_rate_warn', defaults['pass_rate_warn'])}",
        f"  pass_rate_critical: {overrides.get('pass_rate_critical', defaults['pass_rate_critical'])}",
        f"  max_iter_warn_count: {overrides.get('max_iter_warn_count', defaults['max_iter_warn_count'])}",
        "  safety_fail_alert: true",
        "",
    ]
    body = "\n".join(body_lines)
    config.write_text(body, encoding="utf-8")
    return config


class RepoValidationTests(unittest.TestCase):
    def test_repo_passes(self) -> None:
        report = caw.check_all(ROOT)
        self.assertTrue(report["ok"], report["errors"])


class ConfigValidationTests(unittest.TestCase):
    def test_canonical_config_passes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            config = _write_canonical_config(root)
            ok, errors = caw.check_config(config)
            self.assertTrue(ok, errors)

    def test_missing_config_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            ok, errors = caw.check_config(root / "missing.yaml")
            self.assertFalse(ok)

    def test_drifted_threshold_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            config = _write_canonical_config(root, pass_rate_warn=0.99)
            ok, errors = caw.check_config(config)
            self.assertFalse(ok)
            self.assertTrue(any("drifts" in e for e in errors))

    def test_invalid_ordering_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            config = _write_canonical_config(root, pass_rate_warn=0.5, pass_rate_critical=0.9)
            ok, errors = caw.check_config(config)
            self.assertFalse(ok)
            self.assertTrue(any("ordering" in e for e in errors))


class PlanValidationTests(unittest.TestCase):
    def test_no_plans_is_ok(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            ok, errors = caw.check_existing_plans(root, wiring=gaw.DEFAULT_THRESHOLDS)
            self.assertTrue(ok, errors)

    def test_plan_matching_config_passes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "audit-results").mkdir(parents=True)
            plan = root / "audit-results/gcl-alarm-plan-test.json"
            plan.write_text(
                json.dumps(
                    {
                        "thresholds": {
                            "pass_rate_warn": 0.85,
                            "pass_rate_critical": 0.70,
                            "max_iter_warn_count": 3,
                            "safety_fail_alert": True,
                        }
                    }
                ),
                encoding="utf-8",
            )
            ok, errors = caw.check_existing_plans(root, wiring=gaw.DEFAULT_THRESHOLDS)
            self.assertTrue(ok, errors)

    def test_plan_drift_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "audit-results").mkdir(parents=True)
            plan = root / "audit-results/gcl-alarm-plan-test.json"
            plan.write_text(json.dumps({"thresholds": {"pass_rate_warn": 0.50}}), encoding="utf-8")
            ok, errors = caw.check_existing_plans(root, wiring=gaw.DEFAULT_THRESHOLDS)
            self.assertFalse(ok)
            self.assertTrue(any("disagrees" in e for e in errors))


class CliTests(unittest.TestCase):
    def test_main_repo_passes(self) -> None:
        old_argv = sys.argv
        try:
            sys.argv = ["check_gcl_alarm_wire_contract.py", "--root", str(ROOT)]
            with contextlib.redirect_stdout(io.StringIO()) as stdout:
                rc = caw.main()
            self.assertEqual(rc, 0)
            self.assertIn("OK", stdout.getvalue())
        finally:
            sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
