package l4

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestEnsureMemoryDir(t *testing.T) {
	root := t.TempDir()
	dir, err := EnsureMemoryDir(root)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if !strings.HasSuffix(dir, ".l4-memory") {
		t.Fatalf("path %q should end with .l4-memory", dir)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Fatalf("dir perm = %o, want 0700", perm)
	}
	dir2, err := EnsureMemoryDir(root)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if dir2 != dir {
		t.Fatalf("second path %q != first %q", dir2, dir)
	}
}

func TestPersistAndLoadTask(t *testing.T) {
	root := t.TempDir()
	state := &TaskState{
		ID:          "testtask123456",
		Fault:       "RDS connection timeout",
		Root:        root,
		CreatedAt:   NowISO(),
		Status:      TaskStatusRunning,
		CurrentStep: 0,
		Steps: []TaskStep{
			{Step: 0, Skill: "huaweicloud-rds-ops", Action: "describe-instance", Risk: "low"},
			{Step: 1, Skill: "huaweicloud-rds-ops", Action: "restart-instance", Risk: "medium"},
		},
	}

	// Persist
	if err := PersistTask(root, state.ID, state); err != nil {
		t.Fatalf("PersistTask: %v", err)
	}

	// Load
	loaded, err := LoadTask(root, state.ID)
	if err != nil {
		t.Fatalf("LoadTask: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadTask returned nil")
	}
	if loaded.ID != state.ID {
		t.Errorf("ID mismatch: got %s, want %s", loaded.ID, state.ID)
	}
	if loaded.Fault != state.Fault {
		t.Errorf("Fault mismatch: got %s, want %s", loaded.Fault, state.Fault)
	}
	if len(loaded.Steps) != len(state.Steps) {
		t.Errorf("Steps length mismatch: got %d, want %d", len(loaded.Steps), len(state.Steps))
	}
}

func TestTaskStateLifecycle(t *testing.T) {
	state := &TaskState{
		ID:     "lifecycle123456",
		Status: TaskStatusRunning,
		Steps: []TaskStep{
			{Step: 0, Skill: "huaweicloud-ecs-ops", Action: "list-servers", Risk: "low"},
			{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "restart-instance", Risk: "medium"},
		},
		CurrentStep: 0,
	}

	// Test NextStep
	step := state.NextStep()
	if step == nil || step.Action != "list-servers" {
		t.Errorf("NextStep: got %v, want action=list-servers", step)
	}

	// Complete first step
	state.CurrentStep++
	state.Results = append(state.Results, StepResult{Step: 0, Success: true})

	// Test Progress
	if got := state.Progress(); got != "1/2 steps completed" {
		t.Errorf("Progress: got %q, want %q", got, "1/2 steps completed")
	}

	// Test CompletedCount / RemainingCount
	if n := state.CompletedCount(); n != 1 {
		t.Errorf("CompletedCount: got %d, want 1", n)
	}
	if n := state.RemainingCount(); n != 1 {
		t.Errorf("RemainingCount: got %d, want 1", n)
	}

	// Complete all
	state.CurrentStep = len(state.Steps)
	CompleteTask(state)
	if state.Status != TaskStatusCompleted {
		t.Errorf("CompleteTask: got status %s, want %s", state.Status, TaskStatusCompleted)
	}

	// ResumeTask should fail on completed task
	if err := ResumeTask(state); err == nil {
		t.Error("ResumeTask: expected error on completed task")
	}
}

func TestResumeTask(t *testing.T) {
	state := &TaskState{
		ID:          "resumetest123456",
		Status:      TaskStatusPaused,
		CurrentStep: 1,
		Steps: []TaskStep{
			{Step: 0, Skill: "huaweicloud-rds-ops", Action: "describe", Risk: "low"},
			{Step: 1, Skill: "huaweicloud-rds-ops", Action: "restart", Risk: "medium"},
		},
	}

	if err := ResumeTask(state); err != nil {
		t.Fatalf("ResumeTask: %v", err)
	}
	if state.Status != TaskStatusRunning {
		t.Errorf("Status after ResumeTask: got %s, want %s", state.Status, TaskStatusRunning)
	}
	if state.RecoveryCount != 1 {
		t.Errorf("RecoveryCount: got %d, want 1", state.RecoveryCount)
	}
}

func TestIsStale(t *testing.T) {
	// Create a task with an old UpdatedAt.
	state := &TaskState{
		ID:        "staletest123456",
		Status:    TaskStatusRunning,
		UpdatedAt: time.Now().Add(-25 * time.Hour).UTC().Format("2006-01-02T15:04:05Z"),
	}

	if !state.IsStale(DefaultTaskTTL) {
		t.Error("IsStale: expected true for 25h old running task")
	}

	// Non-running tasks are never stale.
	state.Status = TaskStatusPaused
	if state.IsStale(DefaultTaskTTL) {
		t.Error("IsStale: paused task should not be stale")
	}
}

func TestListTasks(t *testing.T) {
	root := t.TempDir()

	// Persist two tasks with simple IDs.
	taskIDs := []string{"listtest001", "listtest002"}
	for _, id := range taskIDs {
		state := &TaskState{
			ID:        id,
			Fault:     "test",
			Root:      root,
			CreatedAt: NowISO(),
			Status:    TaskStatusCompleted,
			Steps:     nil,
		}
		if err := PersistTask(root, state.ID, state); err != nil {
			t.Fatalf("PersistTask %s: %v", id, err)
		}
	}

	ids, err := ListTasks(root)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("ListTasks: got %d tasks, want 2", len(ids))
	}
}

func TestAbortTask(t *testing.T) {
	state := &TaskState{
		ID:     "aborttest123456",
		Status: TaskStatusRunning,
	}

	AbortTask(state)
	if state.Status != TaskStatusAborted {
		t.Errorf("AbortTask: got %s, want %s", state.Status, TaskStatusAborted)
	}

	// Cannot resume aborted task.
	if err := ResumeTask(state); err == nil {
		t.Error("ResumeTask: expected error on aborted task")
	}
}

func TestParseTaskID(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"task123456789012", "task123456789012"},
		{"/some/path/.l4-tasks/abcdefgh.json", "abcdefgh"},
		{"/audit-results/orchestrator-trace-abcdefgh.json", "abcdefgh"},
	}

	for _, c := range cases {
		got := ParseTaskID(c.input)
		if got != c.expected {
			t.Errorf("ParseTaskID(%q): got %q, want %q", c.input, got, c.expected)
		}
	}
}

func TestPreFetchFailurePatterns_EmptyAndMissing(t *testing.T) {
	root := t.TempDir()
	if got := preFetchFailurePatterns(root, nil); len(got) != 0 {
		t.Fatalf("empty skills → empty cache, got %d", len(got))
	}
	// Missing skill assets are best-effort omitted (no panic, no error).
	got := preFetchFailurePatterns(root, []string{"huaweicloud-ecs-ops", "huaweicloud-no-such-ops"})
	if len(got) != 0 {
		t.Fatalf("missing assets should omit entries, got %v", got)
	}
}

func TestIsResumable(t *testing.T) {
	cases := []struct {
		status   TaskStatus
		expected bool
	}{
		{TaskStatusRunning, false},
		{TaskStatusPaused, true},
		{TaskStatusFailed, true},
		{TaskStatusCompleted, false},
		{TaskStatusAborted, false},
	}

	for _, c := range cases {
		state := &TaskState{Status: c.status}
		if got := state.IsResumable(); got != c.expected {
			t.Errorf("IsResumable(status=%s): got %v, want %v", c.status, got, c.expected)
		}
	}
}
