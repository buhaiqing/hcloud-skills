#!/usr/bin/env python3
"""PII masking contract for `operation_intent.resource_scope`.

Four-gate enforcement keeps the masked shape consistent across the pipeline:

1. **Schema gate** — ``gcl-trace.schema.json`` declares a tight ``anyOf`` for
   each item: pure ``***``, ``<masked>``, or ``<prefix>-***``. Raw IDs like
   ``i-abc123`` MUST fail validation.
2. **Code gate** — ``gcl_runner.mask_resource_id`` masks every known ID shape
   and falls back to a wholesale ``***`` for anything unrecognized. The
   sanitizer MUST run before ``_enforce_safety_class_enum`` so even invalid
   safety classes get their resource_scope cleaned first.
3. **Runner gate** — the runner's ``masked_fields`` list includes
   ``operation_intent`` (and therefore ``resource_scope``).
4. **Trace gate** — any persisted ``audit-results/gcl-trace-*.json`` whose
   ``operation_intent.resource_scope`` contains a raw identifier is flagged
   for redaction. The check tolerates absent fields.

A single raw identifier in a trace is treated as a contract violation, since
the agent runtime may load it back and re-emit it.
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

ALLOWED_ITEM_PATTERNS: tuple[str, ...] = (
    r"^\*+$",  # pure ****
    r"^<masked>$",  # explicit placeholder
    r"^[A-Za-z][A-Za-z0-9-]*-\*+$",  # prefix-***
)

_MASKED_TOKEN = re.compile(r"[*]{3,}|<masked>")


def _is_masked_item(item: str) -> bool:
    if not isinstance(item, str):
        return False
    return any(re.match(pattern, item) for pattern in ALLOWED_ITEM_PATTERNS)


def check_schema(root: Path) -> tuple[bool, list[str]]:
    errors: list[str] = []
    schema_path = root / SCHEMA_RELATIVE
    if not schema_path.is_file():
        return False, [f"{schema_path}: missing canonical GCL trace schema"]
    schema = json.loads(schema_path.read_text(encoding="utf-8"))
    intent = schema.get("properties", {}).get("operation_intent", {})
    rs = intent.get("properties", {}).get("resource_scope") if isinstance(intent, dict) else None
    if not isinstance(rs, dict):
        return False, [f"{schema_path}: operation_intent.resource_scope missing"]
    if rs.get("type") != "array":
        return False, [f"{schema_path}: operation_intent.resource_scope.type != 'array'"]
    items = rs.get("items")
    if not isinstance(items, dict):
        return False, [f"{schema_path}: operation_intent.resource_scope.items missing"]
    if items.get("type") != "string":
        errors.append(f"{schema_path}: operation_intent.resource_scope.items.type != 'string'")
    any_of = items.get("anyOf")
    if not isinstance(any_of, list):
        return False, [f"{schema_path}: operation_intent.resource_scope.items.anyOf missing"]
    actual_patterns = [p.get("pattern") for p in any_of if isinstance(p, dict)]
    if tuple(actual_patterns) != ALLOWED_ITEM_PATTERNS:
        return False, [
            f"{schema_path}: resource_scope anyOf patterns {actual_patterns!r} != {list(ALLOWED_ITEM_PATTERNS)}"
        ]
    return not errors, errors


def check_code() -> tuple[bool, list[str]]:
    errors: list[str] = []
    if not hasattr(gaw, "mask_resource_id"):
        return False, ["scripts/gcl_runner.py: mask_resource_id missing"]
    mask = gaw.mask_resource_id
    raw_samples = {
        "i-abc123def456": "i-***",
        "sg-0f8c9a1b": "sg-***",
        "vpc-prod-1": "vpc-***",
        "elb-abcdef": "elb-***",
        "rds-mysql-prod-01": "rds-***",
        "acs:rds:cn-north-4:12345:instance/i-abc": "acs:rds:cn-north-4:12345:instance/***",
        "12345678-90ab-cdef-1234-567890abcdef": "***",
    }
    for sample, expected in raw_samples.items():
        try:
            got = mask(sample)
        except Exception as exc:  # noqa: BLE001
            errors.append(f"scripts/gcl_runner.py: mask_resource_id({sample!r}) raised {exc}")
            continue
        if got != expected:
            errors.append(f"scripts/gcl_runner.py: mask_resource_id({sample!r})={got!r} expected {expected!r}")
    if mask("***") != "***":
        errors.append("scripts/gcl_runner.py: already-masked values must pass through unchanged")
    # Empty string has no type prefix → fallback to plain "***"
    if mask("") != "***":
        errors.append("scripts/gcl_runner.py: empty input must fall back to '***'")

    sanitized = gaw.sanitize_operation_intent(
        json.dumps(
            {
                "operation": "delete",
                "resource_scope": ["i-abc123", "sg-0f8c9a1b"],
                "expected_state": "gone",
                "safety_class": "destructive",
            }
        )
    )
    if not isinstance(sanitized, dict):
        return False, ["scripts/gcl_runner.py: sanitize_operation_intent lost dict shape"]
    rs = sanitized.get("resource_scope")
    if rs != ["i-***", "sg-***"]:
        errors.append(
            f"scripts/gcl_runner.py: sanitize_operation_intent.resource_scope={rs!r} expected ['i-***', 'sg-***']"
        )
    return not errors, errors


def check_runner_masked_fields() -> tuple[bool, list[str]]:
    src = Path(gaw.__file__).read_text(encoding="utf-8")
    if '"operation_intent"' not in src and "'operation_intent'" not in src:
        return False, ["scripts/gcl_runner.py: masked_fields MUST include 'operation_intent'"]
    return True, []


def check_docs(root: Path) -> tuple[bool, list[str]]:
    errors: list[str] = []
    for rel in (SPEC_RELATIVE, BACKBONE_RELATIVE):
        path = root / rel
        if not path.is_file():
            errors.append(f"{rel}: missing")
            continue
        text = path.read_text(encoding="utf-8")
        if "resource_scope" not in text:
            errors.append(f"{rel}: missing documented `resource_scope`")
        if "mask" not in text.lower():
            errors.append(f"{rel}: missing any reference to masking")
    return not errors, errors


def _looks_like_raw_id(value: str) -> bool:
    if not isinstance(value, str) or _MASKED_TOKEN.search(value):
        return False
    if _is_masked_item(value):
        return False
    return bool(re.search(r"[A-Za-z0-9]{4,}", value))


def check_traces_under_audit(root: Path) -> tuple[bool, list[str]]:
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
        rs = intent.get("resource_scope")
        if not isinstance(rs, list):
            continue
        for index, item in enumerate(rs):
            if _looks_like_raw_id(item):
                errors.append(
                    f"{path}: operation_intent.resource_scope[{index}]={item!r} is not masked; "
                    "rerun gcl_runner.py to apply mask_resource_id"
                )
    return not errors, errors


def check_all(root: Path) -> dict[str, Any]:
    sections = {
        "schema": check_schema(root),
        "code": check_code(),
        "runner_masked_fields": check_runner_masked_fields(),
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
            print("\n[resource_scope PII contract] OK")
        else:
            print(f"\n[resource_scope PII contract] FAIL: {len(report['errors'])} issue(s)")
    return 0 if report["ok"] else 1


__all__ = [
    "ALLOWED_ITEM_PATTERNS",
    "check_all",
    "check_code",
    "check_docs",
    "check_runner_masked_fields",
    "check_schema",
    "check_traces_under_audit",
]


if __name__ == "__main__":
    sys.exit(main())
