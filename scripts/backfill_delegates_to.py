"""Backfill SKILL.md frontmatter `delegates_to:` field for every skill.

For each skill, walks SKILL.md body + references/integration.md, collects the unique
set of huaweicloud-X-ops skills it references, and writes (or updates) the
`delegates_to:` YAML field in the frontmatter. Idempotent — safe to re-run.

Run: python3 scripts/backfill_delegates_to.py [--apply]
"""
from __future__ import annotations

import argparse
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

SKILL_RE = re.compile(r"huaweicloud-([a-z]+)-ops")
FRONTMATTER_RE = re.compile(r"^---\n(.*?)\n---\n", re.DOTALL)


def collect_delegations(skill_md: Path, integration_md: Path) -> set[str]:
    """Return sorted set of huaweicloud-X-ops short names referenced in this skill's docs."""
    referenced: set[str] = set()
    primary_short = skill_md.parent.name.replace("huaweicloud-", "").replace("-ops", "")

    for path in (skill_md, integration_md):
        if not path.exists():
            continue
        text = path.read_text(encoding="utf-8")
        # Skip the YAML frontmatter block — only parse body content
        body = FRONTMATTER_RE.sub("", text, count=1)
        for short in SKILL_RE.findall(body):
            if short != primary_short:
                referenced.add(short)

    return referenced


def update_frontmatter(skill_md: Path, delegations: set[str], apply: bool) -> tuple[bool, str]:
    """Insert or update the `delegates_to:` field. Returns (changed, status_msg)."""
    text = skill_md.read_text(encoding="utf-8")
    m = FRONTMATTER_RE.search(text)
    if not m:
        return False, "no frontmatter"

    fm_block = m.group(1)
    fm_start, fm_end = m.span(1)

    if not delegations:
        # Ensure the key exists as empty list (or remove if present?)
        # Convention: always include the key, even when empty, to make generator output deterministic.
        new_listing = "delegates_to: []\n"
    else:
        new_listing = "delegates_to:\n" + "".join(f"  - huaweicloud-{s}-ops\n" for s in sorted(delegations))

    # Match existing delegates_to: block (multiline YAML list OR inline [] form)
    existing_re = re.compile(
        r"^delegates_to:\s*(?:\n(?:\s*-\s*huaweicloud-[a-z]+-ops\s*\n)*|\[[^\]]*\])\s*\n?",
        re.MULTILINE,
    )
    m_existing = existing_re.search(fm_block)
    if m_existing:
        # Replace the existing block in place
        new_fm = fm_block[: m_existing.start()] + new_listing + fm_block[m_existing.end() :]
    elif delegations:
        # No existing block AND we have delegations → insert before the first non-mapping key
        # Convention: insert right after `name:` for visibility (top of metadata block)
        name_match = re.search(r"^name:\s*.+\n", fm_block, re.MULTILINE)
        if name_match:
            new_fm = (
                fm_block[: name_match.end()]
                + new_listing
                + fm_block[name_match.end() :]
            )
        else:
            new_fm = new_listing + fm_block
    else:
        # No delegations AND no existing field → nothing to do
        return False, "no change"

    if new_fm == fm_block:
        return False, "no change"

    new_text = text[:fm_start] + new_fm + text[fm_end:]
    if apply:
        skill_md.write_text(new_text, encoding="utf-8")
    return True, f"updated → {len(delegations)} skills"


def main() -> int:
    ap = argparse.ArgumentParser(description="Backfill SKILL.md `delegates_to:` frontmatter.")
    ap.add_argument("--apply", action="store_true", help="Write changes (default: dry-run)")
    args = ap.parse_args()

    changed_count = 0
    for skill_dir in sorted(ROOT.glob("huaweicloud-*-ops")):
        skill_md = skill_dir / "SKILL.md"
        integration_md = skill_dir / "references" / "integration.md"
        delegations = collect_delegations(skill_md, integration_md)
        if not delegations and not (skill_md.read_text(encoding="utf-8").startswith("---\ndelegates_to:")):
            # No delegations AND no existing field → skip silently
            continue
        changed, msg = update_frontmatter(skill_md, delegations, args.apply)
        prefix = "WOULD" if not args.apply else "DID"
        if changed:
            changed_count += 1
        print(f"  {prefix} {skill_dir.name}: {msg}")

    print(f"\n{'Applied' if args.apply else 'Would apply'} {changed_count} skill(s)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())