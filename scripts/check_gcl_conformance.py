#!/usr/bin/env python3
"""GCL Tier-A conformance checker for hcloud-skills.

Verifies each `huaweicloud-*-ops` skill ships the canonical GCL artifact set:

- `references/rubric.md` with numbered sections `## 1.` through `## 8.`
- `references/prompt-templates.md` with numbered sections `## 1.` through `## 7.`
- `SKILL.md` containing `## Quality Gate (GCL)`

Usage:
  python3 scripts/check_gcl_conformance.py
  python3 scripts/check_gcl_conformance.py --json
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

GCL_SKILLS: frozenset[str] = frozenset(
    {
        "huaweicloud-billing-ops",
        "huaweicloud-cbr-ops",
        "huaweicloud-cce-ops",
        "huaweicloud-ces-ops",
        "huaweicloud-css-ops",
        "huaweicloud-cts-ops",
        "huaweicloud-dcs-ops",
        "huaweicloud-dms-ops",
        "huaweicloud-ecs-ops",
        "huaweicloud-eip-ops",
        "huaweicloud-elb-ops",
        "huaweicloud-functiongraph-ops",
        "huaweicloud-gaussdb-ops",
        "huaweicloud-hss-ops",
        "huaweicloud-iam-ops",
        "huaweicloud-lts-ops",
        "huaweicloud-obs-ops",
        "huaweicloud-rds-ops",
        "huaweicloud-swr-ops",
        "huaweicloud-vpc-ops",
        "huaweicloud-waf-ops",
    }
)


def _count_numbered_sections(text: str, target: int) -> int:
    for number in range(1, target + 1):
        if not re.search(rf"^## {number}\. ", text, re.MULTILINE):
            return 0
    return target


def _has_bare_placeholders(text: str) -> bool:
    fenced = re.sub(r"```[\s\S]*?```", "", text)
    return bool(re.search(r"(?<!\{)\{[a-zA-Z_][a-zA-Z0-9_.-]*\}(?!\})", fenced))


def check_skill(root: Path, skill: str) -> dict[str, Any]:
    skill_dir = root / skill
    rubric_path = skill_dir / "references" / "rubric.md"
    prompt_path = skill_dir / "references" / "prompt-templates.md"
    skill_path = skill_dir / "SKILL.md"

    rubric_text = rubric_path.read_text(encoding="utf-8") if rubric_path.is_file() else ""
    prompt_text = prompt_path.read_text(encoding="utf-8") if prompt_path.is_file() else ""
    skill_text = skill_path.read_text(encoding="utf-8") if skill_path.is_file() else ""

    rubric_sections = _count_numbered_sections(rubric_text, 8) if rubric_text else 0
    prompt_sections = _count_numbered_sections(prompt_text, 7) if prompt_text else 0
    has_quality_gate = bool(re.search(r"^## Quality Gate \(GCL\)$", skill_text, re.MULTILINE))
    prompt_has_operation_intent = "{{output.operation_intent}}" in prompt_text or "operation_intent" in prompt_text
    prompt_has_no_bare_placeholders = not _has_bare_placeholders(prompt_text)

    rubric_ok = rubric_sections == 8
    prompt_ok = prompt_sections == 7 and prompt_has_operation_intent and prompt_has_no_bare_placeholders
    skill_ok = has_quality_gate

    return {
        "skill": skill,
        "rubric_sections": rubric_sections,
        "prompt_sections": prompt_sections,
        "has_quality_gate": has_quality_gate,
        "prompt_has_operation_intent": prompt_has_operation_intent,
        "prompt_has_no_bare_placeholders": prompt_has_no_bare_placeholders,
        "rubric_ok": rubric_ok,
        "prompt_ok": prompt_ok,
        "skill_ok": skill_ok,
        "ok": rubric_ok and prompt_ok and skill_ok,
    }


def check_all(root: Path) -> list[dict[str, Any]]:
    return [check_skill(root, skill) for skill in sorted(GCL_SKILLS)]


def _format_human(reports: list[dict[str, Any]]) -> str:
    passing = sum(1 for report in reports if report["ok"])
    total = len(reports)
    lines = [f"GCL conformance: {passing}/{total} skills conform."]
    for report in reports:
        if report["ok"]:
            continue
        reasons: list[str] = []
        if not report["rubric_ok"]:
            reasons.append(f"rubric_sections={report['rubric_sections']}/8")
        if not report["prompt_ok"]:
            reasons.append(f"prompt_sections={report['prompt_sections']}/7")
            if not report["prompt_has_operation_intent"]:
                reasons.append("missing operation_intent in prompt templates")
            if not report["prompt_has_no_bare_placeholders"]:
                reasons.append("bare placeholder detected")
        if not report["skill_ok"]:
            reasons.append("missing `## Quality Gate (GCL)` in SKILL.md")
        lines.append(f"  FAIL {report['skill']}: {', '.join(reasons)}")
    return "\n".join(lines) + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    reports = check_all(args.root)
    passing = sum(1 for report in reports if report["ok"])
    if args.json:
        print(
            json.dumps(
                {
                    "summary": {"total": len(reports), "passing": passing, "failing": len(reports) - passing},
                    "reports": reports,
                },
                indent=2,
                sort_keys=True,
            )
        )
    else:
        print(_format_human(reports), end="")
    return 0 if passing == len(reports) else 1


if __name__ == "__main__":
    sys.exit(main())
