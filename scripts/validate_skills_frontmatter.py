#!/usr/bin/env python3
"""Validate SKILL.md YAML frontmatter across huaweicloud-* skill directories."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

from check_gcl_conformance import GCL_SKILLS

CLI_APPLICABILITY = frozenset({"dual-path", "cli-first", "cli-only", "sdk-only"})
FRONTMATTER = re.compile(r"^---\s*\n(.*?)\n---", re.DOTALL)
SKILL_GLOB = "huaweicloud-*/SKILL.md"
OPTIONAL_NO_CLI = frozenset({"huaweicloud-billing-ops", "huaweicloud-skill-generator"})
OPTIONAL_NO_GCL = frozenset({"huaweicloud-skill-generator"})


def extract_frontmatter(path: Path) -> tuple[str | None, list[str]]:
    text = path.read_text(encoding="utf-8")
    match = FRONTMATTER.match(text)
    if not match:
        return None, [f"{path}: missing YAML frontmatter"]
    return match.group(1), []


def has_key(block: str, key: str) -> bool:
    return bool(re.search(rf"^{re.escape(key)}:\s", block, re.MULTILINE))


def nested_metadata_field(block: str, field: str) -> str | None:
    if not has_key(block, "metadata"):
        return None
    match = re.search(rf"^\s+{re.escape(field)}:\s*[\"']?([^\"'\n]+)", block, re.MULTILINE)
    return match.group(1).strip('"').strip("'") if match else None


def top_level_field(block: str, field: str) -> str | None:
    match = re.search(rf"^{re.escape(field)}:\s*[\"']?([^\"'\n]+)", block, re.MULTILINE)
    return match.group(1).strip('"').strip("'") if match else None


def has_nested_block(block: str, parent: str, child: str) -> bool:
    if not has_key(block, parent):
        return False
    return bool(re.search(rf"^\s+{re.escape(child)}:\s", block, re.MULTILINE))


def validate_skill(path: Path) -> list[str]:
    block, errors = extract_frontmatter(path)
    if block is None:
        return errors

    skill_dir = path.parent.name
    name = top_level_field(block, "name")
    if not name or not name.startswith("huaweicloud-"):
        errors.append(f"{path}: missing or invalid 'name' (must start with huaweicloud-)")
    elif name != skill_dir:
        errors.append(f"{path}: name {name!r} does not match directory {skill_dir!r}")

    if not has_key(block, "description"):
        errors.append(f"{path}: missing 'description'")

    if not has_key(block, "compatibility"):
        errors.append(f"{path}: missing 'compatibility'")

    if not has_key(block, "license"):
        errors.append(f"{path}: missing 'license'")

    if not has_key(block, "metadata"):
        errors.append(f"{path}: missing 'metadata'")
        return errors

    version = nested_metadata_field(block, "version")
    updated = nested_metadata_field(block, "last_updated")
    if not version:
        errors.append(f"{path}: missing metadata.version")
    if not updated:
        errors.append(f"{path}: missing metadata.last_updated")

    skill_name = name or skill_dir
    cli = nested_metadata_field(block, "cli_applicability") or top_level_field(block, "cli_applicability")
    if cli and cli not in CLI_APPLICABILITY:
        errors.append(f"{path}: invalid cli_applicability {cli!r}")
    elif not cli and skill_name not in OPTIONAL_NO_CLI:
        errors.append(f"{path}: missing metadata.cli_applicability")

    if (
        skill_name in GCL_SKILLS
        and skill_name not in OPTIONAL_NO_GCL
        and not has_nested_block(block, "metadata", "gcl")
    ):
        errors.append(f"{path}: missing metadata.gcl for Tier-A GCL skill")

    return errors


def discover_skills(root: Path) -> list[Path]:
    return sorted(root.glob(SKILL_GLOB))


def validate_all(root: Path, skills: list[Path] | None = None) -> dict[str, Any]:
    skill_paths = skills or discover_skills(root)
    results: list[dict[str, Any]] = []
    all_errors: list[str] = []
    for skill_path in skill_paths:
        errors = validate_skill(skill_path)
        rel = str(skill_path.relative_to(root))
        results.append({"file": rel, "ok": not errors, "errors": errors})
        all_errors.extend(errors)
    return {
        "ok": not all_errors,
        "count": len(skill_paths),
        "error_count": len(all_errors),
        "results": results,
        "errors": all_errors,
    }


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("skills", nargs="*", type=Path, help="Specific SKILL.md paths")
    parser.add_argument("--json", action="store_true")
    return parser


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    skills = args.skills or discover_skills(root)
    if not skills:
        print("No SKILL.md files found", file=sys.stderr)
        return 1

    report = validate_all(root, skills)
    if args.json:
        print(json.dumps(report, indent=2, ensure_ascii=False))
    elif report["ok"]:
        print(f"OK: {report['count']} SKILL.md frontmatter files validated")
    else:
        print(f"FAIL: {report['error_count']} error(s) in {report['count']} skills\n")
        for error in report["errors"]:
            print(f"  - {error}")
    return 0 if report["ok"] else 1


__all__ = [
    "CLI_APPLICABILITY",
    "discover_skills",
    "extract_frontmatter",
    "validate_all",
    "validate_skill",
]


if __name__ == "__main__":
    sys.exit(main())
