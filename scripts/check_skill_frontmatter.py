"""Lint SKILL.md frontmatter for structural drift.

Scans every `huaweicloud-*-ops/SKILL.md` and checks:
  1. YAML parses without error (catches dangling/orphan lines like the ECS 56-bug)
  2. `name:` field present and matches the directory name
  3. `description:` field present and non-empty
  4. `delegates_to:` field present (L4 cross-skill awareness)
  5. Indentation consistency in folded blocks (`>-` / `|` / `|+`)

Exit code: 0 = all clean, 1 = at least one violation found.
Use `--skill <name>` to lint one skill; default scans all.

Add to validate_local.py as Step("SKILL.md frontmatter lint", (python3, ...)).
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT_DEFAULT = Path(__file__).resolve().parents[1]

FRONTMATTER_RE = re.compile(r"^---\n(.*?)\n---\n", re.DOTALL)


def extract_frontmatter(text: str) -> tuple[str, int, int] | None:
    """Return (yaml_block, start_line, end_line) of the frontmatter, or None if missing."""
    lines = text.splitlines(keepends=True)
    if not lines or not lines[0].startswith("---"):
        return None
    # find closing fence
    start = 0
    for i in range(1, len(lines)):
        if lines[i].startswith("---"):
            block = "".join(lines[start + 1 : i])
            return block, start + 2, i + 1  # 1-based line numbers
    return None


def check_one(skill_md: Path) -> list[str]:
    """Return list of violation messages for one SKILL.md (empty = clean)."""
    violations: list[str] = []
    text = skill_md.read_text(encoding="utf-8")
    expected_name = skill_md.parent.name  # e.g., huaweicloud-rds-ops

    fm = extract_frontmatter(text)
    if fm is None:
        return [f"{skill_md}: missing YAML frontmatter (no `---` fence)"]
    yaml_block, start_line, end_line = fm

    # 1. YAML must parse
    try:
        import yaml  # type: ignore[import-untyped]
        data = yaml.safe_load(yaml_block)
    except ImportError:
        # Fallback: regex-only checks (still catches dangling-line and missing fields)
        data = None
    except Exception as e:  # yaml.YAMLError or scanner errors
        # Find the offending line in the file for actionable error message
        first_err_line = 0
        for tok in getattr(e, "problem_mark", []) and [getattr(e, "problem_mark", None)] or []:
            if tok:
                first_err_line = tok.line + 1
        msg = f"{skill_md}: YAML parse error: {type(e).__name__}"
        if first_err_line:
            abs_line = start_line + first_err_line - 1
            msg += f" (line {abs_line})"
        msg += f" — {str(e).splitlines()[0]}"
        violations.append(msg)
        return violations  # structural error short-circuits semantic checks

    if not isinstance(data, dict):
        violations.append(f"{skill_md}: frontmatter did not parse to a mapping (got {type(data).__name__})")
        return violations

    # 2. name: present and matches directory
    name = data.get("name")
    if not name:
        violations.append(f"{skill_md}: missing required `name:` field in frontmatter")
    elif name != expected_name:
        violations.append(f"{skill_md}: frontmatter `name: {name}` does not match directory `{expected_name}`")

    # 3. description: present and non-empty
    desc = data.get("description")
    if not desc or not str(desc).strip():
        violations.append(f"{skill_md}: missing or empty `description:` field")

    # 4. delegates_to: present (L4 cross-skill awareness — backfilled everywhere)
    if "delegates_to" not in data:
        violations.append(f"{skill_md}: missing `delegates_to:` field (run scripts/backfill_delegates_to.py)")

    # 5. Indentation consistency in folded/literal blocks (regex-based)
    # Walk lines; for any `>-` or `|` block, all continuation lines must have GREATER indent
    # than the key. If we find a line with the SAME indent as the block key, it likely
    # belongs to a previous block — dangling/orphan pattern (the ECS 56 bug).
    lines = yaml_block.splitlines()
    for i, line in enumerate(lines):
        stripped = line.lstrip()
        if not stripped or stripped.startswith("#"):
            continue
        m = re.match(r"^([\s-]*)([A-Za-z_][\w-]*):\s*(>[-]?|\|[+-]?)\s*$", line)
        if not m:
            continue
        key_indent = len(m.group(1))
        block_indent = len(line) - len(stripped)
        # Verify all subsequent lines (until a line with indent <= block_indent) have indent > block_indent
        for j in range(i + 1, len(lines)):
            cont = lines[j]
            cont_stripped = cont.lstrip()
            if not cont_stripped or cont_stripped.startswith("#"):
                continue
            cont_indent = len(cont) - len(cont_stripped)
            if cont_indent <= block_indent:
                break  # exited the block
            # Continuation lines must be strictly deeper than the KEY (not just the block opener)
            if cont_indent <= key_indent:
                violations.append(
                    f"{skill_md}: line {start_line + j}: continuation indent {cont_indent} "
                    f"not deeper than key indent {key_indent} (dangling-line risk)"
                )

    return violations


def cmd_check(args: argparse.Namespace) -> int:
    root: Path = args.root
    if args.skill:
        targets = [root / args.skill / "SKILL.md"]
    else:
        targets = sorted(root.glob("huaweicloud-*-ops/SKILL.md"))

    all_violations: list[str] = []
    scanned = 0
    for skill_md in targets:
        if not skill_md.exists():
            all_violations.append(f"MISSING: {skill_md}")
            continue
        scanned += 1
        all_violations.extend(check_one(skill_md))

    print(f"Scanned {scanned} SKILL.md file(s) in {root}")
    if all_violations:
        print(f"\nFAIL: {len(all_violations)} violation(s):", file=sys.stderr)
        for v in all_violations:
            print(f"  - {v}", file=sys.stderr)
        return 1
    print("OK: all SKILL.md frontmatter lint checks passed")
    return 0


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    p.add_argument("--root", type=Path, default=ROOT_DEFAULT, help="Repo root")
    p.add_argument("--skill", default=None, help="Lint only this skill (e.g., huaweicloud-ecs-ops)")
    p.set_defaults(func=cmd_check)
    return p


def main() -> int:
    args = build_parser().parse_args()
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())