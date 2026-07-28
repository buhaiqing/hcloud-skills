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

// outcomeKeyCacheSize is the per-(skill, action) RecentOutcomes window
// (Eng-T4 / T-4). Matches trustOutcomeMaxRecords so trust/healing hot paths
// answer from memory after the first disk scan.
const outcomeKeyCacheSize = 100

// OutcomeMemory is an append-only outcome store backed by a single JSONL file.
//
// RecentOutcomes keeps an in-memory cache of the newest outcomeKeyCacheSize
// records per (skill, action), filled on first read and updated on Record.
// PruneOlderThan clears the cache. MatchOutcomes still scans the file (rarer,
// lookback/hash filtered).
type OutcomeMemory struct {
	path string
	mu   sync.Mutex
	// keyCache maps skill\x00action → records newest-first, capped at
	// outcomeKeyCacheSize. Missing key = cold (not yet loaded from disk).
	keyCache map[string][]OutcomeRecord
	// fullScans counts JSONL full-file parses (tests / light observability).
	fullScans int
}

// NewOutcomeMemory ensures <root>/.l4-memory/ exists and returns a store
// pointing at <root>/.l4-memory/outcomes.jsonl. Auto-prunes records older
// than 90 days on first open.
func NewOutcomeMemory(root string) (*OutcomeMemory, error) {
	dir, err := EnsureMemoryDir(root)
	if err != nil {
		return nil, fmt.Errorf("outcome memory: mkdir: %w", err)
	}
	mem := &OutcomeMemory{
		path:     filepath.Join(dir, "outcomes.jsonl"),
		keyCache: map[string][]OutcomeRecord{},
	}
	if _, err := mem.PruneOlderThan(time.Now().Add(-90 * 24 * time.Hour)); err != nil {
		return nil, fmt.Errorf("outcome memory: initial prune: %w", err)
	}
	return mem, nil
}

func outcomeCacheKey(skill, action string) string {
	return skill + "\x00" + action
}

// Record appends one OutcomeRecord as a single JSON line.
// fsync is intentionally NOT called per-record — the append-only file is
// recovered by PruneOlderThan or by readAll which scans forward. Per-record
// fsync would tank write throughput (NFR-3: >= 1000 records/s).
//
// When a (skill, action) cache entry is warm, the new record is prepended
// (newest-first) and trimmed to outcomeKeyCacheSize — no invalidation wipe.
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
	k := outcomeCacheKey(r.Skill, r.Action)
	if cached, ok := m.keyCache[k]; ok {
		m.keyCache[k] = trimNewestFirst(append([]OutcomeRecord{r}, cached...), outcomeKeyCacheSize)
	}
	return nil
}

// FullScans returns how many times the JSONL file was fully parsed.
// Intended for tests proving the RecentOutcomes cache avoids re-reads.
func (m *OutcomeMemory) FullScans() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fullScans
}

// readAll parses the entire JSONL file. Malformed lines are skipped silently.
// Returns an empty slice (not an error) if the file does not exist yet.
func (m *OutcomeMemory) readAll() ([]OutcomeRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readAllUnlocked()
}

// readAllUnlocked parses the JSONL file. Caller must hold m.mu.
func (m *OutcomeMemory) readAllUnlocked() ([]OutcomeRecord, error) {
	m.fullScans++
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
// recent first.
//
// n > 0: served from a per-key cache of the newest outcomeKeyCacheSize
// matches (first call scans disk once; later calls + Record hit memory).
// n <= 0: full-file scan returning every match (uncapped; not served from
// the size-limited cache).
func (m *OutcomeMemory) RecentOutcomes(skill, action string, n int) ([]OutcomeRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if n <= 0 {
		all, err := m.readAllUnlocked()
		if err != nil {
			return nil, err
		}
		match := filterSkillAction(all, skill, action)
		sortNewestFirst(match)
		return append([]OutcomeRecord(nil), match...), nil
	}

	k := outcomeCacheKey(skill, action)
	cached, ok := m.keyCache[k]
	if !ok {
		all, err := m.readAllUnlocked()
		if err != nil {
			return nil, err
		}
		cached = filterSkillAction(all, skill, action)
		sortNewestFirst(cached)
		if len(cached) > outcomeKeyCacheSize {
			cached = cached[:outcomeKeyCacheSize]
		}
		m.keyCache[k] = append([]OutcomeRecord(nil), cached...)
	}
	return cloneOutcomes(cached, n), nil
}

// MatchOutcomes returns records matching (skill, action, contextHash) whose
// Timestamp is within `lookback` of now. lookback <= 0 means "no time filter".
// Always reads from disk (hash/lookback queries are not covered by the
// per-key recent cache).
func (m *OutcomeMemory) MatchOutcomes(skill, action, contextHash string, lookback time.Duration) ([]OutcomeRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	all, err := m.readAllUnlocked()
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
	sortNewestFirst(match)
	return match, nil
}

// PruneOlderThan drops records whose Timestamp is strictly before cutoff.
// Returns the number of records removed. Safe to call on an empty file.
// Holds m.mu across the entire read→write→rename sequence to prevent
// data loss: a concurrent Record() between read and rename would
// otherwise be silently dropped when the rename overwrites the file.
// Clears the RecentOutcomes key cache.
func (m *OutcomeMemory) PruneOlderThan(cutoff time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keyCache = map[string][]OutcomeRecord{}
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
	// Count as a full scan (parse every line).
	m.fullScans++
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

func filterSkillAction(all []OutcomeRecord, skill, action string) []OutcomeRecord {
	var match []OutcomeRecord
	for _, r := range all {
		if r.Skill == skill && r.Action == action {
			match = append(match, r)
		}
	}
	return match
}

func sortNewestFirst(match []OutcomeRecord) {
	sort.SliceStable(match, func(i, j int) bool {
		return match[i].Timestamp > match[j].Timestamp
	})
}

func trimNewestFirst(recs []OutcomeRecord, n int) []OutcomeRecord {
	if n > 0 && len(recs) > n {
		return recs[:n]
	}
	return recs
}

func cloneOutcomes(cached []OutcomeRecord, n int) []OutcomeRecord {
	if n > 0 && len(cached) > n {
		cached = cached[:n]
	}
	return append([]OutcomeRecord(nil), cached...)
}
