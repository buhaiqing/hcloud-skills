#!/usr/bin/env python3
"""
Rule Compliance Probe (RSI self-assessment hook).

Static audit of a rule-bearing markdown file (default: repo AGENTS.md). Splits
the document into rule blocks (### headings or - **Rxx**: list items), then
classifies each as:

  verifiable  — contains at least one mechanical anchor (shell command,
                file path glob, regex, numeric threshold)
  manual      — declarative/process rule requiring human judgment
                (contains SHOULD/MUST-style verb but lacks machine anchor)
  vague       — lacks both anchor and executable verb

Optional --exec-check runs the embedded ```bash``` block through a read-only
command allowlist; unsafe commands are skipped and tagged `skipped_unsafe`.

Stdlib only. Designed to run in worktree/CI without network.

Exit codes:
  0  ok (report printed)
  1  bad arguments / unreadable rules file
  2  --exec-check ran a command that exited non-zero
"""
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from collections import Counter
from pathlib import Path
from typing import Any

# Mechanical anchors — any of these bumps a rule into `verifiable`.
#
# IMPORTANT (FIX-B1):
#   - re.ASCII prevents Chinese / CJK characters from masquerading as word
#     characters via Python's default Unicode \w semantics. Without this flag,
#     a Chinese term like "任何涉及代码/" would match the path branch because
#     `[\w./-]*` greedily absorbs the CJK characters preceding the slash.
#   - Each alternation branch is wrapped in its own non-capturing group with
#     a boundary assertion on both sides, so the boundary check is enforced
#     per-branch (not just on the first branch).
#   - BACKTICK_PATH anchors the match entirely inside a backtick span
#     (opening ` ... closing `) AND the span content must look like a pure
#     path (allowed chars only) — text like `spec/plan`, `proxy/heuristic`,
#     or `block` is treated as prose/multi-token reference and must NOT
#     trigger a path anchor.
_PATH_EXT = r"(?:py|go|sh|md|json|ya?ml|toml|tf|hcl|txt)"
_PATH_TOKEN = r"[A-Za-z0-9_][A-Za-z0-9_./-]*"
_BOUNDARY = r"(?:^|[\s`(])"
PATH_ANCHOR = re.compile(
    r"(?:" + _BOUNDARY + r"(?:\.{1,2}/|/)?" + _PATH_TOKEN + r"\." + _PATH_EXT + r"(?:\b|$))",
    re.ASCII,
)
# Match a single backticked span whose content is a "pure path" (only
# [A-Za-z0-9_./-] + leading dot). The lookaround ensures the closing backtick
# is the first backtick after the opening one — no embedded whitespace.
_BACKTICK_PURE_PATH = re.compile(
    # ordinary path: optional ./ or ../ prefix, then an alphanumeric segment
    r"`(?!\s)(?:\.{0,2}/)?[A-Za-z0-9_][A-Za-z0-9_./-]*`"
    # dotfile (`.gitignore`, `.pre-commit-config.yaml`): the first character is
    # a dot, which the branch above can never match — without this alternative
    # the validator's dotfile arm was unreachable dead code.
    r"|`\.[A-Za-z0-9_][A-Za-z0-9_.-]*`",
    re.ASCII,
)
# Pre-compile a path-content validator: a pure-path span must be one of:
#   (a) ends with a known extension (e.g. `bin/foo.py`)
#   (b) ends with a trailing slash (directory path like `bin/`)
#   (c) starts with `.` and is a dotfile (e.g. `.gitignore`)
# Anything else (e.g. `bin`, `block`, `spec/plan`, `proxy/heuristic`) is
# NOT a verifiable path.
_BACKTICK_PATH_VALIDATOR = re.compile(
    r"^(?:\.{0,2}/)?[A-Za-z0-9_][A-Za-z0-9_./-]*\.(" + _PATH_EXT.strip("(?:)") + r")$"
    r"|^(?:\.{0,2}/)?[A-Za-z0-9_][A-Za-z0-9_./-]*/$"  # dir path ending in /
    r"|^\.{1,2}[A-Za-z0-9_][A-Za-z0-9_./-]*$",          # dotfile like .gitignore
    re.ASCII,
)
SHELL_CMD = re.compile(r"```(?:bash|sh|shell)\n(.*?)```", re.DOTALL)
# Numeric / threshold anchors. ASCII comparison operators are only half the
# vocabulary: CJK rules lean on ≥/≤ and on quantifiers ("最多 3 轮", "达到
# 80%"), and missing those made the probe under-report verifiable rules against
# the hand-labelled gold set. CJK alternatives are literal text (not \w
# classes), so re.ASCII cannot affect them.
NUMERIC_THRESHOLD = re.compile(
    r"[<>]=?\s*\d+(?:\.\d+)?"
    r"|=\s*1\.0+"
    r"|=\s*0(?:\.5)?"
    r"|[≥≤]\s*\d+(?:\.\d+)?"
    r"|\d+(?:\.\d+)?\s*%"
    r"|(?:至少|最多|达到|超过|不低于|不高于|连续)\s*\d+",
    re.ASCII,
)
REGEX_PATTERN = re.compile(r"`([^`]*\\[a-z][^`]*?)`")  # backticked regex-ish
BACKTICK_FRAGMENT = re.compile(r"`([^`\n]+)`")


