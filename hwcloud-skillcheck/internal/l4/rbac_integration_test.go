package l4

import (
	"strings"
	"testing"
)

// TestRBAC_Integration_HighRiskBlocked verifies the RBAC gate inside the
// execution loop blocks a destructive command: task fails with blocked_by_rbac.
func TestRBAC_Integration_HighRiskBlocked(t *testing.T) {
	root := t.TempDir()
	task := &TaskState{
		ID: "rbac-high", Status: TaskStatusRunning, CurrentStep: 0,
		Steps: []TaskStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "delete-instances", Verb: "delete", Risk: "high"}},
	}
	plan := &ExecutionPlan{Steps: []PlanStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "delete-instances"}}}
	// StubExecutor should NOT be called if RBAC blocks the step.
	exec := &StubExecutor{Outcomes: []StubStep{{ExitCode: 0}}}

	out := RunExecutionLoopWithHealing(root, task, plan, nil, nil, HealingPolicy{}, exec)
	if out.Status != TaskStatusFailed {
		t.Fatalf("want failed (RBAC blocked), got %s", out.Status)
	}
	if len(out.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(out.Results))
	}
	if out.Results[0].GCLDecision != "blocked_by_rbac" {
		t.Fatalf("want blocked_by_rbac, got %q", out.Results[0].GCLDecision)
	}
	if !strings.Contains(out.Results[0].Error, "exceeds max auto-approval") {
		t.Fatalf("want RBAC deny reason, got %q", out.Results[0].Error)
	}
}

// TestRBAC_Integration_ImmutableBlocked verifies immutable constraints block
// even a low-verb command that maps to an immutable op (e.g. delete-security-group).
func TestRBAC_Integration_ImmutableBlocked(t *testing.T) {
	root := t.TempDir()
	task := &TaskState{
		ID: "rbac-imm", Status: TaskStatusRunning, CurrentStep: 0,
		Steps: []TaskStep{{Step: 1, Skill: "huaweicloud-vpc-ops", Action: "delete-security-group", Verb: "delete", Risk: "high"}},
	}
	plan := &ExecutionPlan{Steps: []PlanStep{{Step: 1, Skill: "huaweicloud-vpc-ops", Action: "delete-security-group"}}}
	exec := &StubExecutor{Outcomes: []StubStep{{ExitCode: 0}}}

	out := RunExecutionLoopWithHealing(root, task, plan, nil, nil, HealingPolicy{}, exec)
	if out.Status != TaskStatusFailed {
		t.Fatalf("want failed (immutable), got %s", out.Status)
	}
	if len(out.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(out.Results))
	}
	if out.Results[0].GCLDecision != "blocked_by_rbac" {
		t.Fatalf("want blocked_by_rbac, got %q", out.Results[0].GCLDecision)
	}
}

// TestRBAC_Integration_ReadOnlyAllowed verifies a read-only command passes RBAC
// and executes (StubExecutor returns success).
func TestRBAC_Integration_ReadOnlyAllowed(t *testing.T) {
	root := t.TempDir()
	task := &TaskState{
		ID: "rbac-read", Status: TaskStatusRunning, CurrentStep: 0,
		Steps: []TaskStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list", Verb: "list", Risk: "low"}},
	}
	plan := &ExecutionPlan{Steps: []PlanStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list"}}}
	exec := &StubExecutor{Outcomes: []StubStep{{ExitCode: 0, Stdout: "[]"}}}

	out := RunExecutionLoopWithHealing(root, task, plan, nil, nil, HealingPolicy{}, exec)
	if out.Status != TaskStatusCompleted {
		t.Fatalf("want completed, got %s", out.Status)
	}
	if len(out.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(out.Results))
	}
	if out.Results[0].GCLDecision == "blocked_by_rbac" {
		t.Fatalf("read-only must not be blocked by RBAC, got %q", out.Results[0].GCLDecision)
	}
	if !out.Results[0].Success {
		t.Fatalf("read-only command should execute successfully, err=%v", out.Results[0].Error)
	}
}

