package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAggregateTraceGood(t *testing.T) {
	root := t.TempDir()
	writeTraceJSON(t, root, "gcl-trace-20260701-000000.json", traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0))
	writeTraceJSON(t, root, "gcl-trace-20260701-000001.json", traceFixture("huaweicloud-ecs-ops", "PASS", 2, 1.0))
	writeTraceJSON(t, root, "gcl-trace-20260701-000002.json", traceFixture("huaweicloud-rds-ops", "SAFETY_FAIL", 3, 0.0))

	out := filepath.Join(root, "summary-out.json")
	if err := runAggregate([]string{"trace", "--root", root, "--output", out}); err != nil {
		t.Fatalf("aggregate should succeed, got: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var summary map[string]any
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	totals, ok := summary["totals"].(map[string]any)
	if !ok {
		t.Fatalf("missing totals in summary: %v", summary)
	}
	if numOf(totals["total_runs"]) != 3 {
		t.Fatalf("expected total_runs=3, got %v", totals["total_runs"])
	}
	if numOf(totals["PASS"]) != 2 {
		t.Fatalf("expected PASS=2, got %v", totals["PASS"])
	}
	if numOf(totals["SAFETY_FAIL"]) != 1 {
		t.Fatalf("expected SAFETY_FAIL=1, got %v", totals["SAFETY_FAIL"])
	}
	if diff := summary["pass_rate"].(float64) - 2.0/3.0; diff > 0.0001 || diff < -0.0001 {
		t.Fatalf("expected pass_rate~=%.4f, got %v", 2.0/3.0, summary["pass_rate"])
	}
	bySkill, ok := summary["by_skill"].(map[string]any)
	if !ok {
		t.Fatalf("missing by_skill: %v", summary)
	}
	if _, ok := bySkill["huaweicloud-ecs-ops"]; !ok {
		t.Fatal("expected huaweicloud-ecs-ops in by_skill")
	}
}

func TestAggregateTraceNoFiles(t *testing.T) {
	root := t.TempDir()
	// No audit-results/gcl-trace-*.json => WARN and skip, exit 0 (Spec §4).
	if err := runAggregate([]string{"trace", "--root", root}); err != nil {
		t.Fatalf("no-trace aggregate should WARN+exit 0, got: %v", err)
	}
}

func TestAggregateTraceSinceHours(t *testing.T) {
	root := t.TempDir()
	// A fresh trace should be picked up by --since-hours 1.
	writeTraceJSON(t, root, "gcl-trace-recent.json", traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0))
	if err := runAggregate([]string{"trace", "--root", root, "--since-hours", "1"}); err != nil {
		t.Fatalf("recent trace should aggregate, got: %v", err)
	}
}

