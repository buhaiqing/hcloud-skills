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

func TestStripVolatileArgs_DropsTimeAndPaginationFlags(t *testing.T) {
	in := "hcloud ces list-metrics --namespace SYS.ECS --query-window 1h --start-time 2026-07-28T00:00:00Z --marker abc --instance-id i-1"
	got := stripVolatileArgs(in)
	want := "hcloud ces list-metrics --namespace SYS.ECS --instance-id i-1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStripVolatileArgs_InlineEqualsForm(t *testing.T) {
	in := "hcloud ces list-metrics --query-window=1h --end-time=2026-07-28T01:00:00Z --namespace SYS.ECS"
	got := stripVolatileArgs(in)
	want := "hcloud ces list-metrics --namespace SYS.ECS"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestHashContext_StableAcrossVolatileWindows(t *testing.T) {
	a := hashContext("hcloud ces list-metrics --namespace SYS.ECS --query-window 1h --start-time 2026-07-28T00:00:00Z")
	b := hashContext("hcloud ces list-metrics --namespace SYS.ECS --query-window 24h --start-time 2026-07-27T00:00:00Z")
	if a != b {
		t.Fatalf("volatile windows changed hash: %s vs %s", a, b)
	}
	c := hashContext("hcloud ces list-metrics --namespace SYS.ECS --instance-id i-other")
	if a == c {
		t.Fatal("stable resource id change should change hash")
	}
	if len(a) != 16 {
		t.Fatalf("hash len=%d, want 16", len(a))
	}
}
