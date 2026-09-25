package l4

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
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
	ActionDigest string `json:"action_digest,omitempty"`
	ContextHash  string `json:"context_hash"`
	Outcome      string `json:"outcome"`
	ErrorClass   string `json:"error_class"`
	ErrorMsg     string `json:"error_msg,omitempty"`
	RetryCount   int    `json:"retry_count"`
	DurationMS   int64  `json:"duration_ms"`
	Risk         string `json:"risk"`
	RBACDecision string `json:"rbac_decision"`
	GCLDecision  string `json:"gcl_decision"`

	// GCL campaign process metrics. A GCL campaign measures the improvement
	// loop, not just the code it ships: how often dispatches failed, how much
	// of the batch was rewritten in later fix rounds, and how many fix rounds
	// got re-reviewed. Without these, "the GCL worked" is unfalsifiable across
	// sessions.
	//
	// Rates are pointers so a legitimately measured 0.0 stays distinguishable
	// from "not measured" under omitempty. Critic-signal quality is
	// intentionally absent: a single false-positive rate would conflate a Critic
	// being wrong with an Adjudicator wrongly overturning a Critic, so those
	// counts stay raw in audit-results/gcl-campaigns/<id>.json instead.
	GCLExperimentID         string   `json:"gcl_experiment_id,omitempty"`
	DispatchFailureRate     *float64 `json:"dispatch_failure_rate,omitempty"`
	FixReworkRate           *float64 `json:"fix_rework_rate,omitempty"`
	PostFixRereviewCoverage *float64 `json:"post_fix_rereview_coverage,omitempty"`
}

// outcomeKeyCacheSize is the per-(skill, action) RecentOutcomes window
// (Eng-T4 / T-4). Matches trustOutcomeMaxRecords so trust/healing hot paths
// answer from memory after the first disk scan.
const outcomeKeyCacheSize = 100

var outcomeIdentityKey = func() []byte {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		panic("outcome memory: system CSPRNG unavailable")
	}
	return key[:]
}()

func outcomeActionDigest(action string) string {
	mac := hmac.New(sha256.New, outcomeIdentityKey)
	_, _ = mac.Write([]byte(action))
	return hex.EncodeToString(mac.Sum(nil))
}

func withOutcomeFileLock(path string, fn func() error) error {
	lockPath := path + ".lock"
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := lock.Chmod(0o600); err != nil {
		return err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	return fn()
}

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
	rawAction := r.Action
	persisted := r
	persisted.ActionDigest = outcomeActionDigest(rawAction)
	// ContextHash is computed from the raw command before Record and must stay
	// byte-for-byte stable for correlation. Every other string is externally
	// supplied and crosses the persistence boundary through this one redactor.
	for _, field := range []struct {
		name  string
		value *string
	}{
		{"id", &persisted.ID}, {"timestamp", &persisted.Timestamp},
		{"task id", &persisted.TaskID}, {"skill", &persisted.Skill},
		{"action", &persisted.Action}, {"outcome", &persisted.Outcome},
		{"error class", &persisted.ErrorClass}, {"error message", &persisted.ErrorMsg},
		{"risk", &persisted.Risk}, {"rbac decision", &persisted.RBACDecision},
		{"gcl decision", &persisted.GCLDecision}, {"gcl experiment id", &persisted.GCLExperimentID},
	} {
		safe, err := sanitizeStringForPersistence(field.name, *field.value)
		if err != nil {
			if field.name == "action" {
				safe = persistenceMask
			} else {
				return fmt.Errorf("outcome memory: %w", err)
			}
		}
		*field.value = safe
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	line, err := json.Marshal(persisted)
	if err != nil {
		return fmt.Errorf("outcome memory: marshal: %w", err)
	}
	return withOutcomeFileLock(m.path, func() error {
		if err := os.Chmod(m.path, 0o600); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("outcome memory: chmod: %w", err)
		}
		if err := repairOutcomeTail(m.path); err != nil {
			return err
		}
		f, err := os.OpenFile(m.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("outcome memory: open: %w", err)
		}
		n, writeErr := f.Write(append(line, '\n'))
		closeErr := f.Close()
		if writeErr != nil {
			return fmt.Errorf("outcome memory: write: %w", writeErr)
		}
		if n != len(line)+1 {
			return fmt.Errorf("outcome memory: write: %w", io.ErrShortWrite)
		}
		if closeErr != nil {
			return fmt.Errorf("outcome memory: close: %w", closeErr)
		}
		k := outcomeCacheKey(persisted.Skill, persisted.ActionDigest)
		if cached, ok := m.keyCache[k]; ok {
			m.keyCache[k] = trimNewestFirst(append([]OutcomeRecord{persisted}, cached...), outcomeKeyCacheSize)
		}
		return nil
	})
}

func repairOutcomeTail(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.Size() == 0 {
		return err
	}
	buf := make([]byte, 1)
	if _, err := f.ReadAt(buf, info.Size()-1); err != nil {
		return err
	}
	if buf[0] == '\n' {
		return nil
	}
	raw := make([]byte, info.Size())
	if _, err := f.ReadAt(raw, 0); err != nil {
		return err
	}
	last := bytes.LastIndexByte(raw, '\n')
	if last < 0 {
		return f.Truncate(0)
	}
	return f.Truncate(int64(last + 1))
}

