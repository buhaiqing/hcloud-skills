package l4

import (
	"testing"
)

// --- trust evaluation ---

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

// TestZeroTrustScore verifies the empty-history shape: components
// reflect the neutral baselines (consistency=0.5 for insufficient data,
// mastery=0.5 for no complex ops, recovery=0.7 for no failures) and
// the composite is well below the L1 threshold so L0_new is selected.
func TestZeroTrustScore(t *testing.T) {
	score := ComputeTrustScoreFromOutcome("huaweicloud-ecs-ops", "list-servers", nil, "")
	if score.Level != "L0_new" {
		t.Errorf("level=%q, want L0_new for nil mem", score.Level)
	}
	if score.Score >= 0.3 {
		t.Errorf("score=%v, want < 0.3 (L1 threshold)", score.Score)
	}
	if score.HistorySize != 0 {
		t.Errorf("HistorySize=%d, want 0", score.HistorySize)
	}
}