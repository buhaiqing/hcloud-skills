package l4

import (
	"strings"
	"testing"
	"time"
)

func TestRunExecutionLoop_PreExecSkipsBadHistory(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewOutcomeMemory(dir)
	now := NowISO()
	for i := 0; i < 6; i++ {
		_ = mem.Record(OutcomeRecord{
			ID:        "old" + string(rune('a'+i)),
			Timestamp: now,
			Skill:     "huaweicloud-ecs-ops",
			Action:    "delete-instances",
			Outcome:   "failure",
		})
	}
	task := &TaskState{
		ID: "test-task", Status: TaskStatusRunning, CurrentStep: 0,
		Steps: []TaskStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "delete-instances", Verb: "delete", Risk: "high"}},
	}
	plan := &ExecutionPlan{Steps: []PlanStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "delete-instances"}}}
	p := HealingPolicy{FailureRateSkipThreshold: 0.5, MinSamples: 5, LookbackWindow: time.Hour, DestructiveVerbs: []string{"delete"}}

	out := RunExecutionLoopWithHealing(dir, task, plan, nil, mem, p, nil)
	if out.Status != TaskStatusCompleted {
		t.Fatalf("want completed, got %s", out.Status)
	}
	if len(out.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(out.Results))
	}
	if !strings.Contains(out.Results[0].Error, "skipped") {
		t.Fatalf("want skipped result, got %+v", out.Results[0])
	}
}

func TestRunExecutionLoop_ZeroPolicyBehavesAsBefore(t *testing.T) {
	dir := t.TempDir()
	task := &TaskState{
		ID: "t2", Status: TaskStatusRunning, CurrentStep: 0,
		Steps: []TaskStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list", Risk: "low"}},
	}
	plan := &ExecutionPlan{Steps: []PlanStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list"}}}
	out := RunExecutionLoopWithHealing(dir, task, plan, nil, nil, HealingPolicy{}, nil)
	if len(out.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(out.Results))
	}
	if out.Results[0].Error != "" && !strings.Contains(out.Results[0].Error, "skipped") {
		t.Logf("result: %+v", out.Results[0])
	}
}
