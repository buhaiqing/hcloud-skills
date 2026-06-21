#!/usr/bin/env python3
"""Verify that every Python script under ``scripts/`` runs on Python 3.10.

CI lints the repository with Python 3.11, but the agent runtime environment is
still on 3.10. Any script that uses a 3.11-only feature silently breaks the
agent. This gate runs **two** checks per script under ``python3.10``:

1. ``py_compile`` — catches parse-time syntax errors (PEP 695, etc.).
2. **Import dry-run** — actually loads the module via ``importlib`` so that
   import-time 3.11-only stdlib names (``from datetime import UTC``,
   ``import tomllib``, ``typing.Self`` without ``from __future__``, etc.) are
   caught. ``py_compile`` alone misses these because the syntax is valid in
   3.10; only resolution fails at runtime. See AGENTS.md §Python 3.10 Syntax
   Compatibility for the forbidden-symbol list.

Both checks run in fresh subprocesses so module-level state never leaks
between scripts.

Usage:
  python3 scripts/check_py310_compat.py
  python3 scripts/check_py310_compat.py --python-bin python3.10
  python3 scripts/check_py310_compat.py --no-import-check scripts/check_x.py
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


IMPORT_PROBE_TEMPLATE = (
    "import importlib.util, sys, pathlib; "
    "p = pathlib.Path({path!r}); "
    "sys.path.insert(0, str(p.parent)); "
    "spec = importlib.util.spec_from_file_location('_py310_probe', p); "
    "mod = importlib.util.module_from_spec(spec); "
    # Register the module in sys.modules BEFORE exec_module so that decorators
    # like @dataclass (which introspect sys.modules[cls.__module__]) work on
    # Python 3.10. Without this, scripts that combine
    # `from __future__ import annotations` with @dataclass raise a confusing
    # AttributeError on 3.10 when loaded via importlib. This mirrors the
    # normal "run as __main__" behaviour where the module is auto-registered.
    "sys.modules[spec.name] = mod; "
    "spec.loader.exec_module(mod); "
    "sys.exit(0)"
)


def import_one(python_bin: str, script: Path) -> tuple[bool, str]:
    """Run a 3.10 import dry-run for ``script``.

    We deliberately do NOT just ``python3.10 -c 'import script_name'`` because
    module names with ``-`` and the test files (``*_test.py``) cannot be
    imported by their bare name. ``importlib.util.spec_from_file_location``
    loads the file by path so every script under ``scripts/`` can be probed
    uniformly.
    """
    proc = subprocess.run(
        [python_bin, "-c", IMPORT_PROBE_TEMPLATE.format(path=str(script))],
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
    parser.add_argument(
        "--no-import-check",
        action="store_true",
        help="Skip the import dry-run (syntax check only). Use only when bisecting a gate failure.",
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
        try:
            display = str(script.relative_to(root))
        except ValueError:
            display = str(script)
        ok, message = compile_one(python_bin, script)
        if not ok:
            results.append({"script": display, "ok": False, "stage": "compile", "error": message})
            continue
        if not args.no_import_check:
            ok, message = import_one(python_bin, script)
            if not ok:
                results.append({"script": display, "ok": False, "stage": "import", "error": message})
                continue
        results.append({"script": display, "ok": True, "stage": "ok", "error": ""})

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
                print(f"  - [{result['stage']}] {result['error']}")
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
