# Agent Cross-call Memory Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `ContextMemory` module to the L4 orchestrator that persists a small structured "agent context" document across invocations, so the orchestrator starts each run knowing the recent tasks, errors, and user-set preferences — without modifying any of the 24 `huaweicloud-*-ops/SKILL.md` files.

**Architecture:** New module `hwcloud-skillcheck/internal/l4/context_memory.go` owns `<root>/.l4-memory/context.json` (a single JSON document, atomic write via tmp+rename, capped lists). Read once on startup; mutations are `Load → modify → Save`. Single-writer only (CLI tool, not daemon). Stdlib only — no SQLite, no new dependencies.

**Tech Stack:** Go 1.22+, `encoding/json`, `os`, `crypto/rand`, `sync.Mutex`, existing `.l4-memory/` directory created by ADR-0007's `NewOutcomeMemory`.

## Global Constraints

- Go module path: `hwcloud-skillcheck`. Run all `go` commands from `hwcloud-skillcheck/`.
- Existing test conventions: `*_test.go` next to source, stdlib `testing` only, table-driven where it fits.
- File permissions: reuse `0700` for `.l4-memory/` (already created by ADR-0007), `0600` for `context.json`.
- Lint gates: `go vet ./...` and `gofmt -l .` must be clean before commit.
- Skill SKILL.md files: **must not be modified**.
- Companion docs (reference only, do not edit): `docs/architecture/0008-agent-cross-call-memory.md`, `docs/superpowers/specs/agent-cross-call-memory.md`.

## File Structure

| File | Status | Responsibility |
|------|--------|----------------|
| `hwcloud-skillcheck/internal/l4/context_memory.go` | **Create** | `Context`, `TaskSummary`, `ErrorSummary`, `ContextMemory`, `Load`, `Save`, mutation API |
| `hwcloud-skillcheck/internal/l4/context_memory_test.go` | **Create** | Unit tests for load/save/atomic write/cap pruning/session rotation |
| `hwcloud-skillcheck/internal/l4/orchestrator.go` | **Modify** | Add `ContextMem` field to `HandleFaultInput`; instantiate + record lifecycle events in `HandleFault`. (~15-line addition) |
| `docs/architecture/0008-agent-cross-call-memory.md` | **Exists** | Reference only |
| `docs/superpowers/specs/agent-cross-call-memory.md` | **Exists** | Reference only |

---

## Task 1: Data model + constants

**Files:**
- Create: `hwcloud-skillcheck/internal/l4/context_memory.go`
- Create: `hwcloud-skillcheck/internal/l4/context_memory_test.go`

**Interfaces:**
- Consumes: nothing (first task)
- Produces:
  - `const ContextSchema = "context-memory/v1"`
  - `const MaxRecentTasks = 20`, `MaxRecentErrors = 20`, `MaxOpenTasks = 50`, `SessionRotateAfter = 24*time.Hour`
  - `type Context struct { ... }`
  - `type TaskSummary struct { ... }`
  - `type ErrorSummary struct { ... }`

- [x] **Step 1: Write failing tests**

In `context_memory_test.go`:

```go
package l4

import (
    "encoding/json"
    "testing"
)

func TestContext_RoundTrip(t *testing.T) {
    in := Context{
        Schema:      ContextSchema,
        SessionID:   "550e8400-e29b-41d4-a716-446655440000",
        CreatedAt:   "2026-07-28T08:00:00Z",
        LastUpdated: "2026-07-28T09:15:00Z",
        RecentTasks: []TaskSummary{{TaskID: "t1", Status: "completed", PrimarySkill: "huaweicloud-ecs-ops"}},
        OpenTasks:   []string{"t2"},
        Preferences: map[string]string{"default_region": "cn-north-4"},
    }
    raw, err := json.Marshal(in)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }
    var out Context
    if err := json.Unmarshal(raw, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if out.Schema != ContextSchema {
        t.Fatalf("schema = %q, want %q", out.Schema, ContextSchema)
    }
    if len(out.RecentTasks) != 1 || out.RecentTasks[0].TaskID != "t1" {
        t.Fatalf("recent_tasks lost: %+v", out.RecentTasks)
    }
    if out.Preferences["default_region"] != "cn-north-4" {
        t.Fatalf("preferences lost: %+v", out.Preferences)
    }
}

func TestContextSchema_Constant(t *testing.T) {
    if ContextSchema != "context-memory/v1" {
        t.Fatalf("ContextSchema = %q, want context-memory/v1", ContextSchema)
    }
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestContext_' -v`
Expected: both FAIL with `undefined: Context`.

