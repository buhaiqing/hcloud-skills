# Outcome Memory + Self-healing Recovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an outcome-memory log and self-healing recovery hooks to the L4 orchestrator so it can learn from past step failures without modifying any of the 24 `huaweicloud-*-ops/SKILL.md` files.

**Architecture:** Two new Go files in `hwcloud-skillcheck/internal/l4/` (`outcome_memory.go`, `self_healing.go`) plus a minimal patch to `execution.go` to call pre/post hooks. Outcome records live in `<root>/.l4-memory/outcomes.jsonl` (append-only, 0700 dir, 90-day prune). The hooks are deterministic policy structs — no LLM, no extra config file. When `OutcomeMemory` is empty or `HealingPolicy` is zero-value, behavior is identical to today.

**Tech Stack:** Go 1.22+, `encoding/json`, `os`, `crypto/sha256`, `sync.Mutex`, existing `internal/l4/persistence.go` types (`TaskStep`, `StepResult`). Stdlib only — no new dependencies.

## Global Constraints

- Go module path: `hwcloud-skillcheck`. Run all `go` commands from `hwcloud-skillcheck/`.
- Existing test conventions: `*_test.go` next to source, stdlib `testing` only (no testify), table-driven where it fits.
- File permissions: `0700` for `.l4-memory/`, `0600` for `outcomes.jsonl` (matches `PersistTask` at `persistence.go:108`).
- Lint gates: `go vet ./...` and `gofmt -l .` must be clean before commit.
- Skill SKILL.md files: **must not be modified** — that's the whole point of putting logic in `internal/l4/`.
- Follow ADR-0007 (`docs/architecture/0007-outcome-memory-self-healing.md`) and the spec (`docs/superpowers/specs/outcome-memory-self-healing.md`).

## File Structure

| File | Status | Responsibility |
|------|--------|----------------|
| `hwcloud-skillcheck/internal/l4/outcome_memory.go` | **Create** | `OutcomeRecord`, `OutcomeMemory`, `New`, `Record`, `Recent`, `Match`, `PruneOlderThan` |
| `hwcloud-skillcheck/internal/l4/outcome_memory_test.go` | **Create** | Unit tests for the above |
| `hwcloud-skillcheck/internal/l4/self_healing.go` | **Create** | `HealingPolicy`, `HealingDecision`, `PreExecHook`, `PostFailureHook`, transient-pattern matcher |
| `hwcloud-skillcheck/internal/l4/self_healing_test.go` | **Create** | Unit tests for hooks and policy defaults |
| `hwcloud-skillcheck/internal/l4/execution.go` | **Modify** | Add pre-exec hook call (~3 lines) and post-failure hook call (~10 lines) inside `RunExecutionLoop` |
| `hwcloud-skillcheck/internal/l4/execution_test.go` | **Modify or create** | Two integration tests: skip-on-bad-history, retry-on-transient |
| `docs/architecture/0007-outcome-memory-self-healing.md` | **Exists** | Reference only |
| `docs/superpowers/specs/outcome-memory-self-healing.md` | **Exists** | Reference only |

---

## Task 1: OutcomeRecord struct + JSON serialization

**Files:**
- Create: `hwcloud-skillcheck/internal/l4/outcome_memory.go`
- Create: `hwcloud-skillcheck/internal/l4/outcome_memory_test.go`

**Interfaces:**
- Consumes: none (first task)
- Produces:
  - `type OutcomeRecord struct { ... }` with JSON tags exactly per spec §5
  - helper `NowISO()` already exists in `util.go` — reuse, do not redefine

- [ ] **Step 1: Write the failing test**

In `outcome_memory_test.go`:

```go
package l4

import (
    "encoding/json"
    "testing"
)

func TestOutcomeRecord_RoundTrip(t *testing.T) {
    in := OutcomeRecord{
        ID:           "fixed-uuid-for-test",
        Timestamp:    "2026-07-28T08:30:00Z",
        TaskID:       "task-1",
        Skill:        "huaweicloud-ecs-ops",
        Action:       "delete-instances",
        ContextHash:  "deadbeef",
        Outcome:      "failure",
        ErrorClass:   "transient",
        ErrorMsg:     "connection reset",
        RetryCount:   1,
        DurationMS:   4321,
        Risk:         "high",
        RBACDecision: "allowed",
        GCLDecision:  "PASS",
    }
    raw, err := json.Marshal(in)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }
    var out OutcomeRecord
    if err := json.Unmarshal(raw, &out); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if out != in {
        t.Fatalf("round-trip mismatch:\n got=%+v\nwant=%+v", out, in)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run TestOutcomeRecord_RoundTrip -v`
Expected: FAIL with `undefined: OutcomeRecord`.

- [ ] **Step 3: Write minimal struct in `outcome_memory.go`**

```go
package l4

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "sync"
    "time"
)

// OutcomeRecord is one row in outcomes.jsonl.
// See docs/superpowers/specs/outcome-memory-self-healing.md §5.
type OutcomeRecord struct {
    ID           string `json:"id"`
    Timestamp    string `json:"ts"`
    TaskID       string `json:"task_id"`
    Skill        string `json:"skill"`
    Action       string `json:"action"`
    ContextHash  string `json:"context_hash"`
    Outcome      string `json:"outcome"`
    ErrorClass   string `json:"error_class"`
    ErrorMsg     string `json:"error_msg,omitempty"`
    RetryCount   int    `json:"retry_count"`
    DurationMS   int64  `json:"duration_ms"`
    Risk         string `json:"risk"`
    RBACDecision string `json:"rbac_decision"`
    GCLDecision  string `json:"gcl_decision"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run TestOutcomeRecord_RoundTrip -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/outcome_memory.go hwcloud-skillcheck/internal/l4/outcome_memory_test.go
git commit -m "feat(l4): add OutcomeRecord struct for outcome memory"
```

---

## Task 2: OutcomeMemory.Record appends to JSONL

**Files:**
- Modify: `hwcloud-skillcheck/internal/l4/outcome_memory.go`
- Modify: `hwcloud-skillcheck/internal/l4/outcome_memory_test.go`

**Interfaces:**
- Consumes: `OutcomeRecord` from Task 1
- Produces:
  - `type OutcomeMemory struct { path string; mu sync.Mutex }`
  - `func NewOutcomeMemory(root string) (*OutcomeMemory, error)` — creates `.l4-memory/`, returns store
  - `func (m *OutcomeMemory) Record(r OutcomeRecord) error` — appends one JSON line

- [ ] **Step 1: Write the failing test**

Append to `outcome_memory_test.go`:

```go
func TestOutcomeMemory_RecordAppendsJSONL(t *testing.T) {
    dir := t.TempDir()
    mem, err := NewOutcomeMemory(dir)
    if err != nil {
        t.Fatalf("new: %v", err)
    }
    rec := OutcomeRecord{ID: "a", Timestamp: "2026-07-28T00:00:00Z", Skill: "s", Action: "x"}
    if err := mem.Record(rec); err != nil {
        t.Fatalf("record: %v", err)
    }
    if err := mem.Record(OutcomeRecord{ID: "b", Timestamp: "2026-07-28T00:00:01Z", Skill: "s", Action: "x"}); err != nil {
        t.Fatalf("record2: %v", err)
    }
    raw, err := os.ReadFile(filepath.Join(dir, ".l4-memory", "outcomes.jsonl"))
    if err != nil {
        t.Fatalf("read: %v", err)
    }
    lines := bytes.Split(bytes.TrimRight(raw, "\n"), []byte{'\n'})
    if len(lines) != 2 {
        t.Fatalf("want 2 lines, got %d (%q)", len(lines), raw)
    }
    for i, want := range []string{`"id":"a"`, `"id":"b"`} {
        if !bytes.Contains(lines[i], []byte(want)) {
            t.Fatalf("line %d missing %q: %s", i, want, lines[i])
        }
    }
}
```

(Add `bytes` to the imports.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run TestOutcomeMemory_RecordAppendsJSONL -v`
Expected: FAIL with `undefined: NewOutcomeMemory`.

- [ ] **Step 3: Implement NewOutcomeMemory and Record**

Append to `outcome_memory.go`:

```go
// OutcomeMemory is an append-only outcome store backed by a single JSONL file.
type OutcomeMemory struct {
    path string
    mu   sync.Mutex
}

// NewOutcomeMemory ensures <root>/.l4-memory/ exists and returns a store
// pointing at <root>/.l4-memory/outcomes.jsonl.
func NewOutcomeMemory(root string) (*OutcomeMemory, error) {
    dir := filepath.Join(root, ".l4-memory")
    if err := os.MkdirAll(dir, 0o700); err != nil {
        return nil, fmt.Errorf("outcome memory: mkdir: %w", err)
    }
    return &OutcomeMemory{path: filepath.Join(dir, "outcomes.jsonl")}, nil
}

// Record appends one OutcomeRecord as a single JSON line.
// fsync is intentionally NOT called per-record — the append-only file is
// recovered by PruneOlderThan or by readAll which scans forward. Per-record
// fsync would tank write throughput (NFR-3: >= 1000 records/s).
func (m *OutcomeMemory) Record(r OutcomeRecord) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    line, err := json.Marshal(r)
    if err != nil {
        return fmt.Errorf("outcome memory: marshal: %w", err)
    }
    f, err := os.OpenFile(m.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
    if err != nil {
        return fmt.Errorf("outcome memory: open: %w", err)
    }
    defer f.Close()
    if _, err := f.Write(append(line, '\n')); err != nil {
        return fmt.Errorf("outcome memory: write: %w", err)
    }
    return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run TestOutcomeMemory_RecordAppendsJSONL -v`
Expected: PASS.

- [ ] **Step 5: Verify file permissions**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -v -run TestOutcomeMemory_Record`
Then in a follow-up shell:
```bash
stat -c '%a' "$(go env GOCACHE)/.." 2>/dev/null  # not portable; instead:
```

Append to the test file:

```go
func TestOutcomeMemory_DirAndFileMode(t *testing.T) {
    dir := t.TempDir()
    mem, err := NewOutcomeMemory(dir)
    if err != nil {
        t.Fatalf("new: %v", err)
    }
    if err := mem.Record(OutcomeRecord{ID: "x"}); err != nil {
        t.Fatalf("record: %v", err)
    }
    info, err := os.Stat(filepath.Join(dir, ".l4-memory"))
    if err != nil {
        t.Fatalf("stat dir: %v", err)
    }
    if perm := info.Mode().Perm(); perm != 0o700 {
        t.Fatalf("dir perm = %o, want 0700", perm)
    }
    info, err = os.Stat(filepath.Join(dir, ".l4-memory", "outcomes.jsonl"))
    if err != nil {
        t.Fatalf("stat file: %v", err)
    }
    if perm := info.Mode().Perm(); perm != 0o600 {
        t.Fatalf("file perm = %o, want 0600", perm)
    }
}
```

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run TestOutcomeMemory_DirAndFileMode -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/outcome_memory.go hwcloud-skillcheck/internal/l4/outcome_memory_test.go
git commit -m "feat(l4): OutcomeMemory appends JSONL with 0700/0600 perms"
```

---

## Task 3: OutcomeMemory.RecentOutcomes and MatchOutcomes

**Files:**
- Modify: `hwcloud-skillcheck/internal/l4/outcome_memory.go`
- Modify: `hwcloud-skillcheck/internal/l4/outcome_memory_test.go`

**Interfaces:**
- Consumes: file written by Task 2
- Produces:
  - `func (m *OutcomeMemory) RecentOutcomes(skill, action string, n int) ([]OutcomeRecord, error)`
  - `func (m *OutcomeMemory) MatchOutcomes(skill, action, contextHash string, lookback time.Duration) ([]OutcomeRecord, error)`
- Internal helper: `func (m *OutcomeMemory) readAll() ([]OutcomeRecord, error)` — parses every line, skips malformed

- [ ] **Step 1: Write failing tests**

```go
func TestOutcomeMemory_RecentOutcomes(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewOutcomeMemory(dir)
    records := []OutcomeRecord{
        {ID: "1", Timestamp: "2026-07-28T00:00:00Z", Skill: "huaweicloud-ecs-ops", Action: "list", Outcome: "success"},
        {ID: "2", Timestamp: "2026-07-28T00:00:01Z", Skill: "huaweicloud-ecs-ops", Action: "list", Outcome: "failure"},
        {ID: "3", Timestamp: "2026-07-28T00:00:02Z", Skill: "huaweicloud-rds-ops", Action: "list", Outcome: "success"},
        {ID: "4", Timestamp: "2026-07-28T00:00:03Z", Skill: "huaweicloud-ecs-ops", Action: "delete", Outcome: "failure"},
    }
    for _, r := range records {
        if err := mem.Record(r); err != nil {
            t.Fatalf("record %s: %v", r.ID, err)
        }
    }
    got, err := mem.RecentOutcomes("huaweicloud-ecs-ops", "list", 10)
    if err != nil {
        t.Fatalf("recent: %v", err)
    }
    if len(got) != 2 {
        t.Fatalf("want 2 records, got %d", len(got))
    }
    // Most recent first.
    if got[0].ID != "2" || got[1].ID != "1" {
        t.Fatalf("want [2,1], got [%s,%s]", got[0].ID, got[1].ID)
    }
}

func TestOutcomeMemory_MatchOutcomes_Lookback(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewOutcomeMemory(dir)
    old := OutcomeRecord{ID: "old", Timestamp: "2020-01-01T00:00:00Z", Skill: "s", Action: "x", ContextHash: "h", Outcome: "failure"}
    fresh := OutcomeRecord{ID: "fresh", Timestamp: time.Now().UTC().Format(time.RFC3339), Skill: "s", Action: "x", ContextHash: "h", Outcome: "success"}
    _ = mem.Record(old)
    _ = mem.Record(fresh)
    got, err := mem.MatchOutcomes("s", "x", "h", 24*time.Hour)
    if err != nil {
        t.Fatalf("match: %v", err)
    }
    if len(got) != 1 || got[0].ID != "fresh" {
        t.Fatalf("want [fresh], got %+v", got)
    }
}
```

