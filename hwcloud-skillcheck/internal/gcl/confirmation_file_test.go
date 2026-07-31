package gcl

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// newFileStoreT is a test helper that builds a FileStore under a temp dir
// and fails the test on construction error.
func newFileStoreT(t *testing.T, baseDir string) *FileStore {
	t.Helper()
	s, err := NewFileStore(baseDir)
	if err != nil {
		t.Fatalf("NewFileStore(%q): %v", baseDir, err)
	}
	return s
}

// TestFileStore_SurvivesRestart is the headline durability test: a nonce
// issued into one FileStore is still consumable by a *fresh* FileStore
// loading the same on-disk path (simulating a process restart).
func TestFileStore_SurvivesRestart(t *testing.T) {
	dir := t.TempDir()

	store1 := newFileStoreT(t, dir)
	reg1 := NewConfirmationRegistryWithStore(60*time.Second, store1)
	nonce, err := reg1.Issue("op:delete ecs-abc12345", "alice")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	reg1.Stop()

	// Confirm the file actually exists on disk.
	path := filepath.Join(dir, confirmationDirName, confirmationFileName)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected persisted file at %s: %v", path, err)
	}

	// "Restart": brand-new store + registry over the same path.
	store2 := newFileStoreT(t, dir)
	reg2 := NewConfirmationRegistryWithStore(60*time.Second, store2)
	defer reg2.Stop()

	ok, err := reg2.Verify(nonce)
	if err != nil {
		t.Fatalf("Verify after restart: %v", err)
	}
	if !ok {
		t.Fatal("Verify after restart returned false — nonce did not survive")
	}
}

// TestFileStore_ReplaySurvivesRestart proves one-time consumption is durable:
// a nonce consumed before restart must be rejected as a replay afterwards.
func TestFileStore_ReplaySurvivesRestart(t *testing.T) {
	dir := t.TempDir()

	store1 := newFileStoreT(t, dir)
	reg1 := NewConfirmationRegistryWithStore(60*time.Second, store1)
	nonce, err := reg1.Issue("op:destructive", "bob")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if ok, err := reg1.Verify(nonce); err != nil || !ok {
		t.Fatalf("first Verify: ok=%v err=%v", ok, err)
	}
	reg1.Stop()

	// Restart and replay — must fail-closed.
	store2 := newFileStoreT(t, dir)
	reg2 := NewConfirmationRegistryWithStore(60*time.Second, store2)
	defer reg2.Stop()

	ok, err := reg2.Verify(nonce)
	if ok {
		t.Fatal("replay after restart returned ok=true — must be false")
	}
	if !errors.Is(err, ErrReplay) {
		t.Fatalf("expected ErrReplay after restart, got %v", err)
	}
}

// TestFileStore_TTLExpiryConsume checks Consume rejects an entry whose TTL
// has elapsed (fake clock advanced past expiry), returning ErrExpired.
func TestFileStore_TTLExpiryConsume(t *testing.T) {
	dir := t.TempDir()
	store := newFileStoreT(t, dir)
	reg := NewConfirmationRegistryWithStore(60*time.Second, store)
	defer reg.Stop()

	nonce, err := reg.Issue("huaweicloud-ecs-ops", "delete")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	// Advance the clock past the TTL for both registry and store.
	reg.withClock(func() time.Time { return time.Now().Add(61 * time.Second) })

	err = reg.ValidateAndConsume(nonce, "huaweicloud-ecs-ops", "delete")
	if !errors.Is(err, ErrNonceExpired) {
		t.Fatalf("expected ErrNonceExpired, got %v", err)
	}
}

// TestFileStore_LoadDropsExpired ensures the load path drops entries whose
// expiry is already in the past: a short-TTL nonce persisted, then reloaded
// after real-time expiry, must be gone (Verify → ErrUnknown, not ErrExpired).
func TestFileStore_LoadDropsExpired(t *testing.T) {
	dir := t.TempDir()

	store1 := newFileStoreT(t, dir)
	// Issue directly through the store with an already-past expiry to make
	// the on-disk file contain a stale entry deterministically.
	past := time.Now().Add(-time.Hour)
	if err := store1.Issue(ConfirmationStoreEntry{
		Nonce:     "cfm_stale0000000",
		Op:        "op:x",
		Actor:     "y",
		IssuedAt:  past.Add(-time.Minute),
		ExpiresAt: past,
	}); err != nil {
		t.Fatalf("Issue stale: %v", err)
	}

	// Reload: the stale entry must be dropped on load.
	store2 := newFileStoreT(t, dir)
	if _, err := store2.Consume("cfm_stale0000000", "", "", false); !errors.Is(err, ErrUnknown) {
		t.Fatalf("expected ErrUnknown for expired-and-dropped entry, got %v", err)
	}
	if p := store2.Pending(); len(p) != 0 {
		t.Fatalf("Pending should be empty after dropping expired, got %d", len(p))
	}
}

