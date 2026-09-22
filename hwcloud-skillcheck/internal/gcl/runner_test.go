package gcl

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// maskPattern is the raw string form of "<masked>" as it appears in a JSON file.
// Go's json.Marshal encodes < as \u003c, so this is the actual bytes in the file.
const maskPattern = "<masked>"

// ---- Decide ---------------------------------------------------------------

func TestDecide_Pass(t *testing.T) {
	scores := map[string]float64{
		"correctness":     1.0,
		"safety":          1.0,
		"idempotency":     0.5,
		"traceability":    0.5,
		"spec_compliance": 1.0,
	}
	if got := Decide(scores); got != "PASS" {
		t.Errorf("Decide all-pass: got %q, want PASS", got)
	}
}

func TestDecide_SafetyFail(t *testing.T) {
	// Safety score of 0 must always return SAFETY_FAIL, regardless of other scores.
	scores := map[string]float64{
		"correctness":     1.0,
		"safety":          0.0,
		"idempotency":     1.0,
		"traceability":    1.0,
		"spec_compliance": 1.0,
	}
	if got := Decide(scores); got != "SAFETY_FAIL" {
		t.Errorf("Decide safety=0: got %q, want SAFETY_FAIL", got)
	}
}

func TestDecide_Retry(t *testing.T) {
	// Below-threshold dimension should return RETRY (not MAX_ITER which is only at loop end).
	scores := map[string]float64{
		"correctness":     0.0, // below 0.5 threshold
		"safety":          1.0,
		"idempotency":     1.0,
		"traceability":    1.0,
		"spec_compliance": 1.0,
	}
	if got := Decide(scores); got != "RETRY" {
		t.Errorf("Decide below threshold: got %q, want RETRY", got)
	}
}

func TestDecide_AllBelowThreshold(t *testing.T) {
	// All dimensions below threshold → RETRY.
	// Note: MAX_ITER is a loop-exit status, not a Decide() return value.
	// Decide() only returns PASS | RETRY | SAFETY_FAIL.
	scores := map[string]float64{
		"correctness":     0.0,
		"safety":          1.0,
		"idempotency":     0.0,
		"traceability":    0.0,
		"spec_compliance": 0.0,
	}
	if got := Decide(scores); got != "RETRY" {
		t.Errorf("Decide all-fail: got %q, want RETRY", got)
	}
}

// ---- StructuralCritic ----------------------------------------------------

func TestStructuralCritic_ExitCodeZero(t *testing.T) {
	gen := GeneratorOutput{
		Command:       "echo hello",
		ExitCode:      0,
		ResultExcerpt: "hello",
		StdoutLen:     5,
		StderrLen:     0,
	}
	result := StructuralCritic(gen)
	if result.Scores["correctness"] != 1.0 {
		t.Errorf("correctness with exit 0: got %.2f, want 1.0", result.Scores["correctness"])
	}
	if result.Scores["safety"] != 1.0 {
		t.Errorf("safety with no leak: got %.2f, want 1.0", result.Scores["safety"])
	}
	if result.Blocking {
		t.Error("Blocking should be false when exit_code=0 and no leak")
	}
}

func TestStructuralCritic_ExitCodeNonZero(t *testing.T) {
	gen := GeneratorOutput{
		Command:       "hcloud ecs list",
		ExitCode:      1,
		ResultExcerpt: "InvalidParameter",
		StdoutLen:     0,
		StderrLen:     50,
	}
	result := StructuralCritic(gen)
	if result.Scores["correctness"] != 0.0 {
		t.Errorf("correctness with exit 1: got %.2f, want 0.0", result.Scores["correctness"])
	}
	if !result.Blocking {
		t.Error("Blocking should be true when exit_code != 0")
	}
}

func TestStructuralCritic_CredentialLeak(t *testing.T) {
	// Result excerpt contains a credential pattern that should be detected.
	gen := GeneratorOutput{
		Command:       "echo HW_SECRET_ACCESS_KEY=ABCDEFGHIJKLMNOPQRST",
		ExitCode:      0,
		ResultExcerpt: "access key: secret123",
		StdoutLen:     30,
		StderrLen:     0,
	}
	result := StructuralCritic(gen)
	if result.Scores["safety"] != 0.0 {
		t.Errorf("safety with credential leak: got %.2f, want 0.0", result.Scores["safety"])
	}
	if !result.Blocking {
		t.Error("Blocking should be true when credential leak detected")
	}
}

