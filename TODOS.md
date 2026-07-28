# TODOS — hcloud-skills

> Living backlog. Each entry has a stable ID (T-N), provenance (ADR/plan/AGENTS
> section), effort estimate (S/M/L), and a one-line description. Priority uses
> P0 (blocker) / P1 (next release) / P2 (next quarter) / P3 (someday).

## P0 — Blocker

(none right now)

## P1 — Next release

(none right now)

## P2 — Next quarter

(none right now)

## P3 — Someday / L4→L5

(none right now)

## Resolved (last 30 days)

(Use this section to log recently closed items for audit trail.)

- 2026-07-28: ADR-0007 outcome memory + self-healing shipped (v0.2.1)
- 2026-07-28: ADR-0008 cross-call memory shipped
- 2026-07-28: ADR-0009 trust from outcome memory Phase 1+2 shipped
- 2026-07-28: ADR-0010 RealExecutor + subprocess semantics shipped
- 2026-07-28: Healing decision observability (slog + counters + memory inspect) shipped
- 2026-07-28: Cross-platform release workflow (Taskfile + CI) shipped
- 2026-07-28: CLI manual updated to cover all current subcommands
- 2026-07-28: T-1 Trust Phase 3 — `ComputeTrustScore([]OpHistory)` deprecated; new call sites routed through `ComputeTrustScoreFromOutcome`
- 2026-07-28: T-2 Trust Phase 4 — curator pipeline removed; trust single source = outcome memory; `OpHistory` type dropped (`TestOpHistory_CompletelyRemoved`)
- 2026-07-28: T-3 cross-skill runtime orchestration — `ExpandMatchedWithDelegates` wires transitive `DelegatesTo` skills into execution plan; `HandleFault` dispatches all planned steps via `RunExecutionLoopWithHealing`
- 2026-07-28: T-5 ContextMemory batched mutations — mutation API queues dirty buffer; `Flush()` once at task-finalize (`HandleFault`)
- 2026-07-28: T-6 `hashContext` strips volatile CLI flags (time windows, pagination, client/request IDs) so MatchOutcomes correlates repeats
- 2026-07-28: T-7 `preFetchFailurePatterns` extracted to `persistence.go`; orchestrator + execution share one helper
- 2026-07-28: T-8 ADR-0011 cross-skill delegation protocol accepted (orchestrator-mediated sync pipeline)
- 2026-07-28: T-4 OutcomeMemory per-(skill,action) recent cache (≤100) — first RecentOutcomes scans disk once; later hits + Record updates skip rescan
- 2026-07-28: T-9 `metrics` subcommand serves healing/trust counters in Prometheus text on `:9090/metrics`
- 2026-07-28: T-10 `EnsureMemoryDir()` shared helper — `NewOutcomeMemory` / `NewContextMemory` both call it
- 2026-07-28: Doc/Spec backlog closed — GCL trust-boundary P0 + Harness P1P2 specs → Accepted; P1 evidence DoD checked; CLI Alpine no-python smoke wired; `.planning/.../l4-orchestration/SPEC-PLAN.md` superseded by Go L4 + ADR-0007…0011; `02-REVIEW.md` WR-02/03/04/06 closed
