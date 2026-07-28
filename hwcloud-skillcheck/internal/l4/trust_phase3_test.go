package l4

import "testing"

// TestComputeTrustScoreFromOpHistory_DeprecationMarker verifies the
// Phase 3 wrapper:
//   - delegates to the deprecated ComputeTrustScore (same numeric score),
//   - bumps DeprecationCount exactly once per call,
//   - tolerates a nil DefaultTrustSource without panicking.
//
// Phase 4 removes the wrapper; this test then compiles only if both
// ComputeTrustScore and OpHistory are gone.
func TestComputeTrustScoreFromOpHistory_DeprecationMarker(t *testing.T) {
	prev := DefaultTrustSource
	t.Cleanup(func() { DefaultTrustSource = prev })
	DefaultTrustSource = &TrustSourceCounter{}

	h := []OpHistory{
		{Outcome: "success", RiskLevel: "low"},
		{Outcome: "success", RiskLevel: "low"},
		{Outcome: "success", RiskLevel: "low"},
	}
	got := ComputeTrustScoreFromOpHistory(h)
	if got.Score <= 0 {
		t.Errorf("score=%v, want >0 for all-success legacy history", got.Score)
	}
	if DefaultTrustSource.DeprecationCount.Load() != 1 {
		t.Errorf("DeprecationCount=%d, want 1 after one wrapper call",
			DefaultTrustSource.DeprecationCount.Load())
	}

	// nil counter must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("nil counter panicked: %v", r)
		}
	}()
	DefaultTrustSource = nil
	_ = ComputeTrustScoreFromOpHistory(h)
}

// TestTrustSourceCounter_DeprecationCount verifies the new atomic
// counter is independent of FromOutcomeMemory / FromOpHistory and that
// RecordDeprecation is safe on a nil receiver.
func TestTrustSourceCounter_DeprecationCount(t *testing.T) {
	c := &TrustSourceCounter{}
	c.RecordDeprecation()
	c.RecordDeprecation()
	c.RecordDeprecation()
	if got := c.DeprecationCount.Load(); got != 3 {
		t.Errorf("DeprecationCount=%d, want 3", got)
	}
	// Record / RecordDeprecation do not leak into the per-source counters.
	if c.FromOutcomeMemory.Load() != 0 || c.FromOpHistory.Load() != 0 {
		t.Errorf("source counters moved: outcome=%d op=%d, want 0/0",
			c.FromOutcomeMemory.Load(), c.FromOpHistory.Load())
	}

	// nil receiver must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("nil counter RecordDeprecation panicked: %v", r)
		}
	}()
	var nilC *TrustSourceCounter
	nilC.RecordDeprecation()
}