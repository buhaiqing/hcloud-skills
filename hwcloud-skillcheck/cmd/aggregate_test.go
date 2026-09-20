package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

// traceFixture builds a canonical-schema-conforming gcl trace — the shape
// internal/gcl.PersistTrace writes — with `iters` iterations, each scored
// `score`, and the given terminal status. Every fixture in this file must
// conform: cmd/aggregate.go classifies non-conforming files as invalid_trace
// and excludes them from every metric, so a sloppy fixture would test the
// rejection path instead of the metric path.
func traceFixture(skill, status string, iters int, score float64) string {
	return traceFixtureWithRequest(skill, "list servers for the aggregation fixture", status, iters, score)
}

func traceFixtureWithRequest(skill, request, status string, iters int, score float64) string {
	type iter struct {
		Iter      int `json:"iter"`
		Generator struct {
			Command       string         `json:"command"`
			ExitCode      int            `json:"exit_code"`
			ResultExcerpt string         `json:"result_excerpt"`
			StdoutLen     int            `json:"stdout_len"`
			StderrLen     int            `json:"stderr_len"`
			Args          map[string]any `json:"args"`
		} `json:"generator"`
		Critic struct {
			Scores      map[string]float64 `json:"scores"`
			Suggestions []any              `json:"suggestions"`
			Blocking    bool               `json:"blocking"`
		} `json:"critic"`
		Decision string `json:"decision"`
	}
	iterations := make([]iter, 0, iters)
	for i := 1; i <= iters; i++ {
		var it iter
		it.Iter = i
		it.Generator.Command = "hcloud ecs list-servers"
		it.Generator.ResultExcerpt = "servers: 3 items"
		it.Generator.StdoutLen = 256
		it.Generator.Args = map[string]any{"iter": i, "critic_feedback": nil}
		it.Critic.Scores = map[string]float64{
			"correctness": score, "safety": score, "idempotency": score,
			"traceability": score, "spec_compliance": score,
		}
		it.Critic.Suggestions = []any{}
		it.Decision = decisionFor(status)
		iterations = append(iterations, it)
	}
	b, _ := json.Marshal(map[string]any{
		"trace_schema_version": "v1",
		"skill":                skill,
		"request":              request,
		"rubric_version":       "1.0",
		"masked_fields":        []any{},
		"iterations":           iterations,
		"final": map[string]any{
			"status": status, "iter": max(iters, 1), "output": nil, "failure_pattern": nil,
		},
	})
	return string(b)
}

// decisionFor maps a terminal status onto the per-iteration decision enum
// (PASS|RETRY|SAFETY_FAIL — MAX_ITER is a terminal status, not a decision).
func decisionFor(status string) string {
	switch status {
	case "PASS":
		return "PASS"
	case "SAFETY_FAIL":
		return "SAFETY_FAIL"
	default:
		return "RETRY"
	}
}

// l4TraceFixture builds the payload internal/l4.HandleFault writes to
// audit-results/orchestrator-trace-<faultID>.json: the canonical required
// fields (trace_schema_version/skill/request/rubric_version/masked_fields/
// iterations/final), source:"l4", a `fault` field, orchestration.step_count,
// one dry-run iteration carrying the real structural-critic scores, and a
// `final` block with critic_type:"structural". It mirrors the L4 writer's
// post-fix shape (see internal/l4/orchestrator.go) so this fixture exercises
// the same acceptance path as a real orchestrator trace.
func l4TraceFixture(skill, fault, status string, stepCount int, scores map[string]float64) string {
	iterations := []any{}
	if stepCount > 0 {
		iterations = append(iterations, map[string]any{
			"iter": 1,
			"generator": map[string]any{
				"command": "hcloud ecs list-servers", "exit_code": 0, "result_excerpt": "dry-run",
				"stdout_len": 0, "stderr_len": 0,
				"args": map[string]any{"iter": 1, "critic_feedback": nil},
			},
			"critic":   map[string]any{"scores": scores, "suggestions": []any{}, "blocking": false},
			"decision": decisionFor(status),
		})
	}
	b, _ := json.Marshal(map[string]any{
		"trace_schema_version": "v1",
		"trace_id":             "0123456789abcdef",
		"skill":                skill,
		"request":              fault,
		"fault":                fault,
		"source":               "l4",
		"rubric_version":       "v1",
		"masked_fields":        []any{},
		"iterations":           iterations,
		"orchestration":        map[string]any{"step_count": stepCount},
		"final": map[string]any{
			"status": status, "iter": 1, "output": "hcloud ecs list-servers",
			"dimensions": scores, "overall": 0.9, "critic_type": "structural",
			"failure_pattern": nil,
		},
	})
	return string(b)
}

