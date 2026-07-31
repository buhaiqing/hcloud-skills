// Package gcl — confirmation.go
//
// ConfirmationRegistry implements a nonce store for destructive-
// operation approvals. The GCL loop hands a fresh nonce to a human reviewer
// when a request scores as destructive; the reviewer calls Verify (or
// VerifyBound when the caller can supply the original intent+actor) to
// consume the nonce exactly once.
//
// Design choices (matching the spec in docs/gcl-spec.md §confirmations):
//
//   - Storage is pluggable via ConfirmationStore. The default MemoryStore
//     keeps entries in-process (nonces do NOT survive process restart);
//     FileStore (confirmation_file.go) persists them to disk so a nonce
//     issued before a restart can still be consumed afterwards.
//   - TTL defaults to 60 seconds. Entries older than TTL are pruned either
//     lazily on Verify (so callers never see a stale nonce) or eagerly by
//     the background sweep loop.
//   - One-time consumption: the same nonce can be verified exactly once.
//     A second Verify returns a "replay/consumed" error and ok=false.
//   - Binding: Issue binds the nonce to (op, actor). Verify (no binding)
//     consumes regardless of op/actor — useful for the CLI adapter that
//     doesn't track intent; VerifyBound requires a matching binding and is
//     the fail-closed path for programmatic callers.
package gcl

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// DefaultConfirmationTTL is the spec default: 60s from Issue().
const DefaultConfirmationTTL = 60 * time.Second

// ConfirmationPrefix tags every nonce emitted by this registry so callers
// can tell a confirmation nonce apart from any other opaque id.
const ConfirmationPrefix = "cfm_"

