---
phase: 02-code-review-command
reviewed: 2026-07-18T22:00:00Z
resolved: 2026-07-28
depth: deep
files_reviewed: 20
files_reviewed_list:
  - hwcloud-skillcheck/main.go
  - hwcloud-skillcheck/cmd/root.go
  - hwcloud-skillcheck/cmd/validate.go
  - hwcloud-skillcheck/cmd/validate_eval.go
  - hwcloud-skillcheck/cmd/validate_repo.go
  - hwcloud-skillcheck/cmd/check.go
  - hwcloud-skillcheck/cmd/scan.go
  - hwcloud-skillcheck/cmd/aggregate.go
  - hwcloud-skillcheck/cmd/lint.go
  - hwcloud-skillcheck/internal/schema/schema.go
  - hwcloud-skillcheck/internal/security/security.go
  - hwcloud-skillcheck/internal/yaml/yaml.go
  - hwcloud-skillcheck/internal/coverage/coverage.go
  - hwcloud-skillcheck/internal/embed/embed.go
  - hwcloud-skillcheck/cmd/cmd_test.go
  - hwcloud-skillcheck/cmd/aggregate_test.go
  - hwcloud-skillcheck/cmd/check_test.go
  - hwcloud-skillcheck/cmd/scan_test.go
  - hwcloud-skillcheck/cmd/validate_repo_test.go
  - hwcloud-skillcheck/internal/schema/schema_test.go
  - hwcloud-skillcheck/internal/security/security_test.go
  - hwcloud-skillcheck/internal/yaml/yaml_test.go
  - hwcloud-skillcheck/internal/coverage/coverage_test.go
  - hwcloud-skillcheck/internal/embed/embed_test.go
  - hwcloud-skillcheck/testdata/equivalence_test.py
  - docs/superpowers/specs/hwcloud-skillcheck-cli.md
findings:
  critical: 2
  warning: 6
  info: 5
  total: 13
status: resolved
---

# Phase 2: Code Review Report

**Reviewed:** 2026-07-18T22:00:00Z
**Resolved:** 2026-07-28
**Depth:** deep
**Files Reviewed:** 20 source + 10 test + 2 docs
**Status:** resolved

## Resolution checklist (2026-07-28)

| ID | Severity | Status | Notes |
|----|----------|--------|-------|
| CR-01 | CRITICAL | fixed | `runeAt` → `utf8.DecodeRuneInString` |
| CR-02 | CRITICAL | fixed | lint `--fix`: `-l` then conditional `-w` |
| WR-01 | WARNING | fixed | removed `marshalJSON` |
| WR-02 | WARNING | fixed | `validateEvalQueries` calls shared `detectEvalFormat` |
| WR-03 | WARNING | fixed | `discoverSkillDirs` returns `ReadDir` error; regression test |
| WR-04 | WARNING | fixed | `TestScanJSON` table-driven |
| WR-05 | WARNING | fixed | `ScanContent` errors checked in `scan.go` |
| WR-06 | WARNING | fixed | removed `splitComma`; use `splitOnComma` |
| IN-01…05 | INFO | fixed / deferred | see migration review; not blocking |

## Summary

The hwcloud-skillcheck Go CLI is a well-structured migration of ~5000 lines of Python validation scripts into a single Go binary. The codebase follows Go conventions well overall, has good test coverage for a v1, and demonstrates clear understanding of the domain. Two CRITICAL bugs and six WARNING items from this review are closed as of 2026-07-28.

## Critical Issues

### CR-01: runeAt() byte indexing — FIXED

Use `utf8.DecodeRuneInString(s[i:])` instead of `rune(s[i])`.

### CR-02: lint --fix mode — FIXED

Always list with `gofmt -l`, then conditionally `gofmt -w`.

## Warnings

### WR-01: marshalJSON — FIXED

Removed; call sites use `json.Marshal` directly.

### WR-02: detectEvalFormat duplicated — FIXED

`validateEvalQueries` now calls shared `detectEvalFormat`. Parse failures remain hard errors (`parse instance:`); shape mismatches stay soft `[]string` messages. `validateEvalQueriesFile` already used `detectEvalFormat`.

### WR-03: discoverSkillDirs swallows errors — FIXED

Returns `os.ReadDir` error; `runCheckExampleConfig` wraps it. Regression: `TestDiscoverSkillDirsMissingRoot`.

### WR-04: ScanJSON missing tests — FIXED

Added `TestScanJSON` (clean / nested leak / masked skip / invalid JSON / UseNumber).

### WR-05: ScanContent error ignored — FIXED

`scan.go` checks and surfaces errors.

### WR-06: splitComma wrapper — FIXED

Removed; callers use `splitOnComma` only.

## Info

### IN-01: itoa → strconv.Itoa — FIXED
### IN-02: hasKey helper — deferred (style)
### IN-03: validate_eval.go naming — deferred (not required to close WARNINGs)
### IN-04: decodeJSON extraction — FIXED
### IN-05: numOf vs intOf — deferred (test helper)

---

_Reviewed: 2026-07-18T22:00:00Z_
_Resolved: 2026-07-28_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
