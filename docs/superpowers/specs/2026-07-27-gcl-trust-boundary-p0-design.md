# Spec: GCL Trust Boundary P0 — Critic/Generator Isolation + Confirmation Token

> Status: **DRAFT** — pending user review
> Last updated: 2026-07-27

## Background

`hwcloud-skillcheck` runs Generator → Critic → Orchestrator (GCL) loops
on every required/recommended skill to add an adversarial quality gate
before destructive cloud operations (`delete`, `stop`, IAM/KMS/DDL,
cluster reset). The runtime lives in `hwcloud-skillcheck/internal/gcl/`,
port over from `scripts/gcl_runner.py`. The L4 trust layer
(`internal/l4/`) tracks tasks, RBAC permissions, and confirmation
tokens, but is **not yet wired into the GCL Runner**.

Five P0 trust-boundary gaps were identified by self-review on
2026-07-27:

1. **Critic sees raw user request.** `cfg.Request` is stored verbatim
   in `GCLTrace.Request`; `MaskedFields` lists `request` but
   `PersistTrace` never applies the mask.
2. **No feedback-driven retry.** Critic `Suggestions` are captured but
   never returned to the Generator; the Runner just re-executes the
   same `cfg.Command` until `MAX_ITER`.
3. **Schema validation missing on two external boundaries.** Neither
   `operation_intent` input nor external Critic JSON output is
   validated against a schema before use.
4. **Runtime Safety relies on `operation_intent.safety_class` being
   correctly declared by the caller.** Destructive verbs are not
   auto-detected from the command by Harness.
5. **No confirmation token binding between execution plan and
   operation.** Destructive ops without an explicit confirmation
   proceed unchecked.

This spec defines five interdependent fixes that close those
boundary gaps and establish the trust contract for P1 features
(auto-remediation, multi-agent fan-out).

## Goals

- Eliminate raw `request` text from persisted trace artifacts.
- Make Critic feedback drive Generator retries, with a minimal prompt
  contract (failed-dimension scores + suggestions only).
- Validate JSON boundaries (`operation_intent` in, `critic.scores`
  out) against embedded JSON Schemas.
- Auto-detect destructive verb patterns in Generator commands and
  require an explicit confirmation token before execution.
- Bind the confirmation token to a specific execution plan via an
  in-memory nonce registry with TTL + one-time consumption.

## Non-Goals

- Multi-agent GCL fan-out (L5) — P1.
- Persistent confirmation ledger — out of scope; in-memory only.
- Replacing the existing Python `scripts/gcl_runner.py` shim —
  defer to a P1 cleanup pass once P0 ships.
- Any new `huaweicloud-*-ops/SKILL.md` content.

## In-Scope Files

| Path | Change |
|---|---|
| `hwcloud-skillcheck/internal/gcl/runner.go` | Add `SanitizedRequest`; route retry through `RetryPromptBuilder`; pre-execution RBAC + confirmation gate |
| `hwcloud-skillcheck/internal/gcl/critic.go` | Schema-validate external Critic output JSON |
| `hwcloud-skillcheck/internal/gcl/sanitizer.go` | Add `SanitizeRequest(text) (string, error)` |
| `hwcloud-skillcheck/internal/gcl/confirmation.go` (new) | `ConfirmationRegistry` (nonce issuance, validation, GC) |
| `hwcloud-skillcheck/internal/gcl/retry.go` (new) | `RetryPromptBuilder` interface + `MinimalFeedbackRetry` default |
| `hwcloud-skillcheck/internal/gcl/schemas/operation_intent.schema.json` (new, embedded) | Input schema |
| `hwcloud-skillcheck/internal/gcl/schemas/critic_output.schema.json` (new, embedded) | Output schema |
| `hwcloud-skillcheck/internal/embed/schemas/` | Re-export the two new schemas |
| `hwcloud-skillcheck/internal/l4/rbac.go` | No change — reused as-is |
| `cmd/gcl_run.go` | Add `--confirm-nonce <hex>` flag (optional) |
| `internal/gcl/runner_test.go` | New + extended tests for #1, #2, #5 |
| `internal/gcl/critic_test.go` | New test for schema-invalid critic output |
| `internal/gcl/confirmation_test.go` (new) | Tests for nonce TTL, one-time consumption |
| `docs/gcl-spec.md` | Append §14 (trust-boundary contract) referencing the new files |

## Design

### 1. Critic request leak — `SanitizedRequest` + real masking

