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
