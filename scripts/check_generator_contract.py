#!/usr/bin/env python3
"""Validate huaweicloud-skill-generator GCL template contract.

This guards future skill generation against regressions: new/updated skills must
continue to inherit GCL metadata, rubric artifacts, prompt templates, sanitized
operation intent, and isolated Critic constraints.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

CONTRACT_ITEMS: tuple[tuple[str, str, str], ...] = (
    ("template", "metadata.gcl.required", r"(?m)^  gcl:\n    required: true"),
    ("template", "metadata.gcl.default_max_iter", r"(?m)^    default_max_iter: 2"),
    ("template", "metadata.gcl.rubric_version", r"(?m)^    rubric_version: \"v1\""),
    ("template", "metadata.gcl.trace_path", r"audit-results/gcl-trace-YYYYMMDD-HHMMSS\.json"),
    ("template", "quality_gate_heading", r"(?m)^## Quality Gate \(GCL\)$"),
    ("template", "rubric_artifact", r"references/rubric\.md"),
    ("template", "prompt_templates_artifact", r"references/prompt-templates\.md"),
    ("template", "operation_intent", r"operation_intent"),
    ("template", "shared_backbone_reference", r"gcl-prompt-backbone\.md"),
    ("generator_skill", "compat_backbone", r"references/gcl-prompt-backbone\.md"),
    ("generator_skill", "output_rubric", r"`references/rubric\.md`"),
    ("generator_skill", "output_prompt_templates", r"`references/prompt-templates\.md`"),
    ("generator_skill", "metadata_gcl_instruction", r"`metadata\.gcl`"),
    ("backbone", "generator_section", r"(?m)^## 1\. Generator prompt template$"),
    ("backbone", "critic_section", r"(?m)^## 2\. Critic prompt template$"),
    ("backbone", "orchestrator_section", r"(?m)^## 3\. Orchestrator prompt template$"),
    ("backbone", "hcloud_primary", r"PRIMARY: hcloud"),
    ("backbone", "go_sdk_fallback", r"huaweicloud-sdk-go-v3"),
    ("backbone", "operation_intent", r"\{\{output\.operation_intent\}\}"),
    ("backbone", "critic_no_raw_request", r"Do NOT consider the original user request"),
    ("backbone", "critic_read_only", r"read-only"),
    ("backbone", "trace_persistence", r"audit-results/gcl-trace-YYYYMMDD-HHMMSS\.json"),
)

REQUIRED_FILES = {
    "template": Path("huaweicloud-skill-generator/references/huaweicloud-skill-template.md"),
    "generator_skill": Path("huaweicloud-skill-generator/SKILL.md"),
    "backbone": Path("huaweicloud-skill-generator/references/gcl-prompt-backbone.md"),
}


def read_files(root: Path) -> tuple[dict[str, str], list[dict[str, str]]]:
    texts: dict[str, str] = {}
    failures: list[dict[str, str]] = []
    for key, rel_path in REQUIRED_FILES.items():
        path = root / rel_path
        if not path.is_file():
            failures.append({"scope": key, "item": "file_exists", "path": str(rel_path), "reason": "missing file"})
            texts[key] = ""
            continue
        texts[key] = path.read_text(encoding="utf-8")
    return texts, failures


def has_bare_placeholders(text: str) -> bool:
    allowed = re.sub(r"\{\{\s*(env|user|output)\.[^{}]+\}\}", "", text)
    allowed = re.sub(r"\$\{[A-Z_][A-Z0-9_]*\}", "", allowed)
    return bool(re.search(r"(?<!\{)\{[a-zA-Z_][a-zA-Z0-9_.-]*\}(?!\})", allowed))


def check_contract(root: Path) -> dict[str, Any]:
    texts, failures = read_files(root)
    checks: list[dict[str, Any]] = []

    for scope, item, pattern in CONTRACT_ITEMS:
        ok = bool(re.search(pattern, texts.get(scope, "")))
        check = {"scope": scope, "item": item, "ok": ok}
        checks.append(check)
        if not ok:
            failures.append({"scope": scope, "item": item, "reason": f"pattern not found: {pattern}"})

    for scope in ("template", "backbone"):
        ok = not has_bare_placeholders(texts.get(scope, ""))
        checks.append({"scope": scope, "item": "no_bare_placeholders", "ok": ok})
        if not ok:
            failures.append({"scope": scope, "item": "no_bare_placeholders", "reason": "bare {placeholder} detected"})

    ok = not failures
    return {
        "ok": ok,
        "summary": {
            "total": len(checks),
            "passing": sum(1 for check in checks if check["ok"]),
            "failing": len(failures),
        },
        "checks": checks,
        "failures": failures,
    }


def format_human(report: dict[str, Any]) -> str:
    summary = report["summary"]
    lines = [f"Generator GCL contract: {summary['passing']}/{summary['total']} checks pass."]
    for failure in report["failures"]:
        lines.append(f"  FAIL {failure['scope']}.{failure['item']}: {failure['reason']}")
    return "\n".join(lines) + "\n"


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--json", action="store_true")
    return parser


def main() -> int:
    args = build_parser().parse_args()
    report = check_contract(args.root.resolve())
    if args.json:
        print(json.dumps(report, indent=2, sort_keys=True, ensure_ascii=False))
    else:
        print(format_human(report), end="")
    return 0 if report["ok"] else 1


if __name__ == "__main__":
    sys.exit(main())