// TestRBAC_Integration_MixedPlan verifies a plan with a read-only step (runs)
// then a destructive step (blocked) — both RBAC decisions surface in results.
func TestRBAC_Integration_MixedPlan(t *testing.T) {
	root := t.TempDir()
	task := &TaskState{
		ID: "rbac-mixed", Status: TaskStatusRunning, CurrentStep: 0,
		Steps: []TaskStep{
			{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list", Verb: "list", Risk: "low"},
			{Step: 2, Skill: "huaweicloud-ecs-ops", Action: "delete-instances", Verb: "delete", Risk: "high"},
		},
	}
	plan := &ExecutionPlan{Steps: []PlanStep{
		{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list"},
		{Step: 2, Skill: "huaweicloud-ecs-ops", Action: "delete-instances"},
	}}
	exec := &StubExecutor{Outcomes: []StubStep{{ExitCode: 0, Stdout: "[]"}}}

	out := RunExecutionLoopWithHealing(root, task, plan, nil, nil, HealingPolicy{}, exec)
	if out.Status != TaskStatusFailed {
		t.Fatalf("want failed (step2 destructive), got %s", out.Status)
	}
	if len(out.Results) != 2 {
		t.Fatalf("want 2 results (step1 ran, step2 blocked), got %d", len(out.Results))
	}
	// Step 1 (read-only) executed successfully.
	if !out.Results[0].Success {
		t.Fatalf("step1 (read-only) should execute, err=%v", out.Results[0].Error)
	}
	// Step 2 (destructive) blocked by RBAC.
	if out.Results[1].GCLDecision != "blocked_by_rbac" {
		t.Fatalf("step2 want blocked_by_rbac, got %q", out.Results[1].GCLDecision)
	}
}

// TestRiskForStep_NeverDowngradesDestructive pins the fail-closed helper:
// caller input cannot downgrade an action-inferred high risk. Non-destructive
// caller classifications remain authoritative so diagnose_and_remediate can
// keep its existing low-risk autonomous loop.
func TestRiskForStep_NeverDowngradesDestructive(t *testing.T) {
	cases := []struct {
		name     string
		caller   string
		inferred string
		want     string
	}{
		{"caller_low_cannot_downgrade_destructive", "low", "high", "high"},
		{"caller_low_diagnose_stays_low", "low", "medium", "low"},
		{"caller_low_readonly_stays_low", "low", "low", "low"},
		{"caller_medium_readonly_stays_medium", "medium", "low", "medium"},
		{"caller_high_cannot_downgrade", "high", "high", "high"},
		{"unknown_caller_destructive_stays_high", "bogus", "high", "high"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := riskForStep(tc.caller, tc.inferred); got != tc.want {
				t.Errorf("riskForStep(%q,%q) = %q, want %q", tc.caller, tc.inferred, got, tc.want)
			}
		})
	}
}

// TestOrchestrator_CallerLowCannotDowngradeDestructiveAction verifies the
// end-to-end safety gate: even if the caller's risk is forced to "low",
// a destructive action (delete-instances) is still blocked by RBAC inside
// the execution loop. This is the direct regression test for the Critic
// BLOCKER on the orchestrator's risk-overwrite fix.
func TestOrchestrator_CallerLowCannotDowngradeDestructiveAction(t *testing.T) {
	root := t.TempDir()
	// Simulate the orchestrator's post-BuildTaskFromPlan fix path by
	// applying the same per-step maxRiskLevel we now use in production.
	step := TaskStep{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "delete-instances", Verb: "delete"}
	step.Risk = riskForStep("low", inferRiskFromAction(step.Action))
	if step.Risk != "high" {
		t.Fatalf("caller=low must not downgrade delete-instances, got risk=%q", step.Risk)
	}

	task := &TaskState{
		ID: "orch-low-destructive", Status: TaskStatusRunning, CurrentStep: 0,
		Steps: []TaskStep{step},
	}
	plan := &ExecutionPlan{Steps: []PlanStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "delete-instances"}}}
	exec := &StubExecutor{Outcomes: []StubStep{{ExitCode: 0}}}

	out := RunExecutionLoopWithHealing(root, task, plan, nil, nil, HealingPolicy{}, exec)
	if out.Status != TaskStatusFailed {
		t.Fatalf("want failed (RBAC blocked destructive), got %s", out.Status)
	}
	if len(out.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(out.Results))
	}
	if out.Results[0].GCLDecision != "blocked_by_rbac" {
		t.Fatalf("destructive must be blocked_by_rbac, got %q", out.Results[0].GCLDecision)
	}
	if !strings.Contains(out.Results[0].Error, "exceeds max auto-approval") {
		t.Fatalf("want RBAC deny reason, got %q", out.Results[0].Error)
	}
}

// TestOrchestrator_CallerLowDiagnoseStaysLow verifies the default
// diagnose_and_remediate path remains low-risk under explicit caller=low.
func TestOrchestrator_CallerLowDiagnoseStaysLow(t *testing.T) {
	step := TaskStep{Step: 1, Skill: "huaweicloud-vpc-ops", Action: "diagnose_and_remediate", Verb: "diagnose"}
	step.Risk = riskForStep("low", inferRiskFromAction(step.Action))
	if step.Risk != "low" {
		t.Fatalf("caller=low + diagnose_and_remediate must stay low, got risk=%q", step.Risk)
	}
}