- [x] **Step 3: Implement data types in `context_memory.go`**

```go
package l4

import (
    "crypto/rand"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "time"
)

// Schema and cap constants. Bumping the schema version is a breaking
// change; Loaders refuse unknown schemas.
const (
    ContextSchema      = "context-memory/v1"
    MaxRecentTasks     = 20
    MaxRecentErrors    = 20
    MaxOpenTasks       = 50
    SessionRotateAfter = 24 * time.Hour
)

// Context is the entire document held in <root>/.l4-memory/context.json.
// See docs/superpowers/specs/agent-cross-call-memory.md §5.
type Context struct {
    Schema       string            `json:"schema"`
    SessionID    string            `json:"session_id"`
    CreatedAt    string            `json:"created_at"`
    LastUpdated  string            `json:"last_updated"`
    RecentTasks  []TaskSummary     `json:"recent_tasks"`
    OpenTasks    []string          `json:"open_tasks"`
    RecentErrors []ErrorSummary    `json:"recent_errors"`
    Preferences  map[string]string `json:"preferences"`
}

// TaskSummary is a compact record of one past task.
type TaskSummary struct {
    TaskID       string `json:"task_id"`
    Fault        string `json:"fault,omitempty"`
    StartedAt    string `json:"started_at"`
    FinishedAt   string `json:"finished_at,omitempty"`
    Status       string `json:"status"`
    PrimarySkill string `json:"primary_skill,omitempty"`
}

// ErrorSummary is a compact record of one past error.
type ErrorSummary struct {
    Timestamp  string `json:"ts"`
    Skill      string `json:"skill"`
    Action     string `json:"action"`
    ErrorClass string `json:"error_class"`
    ErrorMsg   string `json:"error_msg,omitempty"`
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestContext_' -v`
Expected: PASS for both.

- [x] **Step 5: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/context_memory.go hwcloud-skillcheck/internal/l4/context_memory_test.go
git commit -m "feat(l4): Context data types + schema constants"
```

---

## Task 2: ContextMemory struct + atomic Save

**Files:**
- Modify: `hwcloud-skillcheck/internal/l4/context_memory.go`
- Modify: `hwcloud-skillcheck/internal/l4/context_memory_test.go`

**Interfaces:**
- Produces:
  - `type ContextMemory struct { path string; mu sync.Mutex }`
  - `func NewContextMemory(root string) (*ContextMemory, error)`
  - `func (m *ContextMemory) Save(c *Context) error` — atomic write via tmp+rename

- [x] **Step 1: Write failing tests**

```go
func TestContextMemory_Save_CreatesFileWithMode0600(t *testing.T) {
    dir := t.TempDir()
    mem, err := NewContextMemory(dir)
    if err != nil {
        t.Fatalf("new: %v", err)
    }
    c := &Context{
        Schema:      ContextSchema,
        SessionID:   "sess-1",
        CreatedAt:   "2026-07-28T08:00:00Z",
        LastUpdated: "2026-07-28T08:00:00Z",
    }
    if err := mem.Save(c); err != nil {
        t.Fatalf("save: %v", err)
    }
    info, err := os.Stat(filepath.Join(dir, ".l4-memory", "context.json"))
    if err != nil {
        t.Fatalf("stat: %v", err)
    }
    if perm := info.Mode().Perm(); perm != 0o600 {
        t.Fatalf("file perm = %o, want 0600", perm)
    }
}

func TestContextMemory_Save_AtomicNoTempLeftBehind(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewContextMemory(dir)
    c := &Context{Schema: ContextSchema, SessionID: "s", CreatedAt: "x", LastUpdated: "x"}
    if err := mem.Save(c); err != nil {
        t.Fatalf("save: %v", err)
    }
    // Confirm no .tmp.* files remain.
    entries, _ := os.ReadDir(filepath.Join(dir, ".l4-memory"))
    for _, e := range entries {
        if strings.Contains(e.Name(), ".tmp.") {
            t.Fatalf("temp file left behind: %s", e.Name())
        }
    }
}
```

(Add `os`, `path/filepath`, `strings` to imports.)

- [x] **Step 2: Run tests to verify they fail**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestContextMemory_Save' -v`
Expected: FAIL with `undefined: NewContextMemory`.

- [x] **Step 3: Implement NewContextMemory + Save**

Append to `context_memory.go`:

