# TODOS — hcloud-skills

> Living backlog. Each entry has a stable ID (T-N), provenance (ADR/plan/AGENTS
> section), effort estimate (S/M/L), and a one-line description. Priority uses
> P0 (blocker) / P1 (next release) / P2 (next quarter) / P3 (someday).

## P0 — Blocker

(none right now; all P1 ship work is done)

## P1 — Next release

### T-1: Trust Phase 3 deprecate OpHistory
- **Provenance**: ADR-0009 §Migration
- **Effort**: M
- **What**: Mark `ComputeTrustScore([]OpHistory)` deprecated; route new call sites through `ComputeTrustScoreFromOutcome`; curator pipeline becomes back-fill only.

### T-2: Trust Phase 4 remove OpHistory
- **Provenance**: ADR-0009 §Migration
- **Effort**: M
- **What**: Remove curator pipeline entirely. Trust single source = outcome memory. Drop `OpHistory` type.

### T-3: L4→L5 trajectory: cross-skill orchestration
- **Provenance**: ADR-0009 §Open follow-ups; L4 Orchestrator gap
- **Effort**: L
- **What**: Topology graph exists (topology.go) but no runtime cross-skill delegation. Wire it through `HandleFault` to dispatch a task across multiple `huaweicloud-*-ops` skills.

## P2 — Next quarter

### T-4: Performance — OutcomeMemory.RecentOutcomes O(N) read
- **Provenance**: Eng-review Eng-T4 (2026-07-28)
- **Effort**: M
- **What**: Current implementation reads entire JSONL file per call. With 1k records/s writes, file grows fast; reads will bottleneck. Add in-memory LRU cache (last 100 records) invalidated on `Record`, or document and accept at scale.

### T-5: Performance — ContextMemory batched mutations
- **Provenance**: Eng-review Eng-T5 (2026-07-28)
- **Effort**: S
- **What**: Currently rewrites entire JSON on every mutation. Batch via `Save()` only at task-finalize time. Mutation API queues deltas.

### T-6: hashContext strip volatile args
- **Provenance**: Eng-review Eng-m1 (2026-07-28)
- **Effort**: S
- **What**: If `step.Args` includes timestamps/IDs, every call has unique hash — defeating MatchOutcomes. Strip known-volatile args before hashing; document which fields are stable.

### T-7: preFetchFailurePatterns shared helper
- **Provenance**: Eng-review Eng-M2 (2026-07-28)
- **Effort**: S
- **What**: `preFetchPatterns` is byte-similar in `execution.go` and `orchestrator.go` (50 lines, two mutex names, both `SetLimit(NumCPU)`). Extract to `persistence.go`.

## P3 — Someday / L4→L5

### T-8: ADR-0011: cross-skill delegation protocol
- **Provenance**: T-3 above
- **Effort**: L
- **What**: Define how one skill invokes another (sync vs async, context propagation, error handling). Needed before cross-skill orchestration can ship.

### T-9: Healing decision observability — Prometheus exporter
- **Provenance**: ADR-0009 §Open follow-ups
- **Effort**: M
- **What**: Current counters are in-process. Add a `metrics` subcommand that exposes them in Prometheus text format on `:9090/metrics`. Optional scraping by ops.

### T-10: `EnsureMemoryDir()` shared helper
- **Provenance**: AGENTS.md §Open follow-ups
- **Effort**: S
- **What**: Both `outcome_memory.go` and `context_memory.go` `mkdir .l4-memory` (mode 0700). Extract to a shared helper when a third caller appears (YAGNI until then).

## Resolved (last 30 days)

(Use this section to log recently closed items for audit trail.)

- 2026-07-28: ADR-0007 outcome memory + self-healing shipped (v0.2.1)
- 2026-07-28: ADR-0008 cross-call memory shipped
- 2026-07-28: ADR-0009 trust from outcome memory Phase 1+2 shipped
- 2026-07-28: ADR-0010 RealExecutor + subprocess semantics shipped
- 2026-07-28: Healing decision observability (slog + counters + memory inspect) shipped
- 2026-07-28: Cross-platform release workflow (Taskfile + CI) shipped
- 2026-07-28: CLI manual updated to cover all current subcommands
