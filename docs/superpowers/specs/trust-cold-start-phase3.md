# SPEC: Trust Cold-Start Supervision Ramp (Phase 3)

> Version: 1.0.0
> Created: 2026-08-01
> Status: **FINAL_SPEC** — implemented & merged to `main`; covered by `trust_phase3_test.go`
> Target: `hwcloud-skillcheck/internal/l4` (L4 Agentic maturity, trust gating)
> Related: ADR-0013 (decision), `docs/superpowers/plans/2026-07-31-l4-maturity-upgrade.md` §Phase 3

## 1. Overview

### 1.1 Purpose

Define deterministic supervision for a `(skill, action)` pair that has little
or no outcome history, so the orchestrator never auto-approves risky
operations on an unproven track record — while still relaxing toward
autonomy as the pair demonstrates consecutive success.

### 1.2 Scope

- Per-`(skill, action)` exploration window layered on top of trust-tier gating.
- Linear risk-cap ramp based on **consecutive** success count `k`.
- Configurable window `N` (default 5) with documented provenance.

### 1.3 Exclusions

- **No cross-skill trust propagation** (deferred to Phase 4 per ADR-0013).
- **No change** to `zeroTrustScore()` baseline (still 0.245 → L0_new).
- **No relaxation** of critical/destructive hard safety overrides.

## 2. Functional Requirements

| ID | Requirement |
|----|-------------|
| FR-1 | A `(skill, action)` with **0 history** must never auto-approve, regardless of trust level. |
| FR-2 | During the exploration window (`k < N`), allowed `MaxAutoRisk` is capped by `coldStartMaxRisk(k, N)`. |
| FR-3 | The cap must **only tighten** — it never permits risk above what `EvaluateOperation` already allows. |
| FR-4 | After `k ≥ N` consecutive successes, cold-start is complete and normal tier gating resumes (cap = `""`). |
| FR-5 | `N` (ExplorationWindow) must be overridable via `SetColdStartConfig` and reset via `ResetColdStartConfig`. |
| FR-6 | `consecutiveSuccessCount` must key on the **real** `(skill, action)`, not the trust-tier label. |

## 3. Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-1 | **Deterministic**: identical history → identical decision (no LLM, no randomness). |
| NFR-2 | **O(1) typical**: count reads only the last `N` outcome records (bounded window). |
| NFR-3 | **Testable**: ramp table + edge cases asserted in `trust_phase3_test.go`. |
| NFR-4 | **Revertible**: additive function + config; no migration of existing data. |

## 4. Data Model

### 4.1 ColdStartConfig

```go
type ColdStartConfig struct {
    ExplorationWindow int // default 5; provenance = HealingPolicy.MinSamples
}
func DefaultColdStartConfig() ColdStartConfig // {ExplorationWindow: 5}
```

### 4.2 Cap mapping (`coldStartMaxRisk`)

| `k` range | returned cap | meaning |
|-----------|--------------|---------|
| `k < 2` | `"none"` | always require confirmation |
| `2 ≤ k < 3` | `"low"` | allow low-risk only |
| `3 ≤ k < N` | `"medium"` | allow up to medium-risk |
| `k ≥ N` | `""` | exploration done; tier governs |

### 4.3 API surface (additive)

```go
func EvaluateOperationWithHistory(
    score TrustScore, skill, action, opRisk, opType string, mem *OutcomeMemory,
) EvalResult

func consecutiveSuccessCount(mem *OutcomeMemory, skill, action string) int
func coldStartMaxRisk(k, window int) string

func SetColdStartConfig(c ColdStartConfig)
func ResetColdStartConfig()
```

## 5. Acceptance Criteria (test-backed)

- AC-1 (FR-1): `TestColdStart_FirstExecutionBlocked` — 0 successes → `AutoApproved=false` even at L3.
- AC-2 (FR-2/FR-4): `TestColdStart_Ramp` — k=0,1 blocked; k=2 low allowed; k=3 medium allowed; k=5 high allowed (tier L3).
- AC-3 (FR-6): `TestColdStart_MatureSkillUnaffected` — `≥N` successes → not cold-start-blocked.
- AC-4 (FR-5): `TestColdStart_ConfigOverride` — window overridable; reset reasserts default 5.

## 6. Edge Cases

- `mem == nil` → `consecutiveSuccessCount` returns 0 → cap `"none"` → always confirm.
- Non-consecutive successes (a failure breaks the streak) → `k` resets from the tail, correctly tightening.
- `opRisk` not in `RiskOrder` → treated as risk value 4 (highest) so unknown risk is never silently auto-approved.