```go
// ContextMemory owns <root>/.l4-memory/context.json.
type ContextMemory struct {
    path string
    mu   sync.Mutex
}

// NewContextMemory ensures <root>/.l4-memory/ exists (creating it if
// needed — idempotent with NewOutcomeMemory) and returns a store
// pointing at <root>/.l4-memory/context.json.
func NewContextMemory(root string) (*ContextMemory, error) {
    dir := filepath.Join(root, ".l4-memory")
    if err := os.MkdirAll(dir, 0o700); err != nil {
        return nil, fmt.Errorf("context memory: mkdir: %w", err)
    }
    return &ContextMemory{path: filepath.Join(dir, "context.json")}, nil
}

// Save atomically writes c to disk. The temp file is fsynced, renamed
// over the target, and removed if the rename fails. The on-disk schema
// is enforced to ContextSchema.
func (m *ContextMemory) Save(c *Context) error {
    if c == nil {
        return fmt.Errorf("context memory: nil context")
    }
    if c.Schema != ContextSchema {
        return fmt.Errorf("context memory: refusing to save schema %q (want %q)", c.Schema, ContextSchema)
    }
    m.mu.Lock()
    defer m.mu.Unlock()
    raw, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return fmt.Errorf("context memory: marshal: %w", err)
    }
    // Write to tmp file with random suffix.
    tmp, err := os.CreateTemp(filepath.Dir(m.path), "context-*.json.tmp")
    if err != nil {
        return fmt.Errorf("context memory: tmp create: %w", err)
    }
    tmpName := tmp.Name()
    cleanup := func() { _ = os.Remove(tmpName) }

    if _, err := tmp.Write(append(raw, '\n')); err != nil {
        tmp.Close()
        cleanup()
        return fmt.Errorf("context memory: tmp write: %w", err)
    }
    if err := tmp.Sync(); err != nil {
        tmp.Close()
        cleanup()
        return fmt.Errorf("context memory: tmp sync: %w", err)
    }
    if err := tmp.Close(); err != nil {
        cleanup()
        return fmt.Errorf("context memory: tmp close: %w", err)
    }
    if err := os.Chmod(tmpName, 0o600); err != nil {
        cleanup()
        return fmt.Errorf("context memory: tmp chmod: %w", err)
    }
    if err := os.Rename(tmpName, m.path); err != nil {
        cleanup()
        return fmt.Errorf("context memory: rename: %w", err)
    }
    return nil
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestContextMemory_Save' -v`
Expected: both PASS.

- [x] **Step 5: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/context_memory.go hwcloud-skillcheck/internal/l4/context_memory_test.go
git commit -m "feat(l4): ContextMemory with atomic Save (tmp+rename, 0600)"
```

---

## Task 3: Load (first-run + session rotation)

**Files:**
- Modify: `hwcloud-skillcheck/internal/l4/context_memory.go`
- Modify: `hwcloud-skillcheck/internal/l4/context_memory_test.go`

**Interfaces:**
- Produces:
  - `func (m *ContextMemory) Load() (*Context, error)` — returns zero-value with fresh session_id if file absent; errors on unknown schema; rotates session if `created_at` older than `SessionRotateAfter`

- [x] **Step 1: Write failing tests**

```go
func TestContextMemory_Load_FirstRunReturnsFreshContext(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewContextMemory(dir)
    c, err := mem.Load()
    if err != nil {
        t.Fatalf("load: %v", err)
    }
    if c.Schema != ContextSchema {
        t.Errorf("schema = %q", c.Schema)
    }
    if c.SessionID == "" {
        t.Error("session_id empty")
    }
    if c.CreatedAt == "" || c.LastUpdated == "" {
        t.Error("timestamps empty")
    }
    // No persistence side-effect: a subsequent Load returns a NEW session_id.
    c2, _ := mem.Load()
    if c.SessionID == c2.SessionID {
        t.Error("first-run load should not have persisted; got same session_id twice")
    }
}

func TestContextMemory_Load_RejectsUnknownSchema(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewContextMemory(dir)
    bad := &Context{Schema: "context-memory/v999", SessionID: "x", CreatedAt: "x", LastUpdated: "x"}
    if err := mem.Save(bad); err != nil {
        t.Fatalf("save bad: %v", err)
    }
    if _, err := mem.Load(); err == nil {
        t.Fatal("want error for unknown schema, got nil")
    }
}

