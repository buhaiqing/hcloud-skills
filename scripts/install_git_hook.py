#!/usr/bin/env python3
"""Install the repository-managed git pre-commit hook.

The hook lives under ``.githooks/pre-commit`` (kept in git) and is copied
into ``.git/hooks/pre-commit`` by this script. The destination file is owned
by the user, not the repo, so re-running this script is idempotent and safe.

Usage:
  python3 scripts/install_git_hook.py            # install (default)
  python3 scripts/install_git_hook.py --uninstall # remove the hook
  python3 scripts/install_git_hook.py --check    # print status and exit
"""

from __future__ import annotations

import argparse
import shutil
import stat
import sys
from pathlib import Path

SOURCE = Path(".githooks/pre-commit")
DEST_NAME = "pre-commit"


def find_git_dir(root: Path) -> Path:
    candidate = root / ".git"
    if candidate.is_dir():
        return candidate
    raise SystemExit("ERROR: .git/ not found; run this from inside a git work tree.")


def install(root: Path) -> int:
    git_dir = find_git_dir(root)
    source = root / SOURCE
    if not source.is_file():
        print(f"ERROR: missing source hook at {source}", file=sys.stderr)
        return 1
    dest = git_dir / "hooks" / DEST_NAME
    dest.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(source, dest)
    dest.chmod(dest.stat().st_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)
    print(f"installed: {source} -> {dest}")
    return 0


def uninstall(root: Path) -> int:
    git_dir = find_git_dir(root)
    dest = git_dir / "hooks" / DEST_NAME
    if not dest.exists():
        print(f"not installed: {dest} does not exist")
        return 0
    dest.unlink()
    print(f"removed: {dest}")
    return 0


def check(root: Path) -> int:
    git_dir = find_git_dir(root)
    dest = git_dir / "hooks" / DEST_NAME
    if dest.exists():
        print(f"OK: {dest} is installed")
        return 0
    print(f"MISSING: {dest} not installed; run `python3 scripts/install_git_hook.py`")
    return 1


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--uninstall", action="store_true", help="Remove the installed hook")
    parser.add_argument("--check", action="store_true", help="Print install status and exit")
    return parser


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    if args.check:
        return check(root)
    if args.uninstall:
        return uninstall(root)
    return install(root)


if __name__ == "__main__":
    sys.exit(main())
