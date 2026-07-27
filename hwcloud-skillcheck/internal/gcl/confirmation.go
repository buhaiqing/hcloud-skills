// Package gcl — confirmation.go
//
// ConfirmationRegistry implements an in-memory nonce store for destructive-
// operation approvals. The GCL loop hands a fresh nonce to a human reviewer
// when a request scores as destructive; the reviewer calls Verify (or
// VerifyBound when the caller can supply the original intent+actor) to
// consume the nonce exactly once.
//
// Design choices (matching the spec in docs/gcl-spec.md §confirmations):
//
//   - Storage is in-process: nonces never survive process restart. The
//     producer (Runner) and consumer (review tool) MUST be the same process.
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
	"sort"
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

// ConfirmationRegistry stores issued nonces with TTL and one-time semantics.
// The zero value is NOT usable; construct via NewConfirmationRegistry.
type ConfirmationRegistry struct {
	mu       sync.Mutex
	entries  map[string]*confirmationEntry
	ttl      time.Duration
	stopCh   chan struct{}
	stopOnce sync.Once
	now      func() time.Time
}

// confirmationEntry is the internal store row. consumed flips once on the
// first successful Verify.
type confirmationEntry struct {
	nonce     string
	op        string
	actor     string
	issuedAt  time.Time
	expiresAt time.Time
	consumed  bool
}

// NewConfirmationRegistry starts the background sweeper and returns a
// registry with the given TTL. Stop() must be called to release the
// goroutine; tests use t.Cleanup / defer reg.Stop() for that.
//
// A ttl of zero or negative is normalized to DefaultConfirmationTTL.
func NewConfirmationRegistry(ttl time.Duration) *ConfirmationRegistry {
	if ttl <= 0 {
		ttl = DefaultConfirmationTTL
	}
	r := &ConfirmationRegistry{
		entries: make(map[string]*confirmationEntry),
		ttl:     ttl,
		stopCh:  make(chan struct{}),
		now:     time.Now,
	}
	go r.sweepLoop()
	return r
}

// withClock lets tests inject a fake clock. Production never calls this;
// the public API exposes a single constructor.
func (r *ConfirmationRegistry) withClock(now func() time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.now = now
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
	now := r.now()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[nonce] = &confirmationEntry{
		nonce:     nonce,
		op:        op,
		actor:     actor,
		issuedAt:  now,
		expiresAt: now.Add(r.ttl),
		consumed:  false,
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
	ok, err := r.verify(nonce, skill, cmd, true)
	if err != nil {
		return err
	}
	if !ok {
		return errBindingMismatch
	}
	return nil
}

func (r *ConfirmationRegistry) verify(nonce, op, actor string, enforceBinding bool) (bool, error) {
	if nonce == "" {
		return false, ErrUnknown
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.entries[nonce]
	if !ok {
		return false, ErrUnknown
	}
	if entry.consumed {
		return false, ErrReplay
	}
	if !r.now().Before(entry.expiresAt) {
		// Eagerly remove the stale entry so Pending() never returns it.
		delete(r.entries, nonce)
		return false, ErrExpired
	}
	if enforceBinding && (entry.op != op || entry.actor != actor) {
		return false, errBindingMismatch
	}
	entry.consumed = true
	return true, nil
}

// Revoke forcibly removes a nonce. Useful when an operation is cancelled
// before the reviewer has decided. Returns ErrUnknown when the id is not
// present (revocation is idempotent in the user-facing sense: revoking an
// already-unknown nonce is fine).
func (r *ConfirmationRegistry) Revoke(nonce string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[nonce]; !ok {
		return ErrUnknown
	}
	delete(r.entries, nonce)
	return nil
}

// Pending returns a stable, time-sorted snapshot of issued nonces (most
// recent first). The caller must not mutate the slice.
func (r *ConfirmationRegistry) Pending() []Confirmation {
	r.mu.Lock()
	now := r.now()
	out := make([]Confirmation, 0, len(r.entries))
	for _, e := range r.entries {
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
	r.mu.Unlock()
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].IssuedAt.After(out[j].IssuedAt)
	})
	return out
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
	defer r.mu.Unlock()
	now := r.now()
	for k, e := range r.entries {
		if !now.Before(e.expiresAt) {
			delete(r.entries, k)
		}
	}
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
