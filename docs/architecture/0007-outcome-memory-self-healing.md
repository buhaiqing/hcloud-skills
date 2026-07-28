# ADR-0007: Outcome Memory + Self-healing Recovery for L4 Orchestrator

- Status: Proposed
- Date: 2026-07-28
- Deciders: hcloud-skills maintainers
- Supersedes: —
- Related: ADR-0006 (L4 orchestrator), gcl-spec.md §14 (trust boundary), `docs/superpowers/specs/aiops-l5-autonomous.md`

## Context

The L4 orchestrator (`hwcloud-skillcheck/internal/l4/`) executes multi-step cloud
operations through 24 `huaweicloud-*-ops` skills. Today it has:

- Per-task persistence (`TaskState` → `.l4-tasks/<id>.json`)
- Static topology graph (`topology.go`)
- RBAC pre-check (`rbac.go`)
- GCL structural critic (`gcl.StructuralCritic`)
- Trust scoring with 30-day recency half-life (`trust.go`)

What's missing — and what blocks the L3 → L4 maturity jump identified in the
prior Gartner assessment:

1. **No cross-task outcome memory.** A failure in task `t1` teaches nothing to
   task `t2` run five minutes later, even when `(skill, action, context)` are
   identical. `OpHistory` is fed by an external curator, not by the orchestrator
   itself.
2. **No self-healing.** A step that fails goes straight to `FailTask`. There is
   no in-loop policy that says "this is a transient auth-token error; retry once
   with backoff" or "this `(ecs, delete-instance)` action has failed 4 of the
   last 5 times in the last hour; escalate to human".
3. **Skill SKILL.md files are frozen.** The 24 skills cannot be modified as
   part of this work — that constraint drives where the new logic must live
   (in `internal/l4/`, not in the skill runbooks).

The user has chosen **Outcome Memory + Self-healing** as the highest-ROI
maturity leap (24 skills already exist; marginal returns on a 25th are near
zero; the gap is "learns from mistakes").

## Decision

Introduce two new modules inside `hwcloud-skillcheck/internal/l4/`:

1. **`outcome_memory.go`** — an append-only JSONL outcome log keyed by
   `(skill, action, context_hash)`, queried at the start of every step.
2. **`self_healing.go`** — a `HealingPolicy` struct that consults
   `OutcomeMemory` pre-execution (to bump risk tier or skip) and post-failure
   (to decide retry / skip / escalate).

Wire both as **hooks inside `RunExecutionLoop`** (`execution.go:27`). The hooks
are additive — they call existing functions (`CheckCommandPermission`,
`PersistTask`) and write back to `OutcomeMemory`. **No SKILL.md file is
modified.**

### Why JSONL (not SQLite)

- Append-only writes are atomic by line; no WAL/rollback machinery.
- The orchestrator process is short-lived; we never need concurrent readers.
- `TaskState` already uses the same on-disk pattern (`.l4-tasks/<id>.json`),
  so the storage convention is consistent.
- Ponytail principle: stdlib `encoding/json` + `os.AppendFile` is ~50 lines.
  SQLite would be ~300 lines plus a CGO-free driver choice.

If query latency becomes a bottleneck above ~10k outcome records, migrate to
SQLite in a follow-up ADR — don't pre-build it.

### Why "policy struct" (not a config file or LLM call)

- Healing decisions must be **deterministic and auditable** (GCL spec §14).
- An LLM in the hot path would violate the existing pre-flight → execute →
  validate → recover contract.
- A `HealingPolicy` struct with default fields matches the existing
  `TrustWeights` / `TrustLevels` pattern in `trust.go`.

### Why pre-exec + post-failure (not just post-failure)

- Pre-exec check is **cheap insurance**: if a `(skill, action)` pair has 100%
  failure rate in the last hour, asking RBAC + GCL to evaluate it again is
  wasted work and risks an automatic destructive retry.
- The cost is one map lookup and an integer compare — sub-millisecond.

## Consequences

### Positive

- **MTTR drops** for transient errors (token expiry, rate-limit) without
  human escalation. Expected: 30–60% reduction in failures-that-need-human.
- **Cross-task learning** without changing skill runbooks.
- **Auditable**: every decision is a `HealingDecision` record on disk; the
  trust tier can later consume the same data.
- **Plays nicely with existing persistence**: same directory conventions
  (`.l4-memory/`), same JSON shape, same chmod (0700).

### Negative

- **Storage growth**: at 100 outcomes/day × 24 skills × 365 days ≈ 1M lines
  / year. Mitigated by 90-day retention pruning on init.
- **Risk of over-retry on non-idempotent ops**. Mitigation: the default policy
  refuses to retry any action whose verb matches the destructive list
  (`delete`, `terminate`, `drop`, `remove` — already enumerated in
  `execution.go:226-230` for risk inference).
- **Memory poisoning**: a buggy skill could record false successes. Mitigation:
  trust score continues to be computed from `OpHistory` (separate curation
  path); outcome memory is raw signal, not curated signal.

### Neutral

- The 24 skill SKILL.md files stay untouched. This is by design — it's the
  whole reason the feature lives in `internal/l4/`.
- `hwcloud-skillcheck` gains no new public CLI subcommand in this ADR. The
  hooks are internal to the orchestrator. A future `hwcloud-skillcheck
  memory inspect` subcommand can be added later if needed.

## Alternatives Considered

1. **SQLite-backed outcome store.** Rejected: CGO-free driver choice
   (`modernc.org/sqlite`) is fine but adds ~300 lines and a dependency for
   a feature that handles <10k rows.
2. **Wire self-healing into the skill runbooks themselves.** Rejected: this
   would require editing all 24 SKILL.md files, contradicting the user's
   stated constraint.
3. **LLM-driven healing decision.** Rejected: violates the
   pre-flight → execute → validate → recover contract and adds a
   non-deterministic step in the hot path.
4. **Skip pre-exec check, do post-failure only.** Rejected: pre-exec is
   cheap and prevents asking RBAC/GCL to evaluate commands we already know
   will fail.

## Rollout

Implement in 4 phases via the companion plan (`docs/superpowers/plans/2026-07-28-outcome-memory-self-healing.md`):

1. **Storage** (`outcome_memory.go` + tests)
2. **Policy** (`self_healing.go` + tests)
3. **Wiring** (modify `execution.go`; reuse existing test patterns)
4. **Validation** (integration test against 24-skill fixture corpus)

Each phase is independently testable and reverts cleanly. After phase 3, the
system behaves identically when `OutcomeMemory` is empty (L0 trust policy
default). After phase 4, the self-healing path is exercised on real
fixtures.

## Open Questions

- **Retention default**: 90 days proposed; matches `RecencyHalfLifeDays` × 3.
  Confirm with ops.
- **Per-skill opt-out**: should `huaweicloud-iam-ops` (destructive blast
  radius) be able to disable auto-retry via frontmatter? Defer to a
  follow-up ADR if requested.