def _looks_like_command_fragment(text: str) -> bool:
    """A backticked fragment is command-shaped if it (a) contains at least one
    space, (b) starts with a leading-word token (no leading ./ or / and no
    embedded Chinese punctuation that usually marks prose), and (c) does not
    start with a regex metacharacter. Catches things like
        `ruff check --fix && ruff check`
        `go test ./...`
        `kustomize build`
    while ignoring plain identifier paths such as `bin/` or `.gitignore`.
    """
    for m in BACKTICK_FRAGMENT.finditer(text):
        frag = m.group(1).strip()
        if " " not in frag:
            continue
        if frag[0] in ".<>[]{}\\|^$":
            continue  # regex-ish or path
        # A command named inside a prohibition ("不假设 `go get` 能跑") is not an
        # instruction to run it; counting it made non-mechanical rules look
        # verifiable.
        if _negated_before(text, m.start()):
            continue
        # A space alone is not command-shaped: `module cache` and `spec plan`
        # are prose. The head token must be a known command, which also keeps
        # flag-less forms like `go run` working.
        head = frag.split()[0]
        if head in COMMAND_HEADS:
            return True
    return False

# Executable verbs (manual-class); explicit imperative or modal SHOULD/MUST.
# English verbs use \b word boundaries; CJK verbs (必须/应该/禁止) sit in a
# continuous character stream with no word boundaries, so wrapping them in \b
# produces a dead pattern that never matches. CJK verbs are emitted as a bare
# alternation and rely on the surrounding regex (anchor / non-CJK boundary) to
# avoid spurious matches inside unrelated Chinese text.
EXEC_VERB = re.compile(
    r"(?:\b(?:MUST|SHOULD|MUST NOT|SHOULD NOT|NEVER|ALWAYS|REQUIRED|PROHIBITED)\b"
    # 必须/应该/禁止 are modal; 先确认/审计/逐行 are process imperatives that
    # still need a human, so they classify manual rather than vague.
    r"|(?:必须|应该|禁止|先确认|先审计|审计|逐行|逐条))",
    re.IGNORECASE,
)
# Section reference (process anchor): "follow the rule in section ...", "see X.Y",
# numbered references like "rule 4" or "rule (P0)" — these still require a human
# to follow the pointer, so they classify as manual rather than vague.
SECTION_REF = re.compile(
    r"(?:section|规则|条款|rule)\s*[`\"']?[\w\s.\-()]+[`\"']?"
    r"|(?:见|参见|参考|see)\s+[`\"']?[A-Za-z0-9_.\\-]+"
    r"|[\(\[][A-Z]?\d+[a-z]?[\)\]]",  # e.g. (P0), [R3]
)

# White-listed read-only commands for --exec-check.
# `git` is split out: only read-only subcommands are allowlisted. Anything else
# (reset, clean, checkout, commit, push, ...) is rejected with skipped_unsafe.
EXEC_ALLOWLIST = {
    "echo", "printf", "wc", "grep", "cat", "head", "tail",
    "ls", "stat", "test", "true", "false",
}
EXEC_GIT_READONLY = {"status", "diff", "log", "show"}
EXEC_DENY_SUBSTR = ("|", ">", "<", "&&", "||", "rm ", "mv ", "sed -i", "tee ", "curl ", "wget ", "chmod ", "chown ")

