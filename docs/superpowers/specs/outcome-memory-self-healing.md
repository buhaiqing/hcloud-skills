# Spec: Outcome Memory + Self-healing Recovery

> Companion to `docs/architecture/0007-outcome-memory-self-healing.md`. Implements the
> ADR's "Decision" section in concrete terms.

## 1. Goals

- **G1.** Persist the outcome of every step the L4 orchestrator executes,
  across task boundaries, so the orchestrator can act on its own history.
- **G2.** Decide, before each step, whether prior history for the same
  `(skill, action, context)` makes the step a known-bad idea (skip) or a
  known-noisy idea (raise risk tier).
- **G3.** After a step fails, attempt self-healing when the failure class is
  transient and the action is idempotent, without modifying the 24 skill
  runbooks.
- **G4.** Stay fully observable: every decision is on disk and queryable.

## 2. Non-goals

- **N1.** Do **not** edit any of the 24 `huaweicloud-*-ops/SKILL.md` files.
- **N2.** Do **not** introduce an LLM call in the hot path. Healing decisions
  are deterministic on the outcome log.
- **N3.** Do **not** add a new public CLI subcommand in this iteration.
  (A future `hwcloud-skillcheck memory` subcommand can be added later.)
- **N4.** Do **not** migrate off JSONL. SQLite is a follow-up if the file
  grows past 10k lines.
- **N5.** Do **not** change RBAC semantics. Self-healing only chooses
  retry/skip/escalate; it does **not** override RBAC denials.

## 3. Functional Requirements

### FR-1: Outcome persistence

Every `StepResult` produced by `RunExecutionLoop` (`execution.go:139`) MUST be
appended to `<root>/.l4-memory/outcomes.jsonl` after `PersistTask` succeeds.

Schema per line:
```json
{
  "id": "uuid-v4 string",
  "ts": "2026-07-28T08:30:00Z",
  "task_id": "abcd1234ef567890",
  "skill": "huaweicloud-ecs-ops",
  "action": "delete-instances",
  "context_hash": "sha256-of-step.Args-or-stable-input",
  "outcome": "success|failure|blocked",
  "error_class": "transient|permanent|unknown",
  "error_msg": "truncated to 200 chars",
  "retry_count": 0,
  "duration_ms": 1234,
  "risk": "high",
  "rbac_decision": "allowed",
  "gcl_decision": "PASS"
}
```

- One JSON object per line. No nested arrays.
- `id` MUST be unique across all records (uuid v4).
- `ts` MUST be ISO-8601 UTC, produced by `NowISO()`.
- `error_msg` MUST be truncated to 200 chars (mirror `internal/learning/trace.go:197`).

### FR-2: Outcome query API

Three read methods on the `OutcomeMemory` type:

```go
// RecentOutcomes returns up to n records for (skill, action) ordered
// most-recent first. n <= 0 returns all matching records.
RecentOutcomes(skill, action string, n int) ([]OutcomeRecord, error)

// MatchOutcomes returns records for (skill, action, context_hash) within
// the last `lookback` duration. lookback <= 0 means "all".
MatchOutcomes(skill, action, contextHash string, lookback time.Duration) ([]OutcomeRecord, error)

// PruneOlderThan drops records older than `cutoff`. Called once on init
// from `NewOutcomeMemory`. Records older than 90 days MUST be dropped.
PruneOlderThan(cutoff time.Time) (int, error)
```

### FR-3: Healing policy

A `HealingPolicy` value configures the pre-exec and post-failure hooks. Default
zero value is **safe** (no retries, no skips):

```go
type HealingPolicy struct {
    // MaxRetries is the upper bound on automatic retries per step.
    // 0 (default) = no auto-retry.
    MaxRetries int

    // RetryBackoff is the wait between retry attempts.
    // 0 (default) = no backoff; first retry runs immediately.
    RetryBackoff time.Duration

    // DestructiveVerbs lists action verbs that MUST NEVER be auto-retried.
    // Default = {"delete","terminate","destroy","drop","remove"} — same set
    // already used in execution.go:226-230 for risk inference.
    DestructiveVerbs []string

    // FailureRateSkipThreshold: when recent (skill,action) has at least
    // this fraction of failures (>= MinSamples), skip the step with a
    // HealingDecision.Skipped=true. 0 (default) disables skipping.
    FailureRateSkipThreshold float64

    // MinSamples: minimum number of recent outcomes before skip logic
    // kicks in. Avoids premature skip on noisy first runs. Default 5.
    MinSamples int

    // LookbackWindow: only consider outcomes within this window for
    // skip decisions. Default 1 hour.
    LookbackWindow time.Duration
}
```

