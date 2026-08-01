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