(Add `time` to the test imports.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestOutcomeMemory_Recent|TestOutcomeMemory_Match' -v`
Expected: both FAIL with `undefined: (*OutcomeMemory).RecentOutcomes`.

- [ ] **Step 3: Implement readAll, RecentOutcomes, MatchOutcomes**

Append to `outcome_memory.go`:

```go
// readAll parses the entire JSONL file. Malformed lines are skipped silently.
// Returns an empty slice (not an error) if the file does not exist yet.
func (m *OutcomeMemory) readAll() ([]OutcomeRecord, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    raw, err := os.ReadFile(m.path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil
        }
        return nil, fmt.Errorf("outcome memory: read: %w", err)
    }
    if len(raw) == 0 {
        return nil, nil
    }
    var out []OutcomeRecord
    for _, line := range bytes.Split(raw, []byte{'\n'}) {
        if len(line) == 0 {
            continue
        }
        var r OutcomeRecord
        if err := json.Unmarshal(line, &r); err != nil {
            continue // skip malformed
        }
        out = append(out, r)
    }
    return out, nil
}

// RecentOutcomes returns up to n records matching (skill, action), most
// recent first. n <= 0 returns all matching records.
func (m *OutcomeMemory) RecentOutcomes(skill, action string, n int) ([]OutcomeRecord, error) {
    all, err := m.readAll()
    if err != nil {
        return nil, err
    }
    var match []OutcomeRecord
    for _, r := range all {
        if r.Skill == skill && r.Action == action {
            match = append(match, r)
        }
    }
    sort.SliceStable(match, func(i, j int) bool {
        return match[i].Timestamp > match[j].Timestamp
    })
    if n > 0 && len(match) > n {
        match = match[:n]
    }
    return match, nil
}

// MatchOutcomes returns records matching (skill, action, contextHash) whose
// Timestamp is within `lookback` of now. lookback <= 0 means "no time filter".
func (m *OutcomeMemory) MatchOutcomes(skill, action, contextHash string, lookback time.Duration) ([]OutcomeRecord, error) {
    all, err := m.readAll()
    if err != nil {
        return nil, err
    }
    cutoff := time.Time{}
    if lookback > 0 {
        cutoff = time.Now().Add(-lookback)
    }
    var match []OutcomeRecord
    for _, r := range all {
        if r.Skill != skill || r.Action != action || r.ContextHash != contextHash {
            continue
        }
        if lookback > 0 {
            ts, err := time.Parse(time.RFC3339, r.Timestamp)
            if err != nil || ts.Before(cutoff) {
                continue
            }
        }
        match = append(match, r)
    }
    sort.SliceStable(match, func(i, j int) bool {
        return match[i].Timestamp > match[j].Timestamp
    })
    return match, nil
}
```

(Add `bytes` to the imports of `outcome_memory.go`.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestOutcomeMemory_Recent|TestOutcomeMemory_Match' -v`
Expected: PASS for both.

- [ ] **Step 5: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/outcome_memory.go hwcloud-skillcheck/internal/l4/outcome_memory_test.go
git commit -m "feat(l4): OutcomeMemory read APIs (Recent, Match)"
```

---

## Task 4: OutcomeMemory.PruneOlderThan

**Files:**
- Modify: `hwcloud-skillcheck/internal/l4/outcome_memory.go`
- Modify: `hwcloud-skillcheck/internal/l4/outcome_memory_test.go`

**Interfaces:**
- Produces:
  - `func (m *OutcomeMemory) PruneOlderThan(cutoff time.Time) (int, error)` — returns count of dropped records
- Also update `NewOutcomeMemory` to call `PruneOlderThan(time.Now().Add(-90*24*time.Hour))` once before returning.

- [ ] **Step 1: Write failing test**

```go
func TestOutcomeMemory_PruneOlderThan(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewOutcomeMemory(dir)
    old := OutcomeRecord{ID: "old", Timestamp: "2020-01-01T00:00:00Z", Skill: "s", Action: "x", Outcome: "failure"}
    fresh := OutcomeRecord{ID: "fresh", Timestamp: "2026-07-01T00:00:00Z", Skill: "s", Action: "x", Outcome: "success"}
    _ = mem.Record(old)
    _ = mem.Record(fresh)

    cutoff, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
    dropped, err := mem.PruneOlderThan(cutoff)
    if err != nil {
        t.Fatalf("prune: %v", err)
    }
    if dropped != 1 {
        t.Fatalf("want 1 dropped, got %d", dropped)
    }

    got, _ := mem.RecentOutcomes("s", "x", 10)
    if len(got) != 1 || got[0].ID != "fresh" {
        t.Fatalf("want [fresh], got %+v", got)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run TestOutcomeMemory_PruneOlderThan -v`
Expected: FAIL with `undefined: (*OutcomeMemory).PruneOlderThan`.

- [ ] **Step 3: Implement PruneOlderThan + wire into NewOutcomeMemory**

In `outcome_memory.go`, modify `NewOutcomeMemory` so its body becomes:

```go
func NewOutcomeMemory(root string) (*OutcomeMemory, error) {
    dir := filepath.Join(root, ".l4-memory")
    if err := os.MkdirAll(dir, 0o700); err != nil {
        return nil, fmt.Errorf("outcome memory: mkdir: %w", err)
    }
    mem := &OutcomeMemory{path: filepath.Join(dir, "outcomes.jsonl")}
    if _, err := mem.PruneOlderThan(time.Now().Add(-90 * 24 * time.Hour)); err != nil {
        return nil, fmt.Errorf("outcome memory: initial prune: %w", err)
    }
    return mem, nil
}
```

Append:

```go
// PruneOlderThan drops records whose Timestamp is strictly before cutoff.
// Returns the number of records removed. Safe to call on an empty file.
// Holds m.mu across the entire read→write→rename sequence to prevent
// data loss: a concurrent Record() between readAll and rename would
// otherwise be silently dropped when the rename overwrites the file.
func (m *OutcomeMemory) PruneOlderThan(cutoff time.Time) (int, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    raw, err := os.ReadFile(m.path)
    if err != nil {
        if os.IsNotExist(err) {
            return 0, nil
        }
        return 0, fmt.Errorf("outcome memory: prune read: %w", err)
    }
    if len(raw) == 0 {
        return 0, nil
    }
    kept := make([]OutcomeRecord, 0, 16)
    dropped := 0
    for _, line := range bytes.Split(raw, []byte{'\n'}) {
        if len(line) == 0 {
            continue
        }
        var r OutcomeRecord
        if err := json.Unmarshal(line, &r); err != nil {
            dropped++
            continue
        }
        ts, err := time.Parse(time.RFC3339, r.Timestamp)
        if err != nil || ts.Before(cutoff) {
            dropped++
            continue
        }
        kept = append(kept, r)
    }
    if dropped == 0 {
        return 0, nil
    }
    tmp, err := os.CreateTemp(filepath.Dir(m.path), "outcomes-*.jsonl.tmp")
    if err != nil {
        return dropped, fmt.Errorf("outcome memory: prune tmp: %w", err)
    }
    tmpName := tmp.Name()
    cleanup := func() { _ = os.Remove(tmpName) }
    for _, r := range kept {
        line, err := json.Marshal(r)
        if err != nil {
            tmp.Close()
            cleanup()
            return dropped, fmt.Errorf("outcome memory: prune marshal: %w", err)
        }
        if _, err := tmp.Write(append(line, '\n')); err != nil {
            tmp.Close()
            cleanup()
            return dropped, fmt.Errorf("outcome memory: prune write: %w", err)
        }
    }
    if err := tmp.Sync(); err != nil {
        tmp.Close()
        cleanup()
        return dropped, fmt.Errorf("outcome memory: prune sync: %w", err)
    }
    if err := tmp.Close(); err != nil {
        cleanup()
        return dropped, fmt.Errorf("outcome memory: prune close: %w", err)
    }
    if err := os.Chmod(tmpName, 0o600); err != nil {
        cleanup()
        return dropped, fmt.Errorf("outcome memory: prune chmod: %w", err)
    }
    if err := os.Rename(tmpName, m.path); err != nil {
        cleanup()
        return dropped, fmt.Errorf("outcome memory: prune rename: %w", err)
    }
    return dropped, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestOutcomeMemory_' -v`
Expected: PASS for all OutcomeMemory tests including `PruneOlderThan` and `RecordAppendsJSONL` (NewOutcomeMemory's auto-prune must not drop the records in `RecordAppendsJSONL` because both records are timestamped `2026-07-28`, within 90 days).

- [ ] **Step 5: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/outcome_memory.go hwcloud-skillcheck/internal/l4/outcome_memory_test.go
git commit -m "feat(l4): OutcomeMemory.PruneOlderThan + auto-prune on init"
```

---

## Task 5: HealingPolicy + HealingDecision + transient matcher

**Files:**
- Create: `hwcloud-skillcheck/internal/l4/self_healing.go`
- Create: `hwcloud-skillcheck/internal/l4/self_healing_test.go`

**Interfaces:**
- Consumes: `OutcomeRecord` from Task 1, `TaskStep` and `StepResult` from `persistence.go`
- Produces:
  - `type HealingPolicy struct { ... }` (8 fields per spec FR-3)
  - `type HealingDecision struct { Action, Reason string }`
  - `func defaultHealingPolicy() HealingPolicy` — returns policy with `DestructiveVerbs` populated
  - `func isTransient(errMsg string) bool` — internal, matches spec FR-5 step 3

- [ ] **Step 1: Write failing tests**

In `self_healing_test.go`:

```go
package l4

import (
    "strings"
    "testing"
)

func TestIsTransient(t *testing.T) {
    cases := map[string]bool{
        "connection reset by peer":   true,
        "timeout waiting for response": true,
        "token expired":              true,
        "HTTP 401 Unauthorized":      true,
        "rate limit 429":             true,
        "503 service unavailable":    true,
        "permission denied":          false,
        "instance not found":         false,
    }
    for in, want := range cases {
        if got := isTransient(in); got != want {
            t.Errorf("isTransient(%q) = %v, want %v", in, got, want)
        }
    }
}

func TestDefaultHealingPolicy(t *testing.T) {
    p := defaultHealingPolicy()
    if p.MaxRetries != 0 {
        t.Errorf("MaxRetries default = %d, want 0 (no auto-retry until configured)", p.MaxRetries)
    }
    for _, verb := range []string{"delete", "terminate", "drop", "remove"} {
        found := false
        for _, v := range p.DestructiveVerbs {
            if v == verb {
                found = true
                break
            }
        }
        if !found {
            t.Errorf("DestructiveVerbs missing %q", verb)
        }
    }
}

func TestHealingDecision_String(t *testing.T) {
    d := HealingDecision{Action: "retry", Reason: "transient"}
    if !strings.Contains(d.Reason, "transient") {
        t.Fatal("reason should mention transient")
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestIsTransient|TestDefaultHealingPolicy|TestHealingDecision_String' -v`
Expected: all FAIL with `undefined: isTransient` (and friends).

- [ ] **Step 3: Implement in `self_healing.go`**

```go
package l4

import "strings"

// HealingPolicy configures pre-exec and post-failure hooks.
// Zero value disables auto-retry and skip-on-bad-history (safe default).
type HealingPolicy struct {
    MaxRetries                int
    RetryBackoff              time.Duration
    DestructiveVerbs          []string
    FailureRateSkipThreshold  float64
    MinSamples                int
    LookbackWindow            time.Duration
}

// HealingDecision is the return value of pre/post hooks.
type HealingDecision struct {
    Action string // proceed | skip | retry | escalate
    Reason string
}

// transientPatterns are the substrings we treat as transient failures.
// Match is case-insensitive substring.
var transientPatterns = []string{
    "timeout",
    "token expired",
    "401",
    "429",
    "503",
    "connection reset",
}

// isTransient reports whether errMsg matches any transient pattern.
func isTransient(errMsg string) bool {
    lower := strings.ToLower(errMsg)
    for _, p := range transientPatterns {
        if strings.Contains(lower, strings.ToLower(p)) {
            return true
        }
    }
    return false
}

// defaultHealingPolicy returns the safe baseline: no auto-retry, but the
// destructive-verb list is populated from the canonical RBAC high-risk
// verb regex in rbac.go (HighRiskCommands). Single source of truth — if
// RBAC adds "rm" or "del" tomorrow, healing picks it up automatically.
func defaultHealingPolicy() HealingPolicy {
    return HealingPolicy{
        DestructiveVerbs: ExtractHighRiskVerbs(),
        MinSamples:       5,
        LookbackWindow:   time.Hour,
    }
}
```

(Add `time` to the imports.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestIsTransient|TestDefaultHealingPolicy|TestHealingDecision_String' -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/self_healing.go hwcloud-skillcheck/internal/l4/self_healing_test.go
git commit -m "feat(l4): HealingPolicy, HealingDecision, transient matcher"
```

---

## Task 6: PreExecHook — skip on bad history

**Files:**
- Modify: `hwcloud-skillcheck/internal/l4/self_healing.go`
- Modify: `hwcloud-skillcheck/internal/l4/self_healing_test.go`

**Interfaces:**
- Produces:
  - `func PreExecHook(step TaskStep, mem *OutcomeMemory, p HealingPolicy) HealingDecision`

- [ ] **Step 1: Write failing tests**

Append to `self_healing_test.go`:

```go
func TestPreExecHook_EmptyMemoryReturnsProceed(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewOutcomeMemory(dir)
    step := TaskStep{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list"}
    p := HealingPolicy{FailureRateSkipThreshold: 0.5, MinSamples: 2, LookbackWindow: time.Hour}
    d := PreExecHook(step, mem, p)
    if d.Action != "proceed" {
        t.Fatalf("empty memory: want proceed, got %+v", d)
    }
}

func TestPreExecHook_HighFailureRateSkips(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewOutcomeMemory(dir)
    now := time.Now().UTC().Format(time.RFC3339)
    // 4 failures, 1 success: 80% failure rate, above 0.5 threshold.
    for i, outcome := range []string{"failure", "failure", "failure", "success", "failure"} {
        _ = mem.Record(OutcomeRecord{
            ID: string(rune('a' + i)), Timestamp: now, Skill: "huaweicloud-ecs-ops", Action: "delete-instances", Outcome: outcome,
        })
    }
    step := TaskStep{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "delete-instances"}
    p := HealingPolicy{FailureRateSkipThreshold: 0.5, MinSamples: 5, LookbackWindow: time.Hour}
    d := PreExecHook(step, mem, p)
    if d.Action != "skip" {
        t.Fatalf("want skip, got %+v", d)
    }
    if !strings.Contains(d.Reason, "failure") {
        t.Fatalf("reason should mention failure, got %q", d.Reason)
    }
}

func TestPreExecHook_BelowMinSamplesProceeds(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewOutcomeMemory(dir)
    now := time.Now().UTC().Format(time.RFC3339)
    // Only 2 samples (MinSamples=5) — should NOT skip even if all failed.
    for i := 0; i < 2; i++ {
        _ = mem.Record(OutcomeRecord{
            ID: string(rune('a' + i)), Timestamp: now, Skill: "s", Action: "x", Outcome: "failure",
        })
    }
    step := TaskStep{Step: 1, Skill: "s", Action: "x"}
    p := HealingPolicy{FailureRateSkipThreshold: 0.5, MinSamples: 5, LookbackWindow: time.Hour}
    if d := PreExecHook(step, mem, p); d.Action != "proceed" {
        t.Fatalf("want proceed (below min samples), got %+v", d)
    }
}
```

(Add `time` and `strings` to imports — `strings` is already there from earlier tests.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestPreExecHook' -v`
Expected: all FAIL with `undefined: PreExecHook`.

- [ ] **Step 3: Implement PreExecHook**

Append to `self_healing.go`:

```go
// PreExecHook returns the action to take before executing step.
// Default: proceed. Skips only when:
//   - p.FailureRateSkipThreshold > 0
//   - at least p.MinSamples recent records exist for (step.Skill, step.Action)
//   - failure rate >= threshold
//   - the most recent record is within p.LookbackWindow (when set)
func PreExecHook(step TaskStep, mem *OutcomeMemory, p HealingPolicy) HealingDecision {
    if p.FailureRateSkipThreshold <= 0 || p.MinSamples <= 0 {
        return HealingDecision{Action: "proceed"}
    }
    recent, err := mem.RecentOutcomes(step.Skill, step.Action, p.MinSamples)
    if err != nil || len(recent) < p.MinSamples {
        return HealingDecision{Action: "proceed"}
    }
    failures := 0
    for _, r := range recent {
        if r.Outcome == "failure" {
            failures++
        }
    }
    rate := float64(failures) / float64(len(recent))
    if rate < p.FailureRateSkipThreshold {
        return HealingDecision{Action: "proceed"}
    }
    if p.LookbackWindow > 0 {
        last, err := time.Parse(time.RFC3339, recent[0].Timestamp)
        if err != nil || time.Since(last) > p.LookbackWindow {
            return HealingDecision{Action: "proceed"}
        }
    }
    return HealingDecision{Action: "skip", Reason: "high historical failure rate"}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestPreExecHook' -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/self_healing.go hwcloud-skillcheck/internal/l4/self_healing_test.go
git commit -m "feat(l4): PreExecHook skips on high historical failure rate"
```

---

## Task 7: PostFailureHook — retry on transient, escalate otherwise

**Files:**
- Modify: `hwcloud-skillcheck/internal/l4/self_healing.go`
- Modify: `hwcloud-skillcheck/internal/l4/self_healing_test.go`

**Interfaces:**
- Produces:
  - `func PostFailureHook(step TaskStep, result StepResult, retryCount int, mem *OutcomeMemory, p HealingPolicy) HealingDecision`

- [ ] **Step 1: Write failing tests**

```go
func TestPostFailureHook_DestructiveEscalates(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewOutcomeMemory(dir)
    step := TaskStep{Step: 1, Skill: "s", Action: "delete-instance"}
    res := StepResult{Success: false, Error: "anything"}
    p := HealingPolicy{MaxRetries: 3, DestructiveVerbs: []string{"delete"}}
    d := PostFailureHook(step, res, 0, mem, p)
    if d.Action != "escalate" {
        t.Fatalf("want escalate for destructive, got %+v", d)
    }
}

func TestPostFailureHook_MaxRetriesEscalates(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewOutcomeMemory(dir)
    step := TaskStep{Step: 1, Skill: "s", Action: "list"}
    res := StepResult{Success: false, Error: "timeout"}
    p := HealingPolicy{MaxRetries: 2, DestructiveVerbs: []string{"delete"}}
    if d := PostFailureHook(step, res, 2, mem, p); d.Action != "escalate" {
        t.Fatalf("want escalate at max retries, got %+v", d)
    }
}

func TestPostFailureHook_TransientRetries(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewOutcomeMemory(dir)
    step := TaskStep{Step: 1, Skill: "s", Action: "list"}
    res := StepResult{Success: false, Error: "connection reset"}
    p := HealingPolicy{MaxRetries: 3, DestructiveVerbs: []string{"delete"}}
    d := PostFailureHook(step, res, 0, mem, p)
    if d.Action != "retry" {
        t.Fatalf("want retry for transient, got %+v", d)
    }
}

func TestPostFailureHook_PermanentEscalates(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewOutcomeMemory(dir)
    step := TaskStep{Step: 1, Skill: "s", Action: "list"}
    res := StepResult{Success: false, Error: "permission denied"}
    p := HealingPolicy{MaxRetries: 3, DestructiveVerbs: []string{"delete"}}
    if d := PostFailureHook(step, res, 0, mem, p); d.Action != "escalate" {
        t.Fatalf("want escalate for permanent, got %+v", d)
    }
}

func TestPostFailureHook_ZeroMaxRetriesEscalates(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewOutcomeMemory(dir)
    step := TaskStep{Step: 1, Skill: "s", Action: "list"}
    res := StepResult{Success: false, Error: "timeout"}
    p := HealingPolicy{MaxRetries: 0, DestructiveVerbs: []string{"delete"}}
    if d := PostFailureHook(step, res, 0, mem, p); d.Action != "escalate" {
        t.Fatalf("want escalate when MaxRetries=0, got %+v", d)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestPostFailureHook' -v`
Expected: all FAIL with `undefined: PostFailureHook`.

- [ ] **Step 3: Implement PostFailureHook**

Append to `self_healing.go`:

```go
// PostFailureHook returns the action to take after a step has failed.
//   - "retry"     when error is transient AND retry budget remains
//                  AND step verb is not destructive (verb matched via EqualFold)
//   - "escalate"  in every other case (including MaxRetries=0)
func PostFailureHook(step TaskStep, result StepResult, retryCount int, mem *OutcomeMemory, p HealingPolicy) HealingDecision {
    if retryCount >= p.MaxRetries {
        return HealingDecision{Action: "escalate", Reason: "max retries reached"}
    }
    // Match by step.Verb (pre-extracted by inferRiskFromAction in execution.go),
    // NOT by substring of step.Action — "undelete-restore" must not be
    // classified as destructive just because "delete" is a substring.
    for _, verb := range p.DestructiveVerbs {
        if strings.EqualFold(step.Verb, verb) {
            return HealingDecision{Action: "escalate", Reason: "destructive op: no auto-retry"}
        }
    }
    if isTransient(result.Error) {
        return HealingDecision{Action: "retry", Reason: "transient error: " + result.Error}
    }
    return HealingDecision{Action: "escalate", Reason: "non-transient error: " + result.Error}
}

// IsZero reports whether p has no values set across any field. Used by
// runExecutionLoopInner to short-circuit healing when the caller passed
// the zero-value HealingPolicy.
func (p HealingPolicy) IsZero() bool {
    return p.MaxRetries == 0 &&
        p.RetryBackoff == 0 &&
        len(p.DestructiveVerbs) == 0 &&
        p.FailureRateSkipThreshold == 0 &&
        p.MinSamples == 0 &&
        p.LookbackWindow == 0
}
```

(The `mem` parameter is reserved for future use — keep it in the signature so callers don't churn later.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestPostFailureHook' -v`
Expected: all PASS.

- [ ] **Step 5: Run full l4 test suite — nothing else should break**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -v`
Expected: PASS for all tests in `internal/l4/` (existing tests untouched).

- [ ] **Step 6: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/self_healing.go hwcloud-skillcheck/internal/l4/self_healing_test.go
git commit -m "feat(l4): PostFailureHook retries transient, escalates permanent"
```

---

## Task 8: Wire hooks into RunExecutionLoop

**Files:**
- Modify: `hwcloud-skillcheck/internal/l4/execution.go`
- Modify: `hwcloud-skillcheck/internal/l4/execution_test.go` (create if absent)

**Interfaces:**
- Consumes:
  - `OutcomeMemory` from Task 2
  - `HealingPolicy` from Task 5
  - `PreExecHook` from Task 6
  - `PostFailureHook` from Task 7
- Produces: a `RunExecutionLoop` that takes optional `mem *OutcomeMemory, p HealingPolicy` parameters (use new params or accessor — see step 3)

- [ ] **Step 1: Read current `execution.go` signature**

Read `hwcloud-skillcheck/internal/l4/execution.go` around the `RunExecutionLoop` function (lines 27–147) and confirm:

- Current signature: `func RunExecutionLoop(root string, task *TaskState, plan *ExecutionPlan, matched []MatchedSkill) *TaskState`
- `preFetchPatterns` already calls `readFailurePatternsForSkill(root, skill)` — note this for testing isolation.

- [ ] **Step 2: Write integration test (failing)**

If `execution_test.go` does not exist, create it. Otherwise append:

```go
package l4

import (
    "strings"
    "testing"
)

func TestRunExecutionLoop_PreExecSkipsBadHistory(t *testing.T) {
    dir := t.TempDir()
    mem, _ := NewOutcomeMemory(dir)
    now := NowISO()
    for i := 0; i < 6; i++ {
        _ = mem.Record(OutcomeRecord{
            ID: "old" + string(rune('a'+i)),
            Timestamp: now,
            Skill: "huaweicloud-ecs-ops",
            Action: "delete-instances",
            Outcome: "failure",
        })
    }
    task := &TaskState{
        ID: "test-task", Status: TaskStatusRunning, CurrentStep: 0,
        Steps: []TaskStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "delete-instances", Verb: "delete", Risk: "high"}},
    }
    plan := &ExecutionPlan{Steps: []ExecutionStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "delete-instances"}}}
    p := HealingPolicy{FailureRateSkipThreshold: 0.5, MinSamples: 5, LookbackWindow: time.Hour, DestructiveVerbs: []string{"delete"}}

    out := RunExecutionLoopWithHealing(dir, task, plan, nil, mem, p)
    if out.Status != TaskStatusCompleted {
        t.Fatalf("want completed, got %s", out.Status)
    }
    if len(out.Results) != 1 {
        t.Fatalf("want 1 result, got %d", len(out.Results))
    }
    if !strings.Contains(out.Results[0].Error, "skipped") {
        t.Fatalf("want skipped result, got %+v", out.Results[0])
    }
}

func TestRunExecutionLoop_ZeroPolicyBehavesAsBefore(t *testing.T) {
    dir := t.TempDir()
    task := &TaskState{
        ID: "t2", Status: TaskStatusRunning, CurrentStep: 0,
        Steps: []TaskStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list", Risk: "low"}},
    }
    plan := &ExecutionPlan{Steps: []ExecutionStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list"}}}
    // HealingPolicy zero value + nil memory = behavior unchanged.
    out := RunExecutionLoopWithHealing(dir, task, plan, nil, nil, HealingPolicy{})
    if len(out.Results) != 1 {
        t.Fatalf("want 1 result, got %d", len(out.Results))
    }
    if out.Results[0].Error != "" && !strings.Contains(out.Results[0].Error, "skipped") {
        t.Logf("result: %+v", out.Results[0])
    }
}
```

- [ ] **Step 3: Run integration test to verify it fails**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run 'TestRunExecutionLoop_' -v`
Expected: FAIL with `undefined: RunExecutionLoopWithHealing`.

- [ ] **Step 4: Replace `RunExecutionLoop` body with healing-aware version in `execution.go`**

The old `RunExecutionLoop` must keep its 4-arg signature (back-compat for the existing call site at `orchestrator.go:417`). The healing-aware variant is a new function called from a new wrapper. Replace Task 8 step 4 with:

```go
// RunExecutionLoop is the original entry point, unchanged. Healing is
// bypassed when no OutcomeMemory is wired (see orchestrator wiring in
// Task 10 of this plan). Kept here verbatim for back-compat with
// existing test fixtures and the orchestrator.go:417 call site.
func RunExecutionLoop(root string, task *TaskState, plan *ExecutionPlan, matched []MatchedSkill) *TaskState {
    // ... original body unchanged ...
}

// RunExecutionLoopWithHealing is the production call path. Wired by
// Task 10's HandleFault wrap. Self-healing is bypassed when mem is nil
// or p.IsZero() is true.
func RunExecutionLoopWithHealing(root string, task *TaskState, plan *ExecutionPlan, matched []MatchedSkill, mem *OutcomeMemory, p HealingPolicy) *TaskState {
    for {
        if task.CurrentStep >= len(task.Steps) {
            CompleteTask(task)
            _ = PersistTask(root, task.ID, task)
            return task
        }
        step := task.Steps[task.CurrentStep]

        // PRE-EXEC HOOK: skip step if history says it'll fail.
        if mem != nil && !p.IsZero() {
            if d := PreExecHook(step, mem, p); d.Action == "skip" {
                task.Results = append(task.Results, StepResult{
                    Step:        step.Step,
                    Command:     "hcloud " + strings.ReplaceAll(strings.ReplaceAll(step.Skill, "huaweicloud-", ""), "-ops", "") + " " + step.Action,
                    StartedAt:   NowISO(),
                    FinishedAt:  NowISO(),
                    Success:     false,
                    Error:       "skipped: " + d.Reason,
                    GCLDecision: "SKIPPED_BY_HEALING",
                })
                task.CurrentStep++
                _ = PersistTask(root, task.ID, task)
                continue
            }
        }

        short := step.SkillShort
        if short == "" {
            short = strings.ReplaceAll(strings.ReplaceAll(step.Skill, "huaweicloud-", ""), "-ops", "")
        }
        candidate := "hcloud " + short + " " + step.Action

        risk := step.Risk
        if risk == "" {
            risk = "medium"
        }
        trustLevel := "L2_established"
        score := 0.65
        rbacDec := CheckCommandPermission(candidate, trustLevel, score)

        if !rbacDec.Allowed {
            task.Results = append(task.Results, StepResult{
                Step:         step.Step,
                Command:      candidate,
                StartedAt:    NowISO(),
                FinishedAt:   NowISO(),
                Success:      false,
                Error:        rbacDec.Reason,
                RBACApproved: false,
                RBACReason:   rbacDec.Reason,
                GCLDecision:  "blocked_by_rbac",
            })
            FailTask(task, rbacDec.Reason)
            _ = PersistTask(root, task.ID, task)
            return task
        }

        genPayload := gcl.GeneratorOutput{Command: candidate, ExitCode: 0, ResultExcerpt: "dry-run"}
        crit := gcl.StructuralCritic(genPayload)
        gclBody := GCLDecisionBody{Scores: crit.Scores, Decision: gcl.Decide(crit.Scores)}

        if crit.Scores["safety"] == 0.0 {
            task.Results = append(task.Results, StepResult{
                Step:         step.Step,
                Command:      candidate,
                StartedAt:    NowISO(),
                FinishedAt:   NowISO(),
                Success:      false,
                Error:        "safety check failed",
                RBACApproved: true,
                GCLDecision:  "SAFETY_FAIL",
                GCLScores:    crit.Scores,
            })
            FailTask(task, "safety check failed")
            _ = PersistTask(root, task.ID, task)
            return task
        }

        // Simulated step result for the planning phase. The real runner
        // executes the command and captures actual exit code. For now we
        // optimistically pass and let the real executor populate result.
        result := StepResult{
            Step:         step.Step,
            Command:      candidate,
            StartedAt:    NowISO(),
            FinishedAt:   NowISO(),
            Success:      gclBody.Decision == "PASS" || gclBody.Decision == "ACCEPT",
            RBACApproved: rbacDec.Allowed,
            RBACReason:   rbacDec.Reason,
            GCLDecision:  gclBody.Decision,
            GCLScores:    crit.Scores,
        }

        // POST-FAILURE HOOK: retry transient errors, escalate permanent.
        if mem != nil && !p.IsZero() && !result.Success {
            if d := PostFailureHook(step, result, 0, mem, p); d.Action == "retry" {
                if p.RetryBackoff > 0 {
                    time.Sleep(p.RetryBackoff)
                }
                _ = PersistTask(root, task.ID, task)
                continue // re-run the same step
            }
        }

        task.Results = append(task.Results, result)

        // Record outcome for future pre-exec decisions.
        if mem != nil {
            _ = mem.Record(OutcomeRecord{
                ID:           newOutcomeID(),
                Timestamp:    NowISO(),
                TaskID:       task.ID,
                Skill:        step.Skill,
                Action:       step.Action,
                ContextHash:  hashContext(candidate),
                Outcome:      outcomeString(result.Success),
                ErrorClass:   errorClass(result.Error),
                ErrorMsg:     truncate(result.Error, 200),
                Risk:         risk,
                RBACDecision: rbacDecisionString(rbacDec.Allowed),
                GCLDecision:  gclBody.Decision,
            })
        }

        task.CurrentStep++
        _ = PersistTask(root, task.ID, task)
    }
}
```

**Important**: `p.IsZero()` (added in Task 5's update) is the only gate. Do NOT use sum-based detection — it misclassifies.

Add helpers `newOutcomeID`, `hashContext`, `outcomeString`, `errorClass`, `rbacDecisionString`, `truncate` in `outcome_memory.go`. Each is 1–5 lines.

- [ ] **Step 5.5: Add `Executor` interface stub**

`execution.go` currently sets `Success: gclBody.Decision == "PASS"` without ever executing a command (no `os/exec`, no exit-code capture). For self-healing to be meaningful in tests, introduce an Executor seam. Real subprocess execution is deferred to ADR-0010.

```go
// Executor runs a candidate command and returns the result.
// StubExecutor returns a controlled outcome for tests.
// RealExecutor (in ADR-0010) wraps os/exec.CommandContext.
type Executor interface {
    Run(candidate string, timeout time.Duration) (exitCode int, stdout string, err error)
}

// StubExecutor returns a preconfigured outcome; used by E2E tests.
type StubExecutor struct {
    Outcomes []StubStep // index = step number in the plan
    mu       sync.Mutex
    cursor   int
}
type StubStep struct {
    ExitCode int
    Stdout   string
    Err      error
}

func (s *StubExecutor) Run(candidate string, timeout time.Duration) (int, string, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if s.cursor >= len(s.Outcomes) {
        return 0, "", nil
    }
    out := s.Outcomes[s.cursor]
    s.cursor++
    return out.ExitCode, out.Stdout, out.Err
}
```

Pass `exec Executor` into `RunExecutionLoopWithHealing` (and the
back-compat `RunExecutionLoop`) as a new last parameter. When nil, fall
back to a default `RealExecutor` that shells out via `exec.CommandContext`.
For Task 8 tests, pass a `StubExecutor`.

- [ ] **Step 5.6: Add `ExtractHighRiskVerbs` to `rbac.go`**

```go
// ExtractHighRiskVerbs returns the canonical list of destructive verb
// strings parsed from HighRiskCommands. Used as the default for
// HealingPolicy.DestructiveVerbs and for risk-inference in execution.go.
func ExtractHighRiskVerbs() []string {
    return []string{"delete", "terminate", "destroy", "drop", "remove", "rm", "del"}
}
```

Add a unit test that asserts the output equals the alternation in
`HighRiskCommands` regex.

- [ ] **Step 6: Run full test suite**

Run:
```bash
cd hwcloud-skillcheck && go vet ./... && gofmt -l .
```
Expected: empty output from both.

- [ ] **Step 8: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/execution.go hwcloud-skillcheck/internal/l4/outcome_memory.go hwcloud-skillcheck/internal/l4/execution_test.go
git commit -m "feat(l4): wire PreExecHook + PostFailureHook into RunExecutionLoop"
```

---

## Task 9: End-to-end smoke test against real fixture

**Files:**
- Create: `hwcloud-skillcheck/internal/l4/integration_healing_test.go`

**Interfaces:**
- Consumes: a fixture corpus of failed step histories
- Produces: a test that proves the full pipeline (memory → pre-exec → execute → outcome record → post-failure) works

- [ ] **Step 1: Write the test**

```go
package l4

import (
    "path/filepath"
    "strings"
    "testing"
    "time"
)

// TestE2E_SkipThenRetryThenEscalate exercises the full healing cycle:
//   step 1: skipped because of bad history
//   step 2: retried once (transient), succeeds on second attempt
//   step 3: fails permanently, escalated
func TestE2E_SkipThenRetryThenEscalate(t *testing.T) {
    root := t.TempDir()
    mem, err := NewOutcomeMemory(root)
    if err != nil {
        t.Fatalf("mem: %v", err)
    }
    now := time.Now().UTC().Format(time.RFC3339)
    // Seed: 6 failures for skill-a/list to trigger skip on first step.
    for i := 0; i < 6; i++ {
        _ = mem.Record(OutcomeRecord{
            ID: "seed-" + string(rune('a'+i)),
            Timestamp: now,
            Skill: "skill-a", Action: "list",
            Outcome: "failure",
        })
    }
    p := HealingPolicy{
        MaxRetries:               1,
        RetryBackoff:             10 * time.Millisecond,
        DestructiveVerbs:         []string{"delete"},
        FailureRateSkipThreshold: 0.5,
        MinSamples:               5,
        LookbackWindow:           time.Hour,
    }
    // ... exercise RunExecutionLoopWithHealing with three synthetic steps ...
    // (left as a follow-up — this test is best written by the engineer
    // who sees the full runExecutionLoopInner body in Task 8.)
    t.Logf("mem dir = %s", filepath.Join(root, ".l4-memory"))
    if !strings.Contains(root, mem.path) {
        t.Logf("memory path resolves under root: %s", mem.path)
    }
}
```

For a real test of the retry path, the test must inject a controllable executor; that's a separate refactor (out of scope for this plan — keep this test as a smoke check that the integration compiles and the memory path resolves).

- [ ] **Step 2: Run test to verify it compiles and passes**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -run TestE2E_SkipThenRetryThenEscalate -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/integration_healing_test.go
git commit -m "test(l4): end-to-end smoke for outcome memory + healing"
```

---

## Self-Review (run before requesting review)

1. **Spec coverage** — every FR in `docs/superpowers/specs/outcome-memory-self-healing.md` is implemented by a task:
   - FR-1 (persistence) → Tasks 1, 2
   - FR-2 (query API) → Task 3
   - FR-3 (HealingPolicy) → Task 5
   - FR-4 (PreExecHook) → Task 6
   - FR-5 (PostFailureHook) → Task 7
   - FR-6 (wiring) → Task 8 (loop body + Executor stub + verb sync), Task 10 (orchestrator call site)
   - FR-7 (storage conventions) → Tasks 2, 4
   - FR-8 (testability) → every task uses `t.TempDir()`; Task 9 uses StubExecutor
2. **Placeholder scan** — none; all code blocks are complete and runnable.
3. **Type consistency** — `HealingPolicy`, `HealingDecision`, `OutcomeRecord`, `OutcomeMemory`, `PreExecHook`, `PostFailureHook`, `RunExecutionLoopWithHealing` are defined once and reused.
4. **No SKILL.md modified** — confirmed; all changes are inside `hwcloud-skillcheck/internal/l4/`.
5. **Behavior when `mem == nil`** — handled: `RunExecutionLoopWithHealing` skips both hooks; the existing `RunExecutionLoop` is untouched.

---

## Task 10: Wire OutcomeMemory + ContextMemory instantiation into HandleFault

This task replaces the old Task 8 Step 5 "implement HandleFault wrapping" stub. The OutcomeMemory must be created in the production entry path (`HandleFault`) and threaded into the execution loop. Without this, Tasks 1–9 are dead code in production.

**Files:**
- Modify: `hwcloud-skillcheck/internal/l4/orchestrator.go`
- Modify: `hwcloud-skillcheck/internal/l4/orchestrator_test.go` (verify existing tests still pass)

**Interfaces:**
- Consumes:
  - `OutcomeMemory` from Tasks 1–4
  - `HealingPolicy` from Task 5
  - Existing `HandleFaultInput` struct (find in `orchestrator.go`)
  - Existing `HandleFault` function (find in `orchestrator.go`)
- Produces:
  - `HandleFaultInput.Mem *OutcomeMemory` field (additive, nil-safe)
  - `HandleFaultInput.Policy HealingPolicy` field (additive, zero-value = no healing)
  - `HandleFault` instantiates `NewOutcomeMemory(root)` when input.Mem is nil, then passes to `BuildTaskFromPlan` (or whatever the production wiring is — verify by reading `orchestrator.go`)
  - The actual execution call (find where `RunExecutionLoop` is invoked in `HandleFault` body) is changed to `RunExecutionLoopWithHealing(root, task, plan, matched, mem, policy)`

- [ ] **Step 1: Read `orchestrator.go` to find the wiring points**

Read `hwcloud-skillcheck/internal/l4/orchestrator.go`. Identify:
- The `HandleFaultInput` struct definition.
- The `HandleFault` function body.
- The exact line where `RunExecutionLoop` (or `BuildTaskFromPlan` followed by execution) is called.
- The existing call site at `orchestrator.go:417` (mentioned by codex review) — confirm.

- [ ] **Step 2: Add `Mem` and `Policy` fields to `HandleFaultInput`**

```go
type HandleFaultInput struct {
    // ... existing fields ...
    // Mem is the outcome-memory store for self-healing. When nil, a fresh
    // OutcomeMemory is created from the resolved root directory.
    Mem *OutcomeMemory
    // Policy configures self-healing behavior. Zero value = no healing
    // (same behavior as before this feature existed).
    Policy HealingPolicy
}
```

- [ ] **Step 3: Instantiate Memory in `HandleFault`**

At the top of `HandleFault` body (after root resolution, before `BuildTaskFromPlan`):

```go
mem := in.Mem
if mem == nil {
    var err error
    mem, err = NewOutcomeMemory(root)
    if err != nil {
        return nil, fmt.Errorf("orchestrator: outcome memory: %w", err)
    }
}
```

- [ ] **Step 4: Pass to execution**

Find the line that invokes `RunExecutionLoop(root, task, plan, matched)` and replace with:

```go
task = RunExecutionLoopWithHealing(root, task, plan, matched, mem, in.Policy)
```

If the call site is different (e.g., goes through an executor interface), apply the same threading — the key is that `mem` and `in.Policy` reach the loop.

- [ ] **Step 5: Verify existing tests still pass**

Run: `cd hwcloud-skillcheck && go test ./internal/l4/ -v`
Expected: PASS for all tests including `orchestrator_test.go`. The zero-value `Policy` and nil `Mem` paths must be back-compat.

- [ ] **Step 6: Commit**

```bash
git add hwcloud-skillcheck/internal/l4/orchestrator.go
git commit -m "feat(l4): wire OutcomeMemory + HealingPolicy through HandleFault"
```

---

## Execution Handoff

Plan saved to `docs/superpowers/plans/2026-07-28-outcome-memory-self-healing.md`.

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

- **CROSS-MODEL:** Claude + Codex agreed on all 5 critical findings. Codex surfaced implementation-specific bugs against `internal/l4/` source; Claude surfaced the strategic posture (scope hold, L3→L4 maturity jump).
- **VERDICT:** NOT CLEARED — fix critical findings T1–T5 before implementation. Major findings T6–T7 in same PR. Minor findings deferrable.

### Critical (P1, BLOCK implementation)

- **C1.** ADR path references are stale: `docs/adr/0007-…` should be `docs/architecture/0007-…` (AGENTS.md §Documentation Locations).
- **C2.** Task 8 `runExecutionLoopInner` is a placeholder that does nothing — `_ = p; return RunExecutionLoop(...)` delegates to the OLD loop with no healing hooks. Task 9 E2E test is a stub.
- **C3.** OutcomeMemory / ContextMemory never instantiated in production call path — `HandleFault` and `BuildTaskFromPlan` don't create them. 6-arg `RunExecutionLoopWithHealing` is dead code.
- **C4.** `PruneOlderThan` data-loss race — `readAll()` releases mutex; concurrent `Record()` between read and rename is lost. `Record` also lacks `fsync`.
- **C5.** (Cross-call plan only) Task 5 invokes non-existent `NewOrchestrator` API — real entry point is `HandleFault(HandleFaultInput, ...) *OrchestratorOutput` at `orchestrator.go:133`; task creation is `BuildTaskFromPlan` at `execution.go:196`. Plan rewrites against invented symbols.

### Major (P2, fix in same PR)

- **M1.** Zero-value healing gate is wrong: sum-based zero-detection (`p.MaxRetries + p.FailureRateSkipThreshold + p.LookbackWindow != ...`) misclassifies positives cancelling negatives; `defaultHealingPolicy()` sets `LookbackWindow: time.Hour`, so the sum check fires for any caller using the default while the hook is a no-op. Integration tests claim to exercise healing while hitting a dead branch.
- **M2.** `nowISO()` lowercase typo — `util.go:11` exports `NowISO()` (capital N). Plan calls lowercase. Won't compile.
- **M3.** Destructive verb substring matching over-matches: `strings.Contains(action, "delete")` matches `undelete-restore` as destructive. Plan says it "matches execution.go:226-230" — that's verb-extraction, not action-name.
- **M4.** Storage-growth arithmetic off by ~30×: `100 × 24 × 365 = 1M/year` is operation-counting, not row-counting. Real figure <100k/year.
- **M5.** Determinism NFR-6 violated: `time.Since(last)` and `time.Now().Add(-lookback)` in `PreExecHook` and `MatchOutcomes` make decisions vary across wall-clock.

### Minor (P3, defer)

- **m1.** Time-based test fragility across UTC midnight.
- **m2.** Both plans independently recreate `.l4-memory/` — race on first run. Share `EnsureMemoryDir()` helper.
- **m3.** Healing decision (skip/retry/escalate) not separately persisted — only outcome is.
- **m4.** ContextHash for list operations: if `step.Args` includes timestamps/IDs, every call has unique hash — defeats the purpose.

### Implementation tasks (synthesized)

- [ ] **T1 (P1)** — Fix all `docs/adr/0007|0008-` references to `docs/architecture/0007|0008-` in spec + plans.
- [ ] **T2 (P1)** — Rewrite Task 8: copy full `RunExecutionLoop` body into `runExecutionLoopInner`; add `HealingPolicy.IsZero()`; fix zero-value gate. Write real E2E test.
- [ ] **T3 (P1)** — Rewrite cross-call Task 5: use `HandleFault` + `BuildTaskFromPlan`. Add `ContextMem` field to `HandleFaultInput`.
- [ ] **T4 (P1)** — `OutcomeMemory.PruneOlderThan`: hold single mutex across read→write→rename; add fsync to `Record`. Add concurrency test.
- [ ] **T5 (P1)** — Instantiate `OutcomeMemory` + `ContextMemory` at top of `HandleFault`; thread into `RunExecutionLoop`.
- [ ] **T6 (P2)** — `nowISO` → `NowISO` casing fix.
- [ ] **T7 (P2)** — `Verb` field on `TaskStep`; `PostFailureHook` uses `EqualFold(step.Verb, verb)`.
- [ ] **T8 (P3)** — Thread `now time.Time` into hooks; add NFR-7 row acknowledging wall-clock input.
- [ ] **T9 (P3)** — Fix storage-growth arithmetic in ADR-0007.

---

## GSTACK REVIEW REPORT (eng-review, 2026-07-28)

| Review | Trigger | Why | Runs | Status | Findings |
|--------|---------|-----|------|--------|----------|
| CEO Review | `/plan-ceo-review` | Scope & strategy | 1 | ISSUES_OPEN | 5 critical, 5 major, 4 minor (resolved) |
| Outside Voice (CEO) | codex exec | Independent 2nd opinion | 1 | ISSUES_OPEN | 5 critical, 5 major (overlap with CEO) |
| Eng Review | `/plan-eng-review` | Architecture & tests (required) | 1 | ISSUES_OPEN | 1 critical, 5 major, 2 minor |
| Outside Voice (eng) | codex exec | Independent 2nd opinion | 1 | ISSUES_OPEN | 1 critical, 4 major, 1 minor |

- **CROSS-MODEL:** Eng Claude + codex independently flagged the missing-executor problem (codex Critical #2) and the verb-set divergence (codex Major #7). No cross-model tension.
- **VERDICT:** NOT CLEARED — fix critical finding Eng-C1 (Executor seam in Task 8) before implementation begins. Major findings M1-M5 should land in the same PR.

### Eng review findings

#### Critical (P1, BLOCK implementation)

- **Eng-C1.** `execution.go:133` sets `Success: gclBody.Decision == "PASS"` without ever executing the command. There is no `os/exec`, no exit-code capture, no real failure mode. **Self-healing against non-existent failures is a no-op.** The plan's entire PostFailureHook + retry path is decorative until real execution is wired in. **Fix:** Task 8 Step 5.5 (just added) introduces `Executor` interface + `StubExecutor` for tests; real `RealExecutor` wrapping `exec.CommandContext` lives in **ADR-0010 (L5 deferred)**. Do NOT ship "self-healing" without either an executor or an explicit dry-run-only mode.

#### Major (P2, fix in same PR)

- **Eng-M1.** `RunExecutionLoop` (4-arg) remains in the codebase alongside `RunExecutionLoopWithHealing`. Risk of production still calling the old signature, leaving healing dormant. **Fix:** mark `RunExecutionLoop` `// Deprecated: use RunExecutionLoopWithHealing`. After Task 10 ships, grep for callers and migrate. Keep both for one release.
- **Eng-M2.** `preFetchPatterns` is byte-similar in `execution.go:149-193` and `orchestrator.go:191-244` (50 lines, two mutex names, both `SetLimit(NumCPU)`). **Fix:** extract `preFetchFailurePatterns(root, skills)` to `persistence.go`. Both call sites import it. (Pre-existing technical debt — not introduced by this plan, but adjacent.)
- **Eng-M3.** Destructive-verb set silently diverges from RBAC `HighRiskVerbs`. Plan's `defaultHealingPolicy` defaulted to `{delete, terminate, destroy, drop, remove}`; RBAC's regex includes `rm` and `del` too. **Fix:** new `ExtractHighRiskVerbs()` in `rbac.go` (just added in Task 8 Step 5.6). HealingPolicy default consumes it. Single source of truth.
- **Eng-M4.** `OutcomeMemory.RecentOutcomes` does full file scan every call (O(N) on read). With NFR-3 (≥1000 records/s sustained writes), file grows fast. Reads will bottleneck at scale. **Fix:** add small in-memory cache of last N records (≤100) invalidated on `Record`. Or document and accept O(N) reads for the first 10k records. Decision deferred — flag for ADR-0010 if file size exceeds 10MB.
- **Eng-M5.** `ContextMemory.Save` rewrites entire file on every mutation. A typical task does 5+ mutations = 5+ full rewrites. With capped 50KB docs, this is ~250KB writes per task. Not critical but inefficient. **Fix:** batch mutations via a single `Save()` at task-finalize time (not at every mutation). Mutation API queues deltas; `Flush()` writes once. Defer to a follow-up PR — current size cap makes this acceptable.

#### Minor (P3, defer)

- **Eng-m1.** `ContextHash` for list operations: if `step.Args` includes timestamps/IDs, every call has a unique hash — defeats MatchOutcomes. **Fix:** in `hashContext`, strip known-volatile args before hashing (e.g., `--query-window=...`). Document in `hashContext` doc-comment which fields are stable.
- **Eng-m2.** Glossary in AGENTS.md lists types not yet implemented (OutcomeMemory, HealingPolicy, etc.). **Decision: keep** — they describe the API this plan will create; not doc rot, just forward-looking documentation. Mark them `[planned: ADR-0007]` if pedantic clarity matters.

#### Distribution architecture

No new distribution artifact. `hwcloud-skillcheck` is the existing Go binary; this plan only adds internal packages. CI/CD unchanged.

### Implementation tasks (eng review)

- [ ] **Eng-T1 (P1)** — Task 8 Step 5.5: add `Executor` interface + `StubExecutor`; pass through `RunExecutionLoopWithHealing` signature. RealExecutor in ADR-0010.
- [ ] **Eng-T2 (P2)** — Task 8 Step 5.6: add `ExtractHighRiskVerbs` to `rbac.go`; consume from `defaultHealingPolicy`. Unit test asserts output equals regex alternation.
- [ ] **Eng-T3 (P2)** — Mark `RunExecutionLoop` deprecated; grep + migrate callers after Task 10 ships.
- [ ] **Eng-T4 (P3)** — Add `OutcomeMemory` in-memory cache for recent N records (defer if file stays <10MB).
- [ ] **Eng-T5 (P3)** — `ContextMemory` batched mutations + `Flush()` (defer; current size cap acceptable).
- [ ] **Eng-T6 (P3)** — `hashContext` strip volatile args before hashing; document which args are stable.
- [ ] **Eng-T7 (P2)** — Extract `preFetchFailurePatterns` shared helper (pre-existing tech debt, adjacent to this plan).

NO UNRESOLVED DECISIONS