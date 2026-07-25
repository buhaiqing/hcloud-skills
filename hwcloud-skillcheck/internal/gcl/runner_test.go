package gcl

import (
	"encoding/json"
	"os"
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
