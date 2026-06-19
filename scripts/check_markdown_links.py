#!/usr/bin/env python3
"""Validate local Markdown links and explicit repository path references.

The checker focuses on always-loaded entry docs and top-level extracted specs:
`AGENTS.md`, `README*.md`, and `docs/*.md`. Product skill docs intentionally
contain many intra-skill historical links and are excluded to avoid noisy checks.
"""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path

IGNORED_DIR_PARTS = {".git", ".github", ".omc", ".omo", ".codebuddy", ".claude", ".agents", "audit-results"}
PATH_PREFIXES = (
    "AGENTS.md",
    "CLAUDE.md",
    "README.md",
    "README_CN.md",
    "LICENSE",
    "docs/",
    "scripts/",
    "huaweicloud-",
    ".github/",
)

LINK_RE = re.compile(r"(?<!!)\[[^\]]+\]\(([^)\s]+)(?:\s+\"[^\"]*\")?\)")
BACKTICK_RE = re.compile(r"`([^`]+)`")


@dataclass(frozen=True)
class Finding:
    file: Path
    line: int
    target: str
    reason: str


def iter_markdown_files(root: Path) -> list[Path]:
    candidates = [root / "AGENTS.md", root / "CLAUDE.md", root / "README.md", root / "README_CN.md"]
    docs_dir = root / "docs"
    if docs_dir.is_dir():
        candidates.extend(sorted(docs_dir.glob("*.md")))
    files: list[Path] = []
    for path in candidates:
        if not path.is_file():
            continue
        if any(part in IGNORED_DIR_PARTS for part in path.relative_to(root).parts):
            continue
        files.append(path)
    return sorted(files)


def normalize_target(raw: str) -> str | None:
    target = raw.strip()
    if not target or target.startswith(("http://", "https://", "mailto:", "#", "{{", "<")):
        return None
    target = target.split("#", 1)[0]
    target = target.split("?", 1)[0]
    return target or None


def looks_like_repo_path(text: str) -> bool:
    if any(ch.isspace() for ch in text):
        return False
    if text.startswith(("http://", "https://", "mailto:", "#", "{{", "<")):
        return False
    if "<" in text or ">" in text:
        return False
    if any(symbol in text for symbol in ("*", "|", "--", "=", "[", "]", "{", "}")):
        return False
    if text.startswith("huaweicloud-") and "/" not in text:
        return False
    return text.startswith(PATH_PREFIXES)


def resolve_target(root: Path, source: Path, target: str) -> Path:
    candidate = Path(target)
    if candidate.is_absolute():
        return candidate
    if target.startswith(PATH_PREFIXES):
        return root / candidate
    return source.parent / candidate


def target_exists(root: Path, source: Path, target: str) -> bool:
    path = resolve_target(root, source, target)
    if any(part in ("*", "...") for part in path.parts):
        return True
    return path.exists()


def check_file(root: Path, path: Path) -> list[Finding]:
    findings: list[Finding] = []
    for line_no, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        for match in LINK_RE.finditer(line):
            target = normalize_target(match.group(1))
            if target and not target_exists(root, path, target):
                findings.append(Finding(path, line_no, target, "missing markdown link target"))
        for match in BACKTICK_RE.finditer(line):
            target = normalize_target(match.group(1).strip())
            if target and looks_like_repo_path(target) and not target_exists(root, path, target):
                findings.append(Finding(path, line_no, target, "missing backtick path target"))
    return findings


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args()
    root = args.root.resolve()

    findings: list[Finding] = []
    for path in iter_markdown_files(root):
        findings.extend(check_file(root, path))

    if findings:
        for finding in findings:
            rel = finding.file.relative_to(root).as_posix()
            print(f"{rel}:{finding.line}: {finding.reason}: {finding.target}", file=sys.stderr)
        print(f"ERROR: {len(findings)} broken Markdown path reference(s)", file=sys.stderr)
        return 1

    print("OK: Markdown local links and repository path references validated")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
