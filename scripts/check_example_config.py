#!/usr/bin/env python3
"""Validate assets/example-config.yaml files in huaweicloud-* skills.

Enforces TE-5 (token efficiency) and basic YAML/safety contracts:

- File exists for every Tier-A GCL skill.
- The YAML block (raw or fenced ````yaml```) parses with a minimal structural
  validator: every non-blank line MUST be a key:value pair; embedded JSON
  blocks and ``---`` document separators are tolerated.
- No secret values appear in plaintext (must use `{{env.*}}` placeholders).
- Optional TE-5 warning: when a file repeats the same key 3+ times without
  declaring any YAML anchors, the script suggests extracting a
  `&name` / `*name` / `<<:` alias.
- When YAML anchors ARE declared, anchors referenced via `<<: *name` or
  `*name` MUST be defined in the same document.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

from check_gcl_conformance import GCL_SKILLS

PLACEHOLDER = re.compile(r"\{\{\s*(env|user|output)\.[^{}\s]+\}\}")
BARE_PLACEHOLDER = re.compile(r"(?<!\{)\{[a-zA-Z_][a-zA-Z0-9_.-]*\}(?!\})")
ANCHOR_DEF = re.compile(r"^(\s*)([A-Za-z_][\w-]*):\s+&([A-Za-z_][\w-]*)\s*(?:#|$)")
MERGE_KEY_USE = re.compile(r"<<:\s*\*([A-Za-z_][\w-]*)")
SECRET_LITERAL = re.compile(r"(?i)(?:secret\s*[:=]\s*['\"][^'\"\s]+|sk\s*[:=]\s*['\"]?[A-Za-z0-9+/]{16,})")


def extract_yaml_block(text: str) -> tuple[str, str]:
    """Return (block_text, source_mode)."""
    match = re.search(r"```yaml\s*\n(.*?)\n```", text, re.DOTALL)
    if match:
        return match.group(1), "fenced"
    return text, "raw"


def collect_yaml_lines(text: str) -> list[tuple[int, str]]:
    """Strip comments, blanks, JSON braces, and YAML `---` separators."""
    cleaned: list[tuple[int, str]] = []
    for index, raw_line in enumerate(text.splitlines(), start=1):
        stripped = raw_line.split("#", 1)[0].rstrip()
        if not stripped.strip():
            continue
        if stripped.strip() == "---":
            continue
        if stripped.strip() in {"{", "}", "[", "]"}:
            continue
        cleaned.append((index, stripped))
    return cleaned


def indent_of(line: str) -> int:
    return len(line) - len(line.lstrip(" "))


def detect_repeated_keys(lines: list[tuple[int, str]]) -> dict[str, int]:
    counts: dict[str, int] = {}
    for _, line in lines:
        match = re.match(r"^(\s+)([A-Za-z_][\w-]*):\s", line)
        if not match:
            continue
        if indent_of(line) <= 2:
            continue
        key = match.group(2)
        counts[key] = counts.get(key, 0) + 1
    return counts


def detect_anchors(lines: list[tuple[int, str]]) -> tuple[set[str], list[str]]:
    defined: set[str] = set()
    errors: list[str] = []
    # Skip over inline JSON blocks so YAML anchor scans don't trip on JSON literals.
    sanitized: list[tuple[int, str]] = []
    in_json = False
    json_indent: int | None = None
    for line_no, line in lines:
        stripped = line.strip()
        if not in_json:
            if stripped.endswith("{") or stripped.endswith("["):
                in_json = True
                json_indent = indent_of(line)
            sanitized.append((line_no, line))
            continue
        sanitized.append((line_no, "# json-block"))
        if stripped in {"}", "]"} and json_indent is not None and indent_of(line) == json_indent:
            in_json = False
            json_indent = None
    for line_no, line in sanitized:
        match = ANCHOR_DEF.match(line)
        if match:
            defined.add(match.group(3))
            continue
        if line.startswith("#"):
            continue
        for alias in MERGE_KEY_USE.findall(line):
            if alias not in defined:
                errors.append(f"line {line_no}: anchor {alias!r} referenced before defined")
        value_alias = re.match(r"^[A-Za-z_][\w-]*:\s+\*([A-Za-z_][\w-]*)\s*(?:#|$)", line)
        if value_alias and value_alias.group(1) not in defined:
            errors.append(f"line {line_no}: anchor {value_alias.group(1)!r} referenced before defined")
    return defined, errors


def check_placeholders(text: str) -> list[str]:
    errors: list[str] = []
    for index, raw_line in enumerate(text.splitlines(), start=1):
        if raw_line.lstrip().startswith("#"):
            continue
        if not BARE_PLACEHOLDER.search(raw_line):
            continue
        if PLACEHOLDER.search(raw_line):
            continue
        errors.append(f"line {index}: bare placeholder in {raw_line.strip()[:80]!r}")
    return errors


def check_secrets(text: str) -> list[str]:
    errors: list[str] = []
    for index, raw_line in enumerate(text.splitlines(), start=1):
        if raw_line.lstrip().startswith("#"):
            continue
        if SECRET_LITERAL.search(raw_line):
            errors.append(f"line {index}: looks like plaintext secret — use <masked> or {{env.*}}")
    return errors


def validate_yaml_basic(block: str, source: str) -> list[str]:
    errors: list[str] = []
    for index, line in collect_yaml_lines(block):
        if ":" not in line and not line.startswith(" "):
            errors.append(f"{source}:{index}: expected key:value, got {line.strip()!r}")
    return errors


def validate_file(path: Path, root: Path) -> dict[str, Any]:
    rel = path.relative_to(root)
    if not path.is_file():
        return {
            "file": str(rel),
            "ok": False,
            "errors": [f"{rel}: missing example-config.yaml"],
            "warnings": [],
        }

    text = path.read_text(encoding="utf-8")
    block, mode = extract_yaml_block(text)
    errors: list[str] = []
    warnings: list[str] = []

    errors.extend(check_secrets(text))
    errors.extend(check_placeholders(text))
    errors.extend(validate_yaml_basic(block, rel.as_posix()))

    cleaned = collect_yaml_lines(block)
    defined, anchor_errors = detect_anchors(cleaned)
    errors.extend(anchor_errors)

    counts = detect_repeated_keys(cleaned)
    repeat_keys = sorted(key for key, count in counts.items() if count >= 3)
    if repeat_keys and not defined:
        warnings.append(
            f"{rel}: {len(repeat_keys)} key(s) repeated 3+ times without YAML anchors "
            f"({', '.join(repeat_keys[:5])}{'…' if len(repeat_keys) > 5 else ''})"
        )

    return {
        "file": str(rel),
        "ok": not errors,
        "mode": mode,
        "anchors_defined": sorted(defined),
        "repeat_keys": repeat_keys,
        "errors": errors,
        "warnings": warnings,
    }


def validate_all(root: Path) -> dict[str, Any]:
    reports = [validate_file(root / skill / "assets" / "example-config.yaml", root) for skill in sorted(GCL_SKILLS)]
    all_errors = [error for report in reports for error in report["errors"]]
    all_warnings = [warning for report in reports for warning in report["warnings"]]
    return {
        "ok": not all_errors,
        "files_checked": len(reports),
        "errors": all_errors,
        "warnings": all_warnings,
        "reports": reports,
    }


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--json", action="store_true")
    parser.add_argument(
        "--warn-only",
        action="store_true",
        help="Treat TE-5 repeat warnings as success (recommended for non-blocking rollout)",
    )
    return parser


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    report = validate_all(root)

    if args.json:
        print(json.dumps(report, indent=2, ensure_ascii=False))
    else:
        for item in report["reports"]:
            status = "OK" if item["ok"] else "FAIL"
            anchors = len(item.get("anchors_defined", []))
            repeats = len(item.get("repeat_keys", []))
            print(f"{status}: {item['file']}  anchors={anchors}  repeats={repeats}")
            for error in item["errors"]:
                print(f"  - {error}")
            for warning in item["warnings"]:
                print(f"  ~ {warning}")
        print(
            f"\nChecked {report['files_checked']} example-config.yaml file(s); "
            f"errors={len(report['errors'])}, warnings={len(report['warnings'])}"
        )

    if not report["ok"]:
        return 1
    return 0


__all__ = [
    "collect_yaml_lines",
    "detect_anchors",
    "extract_yaml_block",
    "validate_all",
    "validate_file",
]


if __name__ == "__main__":
    sys.exit(main())
