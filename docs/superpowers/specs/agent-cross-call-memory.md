# Spec: Agent Cross-call Memory

> Companion to `docs/architecture/0008-agent-cross-call-memory.md`. Implements the
> ADR's "Decision" section in concrete terms.

## 1. Goals

- **G1.** Persist a small structured "agent context" document across
  invocations of `hwcloud-skillcheck`, so the orchestrator starts each
  run with knowledge of recent tasks, errors, and user preferences.
- **G2.** The document MUST survive process exit and be loadable on
  startup in O(1) file reads.
- **G3.** Mutations MUST be atomic — partial writes must never leave the
  document in an unreadable state.
- **G4.** The document MUST be human-inspectable (`cat`, `jq`) for ops
  debugging.

## 2. Non-goals

- **N1.** Do **not** share this document across machines or processes.
  Single-writer only.
- **N2.** Do **not** store the document outside `<root>/.l4-memory/`.
- **N3.** Do **not** edit any of the 24 `huaweicloud-*-ops/SKILL.md` files.
- **N4.** Do **not** introduce a CLI subcommand in this iteration.
  (A future `hwcloud-skillcheck memory` subcommand is a follow-up.)
- **N5.** Do **not** add transactional history. The document holds current
  state; history belongs in audit logs / Outcome Memory.
- **N6.** Do **not** auto-prune by time. Pruning is by count cap only.

## 3. Functional Requirements

### FR-1: Storage layout

- Path: `<root>/.l4-memory/context.json`
- Mode: directory `0700` (created if absent, matches ADR-0007), file
  `0600`.
- Format: single UTF-8 JSON document, one top-level object (see §5).
- Schema version: `"schema": "context-memory/v1"` is mandatory in the
  document. Loaders MUST refuse documents with an unknown schema.

### FR-2: Context document shape

```json
{
  "schema": "context-memory/v1",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2026-07-28T08:00:00Z",
  "last_updated": "2026-07-28T09:15:00Z",
  "recent_tasks": [
    {
      "task_id": "abcd1234ef567890",
      "fault": "list my ECS instances",
      "started_at": "2026-07-28T09:00:00Z",
      "finished_at": "2026-07-28T09:00:05Z",
      "status": "completed",
      "primary_skill": "huaweicloud-ecs-ops"
    }
  ],
  "open_tasks": ["deadbeef00112233"],
  "recent_errors": [
    {
      "ts": "2026-07-28T09:01:00Z",
      "skill": "huaweicloud-iam-ops",
      "action": "create-policy",
      "error_class": "permanent",
      "error_msg": "permission denied"
    }
  ],
  "preferences": {
    "default_region": "cn-north-4",
    "ack_low_risk": "true"
  }
}
```

