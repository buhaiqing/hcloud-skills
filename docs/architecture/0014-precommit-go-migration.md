---
title: ADR-0014 — Pre-commit gate migrated into the Go binary
status: Accepted (implemented 2026-08-01)
date: 2026-08-01
decision: Replace `scripts/pre_commit_check.sh` with a `hwcloud-skillcheck check --pre-commit` Go subcommand; the git hook execs the binary directly. Delete the shell script.
commits: `5832c04`..`0d8b8c1` on `feature/phase5-precommit-go`
---

## Context

`scripts/pre_commit_check.sh` ("single shot gun") is the only substantive shell
script in the repo. It wraps 13 gates that all call the `hwcloud-skillcheck` Go
binary or `go` toolchain (gofmt, go vet, validate, audit-results, aggregate
trace, learning gen, l4 handle smoke, golden run, check lanes, ab compare,
advanced-coverage, drift guard, go test). The script itself contains zero
shell logic beyond `run_gate()` + arg parsing — it is a thin orchestration layer
over the Go binary.

Three problems make migration urgent, not cosmetic:

1. **Latent trigger bug.** `.git/hooks/pre-commit` only runs the gate when a
   `scripts/*.py` file is touched (holdover from when `scripts/` had Python).
   `scripts/` is now all-Go, so the local pre-commit gate **silently no-ops** on
   `hwcloud-skillcheck/**/*.go` changes — exactly when it should fire.
2. **Local ≠ CI.** CI does NOT call `pre_commit_check.sh`; it inlines
   `gofmt/vet/test` + a subset of `hwcloud-skillcheck check ...`. The 13-gate
   contract therefore diverges between local hook and CI — a coverage gap.
3. **User preference + AGENTS.md TE-6.** "工具尽可能用 Go"; the binary should be
   the single source of truth. A bash wrapper that only calls the binary is
   pure ceremony.

## Decision

- Add a Go subcommand `hwcloud-skillcheck check --pre-commit`
  (`cmd/check.go`, `precommit` action). It re-implements all 13 gates as Go
  functions returning pass/fail, preserving the `--skip-tests` flag (git hook
  passes it; CI does not).
- Rewrite `.git/hooks/pre-commit` to **exec `hwcloud-skillcheck check
  --pre-commit --skip-tests`** directly. Fix the trigger condition to fire when
  `hwcloud-skillcheck/**/*.go` OR `hwcloud-skillcheck/testdata/*.py` OR
  `scripts/*.go` is touched (mirrors the b-class spec's hook-trigger update).
- **Delete `scripts/pre_commit_check.sh`.** No thin-wrapper retention — it
  would keep a parallel code path alive and re-introduce drift.
- Update `.github/workflows/validate-skills.yml` to call
  `hwcloud-skillcheck check --pre-commit` (no `--skip-tests`) so local + CI share
  one gate definition.

## Consequences

**Positive**
- Single gate definition (Go), exercised identically locally and in CI.
- Kills the stale-Python trigger bug.
- Satisfies "tools in Go" + TE-6 (binary is single source of truth).
- Each gate becomes unit-testable in Go (no shell parsing).

**Negative / costs**
- The git hook remains bash (git requires it) but is now a 3-line exec shim.
- `check --pre-commit` must rebuild the binary or reuse the running binary;
  to avoid stale-binary drift, the command builds `./bin/hwcloud-skillcheck`
  first (mirrors the shell's `SKILLCHECK_SKIP_BUILD` escape hatch).

## Alternatives considered

- **Thin wrapper (shell calls binary).** Rejected: keeps a parallel code path,
  violates "tools in Go", and would still need the trigger fix. No upside over
  deletion.
- **Keep shell, only fix trigger + CI.** Rejected: leaves the 13-gate logic in
  bash, duplicating what the Go binary already does, and contradicts the
  user's stated preference.
- **One combined `check` with no `--pre-commit` subcommand.** Rejected: mixes
  one-off `check audit-results` style commands with the full gate; the
  `--pre-commit` flag keeps the full-suite contract explicit and testable.
