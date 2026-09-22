package l4

import (
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/gcl"
	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/schema"
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

// canonicalTraceSchemaPath is the CES-owned canonical GCL trace schema —
// the same asset cmd/validate_contract.go reads and the one docs/gcl-spec.md
// points at. Tests validate against THIS file (not internal/embed's copy,
// which has drifted) so the writer cannot silently diverge from the contract
// the repo advertises.
var canonicalTraceSchemaPath = filepath.Join(
	"..", "..", "..", "huaweicloud-ces-ops", "assets", "gcl-trace.schema.json",
)

// TestHandleFault_TraceSatisfiesCanonicalSchema is the guard for the P0
// blocker "the L4 writer's own trace fails the canonical schema": every trace
// family the repo advertises as schema-compatible MUST validate against
// huaweicloud-ces-ops/assets/gcl-trace.schema.json with ZERO violations. A
// trace that fails validation is exactly what lets a crafted file impersonate
// a real run — the consumer's schema check is only meaningful if the
// orchestrator's own output passes it.
func TestHandleFault_TraceSatisfiesCanonicalSchema(t *testing.T) {
	schemaData, err := os.ReadFile(canonicalTraceSchemaPath)
	if err != nil {
		t.Fatalf("read canonical trace schema %s: %v", canonicalTraceSchemaPath, err)
	}

	cases := []struct {
		name     string
		fault    string
		resource string
		risk     string
	}{
		{name: "fault with a planned step", fault: "RDS connection timeout", resource: "rds:instance", risk: "medium"},
		// Zero planned steps: the smoke/MAX_ITER shape, which carries an empty
		// iterations array. It must validate too — it is the shape the
		// aggregators see most often.
		{name: "fault matching no skill", fault: "smoke", risk: "low"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			out := HandleFault(HandleFaultInput{
				Root:     root,
				Fault:    tc.fault,
				Resource: tc.resource,
				Risk:     tc.risk,
			}, nil)
			if out.Learning.TracePersisted == "" {
				t.Fatal("orchestrator did not persist a trace")
			}
			raw, err := os.ReadFile(out.Learning.TracePersisted)
			if err != nil {
				t.Fatalf("read trace: %v", err)
			}

			errs, verr := schema.ValidateFile(raw, schemaData)
			if verr != nil {
				t.Fatalf("validate %s: %v", out.Learning.TracePersisted, verr)
			}
			if len(errs) > 0 {
				t.Errorf("orchestrator trace violates %s (%d violations):\n  %s",
					canonicalTraceSchemaPath, len(errs), strings.Join(errs, "\n  "))
			}
		})
	}
}

