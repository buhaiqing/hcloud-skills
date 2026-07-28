package l4

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestContextMemory_Save_CreatesFileWithMode0600(t *testing.T) {
	dir := t.TempDir()
	mem, err := NewContextMemory(dir)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	c := &Context{
		Schema:      ContextSchema,
		SessionID:   "sess-1",
		CreatedAt:   "2026-07-28T08:00:00Z",
		LastUpdated: "2026-07-28T08:00:00Z",
	}
	if err := mem.Save(c); err != nil {
		t.Fatalf("save: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, ".l4-memory", "context.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("file perm = %o, want 0600", perm)
	}
}

func TestContextMemory_Save_AtomicNoTempLeftBehind(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewContextMemory(dir)
	c := &Context{Schema: ContextSchema, SessionID: "s", CreatedAt: "x", LastUpdated: "x"}
	if err := mem.Save(c); err != nil {
		t.Fatalf("save: %v", err)
	}
	// Confirm no .tmp.* files remain.
	entries, _ := os.ReadDir(filepath.Join(dir, ".l4-memory"))
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp.") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}

func TestContextMemory_Load_FirstRunReturnsFreshContext(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewContextMemory(dir)
	c, err := mem.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.Schema != ContextSchema {
		t.Errorf("schema = %q", c.Schema)
	}
	if c.SessionID == "" {
		t.Error("session_id empty")
	}
	if c.CreatedAt == "" || c.LastUpdated == "" {
		t.Error("timestamps empty")
	}
	// No persistence side-effect: a subsequent Load returns a NEW session_id.
	c2, _ := mem.Load()
	if c.SessionID == c2.SessionID {
		t.Error("first-run load should not have persisted; got same session_id twice")
	}
}

func TestContextMemory_Load_RejectsUnknownSchema(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewContextMemory(dir)
	// Write directly to disk to bypass Save's schema enforcement.
	bad := &Context{Schema: "context-memory/v999", SessionID: "x", CreatedAt: "x", LastUpdated: "x"}
	raw, _ := json.MarshalIndent(bad, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, ".l4-memory", "context.json"), append(raw, '\n'), 0o600); err != nil {
		t.Fatalf("seed bad: %v", err)
	}
	if _, err := mem.Load(); err == nil {
		t.Fatal("want error for unknown schema, got nil")
	}
}

func TestContextMemory_Load_RotatesExpiredSession(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewContextMemory(dir)
	old := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	c := &Context{
		Schema: ContextSchema, SessionID: "old-session",
		CreatedAt: old, LastUpdated: old,
		RecentTasks: []TaskSummary{{TaskID: "t1", Status: "completed"}},
	}
	if err := mem.Save(c); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := mem.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.SessionID == "old-session" {
		t.Fatal("session_id not rotated despite expired created_at")
	}
	// The RecentTasks should be preserved across rotation.
	if len(loaded.RecentTasks) != 1 || loaded.RecentTasks[0].TaskID != "t1" {
		t.Fatalf("recent_tasks not preserved on rotation: %+v", loaded.RecentTasks)
	}
	if !strings.HasPrefix(loaded.CreatedAt, time.Now().UTC().Format("2006-01-02")) {
		t.Errorf("created_at not refreshed: %s", loaded.CreatedAt)
	}
}

func TestContextMemory_RecordTask_PrependsAndCapsAt20(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewContextMemory(dir)
	for i := 0; i < 25; i++ {
		if err := mem.RecordTask(TaskSummary{
			TaskID: fmt.Sprintf("t-%02d", i),
			Status: "completed",
		}); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	c, _ := mem.Load()
	if len(c.RecentTasks) != MaxRecentTasks {
		t.Fatalf("want %d, got %d", MaxRecentTasks, len(c.RecentTasks))
	}
	// Newest first: most recent should be t-24, oldest kept t-05.
	if c.RecentTasks[0].TaskID != "t-24" {
		t.Errorf("newest = %s, want t-24", c.RecentTasks[0].TaskID)
	}
	if c.RecentTasks[len(c.RecentTasks)-1].TaskID != "t-05" {
		t.Errorf("oldest kept = %s, want t-05", c.RecentTasks[len(c.RecentTasks)-1].TaskID)
	}
}

func TestContextMemory_RecordTask_RunningAddsToOpen(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewContextMemory(dir)
	if err := mem.RecordTask(TaskSummary{TaskID: "running-1", Status: "running"}); err != nil {
		t.Fatalf("record: %v", err)
	}
	c, _ := mem.Load()
	if len(c.OpenTasks) != 1 || c.OpenTasks[0] != "running-1" {
		t.Fatalf("open_tasks = %+v", c.OpenTasks)
	}
}

func TestContextMemory_CloseTask_RemovesFromOpen(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewContextMemory(dir)
	_ = mem.RecordTask(TaskSummary{TaskID: "r-1", Status: "running"})
	_ = mem.RecordTask(TaskSummary{TaskID: "r-2", Status: "running"})
	if err := mem.CloseTask("r-1"); err != nil {
		t.Fatalf("close: %v", err)
	}
	c, _ := mem.Load()
	if len(c.OpenTasks) != 1 || c.OpenTasks[0] != "r-2" {
		t.Fatalf("open_tasks = %+v", c.OpenTasks)
	}
}

func TestContextMemory_RecordError_CapsAt20(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewContextMemory(dir)
	for i := 0; i < 25; i++ {
		_ = mem.RecordError(ErrorSummary{
			Timestamp: "x", Skill: "s", Action: "a", ErrorClass: "permanent",
		})
	}
	c, _ := mem.Load()
	if len(c.RecentErrors) != MaxRecentErrors {
		t.Fatalf("want %d, got %d", MaxRecentErrors, len(c.RecentErrors))
	}
}

func TestContextMemory_SetPreference_AddsAndDeletes(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewContextMemory(dir)
	if err := mem.SetPreference("default_region", "cn-north-4"); err != nil {
		t.Fatalf("set: %v", err)
	}
	c, _ := mem.Load()
	if c.Preferences["default_region"] != "cn-north-4" {
		t.Fatalf("preferences = %+v", c.Preferences)
	}
	if err := mem.SetPreference("default_region", ""); err != nil {
		t.Fatalf("delete: %v", err)
	}
	c, _ = mem.Load()
	if _, ok := c.Preferences["default_region"]; ok {
		t.Fatalf("key not deleted: %+v", c.Preferences)
	}
}

func TestContextMemory_BatchedMutations_DeferWriteUntilFlush(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewContextMemory(dir)
	path := filepath.Join(dir, ".l4-memory", "context.json")

	for i := 0; i < 5; i++ {
		if err := mem.RecordTask(TaskSummary{
			TaskID: fmt.Sprintf("t-%d", i), Status: "completed",
		}); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	_ = mem.RecordError(ErrorSummary{Timestamp: "x", Skill: "s", Action: "a", ErrorClass: "transient"})
	_ = mem.SetPreference("region", "cn-north-4")

	if !mem.Dirty() {
		t.Fatal("want Dirty=true after mutations")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("context.json written before Flush: err=%v", err)
	}

	// Same-handle Load sees queued mutations without Flush.
	c, err := mem.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(c.RecentTasks) != 5 {
		t.Fatalf("in-memory recent_tasks=%d, want 5", len(c.RecentTasks))
	}

	if err := mem.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if mem.Dirty() {
		t.Fatal("want Dirty=false after Flush")
	}
	if err := mem.Flush(); err != nil {
		t.Fatalf("second flush: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after flush: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("perm=%o, want 0600", info.Mode().Perm())
	}

	// Fresh handle sees the single batched write.
	mem2, _ := NewContextMemory(dir)
	c2, err := mem2.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(c2.RecentTasks) != 5 {
		t.Fatalf("persisted recent_tasks=%d, want 5", len(c2.RecentTasks))
	}
	if c2.Preferences["region"] != "cn-north-4" {
		t.Fatalf("preferences=%+v", c2.Preferences)
	}
}