### FR-4: Pre-exec hook

`PreExecHook(step TaskStep, mem *OutcomeMemory, p HealingPolicy) HealingDecision`

- Returns `HealingDecision{Action: "proceed"}` by default.
- If `mem.RecentOutcomes(step.Skill, step.Action, p.MinSamples)` shows
  failure rate `>= p.FailureRateSkipThreshold`, return
  `HealingDecision{Action: "skip", Reason: "high historical failure rate"}`.
- The hook MUST NOT touch RBAC or GCL — those run after the hook returns
  `proceed`.

### FR-5: Post-failure hook

`PostFailureHook(step TaskStep, result StepResult, retryCount int, mem *OutcomeMemory, p HealingPolicy) HealingDecision`

Inputs:
- `step`: the failing step
- `result`: the just-recorded StepResult with `Success: false`
- `retryCount`: how many retries have already been attempted for this step
- `mem`, `p`: as above

Decision tree:
1. If `retryCount >= p.MaxRetries` → `{Action: "escalate", Reason: "max retries reached"}`.
2. If step verb matches `p.DestructiveVerbs` → `{Action: "escalate", Reason: "destructive op, no auto-retry"}`.
3. If `result.Error` matches a known transient pattern (`"timeout"`, `"token expired"`, `"401"`, `"429"`, `"503"`, `"connection reset"`) → if retryCount < MaxRetries, return `{Action: "retry", Reason: "transient error: <matched>"}`.
4. Otherwise → `{Action: "escalate", Reason: "non-transient error"}`.

The caller is responsible for executing the retry (sleep `RetryBackoff`,
re-run the RBAC + GCL + execute path, then call `PostFailureHook` again
with `retryCount+1`).

### FR-6: Wiring in `RunExecutionLoop`

`internal/l4/execution.go:27` MUST be modified to add three call sites:

1. **Before RBAC check** (around line 64): call
   `PreExecHook(step, mem, p)`. If the decision is `skip`, append a
   `StepResult{Success: false, Error: "skipped: <reason>"}`, increment
   `task.CurrentStep`, persist, and continue to next step. Do NOT call
   `FailTask` — skipping is not failing.

2. **On `Success: false` from a step** (after appending the result, around
   line 139): call `PostFailureHook`. If decision is `retry`, sleep
   `RetryBackoff`, do NOT increment `CurrentStep`, and `continue` the loop
   so the same step re-runs. If decision is `escalate`, fall through to
   existing `FailTask` path.

3. **After successful step**: nothing changes (the existing append already
   records the outcome).

### FR-7: Storage conventions

- Directory: `<root>/.l4-memory/` (matches existing `.l4-tasks/` convention).
- Mode: `0700` (owner-only), same as `PersistTask`.
- File: `outcomes.jsonl`.
- Pruning: `NewOutcomeMemory` MUST call `PruneOlderThan(time.Now().Add(-90*24*time.Hour))`
  before returning.
- Concurrent writers: the file is append-only. Two writers racing on the
  same step is a non-issue at this scale; if it becomes one, switch to
  `flock(LOCK_EX)` per write.

### FR-8: Testability

All three new types (`OutcomeMemory`, `HealingPolicy`, `HealingDecision`)
MUST be unit-testable without touching real disk in the test path —
construct the memory with a temp dir via `t.TempDir()`.

## 4. Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-1 | Pre-exec hook latency | < 5 ms p95 |
| NFR-2 | Post-failure hook latency (no retry) | < 5 ms p95 |
| NFR-3 | Outcome write throughput | >= 1000 records/s sustained |
| NFR-4 | Memory footprint per record | < 1 KB on disk |
| NFR-5 | Retention pruning time | < 100 ms for 10k records |
| NFR-6 | Determinism | Same inputs → same decisions (no RNG, no time-of-day in core logic; `NowISO()` only for the timestamp field) |

## 5. Data Model

