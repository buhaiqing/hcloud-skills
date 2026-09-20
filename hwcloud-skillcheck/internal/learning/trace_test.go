// Package learning also covers trace_learning.py: aggregating GCL trace files
// into a per-skill failure_patterns.json. This test pins the contract.
package learning

import (
	"os"
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

// TestIsSmokeTrace pins the smoke contract shared by this package and
// cmd/aggregate.go: a trace with no `final` block, a smoke request/fault, or
// zero executed steps carries no verifiable signal.
func TestIsSmokeTrace(t *testing.T) {
	final := map[string]any{"status": "PASS"}
	iter := map[string]any{"iter": float64(1)}
	cases := []struct {
		name  string
		trace map[string]any
		want  bool
	}{
		{"nil trace", nil, true},
		{"no final block (pre-fix orchestrator trace)", map[string]any{"skill": "s", "request": "RDS timeout"}, true},
		{"gcl smoke request", map[string]any{"final": final, "request": "smoke", "iterations": []any{iter}}, true},
		{"l4 smoke fault", map[string]any{"final": final, "fault": "SMOKE", "orchestration": map[string]any{"step_count": float64(3)}}, true},
		{"zero top-level step_count", map[string]any{"final": final, "step_count": float64(0), "iterations": []any{iter}}, true},
		{"zero orchestration step_count", map[string]any{"final": final, "orchestration": map[string]any{"step_count": float64(0)}}, true},
		{"empty iterations is a budget failure, not smoke", map[string]any{"final": final, "iterations": []any{}}, false},
		{"budget failure with zero iterations still counts", map[string]any{
			"final": map[string]any{"status": "SAFETY_FAIL"}, "iterations": []any{}, "request": "list servers",
		}, false},
		{"fault merely mentioning smoke", map[string]any{"final": final, "request": "smoke detected in rack 3", "iterations": []any{iter}}, false},
		{"healthy gcl trace", map[string]any{"final": final, "request": "list servers", "iterations": []any{iter}}, false},
		{"healthy l4 trace", map[string]any{"final": final, "fault": "RDS timeout", "orchestration": map[string]any{"step_count": float64(2)}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsSmokeTrace(tc.trace); got != tc.want {
				t.Errorf("IsSmokeTrace=%v, want %v", got, tc.want)
			}
		})
	}
}

// TestScanTraces_IncludesOrchestratorTraces asserts both writers' traces are
// scanned. The L4 loop's orchestrator-trace-*.json files were previously
// invisible to the learner, which left every failure_patterns.json at
// source_traces_analyzed=0.
func TestScanTraces_IncludesOrchestratorTraces(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	mustMkdir(t, audit)
	writeTrace(t, filepath.Join(audit, "gcl-trace-1.json"), "huaweicloud-ecs-ops")
	writeTrace(t, filepath.Join(audit, "orchestrator-trace-abc123.json"), "huaweicloud-ecs-ops")
	// A sibling audit-results artifact that is not a trace must stay out.
	if err := writeJSON(filepath.Join(audit, "gcl-quality-summary-20260701.json"), map[string]any{
		"skill": "huaweicloud-ecs-ops",
	}); err != nil {
		t.Fatalf("write summary fixture: %v", err)
	}

	got := ScanTraces(root, "huaweicloud-ecs-ops", nil)
	if len(got) != 2 {
		t.Fatalf("ScanTraces returned %d traces, want 2 (gcl + orchestrator)", len(got))
	}
	if filepath.Base(got[0].Path) != "gcl-trace-1.json" || filepath.Base(got[1].Path) != "orchestrator-trace-abc123.json" {
		t.Errorf("unexpected scan order: %s, %s", got[0].Path, got[1].Path)
	}
}

// TestAggregate_SmokeTracesExcluded asserts smoke traces never reach
// failure_patterns.json nor the source_traces_analyzed counter, even when they
// carry a failure_pattern block, while a real trace still merges.
func TestAggregate_SmokeTracesExcluded(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	mustMkdir(t, audit)

	const skill = "huaweicloud-ecs-ops"
	writeTraceWithFailure(t, filepath.Join(audit, "gcl-trace-real.json"),
		skill, "runtime", "OOMKilled", "hcloud ecs list-servers")

	smokePattern := func() map[string]any {
		return map[string]any{
			"category": "runtime",
			"skill":    skill,
			"command":  "hcloud ecs reboot-server",
			"error":    "structural critic verdict: MAX_ITER",
			"fix":      "review the planned command",
		}
	}
	// Smoke by fault token, despite having run steps.
	if err := writeJSON(filepath.Join(audit, "orchestrator-trace-smokefault.json"), map[string]any{
		"skill": skill, "source": "l4", "fault": "smoke",
		"orchestration": map[string]any{"step_count": float64(2)},
		"final":         map[string]any{"status": "MAX_ITER", "failure_pattern": smokePattern()},
	}); err != nil {
		t.Fatalf("write smoke-fault trace: %v", err)
	}
	// Smoke by zero steps.
	if err := writeJSON(filepath.Join(audit, "orchestrator-trace-nostep.json"), map[string]any{
		"skill": skill, "source": "l4", "fault": "RDS slow queries",
		"orchestration": map[string]any{"step_count": float64(0)},
		"final":         map[string]any{"status": "MAX_ITER", "failure_pattern": smokePattern()},
	}); err != nil {
		t.Fatalf("write zero-step trace: %v", err)
	}
	// The pre-fix orchestrator shape: no `final` block at all.
	if err := writeJSON(filepath.Join(audit, "orchestrator-trace-legacy.json"), map[string]any{
		"skill": skill, "source": "l4", "request": "RDS connection timeout",
		"orchestration": map[string]any{"step_count": float64(5)},
	}); err != nil {
		t.Fatalf("write legacy trace: %v", err)
	}

	mustMkdir(t, filepath.Join(root, skill, "assets"))
	if err := writeJSON(filepath.Join(root, skill, "assets", "failure_patterns.json"), map[string]any{
		"$schema":  "failure-patterns/v1",
		"skill_id": skill,
		"patterns": []any{},
		"meta":     map[string]any{"total_patterns": 0, "source_traces_analyzed": 0},
	}); err != nil {
		t.Fatalf("seed failure_patterns.json: %v", err)
	}

	res, err := Aggregate(root, skill, nil, false)
	if err != nil {
		t.Fatalf("Aggregate error: %v", err)
	}
	if res.Scanned != 1 {
		t.Errorf("Scanned=%d, want 1 (only the trace with verifiable signal)", res.Scanned)
	}
	if res.SkippedSmoke != 3 {
		t.Errorf("SkippedSmoke=%d, want 3 (smoke fault, zero steps, missing final)", res.SkippedSmoke)
	}
	if res.NewCount != 1 || res.UpdatedCount != 0 {
		t.Errorf("NewCount=%d UpdatedCount=%d, want 1/0", res.NewCount, res.UpdatedCount)
	}

	data := LoadFailurePatterns(root, skill)
	patterns, _ := data["patterns"].([]any)
	if len(patterns) != 1 {
		t.Fatalf("patterns=%d, want 1 (smoke traces must not add patterns)", len(patterns))
	}
	pm, _ := patterns[0].(map[string]any)
	if got, _ := pm["signature"].(map[string]any); got != nil {
		if got["error_message_regex"] != "OOMKilled" {
			t.Errorf("merged pattern error=%v, want the real trace's OOMKilled", got["error_message_regex"])
		}
	}
	meta, _ := data["meta"].(map[string]any)
	if got := toFloat(meta["source_traces_analyzed"]); got != 1 {
		t.Errorf("source_traces_analyzed=%v, want 1 (smoke traces excluded)", got)
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

// TestRecordFixOutcome_Success updates auto_fixed_count + success_rate.
func TestRecordFixOutcome_Success(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-ecs-ops", "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Seed a failure_patterns.json with one pattern, some prior failures.
	fixture := `{
		"$schema": "failure-patterns/v1",
		"skill_id": "huaweicloud-ecs-ops",
		"patterns": [{
			"id": "ECS-FP001",
			"category": "resource_state",
			"signature": {"error_code": "Ecs.0801", "error_message_regex": "InsufficientResource", "command_pattern": "create-server"},
			"fix": {"strategy": "fallback", "action": "Try different AZ"},
			"stats": {
				"occurrence_count": 10,
				"first_seen": "2026-07-01T00:00:00Z",
				"last_seen": "2026-07-31T00:00:00Z",
				"auto_fixed_count": 4,
				"escalated_count": 5,
				"success_rate": 0.5
			}
		}]
	}`
	if err := os.WriteFile(filepath.Join(dir, "failure_patterns.json"), []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RecordFixOutcome(root, "huaweicloud-ecs-ops", "ECS-FP001", true); err != nil {
		t.Fatalf("RecordFixOutcome(success): %v", err)
	}

	data := LoadFailurePatterns(root, "huaweicloud-ecs-ops")
	patterns := data["patterns"].([]any)
	pm := patterns[0].(map[string]any)
	stats := pm["stats"].(map[string]any)
	if got := toFloat(stats["auto_fixed_count"]); got != 5 {
		t.Errorf("auto_fixed_count=%v, want 5", got)
	}
	// success_rate recomputed = auto_fixed / (auto_fixed + escalated) = 5/10.
	if got := toFloat(stats["success_rate"]); got != 0.5 {
		t.Errorf("success_rate=%v, want 0.5 (5/10 after auto-fix)", got)
	}
}

// TestRecordFixOutcome_FailureDeRanks verifies a failed fix bumps
// escalated_count and lowers success_rate (de-ranks the fix).
func TestRecordFixOutcome_FailureDeRanks(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-ecs-ops", "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := `{
		"$schema": "failure-patterns/v1",
		"skill_id": "huaweicloud-ecs-ops",
		"patterns": [{
			"id": "ECS-FP002",
			"category": "resource_state",
			"signature": {"error_code": "X", "error_message_regex": "Y", "command_pattern": "*"},
			"fix": {"strategy": "retry", "action": "fix"},
			"stats": {"occurrence_count": 10, "auto_fixed_count": 5, "escalated_count": 5, "success_rate": 0.5}
		}]
	}`
	if err := os.WriteFile(filepath.Join(dir, "failure_patterns.json"), []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RecordFixOutcome(root, "huaweicloud-ecs-ops", "ECS-FP002", false); err != nil {
		t.Fatalf("RecordFixOutcome(failure): %v", err)
	}

	data := LoadFailurePatterns(root, "huaweicloud-ecs-ops")
	pm := data["patterns"].([]any)[0].(map[string]any)
	stats := pm["stats"].(map[string]any)
	if got := toFloat(stats["escalated_count"]); got != 6 {
		t.Errorf("escalated_count=%v, want 6", got)
	}
	if got := toFloat(stats["success_rate"]); got >= 0.5 {
		t.Errorf("success_rate=%v, want < 0.5 (de-ranked)", got)
	}
}