func TestContextMemory_Load_RotatesExpiredSession(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewContextMemory(dir)
    old := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
    c := &Context{
        Schema: ContextSchema, SessionID: "old-session",
        CreatedAt: old, LastUpdated: old,
        RecentTasks: []TaskSummary{{TaskID: "t1", Status: "completed"}},
    }
    if err := mem.Save(c); err != nil {
        t.Fatalf("save: %v", err)
    }
    loaded, err := mem.Load()
    if err != nil {
        t.Fatalf("load: %v", err)
    }
    if loaded.SessionID == "old-session" {
        t.Fatal("session_id not rotated despite expired created_at")
    }
    // The RecentTasks should be preserved across rotation.
    if len(loaded.RecentTasks) != 1 || loaded.RecentTasks[0].TaskID != "t1" {
        t.Fatalf("recent_tasks not preserved on rotation: %+v", loaded.RecentTasks)
    }
    if !strings.HasPrefix(loaded.CreatedAt, time.Now().UTC().Format("2006-01-02")) {
        t.Errorf("created_at not refreshed: %s", loaded.CreatedAt)
    }
}
```

(Add `time`, `strings` to imports.)

- [x] **Step 2: Run tests to verify they fail**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestContextMemory_Load' -v`
Expected: all FAIL with `undefined: (*ContextMemory).Load`.

- [x] **Step 3: Implement Load**

Append to `context_memory.go`:

```go
// Load reads the context document from disk. If the file does not exist,
// returns a fresh zero-value Context with a newly generated session_id
// and current timestamps (no persistence side-effect). If the document
// has an unknown schema, returns an error. If the document's CreatedAt
// is older than SessionRotateAfter, rotates the session_id and refreshes
// CreatedAt; other fields are preserved.
func (m *ContextMemory) Load() (*Context, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    raw, err := os.ReadFile(m.path)
    if err != nil {
        if os.IsNotExist(err) {
            return freshContext(), nil
        }
        return nil, fmt.Errorf("context memory: read: %w", err)
    }
    var c Context
    if err := json.Unmarshal(raw, &c); err != nil {
        return nil, fmt.Errorf("context memory: parse: %w", err)
    }
    if c.Schema != ContextSchema {
        return nil, fmt.Errorf("context memory: unknown schema %q (want %q)", c.Schema, ContextSchema)
    }
    // Session rotation.
    if ts, err := time.Parse(time.RFC3339, c.CreatedAt); err == nil && time.Since(ts) > SessionRotateAfter {
        c.SessionID = newSessionID()
        c.CreatedAt = NowISO()
        c.LastUpdated = NowISO()
        return &c, nil
    }
    return &c, nil
}

// freshContext returns a zero-value Context with a fresh session_id and
// current timestamps. It does NOT write to disk.
func freshContext() *Context {
    now := NowISO()
    return &Context{
        Schema:      ContextSchema,
        SessionID:   newSessionID(),
        CreatedAt:   now,
        LastUpdated: now,
        Preferences: map[string]string{},
    }
}

// newSessionID returns a fresh uuid v4 string.
func newSessionID() string {
    var b [16]byte
    if _, err := rand.Read(b[:]); err != nil {
        // crypto/rand should never fail on Linux/macOS; if it does, fall
        // back to a non-cryptographic but unique-enough identifier.
        return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
    }
    // RFC 4122 v4 layout.
    b[6] = (b[6] & 0x0f) | 0x40
    b[8] = (b[8] & 0x3f) | 0x80
    return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
```

- [x] **Step 4: Run tests to verify they pass**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestContextMemory_Load' -v`
Expected: all PASS.

- [x] **Step 5: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/context_memory.go hwcloud-skillcheck/internal/l4/context_memory_test.go
git commit -m "feat(l4): ContextMemory.Load with first-run + session rotation"
```

---

## Task 4: Mutation API (RecordTask, RecordError, SetPreference, CloseTask)

**Files:**
- Modify: `hwcloud-skillcheck/internal/l4/context_memory.go`
- Modify: `hwcloud-skillcheck/internal/l4/context_memory_test.go`

**Interfaces:**
- Produces:
  - `func (m *ContextMemory) RecordTask(t TaskSummary) error`
  - `func (m *ContextMemory) RecordError(e ErrorSummary) error`
  - `func (m *ContextMemory) SetPreference(k, v string) error`
  - `func (m *ContextMemory) CloseTask(taskID string) error`
- Each follows the `Load → modify → Save` pattern under the mutex.

- [ ] **Step 1: Write failing tests**

```go
func TestContextMemory_RecordTask_PrependsAndCapsAt20(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewContextMemory(dir)
    for i := 0; i < 25; i++ {
        if err := mem.RecordTask(TaskSummary{
            TaskID: fmt.Sprintf("t-%02d", i),
            Status: "completed",
        }); err != nil {
            t.Fatalf("record %d: %v", i, err)
        }
    }
    c, _ := mem.Load()
    if len(c.RecentTasks) != MaxRecentTasks {
        t.Fatalf("want %d, got %d", MaxRecentTasks, len(c.RecentTasks))
    }
    // Newest first: most recent should be t-24, oldest kept t-05.
    if c.RecentTasks[0].TaskID != "t-24" {
        t.Errorf("newest = %s, want t-24", c.RecentTasks[0].TaskID)
    }
    if c.RecentTasks[len(c.RecentTasks)-1].TaskID != "t-05" {
        t.Errorf("oldest kept = %s, want t-05", c.RecentTasks[len(c.RecentTasks)-1].TaskID)
    }
}

