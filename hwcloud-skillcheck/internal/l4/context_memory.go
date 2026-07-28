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

// ContextMemory owns <root>/.l4-memory/context.json.
//
// Mutations (RecordTask / RecordError / SetPreference / CloseTask) update an
// in-memory dirty buffer only. Call Flush() once at task-finalize to write
// the document to disk (Eng-T5). Same-handle Load() sees queued mutations
// without a Flush; a fresh ContextMemory handle only sees them after Flush.
type ContextMemory struct {
	path  string
	mu    sync.Mutex
	ctx   *Context // working copy; nil until first mutation or Save
	dirty bool
}

// NewContextMemory ensures <root>/.l4-memory/ exists (creating it if
// needed — idempotent with NewOutcomeMemory) and returns a store
// pointing at <root>/.l4-memory/context.json.
func NewContextMemory(root string) (*ContextMemory, error) {
	dir, err := EnsureMemoryDir(root)
	if err != nil {
		return nil, fmt.Errorf("context memory: mkdir: %w", err)
	}
	return &ContextMemory{path: filepath.Join(dir, "context.json")}, nil
}

// Save atomically writes c to disk and replaces the in-memory working copy.
// The temp file is fsynced, renamed over the target, and removed if the
// rename fails. The on-disk schema is enforced to ContextSchema.
func (m *ContextMemory) Save(c *Context) error {
	if c == nil {
		return fmt.Errorf("context memory: nil context")
	}
	if c.Schema != ContextSchema {
		return fmt.Errorf("context memory: refusing to save schema %q (want %q)", c.Schema, ContextSchema)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.writeLocked(c); err != nil {
		return err
	}
	m.ctx = cloneContext(c)
	m.dirty = false
	return nil
}

// Flush writes the dirty in-memory context to disk once. No-op when clean
// or when no working copy has been established. Call at task-finalize.
func (m *ContextMemory) Flush() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.dirty || m.ctx == nil {
		return nil
	}
	if err := m.writeLocked(m.ctx); err != nil {
		return err
	}
	m.dirty = false
	return nil
}

// Dirty reports whether queued mutations have not yet been Flushed.
func (m *ContextMemory) Dirty() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dirty
}

// writeLocked persists c. Caller must hold m.mu.
func (m *ContextMemory) writeLocked(c *Context) error {
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("context memory: marshal: %w", err)
	}
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

// Load returns the working context. If a dirty (or previously loaded)
// working copy exists, that is returned. Otherwise the document is read
// from disk. If the file does not exist, returns a fresh zero-value
// Context with a newly generated session_id (no persistence side-effect
// and no cache). If the document has an unknown schema, returns an
// error. If CreatedAt is older than SessionRotateAfter, rotates the
// session_id in the returned copy; rotation is marked dirty so Flush
// persists it.
func (m *ContextMemory) Load() (*Context, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ctx != nil {
		if !m.dirty {
			if rotateSessionIfExpired(m.ctx) {
				m.dirty = true
			}
		}
		return cloneContext(m.ctx), nil
	}
	c, rotated, err := m.readFromDiskLocked()
	if err != nil {
		return nil, err
	}
	if c == nil {
		// First-run: fresh context, no cache (preserves "two Loads → two
		// session_ids" until the first mutation establishes a working copy).
		return freshContext(), nil
	}
	m.ctx = c
	if rotated {
		m.dirty = true
	}
	return cloneContext(m.ctx), nil
}

// readFromDiskLocked loads from disk. Returns (nil, false, nil) when the
// file does not exist. Caller must hold m.mu.
func (m *ContextMemory) readFromDiskLocked() (*Context, bool, error) {
	raw, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("context memory: read: %w", err)
	}
	var c Context
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, false, fmt.Errorf("context memory: parse: %w", err)
	}
	if c.Schema != ContextSchema {
		return nil, false, fmt.Errorf("context memory: unknown schema %q (want %q)", c.Schema, ContextSchema)
	}
	rotated := rotateSessionIfExpired(&c)
	return &c, rotated, nil
}