// FullScans returns how many times the JSONL file was fully parsed.
func (m *OutcomeMemory) FullScans() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fullScans
}

func (m *OutcomeMemory) readAll() ([]OutcomeRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readAllUnlocked()
}

func (m *OutcomeMemory) readAllUnlocked() ([]OutcomeRecord, error) {
	m.fullScans++
	var raw []byte
	err := withOutcomeFileLock(m.path, func() error { var err error; raw, err = os.ReadFile(m.path); return err })
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("outcome memory: read: %w", err)
	}
	var out []OutcomeRecord
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var r OutcomeRecord
		if json.Unmarshal(line, &r) == nil {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *OutcomeMemory) RecentOutcomes(skill, action string, n int) ([]OutcomeRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	safeSkill, err := sanitizeStringForPersistence("lookup skill", skill)
	if err != nil {
		return nil, fmt.Errorf("outcome memory: %w", err)
	}
	digest := outcomeActionDigest(action)
	safeAction, err := sanitizeStringForPersistence("lookup action", action)
	if err != nil {
		safeAction = persistenceMask
	}
	if n <= 0 {
		all, err := m.readAllUnlocked()
		if err != nil {
			return nil, err
		}
		match := filterOutcomeIdentity(all, safeSkill, safeAction, digest)
		sortNewestFirst(match)
		return append([]OutcomeRecord(nil), match...), nil
	}
	key := outcomeCacheKey(safeSkill, digest)
	cached, ok := m.keyCache[key]
	if !ok {
		all, err := m.readAllUnlocked()
		if err != nil {
			return nil, err
		}
		cached = filterOutcomeIdentity(all, safeSkill, safeAction, digest)
		sortNewestFirst(cached)
		if len(cached) > outcomeKeyCacheSize {
			cached = cached[:outcomeKeyCacheSize]
		}
		m.keyCache[key] = append([]OutcomeRecord(nil), cached...)
	}
	return cloneOutcomes(cached, n), nil
}

func (m *OutcomeMemory) MatchOutcomes(skill, action, contextHash string, lookback time.Duration) ([]OutcomeRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	all, err := m.readAllUnlocked()
	if err != nil {
		return nil, err
	}
	safeSkill, err := sanitizeStringForPersistence("lookup skill", skill)
	if err != nil {
		return nil, fmt.Errorf("outcome memory: %w", err)
	}
	digest := outcomeActionDigest(action)
	safeAction, err := sanitizeStringForPersistence("lookup action", action)
	if err != nil {
		safeAction = persistenceMask
	}
	cutoff := time.Time{}
	if lookback > 0 {
		cutoff = time.Now().Add(-lookback)
	}
	var match []OutcomeRecord
	for _, r := range all {
		if r.Skill != safeSkill || r.ContextHash != contextHash || !outcomeIdentityMatches(r, safeAction, digest) {
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

func outcomeIdentityMatches(r OutcomeRecord, safeAction, digest string) bool {
	return r.ActionDigest == digest || (r.ActionDigest == "" && r.Action == safeAction)
}

func filterOutcomeIdentity(all []OutcomeRecord, skill, action, digest string) []OutcomeRecord {
	var match []OutcomeRecord
	for _, r := range all {
		if r.Skill == skill && outcomeIdentityMatches(r, action, digest) {
			match = append(match, r)
		}
	}
	return match
}

func (m *OutcomeMemory) PruneOlderThan(cutoff time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keyCache = map[string][]OutcomeRecord{}
	dropped := 0
	err := withOutcomeFileLock(m.path, func() error {
		raw, err := os.ReadFile(m.path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("outcome memory: prune read: %w", err)
		}
		if len(raw) == 0 {
			return nil
		}
		m.fullScans++
		kept := make([]OutcomeRecord, 0, 16)
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
			return nil
		}
		tmp, err := os.CreateTemp(filepath.Dir(m.path), "outcomes-*.jsonl.tmp")
		if err != nil {
			return fmt.Errorf("outcome memory: prune tmp: %w", err)
		}
		tmpName := tmp.Name()
		cleanup := func() { _ = os.Remove(tmpName) }
		if err := tmp.Chmod(0o600); err != nil {
			tmp.Close()
			cleanup()
			return err
		}
		for _, r := range kept {
			line, err := json.Marshal(r)
			if err != nil {
				tmp.Close()
				cleanup()
				return err
			}
			data := append(line, '\n')
			n, err := tmp.Write(data)
			if err != nil {
				tmp.Close()
				cleanup()
				return err
			}
			if n != len(data) {
				tmp.Close()
				cleanup()
				return io.ErrShortWrite
			}
		}
		if err := tmp.Sync(); err != nil {
			tmp.Close()
			cleanup()
			return err
		}
		if err := tmp.Close(); err != nil {
			cleanup()
			return err
		}
		if err := os.Rename(tmpName, m.path); err != nil {
			cleanup()
			return err
		}
		return syncDirectory(filepath.Dir(m.path))
	})
	return dropped, err
}

func syncDirectory(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
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