CLASS_ORDER = ("verifiable", "manual", "vague")

# Command heads recognised in backticked fragments. An explicit list beats
# shape heuristics: "module cache" and "spec plan" have the same shape as
# "go run", and the gold set showed shape matching produced false positives on
# both sides.
COMMAND_HEADS = frozenset({
    "go", "git", "ruff", "pytest", "python", "python3", "pip", "npm", "npx",
    "pnpm", "yarn", "make", "cargo", "docker", "kubectl", "helm", "hcloud",
    "hwcloud-skillcheck", "gh", "jq", "yq", "tar", "ssh", "rsync", "curl",
    "wc", "grep", "rg", "sed", "awk", "find", "ls", "cat", "echo", "mkdir",
    "cp", "mv", "rm", "chmod", "systemctl", "journalctl", "psql", "mysql",
})

# Prohibitions and pointer prefixes: a nearby negation or "see ..." turns the
# following token into prose rather than an instruction or an anchor.
_NEGATION_BEFORE = re.compile(
    r"(?:不|勿|禁止|无法|不可|不能|避免|never|avoid|do not)\s*[\w\s\"'`]*$",
    re.IGNORECASE | re.ASCII,
)
_POINTER_BEFORE = re.compile(
    r"(?:见|参见|参考|详见|see)\s*[\w\s\"'`]*$",
    re.IGNORECASE | re.ASCII,
)


def _negated_before(text: str, start: int) -> bool:
    """True when a prohibition immediately precedes the token at `start`."""
    return bool(_NEGATION_BEFORE.search(text[max(0, start - 24):start]))


def _pointer_before(text: str, start: int) -> bool:
    """True when a "see ..." pointer immediately precedes the token at `start`."""
    return bool(_POINTER_BEFORE.search(text[max(0, start - 24):start]))


def _has_backticked_path(text: str) -> bool:
    """True iff at least one `` `...` `` span contains a *pure* path token.

    A pure-path span must (a) contain only [A-Za-z0-9_./-], (b) either end
    with a known file extension, end with a trailing slash, or start with a
    dot (dotfile). This rejects multi-token references like `spec/plan`,
    `proxy/heuristic`, and bare identifiers like `block` — those are
    process anchors (manual), not machine-checkable paths.
    """
    for m in _BACKTICK_PURE_PATH.finditer(text):
        span = m.group()[1:-1]  # strip backticks
        # A path introduced by "见 ... / see ..." is a documentation pointer,
        # not a requirement to check the file.
        if _pointer_before(text, m.start()):
            continue
        if _BACKTICK_PATH_VALIDATOR.match(span):
            return True
    return False


def _has_declared_path(text: str) -> bool:
    """PATH_ANCHOR match that is not a documentation pointer.

    `见 references/cadl-spec.md` names where to read, not a file requirement,
    so it must not make a rule mechanically checkable.
    """
    for m in PATH_ANCHOR.finditer(text):
        if not _pointer_before(text, m.start()):
            return True
    return False


def _detect_anchors(text: str) -> dict[str, Any]:
    """Return anchor inventory + extracted shell block (if any)."""
    anchors: dict[str, Any] = {
        "has_command": False,
        "has_path": False,
        "has_regex": False,
        "has_threshold": False,
        "verification_cmd": "",
    }
    # The rule's own heading names its topic, not a check to run: `go test` in
    # a title ("CA-6 go test / CI 里 os.Args[0] 不可信") is context, so anchors
    # are detected on the body only.
    body = re.sub(r"^#{1,4}\s+.*\n?", "", text, count=1)
    bash_blocks = SHELL_CMD.findall(text)
    if bash_blocks:
        anchors["has_command"] = True
        anchors["verification_cmd"] = bash_blocks[0].strip().rstrip("\n")
    if _has_backticked_path(body) or _has_declared_path(body):
        anchors["has_path"] = True
    if REGEX_PATTERN.search(body):
        anchors["has_regex"] = True
    if NUMERIC_THRESHOLD.search(body):
        anchors["has_threshold"] = True
    # Multi-token backticked fragments are not just paths/regex — they often
    # name a CLI surface (e.g. `ruff check`, `go test ./...`) which makes
    # the rule mechanically checkable even without a full fenced block.
    if _looks_like_command_fragment(body):
        anchors["has_command"] = True
    return anchors