**Current state (bug):** `GCLTrace.Request` holds `cfg.Request`
verbatim. `MaskedFields` is a metadata array but `PersistTrace`
does a raw `json.MarshalIndent(trace)`, so the on-disk trace leaks
the raw request to anyone with read access to `audit-results/`.

**Fix:**

```go
// gcl/sanitizer.go
// SanitizeRequest strips resource IDs and credentials from free-form
// request text, returning a version safe to surface to the Critic
// (and to store as SanitizedRequest on the trace).
func SanitizeRequest(text string) (string, error)

// gcl/runner.go
type GCLTrace struct {
    // ... existing fields
    Request          string `json:"request"`                   // kept for back-compat; PersistTrace masks it
    SanitizedRequest string `json:"sanitized_request,omitempty"`
    MaskedFields     []string `json:"masked_fields"`
}

// PersistTrace is taught to actually walk MaskedFields and replace
// the value with "<masked>":
func applyMaskFields(t *GCLTrace) {
    for _, path := range t.MaskedFields {
        // supports "request", "operation_intent", "generator.command",
        // "generator.result_excerpt"
        maskByPath(t, path)
    }
}
```

**Sanitization contract:**

- Resource IDs (regex `(?i)(?:ecs|rds|vpc|dcs|elb|cce|kms|iam)-[a-z0-9-]{8,}`) → `<id>`
- `HW_SECRET_ACCESS_KEY=...`, `SecretAccessKey=...`, `AK=...`, `SK=...` → `<redacted>`
- ARNs `acs:...:<id>` → `<arn>`

The function returns the sanitized string; if any token cannot be
classified, it returns an error and the caller MUST treat that as
`ExitUsage` rather than fall through.

### 2. Critic feedback → Generator retry

**Current state:** the loop in `Run()` computes `decision` and
either exits (`PASS` / `SAFETY_FAIL`) or proceeds to the next
iteration with the **same** `cfg.Command`. Critic `Suggestions`
are recorded on the trace but the Generator never sees them.

**Fix:** Introduce a `RetryPromptBuilder` indirection between Critic
and the next iteration's command.

```go
// gcl/retry.go
type RetryPromptBuilder interface {
    BuildRetryCommand(iter int, prev CriticResult) (string, error)
}

type MinimalFeedbackRetry struct{}

func (MinimalFeedbackRetry) BuildRetryCommand(
    _ int, prev CriticResult,
) (string, error) {
    // Per Q1-C: only failed dimensions + suggestions.
    var lines []string
    lines = append(lines, "# Critic feedback-driven retry")
    lines = append(lines, "## Failed dimensions")
    for dim, threshold := range RUBRIC_THRESHOLDS {
        if s, ok := prev.Scores[dim]; ok && s < threshold {
            lines = append(lines, fmt.Sprintf("- %s = %.2f (threshold %.2f)", dim, s, threshold))
        }
    }
    lines = append(lines, "## Suggestions")
    if len(prev.Suggestions) == 0 {
        lines = append(lines, "(none)")
    } else {
        for _, s := range prev.Suggestions {
            lines = append(lines, "- "+s)
        }
    }
    return strings.Join(lines, "\n"), nil
}
```

The Generator command for iter `n+1` becomes a bundle containing
both the previous masked output excerpt and the retry prompt. The
actual bundling is the caller's responsibility (the LLM prompt
assembler above the GCL Runner); the Runner only yields the string.

### 3. Schema validation on input and Critic output

**Per Q2-B:** validate both boundaries.

**Input schema (`operation_intent`):**

```json
// gcl/schemas/operation_intent.schema.json
{
  "$id": "gcl/operation_intent.schema.json",
  "type": "object",
  "required": ["operation", "expected_state", "safety_class"],
  "properties": {
    "operation":      { "type": "string", "minLength": 1 },
    "expected_state": { "type": "string" },
    "safety_class":   { "enum": ["read-only", "mutating", "destructive"] },
    "resource_scope": {
      "type": "array",
      "items": { "type": "string", "pattern": "^[\\<\\*a-z-]+$" }
    }
  },
  "additionalProperties": true
}
```

**Critic output schema:**

