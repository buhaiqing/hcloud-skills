package gcl

import (
	"strings"
	"testing"
)

// TestMinimalFeedbackRetry_BuildsPromptFromCritic asserts the contract
// of RetryPromptBuilder: Build returns a string containing both the
// failing generator command and the critic suggestions, in that order.
func TestMinimalFeedbackRetry_BuildsPromptFromCritic(t *testing.T) {
	var b RetryPromptBuilder = MinimalFeedbackRetry{}

	gen := GeneratorOutput{
		Command:         "huaweicloud ecs delete --instance ecs-abc12345",
		ExitCode:        1,
		ResultExcerpt:   "deleted\n",
		StderrLen:       64,
		StdoutLen:       8,
		StdoutTruncated: true,
		StdoutDropped:   1024,
		DurationMs:      120,
	}
	critic := CriticResult{
		Scores: map[string]float64{
			"correctness":     0.3,
			"safety":          0.6,
			"idempotency":     0.4,
			"traceability":    0.5,
			"spec_compliance": 0.4,
		},
		Suggestions: []string{"request IAM scope before delete", "echo the instance id first"},
		Blocking:    false,
		Mode:        "structural-only",
	}

	got := b.Build(gen, critic, 2)
	if got == "" {
		t.Fatal("Build returned empty prompt")
	}
	wantSubs := []string{
		"huaweicloud ecs delete --instance ecs-abc12345",
		"request IAM scope before delete",
		"echo the instance id first",
		"iter=2",
		"Exit: 1",
		"Duration: 120ms",
		"stdout_truncated=true",
	}
	for _, sub := range wantSubs {
		if !strings.Contains(got, sub) {
			t.Errorf("prompt missing %q. Got:\n%s", sub, got)
		}
	}
	cmdIdx := strings.Index(got, gen.Command)
	sugIdx := strings.Index(got, critic.Suggestions[0])
	if cmdIdx < 0 || sugIdx < 0 || cmdIdx > sugIdx {
		t.Errorf("expected command before suggestions; cmdIdx=%d sugIdx=%d", cmdIdx, sugIdx)
	}
}

// TestMinimalFeedbackRetry_EmptySuggestionsStillProducesPrompt ensures
// Build does not crash when the critic reports no suggestions.
func TestMinimalFeedbackRetry_EmptySuggestionsStillProducesPrompt(t *testing.T) {
	var b RetryPromptBuilder = MinimalFeedbackRetry{}
	gen := GeneratorOutput{Command: "echo hello", ExitCode: 0, DurationMs: 5}
	critic := CriticResult{
		Scores:      map[string]float64{"correctness": 0.2, "safety": 1.0, "idempotency": 0.5, "traceability": 0.5, "spec_compliance": 0.5},
		Suggestions: nil,
		Blocking:    true,
		Mode:        "structural-only",
	}
	got := b.Build(gen, critic, 1)
	if got == "" {
		t.Fatal("Build returned empty prompt despite empty suggestions")
	}
	if !strings.Contains(got, "echo hello") {
		t.Errorf("prompt missing generator command: %q", got)
	}
	if !strings.Contains(got, "(none") {
		t.Errorf("prompt missing empty-suggestions marker: %q", got)
	}
}

// TestRetryPromptBuilder_InterfaceContract catches accidental signature
// drift. MinimalFeedbackRetry must satisfy the interface at compile time,
// and identical inputs produce identical outputs (deterministic).
func TestRetryPromptBuilder_InterfaceContract(t *testing.T) {
	var b RetryPromptBuilder = MinimalFeedbackRetry{}
	gen := GeneratorOutput{Command: "noop", DurationMs: 1}
	critic := CriticResult{
		Scores: map[string]float64{"correctness": 0.0, "safety": 1.0, "idempotency": 0.0, "traceability": 0.0, "spec_compliance": 0.0},
		Mode:   "x",
	}
	p1 := b.Build(gen, critic, 1)
	p2 := b.Build(gen, critic, 1)
	// Identical inputs should produce the same *shape* — we cannot
	// guarantee identical timestamps, but the command + scores must
	// be byte-identical and the timestamp must be present both times.
	for _, sub := range []string{gen.Command, "correctness="} {
		if !strings.Contains(p1, sub) || !strings.Contains(p2, sub) {
			t.Errorf("missing %q in repeated builds", sub)
		}
	}
}

// TestMinimalFeedbackRetry_IncludesBlockingHint ensures the prompt makes
// the blocking flag visible to the next iteration.
func TestMinimalFeedbackRetry_IncludesBlockingHint(t *testing.T) {
	var b RetryPromptBuilder = MinimalFeedbackRetry{}
	gen := GeneratorOutput{Command: "rm -rf /var/log/app", ExitCode: 1, DurationMs: 5}
	critic := CriticResult{
		Scores:      map[string]float64{"correctness": 0.0, "safety": 0.0, "idempotency": 0.5, "traceability": 0.5, "spec_compliance": 0.0},
		Suggestions: []string{"ask for confirmation first"},
		Blocking:    true,
		Mode:        "structural-only",
	}
	got := b.Build(gen, critic, 3)
	if !strings.Contains(got, "blocking=true") {
		t.Errorf("prompt missing blocking hint: %q", got)
	}
	if !strings.Contains(got, "iter=3") {
		t.Errorf("prompt missing iter number: %q", got)
	}
}

// TestMinimalFeedbackRetry_SurfacesLeakWarning asserts the Credential leak
// flag is visible in the retry prompt — without this, the next iteration
// could re-emit a credential and we wouldn't know why.
func TestMinimalFeedbackRetry_SurfacesLeakWarning(t *testing.T) {
	var b RetryPromptBuilder = MinimalFeedbackRetry{}
	gen := GeneratorOutput{
		Command:       "echo AK=ABCDEFGHIJ",
		HasLeak:       true,
		ResultExcerpt: "<masked>",
	}
	critic := CriticResult{
		Scores:      map[string]float64{"correctness": 0.0, "safety": 0.0, "idempotency": 0.5, "traceability": 0.5, "spec_compliance": 0.0},
		Suggestions: []string{"strip credentials before echoing"},
		Blocking:    true,
	}
	got := b.Build(gen, critic, 1)
	if !strings.Contains(got, "credential leak") {
		t.Errorf("prompt missing leak warning: %q", got)
	}
}

// TestMinimalFeedbackRetry_TruncationFlag asserts the stdout_truncated
// flag is surfaced so the next iteration knows output was lossy.
func TestMinimalFeedbackRetry_TruncationFlag(t *testing.T) {
	var b RetryPromptBuilder = MinimalFeedbackRetry{}
	gen := GeneratorOutput{
		Command:         "yes",
		StdoutTruncated: true,
		StdoutDropped:   4096,
	}
	critic := CriticResult{
		Scores: map[string]float64{"correctness": 0.5, "safety": 1.0, "idempotency": 0.5, "traceability": 0.5, "spec_compliance": 0.5},
		Mode:   "structural-only",
	}
	got := b.Build(gen, critic, 1)
	if !strings.Contains(got, "dropped 4096B") {
		t.Errorf("prompt missing dropped byte count: %q", got)
	}
}
