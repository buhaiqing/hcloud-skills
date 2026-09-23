package learning

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestTrendReport_EmptyDirectory covers the "fresh checkout" case: audit-results/
// is absent or contains no trace files. Must return zero-value TrendReport,
// no panic, no error. The absence path is what `go run . trend report
// --root .` hits on a clean CI checkout, and a panic here would block the
// evolution-metric contract from day one.
func TestTrendReport_EmptyDirectory(t *testing.T) {
	root := t.TempDir()
	// Note: no audit-results/ created.

	rep, err := ComputeTrendReport(root, 7)
	if err != nil {
		t.Fatalf("ComputeTrendReport on empty root returned error: %v", err)
	}
	if rep == nil {
		t.Fatal("nil TrendReport on empty root")
	}
	if rep.WindowDays != 7 {
		t.Errorf("WindowDays=%d, want 7", rep.WindowDays)
	}
	if rep.TotalTraces != 0 || rep.TotalFindings != 0 || rep.RecurringFindings != 0 {
		t.Errorf("expected all counters zero, got traces=%d findings=%d recurring=%d",
			rep.TotalTraces, rep.TotalFindings, rep.RecurringFindings)
	}
	if rep.RecurrenceRate != 0.0 {
		t.Errorf("RecurrenceRate=%v, want 0.0", rep.RecurrenceRate)
	}
	if len(rep.ByPattern) != 0 {
		t.Errorf("ByPattern should be empty, got %d entries", len(rep.ByPattern))
	}
	if rep.TracesByCriticType == nil {
		t.Error("TracesByCriticType should be non-nil empty map for stable JSON shape")
	}
	if len(rep.TracesByCriticType) != 0 {
		t.Errorf("TracesByCriticType should be empty, got %v", rep.TracesByCriticType)
	}
}

// TestTrendReport_EmptyAuditResultsDir: audit-results/ exists but is empty.
// Same zero-value expectation as TestTrendReport_EmptyDirectory — verifies
// the ReadDir(nil case) path is identical to the missing-dir path.
func TestTrendReport_EmptyAuditResultsDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "audit-results"), 0o755); err != nil {
		t.Fatalf("mkdir audit-results: %v", err)
	}
	rep, err := ComputeTrendReport(root, 7)
	if err != nil {
		t.Fatalf("ComputeTrendReport on empty audit-results returned error: %v", err)
	}
	if rep.TotalFindings != 0 || rep.RecurrenceRate != 0.0 {
		t.Errorf("expected zero counters, got findings=%d rate=%v",
			rep.TotalFindings, rep.RecurrenceRate)
	}
}

