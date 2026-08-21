package l4

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// transientFailExecutor always fails transiently and counts invocations.
// Kept hermetic: no real exec, just a controlled transient error.
type transientFailExecutor struct {
	calls int32
}

func (e *transientFailExecutor) Run(candidate string, timeout time.Duration) (int, string, error) {
	atomic.AddInt32(&e.calls, 1)
	return 1, "", errors.New("connection timeout")
}

// TestRunExecutionLoopWithHealing_RetryBudgetBounded asserts a persistently
// transient step stops retrying after MaxRetries instead of retrying forever.
// The PostFailureHook guard (retryCount >= MaxRetries) can only fire if the
// loop actually feeds it a live per-step counter; hardcoding 0 (the old bug)
// would loop unbounded and this test's exact-count assertion would fail.
func TestRunExecutionLoopWithHealing_RetryBudgetBounded(t *testing.T) {
	dir := t.TempDir()
	mem, err := NewOutcomeMemory(dir)
	if err != nil {
		t.Fatalf("NewOutcomeMemory: %v", err)
	}
	task := &TaskState{
		ID: "retry-budget", Status: TaskStatusRunning, CurrentStep: 0,
		Steps: []TaskStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list", Verb: "list", Risk: "low"}},
	}
	plan := &ExecutionPlan{Steps: []PlanStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list"}}}
	p := HealingPolicy{MaxRetries: 2}
	exec := &transientFailExecutor{}

	out := RunExecutionLoopWithHealing(dir, task, plan, nil, mem, p, exec)

	// Initial attempt + 2 retries, then escalate: bounded, not infinite.
	if got := atomic.LoadInt32(&exec.calls); got != 3 {
		t.Fatalf("executions = %d, want 3 (1 initial + 2 retries before escalate)", got)
	}
	if len(out.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(out.Results))
	}
	if out.Results[0].Success {
		t.Fatal("escalated step must remain a failure")
	}
}