```go
// OutcomeRecord is one row in outcomes.jsonl.
type OutcomeRecord struct {
    ID          string    `json:"id"`
    Timestamp   string    `json:"ts"`
    TaskID      string    `json:"task_id"`
    Skill       string    `json:"skill"`
    Action      string    `json:"action"`
    ContextHash string    `json:"context_hash"`
    Outcome     string    `json:"outcome"`        // success | failure | blocked
    ErrorClass  string    `json:"error_class"`    // transient | permanent | unknown
    ErrorMsg    string    `json:"error_msg,omitempty"`
    RetryCount  int       `json:"retry_count"`
    DurationMS  int64     `json:"duration_ms"`
    Risk        string    `json:"risk"`
    RBACDecision string   `json:"rbac_decision"`
    GCLDecision  string   `json:"gcl_decision"`
    // GCL campaign process metrics; see §5.1. Pointers distinguish a measured
    // 0.0 from "not measured" under omitempty. Legacy rows predate these
    // fields and may carry a 16-hex id instead of the uuid v4 §5 requires.
    GCLExperimentID         string   `json:"gcl_experiment_id,omitempty"`
    DispatchFailureRate     *float64 `json:"dispatch_failure_rate,omitempty"`
    FixReworkRate           *float64 `json:"fix_rework_rate,omitempty"`
    PostFixRereviewCoverage *float64 `json:"post_fix_rereview_coverage,omitempty"`
}

// HealingDecision is the return value of pre/post hooks.
type HealingDecision struct {
    Action string `json:"action"`  // proceed | skip | retry | escalate
    Reason string `json:"reason,omitempty"`
}
```

### 5.1 GCL Campaign Process Metrics

A GCL campaign also measures the improvement loop, not only the code it ships.
Three flat rates ride on the outcome row so a later session can falsify
"the GCL worked":

| Field | Rate | Numerator | Denominator |
|---|---|---|---|
| `dispatch_failure_rate` | subagent dispatch reliability | failed dispatches | total dispatches |
| `fix_rework_rate` | first-pass quality | first-round files modified again in a later fix round | first-round write set size |
| `post_fix_rereview_coverage` | gate discipline | fix rounds re-reviewed | fix rounds |

Plus `gcl_experiment_id`, linking the row to
`audit-results/gcl-campaigns/<id>.json` — the raw counts, waivers, and
timestamps that every rate is derived from.

**Critic-signal quality has no rate on purpose.** A single
`critic_false_positive_rate` would merge two opposite signals: a Critic being
wrong, and an Adjudicator wrongly overturning a correct Critic. The raw counts
(`critic.findings`, `critic.acknowledged_false_positive`,
`adjudicator.overturns`) stay in the campaign artifact until their numerators
are unambiguous; the report prints them side by side.

Rates are `*float64`: a measured `0.0` must stay distinguishable from "not
measured" (`omitempty` would drop a plain `0`). Denominators must be `> 0` and
numerators must be `<= denominator` — a bad count is rejected, never clamped,
because a clamped metric cannot be falsified.

```bash
# record: raw counts in, derived rates + campaign artifact + outcome row out
hwcloud-skillcheck learning campaign record --root . --id <experiment-id> \
  --outcome success --metrics campaign-metrics.json [--dry-run]

# read: the falsifiability loop — one command shows the measured history
hwcloud-skillcheck learning campaign report --root . [--json]
```

`post_fix_rereview` is `reviewed | waived`; `waived` is rejected unless **every**
waiver carries `finding_id`, `file_line`, `command`, and `result`
(docs/gcl-spec.md §4.1). The report recomputes rates from raw counts, so an
edited artifact cannot disagree with its own report; an impossible count pair
renders as `-1` rather than hiding the campaigns around it.

**Retention**: campaign artifacts live under gitignored `audit-results/` and
outcome rows are pruned after 90 days. These numbers are inheritable by later
sessions **on the same working copy within 90 days** — a fresh clone, CI, or
another machine starts from zero. Cross-machine continuity needs a deliberate
export, not an implicit promise.

## 6. Interface

```go
// OutcomeMemory is the append-only outcome store.
type OutcomeMemory struct {
    path string // <root>/.l4-memory/outcomes.jsonl
    mu   sync.Mutex
}

func NewOutcomeMemory(root string) (*OutcomeMemory, error)
func (m *OutcomeMemory) Record(r OutcomeRecord) error
func (m *OutcomeMemory) RecentOutcomes(skill, action string, n int) ([]OutcomeRecord, error)
func (m *OutcomeMemory) MatchOutcomes(skill, action, contextHash string, lookback time.Duration) ([]OutcomeRecord, error)
func (m *OutcomeMemory) PruneOlderThan(cutoff time.Time) (int, error)

// Healing hooks are package-level functions so they're easy to unit test.
func PreExecHook(step TaskStep, mem *OutcomeMemory, p HealingPolicy) HealingDecision
func PostFailureHook(step TaskStep, result StepResult, retryCount int, mem *OutcomeMemory, p HealingPolicy) HealingDecision
```

## 7. Out of Scope (explicit)

- Skill-level opt-out via frontmatter (deferred; see ADR-0007 §Open Questions).
- A `hwcloud-skillcheck memory` CLI subcommand.
- SQLite migration.
- Cross-process locking (`flock`).
- Persisting `HealingPolicy` — it remains a runtime value configured by the
  orchestrator, not on disk.