// TestHandleFault_TraceCanonicalFieldValues pins the values the canonical
// schema cannot (or should not) constrain by itself, so they cannot drift:
// trace_schema_version is the schema's `const`, rubric_version must match the
// value the GCL writer uses, and masked_fields must be the honest declaration
// of what THIS writer masks. internal/l4 applies no masking (it persists the
// caller's fault text verbatim and its readers rely on that: the smoke marker
// is read from `request`/`fault`), so the list must be empty — claiming a
// masked field the writer never masks would be a false trust signal.
func TestHandleFault_TraceCanonicalFieldValues(t *testing.T) {
	root := t.TempDir()
	out := HandleFault(HandleFaultInput{
		Root:     root,
		Fault:    "RDS connection timeout",
		Resource: "rds:instance",
		Risk:     "medium",
	}, nil)
	trace := readTraceMap(t, out.Learning.TracePersisted)

	if got, _ := trace["trace_schema_version"].(string); got != "v1" {
		t.Errorf("trace_schema_version=%q, want \"v1\" (canonical schema const)", got)
	}
	if got, _ := trace["rubric_version"].(string); got != "v1" {
		t.Errorf("rubric_version=%q, want the GCL writer's \"v1\"", got)
	}
	masked, ok := trace["masked_fields"].([]any)
	if !ok {
		t.Fatalf("masked_fields=%T, want a JSON array (canonical schema requires it)", trace["masked_fields"])
	}
	if len(masked) != 0 {
		t.Errorf("masked_fields=%v, want [] — internal/l4 masks nothing, so it must not claim to", masked)
	}
	// A non-canonical `operation_intent` block (the old {goal, risk_class}
	// object) was removed: the canonical schema requires operation /
	// resource_scope / expected_state / safety_class whenever the key is
	// present, and a dry-run planner has no honest value for them.
	if v, ok := trace["operation_intent"]; ok {
		t.Errorf("trace carries operation_intent=%v; the L4 dry-run has no canonical intent (remove it or emit the four required fields)", v)
	}

	// The dry-run iteration must carry the full generator shape the canonical
	// schema requires (stdout_len/stderr_len/args in addition to
	// command/exit_code/result_excerpt).
	iters, _ := trace["iterations"].([]any)
	if len(iters) != 1 {
		t.Fatalf("iterations=%v, want exactly one dry-run iteration", trace["iterations"])
	}
	gen := mapOf(t, mapOf(t, iters[0], "iterations[0]")["generator"], "iterations[0].generator")
	if got := scoreOf(t, gen["stdout_len"], "generator.stdout_len"); got != 0 {
		t.Errorf("generator.stdout_len=%v, want 0 (a dry run produces no output)", got)
	}
	if got := scoreOf(t, gen["stderr_len"], "generator.stderr_len"); got != 0 {
		t.Errorf("generator.stderr_len=%v, want 0 (a dry run produces no output)", got)
	}
	args := mapOf(t, gen["args"], "generator.args")
	if got := scoreOf(t, args["iter"], "generator.args.iter"); got != 1 {
		t.Errorf("generator.args.iter=%v, want 1", got)
	}
	if v, ok := args["critic_feedback"]; !ok || v != nil {
		t.Errorf("generator.args.critic_feedback=%v, want null (iteration 1 had no feedback)", v)
	}
}

// TestHandleFault_TraceWriteErrorIsNotReportedAsPersisted covers the P0 minor
// finding "write errors ignored / TracePersisted set before the write": when
// the trace cannot be written, `learning.trace_persisted` must NOT point at a
// file that does not exist (that is how a silent write failure turns into a
// phantom PASS in the aggregates), and the operator must see a WARN naming the
// path and the error.
func TestHandleFault_TraceWriteErrorIsNotReportedAsPersisted(t *testing.T) {
	root := t.TempDir()
	// A regular file where the audit-results directory must go: MkdirAll and
	// the write both fail with ENOTDIR, deterministically.
	if err := os.WriteFile(filepath.Join(root, "audit-results"), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	var out *OrchestratorOutput
	stderr := captureStderr(t, func() {
		out = HandleFault(HandleFaultInput{
			Root:     root,
			Fault:    "RDS connection timeout",
			Resource: "rds:instance",
			Risk:     "medium",
		}, nil)
	})

	if out.Learning.TracePersisted != "" {
		t.Errorf("learning.trace_persisted=%q after a failed write, want \"\" (nothing was persisted)",
			out.Learning.TracePersisted)
	}
	if !strings.Contains(stderr, "WARN") {
		t.Errorf("no WARN on stderr after a failed trace write; stderr=%q", stderr)
	}
	if !strings.Contains(stderr, "orchestrator-trace-") {
		t.Errorf("WARN does not name the trace path; stderr=%q", stderr)
	}
	if !strings.Contains(stderr, "audit-results") {
		t.Errorf("WARN does not name the failing path; stderr=%q", stderr)
	}
	// The return contract is unchanged: the pipeline still completes.
	if out.Decision == "" {
		t.Error("HandleFault must still return a decision when the trace write fails")
	}
}

// captureStderr runs fn with os.Stderr redirected to a pipe and returns what
// fn wrote to it.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close stderr pipe: %v", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stderr pipe: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close stderr reader: %v", err)
	}
	return string(data)
}
