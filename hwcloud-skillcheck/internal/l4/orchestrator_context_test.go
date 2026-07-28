package l4

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOrchestrator_RecordsTaskLifecycleViaHandleFault(t *testing.T) {
	root := t.TempDir()
	cm, err := NewContextMemory(root)
	if err != nil {
		t.Fatalf("context: %v", err)
	}

	in := HandleFaultInput{
		Root:       root,
		Fault:      "list my ECS",
		Resource:   "ecs:instance",
		Risk:       "low",
		ContextMem: cm,
	}

	_ = HandleFault(in, nil)

	raw, err := os.ReadFile(filepath.Join(root, ".l4-memory", "context.json"))
	if err != nil {
		t.Fatalf("read context: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("context.json is empty")
	}
	c, _ := cm.Load()
	if len(c.RecentTasks) == 0 {
		t.Fatalf("want at least 1 task, got 0")
	}
	// Find the task with our fault.
	found := false
	for _, ts := range c.RecentTasks {
		if ts.Fault == "list my ECS" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("fault not recorded; recent_tasks=%+v", c.RecentTasks)
	}
}