// setFinal patches fields of a fixture's `final` block (used to inject the one
// field a test is about, e.g. critic_type).
func setFinal(t *testing.T, traceJSON string, fields map[string]any) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(traceJSON), &m); err != nil {
		t.Fatal(err)
	}
	final, _ := m["final"].(map[string]any)
	if final == nil {
		final = map[string]any{}
		m["final"] = final
	}
	for k, v := range fields {
		final[k] = v
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
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

// TestAggregateTraceSmokeFiltered asserts the smoke shapes are counted in
// skipped_smoke and excluded from pass_rate, by_source, by_skill, and
// evidence_runs, while a pre-canonical trace (no `final`, no
// trace_schema_version) is counted as invalid_trace instead — the frozen
// classification order is schema-invalid before smoke.
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
	// The pre-canonical orchestrator shape: no `final`, no
	// trace_schema_version. It fails canonical-schema validation, so it is
	// untrusted input (invalid_trace) rather than "smoke" — it never reaches
	// the smoke test.
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
	if got := numOf(summary["skipped_smoke"]); got != 3 {
		t.Errorf("skipped_smoke=%v, want 3 (smoke request, smoke fault, zero steps)", got)
	}
	if got := numOf(summary["invalid_trace"]); got != 1 {
		t.Errorf("invalid_trace=%v, want 1 (the pre-canonical trace)", got)
	}
	if got := numOf(summary["evidence_runs"]); got != 1 {
		t.Errorf("evidence_runs=%v, want 1", got)
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

// TestAggregateTraceRejectsOutOfEnumStatus pins the change of contract for a
// status outside the frozen enum: it used to count in total_runs with an
// UNKNOWN bucket (which let a trace the aggregator cannot interpret still move
// pass_rate). Now final.status is part of the canonical schema, so such a file
// is invalid_trace: untrusted input, excluded from every metric.
func TestAggregateTraceRejectsOutOfEnumStatus(t *testing.T) {
	root := t.TempDir()
	bogus := setFinal(t, traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0),
		map[string]any{"status": "QUARANTINED"})
	writeTraceJSON(t, root, "gcl-trace-20260101-000001.json", bogus)
	writeTraceJSON(t, root, "gcl-trace-20260101-000002.json", traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0))

	out := filepath.Join(root, "summary.json")
	if err := runAggregate([]string{"trace", "--root", root, "--output", out}); err != nil {
		t.Fatalf("aggregate failed: %v", err)
	}
	summary := readSummary(t, out)
	totals, _ := summary["totals"].(map[string]any)
	if got := numOf(totals["total_runs"]); got != 1 {
		t.Errorf("total_runs=%v, want 1 (the out-of-enum trace must not count)", got)
	}
	if got := numOf(summary["invalid_trace"]); got != 1 {
		t.Errorf("invalid_trace=%v, want 1", got)
	}
	if got := numOf(summary["evidence_runs"]); got != 1 {
		t.Errorf("evidence_runs=%v, want 1", got)
	}
	if got := summary["pass_rate"].(float64); got != 1.0 {
		t.Errorf("pass_rate=%v, want 1.0 (1 PASS of 1 evidence run)", got)
	}
	bySkill, _ := summary["by_skill"].(map[string]any)
	bucket, _ := bySkill["huaweicloud-ecs-ops"].(map[string]any)
	if got := numOf(bucket["total"]); got != 1 {
		t.Errorf("by_skill total=%v, want 1", got)
	}
}

