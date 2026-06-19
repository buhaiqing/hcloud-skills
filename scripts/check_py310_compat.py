#!/usr/bin/env python3
"""Verify that every Python script under ``scripts/`` compiles on Python 3.10.

CI runs the repository on Python 3.11, but the agent runtime environment is
still on 3.10. Any script that uses a 3.11-only syntax feature (PEP 695 type
aliases, ``tomllib``, ``Self`` without ``from __future__``, etc.) silently
breaks the agent. This gate fails fast by invoking ``python3.10 -m py_compile``
on every script and reporting any compile error.

Usage:
  python3 scripts/check_py310_compat.py
  python3 scripts/check_py310_compat.py --python-bin python3.10
  python3 scripts/check_py310_compat.py scripts/check_gcl_trace_security.py
"""

from __future__ import annotations

import argparse
import json
import shutil
import subprocess
import sys
from pathlib import Path

DEFAULT_SCRIPT_DIR = Path("scripts")
PYTHON_BIN_CANDIDATES = ("python3.10", "python310")


def resolve_python_bin(explicit: str | None) -> str:
    candidates = [explicit] if explicit else list(PYTHON_BIN_CANDIDATES)
    for candidate in candidates:
        if candidate and shutil.which(candidate):
            return candidate
    raise SystemExit(
        "ERROR: no Python 3.10 interpreter found. Tried: "
        + ", ".join(c for c in candidates if c)
        + ". Install 3.10 or pass --python-bin <path>"
    )


def discover_scripts(root: Path, explicit: list[Path]) -> list[Path]:
    if explicit:
        return [path.resolve() for path in explicit if path.is_file()]
    return sorted(path.resolve() for path in (root / DEFAULT_SCRIPT_DIR).glob("*.py"))


def compile_one(python_bin: str, script: Path) -> tuple[bool, str]:
    proc = subprocess.run(
        [python_bin, "-m", "py_compile", str(script)],
        capture_output=True,
        text=True,
    )
    if proc.returncode == 0:
        return True, ""
    message = (proc.stderr or proc.stdout).strip()
    return False, message


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument(
        "--python-bin",
        default=None,
        help="Python interpreter to use (default: first available among python3.10/python310).",
    )
    parser.add_argument("--json", action="store_true")
    parser.add_argument("script", nargs="*", type=Path)
    return parser


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    python_bin = resolve_python_bin(args.python_bin)
    scripts = discover_scripts(root, args.script)
    if not scripts:
        if args.json:
            print(json.dumps({"ok": True, "python_bin": python_bin, "results": []}, indent=2))
        else:
            print("OK: no Python scripts found")
        return 0

    results: list[dict[str, str]] = []
    for script in scripts:
        ok, message = compile_one(python_bin, script)
        try:
            display = str(script.relative_to(root))
        except ValueError:
            display = str(script)
        results.append({"script": display, "ok": ok, "error": message})

    ok = all(result["ok"] for result in results)
    if args.json:
        print(
            json.dumps(
                {"ok": ok, "python_bin": python_bin, "results": results},
                indent=2,
                ensure_ascii=False,
            )
        )
    else:
        for result in results:
            status = "OK" if result["ok"] else "FAIL"
            print(f"{status}: {result['script']}")
            if not result["ok"]:
                print(f"  - {result['error']}")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