func TestAggregateTraceSelfCheck(t *testing.T) {
	// --self-check aggregates the embedded healthy trace fixture and must
	// produce a well-formed summary (total_runs>=1, pass_rate in [0,1]).
	out := filepath.Join(t.TempDir(), "self-summary.json")
	if err := runAggregate([]string{"trace", "--self-check", "--output", out}); err != nil {
		t.Fatalf("self-check should succeed, got: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var summary map[string]any
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	totals, ok := summary["totals"].(map[string]any)
	if !ok {
		t.Fatalf("missing totals in self-check summary: %v", summary)
	}
	if numOf(totals["total_runs"]) < 1 {
		t.Fatalf("expected total_runs>=1, got %v", totals["total_runs"])
	}
	passRate, ok := summary["pass_rate"].(float64)
	if !ok || passRate < 0 || passRate > 1 {
		t.Fatalf("expected pass_rate in [0,1], got %v", summary["pass_rate"])
	}
}

func traceFixture(skill, status string, iters int, score float64) string {
	return traceFixtureWithRequest(skill, "list servers for the aggregation fixture", status, iters, score)
}

func traceFixtureWithRequest(skill, request, status string, iters int, score float64) string {
	// Build a minimal valid trace with `iters` iterations, each PASS at score.
	type iter struct {
		Iter   int `json:"iter"`
		Critic struct {
			Scores map[string]float64 `json:"scores"`
		} `json:"critic"`
		Decision string `json:"decision"`
	}
	iterations := make([]iter, 0, iters)
	for i := 1; i <= iters; i++ {
		var c struct {
			Scores map[string]float64 `json:"scores"`
		}
		c.Scores = map[string]float64{
			"correctness": score, "safety": score, "idempotency": score,
			"traceability": score, "spec_compliance": score,
		}
		iterations = append(iterations, iter{Iter: i, Critic: c, Decision: status})
	}
	b, _ := json.Marshal(map[string]any{
		"skill":      skill,
		"request":    request,
		"iterations": iterations,
		"final":      map[string]any{"status": status, "iter": iters},
	})
	return string(b)
}

// l4TraceFixture builds the payload internal/l4.HandleFault writes to
// audit-results/orchestrator-trace-<faultID>.json: source:"l4", a `fault`
// field, orchestration.step_count, one dry-run iteration carrying the real
// structural-critic scores, and a `final` block.
func l4TraceFixture(skill, fault, status string, stepCount int, scores map[string]float64) string {
	iterations := []any{}
	if stepCount > 0 {
		iterations = append(iterations, map[string]any{
			"iter": 1,
			"generator": map[string]any{
				"command": "hcloud ecs list-servers", "exit_code": 0, "result_excerpt": "dry-run",
			},
			"critic":   map[string]any{"scores": scores, "suggestions": []any{}, "blocking": false},
			"decision": "PASS",
		})
	}
	b, _ := json.Marshal(map[string]any{
		"trace_id":      "0123456789abcdef",
		"skill":         skill,
		"request":       fault,
		"fault":         fault,
		"source":        "l4",
		"iterations":    iterations,
		"orchestration": map[string]any{"step_count": stepCount},
		"final": map[string]any{
			"status": status, "iter": 1, "output": "hcloud ecs list-servers",
			"dimensions": scores, "overall": 0.9, "critic_type": "structural",
			"failure_pattern": nil,
		},
	})
	return string(b)
}

// readSummary reads a written aggregate summary as a generic map.
func readSummary(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var summary map[string]any
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	return summary
}

// TestAggregateTraceConsumesOrchestratorTraces pins the P0 fix: orchestrator
// traces are read (previously only gcl-trace-*.json was globbed, so all L4
// traces had zero consumers) and their real rubric scores reach the summary.
func TestAggregateTraceConsumesOrchestratorTraces(t *testing.T) {
	root := t.TempDir()
	scores := map[string]float64{
		"correctness": 1.0, "safety": 1.0, "idempotency": 0.5,
		"traceability": 1.0, "spec_compliance": 1.0,
	}
	writeTraceJSON(t, root, "orchestrator-trace-0123456789abcdef.json",
		l4TraceFixture("huaweicloud-ecs-ops", "RDS connection timeout", "PASS", 2, scores))

	out := filepath.Join(root, "summary.json")
	if err := runAggregate([]string{"trace", "--root", root, "--output", out}); err != nil {
		t.Fatalf("aggregate should consume the orchestrator trace, got: %v", err)
	}
	summary := readSummary(t, out)

	totals, _ := summary["totals"].(map[string]any)
	if numOf(totals["total_runs"]) != 1 {
		t.Errorf("expected total_runs=1, got %v", totals["total_runs"])
	}
	if numOf(totals["PASS"]) != 1 {
		t.Errorf("expected PASS=1, got %v", totals["PASS"])
	}
	if got := summary["pass_rate"].(float64); got != 1.0 {
		t.Errorf("pass_rate=%v, want 1.0", got)
	}
	bySource, _ := summary["by_source"].(map[string]any)
	if numOf(bySource["l4"]) != 1 || numOf(bySource["gcl"]) != 0 {
		t.Errorf("by_source=%v, want {gcl:0, l4:1}", bySource)
	}
	if numOf(summary["skipped_smoke"]) != 0 {
		t.Errorf("skipped_smoke=%v, want 0", summary["skipped_smoke"])
	}
	avg, _ := summary["avg_rubric_scores"].(map[string]any)
	if got := avg["idempotency"].(float64); got != 0.5 {
		t.Errorf("avg idempotency=%v, want 0.5 (structural critic), not a hardcoded literal", got)
	}
	if got := avg["spec_compliance"].(float64); got != 1.0 {
		t.Errorf("avg spec_compliance=%v, want 1.0", got)
	}
	bySkill, _ := summary["by_skill"].(map[string]any)
	bucket, _ := bySkill["huaweicloud-ecs-ops"].(map[string]any)
	if bucket == nil || numOf(bucket["total"]) != 1 {
		t.Errorf("by_skill=%v, want huaweicloud-ecs-ops total 1", bySkill)
	}
}

// TestAggregateTraceLegacySourceAbsentIsGCL pins backward compatibility: a
// gcl trace written before the `source` field existed is still aggregated and
// counted as gcl.
func TestAggregateTraceLegacySourceAbsentIsGCL(t *testing.T) {
	root := t.TempDir()
	writeTraceJSON(t, root, "gcl-trace-20260101-000000.json",
		traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0))

	out := filepath.Join(root, "summary.json")
	if err := runAggregate([]string{"trace", "--root", root, "--output", out}); err != nil {
		t.Fatalf("aggregate should consume a source-less gcl trace, got: %v", err)
	}
	summary := readSummary(t, out)

	bySource, _ := summary["by_source"].(map[string]any)
	if numOf(bySource["gcl"]) != 1 || numOf(bySource["l4"]) != 0 {
		t.Errorf("by_source=%v, want {gcl:1, l4:0}", bySource)
	}
	if numOf(summary["skipped_smoke"]) != 0 {
		t.Errorf("skipped_smoke=%v, want 0 for a trace with a final block", summary["skipped_smoke"])
	}
	if numOf(summary["totals"].(map[string]any)["total_runs"]) != 1 {
		t.Errorf("total_runs=%v, want 1", summary["totals"])
	}
}

// TestAggregateTraceSmokeFiltered asserts the four smoke shapes are counted in
// skipped_smoke and excluded from pass_rate, by_source, and by_skill.
func TestAggregateTraceSmokeFiltered(t *testing.T) {
	root := t.TempDir()
	scores := map[string]float64{
		"correctness": 1.0, "safety": 1.0, "idempotency": 0.5,
		"traceability": 1.0, "spec_compliance": 1.0,
	}
	// The only trace with verifiable signal.
	writeTraceJSON(t, root, "gcl-trace-real.json",
		traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0))
	// request == "smoke" (the smoke gate's gcl run).
	writeTraceJSON(t, root, "gcl-trace-smoke.json",
		traceFixtureWithRequest("huaweicloud-ecs-ops", "smoke", "PASS", 1, 1.0))
	// fault == "smoke" on an l4 trace that did run steps.
	writeTraceJSON(t, root, "orchestrator-trace-smokefault.json",
		l4TraceFixture("huaweicloud-ecs-ops", "smoke", "PASS", 2, scores))
	// step_count == 0: nothing was verified.
	writeTraceJSON(t, root, "orchestrator-trace-nostep.json",
		l4TraceFixture("huaweicloud-ecs-ops", "RDS slow queries", "MAX_ITER", 0, scores))
	// no `final` at all: the pre-fix orchestrator shape.
	legacy, _ := json.Marshal(map[string]any{
		"skill":         "huaweicloud-ecs-ops",
		"request":       "RDS connection timeout",
		"source":        "l4",
		"orchestration": map[string]any{"step_count": 5},
	})
	writeTraceJSON(t, root, "orchestrator-trace-legacy.json", string(legacy))

	out := filepath.Join(root, "summary.json")
	if err := runAggregate([]string{"trace", "--root", root, "--output", out}); err != nil {
		t.Fatalf("aggregate failed: %v", err)
	}
	summary := readSummary(t, out)

	totals, _ := summary["totals"].(map[string]any)
	if numOf(totals["total_runs"]) != 1 {
		t.Errorf("total_runs=%v, want 1 (only the non-smoke trace counts)", totals["total_runs"])
	}
	if got := summary["pass_rate"].(float64); got != 1.0 {
		t.Errorf("pass_rate=%v, want 1.0 (smoke traces must not move it)", got)
	}
	if numOf(summary["skipped_smoke"]) != 4 {
		t.Errorf("skipped_smoke=%v, want 4 (smoke request, smoke fault, zero steps, missing final)", summary["skipped_smoke"])
	}
	bySource, _ := summary["by_source"].(map[string]any)
	if numOf(bySource["gcl"]) != 1 || numOf(bySource["l4"]) != 0 {
		t.Errorf("by_source=%v, want {gcl:1, l4:0}", bySource)
	}
	bySkill, _ := summary["by_skill"].(map[string]any)
	bucket, _ := bySkill["huaweicloud-ecs-ops"].(map[string]any)
	if bucket == nil || numOf(bucket["total"]) != 1 {
		t.Errorf("by_skill=%v, want huaweicloud-ecs-ops total 1", bySkill)
	}
	// Every parsed input stays listed as part of the aggregated window, even
	// when it was skipped — the skip is visible via skipped_smoke.
	if files, _ := summary["trace_files"].([]any); len(files) != 5 {
		t.Errorf("trace_files=%d entries, want 5 parsed inputs", len(files))
	}
}