// ---- Run -----------------------------------------------------------------

func TestExternalCritic_PassesThrough(t *testing.T) {
	if _, err := exec.LookPath("/bin/echo"); err != nil {
		t.Skip("/bin/echo not available on this platform")
	}
	// /bin/echo prints its arguments, so we feed it a fixed CriticResult JSON
	// and assert the Run happily accepts the synthetic scores.
	// Skipping when /bin/echo is absent (Windows / stripped containers).
	criticJSON := `{"scores":{"correctness":1,"safety":1,"idempotency":1,"traceability":1,"spec_compliance":1}}`
	cfg := RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "list servers",
		Command: "echo hello",
		MaxIter: 1,
		Timeout: 10,
		Critic:  NewExternalCritic("/bin/echo", criticJSON),
	}
	result := Run(cfg)
	if result.ExitCode != 0 {
		t.Errorf("Run with /bin/echo critic: exit code %d, want 0", result.ExitCode)
	}
}

func TestRun_GenOutput(t *testing.T) {
	cfg := RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "list servers",
		Command: "echo hello",
		MaxIter: 2,
		Timeout: 10,
	}
	result := Run(cfg)

	// Should exit 0 (PASS) because echo always succeeds.
	if result.ExitCode != 0 {
		t.Errorf("Run echo: exit code %d, want 0", result.ExitCode)
	}
	if result.TracePath == "" {
		t.Error("TracePath should not be empty after RUN")
	}
}

func TestRun_SafetyFail(t *testing.T) {
	// A command that outputs a credential leak.
	cfg := RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "get secret",
		Command: "echo HW_SECRET_ACCESS_KEY=MySecretToken123456789012",
		MaxIter: 2,
		Timeout: 10,
	}
	result := Run(cfg)

	// SAFETY_FAIL exit code = 3.
	if result.ExitCode != 3 {
		t.Errorf("Run with credential leak: exit code %d, want 3", result.ExitCode)
	}
}

func TestRun_MaxIter(t *testing.T) {
	// exit 1 → structural critic: correctness=0, safety=1.0 → RETRY.
	// With MaxIter=1, loop exhausts → MAX_ITER → exit 1.
	cfg := RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "list servers",
		Command: "echo 'an error occurred' && exit 1",
		MaxIter: 1,
		Timeout: 10,
	}
	result := Run(cfg)
	// exit 1 → correctness=0 → RETRY → MAX_ITER → exit 1
	if result.ExitCode != 1 {
		t.Errorf("Run with exit 1: exit code %d, want 1 (MAX_ITER)", result.ExitCode)
	}
	if result.TracePath == "" {
		t.Error("TracePath should not be empty after MAX_ITER Run")
	}
}

func TestRun_Timeout(t *testing.T) {
	cfg := RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "sleep",
		Command: "sleep 2",
		MaxIter: 1,
		Timeout: 1, // 1 second timeout for a 2-second command
	}
	result := Run(cfg)

	// With MaxIter=1, timeout produces a RETRY decision (exit 124 → correctness=0
	// from structural critic → RETRY), and the loop exits after 1 iteration → MAX_ITER → exit 1.
	// The key is that Run completes without panicking and produces a trace.
	if result.ExitCode != 1 {
		t.Errorf("Run timeout: exit code %d, want 1 (MAX_ITER)", result.ExitCode)
	}
	if result.TracePath == "" {
		t.Error("TracePath should not be empty after timeout Run")
	}
}

// ---- PersistTrace --------------------------------------------------------

func TestPersistTrace(t *testing.T) {
	tmp := t.TempDir()
	trace := &GCLTrace{
		TraceSchemaVersion: "v1",
		Skill:              "test-skill",
		Request:            "test request",
		OperationIntent:    nil,
		RubricVersion:      "v1",
		MaskedFields:       []string{"request"},
		Iterations:         nil,
		Final: &FinalResult{
			Status: "PASS",
			Iter:   1,
			Output: "ok",
		},
	}

	path, err := PersistTrace(trace, tmp)
	if err != nil {
		t.Fatalf("PersistTrace error: %v", err)
	}
	if path == "" {
		t.Error("PersistTrace returned empty path")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Trace file not created at %s", path)
	}
}

