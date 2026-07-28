package l4

import (
	"testing"
	"time"
)

// TestComputeTrustScoreFromOutcome_Empty verifies that nil mem or no records
// fall back to the same zero-history TrustScore that ComputeTrustScore(nil)
// produces (level L0_new, low composite).
func TestComputeTrustScoreFromOutcome_Empty(t *testing.T) {
	dir := t.TempDir()
	mem, err := NewOutcomeMemory(dir)
	if err != nil {
		t.Fatalf("NewOutcomeMemory: %v", err)
	}
	got := ComputeTrustScoreFromOutcome("huaweicloud-ecs-ops", "list-servers", mem, "")
	if got.Level != "L0_new" {
		t.Errorf("level=%q, want L0_new for empty outcome memory", got.Level)
	}
	// nil mem must not panic and must return the empty-history score.
	if nillScore := ComputeTrustScoreFromOutcome("s", "a", nil, ""); nillScore.Level != "L0_new" {
		t.Errorf("nil mem: level=%q, want L0_new", nillScore.Level)
	}
}

// TestComputeTrustScoreFromOutcome_AllSuccess seeds 5 success records and
// expects a high composite score (>= L2_established ≥0.6).
func TestComputeTrustScoreFromOutcome_AllSuccess(t *testing.T) {
	dir := t.TempDir()
	mem, err := NewOutcomeMemory(dir)
	if err != nil {
		t.Fatalf("NewOutcomeMemory: %v", err)
	}
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		ts := now.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339)
		if err := mem.Record(OutcomeRecord{
			ID:        "s" + ts,
			Timestamp: ts,
			Skill:     "huaweicloud-ecs-ops",
			Action:    "list-servers",
			Outcome:   "success",
			Risk:      "medium",
		}); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}
	got := ComputeTrustScoreFromOutcome("huaweicloud-ecs-ops", "list-servers", mem, "")
	if got.Score < 0.6 {
		t.Errorf("score=%v, want ≥0.6 for 5 fresh successes", got.Score)
	}
}

// TestComputeTrustScoreFromOutcome_Mixed seeds 3 success + 2 failure and
// expects a mid-range composite (0.3 ≤ score < 0.6).
func TestComputeTrustScoreFromOutcome_Mixed(t *testing.T) {
	dir := t.TempDir()
	mem, err := NewOutcomeMemory(dir)
	if err != nil {
		t.Fatalf("NewOutcomeMemory: %v", err)
	}
	now := time.Now().UTC()
	outcomes := []string{"success", "success", "success", "failure", "failure"}
	for i, oc := range outcomes {
		ts := now.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339)
		if err := mem.Record(OutcomeRecord{
			ID:        "m" + ts,
			Timestamp: ts,
			Skill:     "huaweicloud-ecs-ops",
			Action:    "list-servers",
			Outcome:   oc,
			Risk:      "medium",
		}); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}
	got := ComputeTrustScoreFromOutcome("huaweicloud-ecs-ops", "list-servers", mem, "")
	if got.Score < 0.3 || got.Score >= 0.6 {
		t.Errorf("score=%v, want mid-range [0.3, 0.6) for mixed outcomes", got.Score)
	}
}

// TestTrustSourceCounter_Record verifies the counter increments for the
// one remaining source (outcome_memory, post-Phase-4) and silently
// ignores unknown sources.
func TestTrustSourceCounter_Record(t *testing.T) {
	c := &TrustSourceCounter{}
	c.Record("outcome_memory")
	c.Record("outcome_memory")
	c.Record("op_history") // unknown post-Phase-4, must be ignored
	c.Record("unknown")
	if c.FromOutcomeMemory.Load() != 2 {
		t.Errorf("FromOutcomeMemory=%d, want 2", c.FromOutcomeMemory.Load())
	}
	// nil receiver must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("nil counter Record panicked: %v", r)
		}
	}()
	var nilC *TrustSourceCounter
	nilC.Record("outcome_memory")
}
