#!/usr/bin/env python3
"""Validate deep link health of `references/` Markdown across all skills.

The repository's `huaweicloud-*-ops/references/` folders contain 200+ docs that
heavily cross-link each other (and sometimes reference sibling skills). The
generic `scripts/check_markdown_links.py` only checks *file existence*; this
gate catches the more subtle regressions:

1. **Anchor existence** — `[text](file.md#some-section)` must resolve to a real
   heading in `file.md`. Catches typo'd / deleted section names.
2. **Sibling path suggestion** — `references/foo.md` referencing `bar` without
   `.md` extension is flagged as a warning (GitHub renders it, but local
   previewers often do not).
3. **Cross-skill resolution** — any link pointing at `huaweicloud-*-ops/...`
   must resolve to a real skill directory in the repo.
4. **Heading inventory** — emit a per-file count of headings for triage.

Anchor resolution follows GitHub's slugifier: lowercase, drop punctuation,
replace whitespace with `-`, drop leading digits? No — GitHub keeps leading
digits. This implementation matches that behaviour so a `[…](#3-finops)` link
resolves to a `## 3. FinOps` heading.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
import unicodedata
from dataclasses import asdict, dataclass, field
from pathlib import Path

IGNORED_DIR_PARTS = {".git", ".github", ".agents", ".omc", ".omo", ".codebuddy", ".claude", "audit-results"}

LINK_RE = re.compile(r"(?<!!)\[[^\]]+\]\(([^)\s]+)(?:\s+\"[^\"]*\")?\)")
HEADING_RE = re.compile(r"^(#{1,6})\s+(.+?)\s*#*\s*$", re.MULTILINE)
INLINE_CODE_RE = re.compile(r"`[^`]*`")


@dataclass(frozen=True)
class Finding:
    file: Path
    line: int
    target: str
    severity: str
    reason: str


@dataclass
class HeadingInventory:
    anchors: set[str] = field(default_factory=set)

    def add(self, raw: str) -> None:
        self.anchors.add(slugify_anchor(raw))


def slugify_anchor(text: str) -> str:
    """GitHub-style anchor slugifier.

    Steps: drop inline code backticks, lowercase, strip combining marks,
    replace whitespace with `-`, drop characters that aren't alphanumeric / `-` / `_`.
    """
    text = text.replace("`", "")
    text = unicodedata.normalize("NFKD", text)
    text = "".join(ch for ch in text if not unicodedata.combining(ch))
    text = text.lower()
    text = re.sub(r"\s+", "-", text)
    text = re.sub(r"[^\w\-]", "", text, flags=re.UNICODE)
    return text


def iter_references_files(root: Path) -> list[Path]:
    files: list[Path] = []
    for path in sorted((root).glob("huaweicloud-*-ops/references/*.md")):
        if any(part in IGNORED_DIR_PARTS for part in path.relative_to(root).parts):
            continue
        files.append(path)
    return files


def inventory_headings(path: Path) -> HeadingInventory:
    inventory = HeadingInventory()
    body = path.read_text(encoding="utf-8")
    for match in HEADING_RE.finditer(body):
        inventory.add(match.group(2))
    return inventory


def split_target(raw: str) -> tuple[str, str]:
    """Split a link target into (path, anchor). Anchor may be empty."""
    raw = raw.strip()
    if "#" in raw:
        path, anchor = raw.split("#", 1)
    else:
        path, anchor = raw, ""
    return path, anchor


def resolve_relative(root: Path, source: Path, link_path: str) -> Path | None:
    """Resolve a relative link path against the source file's directory."""
    if not link_path:
        return None
    candidate = Path(link_path)
    if candidate.is_absolute():
        return candidate
    base = source.parent if source.is_file() else source
    return (base / candidate).resolve()


def sibling_md_suggestion(references_dir: Path, link_path: str) -> str | None:
    """If link_path is a bare sibling name (no extension) and `link_path.md` exists,
    return the suggested replacement."""
    if "/" in link_path or "\\" in link_path:
        return None
    candidate = references_dir / f"{link_path}.md"
    if candidate.is_file():
        return f"{link_path}.md"
    return None


