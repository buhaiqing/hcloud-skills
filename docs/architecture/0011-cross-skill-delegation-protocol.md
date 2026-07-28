# ADR-0011: Cross-Skill Delegation Protocol

- Status: Accepted
- Date: 2026-07-28
- Deciders: hcloud-skills maintainers
- Supersedes: —
- Related: ADR-0007 (Outcome Memory), ADR-0008 (Context Memory), ADR-0009 (Trust),
  T-3 (`ExpandMatchedWithDelegates`), `internal/l4/orchestration.go`, `internal/l4/orchestrator.go`

## Context

Each `huaweicloud-*-ops` skill declares `delegates_to:` in frontmatter and documents
cross-product hand-offs in `references/integration.md`. The L4 topology graph already
materializes those edges (`DiscoverDynamicEdges`, `SkillCapabilities.DelegatesTo`).

Until T-3, `HandleFault` discovered transitive skills for the orchestration *report*
(`transitive_skills`) but `BuildExecutionPlan` only received keyword-matched primaries.
Delegates never became executable steps.

We need a single protocol answering:

1. **Who decides** which skills run for a fault?
2. **Sync vs async** — does skill A call skill B, or does the orchestrator schedule both?
3. **Context propagation** — what state flows across skill boundaries?
4. **Error handling** — what happens when a delegated step fails?

## Decision

### Model: orchestrator-mediated synchronous pipeline (not peer invocation)

Cross-skill work is **not** skill-to-skill RPC. The L4 orchestrator owns the plan:

```
MatchFaultSkills(fault)
       │
       ▼
DiscoverTransitiveSkills(primaries)   ← BFS over DelegatesTo
       │
       ▼
ExpandMatchedWithDelegates(...)       ← merge; delegates get confidence 0.35
       │
       ▼
SelectStrategy → BuildExecutionPlan   ← pipeline when delegates present
       │
       ▼
RunExecutionLoopWithHealing           ← one TaskState, sequential/pipeline steps
```

Each plan step is `hcloud <short> diagnose_and_remediate` for one skill. Steps share
one task ID, one OutcomeMemory, one ContextMemory, and one GCL/RBAC gate pass.

**Rejected alternatives:**

| Alternative | Why not |
|-------------|---------|
| Skill A shells out to skill B | Dual GCL contexts; credential/token leakage; hard to audit |
| Async fan-out with join | Premature; current Executor is single-threaded; parallel strategy exists for independent skills only |
| Embedding foreign skill body into Generator prompt | Token blow-up; violates skill SRP |

### Sync semantics

- **Default with delegates**: `pipeline` — domain order
  `monitoring → identity → network → compute → database → backup → cost → audit`.
  Step N depends on step N−1.
- **Independent multi-match, no new delegates**: `parallel` (≤3) or `fan_out_collect` (>3).
  Current executor still walks steps sequentially; strategy is advisory for future
  parallel executors.
- **Single skill**: `sequential`.

Timeouts: per-step 300s in the plan; overall `MaxTotalTimeoutSeconds` is the sum.
Healing retries stay inside one step (same skill/action).

### Context propagation

| Artifact | Scope across skills | Rule |
|----------|---------------------|------|
| `TaskState` / task ID | Shared | One task for the whole multi-skill plan |
| `OutcomeMemory` | Shared | Each step `Record`s with that step's `Skill`/`Action` |
| `ContextMemory` | Shared | Mutations queued; `Flush` once at task-finalize (T-5) |
| Trust lookup | Keyword primary only | `LookupTrust(primarySkillFromMatched, that skill's Action)` gates auto-approve — not pipeline `Steps[0]` after domain reorder |
| GCL dry-run | Per step | Every planned skill (including delegates) is scored before exec |
| `in.Skills` allow-list | Filter | Delegates outside the allow-list are dropped (same as primaries) |

Delegated skills carry `MatchedKeywords: ["delegated"]` and `delegateConfidence = 0.35`
so traces can distinguish keyword hits from graph expansion. Primary confidence is never
overwritten by a delegate edge.

### Error handling

1. **RBAC deny / SAFETY_FAIL** on any step → `FailTask`, stop the pipeline.
2. **GCL non-PASS** → step recorded as skipped; pipeline continues (same as single-skill).
3. **Executor failure** → `PostFailureHook` may retry (transient) or escalate; escalate
   ends the step as failure and advances (healing does not abort sibling skills unless
   `FailTask` was called).
4. **No automatic rollback across skills** — `RollbackPolicy: reverse_order` is recorded
   on the plan for operators; the runtime does not auto-execute reverse steps in P0.
5. **Critical / high-risk playbooks** remain human-gated via trust + risk class before
   the loop starts (`human_review_required`).

### What this is not (out of scope)

- Peer `InvokeSkill(skill, action, args)` API for SKILL.md authors.
- Async queues / webhooks between skills.
- Cross-skill transactional rollback.
- Propagating `{{output.*}}` captures from skill A into skill B's CLI args
  (today every step uses the same abstract `diagnose_and_remediate` action).

Those remain future ADRs if product needs appear.

## Consequences

### Positive

- Topology / `delegates_to` finally drive execution, not just reports (T-3).
- One audit trail, one trust gate, one context flush — easier SecOps review.
- Confidence tagging keeps Critic / traces honest about graph vs keyword matches.

### Negative / follow-ups

- Trust is keyed on the **keyword primary** (`primarySkillFromMatched`), not
  pipeline `Steps[0]`. Per-step trust remains a future option if operators need
  delegate-specific auto-approve.
- Abstract `diagnose_and_remediate` actions do not yet carry real product ops;
  RealExecutor will mostly no-op against live `hcloud` until generators emit
  concrete commands per skill.
- Parallel strategy is not yet parallel at the executor layer.

### Glossary pointer

AGENTS.md §术语表 — "L4 Orchestrator" / "Topology Graph"; this ADR is the
delegation protocol those entries reference.

## Verification

- `TestExpandMatchedWithDelegates_*`
- `TestHandleFault_CrossSkillPlanIncludesDelegates`
- `go test ./internal/l4/...`
