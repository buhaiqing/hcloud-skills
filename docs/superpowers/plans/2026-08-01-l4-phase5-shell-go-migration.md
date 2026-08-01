# Plan: L4 Phase 5 — Pre-commit Gate Go Migration

**Date**: 2026-08-01
**Supersedes**: the Phase 5 sketch in `docs/superpowers/plans/2026-07-31-l4-maturity-upgrade.md` (lines 141–156), which under-scoped the gate inventory.
**ADR**: `docs/architecture/0014-precommit-go-migration.md` (wrapper-vs-replace → replace + delete)

## Goal

Move the pre-commit gate out of bash (`scripts/pre_commit_check.sh`) into the
`hwcloud-skillcheck` Go binary as `check --pre-commit`, so:
1. Local git hook and CI run the **same** gate definition.
2. The stale Python-trigger bug in the hook is fixed.
3. The repo's only substantive shell script is removed (tools-in-Go, TE-6).

## Current State (evidence)

`scripts/pre_commit_check.sh` (102 lines) runs **13 gates**:
1. `gofmt -l .` (non-empty → fail)
2. `go vet ./...`
3. `hwcloud-skillcheck validate --root .`
4. `hwcloud-skillcheck check audit-results --root .`
5. `hwcloud-skillcheck aggregate trace --require-traces --root .`
6. `hwcloud-skillcheck learning gen --root .`
7. `hwcloud-skillcheck l4 handle --fault smoke --risk low --root .`
8. `hwcloud-skillcheck golden run --root .` (soft: `|| true`)
9. `hwcloud-skillcheck check lanes --root .`
10. `hwcloud-skillcheck ab compare --root .` (soft: `|| true`)
11. `hwcloud-skillcheck check advanced-coverage --root .`
12. `hwcloud-skillcheck drift sync --apply --root . && drift check --root .`
13. `go test ./... -count=1` (skipped when `--skip-tests`)

**Bug**: `.git/hooks/pre-commit` triggers only on `scripts/*.py` (line ~17), so
Go changes skip the gate entirely.

**Divergence**: `.github/workflows/validate-skills.yml` inlines gofmt/vet/test
+ a subset of `check` commands; it never calls `pre_commit_check.sh`. So CI
covers fewer gates than the shell intends.

## Target Architecture

```
.git/hooks/pre-commit  (bash shim, ~6 lines)
   └─ exec: hwcloud-skillcheck check --pre-commit --skip-tests

hwcloud-skillcheck check --pre-commit [--skip-tests]   (Go, cmd/check.go)
   └─ runs the 13 gates above as Go functions; exit 1 on any hard failure.
```

- New action `precommit` added to `cmd/check.go` (alongside existing
  `audit-results`, `lanes`, `advanced-coverage`, …).
- Each gate = a `func(*config) gateResult` returning `{name, passed, soft}`.
  `check --pre-commit` iterates, prints `==> [gate] <label>`, exits 1 if any
  **hard** gate fails. Soft gates (`golden run`, `ab compare`) warn but don't
  fail.
- `--skip-tests` flag (bool) gates gate #13, mirroring the shell.
- Binary build: to avoid stale-binary drift, the command rebuilds
  `./bin/hwcloud-skillcheck` first unless `SKILLCHECK_SKIP_BUILD=1` (mirrors
  shell line 39). For the `go test` / `go vet` / `gofmt` gates, shell out to the
  `go` toolchain via `os/exec` (same as today).
- The `hwcloud-skillcheck …` gates are invoked by re-calling the binary's own
  command functions in-process (no subprocess) where feasible, falling back to
  `exec` for isolation if needed.

## Migration Steps (each a separate, independently-committable CR)

### Batch E1 — Go subcommand skeleton
- **Files**: `hwcloud-skillcheck/cmd/check.go` (new `precommit` action +
  `gateResult` type + gate registry), `hwcloud-skillcheck/cmd/check_precommit_test.go`.
- **Content**: implement all 13 gates as Go functions; `--skip-tests` flag;
  `SKILLCHECK_SKIP_BUILD` escape hatch; exit-code contract identical to shell.
