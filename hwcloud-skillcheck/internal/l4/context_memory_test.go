package l4

import (
	"encoding/json"
	"testing"
)

func TestContext_RoundTrip(t *testing.T) {
	in := Context{
		Schema:      ContextSchema,
		SessionID:   "550e8400-e29b-41d4-a716-446655440000",
		CreatedAt:   "2026-07-28T08:00:00Z",
		LastUpdated: "2026-07-28T09:15:00Z",
		RecentTasks: []TaskSummary{{TaskID: "t1", Status: "completed", PrimarySkill: "huaweicloud-ecs-ops"}},
		OpenTasks:   []string{"t2"},
		Preferences: map[string]string{"default_region": "cn-north-4"},
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Context
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Schema != ContextSchema {
		t.Fatalf("schema = %q, want %q", out.Schema, ContextSchema)
	}
	if len(out.RecentTasks) != 1 || out.RecentTasks[0].TaskID != "t1" {
		t.Fatalf("recent_tasks lost: %+v", out.RecentTasks)
	}
	if out.Preferences["default_region"] != "cn-north-4" {
		t.Fatalf("preferences lost: %+v", out.Preferences)
	}
}

func TestContextSchema_Constant(t *testing.T) {
	if ContextSchema != "context-memory/v1" {
		t.Fatalf("ContextSchema = %q, want context-memory/v1", ContextSchema)
	}
}
