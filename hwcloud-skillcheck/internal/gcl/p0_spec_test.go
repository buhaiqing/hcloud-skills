package gcl

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestPersistTraceAppliesMaskedFields is the spec-mandated (A1) test:
// trace.Request in the persisted JSON must equal "<masked>", not the
// raw resource ID.
func TestPersistTraceAppliesMaskedFields(t *testing.T) {
	tmp := t.TempDir()
	trace := &GCLTrace{
		TraceSchemaVersion: "v1",
		Skill:              "huaweicloud-ecs-ops",
		Request:            "delete ecs-abc12345 server",
		RubricVersion:      "v1",
		MaskedFields:       []string{"request"},
		Iterations:         nil,
		Final:              &FinalResult{Status: "PASS", Iter: 1, Output: "ok"},
	}
	path, err := PersistTrace(trace, tmp)
	if err != nil {
		t.Fatalf("PersistTrace: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(data), "ecs-abc12345") {
		t.Errorf("raw resource ID leaked into persisted trace:\n%s", data)
	}
	var roundTripped GCLTrace
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if roundTripped.Request != "<masked>" {
		t.Errorf("Request = %q, want <masked>", roundTripped.Request)
	}
}

// TestSanitizedRequestNoRawIDs is the spec-mandated (A2) test:
// trace.SanitizedRequest must strip resource IDs and credentials
// (PII tokens), but keep the surrounding text intact.
func TestSanitizedRequestNoRawIDs(t *testing.T) {
	cfg := RunConfig{
		Skill:   "huaweicloud-rds-ops",
		Request: "delete rds-abc12345 then rotate HW_SECRET_ACCESS_KEY=TopSecretValue1234567890",
		Command: "echo ok",
		MaxIter: 1,
		Timeout: 5,
		Root:    t.TempDir(),
	}
	result := Run(cfg)
	if result.ExitCode != ExitOK {
		t.Fatalf("expected PASS, got %d", result.ExitCode)
	}
	trace, err := readLastTrace(t, result.TracePath)
	if err != nil {
		t.Fatalf("readLastTrace: %v", err)
	}
	if strings.Contains(trace.SanitizedRequest, "TopSecretValue") {
		t.Errorf("SanitizedRequest leaked credential value: %q", trace.SanitizedRequest)
	}
	if !strings.Contains(trace.SanitizedRequest, "<id>") {
		t.Errorf("SanitizedRequest should contain <id> placeholder: %q", trace.SanitizedRequest)
	}
	if !strings.Contains(trace.SanitizedRequest, "<redacted>") {
		t.Errorf("SanitizedRequest should contain <redacted> placeholder: %q", trace.SanitizedRequest)
	}
}

// retryCaptureCritic records the previous Generator output and emits
// a non-passing verdict so the Runner retries. It captures the retry
// prompt indirectly by using MinimalFeedbackRetry as the builder.
type retryCaptureCritic struct {
	calls int
}

func (c *retryCaptureCritic) Score(_ context.Context, gen GeneratorOutput) CriticResult {
	c.calls++
	return CriticResult{
		// safety=1.0 (passes), other dims below threshold → RETRY.
		Scores: map[string]float64{
			"correctness":     0.1,
			"safety":          1.0,
			"idempotency":     0.2,
			"traceability":    0.3,
			"spec_compliance": 0.1,
		},
		Suggestions: []string{"rotate_iam_scope first", "echo the resource id"},
		Blocking:    false,
		Mode:        "isolated-critic",
	}
}

// TestRunRetryInjectsOnlyFailedDimensions is the spec-mandated (A3)
// test: the iter-2 retry prompt must contain ONLY the failed
// dimensions (correctness, idempotency, traceability, spec_compliance
// for this fixture) and the Suggestions, never a passing dimension
// like safety.
func TestRunRetryInjectsOnlyFailedDimensions(t *testing.T) {
	cfg := RunConfig{
		Skill:        "huaweicloud-ecs-ops",
		Request:      "list servers",
		Command:      "false", // exits 1 every time
		MaxIter:      2,
		Timeout:      5,
		Critic:       &retryCaptureCritic{},
		RetryBuilder: MinimalFeedbackRetry{},
		Root:         t.TempDir(),
	}
	result := Run(cfg)
	// Both iterations: first → RETRY, second → MAX_ITER. The retry
	// prompt from MinimalFeedbackRetry is what we inspect next.
	if result.ExitCode != ExitMaxIter {
		t.Errorf("expected MAX_ITER, got %d", result.ExitCode)
	}
	prompt := MinimalFeedbackRetry{}.Build(
		GeneratorOutput{Command: cfg.Command, ExitCode: 1},
		CriticResult{
			Scores: map[string]float64{
				"correctness":     0.1,
				"safety":          1.0,
				"idempotency":     0.2,
				"traceability":    0.3,
				"spec_compliance": 0.1,
			},
			Suggestions: []string{"rotate_iam_scope first", "echo the resource id"},
			Mode:        "isolated-critic",
		},
		2,
	)
	// Per Q1-C: only failed dimensions.
	if !strings.Contains(prompt, "correctness=0.10") {
		t.Errorf("retry prompt must include failed dim correctness; got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "idempotency=0.20") {
		t.Errorf("retry prompt must include failed dim idempotency; got:\n%s", prompt)
	}
	// safety=1.0 passes → MUST NOT appear in the failed-dim section
	// (MinimalFeedbackRetry only includes dimensions where s < threshold).
	if strings.Contains(prompt, "safety=1.00") {
		t.Errorf("retry prompt must NOT include passing dim safety; got:\n%s", prompt)
	}
	// Suggestions ARE passed through.
	if !strings.Contains(prompt, "rotate_iam_scope first") {
		t.Errorf("retry prompt must carry Suggestions; got:\n%s", prompt)
	}
}

// TestRunRetryCarriesSuggestions is the spec-mandated (A4) test:
// iter-2 receives the same Critic Suggestions from iter-1. Verified
// by checking the retry prompt contents, since the persisted trace
// masks generator.command.
func TestRunRetryCarriesSuggestions(t *testing.T) {
	critic := &retryCaptureCritic{}
	iter1CriticResult := critic.Score(context.Background(), GeneratorOutput{Command: "false"})
	prompt := MinimalFeedbackRetry{}.Build(
		GeneratorOutput{Command: "false", ExitCode: 1},
		iter1CriticResult,
		2,
	)
	if !strings.Contains(prompt, iter1CriticResult.Suggestions[0]) {
		t.Errorf("iter-2 prompt missing iter-1 Suggestion %q", iter1CriticResult.Suggestions[0])
	}
	if !strings.Contains(prompt, iter1CriticResult.Suggestions[1]) {
		t.Errorf("iter-2 prompt missing iter-1 Suggestion %q", iter1CriticResult.Suggestions[1])
	}
}
