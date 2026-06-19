#!/usr/bin/env python3
"""Verify `gcl_quality` thresholds are consistently wired across the GCL pipeline.

`scripts/gcl_alarm_wire.py` reads `gcl_quality:` block from
`huaweicloud-ces-ops/assets/example-config.yaml` to drive CES alarm rules.
This guard ensures:

1. `example-config.yaml` for `huaweicloud-ces-ops` defines the canonical
   `gcl_quality` block (`pass_rate_warn`, `pass_rate_critical`,
   `max_iter_warn_count`, `safety_fail_alert`).
2. The block's numeric thresholds are consistent with the defaults hard-coded
   in `gcl_alarm_wire.DEFAULT_THRESHOLDS` (warn > critical; max_iter >= 1;
   safety_fail_alert is boolean).
3. The same defaults documented in `docs/gcl-spec.md` (Phase 4 SLO section).
4. Any `audit-results/gcl-alarm-plan-*.json` already on disk MUST also agree
   with the wiring config (catches stale plans that drifted after a config
   change).
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

import gcl_alarm_wire as gaw
from check_gcl_conformance import GCL_SKILLS

CES_SKILL = "huaweicloud-ces-ops"
CONFIG_RELATIVE = Path("huaweicloud-ces-ops/assets/example-config.yaml")
GCL_SPEC_RELATIVE = Path("docs/gcl-spec.md")
PLAN_GLOB = "audit-results/gcl-alarm-plan-*.json"

NUMERIC_KEYS: tuple[str, ...] = (
    "pass_rate_warn",
    "pass_rate_critical",
    "max_iter_warn_count",
)
BOOLEAN_KEYS: tuple[str, ...] = ("safety_fail_alert",)
DOC_FRAGMENTS: tuple[str, ...] = (
    "pass_rate_warn",
    "pass_rate_critical",
    "max_iter_warn_count",
    "safety_fail_alert",
)


def _strip_yaml_block(text: str) -> str:
    """Return the first ```yaml``` fenced block, or the raw text if none."""
    match = re.search(r"```yaml\s*\n(.*?)\n```", text, re.DOTALL)
    if match:
        return match.group(1)
    return text


def load_wiring_config(config_path: Path) -> dict[str, Any] | None:
    if not config_path.is_file():
        return None
    block = _strip_yaml_block(config_path.read_text(encoding="utf-8"))
    thresholds = gaw.load_thresholds_from_config_for_check(block)
    if not thresholds:
        return None
    return thresholds


def check_config(config_path: Path) -> tuple[bool, list[str]]:
    errors: list[str] = []
    if not config_path.is_file():
        return False, [f"{config_path}: missing canonical CES gcl_quality config"]
    thresholds = load_wiring_config(config_path)
    if thresholds is None:
        return False, [f"{config_path}: no `gcl_quality:` block found"]

    defaults = gaw.DEFAULT_THRESHOLDS
    for key in NUMERIC_KEYS:
        if key not in thresholds:
            errors.append(f"{config_path}: gcl_quality.{key} missing")
            continue
        try:
            value = float(thresholds[key])
        except (TypeError, ValueError):
            errors.append(f"{config_path}: gcl_quality.{key}={thresholds[key]!r} is not numeric")
            continue
        default = float(defaults[key])
        if abs(value - default) > 1e-9:
            errors.append(
                f"{config_path}: gcl_quality.{key}={value} drifts from "
                f"gcl_alarm_wire.DEFAULT_THRESHOLDS[{key!r}]={default}; keep them aligned"
            )
    for key in BOOLEAN_KEYS:
        if key not in thresholds:
            errors.append(f"{config_path}: gcl_quality.{key} missing")
            continue
        if not isinstance(thresholds[key], bool):
            errors.append(f"{config_path}: gcl_quality.{key}={thresholds[key]!r} is not boolean")

    warn = float(thresholds.get("pass_rate_warn", defaults["pass_rate_warn"]))
    critical = float(thresholds.get("pass_rate_critical", defaults["pass_rate_critical"]))
    if not (0.0 <= critical <= warn <= 1.0):
        errors.append(
            f"{config_path}: invalid pass_rate ordering critical={critical} warn={warn}; "
            "require 0 <= critical <= warn <= 1"
        )
    max_iter = thresholds.get("max_iter_warn_count", defaults["max_iter_warn_count"])
    try:
        if int(max_iter) < 1:
            errors.append(f"{config_path}: max_iter_warn_count must be >= 1, got {max_iter}")
    except (TypeError, ValueError):
        pass

    return not errors, errors


def check_doc(root: Path) -> tuple[bool, list[str]]:
    doc = root / GCL_SPEC_RELATIVE
    errors: list[str] = []
    if not doc.is_file():
        return False, [f"{doc}: missing"]
    text = doc.read_text(encoding="utf-8")
    for fragment in DOC_FRAGMENTS:
        if fragment not in text:
            errors.append(f"{doc}: missing documented threshold {fragment!r}")
    return not errors, errors


def check_existing_plans(root: Path, wiring: dict[str, Any] | None) -> tuple[bool, list[str]]:
    errors: list[str] = []
    plan_paths = sorted(root.glob(PLAN_GLOB))
    if not plan_paths:
        return True, []
    if wiring is None:
        return False, [f"plans exist ({len(plan_paths)}) but wiring config missing"]

    latest = plan_paths[-1]
    try:
        plan = json.loads(latest.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        return False, [f"{latest}: invalid JSON ({exc})"]

    plan_thresholds = plan.get("thresholds", {})
    if not isinstance(plan_thresholds, dict):
        return False, [f"{latest}: missing top-level 'thresholds'"]

    for key in NUMERIC_KEYS:
        plan_value = plan_thresholds.get(key)
        if plan_value is None:
            continue
        try:
            plan_f = float(plan_value)
        except (TypeError, ValueError):
            errors.append(f"{latest}: thresholds.{key}={plan_value!r} not numeric")
            continue
        wire_f = float(wiring.get(key, gaw.DEFAULT_THRESHOLDS[key]))
        if abs(plan_f - wire_f) > 1e-9:
            errors.append(
                f"{latest}: thresholds.{key}={plan_f} disagrees with wiring config "
                f"({wire_f}); rerun `gcl_alarm_wire.py plan` after editing gcl_quality"
            )
    return not errors, errors


def check_all(root: Path) -> dict[str, Any]:
    sections: dict[str, tuple[bool, list[str]]] = {}
    config_path = root / CONFIG_RELATIVE
    if CES_SKILL not in GCL_SKILLS:
        sections["config"] = (False, [f"{CES_SKILL} is not in GCL_SKILLS"])
    else:
        sections["config"] = check_config(config_path)
    sections["doc"] = check_doc(root)
    wiring = load_wiring_config(config_path) if config_path.is_file() else None
    sections["plans"] = check_existing_plans(root, wiring)

    all_errors = [error for _, (ok, errors) in sections.items() for error in errors if not ok]
    return {
        "ok": not all_errors,
        "sections": {name: {"ok": ok, "errors": errors} for name, (ok, errors) in sections.items()},
        "errors": all_errors,
    }


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
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
            for error in section["errors"]:
                print(f"  - {error}")
        if report["ok"]:
            print("\n[gcl_alarm_wire contract] OK")
        else:
            print(f"\n[gcl_alarm_wire contract] FAIL: {len(report['errors'])} issue(s)")
    return 0 if report["ok"] else 1


__all__ = [
    "check_all",
    "check_config",
    "check_doc",
    "check_existing_plans",
    "load_wiring_config",
]


if __name__ == "__main__":
    sys.exit(main())