```json
// gcl/schemas/critic_output.schema.json
{
  "$id": "gcl/critic_output.schema.json",
  "type": "object",
  "required": ["scores"],
  "properties": {
    "scores": {
      "type": "object",
      "required": ["correctness","safety","idempotency","traceability","spec_compliance"],
      "properties": {
        "correctness":      { "type": "number", "minimum": 0, "maximum": 1 },
        "safety":           { "type": "number", "enum": [0, 1] },
        "idempotency":      { "type": "number", "minimum": 0, "maximum": 1 },
        "traceability":     { "type": "number", "minimum": 0, "maximum": 1 },
        "spec_compliance":  { "type": "number", "minimum": 0, "maximum": 1 }
      }
    },
    "suggestions": { "type": "array", "items": { "type": "string" } },
    "blocking":    { "type": "boolean" },
    "mode":        { "type": "string" }
  }
}
```

**Integration:**

```go
// runner.go Run() — input boundary
if cfg.OperationIntent != "" {
    intent, err := SanitizeOperationIntent(cfg.OperationIntent)
    if err != nil { return RunResult{ExitCode: ExitUsage} }
    if errs := schema.ValidateFile([]byte(cfg.OperationIntent), opIntentSchema); len(errs) > 0 {
        return RunResult{ExitCode: ExitUsage, Output: strings.Join(errs, "; ")}
    }
    trace.OperationIntent = intent
}

// critic.go ExternalCritic.Score() — output boundary
out, err := cmd.Output()
if err == nil {
    if errs := schema.ValidateFile(out, criticOutputSchema); len(errs) > 0 {
        defaultResult.Mode = "schema-invalid"
        defaultResult.Suggestions = append(defaultResult.Suggestions, errs...)
        return defaultResult
    }
}
```

Process timeout (already present in `runCommand`) is unchanged.

### 4. Harness destructive auto-detection (L4 RBAC reuse)

**Per Q3-C:** zero new regex; reuse L4.

```go
// runner.go Run() — pre-execution gate
func preExecutionGate(cfg RunConfig, intent map[string]any) error {
    // 4a. Reuse L4 RBAC destructive detection
    decision := l4.CheckCommandPermission(cfg.Command, "L4_autonomous", 1.0)
    if decision.Risk == l4.RBACRiskCritical && !decision.Allowed {
        return errRBACBlocked(decision.Reason)
    }
    // 4b. If intent declares destructive, require a confirmation token (#5)
    if safety, _ := intent["safety_class"].(string); safety == "destructive" {
        if cfg.ConfirmationToken == "" {
            return errMissingConfirmation
        }
    }
    return nil
}
```

L4 surfaces two existing regex tables (`HighRiskVerbs`,
`HighRiskCommands`) and an `ImmutableConstraints` list — reusing
them means destructive auto-detection requires **no new logic**,
just one wiring call.

### 5. Confirmation token binding (in-memory nonce registry)

**Per Q4-B:**

```go
// gcl/confirmation.go
type ConfirmationRegistry struct {
    mu     sync.Mutex
    clock  func() time.Time
    ttl    time.Duration  // default 60s
    nonces map[string]nonceEntry
}

type nonceEntry struct {
    planID    string
    skill     string
    cmdHash   string  // sha256(cfg.Command)
    issuedAt  time.Time
    consumed  bool
}

func NewConfirmationRegistry() *ConfirmationRegistry { ... }

// Issue returns a fresh 16-byte hex nonce bound to (skill, cmd).
func (r *ConfirmationRegistry) Issue(skill, cmd string) (planID, nonce string, err error)

// ValidateAndConsume atomically checks the nonce matches (skill, cmd)
// and marks it consumed. Subsequent calls return ErrNonceConsumed.
func (r *ConfirmationRegistry) ValidateAndConsume(nonce, skill, cmd string) error

// PruneExpired is called by the L4 sweep loop (already exists).
func (r *ConfirmationRegistry) PruneExpired()
```

**Issuer flow** (added to `cmd/gcl_run.go`):

```bash
hwcloud-skillcheck gcl run \
  --root . \
  --skill huaweicloud-ecs-ops \
  --command 'hcloud ecs delete-server --id ecs-xxx' \
  --confirm-issue   # prints the nonce; required before destructive ops
```

**Consumer flow:**

```bash
hwcloud-skillcheck gcl run ... --confirm-nonce <hex>
```