// Confirmation is a snapshot of an issued nonce, returned from Pending()
// for observability. Callers MUST NOT mutate it.
type Confirmation struct {
	Nonce     string
	Op        string
	Actor     string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// ErrReplay is returned when the same nonce has been verified once already
// and is now being presented a second time. It is also returned by
// Verify/VerifyBound for a nonce that has been consumed and then revoked.
var ErrReplay = errors.New("confirmation: nonce already consumed (replay)")

// ErrNonceConsumed is the spec-mandated (A8) name for the replay error.
var ErrNonceConsumed = ErrReplay

var ErrExpired = errors.New("confirmation: nonce expired")

// ErrNonceExpired is the spec-mandated (A9) name for the TTL-expiry error.
var ErrNonceExpired = ErrExpired

var ErrUnknown = errors.New("confirmation: nonce unknown")

// errBindingMismatch is returned only by VerifyBound when (op, actor) don't
// match what the nonce was issued against.
var errBindingMismatch = errors.New("confirmation: op/actor binding mismatch")

// ConfirmationStoreEntry is the persistence-facing row handed to a
// ConfirmationStore.Issue. It carries everything a backend needs to
// enforce TTL + binding + one-time semantics later. Fields are exported
// so durable backends (FileStore) can marshal them.
type ConfirmationStoreEntry struct {
	Nonce     string
	Op        string
	Actor     string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// ConfirmationStore is the pluggable persistence backend behind a
// ConfirmationRegistry. Implementations MUST be safe for concurrent use.
// The registry owns nonce minting, TTL computation, and the sweeper; the
// store owns durability, one-time-consumption bookkeeping, and expiry
// enforcement using the clock injected via SetClock.
//
// Consume performs an atomic check-and-consume:
//   - unknown nonce            → (false, ErrUnknown)
//   - already consumed         → (false, ErrReplay)
//   - expired (now >= expiry)  → (false, ErrExpired)   [entry dropped]
//   - enforceBinding && mismatch → (false, errBindingMismatch)
//   - otherwise                → marks consumed, returns (true, nil)
type ConfirmationStore interface {
	Issue(entry ConfirmationStoreEntry) error
	Consume(nonce, op, actor string, enforceBinding bool) (bool, error)
	Revoke(nonce string) error
	Pending() []Confirmation
	PruneExpired()
	// SetClock injects the clock the store uses for expiry decisions. The
	// registry calls this so withClock() stays a single source of truth.
	SetClock(now func() time.Time)
}

// confirmationEntry is the internal MemoryStore row. consumed flips once
// on the first successful Consume.
type confirmationEntry struct {
	nonce     string
	op        string
	actor     string
	issuedAt  time.Time
	expiresAt time.Time
	consumed  bool
}

// ConfirmationRegistry stores issued nonces with TTL and one-time semantics.
// The zero value is NOT usable; construct via NewConfirmationRegistry or
// NewConfirmationRegistryWithStore.
type ConfirmationRegistry struct {
	mu       sync.Mutex
	store    ConfirmationStore
	ttl      time.Duration
	stopCh   chan struct{}
	stopOnce sync.Once
	now      func() time.Time
}

// NewConfirmationRegistry starts the background sweeper and returns a
// registry with the given TTL, backed by an in-process MemoryStore. Stop()
// must be called to release the goroutine; tests use t.Cleanup /
// defer reg.Stop() for that.
//
// A ttl of zero or negative is normalized to DefaultConfirmationTTL.
func NewConfirmationRegistry(ttl time.Duration) *ConfirmationRegistry {
	return NewConfirmationRegistryWithStore(ttl, NewMemoryStore())
}

// NewConfirmationRegistryWithStore is like NewConfirmationRegistry but lets
// the caller supply a durable ConfirmationStore (e.g. FileStore) so nonces
// survive process restart. The registry drives the given store's clock; a
// nil store panics (programmer error).
func NewConfirmationRegistryWithStore(ttl time.Duration, store ConfirmationStore) *ConfirmationRegistry {
	if ttl <= 0 {
		ttl = DefaultConfirmationTTL
	}
	if store == nil {
		panic("gcl: NewConfirmationRegistryWithStore: nil store")
	}
	r := &ConfirmationRegistry{
		store:  store,
		ttl:    ttl,
		stopCh: make(chan struct{}),
		now:    time.Now,
	}
	r.store.SetClock(r.now)
	go r.sweepLoop()
	return r
}

// withClock lets tests inject a fake clock. Production never calls this;
// the public API exposes a single constructor. The clock is propagated to
// the underlying store so expiry decisions stay consistent.
func (r *ConfirmationRegistry) withClock(now func() time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.now = now
	r.store.SetClock(now)
}

// Issue mints a fresh nonce bound to (op, actor) and stores it. The nonce
// is returned to the caller for distribution to the human reviewer. The
// caller does NOT need to retain the (op, actor) — it is encoded into the
// entry so future VerifyBound calls can check the binding.
//
// Idempotency note: issuing the same (op, actor) twice produces two
// independent nonces. Each can be consumed exactly once.
func (r *ConfirmationRegistry) Issue(op, actor string) (string, error) {
	nonce, err := newNonce()
	if err != nil {
		return "", fmt.Errorf("confirmation: mint nonce: %w", err)
	}
	r.mu.Lock()
	now := r.now()
	ttl := r.ttl
	store := r.store
	r.mu.Unlock()
	if err := store.Issue(ConfirmationStoreEntry{
		Nonce:     nonce,
		Op:        op,
		Actor:     actor,
		IssuedAt:  now,
		ExpiresAt: now.Add(ttl),
	}); err != nil {
		return "", err
	}
	return nonce, nil
}

// IssuePlanID is the spec-mandated (§5, gcl-trust-boundary-p0) issuer:
// returns both a fresh planID and a nonce, bound to (skill, cmd). The
// caller (e.g. the L4 Orchestrator) keeps the planID; the human
// reviewer receives the nonce and pastes it back via --confirm-nonce.
// ValidateAndConsume is the matching consumer.
func (r *ConfirmationRegistry) IssuePlanID(skill, cmd string) (planID, nonce string, err error) {
	n, err := r.Issue(skill, cmd)
	if err != nil {
		return "", "", err
	}
	return "plan_" + n, n, nil
}

// Verify consumes the nonce exactly once and returns ok=true. A second
// Verify on the same nonce returns ErrReplay and ok=false. An expired or
// unknown nonce returns its respective error.
//
// Verify ignores the (op, actor) binding. Callers needing binding-level
// enforcement should use VerifyBound.
func (r *ConfirmationRegistry) Verify(nonce string) (bool, error) {
	return r.verify(nonce, "", "", false)
}

// ValidateAndConsume is the spec-mandated (§5, gcl-trust-boundary-p0)
// API: atomically checks the nonce matches the (skill, cmd) it was
// issued against and marks it consumed. Subsequent calls return
// ErrNonceConsumed. An expired nonce returns ErrNonceExpired; an
// unknown nonce returns ErrUnknown. A binding mismatch returns
// errBindingMismatch.
func (r *ConfirmationRegistry) ValidateAndConsume(nonce, skill, cmd string) error {
	// verify returns (true, nil) on success and (false, non-nil-err) on every
	// failure path (unknown/replay/expired/binding-mismatch), so a nil error
	// already implies ok==true — no separate !ok branch is reachable.
	_, err := r.verify(nonce, skill, cmd, true)
	return err
}

func (r *ConfirmationRegistry) verify(nonce, op, actor string, enforceBinding bool) (bool, error) {
	if nonce == "" {
		return false, ErrUnknown
	}
	r.mu.Lock()
	store := r.store
	r.mu.Unlock()
	return store.Consume(nonce, op, actor, enforceBinding)
}

// Revoke forcibly removes a nonce. Useful when an operation is cancelled
// before the reviewer has decided. Returns ErrUnknown when the id is not
// present (revocation is idempotent in the user-facing sense: revoking an
// already-unknown nonce is fine).
func (r *ConfirmationRegistry) Revoke(nonce string) error {
	r.mu.Lock()
	store := r.store
	r.mu.Unlock()
	return store.Revoke(nonce)
}

// Pending returns a stable, time-sorted snapshot of issued nonces (most
// recent first). The caller must not mutate the slice.
func (r *ConfirmationRegistry) Pending() []Confirmation {
	r.mu.Lock()
	store := r.store
	r.mu.Unlock()
	return store.Pending()
}

// Stop halts the background sweeper. Safe to call multiple times.
func (r *ConfirmationRegistry) Stop() {
	r.stopOnce.Do(func() {
		close(r.stopCh)
	})
}

// sweepLoop prunes expired entries every ttl/4 (clamped to >=50ms,
// <=5s) so Pending() can't grow without bound. Each pass scans the full
// map under the registry mutex; the cost is O(n) per sweep.
func (r *ConfirmationRegistry) sweepLoop() {
	interval := r.ttl / 4
	if interval < 50*time.Millisecond {
		interval = 50 * time.Millisecond
	}
	if interval > 5*time.Second {
		interval = 5 * time.Second
	}
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-r.stopCh:
			return
		case <-tick.C:
			r.sweepOnce()
		}
	}
}

func (r *ConfirmationRegistry) sweepOnce() {
	r.mu.Lock()
	store := r.store
	r.mu.Unlock()
	store.PruneExpired()
}

// PruneExpired is the spec-mandated (§5) public sweep. It removes all
// entries whose expiresAt is in the past. The L4 Orchestrator's main
// loop calls this every minute or so to bound registry size; the
// background sweeper goroutine also calls it on a 1/4 TTL cadence.
func (r *ConfirmationRegistry) PruneExpired() {
	r.sweepOnce()
}

// newNonce returns "cfm_" + 12 hex chars (48 bits of crypto/rand entropy).
// 12 hex chars give 2^48 ≈ 2.8e14 distinct nonces — well beyond anything
// a single registry will see in a 60s window.
func newNonce() (string, error) {
	var raw [6]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return ConfirmationPrefix + hex.EncodeToString(raw[:]), nil
}
