# ADR-0008: Agent Cross-call Memory

- Status: Proposed
- Date: 2026-07-28
- Deciders: hcloud-skills maintainers
- Supersedes: —
- Related: ADR-0007 (Outcome Memory), ADR-0006 (L4 orchestrator)

## Context

Today every invocation of `hwcloud-skillcheck` starts cold:

- The user prompt is the only state.
- The 24 skills have no memory of prior interactions.
- Trust scoring (`trust.go`) computes trust from a curated `OpHistory`
  populated by an external pipeline, not by the agent itself.
- A user who runs `hwcloud-skillcheck run "list my ECS"` five times in a day
  has their trust history **recomputed from scratch each time** from whatever
  was externally ingested.

The companion ADR-0007 (Outcome Memory) addresses one slice of this gap — the
**outcome log** of every step. This ADR addresses the complementary slice:
**agent state that survives between invocations**.

Concretely the agent needs to remember, across separate `hwcloud-skillcheck`
process runs:

1. The last N user requests and which skill they routed to.
2. Which tasks are still open (created but not completed/aborted).
3. The last few errors the user has seen, with their remediation status.
4. Light user-set preferences (e.g., default region, default project,
   auto-acknowledge `low` risk).
5. A stable session ID for cross-invocation log correlation.

These are not outcomes — they're **state**. They have different access
patterns, different lifetimes, and a different shape.

## Decision

Add a new module `hwcloud-skillcheck/internal/l4/context_memory.go` that
persists a single JSON document at `<root>/.l4-memory/context.json`:

```json
{
  "schema": "context-memory/v1",
  "session_id": "uuid-v4",
  "created_at": "2026-07-28T08:00:00Z",
  "last_updated": "2026-07-28T09:15:00Z",
  "recent_tasks":  [/* up to 20 TaskSummary records, newest first */],
  "open_tasks":    [/* task IDs still running or paused */],
  "recent_errors": [/* up to 20 ErrorSummary records, newest first */],
  "preferences":   { "default_region": "cn-north-4", "ack_low_risk": "true" }
}
```

The orchestrator loads this document on startup, holds it in memory, and
writes it back atomically (tmp file + rename) after any mutation. Writes are
serialized by a `sync.Mutex` — there is exactly one process owning this
file at a time.

### Why a single JSON file (not SQLite, not JSONL, not KV store)

| Storage | Pros | Cons | Verdict |
|---------|------|------|---------|
| Single JSON file | Atomic write is one rename; one read = full state; human-inspectable | Whole-file rewrite on every mutation | **Chosen** |
| JSONL append log | Simple appends | Reading requires parse-all; state coherence requires compaction; harder to enforce schema | Rejected — overlaps with Outcome Memory's role |
| SQLite | Concurrent reads, partial updates | CGO-free driver choice; ~300 LOC for ~10 fields | Rejected — overkill |
| BoltDB / KV store | Concurrent readers | Single-purpose dep for ~10 keys | Rejected — stdlib covers it |

The mutation rate is **low** (a handful of writes per invocation, not per
step). Rewriting a 5–10 KB JSON file on every mutation is cheap and
simple. The read on startup is one `os.ReadFile` + `json.Unmarshal`.

### Why a separate file from Outcome Memory

- **Different access patterns**: Context Memory is read on startup, written
  occasionally. Outcome Memory is appended on every step.
- **Different shape**: Context is a structured document with nested
  arrays and a preferences map. Outcomes are flat records.
- **Different lifetime**: Context can be edited/rewritten freely. Outcomes
  are append-only.
- **Different consumers**: Context is consumed by the orchestrator at boot
  (to surface recent errors, set defaults). Outcomes are consumed by the
  healing hooks during execution.

Mixing them into one file would force every consumer to parse both shapes.

### Why no schema migration framework

The document carries `schema: "context-memory/v1"`. If a future change
needs new fields, it adds them with defaults and bumps the version.
There's no in-place migration because the file is small and rewriting it
on schema change is acceptable. Ponytail principle: no migration
machinery for a 5 KB file that already lives under `.l4-memory/`.

### Why a Mutex, not a file lock

- Only one `hwcloud-skillcheck` process owns the file at a time (it's a
  CLI tool, not a daemon).
- `sync.Mutex` is sufficient and stdlib-only.
- `flock(2)` would be useful if we ran multi-process, but we don't.
  Adding it now is speculative (YAGNI).

## Consequences

### Positive

- **Cross-invocation continuity**: a user re-running the orchestrator sees
  their last few tasks and errors, not a blank slate.
- **Editable state**: preferences can be set via CLI flag
  (`hwcloud-skillcheck memory set-pref default_region cn-north-4`) without
  inventing a new storage layer.
- **Human-inspectable**: `cat .l4-memory/context.json` is enough for ops
  debugging.
- **No new dependencies**: stdlib `encoding/json` + `os` + `sync.Mutex`.
- **Reuses `.l4-memory/`** directory from ADR-0007 — same chmod (0700),
  same prune lifecycle (we don't prune context; it's small).

### Negative

- **Single-writer**: a CLI invocation can't safely share this file with a
  background process. Mitigation: we don't run a background process.
- **Rewrite cost**: a write rewrites the entire document. With ~20 task
  summaries and ~20 error summaries, the file stays well under 50 KB.
  Acceptable.
- **No transactional history**: context shows the current state, not the
  history of edits. This is intentional — history belongs in Outcome
  Memory or audit logs.

### Neutral

- The 24 skill SKILL.md files stay untouched (same constraint as
  ADR-0007).
- A `hwcloud-skillcheck memory` subcommand is now sensible to add (covers
  both context and outcome memory), but it remains out of scope for this
  ADR.

## Alternatives Considered

1. **Reuse Outcome Memory (JSONL) and read last N rows on startup.** Rejected
   because context is not append-only and the shape is different.
2. **Store context in environment variables.** Rejected: env vars don't
   survive process exit, and structured data (lists, nested prefs) doesn't
   belong there.
3. **Store context in each task's persisted TaskState file.** Rejected: the
   24-skill cross-cutting state would have to be reconstructed by reading
   every `.l4-tasks/*.json` file at startup — slow and fragile.
4. **Add a SQLite-backed KV store.** Rejected: see storage table above.

## Rollout

Implement via the companion plan
(`docs/superpowers/plans/2026-07-28-agent-cross-call-memory.md`) in 3 phases:

1. **Storage** (`context_memory.go` + tests)
2. **Mutation API** (append-task, append-error, set-preference with cap-pruning)
3. **Wiring** (load on `NewOrchestrator`; save on every mutation; CLI flag
   to seed `default_region` and `ack_low_risk`)

Each phase is independently testable. After phase 1, the file exists on
disk. After phase 2, mutations work but aren't wired. After phase 3, the
orchestrator consumes it on startup.

## Open Questions

- **Preferences schema**: should the orchestrator consume any preferences
  in this iteration, or just persist them for a future CLI? Defer to
  follow-up ADR if any consumer request surfaces.
- **Cross-machine sharing**: if a user runs the orchestrator on two
  machines pointing at the same `<root>`, last-writer-wins. Acceptable
  for v1; revisit if multi-machine becomes a real use case.