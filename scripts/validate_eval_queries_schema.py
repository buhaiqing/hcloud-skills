#!/usr/bin/env python3
"""Validate assets/eval_queries.json files against the repository eval-queries contract."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

from json_schema_subset import load_json, resolve_schema_refs, resolve_under_root, validate_value

DEFAULT_SCHEMA = Path("huaweicloud-skill-generator/assets/eval-queries.schema.json")
EVAL_GLOB = "huaweicloud-*/assets/eval_queries.json"

TRIGGER_CATEGORIES = frozenset(
    {
        "should_trigger",
        "should_not_trigger",
        "trigger_accuracy",
        "execution_quality",
        "finops_integration",
        "secops_integration",
        "aiops_integration",
    }
)


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--schema", type=Path, default=DEFAULT_SCHEMA)
    parser.add_argument(
        "eval_files",
        nargs="*",
        type=Path,
        help="Specific eval_queries.json paths (default: all huaweicloud-*/assets/eval_queries.json)",
    )
    parser.add_argument("--json", action="store_true")
    return parser


def discover_eval_files(root: Path) -> list[Path]:
    return sorted(root.glob(EVAL_GLOB))


def skill_name_from_path(eval_path: Path, root: Path) -> str:
    rel = eval_path.resolve().relative_to(root.resolve())
    return rel.parts[0]


def _schema_def(schema: dict[str, Any], name: str) -> dict[str, Any]:
    defs = schema.get("$defs", {})
    if name not in defs:
        raise KeyError(f"missing $defs.{name!r} in schema")
    return resolve_schema_refs({"$ref": f"#/$defs/{name}", "$defs": defs})


def _validate_with_def(schema: dict[str, Any], def_name: str, value: Any, path: str) -> list[str]:
    return validate_value(value, _schema_def(schema, def_name), path)


def load_eval_schema(schema_path: Path) -> dict[str, Any]:
    return load_json(schema_path)


def _detect_array_format(first_item: dict[str, Any]) -> str:
    if "should_activate" in first_item:
        return "activateArrayEntry"
    if "should_match" in first_item:
        return "matchArrayEntry"
    if "should_trigger" in first_item:
        return "triggerArrayEntry"
    if "description" in first_item:
        return "smokeArrayEntry"
    raise ValueError(f"unrecognized array entry keys: {sorted(first_item)}")


def _validate_array_document(data: list[Any], schema: dict[str, Any], path: str, skill_name: str) -> list[str]:
    errors: list[str] = []
    if not data:
        errors.append(f"{path}: expected non-empty array")
        return errors

    if not all(isinstance(item, dict) for item in data):
        errors.append(f"{path}: every array item must be an object")
        return errors

    try:
        entry_def = _detect_array_format(data[0])
    except ValueError as exc:
        errors.append(f"{path}: {exc}")
        return errors

    for index, item in enumerate(data):
        item_path = f"{path}[{index}]"
        try:
            if _detect_array_format(item) != entry_def:
                errors.append(f"{item_path}: mixed array entry shapes are not allowed")
                continue
        except ValueError as exc:
            errors.append(f"{item_path}: {exc}")
            continue
        errors.extend(_validate_with_def(schema, entry_def, item, item_path))
        if entry_def == "matchArrayEntry":
            declared = item.get("skill")
            if declared is not None and declared != skill_name:
                errors.append(f"{item_path}.skill: expected {skill_name!r}, got {declared!r}")

    if entry_def == "activateArrayEntry":
        positives = sum(1 for item in data if item.get("should_activate") is True)
        negatives = sum(1 for item in data if item.get("should_activate") is False)
        if positives < 1:
            errors.append(f"{path}: expected at least one entry with should_activate=true")
        if negatives < 1:
            errors.append(f"{path}: expected at least one entry with should_activate=false")

    if entry_def == "matchArrayEntry":
        positives = sum(1 for item in data if item.get("should_match") is True)
        negatives = sum(1 for item in data if item.get("should_match") is False)
        if positives < 1:
            errors.append(f"{path}: expected at least one entry with should_match=true")
        if negatives < 1:
            errors.append(f"{path}: expected at least one entry with should_match=false")

    if entry_def == "triggerArrayEntry":
        positives = sum(1 for item in data if item.get("should_trigger") is True)
        negatives = sum(1 for item in data if item.get("should_trigger") is False)
        if positives < 1:
            errors.append(f"{path}: expected at least one entry with should_trigger=true")
        if negatives < 1:
            errors.append(f"{path}: expected at least one entry with should_trigger=false")

    if entry_def == "smokeArrayEntry" and len(data) < 3:
        errors.append(f"{path}: smoke format expects at least 3 query/description entries")

    return errors


def _validate_object_document(
    data: dict[str, Any],
    schema: dict[str, Any],
    path: str,
    skill_name: str,
) -> list[str]:
    keys = set(data)
    if "evaluation_queries" in keys:
        return _validate_structured_object(data, schema, path, skill_name)
    if "should_match" in keys:
        return _validate_match_object(data, schema, path, skill_name)
    if "should_trigger" in keys:
        return _validate_trigger_object(data, schema, path)
    return [f"{path}: unrecognized object shape; expected evaluation_queries, should_match, or should_trigger"]


def _validate_match_object(
    data: dict[str, Any],
    schema: dict[str, Any],
    path: str,
    skill_name: str,
) -> list[str]:
    errors = _validate_with_def(schema, "matchObject", data, path)
    declared = data.get("skill")
    if declared is not None and declared != skill_name:
        errors.append(f"{path}.skill: expected {skill_name!r}, got {declared!r}")
    return errors


def _validate_trigger_object(data: dict[str, Any], schema: dict[str, Any], path: str) -> list[str]:
    return _validate_with_def(schema, "triggerObject", data, path)


def _validate_structured_object(
    data: dict[str, Any],
    schema: dict[str, Any],
    path: str,
    skill_name: str,
) -> list[str]:
    errors = _validate_with_def(schema, "structuredObject", data, path)
    declared = data.get("skill_name")
    if declared != skill_name:
        errors.append(f"{path}.skill_name: expected {skill_name!r}, got {declared!r}")

    queries = data.get("evaluation_queries", [])
    ids: set[str] = set()
    has_trigger = False
    has_not_trigger = False
    for index, item in enumerate(queries):
        item_path = f"{path}.evaluation_queries[{index}]"
        entry_id = item.get("id")
        if isinstance(entry_id, str):
            if entry_id in ids:
                errors.append(f"{item_path}.id: duplicate id {entry_id!r}")
            ids.add(entry_id)
        category = item.get("category")
        if category not in TRIGGER_CATEGORIES:
            errors.append(f"{item_path}.category: unsupported category {category!r}")
        if category == "should_trigger" or category == "trigger_accuracy":
            has_trigger = True
        if category == "should_not_trigger":
            has_not_trigger = True
        expected = item.get("expected_skill")
        if expected is not None and expected != skill_name:
            errors.append(f"{item_path}.expected_skill: expected {skill_name!r}, got {expected!r}")

    if not has_trigger:
        errors.append(f"{path}.evaluation_queries: expected at least one should_trigger entry")
    if not has_not_trigger:
        errors.append(f"{path}.evaluation_queries: expected at least one should_not_trigger entry")
    return errors


def validate_eval_document(
    data: Any,
    schema: dict[str, Any],
    *,
    skill_name: str,
    path: str = "$",
) -> list[str]:
    if isinstance(data, list):
        return _validate_array_document(data, schema, path, skill_name)
    if isinstance(data, dict):
        return _validate_object_document(data, schema, path, skill_name)
    return [f"{path}: expected array or object, got {type(data).__name__}"]


def validate_eval_file(root: Path, schema_path: Path, eval_path: Path) -> list[str]:
    schema = load_eval_schema(schema_path)
    data = load_json(eval_path)
    skill_name = skill_name_from_path(eval_path, root)
    rel = str(eval_path.relative_to(root))
    return validate_eval_document(data, schema, skill_name=skill_name, path=rel)


def validate_all(root: Path, schema: Path, eval_files: list[Path]) -> list[dict[str, Any]]:
    schema_path = resolve_under_root(root, schema)
    results: list[dict[str, Any]] = []
    for raw_path in eval_files:
        eval_path = resolve_under_root(root, raw_path)
        errors = validate_eval_file(root, schema_path, eval_path)
        results.append({"file": str(raw_path), "ok": not errors, "errors": errors})
    return results


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    eval_files = args.eval_files or discover_eval_files(root)
    if not eval_files:
        print("No eval_queries.json files found", file=sys.stderr)
        return 1

    results = validate_all(root, args.schema, eval_files)
    ok = all(result["ok"] for result in results)
    if args.json:
        print(
            json.dumps(
                {"ok": ok, "schema": str(args.schema), "count": len(results), "results": results},
                indent=2,
                ensure_ascii=False,
            )
        )
    else:
        for result in results:
            status = "OK" if result["ok"] else "FAIL"
            print(f"{status}: {result['file']}")
            for error in result["errors"]:
                print(f"  - {error}")
        print(f"\nChecked {len(results)} eval_queries.json file(s)")
    return 0 if ok else 1


__all__ = [
    "discover_eval_files",
    "skill_name_from_path",
    "validate_eval_document",
    "validate_eval_file",
    "validate_all",
]


if __name__ == "__main__":
    sys.exit(main())
