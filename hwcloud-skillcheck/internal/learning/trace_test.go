// Package learning also covers trace_learning.py: aggregating GCL trace files
// into a per-skill failure_patterns.json. This test pins the contract.
package learning

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	// Trace-derived entries must be distinguishable from the curated knowledge
	// base (internal/learning/knowledge.go), which carries neither field and is
	// human-reviewed: these values are machine-learned and not yet verified.
	if entry["provenance"] != "trace" {
		t.Errorf("provenance=%v, want \"trace\"", entry["provenance"])
	}
	if verified, ok := entry["verified"].(bool); !ok || verified {
		t.Errorf("verified=%v (type %T), want bool(false)", entry["verified"], entry["verified"])
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
// zero executed steps carries no verifiable signal — unless its terminal status
// is SAFETY_FAIL, which always counts.
func TestIsSmokeTrace(t *testing.T) {
	final := map[string]any{"status": "PASS"}
	safetyFail := map[string]any{"status": "SAFETY_FAIL"}
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
		{"empty iterations is unverified, not evidence", map[string]any{"final": final, "iterations": []any{}}, true},
		{"budget failure with zero iterations still counts", map[string]any{
			"final": safetyFail, "iterations": []any{}, "request": "list servers",
		}, false},
		{"fault merely mentioning smoke", map[string]any{"final": final, "request": "smoke detected in rack 3", "iterations": []any{iter}}, false},
		{"healthy gcl trace", map[string]any{"final": final, "request": "list servers", "iterations": []any{iter}}, false},
		{"healthy l4 trace", map[string]any{"final": final, "fault": "RDS timeout", "orchestration": map[string]any{"step_count": float64(2)}}, false},
		// A real safety failure is never smoke, whatever else the trace says:
		// dropping it deleted the failure from pass_rate and from learning.
		{"smoke request + SAFETY_FAIL", map[string]any{"final": safetyFail, "request": "smoke", "iterations": []any{iter}}, false},
		{"zero steps + SAFETY_FAIL", map[string]any{"final": safetyFail, "step_count": float64(0)}, false},
		{"smoke request + PASS is smoke", map[string]any{"final": final, "request": "smoke", "iterations": []any{iter}}, true},
		{"explicit smoke marker", map[string]any{"final": final, "smoke": true, "request": "<masked>", "iterations": []any{iter}}, true},
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
// carry a failure_pattern block, while a real trace still merges. The
// pre-canonical orchestrator shape (no `final`) is schema-invalid, so it is
// counted as an invalid trace — the frozen order is schema-invalid before
// smoke.
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
	if err := writeJSON(filepath.Join(audit, "orchestrator-trace-smokefault.json"),
		l4Trace(skill, "smoke", 2, terminalFinal("MAX_ITER", smokePattern()))); err != nil {
		t.Fatalf("write smoke-fault trace: %v", err)
	}
	// Smoke by zero steps.
	if err := writeJSON(filepath.Join(audit, "orchestrator-trace-nostep.json"),
		l4Trace(skill, "RDS slow queries", 0, terminalFinal("MAX_ITER", smokePattern()))); err != nil {
		t.Fatalf("write zero-step trace: %v", err)
	}
	// The pre-canonical orchestrator shape: no `final` block at all, and none
	// of the canonical required fields.
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
	if res.SkippedSmoke != 2 {
		t.Errorf("SkippedSmoke=%d, want 2 (smoke fault, zero steps)", res.SkippedSmoke)
	}
	if res.InvalidTraces != 1 {
		t.Errorf("InvalidTraces=%d, want 1 (pre-canonical trace, no `final`)", res.InvalidTraces)
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

// schemaValidTrace builds a trace that conforms to the canonical trace schema
// (internal/embed.TraceSchema) — the shape both writers persist. Every fixture
// in this file must conform: Aggregate and ScanTraces classify non-conforming
// files as invalid and exclude them from learning, so a fixture missing the
// canonical fields would exercise the rejection path instead of the merge path
// it was written to pin.
func schemaValidTrace(skill string, final map[string]any) map[string]any {
	return map[string]any{
		"trace_schema_version": "v1",
		"skill":                skill,
		"request":              "list servers",
		"rubric_version":       "1.0",
		"masked_fields":        []any{},
		"iterations":           []any{schemaValidIteration("PASS")},
		"final":                final,
	}
}

// schemaValidIteration builds one conforming iteration (generator/critic/
// decision are all required).
func schemaValidIteration(decision string) map[string]any {
	return map[string]any{
		"iter": 1,
		"generator": map[string]any{
			"command": "hcloud ecs list-servers", "exit_code": 0, "result_excerpt": "ok",
			"stdout_len": 2, "stderr_len": 0,
			"args": map[string]any{"iter": 1, "critic_feedback": nil},
		},
		"critic": map[string]any{
			"scores": map[string]any{
				"correctness": 1.0, "safety": 1.0, "idempotency": 1.0,
				"traceability": 1.0, "spec_compliance": 1.0,
			},
			"suggestions": []any{},
			"blocking":    false,
		},
		"decision": decision,
	}
}

// terminalFinal builds a conforming `final` block (all four keys are required).
func terminalFinal(status string, failurePattern any) map[string]any {
	return map[string]any{
		"status": status, "iter": 1, "output": nil, "failure_pattern": failurePattern,
	}
}

// l4Trace builds a conforming l4 orchestrator trace with the given fault, step
// count, and final block.
func l4Trace(skill, fault string, stepCount int, final map[string]any) map[string]any {
	iterations := []any{}
	if stepCount > 0 {
		iterations = append(iterations, schemaValidIteration("RETRY"))
	}
	trace := schemaValidTrace(skill, final)
	trace["request"] = fault
	trace["fault"] = fault
	trace["source"] = "l4"
	trace["iterations"] = iterations
	trace["orchestration"] = map[string]any{"step_count": stepCount}
	return trace
}

func writeTrace(t *testing.T, p string, skill string) {
	t.Helper()
	if err := writeJSON(p, schemaValidTrace(skill, terminalFinal("PASS", nil))); err != nil {
		t.Fatalf("write trace: %v", err)
	}
}

func writeTraceWithFailure(t *testing.T, p string, skill, cat, errStr, cmd string) {
	t.Helper()
	trace := schemaValidTrace(skill, terminalFinal("MAX_ITER", map[string]any{
		"category": cat, "error": errStr, "command": cmd, "fix": "retry",
	}))
	trace["iterations"] = []any{schemaValidIteration("RETRY")}
	if err := writeJSON(p, trace); err != nil {
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

// --- Trace trust boundary (canonical-schema validation + pattern validation) ---

// TestValidateTracePattern pins the trust boundary of the write path. A
// failure_pattern extracted from a trace is merged into failure_patterns.json,
// and the knowledge base drives matchPreExecutionRisk, whose match SKIPS the
// planned step (SKIPPED_BY_PATTERN_RISK). Everything below is a shape a crafted
// trace could use to disable steps or smuggle text into the remediation path.
func TestValidateTracePattern(t *testing.T) {
	const skill = "huaweicloud-ecs-ops"
	valid := func() map[string]any {
		return map[string]any{
			"category": "runtime",
			"error":    "OOMKilled",
			"command":  "hcloud ecs list-servers",
			"fix":      "retry with a larger flavor",
		}
	}
	cases := []struct {
		name    string
		skill   string
		mutate  func(fp map[string]any)
		wantErr bool
	}{
		{"valid pattern", skill, func(map[string]any) {}, false},
		{
			// The first token of the command IS the CLI binary for every real
			// trace; command_pattern must stay allowed (only error_message_regex
			// is substring-matched against commands by matchPreExecutionRisk).
			"cli token as command_pattern is legal", skill, func(map[string]any) {}, false,
		},
		{"unknown category", skill, func(fp map[string]any) { fp["category"] = "wat" }, true},
		{"missing category", skill, func(fp map[string]any) { delete(fp, "category") }, true},
		{"non-string category", skill, func(fp map[string]any) { fp["category"] = 7 }, true},
		{"missing error", skill, func(fp map[string]any) { delete(fp, "error") }, true},
		{"empty error", skill, func(fp map[string]any) { fp["error"] = "" }, true},
		{"non-string error", skill, func(fp map[string]any) { fp["error"] = map[string]any{} }, true},
		{
			// Matches everything → every step skippable.
			"error_message_regex matching the empty string", skill,
			func(fp map[string]any) { fp["error"] = ".*" }, true,
		},
		{"error_message_regex does not compile", skill, func(fp map[string]any) { fp["error"] = "[" }, true},
		{
			// The live matcher substrings the first token of
			// error_message_regex into every planned command, so anchoring on
			// the CLI binary matches the whole plan.
			"error anchored on the CLI binary", skill, func(fp map[string]any) { fp["error"] = "hcloud" }, true,
		},
		{
			"error anchored on the product name", skill, func(fp map[string]any) { fp["error"] = "ECS" }, true,
		},
		{
			"command_pattern matching the empty string", skill,
			func(fp map[string]any) { fp["command"] = ".*" }, true,
		},
		{
			"control character in error", skill,
			func(fp map[string]any) { fp["error"] = "OOMKilled\x00PASS" }, true,
		},
		{
			"control character in fix", skill,
			func(fp map[string]any) { fp["fix"] = "retry\x1b[2J" }, true,
		},
		{
			"over-long error", skill,
			func(fp map[string]any) { fp["error"] = strings.Repeat("a", maxPatternStringLen+1) }, true,
		},
		{
			"over-long fix", skill,
			func(fp map[string]any) { fp["fix"] = strings.Repeat("a", maxPatternStringLen+1) }, true,
		},
		{
			// The skill id lands in the knowledge base and in logs.
			"control character in skill", "huaweicloud-ecs\x07-ops", func(map[string]any) {}, true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fp := valid()
			tc.mutate(fp)
			err := ValidateTracePattern(fp, tc.skill)
			if tc.wantErr && err == nil {
				t.Fatalf("ValidateTracePattern(%v, %q) = nil, want rejection", fp, tc.skill)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ValidateTracePattern(%v, %q) = %v, want nil", fp, tc.skill, err)
			}
		})
	}
	if err := ValidateTracePattern(nil, skill); err == nil {
		t.Error("a nil failure_pattern must be rejected")
	}
}

// TestAggregate_RejectsUntrustedPattern pins the safety Critic's second
// blocker: `learning trace aggregate` used to merge an attacker-chosen
// error_message_regex/command_pattern/fix.action from a trace into
// failure_patterns.json, after which matchPreExecutionRisk skipped every
// planned step that matched. The pattern below ("hcloud" — the token in every
// command) must be dropped, counted, and left out of the knowledge base
// byte-for-byte.
func TestAggregate_RejectsUntrustedPattern(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	mustMkdir(t, audit)
	const skill = "huaweicloud-ecs-ops"
	// Canonical-schema-valid on purpose: the trace survives the trace-level
	// trust boundary, so only pattern validation can stop the merge.
	writeTraceWithFailure(t, filepath.Join(audit, "gcl-trace-poison.json"),
		skill, "runtime", "hcloud", "hcloud")

	assets := filepath.Join(root, skill, "assets")
	mustMkdir(t, assets)
	patternsPath := filepath.Join(assets, "failure_patterns.json")
	if err := writeJSON(patternsPath, map[string]any{
		"$schema": "failure-patterns/v1", "skill_id": skill,
		"patterns": []any{map[string]any{
			"id": "ECS-FP001", "category": "resource_state",
			"signature": map[string]any{"error_message_regex": "OOMKilled", "command_pattern": "hcloud"},
			"fix":       map[string]any{"strategy": "retry", "action": "retry"},
			"stats":     map[string]any{"occurrence_count": 3},
		}},
		"meta": map[string]any{"total_patterns": 1, "source_traces_analyzed": 0},
	}); err != nil {
		t.Fatalf("seed failure_patterns.json: %v", err)
	}
	before, err := os.ReadFile(patternsPath)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Aggregate(root, skill, nil, false)
	if err != nil {
		t.Fatalf("Aggregate error: %v", err)
	}
	if res.RejectedPatterns != 1 {
		t.Errorf("RejectedPatterns=%d, want 1", res.RejectedPatterns)
	}
	if res.NewCount != 0 || res.UpdatedCount != 0 {
		t.Errorf("NewCount=%d UpdatedCount=%d, want 0/0 (nothing may be merged)", res.NewCount, res.UpdatedCount)
	}
	if res.Scanned != 1 {
		t.Errorf("Scanned=%d, want 1 (the trace itself is valid evidence)", res.Scanned)
	}
	after, err := os.ReadFile(patternsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("failure_patterns.json changed:\nbefore: %s\nafter:  %s", before, after)
	}
}

// TestAggregate_CountsInvalidTraces pins the trace-level trust boundary: a
// trace-shaped file that does not conform to the canonical schema is counted in
// InvalidTraces and merges nothing — even when it carries a failure_pattern
// block (that is the shape the Critic's crafted file used to reach the
// knowledge base).
func TestAggregate_CountsInvalidTraces(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	mustMkdir(t, audit)
	const skill = "huaweicloud-ecs-ops"
	// No trace_schema_version/rubric_version/masked_fields/iterations: invalid.
	if err := writeJSON(filepath.Join(audit, "orchestrator-trace-crafted.json"), map[string]any{
		"skill": skill,
		"final": map[string]any{
			"status": "MAX_ITER",
			"failure_pattern": map[string]any{
				"category": "runtime", "error": "OOMKilled",
				"command": "hcloud ecs list-servers", "fix": "retry",
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	res, err := Aggregate(root, skill, nil, false)
	if err != nil {
		t.Fatalf("Aggregate error: %v", err)
	}
	if res.InvalidTraces != 1 {
		t.Errorf("InvalidTraces=%d, want 1", res.InvalidTraces)
	}
	if res.Scanned != 0 || res.NewCount != 0 || res.UpdatedCount != 0 || res.RejectedPatterns != 0 {
		t.Errorf("Scanned=%d NewCount=%d UpdatedCount=%d RejectedPatterns=%d, want all 0",
			res.Scanned, res.NewCount, res.UpdatedCount, res.RejectedPatterns)
	}
}

// TestScanTraces_DropsInvalidTraces asserts the scan path applies the same
// trust boundary: an unverifiable trace never leaves ScanTraces.
func TestScanTraces_DropsInvalidTraces(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	mustMkdir(t, audit)
	const skill = "huaweicloud-ecs-ops"
	writeTrace(t, filepath.Join(audit, "gcl-trace-real.json"), skill)
	if err := writeJSON(filepath.Join(audit, "gcl-trace-crafted.json"), map[string]any{
		"skill": skill, "final": map[string]any{"status": "PASS"},
	}); err != nil {
		t.Fatal(err)
	}

	got := ScanTraces(root, skill, nil)
	if len(got) != 1 {
		t.Fatalf("ScanTraces returned %d traces, want 1 (the crafted one must be dropped)", len(got))
	}
	if filepath.Base(got[0].Path) != "gcl-trace-real.json" {
		t.Errorf("kept %s, want gcl-trace-real.json", got[0].Path)
	}
}

// TestAggregate_SafetyFailStaysCounted pins the third Critic finding at the
// learner: a trace whose request is the smoke marker but whose terminal status
// is SAFETY_FAIL must still be counted (and its failure pattern must still be
// learnable), because a real safety failure is never "nothing was verified".
func TestAggregate_SafetyFailStaysCounted(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	mustMkdir(t, audit)
	const skill = "huaweicloud-ecs-ops"
	trace := schemaValidTrace(skill, terminalFinal("SAFETY_FAIL", map[string]any{
		"category": "permission", "error": "privilege escalation blocked",
		"command": "hcloud ecs delete-server", "fix": "require explicit approval",
	}))
	trace["request"] = "smoke"
	trace["iterations"] = []any{schemaValidIteration("SAFETY_FAIL")}
	if err := writeJSON(filepath.Join(audit, "gcl-trace-smokefail.json"), trace); err != nil {
		t.Fatal(err)
	}

	res, err := Aggregate(root, skill, nil, true)
	if err != nil {
		t.Fatalf("Aggregate error: %v", err)
	}
	if res.Scanned != 1 {
		t.Errorf("Scanned=%d, want 1 (a SAFETY_FAIL is real signal)", res.Scanned)
	}
	if res.SkippedSmoke != 0 {
		t.Errorf("SkippedSmoke=%d, want 0", res.SkippedSmoke)
	}
	if res.InvalidTraces != 0 {
		t.Errorf("InvalidTraces=%d, want 0", res.InvalidTraces)
	}
	if res.NewCount != 1 {
		t.Errorf("NewCount=%d, want 1 (its failure pattern is learnable)", res.NewCount)
	}
}

// --- Loop-health observability (skill-mismatch counter + EmptyLoop WARN) ---

// TestAggregate_CountsSkillMismatch pins the new SkippedSkillMismatch counter:
// a writer-attributed trace whose `skill` does not match --skill is now
// counted instead of silently dropped, so a misconfigured writer surfaces in
// the CLI summary. The pre-fix code did `if trace["skill"] != skill { continue }`
// ahead of `res.Scanned++`, which produced source_traces_analyzed=0 across
// every skill and hid the bug.
func TestAggregate_CountsSkillMismatch(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	mustMkdir(t, audit)
	const skill = "huaweicloud-ecs-ops"
	// Real, evidence-classified trace but with skill="unknown" — the exact
	// shape the L4 writer used to emit before orchestrator.resolveTraceSkill.
	writeTrace(t, filepath.Join(audit, "gcl-trace-misattr.json"), "unknown")
	// One trace for the skill we actually asked for, to prove the matcher
	// still picks it up.
	writeTraceWithFailure(t, filepath.Join(audit, "gcl-trace-good.json"),
		skill, "runtime", "OOMKilled", "hcloud ecs list-servers")

	mustMkdir(t, filepath.Join(root, skill, "assets"))
	if err := writeJSON(filepath.Join(root, skill, "assets", "failure_patterns.json"), map[string]any{
		"$schema":  "failure-patterns/v1",
		"skill_id": skill,
		"patterns": []any{},
		"meta":     map[string]any{"total_patterns": 0, "source_traces_analyzed": 0},
	}); err != nil {
		t.Fatalf("seed failure_patterns.json: %v", err)
	}

	res, err := Aggregate(root, skill, nil, true) // dry-run — no write side-effects
	if err != nil {
		t.Fatalf("Aggregate error: %v", err)
	}
	if res.SkippedSkillMismatch != 1 {
		t.Errorf("SkippedSkillMismatch=%d, want 1 (the unknown trace)", res.SkippedSkillMismatch)
	}
	if res.Scanned != 1 {
		t.Errorf("Scanned=%d, want 1 (only the matching trace contributes)", res.Scanned)
	}
	if res.NewCount != 1 {
		t.Errorf("NewCount=%d, want 1", res.NewCount)
	}
}

// TestAggregate_EmptyLoopWarnsOnAllMismatch pins the EmptyLoop alarm: a
// non-empty trace set whose every file was dropped at the skill-mismatch gate
// must (a) set AggregateResult.EmptyLoop, (b) emit exactly one WARN to stderr
// naming skill / counter / possible causes, and (c) do so under dry-run so a
// preview of an empty loop is loud before any file is written.
func TestAggregate_EmptyLoopWarnsOnAllMismatch(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	mustMkdir(t, audit)
	const skill = "huaweicloud-ecs-ops"
	// Three traces, all with skill="unknown" — the silent-attribute shape.
	writeTrace(t, filepath.Join(audit, "gcl-trace-1.json"), "unknown")
	writeTrace(t, filepath.Join(audit, "gcl-trace-2.json"), "unknown")
	writeTrace(t, filepath.Join(audit, "gcl-trace-3.json"), "unknown")

	mustMkdir(t, filepath.Join(root, skill, "assets"))
	if err := writeJSON(filepath.Join(root, skill, "assets", "failure_patterns.json"), map[string]any{
		"$schema":  "failure-patterns/v1",
		"skill_id": skill,
		"patterns": []any{},
		"meta":     map[string]any{"total_patterns": 0, "source_traces_analyzed": 0},
	}); err != nil {
		t.Fatalf("seed failure_patterns.json: %v", err)
	}

	stderr := captureStderr(t, func() {
		res, err := Aggregate(root, skill, nil, true) // dry-run, no write
		if err != nil {
			t.Fatalf("Aggregate error: %v", err)
		}
		if !res.EmptyLoop {
			t.Errorf("EmptyLoop=false, want true (every trace was skill-mismatched)")
		}
		if res.SkippedSkillMismatch != 3 {
			t.Errorf("SkippedSkillMismatch=%d, want 3", res.SkippedSkillMismatch)
		}
		if res.Scanned != 0 {
			t.Errorf("Scanned=%d, want 0", res.Scanned)
		}
	})

	// The WARN must name the skill and the counter that explains the miss,
	// and must not be silent under dry-run.
	for _, want := range []string{
		"WARN:",
		`skill "huaweicloud-ecs-ops"`,
		"skipped_skill_mismatch=3",
		"failure_patterns.json will NOT grow",
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr must contain %q; got:\n%s", want, stderr)
		}
	}
}

// TestAggregate_EmptyLoopSilentOnNoFiles asserts the alarm does NOT fire
// when there are simply no trace files at all — that is "nothing to do", not
// "loop broken". This is what makes EmptyLoop a usable signal: it only
// lights up when a non-empty input produced nothing.
func TestAggregate_EmptyLoopSilentOnNoFiles(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "audit-results")) // exists, empty
	const skill = "huaweicloud-ecs-ops"
	mustMkdir(t, filepath.Join(root, skill, "assets"))
	if err := writeJSON(filepath.Join(root, skill, "assets", "failure_patterns.json"), map[string]any{
		"$schema":  "failure-patterns/v1",
		"skill_id": skill,
		"patterns": []any{},
		"meta":     map[string]any{"total_patterns": 0, "source_traces_analyzed": 0},
	}); err != nil {
		t.Fatalf("seed failure_patterns.json: %v", err)
	}

	stderr := captureStderr(t, func() {
		res, err := Aggregate(root, skill, nil, true)
		if err != nil {
			t.Fatalf("Aggregate error: %v", err)
		}
		if res.EmptyLoop {
			t.Errorf("EmptyLoop=true on an empty trace directory, want false")
		}
	})
	if strings.Contains(stderr, "WARN: learning loop consumed 0 traces") {
		t.Errorf("EmptyLoop WARN must NOT fire on an empty audit-results/; got:\n%s", stderr)
	}
}

// TestAggregate_EmptyLoopAlsoFiresOnAllSmoke mirrors the alarm's other
// common shape: every file is classified as smoke, so even when --skill is
// correct and the writer is correct, source_traces_analyzed still does not
// grow. The operator needs the same warning to act on it.
func TestAggregate_EmptyLoopAlsoFiresOnAllSmoke(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	mustMkdir(t, audit)
	const skill = "huaweicloud-ecs-ops"
	// Smoke by request token — IsSmokeTrace must classify it as smoke.
	writeTraceWithRequest(t, filepath.Join(audit, "gcl-trace-smoke.json"),
		skill, "smoke", terminalFinal("PASS", nil))

	mustMkdir(t, filepath.Join(root, skill, "assets"))
	if err := writeJSON(filepath.Join(root, skill, "assets", "failure_patterns.json"), map[string]any{
		"$schema":  "failure-patterns/v1",
		"skill_id": skill,
		"patterns": []any{},
		"meta":     map[string]any{"total_patterns": 0, "source_traces_analyzed": 0},
	}); err != nil {
		t.Fatalf("seed failure_patterns.json: %v", err)
	}

	stderr := captureStderr(t, func() {
		res, err := Aggregate(root, skill, nil, true)
		if err != nil {
			t.Fatalf("Aggregate error: %v", err)
		}
		if !res.EmptyLoop {
			t.Errorf("EmptyLoop=false, want true (every trace was smoke)")
		}
		if res.SkippedSmoke != 1 {
			t.Errorf("SkippedSmoke=%d, want 1", res.SkippedSmoke)
		}
	})
	if !strings.Contains(stderr, "skipped_smoke=1") {
		t.Errorf("EmptyLoop WARN must name skipped_smoke when smoke is the cause; got:\n%s", stderr)
	}
}

// writeTraceWithRequest is the request-token variant of writeTraceWithFailure
// (the latter hardcodes a failure pattern; smoke traces have a PASS terminal).
func writeTraceWithRequest(t *testing.T, p string, skill, request string, final map[string]any) {
	t.Helper()
	trace := schemaValidTrace(skill, final)
	trace["request"] = request
	if err := writeJSON(p, trace); err != nil {
		t.Fatalf("write trace: %v", err)
	}
}

// captureStderr redirects os.Stderr to a buffer for the duration of fn, then
// restores the original writer. Used to assert that Aggregate's EmptyLoop WARN
// reaches stderr without the test runner losing its own diagnostics.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = orig }()
	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()
	fn()
	_ = w.Close()
	<-done
	return buf.String()
}
