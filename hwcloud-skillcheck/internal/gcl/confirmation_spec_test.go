package gcl

import (
	"errors"
	"testing"
	"time"
)

// TestConfirmationRegistryOneTime is the spec-mandated (A8) test for the
// one-time-consumption contract. Uses the spec-mandated public API:
// ValidateAndConsume + ErrNonceConsumed.
func TestConfirmationRegistryOneTime(t *testing.T) {
	reg := NewConfirmationRegistry(60 * time.Second)
	defer reg.Stop()

	// Issue binds to (skill="huaweicloud-ecs-ops", cmd="delete").
	nonce, err := reg.Issue("huaweicloud-ecs-ops", "delete")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// First ValidateAndConsume must succeed.
	if err := reg.ValidateAndConsume(nonce, "huaweicloud-ecs-ops", "delete"); err != nil {
		t.Fatalf("first ValidateAndConsume: %v", err)
	}

	// Second call must return ErrNonceConsumed.
	err = reg.ValidateAndConsume(nonce, "huaweicloud-ecs-ops", "delete")
	if err == nil {
		t.Fatal("second ValidateAndConsume must fail (replay)")
	}
	if !errors.Is(err, ErrNonceConsumed) {
		t.Fatalf("expected ErrNonceConsumed, got %v", err)
	}
}

// TestConfirmationRegistryTTL is the spec-mandated (A9) test for the
// TTL contract. Uses the spec-mandated public API: ValidateAndConsume
// + ErrNonceExpired. Test injects a fake clock so we don't sleep 60s.
func TestConfirmationRegistryTTL(t *testing.T) {
	reg := NewConfirmationRegistry(60 * time.Second)
	defer reg.Stop()

	nonce, err := reg.Issue("huaweicloud-ecs-ops", "delete")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// Advance the registry's clock past the TTL.
	reg.withClock(func() time.Time { return time.Now().Add(61 * time.Second) })

	err = reg.ValidateAndConsume(nonce, "huaweicloud-ecs-ops", "delete")
	if err == nil {
		t.Fatal("expired nonce must be rejected")
	}
	if !errors.Is(err, ErrNonceExpired) {
		t.Fatalf("expected ErrNonceExpired, got %v", err)
	}
}
