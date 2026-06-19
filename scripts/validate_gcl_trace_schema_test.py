#!/usr/bin/env python3
"""Unit tests for scripts/validate_gcl_trace_schema.py."""

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

import gcl_runner  # noqa: E402
import validate_gcl_trace_schema as vts  # noqa: E402
from json_schema_subset import load_json, validate_value  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]
SCHEMA = ROOT / "huaweicloud-ces-ops/assets/gcl-trace.schema.json"


def sample_trace(status: str = "PASS") -> dict:
    output = "ok" if status != "SAFETY_FAIL" else None
    return {
        "trace_schema_version": "v1",
        "skill": "huaweicloud-ecs-ops",
        "request": "CI smoke test",
        "operation_intent": {
            "operation": "smoke",
            "resource_scope": [],
            "expected_state": "no-op",
            "safety_class": "read-only",
        },
        "rubric_version": "v1",
        "masked_fields": ["request", "operation_intent", "generator.command", "generator.result_excerpt"],
        "iterations": [
            {
                "iter": 1,
                "generator": {
                    "command": "printf ok",
                    "exit_code": 0,
                    "result_excerpt": "ok",
                    "stdout_len": 2,
                    "stderr_len": 0,
                    "args": {"iter": 1, "critic_feedback": None},
                },
                "critic": {
                    "scores": {
                        "correctness": 1,
                        "safety": 1,
                        "idempotency": 0.5,
                        "traceability": 1,
                        "spec_compliance": 0.5,
                    },
                    "suggestions": [],
                    "blocking": False,
                },
                "decision": status if status in {"PASS", "SAFETY_FAIL"} else "RETRY",
            }
        ],
        "final": {"status": status, "iter": 1, "output": output, "failure_pattern": None},
    }


def write_trace(root: Path, payload: dict, name: str = "gcl-trace-test.json") -> Path:
    audit_dir = root / "audit-results"
    audit_dir.mkdir(parents=True, exist_ok=True)
    path = audit_dir / name
    path.write_text(json.dumps(payload), encoding="utf-8")
    return path


class TraceSchemaTests(unittest.TestCase):
    def test_sample_trace_validates(self) -> None:
        schema = load_json(SCHEMA)
        self.assertEqual(validate_value(sample_trace(), schema), [])

    def test_safety_fail_trace_validates(self) -> None:
        schema = load_json(SCHEMA)
        self.assertEqual(validate_value(sample_trace("SAFETY_FAIL"), schema), [])

    def test_missing_final_fails(self) -> None:
        trace = sample_trace()
        del trace["final"]
        schema = load_json(SCHEMA)
        errors = validate_value(trace, schema)
        self.assertTrue(any("missing required property 'final'" in error for error in errors))

    def test_invalid_decision_fails(self) -> None:
        trace = sample_trace()
        trace["iterations"][0]["decision"] = "MAX_ITER"
        schema = load_json(SCHEMA)
        errors = validate_value(trace, schema)
        self.assertTrue(any("expected one of" in error for error in errors))

    def test_runner_output_validates(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            args = gcl_runner.build_parser().parse_args(
                [
                    "run",
                    "--root",
                    str(root),
                    "--skill",
                    "huaweicloud-ecs-ops",
                    "--request",
                    "CI smoke test",
                    "--operation-intent",
                    '{"operation":"smoke","resource_scope":[],"expected_state":"no-op","safety_class":"read-only"}',
                    "--command",
                    "printf ok",
                    "--max-iter",
                    "1",
                    "--structural-critic-only",
                ]
            )
            with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(gcl_runner.cmd_run(args), 0)
            trace_path = next((root / "audit-results").glob("gcl-trace-*.json"))
            errors = vts.validate_traces(root, SCHEMA, [trace_path], latest=False)[0]["errors"]
            self.assertEqual(errors, [])


class CollectTests(unittest.TestCase):
    def test_collect_latest(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            first = write_trace(root, sample_trace(), "gcl-trace-a.json")
            second = write_trace(root, sample_trace(), "gcl-trace-b.json")
            self.assertEqual(vts.collect_trace_paths(root, None, latest=False), [first, second])
            self.assertEqual(vts.collect_trace_paths(root, None, latest=True), [second])

    def test_explicit_inputs_filter_missing(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            trace_path = write_trace(root, sample_trace())
            paths = vts.collect_trace_paths(
                root, [Path("audit-results/gcl-trace-test.json"), Path("missing.json")], latest=False
            )
            self.assertEqual(paths, [trace_path])


class CliTests(unittest.TestCase):
    def test_main_allow_empty(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            old_argv = sys.argv
            try:
                sys.argv = ["validate_gcl_trace_schema.py", "--root", tmp, "--schema", str(SCHEMA), "--allow-empty"]
                with contextlib.redirect_stdout(io.StringIO()) as stdout:
                    rc = vts.main()
                self.assertEqual(rc, 0)
                self.assertIn("OK: no GCL trace files found", stdout.getvalue())
            finally:
                sys.argv = old_argv

    def test_main_invalid_trace_returns_1(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            bad = sample_trace()
            bad["trace_schema_version"] = "v0"
            write_trace(root, bad)
            old_argv = sys.argv
            try:
                sys.argv = ["validate_gcl_trace_schema.py", "--root", str(root), "--schema", str(SCHEMA)]
                with contextlib.redirect_stdout(io.StringIO()) as stdout:
                    rc = vts.main()
                self.assertEqual(rc, 1)
                self.assertIn("FAIL: audit-results/gcl-trace-test.json", stdout.getvalue())
            finally:
                sys.argv = old_argv


if __name__ == "__main__":
    unittest.main()
