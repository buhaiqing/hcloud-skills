package l4

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestOutcomeMemory_RecentOutcomes(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewOutcomeMemory(dir)
	records := []OutcomeRecord{
		{ID: "1", Timestamp: "2026-07-28T00:00:00Z", Skill: "huaweicloud-ecs-ops", Action: "list", Outcome: "success"},
		{ID: "2", Timestamp: "2026-07-28T00:00:01Z", Skill: "huaweicloud-ecs-ops", Action: "list", Outcome: "failure"},
		{ID: "3", Timestamp: "2026-07-28T00:00:02Z", Skill: "huaweicloud-rds-ops", Action: "list", Outcome: "success"},
		{ID: "4", Timestamp: "2026-07-28T00:00:03Z", Skill: "huaweicloud-ecs-ops", Action: "delete", Outcome: "failure"},
	}
	for _, r := range records {
		if err := mem.Record(r); err != nil {
			t.Fatalf("record %s: %v", r.ID, err)
		}
	}
	got, err := mem.RecentOutcomes("huaweicloud-ecs-ops", "list", 10)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 records, got %d", len(got))
	}
	if got[0].ID != "2" || got[1].ID != "1" {
		t.Fatalf("want [2,1], got [%s,%s]", got[0].ID, got[1].ID)
	}
}

func TestOutcomeMemory_MatchOutcomes_Lookback(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewOutcomeMemory(dir)
	old := OutcomeRecord{ID: "old", Timestamp: "2020-01-01T00:00:00Z", Skill: "s", Action: "x", ContextHash: "h", Outcome: "failure"}
	fresh := OutcomeRecord{ID: "fresh", Timestamp: time.Now().UTC().Format(time.RFC3339), Skill: "s", Action: "x", ContextHash: "h", Outcome: "success"}
	_ = mem.Record(old)
	_ = mem.Record(fresh)
	got, err := mem.MatchOutcomes("s", "x", "h", 24*time.Hour)
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if len(got) != 1 || got[0].ID != "fresh" {
		t.Fatalf("want [fresh], got %+v", got)
	}
}

func TestOutcomeMemory_PruneOlderThan(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewOutcomeMemory(dir)
	old := OutcomeRecord{ID: "old", Timestamp: "2020-01-01T00:00:00Z", Skill: "s", Action: "x", Outcome: "failure"}
	fresh := OutcomeRecord{ID: "fresh", Timestamp: "2026-07-01T00:00:00Z", Skill: "s", Action: "x", Outcome: "success"}
	_ = mem.Record(old)
	_ = mem.Record(fresh)

	cutoff, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
	dropped, err := mem.PruneOlderThan(cutoff)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if dropped != 1 {
		t.Fatalf("want 1 dropped, got %d", dropped)
	}

	got, _ := mem.RecentOutcomes("s", "x", 10)
	if len(got) != 1 || got[0].ID != "fresh" {
		t.Fatalf("want [fresh], got %+v", got)
	}
}

func TestOutcomeMemory_RecentCache_SkipsRescan(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewOutcomeMemory(dir)
	for i := 0; i < 5; i++ {
		_ = mem.Record(OutcomeRecord{
			ID:        string(rune('a' + i)),
			Timestamp: time.Date(2026, 7, 28, 0, 0, i, 0, time.UTC).Format(time.RFC3339),
			Skill:     "huaweicloud-ecs-ops",
			Action:    "list",
			Outcome:   "success",
		})
	}
	base := mem.FullScans() // may include prune-on-open scan (0 when empty)

	got, err := mem.RecentOutcomes("huaweicloud-ecs-ops", "list", 3)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
	afterLoad := mem.FullScans()
	if afterLoad != base+1 {
		t.Fatalf("first RecentOutcomes should full-scan once: base=%d got=%d", base, afterLoad)
	}

	got2, _ := mem.RecentOutcomes("huaweicloud-ecs-ops", "list", 3)
	if mem.FullScans() != afterLoad {
		t.Fatalf("second RecentOutcomes must hit cache; scans %d → %d", afterLoad, mem.FullScans())
	}
	if got2[0].ID != got[0].ID {
		t.Fatalf("cache drift: %s vs %s", got2[0].ID, got[0].ID)
	}

	_ = mem.Record(OutcomeRecord{
		ID: "new", Timestamp: "2026-07-28T01:00:00Z",
		Skill: "huaweicloud-ecs-ops", Action: "list", Outcome: "failure",
	})
	got3, _ := mem.RecentOutcomes("huaweicloud-ecs-ops", "list", 3)
	if mem.FullScans() != afterLoad {
		t.Fatalf("Record+Recent must not rescan warm key; scans=%d", mem.FullScans())
	}
	if got3[0].ID != "new" {
		t.Fatalf("warm cache should prepend Record; got %+v", got3)
	}
}

func TestOutcomeMemory_RecentCache_CapsAt100(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewOutcomeMemory(dir)
	for i := 0; i < 150; i++ {
		_ = mem.Record(OutcomeRecord{
			ID:        fmtID(i),
			Timestamp: time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC).Format(time.RFC3339),
			Skill:     "s", Action: "a", Outcome: "success",
		})
	}
	got, err := mem.RecentOutcomes("s", "a", outcomeKeyCacheSize)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(got) != outcomeKeyCacheSize {
		t.Fatalf("want cache cap %d, got %d", outcomeKeyCacheSize, len(got))
	}
	// Newest first: last recorded id.
	if got[0].ID != fmtID(149) {
		t.Fatalf("newest=%s, want %s", got[0].ID, fmtID(149))
	}
}

func TestOutcomeMemory_RecentOutcomes_NLeZeroUncapped(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewOutcomeMemory(dir)
	for i := 0; i < 150; i++ {
		_ = mem.Record(OutcomeRecord{
			ID:        fmtID(i),
			Timestamp: time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC).Format(time.RFC3339),
			Skill:     "s", Action: "a", Outcome: "success",
		})
	}
	got, err := mem.RecentOutcomes("s", "a", 0)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(got) != 150 {
		t.Fatalf("n<=0 must return all matches uncapped; got %d", len(got))
	}
	if got[0].ID != fmtID(149) {
		t.Fatalf("newest=%s, want %s", got[0].ID, fmtID(149))
	}
}

func fmtID(i int) string {
	return time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC).Format("id-20060102T150405")
}
