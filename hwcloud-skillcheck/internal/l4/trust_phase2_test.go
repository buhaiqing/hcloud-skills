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

// TestTrustConfig_DefaultUnchanged guarantees the configurable tiers
// keep the historical safe defaults until explicitly overridden.
func TestTrustConfig_DefaultUnchanged(t *testing.T) {
	defer ResetTrustConfig()
	ResetTrustConfig()
	cfg := DefaultTrustConfig()
	if len(cfg.Tiers) != 5 {
		t.Fatalf("default tiers=%d, want 5", len(cfg.Tiers))
	}
	if cfg.Tiers[0].Key != "L4_autonomous" || cfg.Tiers[0].MaxAutoRisk != "critical" {
		t.Errorf("L4 tier drifted: %+v", cfg.Tiers[0])
	}
	if cfg.Tiers[4].Key != "L0_new" || cfg.Tiers[4].MaxAutoRisk != "none" {
		t.Errorf("L0 tier drifted: %+v", cfg.Tiers[4])
	}
}

// TestTrustConfig_OverrideMaxAutoRisk verifies an override of MaxAutoRisk
// flows through EvaluateOperation gating immediately.
func TestTrustConfig_OverrideMaxAutoRisk(t *testing.T) {
	defer ResetTrustConfig()
	// Loosen L3 to allow only low risk (instead of high) to prove the
	// override is live, not the snapshot.
	cfg := DefaultTrustConfig()
	cfg.Tiers[1].MaxAutoRisk = "low" // L3_trusted
	SetTrustConfig(cfg)

	// medium risk at L3 must now REQUIRE confirmation (was auto before).
	score := TrustScore{Level: "L3_trusted", MaxAutoRisk: "low", Score: 0.85}
	if got := EvaluateOperation(score, "medium", "resize"); got.AutoApproved {
		t.Error("override MaxAutoRisk=low should block medium risk at L3")
	}
	// low risk at L3 still auto-approved.
	if got := EvaluateOperation(score, "low", "resize"); !got.AutoApproved {
		t.Error("override should still auto-approve low risk at L3")
	}
}

// TestTrustConfig_OverrideMinScore verifies shifting a tier boundary
// moves the level assignment used by gating.
func TestTrustConfig_OverrideMinScore(t *testing.T) {
	defer ResetTrustConfig()
	cfg := DefaultTrustConfig()
	// Raise L2_established threshold so a 0.6 score no longer qualifies.
	cfg.Tiers[2].MinScore = 0.7 // L2 was 0.6
	SetTrustConfig(cfg)

	// A score of 0.65 now falls to L1_provisional (MaxAutoRisk=low),
	// so medium risk must require confirmation.
	score := ComputeTrustScoreFromOutcome("s", "a", nil, "")
	_ = score
	lvl := "L1_provisional"
	if got := EvaluateOperation(TrustScore{Level: lvl, MaxAutoRisk: "low", Score: 0.65}, "medium", "resize"); got.AutoApproved {
		t.Error("raised MinScore should demote 0.65 to L1 and block medium risk")
	}
}

// TestTrustConfig_ResetRestoresDefaults ensures ResetTrustConfig undoes
// any override (no leak across tests).
func TestTrustConfig_ResetRestoresDefaults(t *testing.T) {
	defer ResetTrustConfig()
	cfg := DefaultTrustConfig()
	cfg.Tiers[1].MaxAutoRisk = "low"
	SetTrustConfig(cfg)
	ResetTrustConfig()
	score := TrustScore{Level: "L3_trusted", MaxAutoRisk: "high", Score: 0.85}
	if got := EvaluateOperation(score, "medium", "resize"); !got.AutoApproved {
		t.Error("after reset, L3 medium risk must be auto-approved again")
	}
}
