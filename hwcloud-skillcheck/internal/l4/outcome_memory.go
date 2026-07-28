package l4

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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
