// Package gcl — retry.go
//
// RetryPromptBuilder is the seam between the GCL loop's Critic verdict and
// the Generator's next attempt. The Runner (slice 2 will wire it) calls
// Builder.Build after a RETRY decision to assemble a new prompt that the
// LLM can use to fix the most recent failure.
//
// THIS FILE IS NOT WIRED INTO Run() YET — slice 2 owns the integration.
// Slice 3 owns the type + the minimal-implementation reference only.
package gcl

import (
	"fmt"
	"strings"
	"time"
)

// RetryPromptBuilder returns the prompt for the next Generator iteration,
// given the failed output and the Critic verdict that triggered RETRY.
//
// Implementations MUST be deterministic for the same inputs (the GCL
// loop depends on stable test fixtures and reproducible retries).
type RetryPromptBuilder interface {
	Build(gen GeneratorOutput, lastCritic CriticResult, iter int) string
}

// MinimalFeedbackRetry is the reference implementation: it concatenates
// the failing command + a few Critic suggestions + the iteration counter.
//
// The prompt is plain text, formatted for a human-readable CLI log but
// also readable as a system-prompt section by an LLM. The format is
// intentionally trivial — richer builders (e.g. a Jinja-style prompt
// template) are out of scope for the MVP and live behind the interface.
type MinimalFeedbackRetry struct{}

// Build assembles the retry prompt. Shape:
//
//	Retry iter=<n> blocking=<bool>
//	Skill-iteration started at <RFC3339>
//
//	Command: <gen.Command>
//	Exit: <gen.ExitCode>   Duration: <gen.DurationMs>ms
//	Result (masked, max 2000 chars):
//	  <gen.ResultExcerpt>
//
//	Stream sizes: stdout=<…>B stderr=<…>B
//	  stdout_truncated=<bool> (dropped <…>B)
//	  stderr_truncated=<bool> (dropped <…>B)
//	Credential leak detected (raw): <bool>
//
//	Critic mode=<mode>  Scores:
//	  correctness=<…> safety=<…> idempotency=<…> traceability=<…> spec_compliance=<…>
//
//	Suggestions:
//	  - <suggestion 1>
//	  - <suggestion 2>
//	  …
func (MinimalFeedbackRetry) Build(gen GeneratorOutput, lastCritic CriticResult, iter int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Retry iter=%d blocking=%t\n", iter, lastCritic.Blocking)
	fmt.Fprintf(&b, "Skill-iteration started at %s\n", time.Now().UTC().Format(time.RFC3339))

	fmt.Fprintf(&b, "\nCommand: %s\n", gen.Command)
	fmt.Fprintf(&b, "Exit: %d   Duration: %dms\n", gen.ExitCode, gen.DurationMs)

	if gen.ResultExcerpt != "" {
		fmt.Fprintf(&b, "\nResult (masked, max 2000 chars):\n%s\n", gen.ResultExcerpt)
	}

	fmt.Fprintf(&b, "\nStream sizes: stdout=%dB stderr=%dB\n", gen.StdoutLen, gen.StderrLen)
	fmt.Fprintf(&b, "  stdout_truncated=%t (dropped %dB)\n", gen.StdoutTruncated, gen.StdoutDropped)
	fmt.Fprintf(&b, "  stderr_truncated=%t (dropped %dB)\n", gen.StderrTruncated, gen.StderrDropped)
	if gen.HasLeak {
		fmt.Fprintf(&b, "WARNING: credential leak detected in raw output (pre-masking)\n")
	}

	fmt.Fprintf(&b, "\nCritic mode=%s  Failed dimensions (score below threshold):\n", fallback(lastCritic.Mode, "unknown"))
	wrote := 0
	for dim, threshold := range map[string]float64{
		"correctness":     0.5,
		"safety":          1.0,
		"idempotency":     0.5,
		"traceability":    0.5,
		"spec_compliance": 0.5,
	} {
		if s, ok := lastCritic.Scores[dim]; ok && s < threshold {
			fmt.Fprintf(&b, "  %s=%s (threshold %.2f)\n", dim, fmtScore(s), threshold)
			wrote++
		}
	}

	if len(lastCritic.Suggestions) > 0 {
		fmt.Fprintf(&b, "\nSuggestions:\n")
		for _, s := range lastCritic.Suggestions {
			fmt.Fprintf(&b, "  - %s\n", s)
		}
	} else {
		fmt.Fprintf(&b, "\nSuggestions: (none — fix based on exit code + result above)\n")
	}

	return b.String()
}

func fallback(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func fmtScore(v float64) string {
	if v == 0 {
		// ambiguous: 0 could be missing or actually 0.0; surface it as "0.00".
		return "0.00"
	}
	return fmt.Sprintf("%.2f", v)
}
