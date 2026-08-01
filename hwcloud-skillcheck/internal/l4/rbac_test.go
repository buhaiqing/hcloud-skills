package l4

import (
	"testing"
)

func TestCheckPermission_DeleteImperative(t *testing.T) {
	// Delete operations are immutable — require approval even at L4.
	decision := CheckPermission("delete", "L4_autonomous", 0.99)
	if decision.Allowed {
		t.Error("delete should not be auto-approved even at L4_autonomous")
	}
	if decision.ImmutableConstraints == nil || len(decision.ImmutableConstraints) == 0 {
		t.Error("delete should report immutable constraints")
	}
}

func TestCheckPermission_L4Autonomous(t *testing.T) {
	// L4 with high score can auto-approve high risk.
	decision := CheckPermission("restart-instance", "L4_autonomous", 0.99)
	if !decision.Allowed {
		t.Errorf("restart-instance (medium) should be allowed at L4: %+v", decision)
	}
}

func TestCheckPermission_L3Trusted(t *testing.T) {
	// L3 can auto-approve high risk but not critical.
	highDecision := CheckPermission("detach", "L3_trusted", 0.85)
	if !highDecision.Allowed {
		t.Errorf("detach (high) should be allowed at L3: %+v", highDecision)
	}

	criticalDecision := CheckPermission("delete", "L3_trusted", 0.85)
	if criticalDecision.Allowed {
		t.Error("delete (critical) should not be allowed at L3")
	}
}

func TestCheckPermission_L2Established(t *testing.T) {
	// L2 auto-approves only medium and below.
	decision := CheckPermission("create", "L2_established", 0.65)
	if !decision.Allowed {
		t.Errorf("create (medium) should be allowed at L2: %+v", decision)
	}
}

func TestCheckPermission_L1Provisional(t *testing.T) {
	// L1 auto-approves only low risk.
	lowDecision := CheckPermission("list", "L1_provisional", 0.35)
	if !lowDecision.Allowed {
		t.Errorf("list (low) should be allowed at L1: %+v", lowDecision)
	}

	mediumDecision := CheckPermission("create", "L1_provisional", 0.35)
	if mediumDecision.Allowed {
		t.Error("create (medium) should not be allowed at L1")
	}
}

func TestCheckPermission_L0New(t *testing.T) {
	// L0 auto-approves low-risk read-only operations.
	// It should NOT auto-approve medium/high risk.
	decision := CheckPermission("describe", "L0_new", 0.0)
	if !decision.Allowed {
		t.Errorf("read-only (low) should be auto-approved at L0: %+v", decision)
	}

	// L0 should NOT auto-approve medium-risk operations.
	medium := CheckPermission("create", "L0_new", 0.0)
	if medium.Allowed {
		t.Error("create (medium) should not be auto-approved at L0")
	}
}

func TestCheckPermission_ReadOnly(t *testing.T) {
	// Read-only operations should be auto-approved at all levels.
	for _, level := range []string{"L0_new", "L1_provisional", "L2_established", "L3_trusted", "L4_autonomous"} {
		decision := CheckPermission("list", level, 0.0)
		if !decision.Allowed {
			t.Errorf("list should be auto-approved at %s: %+v", level, decision)
		}
	}
}

func TestCheckCommandPermission(t *testing.T) {
	cases := []struct {
		command    string
		trustLevel string
		score      float64
		allowed    bool
	}{
		{"hcloud rds list-instances", "L0_new", 0.0, true},
		{"hcloud rds delete-instance --id foo", "L4_autonomous", 0.99, false},
		{"hcloud rds create-instance", "L3_trusted", 0.85, true},
	}

	for _, c := range cases {
		decision := CheckCommandPermission(c.command, c.trustLevel, c.score)
		if decision.Allowed != c.allowed {
			t.Errorf("CheckCommandPermission(%q, %s, %.2f): allowed=%v, want %v — %s",
				c.command, c.trustLevel, c.score, decision.Allowed, c.allowed, decision.Reason)
		}
	}
}

func TestRequiresApproval(t *testing.T) {
	cases := []string{
		"hcloud rds delete-instance --force",
		"hcloud vpc delete-subnet --delete",
		"hcloud cce delete-cluster --destroy",
		"hcloud rds reset-password --purge",
	}

	for _, cmd := range cases {
		if !RequiresApproval(cmd) {
			t.Errorf("RequiresApproval(%q): expected true", cmd)
		}
	}

	safe := []string{
		"hcloud rds list-instances",
		"hcloud rds show-instance --id foo",
		"hcloud vpc list-vpcs",
	}

	for _, cmd := range safe {
		if RequiresApproval(cmd) {
			t.Errorf("RequiresApproval(%q): expected false", cmd)
		}
	}
}