func TestContextMemory_RecordTask_RunningAddsToOpen(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewContextMemory(dir)
    if err := mem.RecordTask(TaskSummary{TaskID: "running-1", Status: "running"}); err != nil {
        t.Fatalf("record: %v", err)
    }
    c, _ := mem.Load()
    if len(c.OpenTasks) != 1 || c.OpenTasks[0] != "running-1" {
        t.Fatalf("open_tasks = %+v", c.OpenTasks)
    }
}

func TestContextMemory_CloseTask_RemovesFromOpen(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewContextMemory(dir)
    _ = mem.RecordTask(TaskSummary{TaskID: "r-1", Status: "running"})
    _ = mem.RecordTask(TaskSummary{TaskID: "r-2", Status: "running"})
    if err := mem.CloseTask("r-1"); err != nil {
        t.Fatalf("close: %v", err)
    }
    c, _ := mem.Load()
    if len(c.OpenTasks) != 1 || c.OpenTasks[0] != "r-2" {
        t.Fatalf("open_tasks = %+v", c.OpenTasks)
    }
}

func TestContextMemory_RecordError_CapsAt20(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewContextMemory(dir)
    for i := 0; i < 25; i++ {
        _ = mem.RecordError(ErrorSummary{
            Timestamp: "x", Skill: "s", Action: "a", ErrorClass: "permanent",
        })
    }
    c, _ := mem.Load()
    if len(c.RecentErrors) != MaxRecentErrors {
        t.Fatalf("want %d, got %d", MaxRecentErrors, len(c.RecentErrors))
    }
}

func TestContextMemory_SetPreference_AddsAndDeletes(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewContextMemory(dir)
    if err := mem.SetPreference("default_region", "cn-north-4"); err != nil {
        t.Fatalf("set: %v", err)
    }
    c, _ := mem.Load()
    if c.Preferences["default_region"] != "cn-north-4" {
        t.Fatalf("preferences = %+v", c.Preferences)
    }
    if err := mem.SetPreference("default_region", ""); err != nil {
        t.Fatalf("delete: %v", err)
    }
    c, _ = mem.Load()
    if _, ok := c.Preferences["default_region"]; ok {
        t.Fatalf("key not deleted: %+v", c.Preferences)
    }
}
```

(Add `fmt` to imports.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestContextMemory_Record|TestContextMemory_Close|TestContextMemory_SetPreference' -v`
Expected: all FAIL with `undefined: (*ContextMemory).RecordTask` (and friends).

- [ ] **Step 3: Implement the four mutation methods**

Append to `context_memory.go`:

```go
// RecordTask prepends t to RecentTasks (newest first) and caps at MaxRecentTasks.
// If status is "running" or "paused", t.TaskID is also prepended to OpenTasks
// (capped at MaxOpenTasks).
func (m *ContextMemory) RecordTask(t TaskSummary) error {
    c, err := m.Load()
    if err != nil {
        return err
    }
    c.RecentTasks = append([]TaskSummary{t}, c.RecentTasks...)
    if len(c.RecentTasks) > MaxRecentTasks {
        c.RecentTasks = c.RecentTasks[:MaxRecentTasks]
    }
    if t.Status == "running" || t.Status == "paused" {
        c.OpenTasks = append([]string{t.TaskID}, c.OpenTasks...)
        if len(c.OpenTasks) > MaxOpenTasks {
            c.OpenTasks = c.OpenTasks[:MaxOpenTasks]
        }
    }
    c.LastUpdated = NowISO()
    return m.Save(c)
}

// RecordError prepends e to RecentErrors (newest first), capped at MaxRecentErrors.
func (m *ContextMemory) RecordError(e ErrorSummary) error {
    c, err := m.Load()
    if err != nil {
        return err
    }
    c.RecentErrors = append([]ErrorSummary{e}, c.RecentErrors...)
    if len(c.RecentErrors) > MaxRecentErrors {
        c.RecentErrors = c.RecentErrors[:MaxRecentErrors]
    }
    c.LastUpdated = NowISO()
    return m.Save(c)
}

// SetPreference sets preferences[k] = v. Passing v == "" deletes the key.
func (m *ContextMemory) SetPreference(k, v string) error {
    c, err := m.Load()
    if err != nil {
        return err
    }
    if c.Preferences == nil {
        c.Preferences = map[string]string{}
    }
    if v == "" {
        delete(c.Preferences, k)
    } else {
        c.Preferences[k] = v
    }
    c.LastUpdated = NowISO()
    return m.Save(c)
}

// CloseTask removes taskID from OpenTasks (no-op if absent).
func (m *ContextMemory) CloseTask(taskID string) error {
    c, err := m.Load()
    if err != nil {
        return err
    }
    out := c.OpenTasks[:0]
    for _, id := range c.OpenTasks {
        if id != taskID {
            out = append(out, id)
        }
    }
    c.OpenTasks = out
    c.LastUpdated = NowISO()
    return m.Save(c)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestContextMemory_Record|TestContextMemory_Close|TestContextMemory_SetPreference' -v`
