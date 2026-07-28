package l4

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestE2E_SkipThenRetryThenEscalate exercises the full healing cycle with a
// StubExecutor: skip-then-retry-then-escalate.
func TestE2E_SkipThenRetryThenEscalate(t *testing.T) {
	root := t.TempDir()
	mem, err := NewOutcomeMemory(root)
	if err != nil {
		t.Fatalf("mem: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	// Seed: 6 failures for skill-a/list to trigger skip on first step.
	for i := 0; i < 6; i++ {
		_ = mem.Record(OutcomeRecord{
			ID:        "seed-" + string(rune('a'+i)),
			Timestamp: now,
			Skill:     "skill-a", Action: "list",
			Outcome: "failure",
		})
	}
	p := HealingPolicy{
		MaxRetries:               1,
		RetryBackoff:             1 * time.Millisecond,
		DestructiveVerbs:         []string{"delete"},
		FailureRateSkipThreshold: 0.5,
		MinSamples:               5,
		LookbackWindow:           time.Hour,
	}
	// The 3 plan steps:
	//   1) skill-a / list — should be SKIPPED (bad history)
	//   2) skill-b / list — transient then success (retry path)
	//   3) skill-c / list — permanent failure (escalate path)
	task := &TaskState{
		ID: "t-e2e", Status: TaskStatusRunning, CurrentStep: 0,
		Steps: []TaskStep{
			{Step: 1, Skill: "skill-a", Action: "list", Verb: "list", Risk: "low"},
			{Step: 2, Skill: "skill-b", Action: "list", Verb: "list", Risk: "low"},
			{Step: 3, Skill: "skill-c", Action: "list", Verb: "list", Risk: "low"},
		},
	}
	plan := &ExecutionPlan{Steps: []PlanStep{
		{Step: 1, Skill: "skill-a", Action: "list"},
		{Step: 2, Skill: "skill-b", Action: "list"},
		{Step: 3, Skill: "skill-c", Action: "list"},
	}, MaxTotalTimeoutSeconds: 10}
	exec := &StubExecutor{Outcomes: []StubStep{
		// Step 2 first run: transient, retry.
		{ExitCode: 1, Err: errors.New("connection reset by peer")},
		// Step 2 retry: success.
		{ExitCode: 0},
		// Step 3: permanent failure, escalate.
		{ExitCode: 2, Err: errors.New("permission denied")},
	}}

	out := RunExecutionLoopWithHealing(root, task, plan, nil, mem, p, exec)
	if out.Status != TaskStatusCompleted {
		t.Fatalf("want completed, got %s", out.Status)
	}
	if len(out.Results) != 3 {
		t.Fatalf("want 3 results, got %d", len(out.Results))
	}
	// Step 1: SKIPPED_BY_HEALING.
	if !strings.Contains(out.Results[0].Error, "skipped") {
		t.Errorf("step 1: want skipped, got %+v", out.Results[0])
	}
	if out.Results[0].GCLDecision != "SKIPPED_BY_HEALING" {
		t.Errorf("step 1: want SKIPPED_BY_HEALING, got %s", out.Results[0].GCLDecision)
	}
	// Step 2: retried; we expect the final result to be Success=true (after retry).
	if !out.Results[1].Success {
		t.Errorf("step 2: want success after retry, got %+v", out.Results[1])
	}
	// Step 3: permanent failure, never retried.
	if out.Results[2].Success {
		t.Errorf("step 3: want failure, got %+v", out.Results[2])
	}
	if !strings.Contains(out.Results[2].Error, "permission denied") {
		t.Errorf("step 3: want permission denied, got %q", out.Results[2].Error)
	}
	t.Logf("memory dir = %s", filepath.Join(root, ".l4-memory"))
	if !strings.Contains(mem.path, root) {
		t.Logf("memory path resolves under root: %s", mem.path)
	}
}
