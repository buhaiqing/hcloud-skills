# ADR-0013: Trust Cold-Start Supervision Ramp (Phase 3)

- Status: Accepted
- Date: 2026-08-01
- Deciders: hcloud-skills maintainers (agent-autonomous per user mandate)
- Supersedes: —
- Related: ADR-0009 (Trust from Outcome Memory), `internal/l4/trust.go`, `docs/superpowers/plans/2026-07-31-l4-maturity-upgrade.md` §Phase 3

## Context

A brand-new `(skill, action)` pair has **zero outcome history**. The trust
tier computed from an empty `OutcomeMemory` is `zeroTrustScore() = 0.245 →
L0_new`, which already forces `always-confirm`. That satisfies acceptance
criterion #1 ("first execution never auto-approved"), but it does not
define *how supervision relaxes* as the skill proves itself.

Without an explicit exploration window, two failure modes exist:

1. **Too loose**: a skill that happened to succeed twice immediately jumps
   to `L2_established` (MinScore 0.6) and auto-runs `medium`-risk ops. Two
   samples is not statistical evidence.
2. **Too rigid**: the tier never tightens below the global `MaxAutoRisk`
   for the skill's *current* level, so a high-trust skill that tries a
   *new* action reuses the old action's trust — first-time risk on an
   untried operation is unbounded.

We need a deterministic, per-`(skill, action)` **exploration cap** layered
on top of tier gating: widen the allowed risk tier stepwise with consecutive
successes, then hand control back to the trust tier after `N` successes.

## Decision

Add `EvaluateOperationWithHistory(score, skill, action, opRisk, opType, mem)`
in `internal/l4/trust.go`. It wraps `EvaluateOperation` and applies a
**linear cold-start cap** derived from `consecutiveSuccessCount`:

| Consecutive successes `k` | Exploration cap (`coldStartMaxRisk`) |
|---------------------------|--------------------------------------|
| `k < 2`                   | `none` (always confirm)              |
| `2 ≤ k < 3`               | `low`                                |
| `3 ≤ k < N`               | `medium`                             |
| `k ≥ N`                   | `""` → exploration complete; tier governs |

`N = ColdStartConfig.ExplorationWindow`, default **5**. Provenance:
`HealingPolicy.MinSamples = 5` (`self_healing.go`) — the documented
"enough samples to form a reliable prior" lower bound. The cap only
**tightens**; the critical/destructive hard overrides inside
`EvaluateOperation` always win.

The orchestrator call site (`orchestrator.go:292`) passes the **real**
`trustSkill`/`trustAction`, not the trust-tier label, so the count keys on
the actual operation pair.

### Scope: A vs B (autonomous decision)

- **A — exploration-window gate only** (selected). Tightens per-`(skill,
  action)` risk during the first `N` successes. Self-contained, testable,
  reversible.
- **B — A + cross-skill trust propagation** (deferred). Inherit partial
  trust across related operations under the same product. **Rejected for
  this phase**: it couples cross-skill state and conflicts with ADR-0009
  ("trust's single source of truth = outcome memory per `(skill,
  action)`"). Propagating trust during a cold-start window would amplify
  first-cause errors. Revisit in Phase 4 once end-to-end behavior is
  measured.

## Consequences

### Positive

- Deterministic ramp (GCL spec §14): same history → same decision, no LLM.
- First-time operations on a high-trust skill are still supervised until
  they earn their own track record.
- Provenance-backed `N=5` — no magic number.
- Fully testable (`trust_phase3_test.go`): ramp table, first-execution
  blocked, mature-unaffected, config-override.

### Negative

- Adds one config knob (`ColdStartConfig.ExplorationWindow`) to maintain.
- Cold-start window adds `N` forced-HITL round-trips before autonomy;
  acceptable trade for safety.

### Neutral

- Does not change the `zeroTrustScore` baseline (still 0.245 → L0_new).
  The ramp is an *additional* tightening above the tier.

## Alternatives Considered

1. **Raise `zeroTrustScore` to a non-zero prior and skip the window.**
   Rejected: masks the "how does it relax" question; no per-action track.
2. **Bayesian prior + posterior update.** Rejected: over-engineered for a
   count-based ramp; harder to test deterministically.
3. **Cross-skill propagation now (B).** Rejected — see Scope above.

## Rollout

Shipped in Phase 3 commit on `main`. Covered by `trust_phase3_test.go`.
No migration needed (additive function + config). `SetColdStartConfig`/
`ResetColdStartConfig` allow tests and future tuning.

## Open Questions

- Should `N` be per-product (higher blast-radius products → larger window)?
  Defer to Phase 4 measurement.
- Cross-skill propagation (B): revisit after autonomous-loop telemetry.