// TestAggregateTraceRejectsCraftedPassTrace pins the safety Critic's first
// blocker: a crafted audit-results/ file that merely claims a PASS
// (`{"skill":"x","final":{"status":"PASS"}}`) used to be counted as a real
// contributing run, so a single hand-written file produced pass_rate=1.0 with
// nothing behind it. It must now be invalid_trace, must not contribute to
// evidence_runs/pass_rate/by_skill, and must still be listed in trace_files.
func TestAggregateTraceRejectsCraftedPassTrace(t *testing.T) {
	root := t.TempDir()
	writeTraceJSON(t, root, "orchestrator-trace-crafted.json",
		`{"skill":"huaweicloud-ecs-ops","final":{"status":"PASS"}}`)

	out := filepath.Join(root, "summary.json")
	if err := runAggregate([]string{"trace", "--root", root, "--output", out}); err != nil {
		t.Fatalf("aggregate failed: %v", err)
	}
	summary := readSummary(t, out)

	if got := numOf(summary["invalid_trace"]); got != 1 {
		t.Errorf("invalid_trace=%v, want 1 (crafted trace is not a run)", got)
	}
	if got := numOf(summary["evidence_runs"]); got != 0 {
		t.Errorf("evidence_runs=%v, want 0", got)
	}
	if got := numOf(summary["totals"].(map[string]any)["total_runs"]); got != 0 {
		t.Errorf("total_runs=%v, want 0", got)
	}
	if got := summary["pass_rate"].(float64); got != 0 {
		t.Errorf("pass_rate=%v, want 0 (pre-fix this crafted file produced 1.0)", got)
	}
	if bySkill, _ := summary["by_skill"].(map[string]any); len(bySkill) != 0 {
		t.Errorf("by_skill=%v, want no buckets", bySkill)
	}
	if files, _ := summary["trace_files"].([]any); len(files) != 1 {
		t.Errorf("trace_files=%d entries, want 1 parsed input", len(files))
	}

	// --require-evidence must fail (non-zero exit) on this input: a trace set
	// with no verification signal is exactly what the flag exists to catch.
	// The message must name the failing gate, and the flag must be a real flag
	// (an unknown flag would also error, for the wrong reason).
	err := runAggregate([]string{"trace", "--root", root, "--require-evidence"})
	if err == nil {
		t.Fatal("--require-evidence must fail when nothing carried a verification signal")
	}
	if !strings.Contains(err.Error(), "--require-evidence") {
		t.Errorf("error must name the gate that failed, got: %v", err)
	}
	// --require-traces keeps its meaning: files were parsed, so it passes.
	if err := runAggregate([]string{"trace", "--root", root, "--require-traces"}); err != nil {
		t.Fatalf("--require-traces must still pass when trace files were parsed, got: %v", err)
	}
	// Zero trace files is also zero evidence.
	empty := t.TempDir()
	if err := runAggregate([]string{"trace", "--root", empty, "--require-evidence"}); err == nil {
		t.Fatal("--require-evidence must fail when no trace files exist")
	}
}

// TestAggregateTraceRequireEvidenceAllSmoke asserts --require-evidence fails on
// a trace set whose files all classify as smoke, while --require-traces still
// tolerates it (its contract is "at least one parseable trace").
func TestAggregateTraceRequireEvidenceAllSmoke(t *testing.T) {
	root := t.TempDir()
	writeTraceJSON(t, root, "gcl-trace-smoke.json",
		traceFixtureWithRequest("huaweicloud-ecs-ops", "smoke", "PASS", 1, 1.0))

	if err := runAggregate([]string{"trace", "--root", root, "--require-evidence"}); err == nil {
		t.Fatal("--require-evidence must fail when every trace is smoke")
	}
	if err := runAggregate([]string{"trace", "--root", root, "--require-traces"}); err != nil {
		t.Fatalf("--require-traces must still pass on a parseable smoke trace, got: %v", err)
	}
	// Positive control: the flag is recognized and stays out of the way once a
	// trace carries a signal (otherwise "fails on smoke" would also be
	// satisfied by a flag typo).
	writeTraceJSON(t, root, "gcl-trace-real.json",
		traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0))
	if err := runAggregate([]string{"trace", "--root", root, "--require-evidence"}); err != nil {
		t.Fatalf("--require-evidence must pass when a trace carried a verification signal, got: %v", err)
	}
}

// TestAggregateTraceSafetyFailIsNeverSmoke pins the third Critic finding: a real
// trace whose request is the smoke marker but whose terminal status is
// SAFETY_FAIL was classified as smoke, which deleted a genuine safety failure
// from pass_rate (and from learning) at the exact moment it mattered. The
// verdict must win.
func TestAggregateTraceSafetyFailIsNeverSmoke(t *testing.T) {
	root := t.TempDir()
	writeTraceJSON(t, root, "gcl-trace-smokefail.json",
		traceFixtureWithRequest("huaweicloud-ecs-ops", "smoke", "SAFETY_FAIL", 1, 0.0))

	out := filepath.Join(root, "summary.json")
	if err := runAggregate([]string{"trace", "--root", root, "--output", out}); err != nil {
		t.Fatalf("aggregate failed: %v", err)
	}
	summary := readSummary(t, out)

	totals, _ := summary["totals"].(map[string]any)
	if got := numOf(totals["SAFETY_FAIL"]); got != 1 {
		t.Errorf("totals.SAFETY_FAIL=%v, want 1 (a safety failure must never be smoke)", got)
	}
	if got := numOf(summary["evidence_runs"]); got != 1 {
		t.Errorf("evidence_runs=%v, want 1", got)
	}
	if got := numOf(totals["total_runs"]); got != 1 {
		t.Errorf("total_runs=%v, want 1", got)
	}
	if got := numOf(summary["skipped_smoke"]); got != 0 {
		t.Errorf("skipped_smoke=%v, want 0", got)
	}
	if got := summary["pass_rate"].(float64); got != 0 {
		t.Errorf("pass_rate=%v, want 0", got)
	}
}

