// Package learning also covers trace_learning.py: aggregating GCL trace files
// into a per-skill failure_patterns.json. This test pins the contract.
package learning

import (
	"path/filepath"
	"testing"
)

// TestSignatureKey_DedupContract asserts the (category, error, command) tuple
// is the dedup key. Two traces with the same tuple merge; different tuples
// produce separate patterns.
func TestSignatureKey_DedupContract(t *testing.T) {
	// Same inputs (incl. first token) → same key.
	a := SignatureKey("runtime", "ConnectionRefused", "hcloud ecs list-servers")
	b := SignatureKey("runtime", "ConnectionRefused", "hcloud ecs list-servers")
	// Different first token → different key (Python's contract: first token
	// is the dedup discriminator).
	c := SignatureKey("runtime", "ConnectionRefused", "aws rds list-instances")
	if a != b {
		t.Errorf("same inputs should yield same key: %q vs %q", a, b)
	}
	if a == c {
		t.Errorf("different first token should yield different key: %q vs %q", a, c)
	}
}

// TestExtractPatternFromTrace_NilForPASS asserts PASS traces (no
// final.failure_pattern) extract as nil — no learnable signal.
func TestExtractPatternFromTrace_NilForPASS(t *testing.T) {
	trace := map[string]any{
		"final": map[string]any{"decision": "pass"},
	}
	if got := ExtractPatternFromTrace(trace); got != nil {
		t.Errorf("PASS trace should yield nil, got %+v", got)
	}
}

// TestExtractPatternFromTrace_PresentForFAIL asserts a failure_pattern block
// is returned verbatim.
func TestExtractPatternFromTrace_PresentForFAIL(t *testing.T) {
	trace := map[string]any{
		"final": map[string]any{
			"decision": "fail",
			"failure_pattern": map[string]any{
				"category": "runtime",
				"error":    "Throttling: User",
				"command":  "hcloud ecs list-servers",
				"fix":      "Exponential backoff",
			},
		},
	}
	got := ExtractPatternFromTrace(trace)
	if got == nil {
		t.Fatal("FAIL trace should yield non-nil pattern")
	}
	if got["category"] != "runtime" {
		t.Errorf("category=%v, want runtime", got["category"])
	}
}

// TestMakePatternID_Sequential asserts ids are formatted with the
// supplied nextNum.
func TestMakePatternID_Sequential(t *testing.T) {
	got := MakePatternID("huaweicloud-ecs-ops", 4)
	if got != "ECS-FP004" {
		t.Errorf("got %q, want ECS-FP004", got)
	}
}

