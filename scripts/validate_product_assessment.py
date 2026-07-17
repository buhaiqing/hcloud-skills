#!/usr/bin/env python3
"""Validate Worker Output Contract example JSON in well-architected-assessment.md files."""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

from check_gcl_conformance import GCL_SKILLS

ASSESSMENT_GLOB = "huaweicloud-*/references/well-architected-assessment.md"
REQUIRED_TOP = {
    "skill_id",
    "product",
    "region",
    "scope",
    "assessment_date",
    "status",
    "partial",
    "resource_count",
    "pillars",
    "recommendations",
    "trace",
    "errors",
}
PILLARS = frozenset({"reliability", "security", "cost", "efficiency"})
PILLAR_PREFIX = {
    "rel": "reliability",
    "sec": "security",
    "cost": "cost",
    "eff": "efficiency",
}
FINDING_ID = re.compile(r"^([a-z0-9]+)-(rel|sec|cost|eff)-(\d{3})$")
STATUSES = frozenset({"OK", "PARTIAL", "ERROR"})
PILLAR_STATUS = frozenset({"assessed", "not_assessed", "skipped"})
SEVERITIES = frozenset({"Critical", "High", "Medium", "Low"})
CONFIDENCE = frozenset({"HIGH", "MEDIUM", "LOW"})
EFFORT = frozenset({"quick", "medium", "major"})

PRODUCT_BY_SKILL: dict[str, str] = {
    "huaweicloud-billing-ops": "billing",
    "huaweicloud-cbr-ops": "cbr",
    "huaweicloud-cce-ops": "cce",
    "huaweicloud-ces-ops": "ces",
    "huaweicloud-css-ops": "css",
    "huaweicloud-cts-ops": "cts",
    "huaweicloud-cdn-ops": "cdn",
    "huaweicloud-dcs-ops": "dcs",
    "huaweicloud-dms-ops": "dms",
    "huaweicloud-dns-ops": "dns",
    "huaweicloud-ecs-ops": "ecs",
    "huaweicloud-eip-ops": "eip",
    "huaweicloud-elb-ops": "elb",
    "huaweicloud-functiongraph-ops": "functiongraph",
    "huaweicloud-gaussdb-ops": "gaussdb",
    "huaweicloud-hss-ops": "hss",
    "huaweicloud-kms-ops": "kms",
    "huaweicloud-iam-ops": "iam",
    "huaweicloud-lts-ops": "lts",
    "huaweicloud-obs-ops": "obs",
    "huaweicloud-rds-ops": "rds",
    "huaweicloud-swr-ops": "swr",
    "huaweicloud-vpc-ops": "vpc",
    "huaweicloud-waf-ops": "waf",
}


def example_product_assessment(skill_id: str, product: str) -> dict[str, Any]:
    return {
        "skill_id": skill_id,
        "product": product,
        "region": "cn-north-4",
        "scope": "account-wide",
        "assessment_date": "2026-06-19T10:00:00+08:00",
        "status": "OK",
        "partial": False,
        "resource_count": 1,
        "pillars": {pillar: {"score": 80, "status": "assessed", "findings": []} for pillar in sorted(PILLARS)},
        "recommendations": [],
        "trace": {
            "commands": [f"hcloud {product} read-only-list --region cn-north-4 (HW_SECRET_ACCESS_KEY=<masked>)"],
            "request_ids": ["0123456789abcdef0123456789abcdef"],
        },
        "errors": [],
    }


def worker_contract_appendix(skill_id: str, product: str) -> str:
    example = json.dumps(example_product_assessment(skill_id, product), indent=2, ensure_ascii=False)
    return f"""
---

## Worker Output Contract (Read-Only Assessment Mode)

> Invoked when Well-Architected review sets `{{{{user.mode}}}}=well-architected-readonly`.
> Return **`{{{{output.product_assessment}}}}`** — field names MUST match the canonical schema.

**Canonical schema:** [worker-output-schema.md](../../huaweicloud-skill-generator/references/worker-output-schema.md)

| Constant | Value |
|----------|-------|
| `skill_id` | `{skill_id}` |
| `product` | `{product}` |
| Finding `id` pattern | `{product}-{{rel|sec|cost|eff}}-NNN` |

### Pillar → checklist map

| `pillars` key | Checklist source in this document |
|---------------|-------------------------------------|
| `reliability` | Stability / DR / backup sections |
| `security` | IAM / network / encryption sections |
| `cost` | FinOps / billing / idle detection sections |
| `efficiency` | Automation / batch / CI/CD sections |

### Example `{{{{output.product_assessment}}}}`

```json
{example}
```
"""


