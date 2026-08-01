package l4

import (
	"os"
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

// TestRunExecutionLoop_PatternRiskSkipsStep verifies the pre-execution pattern
// risk gate: a command matching a high-risk failure pattern is skipped
// (SKIPPED_BY_PATTERN_RISK) instead of executed (prevent > react).
func TestRunExecutionLoop_PatternRiskSkipsStep(t *testing.T) {
	root := t.TempDir()
	dir := root + "/huaweicloud-ecs-ops/assets"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// failure_patterns.json with a high-risk pattern whose error_message_regex
	// token appears in the command "createServer" (matchPreExecutionRisk scans
	// the command for the regex's first token).
	fixture := `{
		"$schema": "failure-patterns/v1",
		"skill_id": "huaweicloud-ecs-ops",
		"patterns": [{
			"id": "ECS-FP001",
			"category": "resource_state",
			"signature": {"error_code": "Ecs.0801", "error_message_regex": "createServer", "command_pattern": "create-server"},
			"fix": {"strategy": "fallback", "action": "Try different AZ"},
			"stats": {"occurrence_count": 10, "success_rate": 0.2}
		}]
	}`
	if err := os.WriteFile(dir+"/failure_patterns.json", []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}

	task := &TaskState{
		ID: "risk-test", Status: TaskStatusRunning, CurrentStep: 0,
		Steps: []TaskStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "createServer", Verb: "create", Risk: "medium"}},
	}
	plan := &ExecutionPlan{Steps: []PlanStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "createServer"}}}
	// matched provides the skills list for pattern pre-fetch.
	matched := []MatchedSkill{{Skill: "huaweicloud-ecs-ops"}}

	out := RunExecutionLoopWithHealing(root, task, plan, matched, nil, HealingPolicy{}, nil)
	if out.Status != TaskStatusCompleted {
		t.Fatalf("want completed, got %s", out.Status)
	}
	if len(out.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(out.Results))
	}
	if out.Results[0].GCLDecision != "SKIPPED_BY_PATTERN_RISK" {
		t.Fatalf("want SKIPPED_BY_PATTERN_RISK, got %q (err=%v)", out.Results[0].GCLDecision, out.Results[0].Error)
	}
	if out.Results[0].Success {
		t.Fatal("skipped step must not be success")
	}
}
