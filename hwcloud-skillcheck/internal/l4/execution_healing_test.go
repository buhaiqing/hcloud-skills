package l4

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// boundedTransientExecutor fails transiently for the first transientCalls
// invocations, then returns a permanent sentinel. The bound means a retry-budget
// regression (loop never escalates) terminates quickly with a count mismatch
// instead of spinning on transient errors until the package timeout.
type boundedTransientExecutor struct {
	transientCalls int
	calls          int32
}

func (e *boundedTransientExecutor) Run(candidate string, timeout time.Duration) (int, string, error) {
	n := atomic.AddInt32(&e.calls, 1)
	if int(n) > e.transientCalls {
		return 1, "", errors.New("permanent-bounded")
	}
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
	// MaxRetries=2 → at most 2 retries after the initial attempt, so the fake
	// stays transient through call 3 and turns permanent on call 4+. A budget
	// regression that never escalates hits the permanent sentinel and fails.
	exec := &boundedTransientExecutor{transientCalls: 3}

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