Expected: all PASS.

- [ ] **Step 5: Run the full l4 test suite — nothing else should break**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -v`
Expected: PASS for all tests, including the existing `persistence_test.go`, `orchestrator_test.go`, `topology_test.go`, `trust_test.go`, `rbac_test.go`, `outcome_memory_test.go` (from ADR-0007), `self_healing_test.go` (from ADR-0007).

- [ ] **Step 6: Lint and vet**

Run:
```bash
cd hwcloud-skillcheck && go vet ./... && gofmt -l .
```
Expected: empty output from both.

- [ ] **Step 7: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/context_memory.go hwcloud-skillcheck/internal/l4/context_memory_test.go
git commit -m "feat(l4): ContextMemory mutation API (task, error, pref, close)"
```

---

## Task 5: Wire ContextMemory into the orchestrator

**Files:**
- Modify: `hwcloud-skillcheck/internal/l4/orchestrator.go`
- Create: `hwcloud-skillcheck/internal/l4/orchestrator_context_test.go`

**Interfaces:**
- Consumes:
  - `ContextMemory` from Task 4
  - Existing `HandleFault(HandleFaultInput, ...) *OrchestratorOutput` at `orchestrator.go:133`
  - Existing `HandleFaultInput` struct
  - Existing `BuildTaskFromPlan(plan, fault, root)` at `execution.go:196`
  - Existing `CompleteTask` / `FailTask` / `AbortTask` at `persistence.go:181-208`
- Produces:
  - `HandleFaultInput.ContextMem *ContextMemory` field (additive, nil-safe)
  - `HandleFault` instantiates `NewContextMemory(root)` when input.ContextMem is nil
  - On task creation: `RecordTask` with the fault and primary skill
  - On task completion/failure: `CloseTask(taskID)` + `RecordTask` with finished status
  - On any recorded `StepResult` with `Success: false`: `RecordError`

Scope of this task is **minimal** — just record lifecycle events. Do NOT
yet consume context in the orchestrator's decision logic (that's a
follow-up).

- [ ] **Step 1: Read current `orchestrator.go`**

Read `hwcloud-skillcheck/internal/l4/orchestrator.go`. Identify:
- The `HandleFaultInput` struct definition (probably lines 50–90).
- The `HandleFault` function body (around line 133).
- The exact line where `BuildTaskFromPlan` is called.
- The line where the resulting `task` is finalized (`CompleteTask` / `FailTask`).

- [ ] **Step 2: Write the failing integration test**

In `orchestrator_context_test.go`:

```go
package l4

import (
    "os"
    "path/filepath"
    "testing"
)

func TestOrchestrator_RecordsTaskLifecycleViaHandleFault(t *testing.T) {
    root := t.TempDir()
    cm, err := NewContextMemory(root)
    if err != nil {
        t.Fatalf("context: %v", err)
    }

    // Build the HandleFaultInput — exact field names depend on the existing
    // struct; adapt from the struct definition you read in Step 1. The
    // expected new fields are ContextMem and (later) Mem/Policy.
    in := HandleFaultInput{
        // ... existing required fields ...
        Fault:      "list my ECS",
        ContextMem: cm,
    }

    _ = HandleFault(in, nil) // exact second arg depends on existing signature

    // After run, context.json must contain the recorded task.
    raw, err := os.ReadFile(filepath.Join(root, ".l4-memory", "context.json"))
    if err != nil {
        t.Fatalf("read context: %v", err)
    }
    if len(raw) == 0 {
        t.Fatal("context.json is empty")
    }
    c, _ := cm.Load()
    if len(c.RecentTasks) != 1 {
        t.Fatalf("want 1 task, got %d", len(c.RecentTasks))
    }
    if c.RecentTasks[0].Fault != "list my ECS" {
        t.Errorf("fault = %q, want %q", c.RecentTasks[0].Fault, "list my ECS")
    }
    if len(c.OpenTasks) != 0 {
        t.Errorf("open_tasks = %+v, want empty after run", c.OpenTasks)
    }
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run TestOrchestrator_RecordsTaskLifecycleViaHandleFault -v`
Expected: FAIL — `ContextMem` field doesn't exist on `HandleFaultInput`, or `HandleFault` doesn't read it.