def extract_example_jsons(text: str) -> list[tuple[int, dict[str, Any] | object]]:
    pattern = re.compile(r"```json\s*\n(\{.*?\})\n```", re.DOTALL)
    blocks: list[tuple[int, dict[str, Any] | object]] = []
    for match in pattern.finditer(text):
        raw = match.group(1)
        if '"product"' not in raw or '"pillars"' not in raw:
            continue
        line_no = text[: match.start()].count("\n") + 1
        try:
            blocks.append((line_no, json.loads(raw)))
        except json.JSONDecodeError as exc:
            blocks.append((line_no, {"__parse_error__": str(exc)}))
    return blocks


def validate_finding(product: str, pillar_key: str, finding: dict[str, Any], path: str) -> list[str]:
    errors: list[str] = []
    finding_id = finding.get("id", "")
    match = FINDING_ID.match(str(finding_id))
    if not match:
        errors.append(f"{path}: finding id {finding_id!r} invalid (expected {{product}}-{{rel|sec|cost|eff}}-NNN)")
        return errors
    if match.group(1) != product:
        errors.append(f"{path}: finding id product prefix {match.group(1)!r} != top-level product {product!r}")
    expected_pillar = PILLAR_PREFIX[match.group(2)]
    if expected_pillar != pillar_key:
        errors.append(
            f"{path}: finding {finding_id!r} in pillars.{pillar_key} but id implies pillars.{expected_pillar}"
        )
    for field in ("severity", "confidence", "title", "evidence", "recommendation", "effort"):
        if field not in finding:
            errors.append(f"{path}: finding {finding_id!r} missing {field!r}")
    if finding.get("severity") not in SEVERITIES:
        errors.append(f"{path}: finding {finding_id!r} bad severity")
    if finding.get("confidence") not in CONFIDENCE:
        errors.append(f"{path}: finding {finding_id!r} bad confidence")
    if finding.get("effort") not in EFFORT:
        errors.append(f"{path}: finding {finding_id!r} bad effort")
    return errors


def validate_assessment(data: object, source: str) -> list[str]:
    errors: list[str] = []
    if not isinstance(data, dict):
        return [f"{source}: not a JSON object"]
    if "__parse_error__" in data:
        return [f"{source}: JSON parse error: {data['__parse_error__']}"]

    missing = REQUIRED_TOP - set(data.keys())
    if missing:
        errors.append(f"{source}: missing top-level fields: {sorted(missing)}")

    if data.get("status") not in STATUSES:
        errors.append(f"{source}: invalid status {data.get('status')!r}")

    product = data.get("product")
    if not isinstance(product, str) or not product:
        errors.append(f"{source}: product must be non-empty string")

    skill_id = data.get("skill_id")
    if (
        isinstance(skill_id, str)
        and isinstance(product, str)
        and skill_id in PRODUCT_BY_SKILL
        and PRODUCT_BY_SKILL[skill_id] != product
    ):
        errors.append(f"{source}: product {product!r} != registry code for {skill_id!r}")

    pillars = data.get("pillars")
    if not isinstance(pillars, dict):
        errors.append(f"{source}: pillars must be object")
        return errors

    for pillar_key, pillar_value in pillars.items():
        if pillar_key not in PILLARS:
            errors.append(f"{source}: unknown pillar key {pillar_key!r}")
            continue
        if not isinstance(pillar_value, dict):
            errors.append(f"{source}: pillars.{pillar_key} must be object")
            continue
        pillar_status = pillar_value.get("status")
        if pillar_status not in PILLAR_STATUS:
            errors.append(f"{source}: pillars.{pillar_key}.status invalid {pillar_status!r}")
        findings = pillar_value.get("findings", [])
        if not isinstance(findings, list):
            errors.append(f"{source}: pillars.{pillar_key}.findings must be array")
            continue
        if isinstance(product, str):
            for index, finding in enumerate(findings):
                if isinstance(finding, dict):
                    finding_path = f"{source} pillars.{pillar_key}[{index}]"
                    errors.extend(validate_finding(product, pillar_key, finding, finding_path))

    recommendations = data.get("recommendations", [])
    if isinstance(recommendations, list):
        for index, item in enumerate(recommendations):
            if not isinstance(item, dict):
                errors.append(f"{source}: recommendations[{index}] not object")
                continue
            if item.get("pillar") not in PILLARS:
                errors.append(f"{source}: recommendations[{index}].pillar invalid")

    trace = data.get("trace")
    if isinstance(trace, dict):
        commands = trace.get("commands", [])
        if isinstance(commands, list):
            for command in commands:
                if isinstance(command, str) and "SECRET" in command.upper() and "<masked>" not in command:
                    errors.append(f"{source}: trace.commands contains unmasked secret reference")

    return errors


