#!/usr/bin/env python3
"""Verify TE-7 (advanced/) coverage and Security-Sensitive markers for Tier-A GCL skills.

TE-7 requires that deep AIOps / FinOps / SecOps / safety content lives under
`references/advanced/` so SKILL.md and references/*.md stay focused on
agent-executable flows. This script:

- For every Tier-A GCL skill, asserts at least one `references/advanced/*.md`
  file (advanced coverage), unless the skill is exempted by
  `OPTIONAL_NO_ADVANCED` (read-only / governance skills where advanced depth
  is intentionally co-located with the runbook).
- For every `references/advanced/*.md` and `references/*.md`, looks for
  Security-Sensitive markers (English or Chinese) so reviewers can audit which
  destructive operations require explicit operator confirmation.
- Reports per-skill advanced coverage and overall Security-Sensitive marker
  counts; emits WARN-only by default so existing skills can adopt gradually.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

from check_gcl_conformance import GCL_SKILLS

ADVANCED_TOPICS = (
    "aiops",
    "cost",
    "security",
    "safety",
    "observability",
    "knowledge",
)
ADVANCED_FILENAME_PATTERN = re.compile(r"(?:^|-)(?:" + "|".join(ADVANCED_TOPICS) + r")(?:-|\.|$)", re.IGNORECASE)
SECURITY_SENSITIVE_PATTERNS = (
    re.compile(r"Security-Sensitive", re.IGNORECASE),
    re.compile(r"⚠"),
    re.compile(r"\b(?:warning|caution|danger)\b", re.IGNORECASE),
    re.compile(r"(高危|危险|敏感|不可逆|慎用)"),
)
EXEMPT_ADVANCED = frozenset(
    {
        # Meta-skill generator: advanced depth lives in dedicated references/
        # (rubric, prompt-templates, gcl-prompt-backbone, ...).
        "huaweicloud-skill-generator",
    }
)
EXEMPT_TOP_LEVEL_MARKERS = frozenset(
    {
        # Read-only assessment workers do not perform destructive ops; SKILL.md
        # already requires explicit confirmation at the call site.
    }
)


def discover_advanced_files(references_dir: Path) -> list[Path]:
    if not references_dir.is_dir():
        return []
    return sorted((references_dir / "advanced").glob("*.md"))


def collect_reference_files(references_dir: Path) -> list[Path]:
    if not references_dir.is_dir():
        return []
    return sorted(list(references_dir.glob("*.md")) + list((references_dir / "advanced").glob("*.md")))


def count_security_markers(text: str) -> int:
    return sum(len(pattern.findall(text)) for pattern in SECURITY_SENSITIVE_PATTERNS)


def validate_skill(root: Path, skill: str) -> dict[str, Any]:
    references = root / skill / "references"
    advanced_files = discover_advanced_files(references)
    advanced_topics = sorted(
        {
            match.group(0).lower().rstrip("-.")
            for file in advanced_files
            for match in ADVANCED_FILENAME_PATTERN.finditer(file.name)
        }
    )
    all_files = collect_reference_files(references)
    sec_marker_count = 0
    sec_marker_files: list[str] = []
    for path in all_files:
        text = path.read_text(encoding="utf-8")
        hits = count_security_markers(text)
        if hits:
            sec_marker_count += hits
            sec_marker_files.append(f"{path.relative_to(root)}={hits}")

    errors: list[str] = []
    warnings: list[str] = []

    if skill not in EXEMPT_ADVANCED and not advanced_files:
        errors.append(f"{skill}: missing references/advanced/*.md (TE-7 stratification)")

    if not sec_marker_count and skill not in EXEMPT_TOP_LEVEL_MARKERS:
        warnings.append(f"{skill}: no Security-Sensitive markers in any references/*.md")

    return {
        "skill": skill,
        "advanced_files": [str(path.relative_to(root)) for path in advanced_files],
        "advanced_topics": advanced_topics,
        "security_marker_count": sec_marker_count,
        "security_marker_files": sec_marker_files,
        "errors": errors,
        "warnings": warnings,
        "ok": not errors,
    }


def validate_all(root: Path) -> dict[str, Any]:
    reports = [validate_skill(root, skill) for skill in sorted(GCL_SKILLS)]
    all_errors = [error for report in reports for error in report["errors"]]
    all_warnings = [warning for report in reports for warning in report["warnings"]]
    covered = sum(1 for report in reports if report["advanced_files"])
    return {
        "ok": not all_errors,
        "skills_checked": len(reports),
        "skills_with_advanced": covered,
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
        help="Promote missing advanced/ to warnings (gradual rollout).",
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
            adv = len(item["advanced_files"])
            topics = ",".join(item["advanced_topics"]) or "-"
            print(
                f"{status}: {item['skill']:35s}  advanced={adv}  topics={topics:20s}  "
                f"sec_markers={item['security_marker_count']}"
            )
            for error in item["errors"]:
                print(f"  - {error}")
            for warning in item["warnings"]:
                print(f"  ~ {warning}")
        print(
            f"\nChecked {report['skills_checked']} skills; "
            f"with_advanced={report['skills_with_advanced']}; "
            f"errors={len(report['errors'])}; warnings={len(report['warnings'])}"
        )

    if not report["ok"]:
        return 1 if not args.warn_only else 0
    return 0


__all__ = [
    "validate_all",
    "validate_skill",
    "count_security_markers",
    "discover_advanced_files",
]


if __name__ == "__main__":
    sys.exit(main())
