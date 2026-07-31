package l4

import (
	"testing"
	"time"
)

// TestOpHistory_CompletelyRemoved is the Phase 4 regression marker
// (ADR-0009 §Migration). OpHistory, ComputeTrustScore([]OpHistory),
// ComputeTrustScoreFromOpHistory, ComputeSuccessRate/Consistency/Recency/
// ComplexityMastery/ErrorRecovery, opHistorySlice, LoadTrustData,
// SaveTrustData, FromOpHistory, DeprecationCount, and RecordDeprecation
// are all gone. The compile is the actual test: any leftover
// reference would fail this package's build.
//
// We keep this empty test as a documentary marker so future
// contributors see the rule.
func TestOpHistory_CompletelyRemoved(t *testing.T) {}

// seedConsecutiveSuccesses records k trailing successes for (skill, action)
// so consecutiveSuccessCount returns exactly k. A failure before them would
// break the streak, so we only append successes (newest last).
func seedConsecutiveSuccesses(t *testing.T, mem *OutcomeMemory, skill, action string, k int) {
	t.Helper()
	now := time.Now().UTC()
	for i := 0; i < k; i++ {
		ts := now.Add(-time.Duration(k-i) * time.Minute).Format(time.RFC3339)
		if err := mem.Record(OutcomeRecord{
			ID:        skill + action + ts,
			Timestamp: ts,
			Skill:     skill,
			Action:    action,
			Outcome:   "success",
			Risk:      "low",
		}); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}
}

// TestColdStart_Ramp asserts the linear supervision decay: as consecutive
// successes rise, the allowed risk tier widens until the exploration
// window completes and normal tier gating resumes.
func TestColdStart_Ramp(t *testing.T) {
	defer ResetColdStartConfig()
	const skill, action = "huaweicloud-ecs-ops", "restart-instance"

	cases := []struct {
		k        int
		opRisk   string
		wantAuto bool
	}{
		{0, "low", false},      // k<2: always confirm
		{1, "low", false},      // k<2: always confirm
		{2, "low", true},       // k in [2,3): cap=low -> low allowed
		{2, "medium", false},   // k in [2,3): cap=low -> medium blocked
		{3, "medium", true},    // k in [3,window): cap=medium -> medium allowed
		{3, "high", false},     // cap=medium -> high blocked
		{5, "high", true},      // k>=window: fall through to tier (L3 allows high)
	}
	for _, c := range cases {
		ResetColdStartConfig()
		dir := t.TempDir()
		mem, err := NewOutcomeMemory(dir)
		if err != nil {
			t.Fatalf("NewOutcomeMemory: %v", err)
		}
		seedConsecutiveSuccesses(t, mem, skill, action, c.k)
		// L3_trusted baseline so the non-cold-start path would auto-approve
		// medium/high; cold-start must still tighten below the window.
		score := TrustScore{Level: "L3_trusted", MaxAutoRisk: "high", Score: 0.85}
		got := EvaluateOperationWithHistory(score, skill, action, c.opRisk, "restart-instance", mem)
		if got.AutoApproved != c.wantAuto {
			t.Errorf("k=%d risk=%s: AutoApproved=%v, want %v (reason=%q)", c.k, c.opRisk, got.AutoApproved, c.wantAuto, got.Reason)
		}
	}
}

// TestColdStart_FirstExecutionBlocked asserts acceptance criterion 1:
// a brand-new skill/operation with zero history is never auto-approved,
// even at a high trust level.
func TestColdStart_FirstExecutionBlocked(t *testing.T) {
	defer ResetColdStartConfig()
	dir := t.TempDir()
	mem, _ := NewOutcomeMemory(dir) // empty: no successes
	score := TrustScore{Level: "L3_trusted", MaxAutoRisk: "high", Score: 0.85}
	got := EvaluateOperationWithHistory(score, "huaweicloud-ecs-ops", "restart-instance", "low", "restart-instance", mem)
	if got.AutoApproved {
		t.Error("cold-start with 0 successes must require confirmation even at L3")
	}
}

// TestColdStart_MatureSkillUnaffected asserts the ramp does not tighten
// an already-mature skill (>= window successes): normal tier gating wins.
func TestColdStart_MatureSkillUnaffected(t *testing.T) {
	defer ResetColdStartConfig()
	dir := t.TempDir()
	mem, _ := NewOutcomeMemory(dir)
	seedConsecutiveSuccesses(t, mem, "huaweicloud-ecs-ops", "list-servers", 10) // >= window
	score := TrustScore{Level: "L2_established", MaxAutoRisk: "medium", Score: 0.7}
	// L2 allows medium; with >=window successes cold-start must not block it.
	got := EvaluateOperationWithHistory(score, "huaweicloud-ecs-ops", "list-servers", "medium", "list-servers", mem)
	if !got.AutoApproved {
		t.Errorf("mature skill (>=window successes) should not be cold-start-blocked: %q", got.Reason)
	}
}

// TestColdStart_ConfigOverride verifies the window is overridable and the
// default (5) reasserts on reset.
func TestColdStart_ConfigOverride(t *testing.T) {
	defer ResetColdStartConfig()
	if DefaultColdStartConfig().ExplorationWindow != 5 {
		t.Errorf("default window=%d, want 5 (reuse HealingPolicy.MinSamples provenance)",
			DefaultColdStartConfig().ExplorationWindow)
	}
	SetColdStartConfig(ColdStartConfig{ExplorationWindow: 2})
	if coldStartConfig.ExplorationWindow != 2 {
		t.Error("SetColdStartConfig did not apply")
	}
	ResetColdStartConfig()
	if coldStartConfig.ExplorationWindow != 5 {
		t.Error("ResetColdStartConfig did not restore default")
	}
}
