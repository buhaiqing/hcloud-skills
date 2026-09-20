package l4

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/gcl"
)

// readTraceMap parses a persisted orchestrator trace into its generic map
// form — the exact shape cmd/aggregate.go and internal/learning read.
func readTraceMap(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read trace %s: %v", path, err)
	}
	var trace map[string]any
	if err := json.Unmarshal(raw, &trace); err != nil {
		t.Fatalf("parse trace %s: %v", path, err)
	}
	return trace
}

func mapOf(t *testing.T, v any, what string) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("%s: want object, got %T", what, v)
	}
	return m
}

func scoreOf(t *testing.T, v any, what string) float64 {
	t.Helper()
	f, ok := v.(float64)
	if !ok {
		t.Fatalf("%s: want number, got %T (%v)", what, v, v)
	}
	return f
}

// TestHandleFault_TraceConsumableShape pins the P0 trace contract: the
// orchestrator trace must carry a `final` block with the same field names as
// gcl.PersistTrace's FinalResult, a top-level source:"l4", an
// iterations[0].critic.scores vector, and critic scores that are the REAL
// structural-critic output — not the hardcoded literals
// (0.9/0.85/0.95/0.8) that had no relationship to the run.
func TestHandleFault_TraceConsumableShape(t *testing.T) {
	root := t.TempDir()
	const fault = "RDS connection timeout"
	out := HandleFault(HandleFaultInput{
		Root:     root,
		Fault:    fault,
		Resource: "rds:instance",
		Risk:     "medium",
	}, nil)

	trace := readTraceMap(t, out.Learning.TracePersisted)

	if got, _ := trace["source"].(string); got != "l4" {
		t.Errorf("trace source=%q, want \"l4\"", got)
	}
	if got, _ := trace["fault"].(string); got != fault {
		t.Errorf("trace fault=%q, want %q (smoke detection reads this field)", got, fault)
	}
	if got, _ := trace["skill"].(string); got == "" {
		t.Error("trace skill missing: cmd/aggregate.go requires it to bucket the trace")
	}

	final := mapOf(t, trace["final"], "final")
	if got, _ := final["status"].(string); got != "PASS" {
		t.Errorf("final.status=%q, want PASS for a clean dry-run fault", got)
	}
	if _, ok := final["status"].(string); !ok {
		t.Fatal("final.status must be a string (aggregate buckets on it)")
	}
	if iter, _ := final["iter"].(float64); iter < 1 {
		t.Errorf("final.iter=%v, want >= 1", final["iter"])
	}
	if _, ok := final["output"]; !ok {
		t.Error("final.output missing (PersistTrace shape: status/iter/output/failure_pattern)")
	}
	if _, ok := final["failure_pattern"]; !ok {
		t.Error("final.failure_pattern key missing (canonical trace schema requires it)")
	}
	if got, _ := final["critic_type"].(string); got != "structural" {
		t.Errorf("final.critic_type=%q, want \"structural\" (the l4 plan critic is the structural dry-run critic)", got)
	}

	// The real structural critic's output for a `hcloud ...` dry-run payload:
	// correctness/safety/traceability/spec_compliance 1.0, idempotency 0.5.
	// None of these are the old literals, and secops/finops are gone entirely.
	want := map[string]float64{
		"correctness":     1.0,
		"safety":          1.0,
		"idempotency":     0.5,
		"traceability":    1.0,
		"spec_compliance": 1.0,
	}
	scores := mapOf(t, trace["critic_scores"], "critic_scores")
	if len(scores) != len(want) {
		t.Errorf("critic_scores dims=%v, want exactly the structural critic's %d dims %v", scores, len(want), want)
	}
	for dim, w := range want {
		if got := scoreOf(t, scores[dim], "critic_scores."+dim); got != w {
			t.Errorf("critic_scores[%s]=%v, want %v (structural-critic value)", dim, got, w)
		}
	}

	// The persisted vector must equal what the per-step critic actually
	// returned for each planned step (single source of truth).
	if len(out.GCL.Decisions) == 0 {
		t.Fatal("expected at least one planned step for an RDS fault")
	}
	for _, d := range out.GCL.Decisions {
		for dim, w := range want {
			if got := d.GCL.Scores[dim]; got != w {
				t.Errorf("step %d critic %s=%v, want %v", d.Step, dim, got, w)
			}
		}
	}

	// final.dimensions mirrors critic_scores; final.overall is their mean.
	dims := mapOf(t, final["dimensions"], "final.dimensions")
	for dim, w := range want {
		if got := scoreOf(t, dims[dim], "final.dimensions."+dim); got != w {
			t.Errorf("final.dimensions[%s]=%v, want %v", dim, got, w)
		}
	}
	overall := scoreOf(t, final["overall"], "final.overall")
	if math.Abs(overall-0.9) > 1e-9 {
		t.Errorf("final.overall=%v, want 0.9 (mean of the five structural dims)", overall)
	}

	// iterations[0] mirrors the gcl.Iteration shape so cmd/aggregate.go's
	// lastCriticScores() picks up the real scores.
	iters, ok := trace["iterations"].([]any)
	if !ok || len(iters) != 1 {
		t.Fatalf("iterations=%v, want exactly one dry-run iteration", trace["iterations"])
	}
	first := mapOf(t, iters[0], "iterations[0]")
	if first["iter"] != float64(1) {
		t.Errorf("iterations[0].iter=%v, want 1", first["iter"])
	}
	if got, _ := first["decision"].(string); got != "PASS" {
		t.Errorf("iterations[0].decision=%q, want PASS", got)
	}
	gen := mapOf(t, first["generator"], "iterations[0].generator")
	for _, key := range []string{"command", "exit_code", "result_excerpt"} {
		if _, ok := gen[key]; !ok {
			t.Errorf("iterations[0].generator.%s missing", key)
		}
	}
	iterScores := mapOf(t, mapOf(t, first["critic"], "iterations[0].critic")["scores"], "iterations[0].critic.scores")
	for dim, w := range want {
		if got := scoreOf(t, iterScores[dim], "iterations[0].critic.scores."+dim); got != w {
			t.Errorf("iterations[0].critic.scores[%s]=%v, want %v", dim, got, w)
		}
	}

	// The trace records its own path (previously written before the field was
	// assigned, leaving an empty learning.trace_persisted on disk).
	learningBlock := mapOf(t, trace["learning"], "learning")
	if got, _ := learningBlock["trace_persisted"].(string); got != out.Learning.TracePersisted {
		t.Errorf("learning.trace_persisted=%q, want %q", got, out.Learning.TracePersisted)
	}
}

