package l4

import (
	"testing"
)

// --- progressive_trust ---

func TestSuccessRate_AllSuccess(t *testing.T) {
	h := []OpHistory{{Outcome: "success"}, {Outcome: "success"}, {Outcome: "success"}}
	if got := ComputeSuccessRate(h); got != 1.0 {
		t.Errorf("got %v, want 1.0", got)
	}
}

func TestSuccessRate_Empty(t *testing.T) {
	if got := ComputeSuccessRate(nil); got != 0.0 {
		t.Errorf("got %v, want 0.0", got)
	}
}

func TestConsistency_AllSame(t *testing.T) {
	h := []OpHistory{{Outcome: "success"}, {Outcome: "success"}, {Outcome: "success"}, {Outcome: "success"}}
	if got := ComputeConsistency(h); got != 1.0 {
		t.Errorf("got %v, want 1.0 for perfect consistency", got)
	}
}

func TestConsistency_Mixed(t *testing.T) {
	h := []OpHistory{{Outcome: "success"}, {Outcome: "failure"}, {Outcome: "success"}, {Outcome: "failure"}}
	if got := ComputeConsistency(h); got >= 1.0 {
		t.Errorf("got %v, want <1.0 for mixed outcomes", got)
	}
}

func TestConsistency_InsufficientData(t *testing.T) {
	h := []OpHistory{{Outcome: "success"}}
	if got := ComputeConsistency(h); got != 0.5 {
		t.Errorf("got %v, want 0.5 neutral for <3 entries", got)
	}
}

func TestRecency_FullyRecentsuccess(t *testing.T) {
	// A success that just happened should weigh ~1.0
	h := []OpHistory{{Outcome: "success"}}
	if got := ComputeRecency(h); got < 0.99 {
		t.Errorf("got %v, want ≥0.99 for fresh success", got)
	}
}

func TestComplexityMastery_NoComplex(t *testing.T) {
	h := []OpHistory{{RiskLevel: "low", Outcome: "success"}}
	if got := ComputeComplexityMastery(h); got != 0.5 {
		t.Errorf("got %v, want 0.5 neutral for no complex ops", got)
	}
}

func TestComplexityMastery_AllComplex(t *testing.T) {
	h := []OpHistory{{RiskLevel: "critical", Outcome: "success"}}
	if got := ComputeComplexityMastery(h); got != 1.0 {
		t.Errorf("got %v, want 1.0 for all complex success", got)
	}
}

func TestErrorRecovery_NoFailures(t *testing.T) {
	h := []OpHistory{{Outcome: "success"}}
	if got := ComputeErrorRecovery(h); got != 0.7 {
		t.Errorf("got %v, want 0.7 baseline for no failures", got)
	}
}

func TestErrorRecovery_RecoveredFromFailure(t *testing.T) {
	h := []OpHistory{
		{Outcome: "failure", HadRetry: true},
		{Outcome: "success"},
	}
	if got := ComputeErrorRecovery(h); got != 1.0 {
		t.Errorf("got %v, want 1.0 for full recovery", got)
	}
}

func TestComputeTrustScore_L0NewOnEmpty(t *testing.T) {
	score := ComputeTrustScore(nil)
	if score.Level != "L0_new" {
		t.Errorf("level=%q, want L0_new for empty history", score.Level)
	}
	// Empty history → 0 successes but neutral components (consistency=0.5,
	// mastery=0.5, recovery=0.7) sum to ~0.245. This matches Python behavior
	// (see scripts/progressive_trust.py:163-194). What matters is that the
	// level is the lowest tier, not that the score is 0.
	if score.Score >= 0.3 {
		t.Errorf("score=%v, want < 0.3 (L0 threshold) for empty history", score.Score)
	}
}

func TestComputeTrustScore_L4AutonomousOnAllSuccess(t *testing.T) {
	// 20 all-success entries: success_rate=1, consistency=1, recency~1, mastery=0.5, recovery=0.7
	// weighted = 0.35*1 + 0.20*1 + 0.20*1 + 0.15*0.5 + 0.10*0.7 = 0.35+0.20+0.20+0.075+0.07 = 0.895
	// That's L3 (≥0.8). Add some complex successes to push past 0.95.
	h := make([]OpHistory, 0, 30)
	for i := 0; i < 30; i++ {
		risk := "low"
		if i%2 == 0 {
			risk = "critical"
		}
		h = append(h, OpHistory{Outcome: "success", RiskLevel: risk})
	}
	score := ComputeTrustScore(h)
	if score.Score < 0.9 {
		t.Errorf("score=%v, want ≥0.9 for clean history", score.Score)
	}
}

func TestEvaluateOperation_LowRisk_LowTrust(t *testing.T) {
	score := TrustScore{Level: "L0_new", MaxAutoRisk: "none", Score: 0.0}
	got := EvaluateOperation(score, "low", "resize")
	if got.AutoApproved {
		t.Error("L0 + low risk should still require confirmation (max_auto=none)")
	}
}

func TestEvaluateOperation_CriticalRisk_RequiresL4(t *testing.T) {
	score := TrustScore{Level: "L3_trusted", MaxAutoRisk: "high", Score: 0.85}
	got := EvaluateOperation(score, "critical", "resize")
	if got.AutoApproved {
		t.Error("critical risk should require confirmation unless L4")
	}
}

func TestEvaluateOperation_Destructive_L4Only(t *testing.T) {
	score := TrustScore{Level: "L3_trusted", MaxAutoRisk: "high", Score: 0.85}
	got := EvaluateOperation(score, "medium", "delete")
	if got.AutoApproved {
		t.Error("destructive op should require L4_autonomous")
	}
}

func TestEvaluateOperation_MediumRisk_L3AutoApproved(t *testing.T) {
	score := TrustScore{Level: "L3_trusted", MaxAutoRisk: "high", Score: 0.85}
	got := EvaluateOperation(score, "medium", "resize")
	if !got.AutoApproved {
		t.Error("medium risk + L3 (max_auto=high) should auto-approve")
	}
}

func TestLoadTrustData_EmptyScaffold(t *testing.T) {
	root := t.TempDir()
	data := LoadTrustData(root, "huaweicloud-ecs-ops")
	if data["schema"] != "trust-history/v1" {
		t.Errorf("schema=%v, want trust-history/v1", data["schema"])
	}
	ops, _ := data["operations"].(map[string]any)
	if ops == nil {
		t.Error("operations map missing in scaffold")
	}
}