// TestAggregateTraceNormalizesCriticType pins the fourth Critic finding:
// final.critic_type was copied through unchecked, so a trace could claim a
// critic implementation that does not exist ("llm-verdict-v9") and land in a
// trusted bucket. The frozen vocabulary is {structural, external}; everything
// else, including an absent field, is counted as unknown.
func TestAggregateTraceNormalizesCriticType(t *testing.T) {
	root := t.TempDir()
	scores := map[string]float64{
		"correctness": 1.0, "safety": 1.0, "idempotency": 1.0,
		"traceability": 1.0, "spec_compliance": 1.0,
	}
	// structural: the L4 dry-run critic (fixture default).
	writeTraceJSON(t, root, "orchestrator-trace-structural.json",
		l4TraceFixture("huaweicloud-ecs-ops", "RDS timeout", "PASS", 2, scores))
	// external: a real Critic.
	writeTraceJSON(t, root, "gcl-trace-external.json",
		setFinal(t, traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0), map[string]any{"critic_type": "external"}))
	// A critic nothing implements.
	writeTraceJSON(t, root, "gcl-trace-fake.json",
		setFinal(t, traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0), map[string]any{"critic_type": "llm-verdict-v9"}))
	// No critic_type at all.
	writeTraceJSON(t, root, "gcl-trace-absent.json",
		traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0))

	out := filepath.Join(root, "summary.json")
	if err := runAggregate([]string{"trace", "--root", root, "--output", out}); err != nil {
		t.Fatalf("aggregate failed: %v", err)
	}
	summary := readSummary(t, out)

	byCriticType, ok := summary["by_critic_type"].(map[string]any)
	if !ok {
		t.Fatalf("by_critic_type missing from summary: %v", summary)
	}
	want := map[string]int{"structural": 1, "external": 1, "unknown": 2}
	for k, v := range want {
		if got := numOf(byCriticType[k]); got != v {
			t.Errorf("by_critic_type[%s]=%v, want %d", k, got, v)
		}
	}
	if len(byCriticType) != len(want) {
		t.Errorf("by_critic_type has %d keys, want exactly %d: %v", len(byCriticType), len(want), byCriticType)
	}
	if got := numOf(summary["evidence_runs"]); got != 4 {
		t.Errorf("evidence_runs=%v, want 4 (an unrecognized critic_type must not drop the run)", got)
	}
}

// TestAggregateTraceWarnsOnZeroEvidence pins the operator-visible half of the
// "--require-traces passed on a trace set with zero verification signal"
// finding: the flag keeps its contract (at least one parseable trace) but the
// run must say out loud that nothing carried a signal, naming BOTH exclusion
// counters — smoke and schema-invalid — so a trace set of dry runs and crafted
// files cannot read as a green gate.
func TestAggregateTraceWarnsOnZeroEvidence(t *testing.T) {
	root := t.TempDir()
	writeTraceJSON(t, root, "gcl-trace-smoke.json",
		traceFixtureWithRequest("huaweicloud-ecs-ops", "smoke", "PASS", 1, 1.0))
	writeTraceJSON(t, root, "orchestrator-trace-crafted.json",
		`{"skill":"huaweicloud-ecs-ops","final":{"status":"PASS"}}`)

	var err error
	stderr := captureStderr(t, func() {
		err = runAggregate([]string{"trace", "--root", root, "--require-traces"})
	})
	if err != nil {
		t.Fatalf("--require-traces must still pass when files were parsed, got: %v", err)
	}
	for _, want := range []string{"skipped_smoke=1", "invalid_trace=1"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("WARN must name %q when no trace carried a verification signal; stderr:\n%s", want, stderr)
		}
	}
	// The rejected trace is named individually too.
	if !strings.Contains(stderr, "orchestrator-trace-crafted.json") {
		t.Errorf("the rejected trace file must be named; stderr:\n%s", stderr)
	}
}