def _has_example(text: str) -> bool:
    if "\u2705" in text or "\u274c" in text:
        return True
    # fenced code block (any language) without a language tag also counts.
    return bool(re.search(r"```\w*\n", text))


def classify(rule: dict[str, Any]) -> str:
    a = rule["anchors"]
    text = rule["text"]
    if a["has_command"] or a["has_path"] or a["has_regex"] or a["has_threshold"]:
        return "verifiable"
    if EXEC_VERB.search(text):
        return "manual"
    if SECTION_REF.search(text):
        return "manual"
    return "vague"


def split_rules(md_text: str) -> list[dict[str, Any]]:
    """
    Split markdown into rule blocks. Two collectors:
      (a) ### headings
      (b) top-level list items beginning with `- **R..**:` or `- **<slug>**:`
    A rule body is the heading line (or list item line) plus subsequent
    text until the next same/higher-level boundary.
    """
    rules: list[dict[str, Any]] = []
    lines = md_text.splitlines()
    i = 0
    n = len(lines)
    while i < n:
        line = lines[i]
        # ### heading → block until next ### or ##
        m_h = re.match(r"^(#{1,4})\s+(.+)$", line)
        if m_h and m_h.group(1).startswith("###"):
            start = i
            body = [line]
            i += 1
            # Headings inside fenced code blocks are not boundaries: a bash
            # comment like `# 返回 JSON: {...}` used to truncate the rule body.
            in_fence = False
            while i < n:
                candidate = lines[i]
                if candidate.lstrip().startswith("```"):
                    in_fence = not in_fence
                elif not in_fence and re.match(r"^#{1,4}\s+", candidate):
                    break
                body.append(candidate)
                i += 1
            rules.append({
                "id": f"H{i}",
                "title": m_h.group(2).strip(),
                "text": "\n".join(body).strip(),
            })
            continue
        # list item like  - **R1**: foo
        m_l = re.match(r"^\s*-\s+\*\*(.+?)\*\*\s*:?\s*(.*)$", line)
        if m_l and not line.lstrip().startswith("  "):
            indent = len(line) - len(line.lstrip())
            start = i
            body = [line]
            i += 1
            # collect indented continuations (same or deeper indent)
            while i < n:
                nxt = lines[i]
                if not nxt.strip():
                    body.append(nxt)
                    i += 1
                    continue
                nxt_indent = len(nxt) - len(nxt.lstrip())
                if nxt_indent > indent:
                    body.append(nxt)
                    i += 1
                else:
                    break
            rules.append({
                "id": f"L{start + 1}",
                "title": m_l.group(1).strip(),
                "text": "\n".join(body).strip(),
            })
            continue
        i += 1
    return rules


def analyze(rules_path: Path) -> dict[str, Any]:
    text = rules_path.read_text(encoding="utf-8")
    rules = split_rules(text)
    out: list[dict[str, Any]] = []
    counts: Counter[str] = Counter()
    for r in rules:
        anchors = _detect_anchors(r["text"])
        enriched = {
            **r,
            "anchors": anchors,
            "has_example": _has_example(r["text"]),
        }
        enriched["machine_checkable"] = classify(enriched)
        counts[enriched["machine_checkable"]] += 1
        out.append(enriched)
    total = len(out)
    if total == 0:
        rate = 0.0
    else:
        rate = round(counts["verifiable"] / total, 3)
    return {
        "source": str(rules_path),
        "total_rules": total,
        "verifiable": counts.get("verifiable", 0),
        "manual": counts.get("manual", 0),
        "vague": counts.get("vague", 0),
        "verifiable_rate": rate,
        "rules": out,
    }


