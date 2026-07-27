package gcl

import (
	"context"
	"strings"
	"testing"
)

// TestSchemaInvalidOperationIntent is the spec-mandated (A5) test:
// an operation_intent missing the required "safety_class" field must
// cause Run() to exit with ExitUsage (and a schema-error message).
func TestSchemaInvalidOperationIntent(t *testing.T) {
	cfg := RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "delete something",
		Command: "echo ok",
		// operation present, but safety_class missing
		OperationIntent: `{"operation":"delete","expected_state":"gone"}`,
		MaxIter:         1,
		Timeout:         5,
		Root:            t.TempDir(),
	}
	result := Run(cfg)
	if result.ExitCode != ExitUsage {
		t.Errorf("operation_intent missing safety_class must fail with ExitUsage, got %d",
			result.ExitCode)
	}
}

// schemaInvalidFakeCritic returns a Critic whose Score produces a
// schema-invalid result. Used to exercise the spec's A6 path without
// the flaky sh-subprocess fixture.
type schemaInvalidFakeCritic struct{}

func (schemaInvalidFakeCritic) Score(_ context.Context, _ GeneratorOutput) CriticResult {
	return CriticResult{
		// safety=1.0 (passes), other dims below threshold → RETRY.
		// Mode marks the wire failure; Blocking signals fail-closed.
		Scores: map[string]float64{
			"correctness":     0.0,
			"safety":          1.0,
			"idempotency":     0.0,
			"traceability":    0.0,
			"spec_compliance": 0.0,
		},
		Suggestions: []string{"critic output failed schema validation: scores missing required fields"},
		Blocking:    true,
		Mode:        "schema-invalid: 1 issue(s): $.scores: missing property 'correctness'",
	}
}

// TestExternalCriticSchemaInvalid is the spec-mandated (A6) test:
// an external Critic subprocess whose JSON fails schema validation
// must yield Mode starting with "schema-invalid: " and a blocking
// flag set, so the rubric thresholds reject all scores (→ RETRY).
func TestExternalCriticSchemaInvalid(t *testing.T) {
	cfg := RunConfig{
		Skill:   "huaweicloud-ecs-ops",
		Request: "list servers",
		Command: "echo ok",
		MaxIter: 1,
		Timeout: 5,
		Critic:  schemaInvalidFakeCritic{},
		Root:    t.TempDir(),
	}
	result := Run(cfg)
	// With schema-invalid Critic output, every dimension except
	// safety is below threshold → RETRY (no PASS) → MAX_ITER after MaxIter=1.
	if result.ExitCode != ExitMaxIter {
		t.Errorf("schema-invalid Critic → MAX_ITER, got %d", result.ExitCode)
	}
	// Verify the Critic's schema-invalid Mode was actually used in
	// the persisted trace.
	trace, err := readLastTrace(t, result.TracePath)
	if err != nil {
		t.Fatalf("readLastTrace: %v", err)
	}
	if len(trace.Iterations) == 0 {
		t.Fatal("trace should have at least one iteration")
	}
	mode := trace.Iterations[0].Critic.Mode
	if !strings.HasPrefix(mode, "schema-invalid") {
		t.Errorf("Critic.Mode should start with 'schema-invalid', got %q", mode)
	}
	if !trace.Iterations[0].Critic.Blocking {
		t.Error("Schema-invalid Critic should set Blocking=true")
	}
}
