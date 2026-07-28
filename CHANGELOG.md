# Changelog

All notable changes to hcloud-skills are documented here. Versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
