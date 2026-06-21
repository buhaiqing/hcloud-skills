#!/usr/bin/env python3
"""Drift guard for the dual-copy `huaweicloud-skill-generator` skill.

The generator exists in two locations:

* ``huaweicloud-skill-generator/``  — root copy, **canonical, git-tracked**
* ``.agents/skills/huaweicloud-skill-generator/``  — agent runtime copy, **gitignored**

``AGENTS.md`` already documents the trap ("update the root copy; the runtime
copy may drift"). Without a programmatic check, drift accumulates silently and
the agent runtime loads stale instructions.

This guard:

1. **Requires both copies to exist** so the agent runtime can load the skill.
2. **Forces byte-for-byte equality** of every regular file (text or binary)
   under both roots. This is intentionally strict: any drift in the runtime
   copy produces a hard failure.
3. **Reports extra / missing files** in either root so the two trees stay
   symmetric.
4. Offers ``--fix`` to copy the canonical root to the runtime location, with
   ``--dry-run`` to preview.

Why a separate file from the audit-results guard? The runtime copy is loaded
by the agent itself, not produced by an audit tool; treating it as a skill
artifact is cleaner than mixing it with trace persistence.
"""

from __future__ import annotations

import argparse
import filecmp
import hashlib
import json
import shutil
import sys
from pathlib import Path
from typing import Any

ROOT_DEFAULT = Path(__file__).resolve().parents[1]
CANONICAL_REL = Path("huaweicloud-skill-generator")
RUNTIME_REL = Path(".agents/skills/huaweicloud-skill-generator")
SKIP_NAMES: frozenset[str] = frozenset({".DS_Store"})


def _iter_files(root: Path) -> list[Path]:
    if not root.is_dir():
        return []
    return sorted(p for p in root.rglob("*") if p.is_file() and p.name not in SKIP_NAMES)


def _rel(root: Path, path: Path) -> str:
    try:
        return str(path.relative_to(root))
    except ValueError:
        return str(path)


def _hash(path: Path) -> str:
    h = hashlib.sha256()
    h.update(path.read_bytes())
    return h.hexdigest()


def _collect_drift(canonical: Path, runtime: Path) -> dict[str, list[str]]:
    canonical_files = {_rel(canonical, p): p for p in _iter_files(canonical)}
    runtime_files = {_rel(runtime, p): p for p in _iter_files(runtime)}

    only_canonical = sorted(set(canonical_files) - set(runtime_files))
    only_runtime = sorted(set(runtime_files) - set(canonical_files))
    common = sorted(set(canonical_files) & set(runtime_files))

    differing: list[str] = []
    for rel in common:
        if not filecmp.cmp(canonical_files[rel], runtime_files[rel], shallow=False):
            differing.append(rel)
    return {
        "only_canonical": only_canonical,
        "only_runtime": only_runtime,
        "differing": differing,
    }


def check_drift(root: Path) -> dict[str, Any]:
    canonical = root / CANONICAL_REL
    runtime = root / RUNTIME_REL
    errors: list[str] = []
    if not canonical.is_dir():
        errors.append(f"{canonical}: canonical skill root missing")
    if not runtime.is_dir():
        errors.append(f"{runtime}: runtime skill root missing (agent runtime will fail to load)")
    if errors:
        return {"ok": False, "errors": errors, "drift": {}}
    drift = _collect_drift(canonical, runtime)
    if drift["only_canonical"]:
        errors.append(
            f"{runtime}: missing files: "
            + ", ".join(drift["only_canonical"][:5])
            + (" ..." if len(drift["only_canonical"]) > 5 else "")
        )
    if drift["only_runtime"]:
        errors.append(
            f"{runtime}: extra files not in canonical: "
            + ", ".join(drift["only_runtime"][:5])
            + (" ..." if len(drift["only_runtime"]) > 5 else "")
        )
    if drift["differing"]:
        errors.append(
            f"{runtime}: {len(drift['differing'])} file(s) drifted from canonical: "
            + ", ".join(drift["differing"][:5])
            + (" ..." if len(drift["differing"]) > 5 else "")
        )
    return {"ok": not errors, "errors": errors, "drift": drift}


def _sync_files(canonical: Path, runtime: Path, dry_run: bool) -> list[str]:
    actions: list[str] = []
    canonical_root = canonical
    runtime_root = runtime
    drift = _collect_drift(canonical_root, runtime_root)
    for rel in drift["only_canonical"]:
        src = canonical_root / rel
        dst = runtime_root / rel
        actions.append(f"copy {src} -> {dst}")
        if not dry_run:
            dst.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(src, dst)
    for rel in drift["differing"]:
        src = canonical_root / rel
        dst = runtime_root / rel
        actions.append(f"overwrite {dst} from {src}")
        if not dry_run:
            shutil.copy2(src, dst)
    for rel in drift["only_runtime"]:
        dst = runtime_root / rel
        actions.append(f"remove {dst} (not in canonical)")
        if not dry_run:
            dst.unlink()
    return actions


def sync(root: Path, dry_run: bool) -> dict[str, Any]:
    canonical = root / CANONICAL_REL
    runtime = root / RUNTIME_REL
    if not canonical.is_dir():
        return {"ok": False, "errors": [f"{canonical}: missing"], "actions": []}
    # The runtime copy is gitignored and may not exist on a fresh checkout
    # (e.g. CI). Bootstrap it on demand so `sync` is self-healing instead of
    # erroring out before any file copy runs.
    if not runtime.is_dir():
        actions: list[str] = [f"create {runtime}"]
        if not dry_run:
            runtime.mkdir(parents=True, exist_ok=True)
        actions.extend(_sync_files(canonical, runtime, dry_run=dry_run))
        return {"ok": True, "errors": [], "actions": actions, "dry_run": dry_run}
    actions = _sync_files(canonical, runtime, dry_run=dry_run)
    return {"ok": True, "errors": [], "actions": actions, "dry_run": dry_run}


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT_DEFAULT)
    parser.add_argument("--json", action="store_true")
    sub = parser.add_subparsers(dest="cmd", required=True)
    sub.add_parser("check", help="Drift check (default gate)")
    sync_p = sub.add_parser("sync", help="Reconcile runtime copy with canonical")
    sync_p.add_argument("--dry-run", action="store_true", help="Print actions without writing")
    sync_p.add_argument("--apply", action="store_true", help="Apply changes (default is dry-run)")
    return parser


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    if args.cmd == "sync":
        dry_run = not args.apply
        report = sync(root, dry_run=dry_run)
        if args.json:
            print(json.dumps(report, indent=2, ensure_ascii=False))
        else:
            for action in report["actions"]:
                print(("DRY-RUN: " if dry_run else "") + action)
            if not report["actions"]:
                print("no drift; nothing to do")
            elif not dry_run:
                print("synced")
        return 0 if report["ok"] else 1

    report = check_drift(root)
    if args.json:
        print(json.dumps(report, indent=2, ensure_ascii=False))
    else:
        if report["ok"]:
            print("[skill_generator drift] OK: runtime copy matches canonical")
        else:
            for err in report["errors"]:
                print(f"FAIL: {err}")
            print(
                f"\n[skill_generator drift] FAIL: {len(report['errors'])} issue(s); "
                f"run `python3 scripts/check_skill_generator_drift.py sync --apply`"
            )
    return 0 if report["ok"] else 1


__all__ = [
    "CANONICAL_REL",
    "RUNTIME_REL",
    "check_drift",
    "sync",
]


if __name__ == "__main__":
    sys.exit(main())
