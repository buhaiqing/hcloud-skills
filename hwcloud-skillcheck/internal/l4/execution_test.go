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

// writePatternFixture writes a failure_patterns.json for a skill into root.
func writePatternFixture(t *testing.T, root, skill string, patterns string) {
	t.Helper()
	dir := root + "/" + skill + "/assets"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := `{"$schema":"failure-patterns/v1","skill_id":"` + skill + `","patterns":` + patterns + `}`
	if err := os.WriteFile(dir+"/failure_patterns.json", []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestRunExecutionLoop_PatternRiskNoMatchExecutes verifies a command that does
// NOT match any pattern is executed normally (no SKIPPED_BY_PATTERN_RISK).
func TestRunExecutionLoop_PatternRiskNoMatchExecutes(t *testing.T) {
	root := t.TempDir()
	// Pattern regex token "deleteInstance" does not appear in command "listServers".
	writePatternFixture(t, root, "huaweicloud-ecs-ops", `[{"id":"FP1","signature":{"error_message_regex":"deleteInstance"},"fix":{"action":"x"}}]`)

	task := &TaskState{
		ID: "t-nomatch", Status: TaskStatusRunning, CurrentStep: 0,
		Steps: []TaskStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list", Verb: "list", Risk: "low"}},
	}
	plan := &ExecutionPlan{Steps: []PlanStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list"}}}
	matched := []MatchedSkill{{Skill: "huaweicloud-ecs-ops"}}
	exec := &StubExecutor{Outcomes: []StubStep{{ExitCode: 0, Stdout: "[]"}}}

	out := RunExecutionLoopWithHealing(root, task, plan, matched, nil, HealingPolicy{}, exec)
	if out.Status != TaskStatusCompleted {
		t.Fatalf("want completed, got %s", out.Status)
	}
	if len(out.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(out.Results))
	}
	if out.Results[0].GCLDecision == "SKIPPED_BY_PATTERN_RISK" {
		t.Fatalf("non-matching command must not be skipped, got %q", out.Results[0].GCLDecision)
	}
	if !out.Results[0].Success {
		t.Fatalf("non-matching command should execute successfully, err=%v", out.Results[0].Error)
	}
}

// TestRunExecutionLoop_PatternRiskNoPatternFile verifies behavior is unchanged
// when no failure_patterns.json exists for the skill.
func TestRunExecutionLoop_PatternRiskNoPatternFile(t *testing.T) {
	root := t.TempDir() // no failure_patterns.json written
	task := &TaskState{
		ID: "t-nopat", Status: TaskStatusRunning, CurrentStep: 0,
		Steps: []TaskStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list", Verb: "list", Risk: "low"}},
	}
	plan := &ExecutionPlan{Steps: []PlanStep{{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list"}}}
	matched := []MatchedSkill{{Skill: "huaweicloud-ecs-ops"}}
	exec := &StubExecutor{Outcomes: []StubStep{{ExitCode: 0, Stdout: "[]"}}}

	out := RunExecutionLoopWithHealing(root, task, plan, matched, nil, HealingPolicy{}, exec)
	if out.Status != TaskStatusCompleted {
		t.Fatalf("want completed, got %s", out.Status)
	}
	if len(out.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(out.Results))
	}
	if out.Results[0].GCLDecision == "SKIPPED_BY_PATTERN_RISK" {
		t.Fatalf("no pattern file must not skip, got %q", out.Results[0].GCLDecision)
	}
}

// TestRunExecutionLoop_PatternRiskSkipsThenContinues verifies a multi-step task:
// step1 matches a pattern → skipped; step2 does not → executed.
func TestRunExecutionLoop_PatternRiskSkipsThenContinues(t *testing.T) {
	root := t.TempDir()
	// Pattern matches "deleteInstance" (step1 command), not "listServers" (step2).
	writePatternFixture(t, root, "huaweicloud-ecs-ops", `[{"id":"FP1","signature":{"error_message_regex":"deleteInstance"},"fix":{"action":"x"}}]`)

	task := &TaskState{
		ID: "t-2step", Status: TaskStatusRunning, CurrentStep: 0,
		Steps: []TaskStep{
			{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "deleteInstance", Verb: "delete", Risk: "high"},
			{Step: 2, Skill: "huaweicloud-ecs-ops", Action: "list", Verb: "list", Risk: "low"},
		},
	}
	plan := &ExecutionPlan{Steps: []PlanStep{
		{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "deleteInstance"},
		{Step: 2, Skill: "huaweicloud-ecs-ops", Action: "list"},
	}}
	matched := []MatchedSkill{{Skill: "huaweicloud-ecs-ops"}}
	exec := &StubExecutor{Outcomes: []StubStep{{ExitCode: 0, Stdout: "[]"}}}

	out := RunExecutionLoopWithHealing(root, task, plan, matched, nil, HealingPolicy{}, exec)
	if out.Status != TaskStatusCompleted {
		t.Fatalf("want completed, got %s", out.Status)
	}
	if len(out.Results) != 2 {
		t.Fatalf("want 2 results, got %d", len(out.Results))
	}
	if out.Results[0].GCLDecision != "SKIPPED_BY_PATTERN_RISK" {
		t.Fatalf("step1 want SKIPPED_BY_PATTERN_RISK, got %q", out.Results[0].GCLDecision)
	}
	if out.Results[1].GCLDecision == "SKIPPED_BY_PATTERN_RISK" {
		t.Fatalf("step2 (non-matching) must not be skipped, got %q", out.Results[1].GCLDecision)
	}
	if !out.Results[1].Success {
		t.Fatalf("step2 should execute successfully, err=%v", out.Results[1].Error)
	}
}

// TestMatchedSkills_Deduplicates verifies the helper drops duplicate skill IDs.
func TestMatchedSkills_Deduplicates(t *testing.T) {
	in := []MatchedSkill{{Skill: "huaweicloud-ecs-ops"}, {Skill: "huaweicloud-ecs-ops"}, {Skill: "huaweicloud-rds-ops"}}
	got := matchedSkills(in)
	if len(got) != 2 {
		t.Fatalf("want 2 unique skills, got %d: %v", len(got), got)
	}
	seen := map[string]bool{}
	for _, s := range got {
		if seen[s] {
			t.Fatalf("duplicate skill %q in output", s)
		}
		seen[s] = true
	}
}

// TestMatchedSkills_Empty verifies empty input → empty output (no panic).
func TestMatchedSkills_Empty(t *testing.T) {
	if got := matchedSkills(nil); len(got) != 0 {
		t.Fatalf("want 0 skills for nil input, got %v", got)
	}
}

// TestInferRiskFromAction_Destructive verifies destructive verbs → high.
func TestInferRiskFromAction_Destructive(t *testing.T) {
	for _, action := range []string{"delete-instance", "terminate", "destroy", "drop-table", "remove-node"} {
		if got := inferRiskFromAction(action); got != "high" {
			t.Errorf("inferRiskFromAction(%q): got %q, want high", action, got)
		}
	}
}

// TestInferRiskFromAction_ReadOnly verifies read-only verbs → low.
func TestInferRiskFromAction_ReadOnly(t *testing.T) {
	for _, action := range []string{"list", "describe-instance", "show", "get-alarm", "query-metrics", "search-logs"} {
		if got := inferRiskFromAction(action); got != "low" {
			t.Errorf("inferRiskFromAction(%q): got %q, want low", action, got)
		}
	}
}

// TestInferRiskFromAction_Unknown verifies unknown verbs → medium.
func TestInferRiskFromAction_Unknown(t *testing.T) {
	for _, action := range []string{"doSomething", "fooBar", "xyz"} {
		if got := inferRiskFromAction(action); got != "medium" {
			t.Errorf("inferRiskFromAction(%q): got %q, want medium", action, got)
		}
	}
}

// TestInferRiskFromAction_CaseInsensitive verifies uppercase input is handled.
func TestInferRiskFromAction_CaseInsensitive(t *testing.T) {
	if got := inferRiskFromAction("DELETE-INSTANCE"); got != "high" {
		t.Errorf("uppercase DELETE-INSTANCE: got %q, want high", got)
	}
	if got := inferRiskFromAction("ListServers"); got != "low" {
		t.Errorf("mixed-case ListServers: got %q, want low", got)
	}
}