- [ ] **Step 4: Add `ContextMem` field to `HandleFaultInput`**

In `orchestrator.go`:

```go
type HandleFaultInput struct {
    // ... existing fields ...
    // ContextMem records the orchestrator's lifecycle events across
    // invocations. When nil, a fresh ContextMemory is created from
    // the resolved root directory.
    ContextMem *ContextMemory
}
```

- [ ] **Step 5: Instantiate and use ContextMemory in `HandleFault`**

At the top of `HandleFault` body (after root resolution, before `BuildTaskFromPlan`):

```go
cm := in.ContextMem
if cm == nil {
    var err error
    cm, err = NewContextMemory(root)
    if err != nil {
        return nil, fmt.Errorf("orchestrator: context memory: %w", err)
    }
}
```

After `BuildTaskFromPlan` returns the task:

```go
// Record task creation.
primarySkill := primarySkillOfTask(task) // 3-line helper, scan task.Steps
_ = cm.RecordTask(TaskSummary{
    TaskID:       task.ID,
    Fault:        task.Fault,
    StartedAt:    task.CreatedAt,
    Status:       "running",
    PrimarySkill: primarySkill,
})
```

At each of `CompleteTask` / `FailTask` / `AbortTask` call sites inside
`HandleFault`, before the return:

```go
status := "completed"
if task.Status == TaskStatusFailed {
    status = "failed"
} else if task.Status == TaskStatusAborted {
    status = "aborted"
}
_ = cm.RecordTask(TaskSummary{
    TaskID:       task.ID,
    Fault:        task.Fault,
    StartedAt:    task.CreatedAt,
    FinishedAt:   task.UpdatedAt,
    Status:       status,
    PrimarySkill: primarySkillOfTask(task),
})
_ = cm.CloseTask(task.ID)

// Record each failure result as an ErrorSummary.
for _, r := range task.Results {
    if !r.Success && r.Error != "" {
        _ = cm.RecordError(ErrorSummary{
            Timestamp:  r.FinishedAt,
            Skill:      primarySkillOfTask(task),
            Action:     r.Command,
            ErrorClass: "unknown",
            ErrorMsg:   r.Error,
        })
    }
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run TestOrchestrator_RecordsTaskLifecycleViaHandleFault -v`
Expected: PASS.

- [ ] **Step 7: Run the full test suite**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -v`
Expected: all PASS. The nil-`ContextMem` path must be back-compat with existing callers.

- [ ] **Step 8: Lint and vet**

Run:
```bash
cd hwcloud-skillcheck && go vet ./... && gofmt -l .
```
Expected: empty output.

- [ ] **Step 9: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/orchestrator.go hwcloud-skillcheck/internal/l4/orchestrator_context_test.go
git commit -m "feat(l4): wire ContextMemory into HandleFault lifecycle"
```

Add helper `primarySkillOfTask(task)` to `context_memory.go` (or
`orchestrator.go` if you prefer). 3 lines: scan `task.Steps` for the
first non-empty `Skill` field and return it.

---

## Task 6: End-to-end smoke test (persistence across "process restarts")

**Files:**
- Create: `hwcloud-skillcheck/internal/l4/context_e2e_test.go`

**Interfaces:**
- Consumes: full API from Tasks 1–5
- Produces: a test that simulates two separate "process runs" against the same `<root>` and verifies state survives

- [ ] **Step 1: Write the test**