// TestFileStore_RevokeSurvivesRestart confirms revocation is durable.
func TestFileStore_RevokeSurvivesRestart(t *testing.T) {
	dir := t.TempDir()

	store1 := newFileStoreT(t, dir)
	reg1 := NewConfirmationRegistryWithStore(60*time.Second, store1)
	nonce, err := reg1.Issue("op:cancelme", "u")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if err := reg1.Revoke(nonce); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	reg1.Stop()

	store2 := newFileStoreT(t, dir)
	reg2 := NewConfirmationRegistryWithStore(60*time.Second, store2)
	defer reg2.Stop()

	if _, err := reg2.Verify(nonce); !errors.Is(err, ErrUnknown) {
		t.Fatalf("expected ErrUnknown for revoked nonce after restart, got %v", err)
	}
}

// TestFileStore_ConcurrentNoCorruption fires concurrent Issue+Consume at a
// single FileStore and then reloads to prove the JSON on disk is still valid
// (parseable) — i.e. atomic writes never left a torn file. Run with -race.
func TestFileStore_ConcurrentNoCorruption(t *testing.T) {
	dir := t.TempDir()
	store := newFileStoreT(t, dir)
	reg := NewConfirmationRegistryWithStore(60*time.Second, store)
	defer reg.Stop()

	const n = 64
	var wg sync.WaitGroup
	wg.Add(n)
	var verifyOK int32
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			nonce, err := reg.Issue("op:scale-in", "stress")
			if err != nil {
				t.Errorf("Issue: %v", err)
				return
			}
			ok, err := reg.Verify(nonce)
			if err == nil && ok {
				atomic.AddInt32(&verifyOK, 1)
			}
		}()
	}
	wg.Wait()

	if verifyOK != n {
		t.Errorf("expected %d successful verifies, got %d", n, verifyOK)
	}

	// Reload must succeed (file is valid JSON, not torn).
	if _, err := NewFileStore(dir); err != nil {
		t.Fatalf("reload after concurrent writes failed — file may be corrupt: %v", err)
	}
}

// TestFileStore_MissingFileStartsEmpty confirms a fresh dir yields an empty,
// error-free store.
func TestFileStore_MissingFileStartsEmpty(t *testing.T) {
	dir := t.TempDir()
	store := newFileStoreT(t, dir)
	if p := store.Pending(); len(p) != 0 {
		t.Fatalf("fresh store should be empty, got %d pending", len(p))
	}
}

// TestFileStore_CorruptFileErrors confirms a malformed JSON file surfaces as
// a construction error rather than silently starting empty.
func TestFileStore_CorruptFileErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, confirmationDirName, confirmationFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}
	if _, err := NewFileStore(dir); err == nil {
		t.Fatal("expected error loading corrupt file, got nil")
	}
}

// TestStoreParity runs the same core scenarios against MemoryStore and
// FileStore to prove behavioural equivalence on the paths that matter:
// issue→consume, replay, unknown, binding mismatch, revoke.
func TestStoreParity(t *testing.T) {
	dir := t.TempDir()
	backends := map[string]ConfirmationStore{
		"memory": NewMemoryStore(),
		"file":   newFileStoreT(t, dir),
	}
	for name, store := range backends {
		t.Run(name, func(t *testing.T) {
			reg := NewConfirmationRegistryWithStore(60*time.Second, store)
			defer reg.Stop()

			// Happy path: issue → consume once.
			nonce, err := reg.Issue("huaweicloud-ecs-ops", "delete")
			if err != nil {
				t.Fatalf("Issue: %v", err)
			}
			if err := reg.ValidateAndConsume(nonce, "huaweicloud-ecs-ops", "delete"); err != nil {
				t.Fatalf("first consume: %v", err)
			}
			// Replay → ErrNonceConsumed.
			if err := reg.ValidateAndConsume(nonce, "huaweicloud-ecs-ops", "delete"); !errors.Is(err, ErrNonceConsumed) {
				t.Fatalf("replay: expected ErrNonceConsumed, got %v", err)
			}
			// Unknown nonce.
			if ok, err := reg.Verify("cfm_nope00000000"); ok || !errors.Is(err, ErrUnknown) {
				t.Fatalf("unknown: ok=%v err=%v", ok, err)
			}
			// Binding mismatch.
			nonce2, err := reg.Issue("skillA", "cmdA")
			if err != nil {
				t.Fatalf("Issue 2: %v", err)
			}
			if err := reg.ValidateAndConsume(nonce2, "skillB", "cmdB"); err == nil {
				t.Fatal("binding mismatch must fail")
			}
			// nonce2 was NOT consumed by the mismatched attempt — correct binding still works.
			if err := reg.ValidateAndConsume(nonce2, "skillA", "cmdA"); err != nil {
				t.Fatalf("consume after mismatch (correct binding): %v", err)
			}
			// Revoke path.
			nonce3, err := reg.Issue("op:z", "u")
			if err != nil {
				t.Fatalf("Issue 3: %v", err)
			}
			if err := reg.Revoke(nonce3); err != nil {
				t.Fatalf("Revoke: %v", err)
			}
			if _, err := reg.Verify(nonce3); !errors.Is(err, ErrUnknown) {
				t.Fatalf("verify revoked: expected ErrUnknown, got %v", err)
			}
		})
	}
}
