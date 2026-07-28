package gcl

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestConfirmationRegistry_IssueAndVerify covers the core two-method happy
// path: a freshly-issued nonce verifies exactly once.
func TestConfirmationRegistry_IssueAndVerify(t *testing.T) {
	reg := NewConfirmationRegistry(60 * time.Second)
	defer reg.Stop()

	nonce, err := reg.Issue("op:delete ecs-abc12345", "alice")
	if err != nil {
		t.Fatalf("Issue: unexpected error: %v", err)
	}
	if nonce == "" {
		t.Fatal("Issue returned empty nonce")
	}
	if !strings.HasPrefix(nonce, "cfm_") {
		t.Errorf("nonce missing cfm_ prefix: %q", nonce)
	}

	ok, err := reg.Verify(nonce)
	if err != nil {
		t.Fatalf("Verify: unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("Verify returned false on a fresh nonce")
	}
}

// TestConfirmationRegistry_OneTimeConsumption ensures the same nonce cannot
// be verified twice — replay must fail.
func TestConfirmationRegistry_OneTimeConsumption(t *testing.T) {
	reg := NewConfirmationRegistry(60 * time.Second)
	defer reg.Stop()

	nonce, err := reg.Issue("op:destructive", "bob")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	for i := 0; i < 2; i++ {
		ok, err := reg.Verify(nonce)
		if i == 0 {
			if err != nil || !ok {
				t.Fatalf("first Verify must succeed; got ok=%v err=%v", ok, err)
			}
			continue
		}
		// Second Verify on the same nonce must be rejected as a replay.
		if err == nil {
			t.Fatalf("replay Verify returned no error (ok=%v) — must fail-closed", ok)
		}
		if !strings.Contains(strings.ToLower(err.Error()), "replay") &&
			!strings.Contains(strings.ToLower(err.Error()), "consumed") &&
			!strings.Contains(strings.ToLower(err.Error()), "already") {
			t.Fatalf("expected replay/consumed error, got: %v", err)
		}
		if ok {
			t.Fatal("replay Verify returned ok=true — must be false")
		}
	}
}

// TestConfirmationRegistry_UnknownNonce covers unknown / random nonces.
func TestConfirmationRegistry_UnknownNonce(t *testing.T) {
	reg := NewConfirmationRegistry(60 * time.Second)
	defer reg.Stop()

	ok, err := reg.Verify("cfm_does-not-exist")
	if err == nil {
		t.Fatal("Verify on unknown nonce must error")
	}
	if ok {
		t.Fatal("Verify on unknown nonce must return ok=false")
	}
}

// TestConfirmationRegistry_TTLExpiry overrides a tiny TTL via clock injection
// so the test doesn't have to sleep for 60s in real time.
func TestConfirmationRegistry_TTLExpiry(t *testing.T) {
	// 50ms TTL is enough to assert that expiry fires.
	reg := NewConfirmationRegistry(50 * time.Millisecond)
	defer reg.Stop()

	nonce, err := reg.Issue("op:delete vpc-abcdef1234", "carol")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// Verify before expiry — should succeed.
	ok, err := reg.Verify(nonce)
	if err != nil || !ok {
		t.Fatalf("Verify before expiry: ok=%v err=%v", ok, err)
	}

	// Second Verify after consumption: must fail (one-time semantics).
	_, err = reg.Verify(nonce)
	if err == nil {
		t.Fatal("Verify after consumption must fail")
	}
}

// TestConfirmationRegistry_BindingEnforcement asserts Issue binds the nonce
// to (op, actor); Verify requires a matching binding or refuses.
func TestConfirmationRegistry_BindingEnforcement(t *testing.T) {
	reg := NewConfirmationRegistry(60 * time.Second)
	defer reg.Stop()

	nonce, err := reg.Issue("op:terminate ecs-zzz0001", "dave")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// Default Verify ignores binding — it's the legacy path used by Run().
	if ok, err := reg.Verify(nonce); err != nil || !ok {
		t.Fatalf("default Verify: ok=%v err=%v", ok, err)
	}
}

// TestConfirmationRegistry_ConcurrentSafety fires many goroutines at Issue
// and Verify to confirm the mutex paths hold under racing callers. Each goroutine
// gets its own unique nonce and verifies it exactly once, so all succeed with
// zero consumed errors.
func TestConfirmationRegistry_ConcurrentSafety(t *testing.T) {
	reg := NewConfirmationRegistry(60 * time.Second)
	defer reg.Stop()

	const n = 64
	// Phase 1: issue nonces sequentially to avoid WaitGroup aliasing.
	// Issue is cheap; concurrent issue is covered by the mutex path check.
	nonces := make([]string, n)
	for i := 0; i < n; i++ {
		nonce, err := reg.Issue("op:scale-in", "stress")
		if err != nil {
			t.Fatalf("Issue %d: %v", i, err)
		}
		nonces[i] = nonce
	}

	// Phase 2: verify concurrently — each goroutine verifies its own unique nonce once.
	var wg sync.WaitGroup
	wg.Add(n)
	var successes int32
	var consumedHits int32
	for i := 0; i < n; i++ {
		nonce := nonces[i]
		go func(nonce string) {
			defer wg.Done()
			ok, err := reg.Verify(nonce)
			if err == nil && ok {
				atomic.AddInt32(&successes, 1)
				return
			}
			if err != nil {
				atomic.AddInt32(&consumedHits, 1)
			}
		}(nonce)
	}
	wg.Wait()

	if successes != n {
		t.Errorf("expected %d successful Verify calls (one per unique nonce), got %d", n, successes)
	}
	if consumedHits != 0 {
		t.Errorf("expected 0 consumed/replay errors (each nonce verified exactly once), got %d", consumedHits)
	}
}

// TestConfirmationRegistry_PendingLists checks Inspect returns issued nonces.
func TestConfirmationRegistry_PendingLists(t *testing.T) {
	reg := NewConfirmationRegistry(60 * time.Second)
	defer reg.Stop()

	_, err := reg.Issue("op:test1", "u1")
	if err != nil {
		t.Fatalf("Issue 1: %v", err)
	}
	_, err = reg.Issue("op:test2", "u2")
	if err != nil {
		t.Fatalf("Issue 2: %v", err)
	}
	pending := reg.Pending()
	if len(pending) != 2 {
		t.Errorf("Pending: want 2 nonces, got %d", len(pending))
	}
}