func TestMaskedFields(t *testing.T) {
	// Verify that credential values are masked in the persisted trace.
	cfg := RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "list with secret",
		Command: "echo HW_SECRET_ACCESS_KEY=RealSecretToken123456789012",
		MaxIter: 1,
		Timeout: 10,
	}
	result := Run(cfg)

	if result.TracePath == "" {
		t.Fatal("TracePath should not be empty after Run")
	}
	data, err := os.ReadFile(result.TracePath)
	if err != nil {
		t.Fatalf("failed to read trace at %s: %v", result.TracePath, err)
	}
	traceJSON := string(data)
	// The real secret token must NOT appear in the trace.
	if strings.Contains(traceJSON, "RealSecretToken123456789012") {
		t.Error("raw secret token found in trace — MaskSecrets failed")
	}
	// Also verify the trace is valid JSON and the masked value is present.
	var trace GCLTrace
	if err := json.Unmarshal(data, &trace); err != nil {
		t.Fatalf("trace is not valid JSON: %v", err)
	}
	// Verify the masked value is in the unmarshaled trace.
	// After unmarshaling, Go decodes \u003c back to '<'.
	traceStr := strings.Join([]string{
		trace.Request,
		trace.Iterations[0].Generator.Command,
		trace.Iterations[0].Generator.ResultExcerpt,
	}, " ")
	if !strings.Contains(traceStr, "<masked>") {
		t.Errorf("<masked> not found in unmarshaled trace fields")
	}
}

// ---- exit codes -----------------------------------------------------------

func TestExitCodes(t *testing.T) {
	// Verify that the expected exit code constants match our conventions.
	tests := []struct {
		name     string
		command  string
		maxIter  int
		timeout  int
		wantCode int
	}{
		{"pass", "echo ok", 1, 10, 0},
		{"safety_fail", "echo HW_SECRET_ACCESS_KEY=Leak", 1, 10, 3},
		{"timeout", "sleep 10", 1, 1, 1}, // timeout → RETRY → MAX_ITER (exit 1)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := RunConfig{
				Skill:   "test",
				Request: "test",
				Command: tt.command,
				MaxIter: tt.maxIter,
				Timeout: tt.timeout,
			}
			result := Run(cfg)
			if result.ExitCode != tt.wantCode {
				t.Errorf("exit code = %d, want %d", result.ExitCode, tt.wantCode)
			}
		})
	}
}

// ---- Trace path naming ----------------------------------------------------

func TestTracePath_Naming(t *testing.T) {
	tmp := t.TempDir()
	trace := &GCLTrace{
		TraceSchemaVersion: "v1",
		Skill:              "huaweicloud-ecs-ops",
		Request:            "list",
		OperationIntent:    nil,
		RubricVersion:      "v1",
		MaskedFields:       []string{},
		Iterations:         []Iteration{},
		Final:              &FinalResult{Status: "PASS", Iter: 1},
	}
	path, err := PersistTrace(trace, tmp)
	if err != nil {
		t.Fatalf("PersistTrace: %v", err)
	}
	base := filepath.Base(path)
	if base == "" || len(base) < 10 {
		t.Errorf("Trace filename too short or empty: %q", base)
	}
	if filepath.Ext(path) != ".json" {
		t.Errorf("Trace should have .json extension, got %s", filepath.Ext(path))
	}
}

// ---- critic_type provenance (CA-4) ---------------------------------------

// criticAllPassJSON is an ExternalCritic wire payload whose scores satisfy
// every rubric threshold, so Run reaches PASS whichever critic produced them.
const criticAllPassJSON = `{"scores":{"correctness":1,"safety":1,"idempotency":1,"traceability":1,"spec_compliance":1}}`