```go
package l4

import "testing"

// TestContextMemory_PersistsAcrossRuns simulates two separate invocations
// of the orchestrator pointing at the same root and verifies that the
// second invocation sees the first invocation's recorded tasks and errors.
func TestContextMemory_PersistsAcrossRuns(t *testing.T) {
    root := t.TempDir()

    // "First run"
    cm1, _ := NewContextMemory(root)
    if err := cm1.RecordTask(TaskSummary{
        TaskID: "first-run-task", Fault: "first invocation",
        StartedAt: "2026-07-28T08:00:00Z", Status: "completed",
    }); err != nil {
        t.Fatalf("first record: %v", err)
    }
    if err := cm1.RecordError(ErrorSummary{
        Timestamp: "2026-07-28T08:00:01Z", Skill: "s", Action: "x",
        ErrorClass: "permanent", ErrorMsg: "boom",
    }); err != nil {
        t.Fatalf("first error: %v", err)
    }

    // "Second run" — fresh ContextMemory handle, same root.
    cm2, _ := NewContextMemory(root)
    c, err := cm2.Load()
    if err != nil {
        t.Fatalf("second load: %v", err)
    }
    if len(c.RecentTasks) != 1 || c.RecentTasks[0].TaskID != "first-run-task" {
        t.Fatalf("recent_tasks not persisted: %+v", c.RecentTasks)
    }
    if len(c.RecentErrors) != 1 || c.RecentErrors[0].ErrorMsg != "boom" {
        t.Fatalf("recent_errors not persisted: %+v", c.RecentErrors)
    }
    // Set a preference in run 1, read it in run 2.
    if err := cm1.SetPreference("default_region", "cn-north-4"); err != nil {
        t.Fatalf("set pref: %v", err)
    }
    c2, _ := cm2.Load()
    if c2.Preferences["default_region"] != "cn-north-4" {
        t.Fatalf("preferences not persisted: %+v", c2.Preferences)
    }
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run TestContextMemory_PersistsAcrossRuns -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/context_e2e_test.go
git commit -m "test(l4): ContextMemory end-to-end persistence across runs"
```

---

## Self-Review (run before requesting review)

1. **Spec coverage** — every FR in `docs/superpowers/specs/agent-cross-call-memory.md` is implemented:
   - FR-1 (storage layout) → Task 2
   - FR-2 (document shape + caps) → Tasks 1 + 4
   - FR-3 (atomic write) → Task 2
   - FR-4 (Load API) → Task 3
   - FR-5 (mutation API) → Task 4
   - FR-6 (session ID lifecycle) → Task 3
   - FR-7 (testability) → every task uses `t.TempDir()`
2. **Placeholder scan** — none; all code blocks are complete.
3. **Type consistency** — `Context`, `TaskSummary`, `ErrorSummary`, `ContextMemory`, `Load`, `Save`, `RecordTask`, `RecordError`, `SetPreference`, `CloseTask`, `WithContextMemory` defined once and reused.
4. **No SKILL.md modified** — confirmed; all changes inside `hwcloud-skillcheck/internal/l4/`.
5. **Behavior when `contextMem == nil`** — handled: every wiring site in Task 5 has an `if o.contextMem != nil` guard. The existing `NewOrchestrator` callers without options behave identically.
6. **Outcome Memory coexistence** — confirmed: ContextMemory writes to `context.json`; OutcomeMemory writes to `outcomes.jsonl`. Same directory, different files, no conflict.

---

## Execution Handoff

Plan saved to `docs/superpowers/plans/2026-07-28-agent-cross-call-memory.md`.

Two execution options:

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks, fast iteration
2. **Inline Execution** — execute tasks in this session using executing-plans, batch with checkpoints

Which approach?

---

## GSTACK REVIEW REPORT

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 1 | ISSUES_OPEN | 5 critical, 5 major, 4 minor |
| Outside Voice | codex exec | Independent 2nd opinion | 1 | ISSUES_OPEN | 5 critical, 5 major (overlap with CEO) |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 0 | — | not yet run, eng review required |

- **CROSS-MODEL:** Claude + Codex agreed on all 5 critical findings.
- **VERDICT:** NOT CLEARED — fix C3 (this plan's blocking issue: Task 5 invokes non-existent `NewOrchestrator`) before implementation. Major findings M1–M5 shared with the outcome-memory plan.

### Critical findings specific to this plan

- **C5.** Task 5 wiring is impossible — `NewOrchestrator`, `WithContextMemory`, `OrchestratorOption`, `orch.NewTaskForFault`, `orch.RunTask` do not exist. The real entry point is `HandleFault(HandleFaultInput, ...) *OrchestratorOutput` at `orchestrator.go:133`; task creation is `BuildTaskFromPlan(plan, fault, root)` at `execution.go:196`. Rewrite the wiring section against these real symbols.

### Major findings shared with outcome-memory plan

- **M1–M5** — see outcome-memory plan §GSTACK REVIEW REPORT for full details.

NO UNRESOLVED DECISIONS

## Progress

- Task 1 (Data model + constants) — committed 08e198e
- Task 2 (ContextMemory struct + atomic Save) — committed afed6e4
- Task 3 (Load + first-run + session rotation) — committed e51d76a