func writeTraceJSON(t *testing.T, root, name, content string) {
	t.Helper()
	dir := filepath.Join(root, "audit-results")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// numOf normalizes JSON number values (int or float64) to int for assertions.
func numOf(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

// TestAggregateTraceCountsL2SkippedNoSchema pins the observability hook for the
// L2 capability gap: traces whose schema check could not run (no
// references/openapi-schema.json in the skill) are counted in the quality
// summary, so the gap is visible to the alarm consumer instead of reading as a
// clean check. Both the current (status field) and legacy (details prefix)
// encodings count; a passing L2 does not.
func TestAggregateTraceCountsL2SkippedNoSchema(t *testing.T) {
	root := t.TempDir()
	withL2 := func(trace string, l2 map[string]any) string {
		var m map[string]any
		if err := json.Unmarshal([]byte(trace), &m); err != nil {
			t.Fatal(err)
		}
		m["hallucination_detection"] = map[string]any{"blocked": false, "l2": l2}
		b, _ := json.Marshal(m)
		return string(b)
	}
	writeTraceJSON(t, root, "gcl-trace-20260101-000001.json",
		withL2(traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0),
			map[string]any{"blocked": false, "status": "skipped_no_schema",
				"details": "skipped_no_schema: no openapi-schema.json found; skipping L2"}))
	// Legacy shape: status encoded only in details.
	writeTraceJSON(t, root, "gcl-trace-20260101-000002.json",
		withL2(traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0),
			map[string]any{"blocked": false,
				"details": "skipped_no_schema: no openapi-schema.json found; skipping L2"}))
	writeTraceJSON(t, root, "gcl-trace-20260101-000003.json",
		withL2(traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0),
			map[string]any{"blocked": false, "status": "pass", "details": "pass: schema valid"}))

	out := filepath.Join(root, "summary.json")
	if err := runAggregate([]string{"trace", "--root", root, "--output", out}); err != nil {
		t.Fatalf("aggregate failed: %v", err)
	}
	summary := readSummary(t, out)
	if got := numOf(summary["l2_skipped_no_schema"]); got != 2 {
		t.Errorf("l2_skipped_no_schema=%v, want 2 (status field + legacy details fallback)", got)
	}
	if got := numOf(summary["totals"].(map[string]any)["total_runs"]); got != 3 {
		t.Errorf("total_runs=%v, want 3 (L2 status must not change pass-rate math)", got)
	}
}

