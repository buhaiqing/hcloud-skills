---
title: ADR-0015 — Autonomous remediation (autofix) closed loop
status: Accepted
date: 2026-08-01
decision: Wire failure_patterns.json fix / remediation-playbooks.json into an autonomous fix executor that gates on risk + success_rate, executes the remediation, verifies it, rolls back on failure, and records the outcome back into pattern stats (closed learning loop). This upgrades the L4 "self-evolution" dimension toward Gartner L4.
---

## Context

The repo already *learns* failure patterns. `hwcloud-skillcheck learning trace
aggregate` scans GCL traces and writes each skill's `failure_patterns.json`, with
per-pattern stats (`occurrence_count`, `auto_fixed_count`, `escalated_count`,
`success_rate`) — see `internal/learning/trace.go` (`Aggregate`, `MergePattern`).

But the "act" side was severed (before this ADR):

1. **`remediation-playbooks.json` was dead data.** `learning gen` writes it (via
   `RemediationPlaybook`, `internal/learning/playbook.go`), but nothing read it at
   runtime. Its `execute` / `verification` / `rollback` / `auto_execute_threshold`
   fields were never consumed by any executor.
2. **`matchPreExecutionRisk` dropped the fix.** `internal/l4/orchestrator.go` and
   `internal/l4/execution.go` used it purely as a structural-critic risk probe,
   surfacing `matched_pattern_id` / `risk` / `signature` — but never threaded the
   matched pattern's `fix.action` into an execution path.
3. **`MergePattern` never updated the outcome stats.** It bumped
   `occurrence_count` / `last_seen` but not `auto_fixed_count` / `escalated_count`
   / `success_rate`, so the loop had no way to observe whether a fix actually
   worked. Learning existed; acting and feeding back did not.

This ADR closes that loop: learn → gate → execute → verify → rollback → feed back.

## Decision

### New L4 autofix executor

- **`internal/l4/autofix.go`**: `AutoFix(playbooks []PlaybookSpec, command string, cfg AutofixConfig) AutofixResult`.
  The safety gate chain runs **in order**, each step blocking execution:
  1. `dry-run` — `AutoExecute=false` → no execution, ever.
  2. `destructive-verb HITL` — `ExtractHighRiskVerbs()` matched → `skip_hitl`.
  3. `success_rate < auto_execute_threshold` — each playbook gates on **its own**
     `metadata.success_rate` (learned), compared against its own
     `auto_execute_threshold`; a `0.0` (unlearned) rate blocks that playbook
     (conservative bootstrap).
  4. render placeholders → preconditions → execute → verify.
  5. On execute/verify failure → best-effort **rollback** + record outcome.

  `AutofixResult.Action` reports `execute | skip_threshold | skip_hitl |
  dry_run | rollback`.

- **`internal/l4/PlaybookSpec`** (exported): the minimal playbook shape the
  executor needs, deliberately **decoupled from `internal/learning`** to avoid an
  import cycle. The CLI layer bridges `learning.RemediationPlaybook →
  l4.PlaybookSpec`.

### New learning readers / writers

- **`internal/learning/playbook.go`**: `LoadPlaybooks` — the runtime reader of the
  previously-dead `remediation-playbooks.json`. Missing file / malformed JSON →
  empty slice (never an error), matching `LoadFailurePatterns`.
- **`internal/learning/render.go`**: `RenderOutput` (substitutes `{{output.*}}` /
  `{{env.*}}`; unresolved placeholder → block execution) and `EvalPreconditions`
  (each precondition must exit 0).
- **`internal/learning/playbook.go`**: `RecordPlaybookOutcome` — updates a
  playbook's `metadata.success_rate` (EWMA) after an autonomous fix. A success
  nudges it toward 1.0; a failure de-ranks it so the autofix executor's
  `auto_execute_threshold` eventually blocks a flaky fix. This is the
  **playbook-level closed loop** (the autofix executor executes a playbook, so
  the outcome is attributed to that playbook's own stats). (`RecordFixOutcome`
  in `trace.go` remains the pattern-level aggregator; it is not used by autofix.)

### New CLI + orchestration wiring

- **`cmd/autofix.go`**: `hwcloud-skillcheck l4 autofix --skill <id> --command <cmd>
  [--dry-run]`. It is the *only* package importing both `internal/learning` and
  `internal/l4`, performing the bridge without an import cycle.
- **`RunExecutionLoopWithHealing`** (`internal/l4/execution.go`) accepts a variadic
  `AutofixFunc` hook. When provided, the permanent-failure branch attempts
  autonomous remediation after `PostFailureHook` escalates.

### User-confirmed constraints

- **Automation boundary:** full auto-execute + audit for **non-destructive**
  playbooks; destructive verbs **always** HITL. Complete audit + rollback.
- **Learning loop:** full closed loop — verify after fix, write back
  `auto_fixed_count` / `success_rate`; a failed fix auto-de-ranks the pattern.
- **Placeholder rendering** done now (`{{output.*}}` / `{{env.*}}` +
  preconditions).
- **High-risk-but-non-destructive** (e.g. RDS failover, `risk_level=high`,
  threshold `0.95`) gates on threshold — effectively blocked unless near-certain.

## Consequences

**Positive**
- The L4 "self-evolution" dimension now has a real closed loop:
  learn → gate → execute → verify → rollback → feed back. This moves the repo's
  maturity claim from "L4 framework-ready" (ADR-0012) toward "L4 achieved" for
  non-destructive remediation.
- Previously-dead `remediation-playbooks.json` is now read and acted on.
- Failed fixes de-rank patterns automatically — the system gets *more conservative*
  with experience, not less.

**Negative / costs**
- Still deliberately braked: destructive ops require HITL; unlearned patterns
  (`success_rate 0.0`) are blocked. This is a safety design choice, not a gap —
  consistent with `AGENTS.md` and GCL `SAFETY_FAIL`.
- The `success_rate` gate is conservative by construction; high-threshold
  playbooks may stay blocked until enough successful learning accumulates.

**Safety**
- Full audit via autofix trace block + outcome records.
- `--dry-run` never executes — it only reports the would-be action, threshold, and
  current `success_rate`.

## Alternatives considered

- **Keep `remediation-playbooks.json` as dead data (status quo).** Rejected: it
  preserves the severed "act" side and yields no self-evolution — the loop stays
  learn-only.
- **Auto-execute destructive playbooks.** Rejected: violates the safety design in
  `AGENTS.md` and GCL `SAFETY_FAIL` (zero tolerance for unattended destructive
  mutation of production cloud resources). Destructive verbs must remain HITL.