// TestRun_TraceCriticType pins the provenance recorded in the persisted
// trace: final.critic_type must distinguish the in-process deterministic
// proxy ("structural") from a real out-of-process critic ("external") so a
// consumer never reads a heuristic score as an LLM verdict. The assertion is
// against the bytes on disk, not the in-memory trace — the field has to
// survive PersistTrace.
func TestRun_TraceCriticType(t *testing.T) {
	if _, err := exec.LookPath("/bin/echo"); err != nil {
		t.Skip("/bin/echo not available on this platform")
	}
	tests := []struct {
		name   string
		critic Critic
		want   string
	}{
		{"default critic is structural", nil, "structural"},
		{"explicit structural adapter", StructuralCriticAdapter{}, "structural"},
		{"external critic subprocess", NewExternalCritic("/bin/echo", criticAllPassJSON), "external"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Run(RunConfig{
				Skill:   "huaweicloud-ecs-ops",
				Request: "list servers",
				Command: "echo ok",
				MaxIter: 1,
				Timeout: 10,
				Root:    t.TempDir(),
				Critic:  tt.critic,
			})
			if result.ExitCode != ExitOK {
				t.Fatalf("Run exit code = %d, want %d (PASS)", result.ExitCode, ExitOK)
			}
			data, err := os.ReadFile(result.TracePath)
			if err != nil {
				t.Fatalf("read trace: %v", err)
			}
			var trace GCLTrace
			if err := json.Unmarshal(data, &trace); err != nil {
				t.Fatalf("trace is not valid JSON: %v", err)
			}
			if trace.Final == nil {
				t.Fatal("persisted trace has no final block")
			}
			if trace.Final.CriticType != tt.want {
				t.Errorf("final.critic_type = %q, want %q", trace.Final.CriticType, tt.want)
			}
			want := `"critic_type": "` + tt.want + `"`
			if !strings.Contains(string(data), want) {
				t.Errorf("raw trace JSON missing %s", want)
			}
		})
	}
}

// TestRun_MaxIterTraceCarriesFinal pins that a MAX_ITER run persists a final
// block too. cmd/aggregate.go and internal/learning REQUIRE a non-empty
// "final" and classify a trace without one as a smoke artifact, so an omitted
// final would silently drop MAX_ITER runs out of pass_rate/MAX_ITER
// accounting (aggregate would also bucket them as UNKNOWN).
func TestRun_MaxIterTraceCarriesFinal(t *testing.T) {
	result := Run(RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "list servers",
		Command: "echo 'an error occurred' && exit 1",
		MaxIter: 1,
		Timeout: 10,
		Root:    t.TempDir(),
	})
	if result.ExitCode != ExitMaxIter {
		t.Fatalf("Run exit code = %d, want %d (MAX_ITER)", result.ExitCode, ExitMaxIter)
	}
	data, err := os.ReadFile(result.TracePath)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	var trace GCLTrace
	if err := json.Unmarshal(data, &trace); err != nil {
		t.Fatalf("trace is not valid JSON: %v", err)
	}
	if trace.Final == nil {
		t.Fatal("MAX_ITER trace has no final block")
	}
	if trace.Final.Status != "MAX_ITER" {
		t.Errorf("final.status = %q, want MAX_ITER", trace.Final.Status)
	}
	if trace.Final.CriticType != "structural" {
		t.Errorf("final.critic_type = %q, want structural", trace.Final.CriticType)
	}
	if len(trace.Final.Unresolved) == 0 {
		t.Error("MAX_ITER final should list the rubric dimensions that never passed")
	}
}

// TestRun_TraceRecordsL2SkippedNoSchema pins the integration fix for the L2
// hallucination check: the HallucinationResult is now persisted on every run,
// not only when L1/L2 blocked, so "the skill ships no OpenAPI schema, nothing
// was checked" is visible in the trace instead of looking like a clean check.
func TestRun_TraceRecordsL2SkippedNoSchema(t *testing.T) {
	result := Run(RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "list servers",
		Command: "echo '{}'", // valid JSON output, so L2 reaches the schema lookup
		MaxIter: 1,
		Timeout: 10,
		Root:    t.TempDir(), // no references/openapi-schema.json in this skill root
	})
	if result.ExitCode != ExitOK {
		t.Fatalf("Run exit code = %d, want %d (PASS)", result.ExitCode, ExitOK)
	}
	data, err := os.ReadFile(result.TracePath)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("trace is not valid JSON: %v", err)
	}
	hd, _ := raw["hallucination_detection"].(map[string]any)
	if hd == nil {
		t.Fatal("a passing run persisted no hallucination_detection block")
	}
	l2, _ := hd["l2"].(map[string]any)
	if l2 == nil {
		t.Fatal("hallucination_detection has no l2 block")
	}
	if got, _ := l2["status"].(string); got != string(L2StatusSkippedNoSchema) {
		t.Errorf("l2.status = %q, want %q", got, L2StatusSkippedNoSchema)
	}
	if blocked, _ := l2["blocked"].(bool); blocked {
		t.Error("a skipped L2 must not block the run")
	}
}