func TestRiskOrder(t *testing.T) {
	cases := []struct {
		risk  RBACRisk
		order int
	}{
		{RBACRiskNone, 0},
		{RBACRiskLow, 1},
		{RBACRiskMedium, 2},
		{RBACRiskHigh, 3},
		{RBACRiskCritical, 4},
	}

	for _, c := range cases {
		if got := riskOrder(c.risk); got != c.order {
			t.Errorf("riskOrder(%s): got %d, want %d", c.risk, got, c.order)
		}
	}
}

func TestImmutableConstraints(t *testing.T) {
	// Verify immutable constraints are present.
	if len(ImmutableConstraints) == 0 {
		t.Error("ImmutableConstraints should not be empty")
	}

	// Verify specific immutable operations.
	for _, constraint := range ImmutableConstraints {
		t.Logf("Immutable constraint: %s", constraint)
	}
}

func TestCheckPermission_Immutables(t *testing.T) {
	// delete-security-group is immutable.
	decision := CheckPermission("delete-security-group", "L4_autonomous", 1.0)
	if decision.Allowed {
		t.Error("delete-security-group should not be allowed even at L4")
	}
	if len(decision.ImmutableConstraints) == 0 {
		t.Error("delete-security-group should have immutable constraint")
	}
}

func TestTrustToMaxRisk_ByLevel(t *testing.T) {
	cases := []struct {
		level string
		want  RBACRisk
	}{
		{"L4_autonomous", RBACRiskCritical},
		{"L3_trusted", RBACRiskHigh},
		{"L2_established", RBACRiskMedium},
		{"L1_provisional", RBACRiskLow},
		{"L0_new", RBACRiskNone},
		{"UNKNOWN", RBACRiskNone}, // default → none (only read-only bypass)
	}
	for _, c := range cases {
		if got := trustToMaxRisk(c.level, 0.5); got != c.want {
			t.Errorf("trustToMaxRisk(%s): got %s, want %s", c.level, got, c.want)
		}
	}
}

func TestTrustToMaxRisk_ScoreIndependent(t *testing.T) {
	// trustToMaxRisk maps by level only; score is a separate signal.
	for _, score := range []float64{0.0, 0.5, 1.0} {
		if got := trustToMaxRisk("L2_established", score); got != RBACRiskMedium {
			t.Errorf("trustToMaxRisk(L2, %.1f): got %s, want medium", score, got)
		}
	}
}

func TestCheckCommandPermission_EmptyCommand(t *testing.T) {
	// Empty command → no action extractable → denied (critical).
	d := CheckCommandPermission("", "L4_autonomous", 0.99)
	if d.Allowed {
		t.Error("empty command must not be allowed")
	}
	if d.Risk != RBACRiskCritical {
		t.Errorf("empty command risk=%s, want critical", d.Risk)
	}
}

func TestCheckCommandPermission_SingleToken(t *testing.T) {
	// A command with <3 tokens has no extractable action → denied.
	d := CheckCommandPermission("hcloud", "L4_autonomous", 0.99)
	if d.Allowed {
		t.Error("single-token command must not be allowed")
	}
}

func TestCheckCommandPermission_UnknownActionDefaultsHigh(t *testing.T) {
	// Unknown action (not in DefaultOperationPermissions, not a known high-risk
	// verb) defaults to high risk.
	d := CheckCommandPermission("hcloud ecs somethingObscure", "L2_established", 0.65)
	if d.Allowed {
		t.Error("unknown action defaulting to high must not be auto-approved at L2")
	}
	if d.Risk != RBACRiskHigh {
		t.Errorf("unknown action risk=%s, want high", d.Risk)
	}
}

func TestCheckCommandPermission_ReadOnlyAtAllLevels(t *testing.T) {
	// Read-only (low risk) is auto-approved at every trust level, even L0.
	for _, level := range []string{"L0_new", "L1_provisional", "L2_established", "L3_trusted", "L4_autonomous"} {
		d := CheckCommandPermission("hcloud ecs list", level, 0.0)
		if !d.Allowed {
			t.Errorf("read-only list not auto-approved at %s: %+v", level, d)
		}
	}
}

func TestCheckPermission_UnknownVerbHighRiskMatch(t *testing.T) {
	// A verb not in DefaultOperationPermissions but matching a HighRiskVerbs
	// regex is classified high risk.
	// delete is immutable; use a verb with "delete" substring that isn't a known op.
	d := CheckPermission("force-delete", "L2_established", 0.65)
	if d.Allowed {
		t.Error("force-delete should not be auto-approved at L2")
	}
}
