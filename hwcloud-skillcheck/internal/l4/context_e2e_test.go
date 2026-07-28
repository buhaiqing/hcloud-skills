package l4

import "testing"

// TestContextMemory_PersistsAcrossRuns simulates two separate invocations
// of the orchestrator pointing at the same root and verifies that the
// second invocation sees the first invocation's recorded tasks and errors.
func TestContextMemory_PersistsAcrossRuns(t *testing.T) {
	root := t.TempDir()

	// "First run"
	cm1, _ := NewContextMemory(root)
	if err := cm1.RecordTask(TaskSummary{
		TaskID: "first-run-task", Fault: "first invocation",
		StartedAt: "2026-07-28T08:00:00Z", Status: "completed",
	}); err != nil {
		t.Fatalf("first record: %v", err)
	}
	if err := cm1.RecordError(ErrorSummary{
		Timestamp: "2026-07-28T08:00:01Z", Skill: "s", Action: "x",
		ErrorClass: "permanent", ErrorMsg: "boom",
	}); err != nil {
		t.Fatalf("first error: %v", err)
	}
	if err := cm1.Flush(); err != nil {
		t.Fatalf("flush first run: %v", err)
	}

	// "Second run" — fresh ContextMemory handle, same root.
	cm2, _ := NewContextMemory(root)
	c, err := cm2.Load()
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if len(c.RecentTasks) != 1 || c.RecentTasks[0].TaskID != "first-run-task" {
		t.Fatalf("recent_tasks not persisted: %+v", c.RecentTasks)
	}
	if len(c.RecentErrors) != 1 || c.RecentErrors[0].ErrorMsg != "boom" {
		t.Fatalf("recent_errors not persisted: %+v", c.RecentErrors)
	}
	// Set a preference in run 1, flush, read it via a fresh handle.
	if err := cm1.SetPreference("default_region", "cn-north-4"); err != nil {
		t.Fatalf("set pref: %v", err)
	}
	if err := cm1.Flush(); err != nil {
		t.Fatalf("flush pref: %v", err)
	}
	cm3, _ := NewContextMemory(root)
	c2, _ := cm3.Load()
	if c2.Preferences["default_region"] != "cn-north-4" {
		t.Fatalf("preferences not persisted: %+v", c2.Preferences)
	}
}
