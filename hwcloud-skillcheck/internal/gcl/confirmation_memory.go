// Package gcl — confirmation_memory.go
//
// MemoryStore is the default ConfirmationStore: an in-process map with the
// exact one-time-consumption + TTL + binding semantics the registry used
// before the store abstraction was extracted. Nonces do NOT survive a
// process restart; use FileStore (confirmation_file.go) when durability is
// required.
package gcl

import (
	"sort"
	"sync"
	"time"
)

// MemoryStore keeps confirmation entries in a Go map. Safe for concurrent
// use. Construct via NewMemoryStore.
type MemoryStore struct {
	mu      sync.Mutex
	entries map[string]*confirmationEntry
	now     func() time.Time
}

// NewMemoryStore returns an empty in-process store using time.Now as its
// clock. The owning registry overrides the clock via SetClock.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		entries: make(map[string]*confirmationEntry),
		now:     time.Now,
	}
}

// SetClock injects the clock used for expiry decisions.
func (s *MemoryStore) SetClock(now func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = now
}

// Issue stores a fresh entry. Re-issuing the same nonce overwrites (the
// registry never mints a duplicate, so this is effectively insert-only).
func (s *MemoryStore) Issue(e ConfirmationStoreEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[e.Nonce] = &confirmationEntry{
		nonce:     e.Nonce,
		op:        e.Op,
		actor:     e.Actor,
		issuedAt:  e.IssuedAt,
		expiresAt: e.ExpiresAt,
		consumed:  false,
	}
	return nil
}

// Consume performs the atomic check-and-consume described on
// ConfirmationStore.
func (s *MemoryStore) Consume(nonce, op, actor string, enforceBinding bool) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[nonce]
	if !ok {
		return false, ErrUnknown
	}
	if entry.consumed {
		return false, ErrReplay
	}
	if !s.now().Before(entry.expiresAt) {
		// Eagerly remove the stale entry so Pending() never returns it.
		delete(s.entries, nonce)
		return false, ErrExpired
	}
	if enforceBinding && (entry.op != op || entry.actor != actor) {
		return false, errBindingMismatch
	}
	entry.consumed = true
	return true, nil
}

// Revoke removes a nonce, returning ErrUnknown when absent.
func (s *MemoryStore) Revoke(nonce string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[nonce]; !ok {
		return ErrUnknown
	}
	delete(s.entries, nonce)
	return nil
}

// Pending returns a time-sorted snapshot (most recent first) of live,
// non-expired entries.
func (s *MemoryStore) Pending() []Confirmation {
	s.mu.Lock()
	now := s.now()
	out := make([]Confirmation, 0, len(s.entries))
	for _, e := range s.entries {
		if !now.Before(e.expiresAt) {
			continue
		}
		out = append(out, Confirmation{
			Nonce:     e.nonce,
			Op:        e.op,
			Actor:     e.actor,
			IssuedAt:  e.issuedAt,
			ExpiresAt: e.expiresAt,
		})
	}
	s.mu.Unlock()
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].IssuedAt.After(out[j].IssuedAt)
	})
	return out
}

// PruneExpired drops all entries whose expiresAt is in the past.
func (s *MemoryStore) PruneExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for k, e := range s.entries {
		if !now.Before(e.expiresAt) {
			delete(s.entries, k)
		}
	}
}