The CLI re-uses the **same** `ConfirmationRegistry` instance if
multiple `gcl run` invocations happen in the same process; otherwise
the Harness must persist to a temp file (out of scope for P0 —
caller's responsibility to issue & consume in one process).

## Data Flow

```
Caller
  │
  ▼
[1] gcl.Run(cfg)
      │
      ├── RunConfig.Request ─────► SanitizeRequest ─► trace.SanitizedRequest
      │
      ├── RunConfig.OperationIntent ──► Schema validate ─► Sanitize ─► trace.OperationIntent
      │
      ├── preExecutionGate(cfg)
      │     ├── l4.CheckCommandPermission  (auto-detect destructive verb)
      │     └── if destructive: ConfirmationRegistry.ValidateAndConsume(cfg.ConfirmationToken)
      │
      ▼
[2] loop iter = 1..MaxIter
      │
      ├── runCommand(cfg.Command, timeout=120s)  ─► GeneratorOutput
      │
      ├── critic.Score(gen)  ──► (if ExternalCritic) Schema validate
      │                              └─► CriticResult
      │
      ├── Decide(critic.Scores)  ─► PASS | RETRY | SAFETY_FAIL
      │
      └── on RETRY: RetryPromptBuilder.BuildRetryCommand()  (Q1-C: failed dim + suggestions)
            └─► next iter command bundle  (caller-provided prompt assembler)
      │
      ▼
[3] PersistTrace(trace, root)
      │
      └── applyMaskFields(trace, MaskedFields)  ─► true masking, then json.MarshalIndent
      │
      ▼
   audit-results/gcl-trace-<ts>-<hex8>.json  (mode 0600)
```

## Acceptance Criteria (DoD)

| # | Criteria | Test |
|---|---|---|
| A1 | `cfg.Request = "delete ecs-abc123"` → trace JSON has `"request":"<masked>"` | `TestPersistTraceAppliesMaskedFields` |
| A2 | `trace.SanitizedRequest` carries the masked version (strip of PII/IDs), not the raw | `TestSanitizedRequestNoRawIDs` |
| A3 | Critic feedback injected into iter 2 command: only failed-dim + suggestions | `TestRunRetryInjectsOnlyFailedDimensions` |
| A4 | Iter 2 receives same Critic `Suggestions` from iter 1 (not lost) | `TestRunRetryCarriesSuggestions` |
| A5 | `operation_intent` missing required field → `ExitUsage` with schema errors | `TestSchemaInvalidOperationIntent` |
| A6 | External Critic returns malformed JSON → `Mode=schema-invalid` + RETRY | `TestExternalCriticSchemaInvalid` |
| A7 | `cfg.Command = "hcloud rds delete-instance"` without nonce → `ExitSafety` | `TestRunDestructiveBlockedWithoutNonce` |
| A8 | Same nonce consumed twice → second call `ErrNonceConsumed` | `TestConfirmationRegistryOneTime` |
| A9 | Nonce expired (>60s) → `ErrNonceExpired` | `TestConfirmationRegistryTTL` |
| A10 | Existing tests `TestRun_Timeout`, `TestRun_HasLeak`, `TestMaskedFields` still pass | existing suite |

## Risks & Trade-offs

| Risk | Mitigation |
|---|---|
| Breaking change: `trace.Request` becoming masked breaks downstream consumers that rely on raw text | Keep `Request` field with back-compat semantics; add `SanitizedRequest`; document in `docs/gcl-spec.md §14` |
| `MinimalFeedbackRetry` is too sparse for LLM generator | LLM can ask follow-up; this is the minimal contract (Q1-C choice); expand later if P1 shows it's not enough |
| `ConfirmationRegistry` is in-memory; a crash loses pending nonces | Caller must issue & consume in same process for P0; persistent ledger is a P1 task |
| Schema strictness: real-world `operation_intent` from existing callers has extra fields | `additionalProperties: true` on root level keeps intake non-breaking |
| L4 RBAC has `ImmutableConstraints` that already hard-block `delete-*` even with nonce; the gate looks redundant | Documented; the gate is for `destructive` class without immutable verb match |

## Out of Scope

- Persistent nonce ledger (P1; consider SQLite via `internal/l4/persistence.go` pattern).
- Replacing `scripts/gcl_runner.py` shim.
- Skill-level (`huaweicloud-*-ops/SKILL.md`) updates — defer until GCL confirms adoption.
- New YAML / example-config changes.

## Dependencies

- `github.com/huaweicloud/huaweicloud-sdk-go-v3` — unchanged.
- `internal/l4/rbac.go` — reused, not modified.
- `internal/embed/schemas/` — extended with two new files.

## Changelog

| Version | Date | Change |
|---|---|---|
| 0.1.0 | 2026-07-27 | Initial P0 design — trust boundary close-out |
