# Changelog

All notable changes to hcloud-skills are documented here. Versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.4.0] - 2026-08-01

### Added
- **`check merge-verify`** — parallel-execution collection gate. After multiple
  subagents modify files in parallel, validates cross-file consistency (markdown
  links / backtick path targets) across every changed file and emits a per-agent
  execution log. Fails (exit 1) when any subagent reports `status: failed`.
  Docs + example fixture (`scripts/fixtures/merge-verify-example.json`) + unit
  and orchestrator integration tests.

### Fixed
- **Go SDK module paths misclassified as repo paths** — `looksLikeRepoPath`
  (markdown-links backtick branch) and `checkReferencesFile` (references-links)
  now share `isGoSDKModulePath`, excluding `huaweicloud-sdk-*` module paths
  (e.g. `huaweicloud-sdk-go-v3/...`) from false "missing target" reports. The
  regular-link branch (`mdLinkRe`) was also fixed (GCL 3-round review).
- **Broken `scripts/gcl_runner.py` backtick refs** — removed from 24 SKILL.md
  files + 3 design specs (the script was deleted in the Phase 5 Go migration but
  references lingered).

## [v0.3.0] - 2026-08-01

### Added
- **`check --pre-commit` Go subcommand** — the unified pre-commit gate replacing
  `scripts/pre_commit_check.sh` (ADR-0014). Local git hook and CI now share one
  gate definition. `--check-only` (CI mode: skip rebuild, add CI-only soft gates)
  and `--test-retries N` (flaky-race retry loop) modes.
- **`.githooks/pre-commit` template** — repository-managed hook with bootstrap
  guard (builds `bin/hwcloud-skillcheck` on demand on fresh clones).
- **`l4 autofix`** — autonomous remediation executor with safety gate chain
  (dry-run → destructive HITL → success-rate threshold → render → preconditions →
  execute → verify → rollback + record); closed-loop feedback into playbook stats.
- **Pre-execution pattern risk gate** — a command matching a high-risk failure
  pattern is skipped (`SKIPPED_BY_PATTERN_RISK`) instead of executed-then-failed.
- `references/go-coding-standards.md` — G1-G10 Go coding standards.

### Changed
- **CI unified** — 12 scattered shell steps in `validate-skills.yml` replaced with
  a single `check --pre-commit --check-only --test-retries 2` call.
- **AGENTS.md** — added GCL Auto-Execution Gate, Phase 6 rules, and CA-11..CA-16
  compound-asset entries.

### Removed
- **`scripts/pre_commit_check.sh`** — deleted (replaced by `check --pre-commit`).

### Fixed
- `TestHandleFault_DecisionAutoProceed` flakiness — added Priority tie-breaker to
  `MatchFaultSkills` sort (non-deterministic `sort.Slice` on equal confidence).

## [v0.2.2] - 2026-07-28

### Added
- **ADR-0010: Real Executor** — `os/exec.CommandContext` wrapper in `internal/l4/execution.go` replaces the dry-run-only StubExecutor. Default timeout 60s, output cap 1 MB, env passthrough. Bash-`c` shell preserves user shell semantics.
- **Trust Phase 2 (ADR-0009 cutover)** — `ComputeTrustScoreFromOutcome(skill, action, mem, policyHash)` reads OutcomeRecord history and computes composite TrustScore. `LookupTrust()` is the production path; `OpHistory` becomes back-fill only.
- **`trust stats` subcommand** — exposes `TrustSourceCounter` (FromOutcomeMemory / FromOpHistory) for monitoring cutover progress.
- **Healing decision observability** — `slog.Info("healing_decision", ...)` on every non-proceed hook; `HealingMetrics` atomic counters for monitoring.
- **`memory inspect` CLI** — `hwcloud-skillcheck memory inspect --root .` prints outcome + context memory state.
- **`/v0.2.2` workflow** — `task release VERSION=0.2.2` tags + pushes; CI builds 5 platform binaries + SHA256SUMS via `softprops/action-gh-release`.
- **TODOS.md** — consolidated P1/P2/P3 backlog with provenance + effort estimates.
- **docs/manual/hwcloud-skillcheck.md** — extended from 111 to ~600 lines; covers all current CLI subcommands.

### Changed
- `RunExecutionLoop` / `RunExecutionLoopWithHealing` now use the real executor by default (was StubExecutor). `result.Success` reflects actual exit code, not GCL critic verdict.
- AGENTS.md glossary extended with RealExecutor / Phase 2 / observability terms.
- README.md (EN + CN) updated with the full release workflow (`task release` / `task release-build` / `task release-local`).

### Deprecated
- `ComputeTrustScore([]OpHistory)` — superseded by `ComputeTrustScoreFromOutcome`. Will be removed in v0.3.0.

### Fixed
- `execute.go` `io.LimitWriter` undefined (replaced with custom `limitedBuffer` type that applies backpressure).
- `Taskfile.yml` inline `|| { ... }` shell blocks rejected by YAML parser (split into standalone cmds).

## [v0.2.1] - 2026-07-28

### Added
- **ADR-0007: Outcome Memory + Self-healing** — `OutcomeMemory` (JSONL append-log, 90-day prune) + `Self-healing` hooks (`PreExecHook` / `PostFailureHook`).
- **ADR-0008: Cross-call Memory** — `ContextMemory` (single JSON doc, atomic save, 24h session rotation).
- **ADR-0009: Trust from Outcome Memory** — Phase 1 (coexist) of trust score cutover.
- Cross-platform release workflow via Taskfile + GitHub Action.
