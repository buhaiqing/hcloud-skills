package l4

import (
	"bytes"
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