// TestAggregateTraceBucketsUnknownStatus pins the Critic finding that a trace
// whose final.status the aggregator does not know still counts in total_runs:
// without a bucket of its own it silently depressed pass_rate with nothing in
// the summary to explain it.
func TestAggregateTraceBucketsUnknownStatus(t *testing.T) {
	root := t.TempDir()
	var m map[string]any
	if err := json.Unmarshal([]byte(traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0)), &m); err != nil {
		t.Fatal(err)
	}
	m["final"].(map[string]any)["status"] = "QUARANTINED" // future schema value
	b, _ := json.Marshal(m)
	writeTraceJSON(t, root, "gcl-trace-20260101-000001.json", string(b))
	writeTraceJSON(t, root, "gcl-trace-20260101-000002.json", traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0))

	out := filepath.Join(root, "summary.json")
	if err := runAggregate([]string{"trace", "--root", root, "--output", out}); err != nil {
		t.Fatalf("aggregate failed: %v", err)
	}
	summary := readSummary(t, out)
	totals, _ := summary["totals"].(map[string]any)
	if got := numOf(totals["UNKNOWN"]); got != 1 {
		t.Errorf("totals.UNKNOWN=%v, want 1 (unclassified status must stay visible)", got)
	}
	if got := numOf(totals["total_runs"]); got != 2 {
		t.Errorf("total_runs=%v, want 2", got)
	}
	if got := summary["pass_rate"].(float64); got != 0.5 {
		t.Errorf("pass_rate=%v, want 0.5 (1 PASS of 2 runs)", got)
	}
	bySkill, _ := summary["by_skill"].(map[string]any)
	bucket, _ := bySkill["huaweicloud-ecs-ops"].(map[string]any)
	if got := numOf(bucket["UNKNOWN"]); got != 1 {
		t.Errorf("by_skill.UNKNOWN=%v, want 1", got)
	}
}
