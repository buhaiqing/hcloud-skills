#!/usr/bin/env python3
"""Validate GCL quality summary JSON files against the repository schema.

The repository avoids runtime third-party dependencies, so this implements the
small JSON Schema subset used by `huaweicloud-ces-ops/assets/gcl-quality-summary.schema.json`.
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime
from pathlib import Path
from typing import Any

DEFAULT_SCHEMA = Path("huaweicloud-ces-ops/assets/gcl-quality-summary.schema.json")
DEFAULT_SUMMARY = Path("scripts/fixtures/gcl-quality-summary-healthy.json")


def json_type(value: Any) -> str:
    if value is None:
        return "null"
    if isinstance(value, bool):
        return "boolean"
    if isinstance(value, int) and not isinstance(value, bool):
        return "integer"
    if isinstance(value, float):
        return "number"
    if isinstance(value, str):
        return "string"
    if isinstance(value, list):
        return "array"
    if isinstance(value, dict):
        return "object"
    return type(value).__name__


def type_matches(value: Any, expected: str | list[str]) -> bool:
    expected_types = [expected] if isinstance(expected, str) else expected
    actual = json_type(value)
    if actual in expected_types:
        return True
    return actual == "integer" and "number" in expected_types


def validate_datetime(value: str, path: str) -> list[str]:
    try:
        datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return [f"{path}: expected RFC3339/date-time string"]
    return []


def validate_value(value: Any, schema: dict[str, Any], path: str = "$") -> list[str]:
    errors: list[str] = []

    if "type" in schema and not type_matches(value, schema["type"]):
        errors.append(f"{path}: expected type {schema['type']}, got {json_type(value)}")
        return errors

    if "const" in schema and value != schema["const"]:
        errors.append(f"{path}: expected const {schema['const']!r}, got {value!r}")

    if isinstance(value, (int, float)) and not isinstance(value, bool):
        if "minimum" in schema and value < schema["minimum"]:
            errors.append(f"{path}: value {value} < minimum {schema['minimum']}")
        if "maximum" in schema and value > schema["maximum"]:
            errors.append(f"{path}: value {value} > maximum {schema['maximum']}")

    if schema.get("format") == "date-time" and isinstance(value, str):
        errors.extend(validate_datetime(value, path))

    if isinstance(value, dict):
        required = schema.get("required", [])
        for key in required:
            if key not in value:
                errors.append(f"{path}: missing required property {key!r}")

        properties = schema.get("properties", {})
        for key, prop_schema in properties.items():
            if key in value:
                errors.extend(validate_value(value[key], prop_schema, f"{path}.{key}"))

        additional = schema.get("additionalProperties", True)
        for key, item in value.items():
            if key in properties:
                continue
            if additional is False:
                errors.append(f"{path}: additional property {key!r} is not allowed")
            elif isinstance(additional, dict):
                errors.extend(validate_value(item, additional, f"{path}.{key}"))

    if isinstance(value, list) and isinstance(schema.get("items"), dict):
        item_schema = schema["items"]
        for index, item in enumerate(value):
            errors.extend(validate_value(item, item_schema, f"{path}[{index}]"))

    return errors


def load_json(path: Path) -> Any:
    return json.loads(path.read_text(encoding="utf-8"))


def validate_file(summary_path: Path, schema_path: Path) -> list[str]:
    summary = load_json(summary_path)
    schema = load_json(schema_path)
    return validate_value(summary, schema)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--schema", type=Path, default=DEFAULT_SCHEMA)
    parser.add_argument("summary", nargs="*", type=Path, default=[DEFAULT_SUMMARY])
    parser.add_argument("--json", action="store_true")
    return parser


def resolve_under_root(root: Path, path: Path) -> Path:
    return path if path.is_absolute() else root / path


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    schema_path = resolve_under_root(root, args.schema)
    results: list[dict[str, Any]] = []

    for raw_summary in args.summary:
        summary_path = resolve_under_root(root, raw_summary)
        errors = validate_file(summary_path, schema_path)
        results.append({"summary": str(raw_summary), "ok": not errors, "errors": errors})

    ok = all(result["ok"] for result in results)
    if args.json:
        print(json.dumps({"ok": ok, "schema": str(args.schema), "results": results}, indent=2, ensure_ascii=False))
    else:
        for result in results:
            status = "OK" if result["ok"] else "FAIL"
            print(f"{status}: {result['summary']}")
            for error in result["errors"]:
                print(f"  - {error}")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
