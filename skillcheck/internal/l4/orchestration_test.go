// Package l4 holds the L4 runtime engines: orchestration, topology, predictive,
// trust. The runtime_orchestrator wires them together. These tests pin the
// public contracts of each engine so the Python script can be deleted without
// behavior drift.
package l4

import (
	"strings"
	"testing"
)

// --- orchestration ---

func TestMatchFaultSkills_NetworkFault(t *testing.T) {
	matches := MatchFaultSkills("ECS instance unreachable", nil)
	if len(matches) == 0 {
		t.Fatal("network fault should match at least one skill")
	}
	// vpc-ops is the priority for connectivity keywords.
	var hasVPC bool
	for _, m := range matches {
		if m.Skill == "huaweicloud-vpc-ops" {
			hasVPC = true
			if m.Confidence <= 0 || m.Confidence > 1 {
				t.Errorf("confidence out of range: %v", m.Confidence)
			}
		}
	}
	if !hasVPC {
		t.Error("expected vpc-ops in matches for connectivity fault")
	}
}

func TestMatchFaultSkills_PermissionFault(t *testing.T) {
	matches := MatchFaultSkills("permission denied on delete", nil)
	var hasIAM bool
	for _, m := range matches {
		if m.Skill == "huaweicloud-iam-ops" {
			hasIAM = true
		}
	}
	if !hasIAM {
		t.Error("expected iam-ops for permission denied fault")
	}
}

func TestMatchFaultSkills_NoMatch(t *testing.T) {
	matches := MatchFaultSkills("xxxxxxxxxxxxx", nil)
	if len(matches) != 0 {
		t.Errorf("unrelated fault should yield zero matches, got %d", len(matches))
	}
}

func TestMatchFaultSkills_AvailableFilter(t *testing.T) {
	// Without iam-ops in available, permission fault should not surface it.
	matches := MatchFaultSkills("permission denied", []string{"huaweicloud-ecs-ops"})
	for _, m := range matches {
		if m.Skill == "huaweicloud-iam-ops" {
			t.Error("iam-ops filtered out by available list but still matched")
		}
	}
}

func TestDiscoverTransitiveSkills_BFS(t *testing.T) {
	got := DiscoverTransitiveSkills([]string{"huaweicloud-ecs-ops"})
	// ecs-ops delegates to vpc/ces/elb; vpc-ops delegates back to ecs/elb; ces delegates to nothing.
	for _, want := range []string{"huaweicloud-vpc-ops", "huaweicloud-ces-ops", "huaweicloud-elb-ops"} {
		found := false
		for _, s := range got {
			if s == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("BFS missing %s, got %v", want, got)
		}
	}
}

func TestSelectStrategy_SingleSequential(t *testing.T) {
	if got := SelectStrategy(1, false); got != "sequential" {
		t.Errorf("SelectStrategy(1, false) = %q, want sequential", got)
	}
}

func TestSelectStrategy_PipelineWithDeps(t *testing.T) {
	if got := SelectStrategy(2, true); got != "pipeline" {
		t.Errorf("SelectStrategy(2, true) = %q, want pipeline", got)
	}
}

func TestSelectStrategy_ParallelFew(t *testing.T) {
	if got := SelectStrategy(3, false); got != "parallel" {
		t.Errorf("SelectStrategy(3, false) = %q, want parallel", got)
	}
}

func TestSelectStrategy_FanOutMany(t *testing.T) {
	if got := SelectStrategy(5, false); got != "fan_out_collect" {
		t.Errorf("SelectStrategy(5, false) = %q, want fan_out_collect", got)
	}
}

func TestBuildExecutionPlan_PipelineOrderedByDomain(t *testing.T) {
	skills := []MatchedSkill{
		{Skill: "huaweicloud-rds-ops", Confidence: 0.8, Domain: "database"},
		{Skill: "huaweicloud-ces-ops", Confidence: 0.9, Domain: "monitoring"},
		{Skill: "huaweicloud-vpc-ops", Confidence: 0.7, Domain: "network"},
	}
	plan := BuildExecutionPlan("db conn timeout", skills, "pipeline")
	if plan.Strategy != "pipeline" {
		t.Errorf("plan.strategy=%q, want pipeline", plan.Strategy)
	}
	if len(plan.Steps) != 3 {
		t.Fatalf("plan has %d steps, want 3", len(plan.Steps))
	}
	// Pipeline order: monitoring → network → database (domain_order 0,2,4)
	if plan.Steps[0].Skill != "huaweicloud-ces-ops" {
		t.Errorf("step[0]=%s, want ces-ops (monitoring first)", plan.Steps[0].Skill)
	}
	if plan.Steps[1].Skill != "huaweicloud-vpc-ops" {
		t.Errorf("step[1]=%s, want vpc-ops", plan.Steps[1].Skill)
	}
	if plan.Steps[2].Skill != "huaweicloud-rds-ops" {
		t.Errorf("step[2]=%s, want rds-ops", plan.Steps[2].Skill)
	}
	// Each step depends on prior in pipeline.
	if len(plan.Steps[1].DependsOn) == 0 || plan.Steps[1].DependsOn[0] != 1 {
		t.Errorf("step[1].depends_on=%v, want [1]", plan.Steps[1].DependsOn)
	}
}

func TestBuildExecutionPlan_ParallelIndependentDeps(t *testing.T) {
	skills := []MatchedSkill{
		{Skill: "huaweicloud-ecs-ops", Confidence: 0.5, Domain: "compute"},
		{Skill: "huaweicloud-ces-ops", Confidence: 0.5, Domain: "monitoring"},
	}
	plan := BuildExecutionPlan("perf slow", skills, "parallel")
	for i, s := range plan.Steps {
		if len(s.DependsOn) != 0 {
			t.Errorf("parallel step[%d].depends_on=%v, want []", i, s.DependsOn)
		}
	}
}

func TestBuildExecutionPlan_PlanIDFormat(t *testing.T) {
	plan := BuildExecutionPlan("x", []MatchedSkill{{Skill: "huaweicloud-ecs-ops"}}, "sequential")
	if !strings.HasPrefix(plan.PlanID, "orch-") {
		t.Errorf("plan_id=%q, want orch- prefix", plan.PlanID)
	}
	if len(plan.PlanID) < len("orch-")+12 {
		t.Errorf("plan_id=%q, want 12 hex chars after orch-", plan.PlanID)
	}
}