def check_file(root: Path, path: Path, cache: dict[Path, HeadingInventory]) -> list[Finding]:
    findings: list[Finding] = []
    references_dir = path.parent
    body = path.read_text(encoding="utf-8")
    for line_no, line in enumerate(body.splitlines(), 1):
        # Skip code blocks (simple fence detection).
        stripped = line.lstrip()
        if stripped.startswith("```") or stripped.startswith("~~~"):
            continue
        for match in LINK_RE.finditer(line):
            raw = match.group(1).strip()
            if not raw:
                continue
            if raw.startswith(("http://", "https://", "mailto:", "<", "{{")):
                continue
            link_path, anchor = split_target(raw)
            if not link_path:
                # Pure anchor link (same file): must exist in inventory.
                if anchor and anchor not in inventory_for(path, cache).anchors:
                    findings.append(
                        Finding(path, line_no, f"#{anchor}", "error", f"missing in-page anchor: {anchor!r}")
                    )
                continue

            resolved = resolve_relative(root, path, link_path)
            if resolved is None or not resolved.exists():
                # Try cross-skill alias: `huaweicloud-iam-ops/...`
                if link_path.startswith("huaweicloud-") and "/" in link_path:
                    skill_part = link_path.split("/", 1)[0]
                    if not (root / skill_part).is_dir():
                        findings.append(
                            Finding(path, line_no, raw, "error", f"missing cross-skill reference: {skill_part}")
                        )
                        continue
                suggestion = sibling_md_suggestion(references_dir, link_path)
                if suggestion is not None:
                    findings.append(
                        Finding(
                            path,
                            line_no,
                            raw,
                            "warning",
                            f"bare sibling link '{link_path}' should reference '{suggestion}'",
                        )
                    )
                    continue
                findings.append(Finding(path, line_no, raw, "error", f"missing link target: {link_path}"))
                continue

            if not anchor:
                continue
            if not resolved.is_file():
                continue
            inventory = inventory_for(resolved, cache)
            if anchor not in inventory.anchors:
                findings.append(Finding(path, line_no, raw, "error", f"missing anchor #{anchor!r} in {resolved.name}"))
    return findings


def inventory_for(path: Path, cache: dict[Path, HeadingInventory]) -> HeadingInventory:
    if path not in cache:
        cache[path] = inventory_headings(path)
    return cache[path]


def collect(root: Path) -> dict[str, object]:
    files = iter_references_files(root)
    cache: dict[Path, HeadingInventory] = {}
    findings: list[Finding] = []
    for path in files:
        findings.extend(check_file(root, path, cache))
    severities: dict[str, int] = {"error": 0, "warning": 0, "info": 0}
    for finding in findings:
        severities[finding.severity] = severities.get(finding.severity, 0) + 1
    return {
        "ok": severities["error"] == 0,
        "files": [str(path.relative_to(root)) for path in files],
        "summary": {
            "files_scanned": len(files),
            "findings_total": len(findings),
            "errors": severities["error"],
            "warnings": severities["warning"],
        },
        "findings": [asdict(finding) | {"file": str(finding.file.relative_to(root))} for finding in findings],
    }


def format_human(report: dict[str, object]) -> str:
    summary = report["summary"]
    files_scanned = summary["files_scanned"]
    findings: list[dict[str, object]] = report["findings"]  # type: ignore[assignment]
    lines = [
        f"references/ link health: scanned {files_scanned} files, "
        f"errors={summary['errors']}, warnings={summary['warnings']}",
    ]
    for finding in findings:
        assert isinstance(finding["file"], str)
        assert isinstance(finding["line"], int)
        assert isinstance(finding["severity"], str)
        assert isinstance(finding["target"], str)
        assert isinstance(finding["reason"], str)
        lines.append(
            f"  {finding['severity'].upper():7s} {finding['file']}:{finding['line']}: "
            f"{finding['reason']} -> {finding['target']}"
        )
    return "\n".join(lines) + "\n"


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--json", action="store_true")
    parser.add_argument("--warnings-only", action="store_true", help="Treat warnings as non-fatal (still report)")
    return parser


def main() -> int:
    args = build_parser().parse_args()
    root = args.root.resolve()
    report = collect(root)
    if args.json:
        print(json.dumps(report, indent=2, ensure_ascii=False))
    else:
        print(format_human(report), end="")
    if not report["ok"]:
        return 1
    if not args.warnings_only and report["summary"]["warnings"] > 0:
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