// TestMaxPatternID asserts the helper that callers use to feed
// MakePatternID scans existing patterns for the highest numeric suffix.
func TestMaxPatternID(t *testing.T) {
	patterns := []any{
		map[string]any{"id": "ECS-FP001"},
		map[string]any{"id": "ECS-FP003"},
		"not a map",
		map[string]any{"no_id": true},
	}
	if got := MaxPatternID(patterns); got != 3 {
		t.Errorf("got %d, want 3", got)
	}
	if got := MaxPatternID(nil); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

// TestCreatePatternEntry_Shape asserts the entry has the full schema
// expected by downstream consumers.
func TestCreatePatternEntry_Shape(t *testing.T) {
	fp := map[string]any{
		"category": "permission",
		"error":    "AccessDenied",
		"command":  "hcloud rds create-instance",
		"fix":      "Delegate to IAM",
	}
	entry := CreatePatternEntry(fp, "huaweicloud-rds-ops", 1, "trace-001.json")

	if entry["id"] != "RDS-FP001" {
		t.Errorf("id=%v, want RDS-FP001", entry["id"])
	}
	if entry["category"] != "permission" {
		t.Errorf("category=%v, want permission", entry["category"])
	}
	sig, _ := entry["signature"].(map[string]any)
	if sig == nil {
		t.Fatal("signature missing")
	}
	if sig["command_pattern"] != "hcloud" {
		t.Errorf("command_pattern=%v, want hcloud (first token)", sig["command_pattern"])
	}
	stats, _ := entry["stats"].(map[string]any)
	if stats == nil {
		t.Fatal("stats missing")
	}
	// After round-trip via JSON, Go ints come back as float64; but here we
	// are checking the in-memory map before marshal, so it's int.
	got, ok := stats["occurrence_count"].(int)
	if !ok || got != 1 {
		t.Errorf("stats.occurrence_count=%v (type %T), want int(1)", stats["occurrence_count"], stats["occurrence_count"])
	}
}

// TestScanTraces_EmptyDir asserts no panic and empty result when no audit-results.
func TestScanTraces_EmptyDir(t *testing.T) {
	root := t.TempDir()
	got := ScanTraces(root, "huaweicloud-ecs-ops", nil)
	if len(got) != 0 {
		t.Errorf("empty dir should yield 0 traces, got %d", len(got))
	}
}

// TestScanTraces_FiltersBySkill asserts traces with mismatched skill are skipped.
func TestScanTraces_FiltersBySkill(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	mustMkdir(t, audit)
	writeTrace(t, filepath.Join(audit, "gcl-trace-1.json"), "huaweicloud-ecs-ops")
	writeTrace(t, filepath.Join(audit, "gcl-trace-2.json"), "huaweicloud-rds-ops")

	got := ScanTraces(root, "huaweicloud-ecs-ops", nil)
	if len(got) != 1 {
		t.Errorf("expected 1 ecs trace, got %d", len(got))
	}
}

// TestAggregate_WritesUpdatedPatterns asserts the round-trip:
// scan → extract → merge → write produces a valid failure_patterns.json
// with new_count or updated_count > 0.
func TestAggregate_WritesUpdatedPatterns(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	mustMkdir(t, audit)
	writeTraceWithFailure(t, filepath.Join(audit, "gcl-trace-a.json"),
		"huaweicloud-ecs-ops", "runtime", "OOMKilled", "hcloud ecs list-servers")

	// Set up an empty failure_patterns.json so aggregate can append.
	mustMkdir(t, filepath.Join(root, "huaweicloud-ecs-ops", "assets"))
	if err := writeJSON(filepath.Join(root, "huaweicloud-ecs-ops", "assets", "failure_patterns.json"), map[string]any{
		"$schema":  "failure-patterns/v1",
		"skill_id": "huaweicloud-ecs-ops",
		"patterns": []any{},
		"meta":     map[string]any{"total_patterns": 0, "source_traces_analyzed": 0},
	}); err != nil {
		t.Fatalf("seed failure_patterns.json: %v", err)
	}

	res, err := Aggregate(root, "huaweicloud-ecs-ops", nil, false)
	if err != nil {
		t.Fatalf("Aggregate error: %v", err)
	}
	if res.NewCount != 1 {
		t.Errorf("NewCount=%d, want 1", res.NewCount)
	}
	if res.UpdatedCount != 0 {
		t.Errorf("UpdatedCount=%d, want 0", res.UpdatedCount)
	}
}

// --- Test helpers (these will be replaced by the package's own test helpers) ---

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := mkdirAll(p); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
}

func writeTrace(t *testing.T, p string, skill string) {
	t.Helper()
	if err := writeJSON(p, map[string]any{"skill": skill, "final": map[string]any{"decision": "pass"}}); err != nil {
		t.Fatalf("write trace: %v", err)
	}
}

func writeTraceWithFailure(t *testing.T, p string, skill, cat, errStr, cmd string) {
	t.Helper()
	err := writeJSON(p, map[string]any{
		"skill": skill,
		"final": map[string]any{
			"decision": "fail",
			"failure_pattern": map[string]any{
				"category": cat, "error": errStr, "command": cmd, "fix": "retry",
			},
		},
	})
	if err != nil {
		t.Fatalf("write failure trace: %v", err)
	}
}
