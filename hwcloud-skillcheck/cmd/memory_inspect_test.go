package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/l4"
)

func TestRunMemoryInspect(t *testing.T) {
	root := t.TempDir()
	outcomes, err := l4.NewOutcomeMemory(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := outcomes.Record(l4.OutcomeRecord{
		ID: "outcome-1", Timestamp: "2026-07-28T21:00:00Z", Skill: "huaweicloud-ecs-ops", Action: "delete-instance",
		Outcome: "failure", ErrorClass: "transient", RetryCount: 1,
	}); err != nil {
		t.Fatal(err)
	}
	contextMemory, err := l4.NewContextMemory(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := contextMemory.Save(&l4.Context{
		Schema: l4.ContextSchema, SessionID: "550e8400-e29b-41d4-a716-446655440000",
		CreatedAt: time.Now().UTC().Add(-13 * time.Hour).Format(time.RFC3339), LastUpdated: time.Now().UTC().Format(time.RFC3339),
		RecentTasks: []l4.TaskSummary{{TaskID: "abcd1234", FinishedAt: "2026-07-28T20:30:00Z", Status: "completed", PrimarySkill: "huaweicloud-ecs-ops"}},
		Preferences: map[string]string{"default_region": "cn-north-4"},
	}); err != nil {
		t.Fatal(err)
	}

	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previousStdout := os.Stdout
	os.Stdout = writeEnd
	defer func() { os.Stdout = previousStdout }()

	if err := runMemoryInspect([]string{"inspect", "--root", root}); err != nil {
		t.Fatal(err)
	}
	if err := writeEnd.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(readEnd)
	if err != nil {
		t.Fatal(err)
	}

	for _, field := range []string{
		"outcome_memory_path:",
		"outcome_records:    1",
		"huaweicloud-ecs-ops / delete-instance",
		"outcome=failure",
		"error_class=transient",
		"retry=1",
		"context_memory_path:",
		"session_id:          550e8400-e29b-41d4-a716-446655440000",
		"recent_tasks:        1 of 20",
		"task=abcd1234",
		"status=completed",
		`preferences:         {"default_region":"cn-north-4"}`,
	} {
		if !strings.Contains(string(output), field) {
			t.Errorf("output missing %q:\n%s", field, output)
		}
	}
}