// TestCriticTypeOf_WrappedExternalCritic pins the Critic finding that kind
// reporting must survive composition: a wrapper that embeds ExternalCritic is
// still an external critic, not the in-process proxy (a wrong label would
// present heuristic scores as an LLM verdict in the trace).
func TestCriticTypeOf_WrappedExternalCritic(t *testing.T) {
	type wrappedCritic struct{ ExternalCritic }
	base := NewExternalCritic("/bin/echo")
	wrapped := &wrappedCritic{ExternalCritic: *base}

	cases := []struct {
		name   string
		critic Critic
		want   string
	}{
		{"pointer", base, "external"},
		{"value", *base, "external"},
		{"wrapped pointer", wrapped, "external"},
		{"nil critic", nil, "structural"},
		{"structural adapter", StructuralCriticAdapter{}, "structural"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := criticTypeOf(tc.critic); got != tc.want {
				t.Errorf("criticTypeOf(%T) = %q, want %q", tc.critic, got, tc.want)
			}
		})
	}
}

// TestMaskFlagValues pins the redaction rule: flag names stay (they are the
// diagnostic value), values disappear for both `--flag=value` and
// `--flag value` forms. A trace must never echo a password-shaped value the
// generator put on the command line.
func TestMaskFlagValues(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"equals form", "--prod-db-password=Sup3rSecretValue123", "--prod-db-password=<masked>"},
		{"space form", "--server-id ecs-abc12345", "--server-id <masked>"},
		{"mixed", "hcloud ecs list --server-id=ecs-abc --region cn-north-4", "hcloud ecs list --server-id=<masked> --region <masked>"},
		{"no flags", "invalid JSON body", "invalid JSON body"},
		{"hyphenated word is not a flag", "hcloud ecs list-servers cn-north-4", "hcloud ecs list-servers cn-north-4"},
		{"inside bracket list", "flags [--server-id=ecs-abc12345 blocked]", "flags [--server-id=<masked> blocked]"},
		{"empty", "", ""},
		{"trailing flag", "--verbose", "--verbose"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := maskFlagValues(tc.in); got != tc.want {
				t.Errorf("maskFlagValues(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestRun_HallucinationBlockIsMasked is the regression test for the leak the
// safety Critic executed: a command carrying a password-shaped flag value and a
// resource ID appeared verbatim in hallucination_detection.l1.invalid_flags /
// .details / .summary and in final.failure_pattern.command while
// iterations[].generator.command was already "<masked>".
func TestRun_HallucinationBlockIsMasked(t *testing.T) {
	const secret = "Sup3rSecretValue123"
	const resource = "ecs-abc12345"

	result := Run(RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "list servers",
		Command: "echo ok --server-id=" + resource + " --prod-db-password=" + secret,
		MaxIter: 1,
		Timeout: 10,
		Root:    t.TempDir(),
	})
	data, err := os.ReadFile(result.TracePath)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	raw := string(data)
	if strings.Contains(raw, secret) {
		t.Errorf("trace leaks the password-shaped flag value %q", secret)
	}
	if strings.Contains(raw, resource) {
		t.Errorf("trace leaks the resource ID %q", resource)
	}
	var trace GCLTrace
	if err := json.Unmarshal(data, &trace); err != nil {
		t.Fatalf("trace is not valid JSON: %v", err)
	}
	if trace.HallucinationDetection != nil {
		if !strings.Contains(trace.HallucinationDetection.Summary, "<masked>") &&
			len(trace.HallucinationDetection.Summary) > 0 {
			t.Logf("summary kept no masked marker: %q", trace.HallucinationDetection.Summary)
		}
	}
	if trace.Final != nil && trace.Final.FailurePattern != nil {
		if trace.Final.FailurePattern.Command != "<masked>" {
			t.Errorf("final.failure_pattern.command = %q, want <masked>", trace.Final.FailurePattern.Command)
		}
	}
}
