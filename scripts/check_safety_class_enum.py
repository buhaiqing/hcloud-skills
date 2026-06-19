#!/usr/bin/env python3
"""Verify `operation_intent.safety_class` enum contract across the GCL pipeline.

Two gates:

1. **Schema gate** — `huaweicloud-ces-ops/assets/gcl-trace.schema.json` MUST
   declare `operation_intent.safety_class` as an enum of
   ``read-only|mutating|destructive``.
2. **Code gate** — `scripts/gcl_runner.py` MUST define `SAFETY_CLASS_VALUES`
   that exactly matches the schema enum, and `sanitize_operation_intent`
   MUST raise ``ValueError`` for any other value (fail-closed).

Plus: walk every ``gcl-prompt-backbone.md`` and ``docs/gcl-spec.md`` to
confirm the documented enum is in lock-step with the code, so the trio
(schema, code, doc) never drifts silently.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

import gcl_runner as gaw

ROOT_DEFAULT = Path(__file__).resolve().parents[1]
SCHEMA_RELATIVE = Path("huaweicloud-ces-ops/assets/gcl-trace.schema.json")
SPEC_RELATIVE = Path("docs/gcl-spec.md")
BACKBONE_RELATIVE = Path("huaweicloud-skill-generator/references/gcl-prompt-backbone.md")

EXPECTED_VALUES: tuple[str, ...] = ("read-only", "mutating", "destructive")


def check_schema(root: Path) -> tuple[bool, list[str]]:
    errors: list[str] = []
    schema_path = root / SCHEMA_RELATIVE
    if not schema_path.is_file():
        return False, [f"{schema_path}: missing canonical GCL trace schema"]
    schema = json.loads(schema_path.read_text(encoding="utf-8"))
    intent = schema.get("properties", {}).get("operation_intent", {})
    props = intent.get("properties", {}) if isinstance(intent, dict) else {}
    sc = props.get("safety_class")
    if not isinstance(sc, dict):
        return False, [f"{schema_path}: operation_intent.safety_class missing"]
    declared = sc.get("enum")
    if not isinstance(declared, list) or tuple(declared) != EXPECTED_VALUES:
        return False, [f"{schema_path}: operation_intent.safety_class.enum={declared!r} != {list(EXPECTED_VALUES)}"]
    required = intent.get("required", [])
    if "safety_class" not in required:
        errors.append(f"{schema_path}: operation_intent.safety_class must be in `required`")
    for key in ("operation", "resource_scope", "expected_state"):
        if key not in required:
            errors.append(f"{schema_path}: operation_intent.{key} must be in `required`")
    return not errors, errors


def check_code() -> tuple[bool, list[str]]:
    errors: list[str] = []
    values = getattr(gaw, "SAFETY_CLASS_VALUES", None)
    if tuple(values or ()) != EXPECTED_VALUES:
        errors.append(f"scripts/gcl_runner.py: SAFETY_CLASS_VALUES={values!r} != {list(EXPECTED_VALUES)}")
    if not hasattr(gaw, "sanitize_operation_intent"):
        errors.append("scripts/gcl_runner.py: sanitize_operation_intent missing")
        return not errors, errors
    try:
        gaw.sanitize_operation_intent(
            json.dumps(
                {
                    "operation": "delete",
                    "resource_scope": ["i-123"],
                    "expected_state": "gone",
                    "safety_class": "explosive",
                }
            )
        )
    except ValueError as exc:
        if "safety_class" not in str(exc):
            errors.append(f"scripts/gcl_runner.py: unexpected ValueError text: {exc}")
    else:
        errors.append(
            "scripts/gcl_runner.py: sanitize_operation_intent accepted invalid safety_class; expected ValueError"
        )
    for allowed in EXPECTED_VALUES:
        try:
            sanitized = gaw.sanitize_operation_intent(
                json.dumps(
                    {
                        "operation": "list",
                        "resource_scope": [],
                        "expected_state": "no-op",
                        "safety_class": allowed,
                    }
                )
            )
        except ValueError as exc:
            errors.append(f"scripts/gcl_runner.py: rejected valid safety_class={allowed!r}: {exc}")
            continue
        if not isinstance(sanitized, dict) or sanitized.get("safety_class") != allowed:
            errors.append(f"scripts/gcl_runner.py: safety_class={allowed!r} not preserved through sanitizer")
    return not errors, errors


def _doc_fragments(path: Path) -> list[str]:
    if not path.is_file():
        return []
    return [path.read_text(encoding="utf-8")]


def check_docs(root: Path) -> tuple[bool, list[str]]:
    errors: list[str] = []
    for rel in (SPEC_RELATIVE, BACKBONE_RELATIVE):
        text_chunks = _doc_fragments(root / rel)
        if not text_chunks:
            errors.append(f"{rel}: missing")
            continue
        corpus = "\n".join(text_chunks)
        for value in EXPECTED_VALUES:
            if not re.search(rf"\b{re.escape(value)}\b", corpus):
                errors.append(f"{rel}: enum value {value!r} not documented")
    return not errors, errors


def check_traces_under_audit(root: Path) -> tuple[bool, list[str]]:
    """Best-effort check: any persisted trace MUST use the enum when present.

    This is non-blocking for empty audit dirs (production rarely has traces on a
    clean checkout). It is intended to catch regression drift on re-runs.
    """

    errors: list[str] = []
    audit = root / "audit-results"
    if not audit.is_dir():
        return True, []
    for path in sorted(audit.glob("gcl-trace-*.json")):
        try:
            data = json.loads(path.read_text(encoding="utf-8"))
        except json.JSONDecodeError:
            continue
        intent = data.get("operation_intent")
        if not isinstance(intent, dict):
            continue
        sc = intent.get("safety_class")
        if sc is not None and sc not in EXPECTED_VALUES:
            errors.append(f"{path}: operation_intent.safety_class={sc!r} is not in the canonical enum")
    return not errors, errors


def check_all(root: Path) -> dict[str, Any]:
    sections = {
        "schema": check_schema(root),
        "code": check_code(),
        "docs": check_docs(root),
        "traces": check_traces_under_audit(root),
    }
    all_errors = [err for ok, errs in sections.values() for err in errs if not ok]
    return {
        "ok": not all_errors,
        "sections": {name: {"ok": ok, "errors": errs} for name, (ok, errs) in sections.items()},
        "errors": all_errors,
    }


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT_DEFAULT)
    parser.add_argument("--json", action="store_true")
    return parser


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    report = check_all(root)
    if args.json:
        print(json.dumps(report, indent=2, ensure_ascii=False))
    else:
        for name, section in report["sections"].items():
            status = "OK" if section["ok"] else "FAIL"
            print(f"{status}: {name}")
            for err in section["errors"]:
                print(f"  - {err}")
        if report["ok"]:
            print("\n[safety_class enum] OK")
        else:
            print(f"\n[safety_class enum] FAIL: {len(report['errors'])} issue(s)")
    return 0 if report["ok"] else 1


__all__ = [
    "EXPECTED_VALUES",
    "check_all",
    "check_code",
    "check_docs",
    "check_schema",
    "check_traces_under_audit",
]


if __name__ == "__main__":
    sys.exit(main())
