// Package gcl — confirmation_file.go
//
// FileStore is a durable ConfirmationStore that persists issued nonces to
// .l4-memory/confirmations.json (relative to a base dir). Unlike MemoryStore,
// a nonce issued before a process restart can still be consumed afterwards:
// a new FileStore loading the same path replays the on-disk entries (dropping
// any already expired).
//
// Durability contract:
//   - Every Issue / Consume / Revoke / PruneExpired flushes the full state
//     to disk via an atomic temp-file + os.Rename, so readers never observe
//     a partially-written file.
//   - Load-on-init tolerates a missing file (starts empty, no error) and a
//     corrupt/truncated file is surfaced as an error from NewFileStore.
package gcl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// confirmationDirName / confirmationFileName give the on-disk location of
// the persisted store, joined onto the base dir passed to NewFileStore.
const (
	confirmationDirName  = ".l4-memory"
	confirmationFileName = "confirmations.json"
)

// fileEntry is the JSON-serialisable row. It mirrors confirmationEntry but
// with exported fields + tags so it survives a marshal round-trip.
type fileEntry struct {
	Nonce     string    `json:"nonce"`
	Op        string    `json:"op"`
	Actor     string    `json:"actor"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Consumed  bool      `json:"consumed"`
}

// FileStore is a durable, disk-backed ConfirmationStore. Safe for
// concurrent use. Construct via NewFileStore.
type FileStore struct {
	mu      sync.Mutex
	path    string
	entries map[string]*fileEntry
	now     func() time.Time
}

// NewFileStore returns a FileStore persisting to
// <baseDir>/.l4-memory/confirmations.json. An empty baseDir defaults to ".".
// Existing on-disk state is loaded immediately; entries already expired
// (relative to time.Now at load) are dropped and NOT written back until the
// next mutation. A missing file is not an error; a malformed file is.
func NewFileStore(baseDir string) (*FileStore, error) {
	if baseDir == "" {
		baseDir = "."
	}
	s := &FileStore{
		path:    filepath.Join(baseDir, confirmationDirName, confirmationFileName),
		entries: make(map[string]*fileEntry),
		now:     time.Now,
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// load reads the JSON file (if present) into memory, dropping expired
// entries. Called once from the constructor before the store is shared.
func (s *FileStore) load() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // fresh store
		}
		return fmt.Errorf("confirmation: read %s: %w", s.path, err)
	}
	if len(raw) == 0 {
		return nil // empty file == fresh store
	}
	var rows []fileEntry
	if err := json.Unmarshal(raw, &rows); err != nil {
		return fmt.Errorf("confirmation: parse %s: %w", s.path, err)
	}
	now := s.now()
	for i := range rows {
		e := rows[i]
		if !now.Before(e.ExpiresAt) {
			continue // drop expired on load
		}
		s.entries[e.Nonce] = &e
	}
	return nil
}

// SetClock injects the clock used for expiry decisions.
func (s *FileStore) SetClock(now func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = now
}

// Issue stores a fresh entry and flushes to disk.
func (s *FileStore) Issue(e ConfirmationStoreEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[e.Nonce] = &fileEntry{
		Nonce:     e.Nonce,
		Op:        e.Op,
		Actor:     e.Actor,
		IssuedAt:  e.IssuedAt,
		ExpiresAt: e.ExpiresAt,
		Consumed:  false,
	}
	return s.flushLocked()
}

// Consume performs the atomic check-and-consume described on
// ConfirmationStore, persisting the consumed flag (or the expired-entry
// removal) to disk before returning success.
func (s *FileStore) Consume(nonce, op, actor string, enforceBinding bool) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[nonce]
	if !ok {
		return false, ErrUnknown
	}
	if entry.Consumed {
		return false, ErrReplay
	}
	if !s.now().Before(entry.ExpiresAt) {
		delete(s.entries, nonce)
		if err := s.flushLocked(); err != nil {
			return false, err
		}
		return false, ErrExpired
	}
	if enforceBinding && (entry.Op != op || entry.Actor != actor) {
		return false, errBindingMismatch
	}
	entry.Consumed = true
	if err := s.flushLocked(); err != nil {
		// Roll back the in-memory flag so a failed flush doesn't leave the
		// store claiming a nonce is consumed when disk says otherwise.
		entry.Consumed = false
		return false, err
	}
	return true, nil
}

// Revoke removes a nonce (returning ErrUnknown when absent) and flushes.
func (s *FileStore) Revoke(nonce string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[nonce]; !ok {
		return ErrUnknown
	}
	delete(s.entries, nonce)
	return s.flushLocked()
}

// Pending returns a time-sorted snapshot (most recent first) of live,
// non-expired entries. Matches MemoryStore: consumed-but-not-expired
// entries are still listed.
func (s *FileStore) Pending() []Confirmation {
	s.mu.Lock()
	now := s.now()
	out := make([]Confirmation, 0, len(s.entries))
	for _, e := range s.entries {
		if !now.Before(e.ExpiresAt) {
			continue
		}
		out = append(out, Confirmation{
			Nonce:     e.Nonce,
			Op:        e.Op,
			Actor:     e.Actor,
			IssuedAt:  e.IssuedAt,
			ExpiresAt: e.ExpiresAt,
		})
	}
	s.mu.Unlock()
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].IssuedAt.After(out[j].IssuedAt)
	})
	return out
}

// PruneExpired drops expired entries and flushes if anything changed.
func (s *FileStore) PruneExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	changed := false
	for k, e := range s.entries {
		if !now.Before(e.ExpiresAt) {
			delete(s.entries, k)
			changed = true
		}
	}
	if changed {
		// Best-effort: a flush failure here is non-fatal (the entries are
		// already gone in memory; next mutation will re-flush).
		_ = s.flushLocked()
	}
}

// flushLocked atomically writes the current entry set to disk. Caller MUST
// hold s.mu. Writes to a temp file in the same directory, then renames over
// the target so readers never see a partial file.
func (s *FileStore) flushLocked() error {
	dir := filepath.Dir(s.path)
	// 0o700: the confirmation store holds security-gate nonces; keep the
	// .l4-memory dir owner-only (not world-traversable) as defense-in-depth.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("confirmation: mkdir %s: %w", dir, err)
	}
	rows := make([]fileEntry, 0, len(s.entries))
	for _, e := range s.entries {
		rows = append(rows, *e)
	}
	// Deterministic order keeps the file stable for diffing/tests.
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].IssuedAt.Equal(rows[j].IssuedAt) {
			return rows[i].Nonce < rows[j].Nonce
		}
		return rows[i].IssuedAt.Before(rows[j].IssuedAt)
	})
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return fmt.Errorf("confirmation: marshal: %w", err)
	}
	tmp, err := os.CreateTemp(dir, confirmationFileName+".tmp-*")
	if err != nil {
		return fmt.Errorf("confirmation: create temp: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("confirmation: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("confirmation: fsync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("confirmation: close temp: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("confirmation: rename %s: %w", s.path, err)
	}
	return nil
}