- **Verify**: `go test ./cmd/...`; manual `./bin/hwcloud-skillcheck check --pre-commit` from a clean tree prints all 13 gates + "All pre-commit gates passed."
- **GCL**: >5 lines → Generator-Critic-Loop multi-sub-agent (per AGENTS.md §16.1).

### Batch E2 — Git hook rewrite + trigger fix
- **Files**: **create** `.githooks/pre-commit` (tracked template, currently
  MISSING — `scripts/install_hook.go` would `fail()` because it copies from
  `.githooks/pre-commit` which does not exist). Also rewrite the active
  `.git/hooks/pre-commit` (installed copy) to match.
- **Content**: tracked template is a 6-line exec shim:
  ```bash
  #!/usr/bin/env bash
  set -euo pipefail
  ROOT="$(git rev-parse --show-toplevel)"
  # fire only when Go/runtime surfaces change; markdown-only stays fast
  if git diff --cached --name-only | grep -qE '^(hwcloud-skillcheck/.*\.go|hwcloud-skillcheck/testdata/.*\.py|scripts/.*\.go)$'; then
    exec "$ROOT/bin/hwcloud-skillcheck" check --pre-commit --skip-tests
  fi
  ```
  (was: trigger on `scripts/*.py`, which is now empty → gate silently no-op'd.)
- **Verify**: `bash -n .githooks/pre-commit`; `go run scripts/install_hook.go
  --check` (or `--install`) succeeds; touch a `.go` file + `git commit` → gate
  fires; touch only a `.md` → gate skips.

### Batch E3 — CI convergence
- **Files**: `.github/workflows/validate-skills.yml`.
- **Content**: replace the inlined gofmt/vet/test + partial `check` calls with a
  single `./bin/hwcloud-skillcheck check --pre-commit` (no `--skip-tests`) step,
  so CI == local gate. Keep the `build-skillcheck.yml` Go-build job as-is
  (it provides the binary).
- **Verify**: CI run shows the unified gate; spot-check that no gate is dropped
  vs the 13-list.

### Batch E4 — Delete shell + docs
- **Files**: delete `scripts/pre_commit_check.sh`; update `AGENTS.md`
  (§ "A single shot gun…" → point to `check --pre-commit`),
  `docs/superpowers/plans/2026-07-31-l4-maturity-upgrade.md` (mark Phase 5 done),
  `docker/README.md` if it references the script, `docs/superpowers/specs/*`
  that mention the shell.
- **Verify**: `grep -rn pre_commit_check.sh` returns only the deletion commit /
  historical plan refs; `hwcloud-skillcheck validate --root .` still passes.

## Acceptance Criteria

- [ ] `hwcloud-skillcheck check --pre-commit` covers all 13 gates; exit code 1 on
      any hard failure, 0 on success.
- [ ] `check --pre-commit --skip-tests` skips `go test` (#13) but runs the rest.
- [ ] Local hook fires on Go changes (bug fixed) and is a no-op on markdown-only.
- [ ] CI invokes the same command; local and CI gate definitions are identical.
- [ ] `scripts/pre_commit_check.sh` deleted; zero remaining references in active
      code/docs (historical plan mentions allowed).
- [ ] Go unit tests for the subcommand pass; each gate function has at least one
      test (happy path + one failure path).

## Out of Scope (explicitly)

- `docker/entrypoint.sh`, `docker/scripts/run-tests.sh`, `docker/scripts/skill-exec.sh`
  — container runtime, not pre-commit tooling. Separate decision if desired.
- `clean_python.sh` — one-off repo cleanup, unrelated to the gate.
- `scripts/install_hook.go` — already Go; only its trigger constant may need a
  one-line update in Batch E2.

## Risks

- **Stale binary in CI**: CI must build the binary before running
  `check --pre-commit` (handled by `build-skillcheck.yml` job + the in-command
  rebuild as defense-in-depth).
- **Soft-gate semantics**: `golden run` / `ab compare` must remain non-fatal
  (`|| true`) to avoid blocking first-run / no-baseline scenarios.
- **Gate count drift**: if a future gate is added to the shell but not Go (or
  vice-versa), they diverge again. Mitigation: the Go subcommand is now the
  single definition; the deleted shell removes the parallel path.