def run_safe(cmd: str, cwd: Path) -> dict[str, Any]:
    """Whitelist-only execution. First token must be an allowlisted command;
    for `git` the second token must be a read-only subcommand.
    Any pipe/redirect/chained-command token forces `skipped_unsafe`."""
    stripped = cmd.strip().rstrip("\n")
    first = stripped.split(maxsplit=1)
    if not first or first[0] not in EXEC_ALLOWLIST and first[0] != "git":
        return {"ran": False, "skipped_unsafe": True, "reason": f"command '{first[0] if first else ''}' not allowlisted"}
    if first[0] == "git":
        # require second token to be a read-only git subcommand
        rest = first[1].split() if len(first) > 1 else []
        if not rest or rest[0] not in EXEC_GIT_READONLY:
            sub = rest[0] if rest else ""
            return {"ran": False, "skipped_unsafe": True, "reason": f"git subcommand '{sub}' not in read-only allowlist"}
    if any(tok in stripped for tok in EXEC_DENY_SUBSTR):
        return {"ran": False, "skipped_unsafe": True, "reason": "forbidden operator (pipe/redirect/chain/rm/curl/...)"}
    # List-exec (no shell): we already vetted first/second tokens above and
    # the EXEC_DENY_SUBSTR check rejects pipe/redirect/chain markers, so the
    # deny-substring guard is the only shell-feature tripwire we still need
    # before handing argv straight to subprocess.run.
    try:
        completed = subprocess.run(
            [first[0]] + (first[1].split() if len(first) > 1 else []),
            cwd=str(cwd),
            capture_output=True,
            text=True,
            timeout=10,
            check=False,
        )
    except subprocess.TimeoutExpired:
        return {"ran": False, "skipped_unsafe": True, "reason": "timeout"}
    return {
        "ran": True,
        "returncode": completed.returncode,
        "stdout": completed.stdout[:400],
        "stderr": completed.stderr[:400],
    }


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description="Rule compliance probe (RSI).")
    ap.add_argument("--rules", type=Path, default=Path("AGENTS.md"),
                    help="Path to rule-bearing markdown (default: ./AGENTS.md)")
    ap.add_argument("--json", action="store_true", help="Emit JSON report only")
    ap.add_argument("--exec-check", action="store_true",
                    help="Allowlist-execute embedded ```bash blocks (off by default)")
    ap.add_argument("--cwd", type=Path, default=Path("."),
                    help="Working dir for executed commands (default: .)")
    args = ap.parse_args(argv)

    if not args.rules.is_file():
        print(f"ERROR: rules file not found: {args.rules}", file=sys.stderr)
        return 1

    try:
        report = analyze(args.rules)
    except (OSError, UnicodeDecodeError) as e:
        print(f"ERROR: cannot read {args.rules}: {e}", file=sys.stderr)
        return 1

    # Optional execution
    ran_anything = False
    failed_exec = False
    if args.exec_check:
        for r in report["rules"]:
            if r["machine_checkable"] != "verifiable":
                continue
            cmd = r["anchors"].get("verification_cmd", "")
            if not cmd:
                continue
            result = run_safe(cmd, args.cwd)
            r["exec_result"] = result
            if result.get("ran"):
                ran_anything = True
                if result["returncode"] != 0:
                    failed_exec = True
                    r["exec_result"]["status"] = "exec_failed"

    if args.json:
        json.dump(report, sys.stdout, indent=2, ensure_ascii=False)
        sys.stdout.write("\n")
    else:
        out = sys.stderr if args.exec_check else sys.stdout
        summary = (
            f"source={report['source']} total={report['total_rules']} "
            f"verifiable={report['verifiable']} manual={report['manual']} "
            f"vague={report['vague']} verifiable_rate={report['verifiable_rate']}"
        )
        print(summary, file=out)
        for r in report["rules"]:
            extras = ""
            if "exec_result" in r:
                er = r["exec_result"]
                extras = f" exec={er.get('returncode', 'skipped')}"
            print(f"  [{r['machine_checkable']:>10}] {r['id']:<6} {r['title'][:60]}{extras}", file=out)
        if not ran_anything and args.exec_check:
            print("(no verifiable ```bash blocks executed)", file=out)

    return 2 if failed_exec else 0


if __name__ == "__main__":
    sys.exit(main())
