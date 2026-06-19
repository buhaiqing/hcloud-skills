#!/usr/bin/env python3
"""Small JSON Schema subset validator used by repository checks."""

from __future__ import annotations

import json
from datetime import datetime
from pathlib import Path
from typing import Any


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

    if "enum" in schema and value not in schema["enum"]:
        errors.append(f"{path}: expected one of {schema['enum']!r}, got {value!r}")

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


def validate_file(instance_path: Path, schema_path: Path) -> list[str]:
    instance = load_json(instance_path)
    schema = load_json(schema_path)
    return validate_value(instance, schema)


def resolve_under_root(root: Path, path: Path) -> Path:
    return path if path.is_absolute() else root / path