// rotateSessionIfExpired mutates c in place when CreatedAt is past
// SessionRotateAfter. Returns true when a rotation happened.
func rotateSessionIfExpired(c *Context) bool {
	if c == nil {
		return false
	}
	ts, err := time.Parse(time.RFC3339, c.CreatedAt)
	if err != nil || time.Since(ts) <= SessionRotateAfter {
		return false
	}
	c.SessionID = newSessionID()
	c.CreatedAt = NowISO()
	c.LastUpdated = NowISO()
	return true
}

// ensureLoadedLocked establishes m.ctx from disk or a fresh context.
// Caller must hold m.mu.
func (m *ContextMemory) ensureLoadedLocked() error {
	if m.ctx != nil {
		return nil
	}
	c, rotated, err := m.readFromDiskLocked()
	if err != nil {
		return err
	}
	if c == nil {
		m.ctx = freshContext()
		return nil
	}
	m.ctx = c
	if rotated {
		m.dirty = true
	}
	return nil
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

// cloneContext returns a deep-enough copy for callers to read safely.
func cloneContext(c *Context) *Context {
	if c == nil {
		return nil
	}
	out := *c
	if c.RecentTasks != nil {
		out.RecentTasks = append([]TaskSummary(nil), c.RecentTasks...)
	}
	if c.OpenTasks != nil {
		out.OpenTasks = append([]string(nil), c.OpenTasks...)
	}
	if c.RecentErrors != nil {
		out.RecentErrors = append([]ErrorSummary(nil), c.RecentErrors...)
	}
	if c.Preferences != nil {
		out.Preferences = make(map[string]string, len(c.Preferences))
		for k, v := range c.Preferences {
			out.Preferences[k] = v
		}
	}
	return &out
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

// RecordTask prepends t to RecentTasks (newest first) and caps at MaxRecentTasks.
// If status is "running" or "paused", t.TaskID is also prepended to OpenTasks
// (capped at MaxOpenTasks). Queues the change; call Flush to persist.
func (m *ContextMemory) RecordTask(t TaskSummary) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureLoadedLocked(); err != nil {
		return err
	}
	c := m.ctx
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
	m.dirty = true
	return nil
}

// RecordError prepends e to RecentErrors (newest first), capped at MaxRecentErrors.
// Queues the change; call Flush to persist.
func (m *ContextMemory) RecordError(e ErrorSummary) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureLoadedLocked(); err != nil {
		return err
	}
	c := m.ctx
	c.RecentErrors = append([]ErrorSummary{e}, c.RecentErrors...)
	if len(c.RecentErrors) > MaxRecentErrors {
		c.RecentErrors = c.RecentErrors[:MaxRecentErrors]
	}
	c.LastUpdated = NowISO()
	m.dirty = true
	return nil
}

// SetPreference sets preferences[k] = v. Passing v == "" deletes the key.
// Queues the change; call Flush to persist.
func (m *ContextMemory) SetPreference(k, v string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureLoadedLocked(); err != nil {
		return err
	}
	c := m.ctx
	if c.Preferences == nil {
		c.Preferences = map[string]string{}
	}
	if v == "" {
		delete(c.Preferences, k)
	} else {
		c.Preferences[k] = v
	}
	c.LastUpdated = NowISO()
	m.dirty = true
	return nil
}

// CloseTask removes taskID from OpenTasks (no-op if absent).
// Queues the change; call Flush to persist.
func (m *ContextMemory) CloseTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureLoadedLocked(); err != nil {
		return err
	}
	c := m.ctx
	out := c.OpenTasks[:0]
	for _, id := range c.OpenTasks {
		if id != taskID {
			out = append(out, id)
		}
	}
	c.OpenTasks = out
	c.LastUpdated = NowISO()
	m.dirty = true
	return nil
}

// primarySkillOfTask returns the first non-empty Skill from task.Steps,
// or "" when the task has no steps.
func primarySkillOfTask(task *TaskState) string {
	if task == nil {
		return ""
	}
	for _, s := range task.Steps {
		if s.Skill != "" {
			return s.Skill
		}
	}
	return ""
}
