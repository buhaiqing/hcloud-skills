package l4

import (
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
