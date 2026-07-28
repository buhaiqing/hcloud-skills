package l4

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOutcomeRecord_RoundTrip(t *testing.T) {
	in := OutcomeRecord{
		ID:           "fixed-uuid-for-test",
		Timestamp:    "2026-07-28T08:30:00Z",
		TaskID:       "task-1",
		Skill:        "huaweicloud-ecs-ops",
		Action:       "delete-instances",
		ContextHash:  "deadbeef",
		Outcome:      "failure",
		ErrorClass:   "transient",
		ErrorMsg:     "connection reset",
		RetryCount:   1,
		DurationMS:   4321,
		Risk:         "high",
		RBACDecision: "allowed",
		GCLDecision:  "PASS",
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out OutcomeRecord
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch:\n got=%+v\nwant=%+v", out, in)
	}
}

func TestOutcomeMemory_RecordAppendsJSONL(t *testing.T) {
	dir := t.TempDir()
	mem, err := NewOutcomeMemory(dir)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	rec := OutcomeRecord{ID: "a", Timestamp: "2026-07-28T00:00:00Z", Skill: "s", Action: "x"}
	if err := mem.Record(rec); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := mem.Record(OutcomeRecord{ID: "b", Timestamp: "2026-07-28T00:00:01Z", Skill: "s", Action: "x"}); err != nil {
		t.Fatalf("record2: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".l4-memory", "outcomes.jsonl"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := bytes.Split(bytes.TrimRight(raw, "\n"), []byte{'\n'})
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d (%q)", len(lines), raw)
	}
	for i, want := range []string{`"id":"a"`, `"id":"b"`} {
		if !bytes.Contains(lines[i], []byte(want)) {
			t.Fatalf("line %d missing %q: %s", i, want, lines[i])
		}
	}
}

func TestOutcomeMemory_DirAndFileMode(t *testing.T) {
	dir := t.TempDir()
	mem, err := NewOutcomeMemory(dir)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := mem.Record(OutcomeRecord{ID: "x"}); err != nil {
		t.Fatalf("record: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, ".l4-memory"))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Fatalf("dir perm = %o, want 0700", perm)
	}
	info, err = os.Stat(filepath.Join(dir, ".l4-memory", "outcomes.jsonl"))
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("file perm = %o, want 0600", perm)
	}
}