Caps:
- `recent_tasks`: max 20 entries (oldest dropped on overflow).
- `recent_errors`: max 20 entries (oldest dropped on overflow).
- `open_tasks`: max 50 IDs (oldest dropped on overflow).
- `preferences`: unlimited (it's a flat map; user-set keys).

### FR-3: Atomic write

Every `Save` MUST follow this sequence:

1. Marshal the document to a temp file `<dir>/context.json.tmp.<random>`
   with mode `0600`.
2. `fsync(2)` the temp file (`f.Sync()`).
3. `rename(2)` the temp file over `context.json`.
4. On any error before step 3 succeeds, remove the temp file and leave the
   existing document untouched.

The reader MUST tolerate the document not existing yet (first-ever run)
by returning a zero-value `Context{Schema: "context-memory/v1",
SessionID: <fresh uuid>, ...}`.

### FR-4: Load API

```go
// Load reads the context document. If it does not exist, returns a fresh
// zero-value Context with a newly generated session_id and timestamps.
// If the document exists but has an unknown schema, returns an error.
func (m *ContextMemory) Load() (*Context, error)
```

### FR-5: Mutation API

```go
// RecordTask appends t to recent_tasks (newest first) and updates
// last_updated. If status is "running" or "paused", t.TaskID is also
// added to open_tasks. Caps: recent_tasks <= 20, open_tasks <= 50.
func (m *ContextMemory) RecordTask(t TaskSummary) error

// RecordError appends e to recent_errors (newest first).
func (m *ContextMemory) RecordError(e ErrorSummary) error

// SetPreference sets preferences[k] = v. v may be empty (to delete a key).
func (m *ContextMemory) SetPreference(k, v string) error

// CloseTask removes taskID from open_tasks.
func (m *ContextMemory) CloseTask(taskID string) error
```

Each mutation MUST be: `Load → modify in memory → Save`. Concurrent
mutations from multiple goroutines MUST be serialized by an internal
`sync.Mutex`.

### FR-6: Session ID lifecycle

- A new `session_id` (uuid v4) is generated when:
  - The document does not exist (first run).
  - The document's `created_at` is more than 24 hours old.
- Otherwise, `session_id` is preserved across runs.
- A new `created_at` is stamped only when `session_id` is regenerated;
  `last_updated` is stamped on every `Save`.

### FR-7: Testability

The `ContextMemory` type MUST be unit-testable without touching real disk
in the hot path — use `t.TempDir()` as the root.

## 4. Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-1 | Load latency on startup | < 10 ms p95 for a 50 KB document |
| NFR-2 | Save latency per mutation | < 5 ms p95 |
| NFR-3 | Document size on disk | < 50 KB after 1000 invocations (cap-pruning guarantees this) |
| NFR-4 | Memory footprint in process | < 100 KB per loaded document |
| NFR-5 | Crash safety | A crash mid-Save MUST NOT corrupt the existing document; the temp file MUST be cleaned up on next Load |
| NFR-6 | Determinism | All mutations are deterministic on the loaded state; only `NowISO()` and uuid generation use entropy |

## 5. Data Model

```go
// Context is the entire document held in <root>/.l4-memory/context.json.
type Context struct {
    Schema       string         `json:"schema"`         // always "context-memory/v1"
    SessionID    string         `json:"session_id"`     // uuid v4
    CreatedAt    string         `json:"created_at"`     // ISO-8601 UTC
    LastUpdated  string         `json:"last_updated"`   // ISO-8601 UTC
    RecentTasks  []TaskSummary  `json:"recent_tasks"`   // newest first, cap 20
    OpenTasks    []string       `json:"open_tasks"`     // task IDs, cap 50
    RecentErrors []ErrorSummary `json:"recent_errors"`  // newest first, cap 20
    Preferences  map[string]string `json:"preferences"` // flat map
}

// TaskSummary is a compact record of one past task.
type TaskSummary struct {
    TaskID       string `json:"task_id"`
    Fault        string `json:"fault,omitempty"`
    StartedAt    string `json:"started_at"`
    FinishedAt   string `json:"finished_at,omitempty"`
    Status       string `json:"status"` // running | paused | completed | failed | aborted
    PrimarySkill string `json:"primary_skill,omitempty"`
}

// ErrorSummary is a compact record of one past error.
type ErrorSummary struct {
    Timestamp   string `json:"ts"`
    Skill       string `json:"skill"`
    Action      string `json:"action"`
    ErrorClass  string `json:"error_class"` // transient | permanent | unknown
    ErrorMsg    string `json:"error_msg,omitempty"`
}
```

## 6. Interface

```go
// ContextMemory owns <root>/.l4-memory/context.json.
type ContextMemory struct {
    path string
    mu   sync.Mutex
}

func NewContextMemory(root string) (*ContextMemory, error)
func (m *ContextMemory) Load() (*Context, error)
func (m *ContextMemory) Save(c *Context) error
func (m *ContextMemory) RecordTask(t TaskSummary) error
func (m *ContextMemory) RecordError(e ErrorSummary) error
func (m *ContextMemory) SetPreference(k, v string) error
func (m *ContextMemory) CloseTask(taskID string) error

// Constants for caps and schema name.
const (
    ContextSchema        = "context-memory/v1"
    MaxRecentTasks       = 20
    MaxRecentErrors      = 20
    MaxOpenTasks         = 50
    SessionRotateAfter   = 24 * time.Hour
)
```

## 7. Out of Scope (explicit)

- CLI subcommand to view/edit context.
- Cross-process locking (`flock`).
- Schema migration framework (rejected per ADR-0008 §"Why no schema migration
  framework").
- Multi-machine sync / merge.
- Per-skill preference scoping (all preferences are global).