def discover_assessment_files(root: Path) -> list[Path]:
    return sorted(root.glob(ASSESSMENT_GLOB))


def validate_file(path: Path, root: Path, *, required: bool) -> list[str]:
    rel = path.relative_to(root)
    skill_name = rel.parts[0]
    errors: list[str] = []

    if not path.is_file():
        if required:
            errors.append(f"{rel}: missing well-architected-assessment.md for Tier-A skill {skill_name!r}")
        return errors

    text = path.read_text(encoding="utf-8")
    if "Worker Output Contract" not in text:
        errors.append(f"{rel}: missing 'Worker Output Contract' section")
    examples = extract_example_jsons(text)
    if not examples:
        errors.append(f"{rel}: no product_assessment JSON example found")
        return errors

    for line_no, payload in examples:
        errors.extend(validate_assessment(payload, f"{rel}:{line_no}"))

    expected_product = PRODUCT_BY_SKILL.get(skill_name)
    for _, payload in examples:
        if isinstance(payload, dict) and payload.get("skill_id") != skill_name:
            errors.append(f"{rel}: example skill_id {payload.get('skill_id')!r} != {skill_name!r}")
        if expected_product and isinstance(payload, dict) and payload.get("product") != expected_product:
            errors.append(f"{rel}: example product {payload.get('product')!r} != {expected_product!r}")

    return errors


def validate_all(root: Path) -> dict[str, Any]:
    errors: list[str] = []
    files_checked = 0
    examples_checked = 0

    for skill in sorted(GCL_SKILLS):
        path = root / skill / "references" / "well-architected-assessment.md"
        files_checked += 1
        file_errors = validate_file(path, root, required=True)
        if path.is_file():
            examples_checked += len(extract_example_jsons(path.read_text(encoding="utf-8")))
        errors.extend(file_errors)

    return {
        "ok": not errors,
        "files_checked": files_checked,
        "examples_checked": examples_checked,
        "errors": errors,
    }


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--json", action="store_true")
    return parser


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    report = validate_all(root)
    if args.json:
        print(json.dumps(report, indent=2, ensure_ascii=False))
    elif report["ok"]:
        print(f"OK: {report['files_checked']} files, {report['examples_checked']} example JSON blocks validated")
    else:
        print(
            f"FAIL: {len(report['errors'])} error(s) in {report['files_checked']} files "
            f"({report['examples_checked']} examples)\n"
        )
        for error in report["errors"]:
            print(f"  - {error}")
    return 0 if report["ok"] else 1


__all__ = [
    "PRODUCT_BY_SKILL",
    "example_product_assessment",
    "extract_example_jsons",
    "validate_all",
    "validate_assessment",
    "validate_file",
    "worker_contract_appendix",
]


if __name__ == "__main__":
    sys.exit(main())
