# ADR-0009: Trust Score from Outcome Memory (L5 Self-evolution)

- Status: Proposed
- Date: 2026-07-28
- Deciders: hcloud-skills maintainers
- Supersedes: —
- Related: ADR-0007 (Outcome Memory), `internal/l4/trust.go` (existing curated OpHistory)

## Context

Today the orchestrator computes trust from `OpHistory` (`internal/l4/trust.go:49`):

```go
type OpHistory struct {
    Outcome   string  // success / failure
    Timestamp string
    RiskLevel string
    HadRetry  bool
}
```

The history is **curated externally** — populated by a separate pipeline that ingests GCL traces after the fact. This means:

1. Trust is stale. A skill that fails 10 times today doesn't update trust until tomorrow's batch.
2. Trust ignores the orchestrator's own self-healing decisions. A `(skill, action)` pair that always retries-then-succeeds looks like 100% failures to the curator.
3. The curator is a separate process. If it breaks, trust silently stops updating.

ADR-0007's Outcome Memory records every step the orchestrator executes, in real time, with structured fields. The same data can feed trust scoring without depending on the curator pipeline.

## Decision

Make Outcome Memory the **primary** source of trust history. Keep `OpHistory` as a back-compat shim for one release; deprecate it after that.

### Data flow

```
  Step dispatched
       │
       ▼
  RunExecutionLoopWithHealing
       │
       ├─► OutcomeMemory.Record(OutcomeRecord)  ──── NEW: source of truth
       │                                              │
       │                                              ▼
       │                                       (optional) fan out to
       │                                              │
       │                                              ▼
       │                                       TrustScore.ComputeFromRecords()
       │                                              │
       ▼                                              ▼
  StepResult                              trust.score (in-memory cache, refreshed on Record)
```

The orchestrator holds a small in-memory `trustCache map[skill]*TrustScore` that's
incrementally updated on each `Record()`. No background scan, no batch job.

### Mapping OutcomeRecord → trust inputs

| trust.go input | OutcomeRecord field | Notes |
|----------------|---------------------|-------|
| `Outcome` (success/failure) | `Outcome` (success/failure/blocked) | blocked → counted as failure (RBAC denied = bad outcome) |
| `Timestamp` | `Timestamp` | RFC3339 → already in `trust.go` format |
| `RiskLevel` | `Risk` (high/medium/low/critical) | same enum |
| `HadRetry` | `RetryCount > 0` | new — was not tracked before |

`error_class` (transient/permanent/unknown) is **not** mapped — trust should not
penalize transient failures that were retried successfully. The retry
excludes the failure from the success-rate calculation, mirroring the
trust.go `error_recovery` weight.

### Compute algorithm

Replace `ComputeTrustScore([]OpHistory)` with `ComputeTrustScore([]OutcomeRecord)`
in `trust.go`. Same weights (success_rate 0.35, consistency 0.20, recency 0.20,
complexity_mastery 0.15, error_recovery 0.10). Adjust components:

- `success_rate` → same definition but on OutcomeRecord.Outcome
- `consistency` → same (variance of outcomes)
- `recency` → same (30-day half-life)
- `complexity_mastery` → same (inverse-weighted by risk tier; now feeds
  from OutcomeRecord.Risk directly instead of being inferred)
- `error_recovery` → new: `(retries-then-success) / (retries)` from
  `OutcomeRecord.RetryCount > 0 AND OutcomeRecord.Outcome == "success"`.

### Migration

1. **Phase 1 (this ADR, alongside ADR-0007)**: Keep `OpHistory` API.
   Add `ComputeTrustScoreFromOutcome([]OutcomeRecord)` next to
   `ComputeTrustScore([]OpHistory)`. Call sites that have access to
   `*OutcomeMemory` use the new function; existing curated callers keep
   the old.
2. **Phase 2 (post-merge, ~1 release)**: Default new call sites to the
   outcome-memory path. Add a metric `trust_source{from="outcome_memory"}`
   so we can watch the cutover.
3. **Phase 3 (~2 releases)**: Mark `ComputeTrustScore([]OpHistory)`
   deprecated. Curator pipeline becomes a back-fill for the gap between
   ADR-0007 shipping and Phase 1 cutover.
4. **Phase 4 (TBD)**: Remove curator pipeline.

### Why incremental cache instead of background scan

- Write throughput is 1 outcome/step; 1000 records/s NFR-3 from
  ADR-0007 spec.
- Cache hit rate is ~100% for in-flight tasks (same `(skill, action)`).
- Background scans would race with concurrent Record() — same data-loss
  class as ADR-0007's PruneOlderThan.

### Why not LLM-derived trust

- Trust scoring must be deterministic (GCL spec §14).
- LLM in the hot path violates the pre-flight → execute → validate →
  recover contract.
- The metrics are simple arithmetic; an LLM would add latency without
  insight.

## Consequences

### Positive

- **Trust reflects reality in real time.** A skill that fails 5 times in
  an hour degrades immediately, not at next curator run.
- **Self-healing counts.** A `(skill, action)` with `retry_count=2,
  outcome=success` raises trust, not lowers it — opposite of what the
  curator pipeline computes.
- **No external dependency.** Trust survives a broken curator pipeline.
- **One source of truth.** No drift between curator view and live view.

### Negative

- **Migration risk.** Phase 2-3 cutover needs careful monitoring.
  Curator-sourced scores will diverge from outcome-memory-sourced scores
  during the gap.
- **Cache invalidation.** If `HealingPolicy` changes between
  invocations, the trust cache is stale. Mitigation: cache key
  includes policy hash.
- **No ground truth.** Outcome memory is what the orchestrator did;
  it isn't what the user wanted. Mitigation: a small user-feedback
  field on `TaskSummary` (future) lets users mark "this was wrong" to
  override the auto-derived trust.

### Neutral

- ADR-0007's Outcome Memory schema needs one extra field
  (`retry_count`) — already present in the spec.
- Existing curator pipeline stays as a back-fill until Phase 4.

## Alternatives Considered

1. **Keep curator as the only source.** Rejected: solves nothing — the
   curator's gap is exactly the motivation for ADR-0007.
2. **Replace curator entirely on Phase 1.** Rejected: too aggressive —
   curated history covers pre-ADR-0007 runs and external workflows.
3. **LLM summarizer over Outcome Memory for trust.** Rejected:
   non-deterministic, slow, and adds no insight over arithmetic.
4. **Background scan over the JSONL file every N minutes.** Rejected:
   same race class as PruneOlderThan; incremental cache is simpler.

## Rollout

Phases above. Phase 1 ships with ADR-0007 implementation (~1 week of work
beyond ADR-0007). Phase 2 starts after 1 release of stable behavior.

## Open Questions

- **Cache invalidation across processes**: if two `hwcloud-skillcheck`
  processes run against the same root, each holds its own cache.
  Last-writer-wins is acceptable for v1.
- **User-feedback field**: defer to a follow-up ADR if requested.