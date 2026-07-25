#!/usr/bin/env python3
"""Validate local path references in Markdown files.

Scans all tracked `.md` files for relative links (`[text](path)`) and verifies
that the target file exists on disk. Reports broken references as errors.

Usage:
  python3 scripts/check_markdown_links.py
  python3 scripts/check_markdown_links.py --fix-suggestions
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

# Match markdown links: [text](target) — skip external URLs, anchors, and placeholders
LINK_RE = re.compile(r"\[([^\]]*)\]\(([^)]+)\)")

# Directories to skip (gitignored / generated)
SKIP_DIRS = {".git", "node_modules", "__pycache__", ".ruff_cache", ".pi-subagents", ".claude"}

# Files to skip (templates with placeholder links resolved at generation time)
SKIP_FILES = {"huaweicloud-skill-template.md"}


def should_skip(path: Path) -> bool:
    """Return True if the file is in a skipped directory or is a skipped file."""
    if path.name in SKIP_FILES:
        return True
    return any(part in SKIP_DIRS for part in path.parts)


# Strip fenced code blocks and inline code to avoid false positives
FENCED_CODE_RE = re.compile(r"^```[^\n]*\n.*?^```", re.MULTILINE | re.DOTALL)
INLINE_CODE_RE = re.compile(r"`[^`]+`")


def extract_local_links(text: str) -> list[str]:
    """Extract relative link targets from markdown text."""
    links: list[str] = []
    # Remove fenced code blocks first, then inline code spans
    cleaned = FENCED_CODE_RE.sub("", text)
    cleaned = INLINE_CODE_RE.sub("", cleaned)
    for _text, target in LINK_RE.findall(cleaned):
        # Skip external URLs, pure anchors, protocol links, and placeholders
        if target.startswith(("http://", "https://", "#", "mailto:", "{{")):
            continue
        # Strip anchor fragment from path
        path_part = target.split("#")[0]
        if not path_part:
            continue
        links.append(path_part)
    return links


def check_file(md_path: Path, root: Path) -> list[str]:
    """Check a single markdown file for broken local links. Returns error messages."""
    errors: list[str] = []
    try:
        text = md_path.read_text(encoding="utf-8")
    except OSError:
        return errors

    parent = md_path.parent
    for target in extract_local_links(text):
        resolved = (parent / target).resolve()
        if not resolved.exists():
            rel_md = md_path.relative_to(root)
            errors.append(f"  {rel_md}: broken link -> {target}")
    return errors


def collect_markdown_files(root: Path) -> list[Path]:
    """Collect all .md files under root, excluding skipped directories."""
    files: list[Path] = []
    for path in sorted(root.rglob("*.md")):
        if should_skip(path.relative_to(root)):
            continue
        files.append(path)
    return files


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Validate local path references in Markdown files.",
    )
    parser.add_argument(
        "--root",
        type=Path,
        default=ROOT,
        help="Repository root (default: auto-detected).",
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Print checked file count even when all pass.",
    )
    args = parser.parse_args(argv)

    root = args.root.resolve()
    md_files = collect_markdown_files(root)
    all_errors: list[str] = []

    for md_path in md_files:
        all_errors.extend(check_file(md_path, root))

    if all_errors:
        print(f"BROKEN LINKS ({len(all_errors)}):", file=sys.stderr)
        for err in all_errors:
            print(err, file=sys.stderr)
        return 1

    if args.verbose:
        print(f"OK: {len(md_files)} markdown files checked, 0 broken links.")
    else:
        print("OK: no broken local links.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