// TestHandleFault_UnmatchedFaultIsMaxIter asserts that a run with no planned
// steps reports MAX_ITER: nothing was verified, so it must not count as a PASS
// in the aggregate pass rate. It also asserts the trace still carries real
// critic output (computed on the empty dry-run payload) rather than literals.
func TestHandleFault_UnmatchedFaultIsMaxIter(t *testing.T) {
	root := t.TempDir()
	out := HandleFault(HandleFaultInput{Root: root, Fault: "smoke", Risk: "low"}, nil)
	trace := readTraceMap(t, out.Learning.TracePersisted)

	orch := mapOf(t, trace["orchestration"], "orchestration")
	steps := int(scoreOf(t, orch["step_count"], "orchestration.step_count"))
	if steps != 0 {
		t.Fatalf("orchestration.step_count=%d, want 0 for a fault matching no skill (this is the smoke shape the aggregators must skip)", steps)
	}

	final := mapOf(t, trace["final"], "final")
	if got, _ := final["status"].(string); got != "MAX_ITER" {
		t.Errorf("final.status=%q, want MAX_ITER (no step was verified)", got)
	}
	if iters, _ := trace["iterations"].([]any); len(iters) != 0 {
		t.Errorf("iterations=%v, want empty for a zero-step run", trace["iterations"])
	}
	if got, _ := trace["request"].(string); got != "smoke" {
		t.Errorf("request=%q, want smoke", got)
	}
	if got, _ := trace["fault"].(string); got != "smoke" {
		t.Errorf("fault=%q, want the smoke marker so readers skip this trace", got)
	}
	// Scores still come from the real critic (empty-plan evaluation).
	scores := mapOf(t, trace["critic_scores"], "critic_scores")
	want := gcl.StructuralCritic(gcl.GeneratorOutput{Command: "", ExitCode: 0, ResultExcerpt: "dry-run"}).Scores
	for dim, w := range want {
		if got := scoreOf(t, scores[dim], "critic_scores."+dim); got != w {
			t.Errorf("critic_scores[%s]=%v, want %v (empty-plan structural critic)", dim, got, w)
		}
	}
}