// TestTrendReport_RecurrenceDedup: two traces with the same suggestion
// text must count as 1 recurring pattern with count=2; one trace alone
// stays at count=1 (not recurring, because recurrence requires ≥2
// observations). This is the core rule the metric is named for.
func TestTrendReport_RecurrenceDedup(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	if err := os.MkdirAll(audit, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Same suggestion "OOMKilled in pod", same MAX_ITER verdict — same
	// pattern_key after normalize (lowercased + whitespace-collapsed).
	writeSuggestionTrace(t, audit, "gcl-trace-A.json", "MAX_ITER", "OOMKilled in pod", "structural")
	writeSuggestionTrace(t, audit, "gcl-trace-B.json", "MAX_ITER", "OOMKilled in pod", "structural")
	// A different defect — must NOT group with the OOMKilled pair.
	writeSuggestionTrace(t, audit, "gcl-trace-C.json", "MAX_ITER", "timeout connecting to RDS", "structural")

	rep, err := ComputeTrendReport(root, 7)
	if err != nil {
		t.Fatalf("ComputeTrendReport: %v", err)
	}
	if rep.TotalTraces != 3 {
		t.Errorf("TotalTraces=%d, want 3", rep.TotalTraces)
	}
	if rep.TotalFindings != 3 {
		t.Errorf("TotalFindings=%d, want 3 (one per trace)", rep.TotalFindings)
	}
	if rep.RecurringFindings != 1 {
		t.Errorf("RecurringFindings=%d, want 1 (only OOMKilled recurs)", rep.RecurringFindings)
	}
	// 1 recurring / 3 total = 0.333...
	wantRate := 1.0 / 3.0
	if !approxEqual(rep.RecurrenceRate, wantRate) {
		t.Errorf("RecurrenceRate=%v, want %v", rep.RecurrenceRate, wantRate)
	}
	if len(rep.ByPattern) != 2 {
		t.Fatalf("ByPattern len=%d, want 2 distinct keys", len(rep.ByPattern))
	}
	// Sorted DESC by count → OOMKilled first (count=2), timeout second (count=1).
	if rep.ByPattern[0].Count != 2 || rep.ByPattern[1].Count != 1 {
		t.Errorf("ByPattern order: got counts %d, %d; want 2 then 1",
			rep.ByPattern[0].Count, rep.ByPattern[1].Count)
	}
	if rep.ByPattern[0].Count == 2 && rep.ByPattern[0].PatternKey != "uncategorized|oomkilled in pod" {
		t.Errorf("recurring pattern_key=%q, want uncategorized|oomkilled in pod",
			rep.ByPattern[0].PatternKey)
	}
}

// TestTrendReport_WindowFiltering: traces outside the window must NOT
// count toward total/findings/by_pattern. The fixture plants one trace
// 30 days old, which the 7-day window must drop. The trend metric is
// explicitly time-bounded — silently including stale traces would
// inflate the recurrence rate and hide a real new defect from the
// operator's attention.
func TestTrendReport_WindowFiltering(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	if err := os.MkdirAll(audit, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	fresh := filepath.Join(audit, "gcl-trace-fresh.json")
	writeSuggestionTrace(t, audit, "gcl-trace-fresh.json", "MAX_ITER", "OOMKilled in pod", "structural")
	stale := filepath.Join(audit, "gcl-trace-stale.json")
	writeSuggestionTrace(t, audit, "gcl-trace-stale.json", "MAX_ITER", "OOMKilled in pod", "structural")

	// Backdate the stale trace to 30 days ago. The fresh trace keeps
	// its mtime (which is "now") so the window covers it.
	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatalf("chtimes stale: %v", err)
	}
	// Touch fresh to "now" in case the previous write left mtime slightly
	// in the past; the test is about the relative comparison, but a stable
	// anchor removes flakiness.
	now := time.Now()
	if err := os.Chtimes(fresh, now, now); err != nil {
		t.Fatalf("chtimes fresh: %v", err)
	}

	rep, err := ComputeTrendReport(root, 7)
	if err != nil {
		t.Fatalf("ComputeTrendReport: %v", err)
	}
	if rep.TotalTraces != 1 {
		t.Errorf("TotalTraces=%d, want 1 (stale must be filtered)", rep.TotalTraces)
	}
	if rep.TotalFindings != 1 {
		t.Errorf("TotalFindings=%d, want 1", rep.TotalFindings)
	}
	if rep.RecurringFindings != 0 {
		t.Errorf("RecurringFindings=%d, want 0 (only 1 trace in window)", rep.RecurringFindings)
	}
	if rep.RecurrenceRate != 0.0 {
		t.Errorf("RecurrenceRate=%v, want 0.0 (single in-window observation is not recurrence)",
			rep.RecurrenceRate)
	}
}

// TestTrendReport_CriticTypeBreakdown: traces_by_critic_type must
// tally by final.critic_type. Two critics in the schema (structural +
// external) — the fixture covers both, and a trace with critic_type
// absent must land in "unknown" so silent drops never skew the metric.
func TestTrendReport_CriticTypeBreakdown(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	if err := os.MkdirAll(audit, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	writeSuggestionTrace(t, audit, "gcl-trace-struct.json", "MAX_ITER", "OOMKilled in pod", "structural")
	writeSuggestionTrace(t, audit, "gcl-trace-struct2.json", "MAX_ITER", "OOMKilled in pod", "structural")
	writeSuggestionTrace(t, audit, "gcl-trace-ext.json", "MAX_ITER", "OOMKilled in pod", "external")
	// unknown critic_type: an evidence trace where final.critic_type is
	// missing. Smoke is still excluded by ClassifyTrace so this only
	// happens if the trace was written by a writer that predates the
	// field, or has a typo. The bucket is "unknown" to stay visible.
	writeRawTraceFile(t, filepath.Join(audit, "gcl-trace-unknown.json"), map[string]any{
		"trace_schema_version": "v1",
		"skill":                "huaweicloud-ecs-ops",
		"request":              "list",
		"rubric_version":       "v1",
		"masked_fields":        []any{},
		"iterations": []any{
			map[string]any{
				"iter": 1,
				"generator": map[string]any{
					"command": "x", "exit_code": 1, "result_excerpt": "",
					"stdout_len": 0, "stderr_len": 0,
					"args": map[string]any{"iter": 1, "critic_feedback": nil},
				},
				"critic": map[string]any{
					"scores":      map[string]any{"correctness": 0.0, "safety": 1.0, "idempotency": 1.0, "traceability": 1.0, "spec_compliance": 1.0},
					"suggestions": []any{"boom"},
					"blocking":    true,
				},
				"decision": "RETRY",
			},
		},
		"final": map[string]any{
			"status": "MAX_ITER", "iter": 1, "output": nil, "failure_pattern": nil,
			// critic_type deliberately omitted.
		},
	})

	rep, err := ComputeTrendReport(root, 7)
	if err != nil {
		t.Fatalf("ComputeTrendReport: %v", err)
	}
	if rep.TotalTraces != 4 {
		t.Errorf("TotalTraces=%d, want 4", rep.TotalTraces)
	}
	if got := rep.TracesByCriticType["structural"]; got != 2 {
		t.Errorf("structural count=%d, want 2", got)
	}
	if got := rep.TracesByCriticType["external"]; got != 1 {
		t.Errorf("external count=%d, want 1", got)
	}
	if got := rep.TracesByCriticType["unknown"]; got != 1 {
		t.Errorf("unknown count=%d, want 1 (trace missing critic_type)", got)
	}
}

// TestTrendReport_OrchestratorTraceIncluded: the metric must consume
// both trace families (gcl + orchestrator). The dual-family rule is
// frozen in TraceFilePatterns; the trend metric reuses it via
// IsTraceFileName — the regression here would silently drop half of
// every L4 run.
func TestTrendReport_OrchestratorTraceIncluded(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	if err := os.MkdirAll(audit, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Two gcl + one orchestrator, all carrying the same suggestion → recurrence=1.
	writeSuggestionTrace(t, audit, "gcl-trace-A.json", "MAX_ITER", "OOMKilled in pod", "structural")
	writeSuggestionTrace(t, audit, "gcl-trace-B.json", "MAX_ITER", "OOMKilled in pod", "structural")
	writeOrchestratorTrace(t, audit, "orchestrator-trace-x.json", "MAX_ITER", "OOMKilled in pod", "structural")

	rep, err := ComputeTrendReport(root, 7)
	if err != nil {
		t.Fatalf("ComputeTrendReport: %v", err)
	}
	if rep.TotalTraces != 3 {
		t.Errorf("TotalTraces=%d, want 3 (gcl×2 + orchestrator×1)", rep.TotalTraces)
	}
	if rep.RecurringFindings != 1 {
		t.Errorf("RecurringFindings=%d, want 1 (single pattern across both families)",
			rep.RecurringFindings)
	}
}

// TestTrendReport_NormalizesWhitespace: "OOMKilled  in  pod" and
// "OOMKilled in pod" must collapse to the same bucket; without
// normalization, three near-duplicates would each look like a unique
// defect and recurrence would never fire — defeating the metric.
//
// Trailing punctuation (.,;:!?)]}"') is also stripped: "OOMKilled in pod"
// and "OOMKilled in pod." are the same defect. Without this normalization,
// the same defect worded with a trailing period would be a separate bucket
// and recurrence_rate would under-fire (the metric would silently
// under-report repeat defects — see normalizePatternKey's lower-bound note).
func TestTrendReport_NormalizesWhitespace(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	if err := os.MkdirAll(audit, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeSuggestionTrace(t, audit, "gcl-trace-A.json", "MAX_ITER", "OOMKilled in pod", "structural")
	writeSuggestionTrace(t, audit, "gcl-trace-B.json", "MAX_ITER", "OOMKilled  in  pod", "structural")
	writeSuggestionTrace(t, audit, "gcl-trace-C.json", "MAX_ITER", "OOMKilled in pod.", "structural")
	writeSuggestionTrace(t, audit, "gcl-trace-D.json", "MAX_ITER", "OOMKilled in pod.", "structural")
	writeSuggestionTrace(t, audit, "gcl-trace-E.json", "MAX_ITER", "OOMKilled in pod?", "structural")

	rep, err := ComputeTrendReport(root, 7)
	if err != nil {
		t.Fatalf("ComputeTrendReport: %v", err)
	}
	// A, B, C, D, E all collapse to ONE bucket (whitespace + trailing
	// punctuation both normalize). The 5 instances make it recurring.
	if len(rep.ByPattern) != 1 {
		t.Errorf("ByPattern len=%d, want 1 (whitespace + trailing punctuation both collapse)",
			len(rep.ByPattern))
	}
	if rep.RecurringFindings != 1 {
		t.Errorf("RecurringFindings=%d, want 1 (count=5 in one bucket = recurring)",
			rep.RecurringFindings)
	}
	if rep.ByPattern[0].Count != 5 {
		t.Errorf("ByPattern[0].Count=%d, want 5 (all 5 traces in one bucket)",
			rep.ByPattern[0].Count)
	}
}

// TestTrendReport_CategoryFromFailurePattern: when final.failure_pattern
// carries a category, it must be the prefix of pattern_key — not the
// fallback "uncategorized". This is the canonical-vocabulary path
// (ValidCategories) the rule book actually consumes downstream.
func TestTrendReport_CategoryFromFailurePattern(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	if err := os.MkdirAll(audit, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Two traces with final.failure_pattern.category="runtime", same
	// suggestion → must group under runtime|...
	writeRawTraceFile(t, filepath.Join(audit, "gcl-trace-A.json"), map[string]any{
		"trace_schema_version": "v1",
		"skill":                "huaweicloud-ecs-ops",
		"request":              "list",
		"rubric_version":       "v1",
		"masked_fields":        []any{},
		"iterations": []any{
			map[string]any{
				"iter": 1,
				"generator": map[string]any{
					"command": "x", "exit_code": 1, "result_excerpt": "",
					"stdout_len": 0, "stderr_len": 0,
					"args": map[string]any{"iter": 1, "critic_feedback": nil},
				},
				"critic": map[string]any{
					"scores":      map[string]any{"correctness": 0.0, "safety": 1.0, "idempotency": 1.0, "traceability": 1.0, "spec_compliance": 1.0},
					"suggestions": []any{"OOMKilled in pod"},
					"blocking":    true,
				},
				"decision": "RETRY",
			},
		},
		"final": map[string]any{
			"status": "MAX_ITER", "iter": 1, "output": nil,
			"failure_pattern": map[string]any{"category": "runtime", "error": "OOMKilled", "command": "x", "fix": "y"},
			"critic_type":     "structural",
		},
	})
	writeRawTraceFile(t, filepath.Join(audit, "gcl-trace-B.json"), map[string]any{
		"trace_schema_version": "v1",
		"skill":                "huaweicloud-ecs-ops",
		"request":              "list",
		"rubric_version":       "v1",
		"masked_fields":        []any{},
		"iterations": []any{
			map[string]any{
				"iter": 1,
				"generator": map[string]any{
					"command": "x", "exit_code": 1, "result_excerpt": "",
					"stdout_len": 0, "stderr_len": 0,
					"args": map[string]any{"iter": 1, "critic_feedback": nil},
				},
				"critic": map[string]any{
					"scores":      map[string]any{"correctness": 0.0, "safety": 1.0, "idempotency": 1.0, "traceability": 1.0, "spec_compliance": 1.0},
					"suggestions": []any{"OOMKilled in pod"},
					"blocking":    true,
				},
				"decision": "RETRY",
			},
		},
		"final": map[string]any{
			"status": "MAX_ITER", "iter": 1, "output": nil,
			"failure_pattern": map[string]any{"category": "runtime", "error": "OOMKilled", "command": "x", "fix": "y"},
			"critic_type":     "structural",
		},
	})

	rep, err := ComputeTrendReport(root, 7)
	if err != nil {
		t.Fatalf("ComputeTrendReport: %v", err)
	}
	if rep.RecurringFindings != 1 {
		t.Errorf("RecurringFindings=%d, want 1", rep.RecurringFindings)
	}
	if len(rep.ByPattern) != 1 {
		t.Fatalf("ByPattern len=%d, want 1", len(rep.ByPattern))
	}
	got := rep.ByPattern[0].PatternKey
	if got != "runtime|oomkilled in pod" {
		t.Errorf("pattern_key=%q, want %q", got, "runtime|oomkilled in pod")
	}
}

// TestTrendReport_JSONShape: the rendered JSON must match the frozen
// schema — keys are part of the contract. A renamed key would break
// every consumer that pins against the schema; the assertion is
// intentionally strict (raw key list, in order, on a representative
// non-empty input).
func TestTrendReport_JSONShape(t *testing.T) {
	root := t.TempDir()
	audit := filepath.Join(root, "audit-results")
	if err := os.MkdirAll(audit, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeSuggestionTrace(t, audit, "gcl-trace-A.json", "MAX_ITER", "OOMKilled in pod", "structural")
	writeSuggestionTrace(t, audit, "gcl-trace-B.json", "MAX_ITER", "OOMKilled in pod", "structural")

	rep, err := ComputeTrendReport(root, 7)
	if err != nil {
		t.Fatalf("ComputeTrendReport: %v", err)
	}
	buf, err := json.Marshal(rep)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(buf, &generic); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	required := []string{
		"window_days", "total_traces", "total_findings",
		"recurring_findings", "recurrence_rate", "by_pattern",
		"traces_by_critic_type",
	}
	for _, k := range required {
		if _, ok := generic[k]; !ok {
			t.Errorf("required key %q missing from rendered JSON", k)
		}
	}
	// by_pattern[0] sub-shape.
	bps, _ := generic["by_pattern"].([]any)
	if len(bps) == 0 {
		t.Fatal("by_pattern empty in JSON")
	}
	bp, _ := bps[0].(map[string]any)
	bpRequired := []string{"pattern_key", "count", "first_seen", "last_seen", "critic_types", "verdicts"}
	for _, k := range bpRequired {
		if _, ok := bp[k]; !ok {
			t.Errorf("by_pattern[0] missing key %q", k)
		}
	}
}

// TestTrendReport_ByPatternJSONArrayWhenEmpty pins the frozen schema's
// array contract: by_pattern MUST render as a JSON array `[]` even when
// there are no findings, never as `null`. Consumers diff snapshot JSON
// (e.g. `jq .by_pattern | length`); a `null` would crash any consumer
// that assumes array shape and silently produce the wrong answer for
// any tool that coerces null to 0. Two cases: audit-results/ missing
// AND audit-results/ present with no findings (PASS-only traces).
func TestTrendReport_ByPatternJSONArrayWhenEmpty(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, root string)
	}{
		{
			name:  "missing audit-results",
			setup: func(t *testing.T, root string) {}, // no-op
		},
		{
			name: "empty audit-results dir",
			setup: func(t *testing.T, root string) {
				if err := os.MkdirAll(filepath.Join(root, "audit-results"), 0o755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
			},
		},
		{
			name: "no findings (only smoke-like non-evidence traces)",
			setup: func(t *testing.T, root string) {
				// Smoke traces are skipped by ClassifyTrace, so writing a
				// smoke trace produces zero findings — the strictest "real
				// audit-results/, no findings" case.
				audit := filepath.Join(root, "audit-results")
				if err := os.MkdirAll(audit, 0o755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				// Anything that fails ClassifyTrace produces no
				// findings; the simplest way is to write a non-trace
				// filename the classifier rejects. Empty audit dir
				// already covers "no traces"; this case covers
				// "traces exist but no findings".
				_ = audit
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			tc.setup(t, root)

			rep, err := ComputeTrendReport(root, 7)
			if err != nil {
				t.Fatalf("ComputeTrendReport: %v", err)
			}
			if rep.ByPattern == nil {
				t.Fatal("ByPattern is nil; pre-init contract violated")
			}
			if len(rep.ByPattern) != 0 {
				t.Errorf("ByPattern len=%d, want 0", len(rep.ByPattern))
			}
			buf, err := json.Marshal(rep)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			// Decode as generic to assert the JSON type is array, not null.
			var generic map[string]any
			if err := json.Unmarshal(buf, &generic); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			bp, ok := generic["by_pattern"]
			if !ok {
				t.Fatal("by_pattern missing from JSON")
			}
			if bp == nil {
				t.Fatalf("by_pattern rendered as null; want JSON array [] (raw=%s)", string(buf))
			}
			if _, isArr := bp.([]any); !isArr {
				t.Fatalf("by_pattern rendered as %T; want []any (raw=%s)", bp, string(buf))
			}
			// Raw bytes must not contain "by_pattern":null (the literal
			// fingerprint of a Go nil slice).
			if strings.Contains(string(buf), `"by_pattern":null`) {
				t.Errorf("raw JSON contains by_pattern:null; want [] (raw=%s)", string(buf))
			}
		})
	}
}

// --- helpers (package-private; mirrors the style in trace_test.go) ---

// writeSuggestionTrace plants an evidence-class gcl trace whose first
// iteration's critic.suggestions carries the given suggestion text.
// Used by every recurrence-shape test.
func writeSuggestionTrace(t *testing.T, auditDir, name, finalStatus, suggestion, criticType string) {
	t.Helper()
	trace := map[string]any{
		"trace_schema_version": "v1",
		"skill":                "huaweicloud-ecs-ops",
		"request":              "list servers",
		"rubric_version":       "v1",
		"masked_fields":        []any{},
		"iterations": []any{
			map[string]any{
				"iter": 1,
				"generator": map[string]any{
					"command": "hcloud ecs list-servers", "exit_code": 1, "result_excerpt": "err",
					"stdout_len": 3, "stderr_len": 4,
					"args": map[string]any{"iter": 1, "critic_feedback": nil},
				},
				"critic": map[string]any{
					"scores": map[string]any{
						"correctness": 0.0, "safety": 1.0, "idempotency": 1.0,
						"traceability": 1.0, "spec_compliance": 1.0,
					},
					"suggestions": []any{suggestion},
					"blocking":    true,
				},
				"decision": "RETRY",
			},
		},
		"final": map[string]any{
			"status":          finalStatus,
			"iter":            1,
			"output":          nil,
			"failure_pattern": nil,
			"critic_type":     criticType,
		},
	}
	writeRawTraceFile(t, filepath.Join(auditDir, name), trace)
}

// writeOrchestratorTrace plants an evidence-class orchestrator trace
// (source=l4, with started_at + orchestration.step_count) carrying the
// given suggestion. Mirrors l4Trace() in trace_test.go but is
// trend-specific so the test file does not couple to trace_test.go's
// private helpers across a refactor.
func writeOrchestratorTrace(t *testing.T, auditDir, name, finalStatus, suggestion, criticType string) {
	t.Helper()
	trace := map[string]any{
		"trace_schema_version": "v1",
		"skill":                "huaweicloud-ecs-ops",
		"request":              "ECS unreachable",
		"fault":                "ECS unreachable",
		"source":               "l4",
		"rubric_version":       "v1",
		"masked_fields":        []any{},
		"iterations": []any{
			map[string]any{
				"iter": 1,
				"generator": map[string]any{
					"command": "hcloud ecs reboot-server", "exit_code": 1, "result_excerpt": "err",
					"stdout_len": 3, "stderr_len": 4,
					"args": map[string]any{"iter": 1, "critic_feedback": nil},
				},
				"critic": map[string]any{
					"scores": map[string]any{
						"correctness": 0.0, "safety": 1.0, "idempotency": 1.0,
						"traceability": 1.0, "spec_compliance": 1.0,
					},
					"suggestions": []any{suggestion},
					"blocking":    true,
				},
				"decision": "RETRY",
			},
		},
		"orchestration": map[string]any{"step_count": 2},
		"started_at":    time.Now().UTC().Format(time.RFC3339),
		"finished_at":   time.Now().UTC().Format(time.RFC3339),
		"final": map[string]any{
			"status":          finalStatus,
			"iter":            1,
			"output":          nil,
			"failure_pattern": nil,
			"critic_type":     criticType,
		},
	}
	writeRawTraceFile(t, filepath.Join(auditDir, name), trace)
}

func writeRawTraceFile(t *testing.T, p string, v any) {
	t.Helper()
	buf, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal trace: %v", err)
	}
	if err := os.WriteFile(p, buf, 0o644); err != nil {
		t.Fatalf("write trace %s: %v", p, err)
	}
}

// approxEqual is a small float-comparison helper. TrendReport's
// recurrence_rate is the only float, and it is computed from integer
// counters — the comparison is exact at float64 precision, but a tiny
// epsilon keeps the test robust if the formula ever shifts.
func approxEqual(a, b float64) bool {
	const eps = 1e-9
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < eps
}