// TestFoldCriticScores pins the per-step reduction: safety is the AND across
// steps (one unsafe step fails the run), the other dimensions are means, and a
// dimension only some steps report is averaged over the steps that have it.
func TestFoldCriticScores(t *testing.T) {
	critic := func(scores map[string]float64) gcl.CriticResult {
		return gcl.CriticResult{Scores: scores}
	}
	cases := []struct {
		name string
		in   []gcl.CriticResult
		want map[string]float64
	}{
		{
			name: "safety is the minimum across steps",
			in: []gcl.CriticResult{
				critic(map[string]float64{"safety": 1.0, "correctness": 1.0}),
				critic(map[string]float64{"safety": 0.0, "correctness": 0.0}),
			},
			want: map[string]float64{"safety": 0.0, "correctness": 0.5},
		},
		{
			name: "dimensions are averaged across steps",
			in: []gcl.CriticResult{
				critic(map[string]float64{"safety": 1.0, "idempotency": 0.5}),
				critic(map[string]float64{"safety": 1.0, "idempotency": 1.0}),
			},
			want: map[string]float64{"safety": 1.0, "idempotency": 0.75},
		},
		{
			name: "dimension absent from some steps averages over the steps that report it",
			in: []gcl.CriticResult{
				critic(map[string]float64{"safety": 1.0, "traceability": 0.5}),
				critic(map[string]float64{"safety": 1.0}),
			},
			want: map[string]float64{"safety": 1.0, "traceability": 0.5},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := foldCriticScores(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("foldCriticScores=%v, want %v", got, tc.want)
			}
			for k, w := range tc.want {
				if got[k] != w {
					t.Errorf("folded[%s]=%v, want %v", k, got[k], w)
				}
			}
		})
	}
}

// TestTraceCriticResult_NoStepsUsesRealCritic asserts the zero-step path still
// produces real critic output instead of leaving an empty/None vector.
func TestTraceCriticResult_NoStepsUsesRealCritic(t *testing.T) {
	got := traceCriticResult(nil, "hcloud ecs list-servers")
	want := gcl.StructuralCritic(gcl.GeneratorOutput{Command: "hcloud ecs list-servers", ExitCode: 0, ResultExcerpt: "dry-run"})
	if len(got.Scores) != len(want.Scores) {
		t.Fatalf("scores=%v, want the structural critic's %v", got.Scores, want.Scores)
	}
	for dim, w := range want.Scores {
		if got.Scores[dim] != w {
			t.Errorf("scores[%s]=%v, want %v", dim, got.Scores[dim], w)
		}
	}
	if got.Mode != "structural-only" {
		t.Errorf("mode=%q, want structural-only", got.Mode)
	}
}

// TestGCLFinalStatus pins the mapping onto the GCL trace status enum: a RETRY
// verdict (a step below a rubric threshold) is reported as MAX_ITER because the
// dry-run pipeline never re-runs, and a zero-step run verified nothing.
func TestGCLFinalStatus(t *testing.T) {
	cases := []struct {
		name          string
		scores        map[string]float64
		overallSafety bool
		steps         int
		want          string
	}{
		{
			name:          "all dimensions pass",
			scores:        map[string]float64{"correctness": 1.0, "safety": 1.0, "idempotency": 1.0, "traceability": 1.0, "spec_compliance": 1.0},
			overallSafety: true,
			steps:         2,
			want:          "PASS",
		},
		{
			name:          "one dimension below threshold is not a pass",
			scores:        map[string]float64{"correctness": 0.4, "safety": 1.0, "idempotency": 1.0, "traceability": 1.0, "spec_compliance": 1.0},
			overallSafety: true,
			steps:         1,
			want:          "MAX_ITER",
		},
		{
			name:          "unsafe run is SAFETY_FAIL",
			scores:        map[string]float64{"correctness": 1.0, "safety": 0.0, "idempotency": 1.0, "traceability": 1.0, "spec_compliance": 1.0},
			overallSafety: false,
			steps:         1,
			want:          "SAFETY_FAIL",
		},
		{
			name:          "zero steps verified nothing",
			scores:        map[string]float64{"correctness": 1.0, "safety": 1.0, "idempotency": 1.0, "traceability": 1.0, "spec_compliance": 1.0},
			overallSafety: true,
			steps:         0,
			want:          "MAX_ITER",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := gclFinalStatus(tc.scores, tc.overallSafety, tc.steps); got != tc.want {
				t.Errorf("gclFinalStatus=%q, want %q", got, tc.want)
			}
		})
	}
}
