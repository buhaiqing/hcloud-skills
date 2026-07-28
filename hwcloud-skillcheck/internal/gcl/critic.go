package gcl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/embed"
	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/schema"
)

// Critic is the interface a GCL loop delegates scoring to.
//
// StructuralCritic is one implementation — it lives in runner.go and is
// the default. External critics (processes, sidecars, anything else that
// can take GeneratorOutput and emit CriticResult over stdin/stdout JSON)
// implement the same contract by wrapping themselves in ExternalCritic,
// which is constructed by NewExternalCritic.
//
// Contract with an ExternalCritic subprocess:
//   - Input: GeneratorOutput JSON, written to the Critic's stdin.
//   - Output: CriticResult JSON on stdout. See ExternalCriticResult.
//   - Subprocess exits 0 on success, non-zero otherwise. Non-zero exit
//     (or stderr noise) becomes an empty CriticResult passed to Decide,
//     which the rubric thresholds all reject (→ RETRY/HALT path).
type Critic interface {
	Score(ctx context.Context, gen GeneratorOutput) CriticResult
}

// StructuralCriticAdapter wraps the in-package StructuralCritic function
// to satisfy the Critic interface. Runner uses this by default.
type StructuralCriticAdapter struct{}

func (StructuralCriticAdapter) Score(_ context.Context, gen GeneratorOutput) CriticResult {
	return StructuralCritic(gen)
}

// ExternalCriticResult is the wire format expected from a Critic
// subprocess. The fields here are the only ones consumed by Run;
// any extras are ignored.
type ExternalCriticResult struct {
	Scores      map[string]float64 `json:"scores"`
	Suggestions []string           `json:"suggestions"`
	Blocking    bool               `json:"blocking"`
	Mode        string             `json:"mode,omitempty"`
}

// ExternalCritic runs `cmd <args>` as a subprocess, pipes GeneratorOutput
// JSON to its stdin, and reads ExternalCriticResult JSON from stdout.
//
// The Critic process inherits the Generator's context (via the parent
// ctx passed in). The Runner-level timeout that fires on the
// Generator also cancels the Critic subprocess on SIGKILL — same
// wall-clock budget as the Generator command.
type ExternalCritic struct {
	Path string
	Args []string
}

// NewExternalCritic builds an ExternalCritic for the given executable
// path + args. Path must be an absolute path or resolvable via $PATH.
func NewExternalCritic(path string, args ...string) *ExternalCritic {
	return &ExternalCritic{Path: path, Args: args}
}

// Score implements Critic.
func (e ExternalCritic) Score(ctx context.Context, gen GeneratorOutput) CriticResult {
	defaultResult := CriticResult{
		Scores:      map[string]float64{},
		Suggestions: []string{"external critic failed to produce a CriticResult"},
		Blocking:    false,
		Mode:        "unconfigured",
	}
	if e.Path == "" {
		return defaultResult
	}

	// 60s worst case for a Critic — matches the per-iteration budget
	// the Runner applies to the Generator command. The Critic should
	// be quick; this is a circuit breaker.
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cctx, e.Path, e.Args...)
	cmd.Stderr = os.Stderr
	// Build a stdin pipe, write the GeneratorOutput JSON to it, then close
	// it so the child sees EOF on stdin. Without the close, a child that
	// reads stdin (the contract for an ExternalCritic) would block forever
	// and the 60s ctx would fire, leaving empty stdout → decode-error.
	in, err := cmd.StdinPipe()
	if err != nil {
		defaultResult.Mode = "stdin-pipe-error"
		return defaultResult
	}
	genBytes, mErr := json.Marshal(gen)
	if mErr != nil {
		defaultResult.Mode = "marshal-error: " + mErr.Error()
		_ = in.Close()
		return defaultResult
	}
	if _, wErr := in.Write(genBytes); wErr != nil {
		defaultResult.Mode = "stdin-write-error: " + wErr.Error()
		_ = in.Close()
		return defaultResult
	}
	_ = in.Close()
	out, err := cmd.Output()
	if err != nil {
		// Context deadline exceeded or subprocess crashed — still return a result
		// so callers get a populated Mode rather than the unconfigured default.
		defaultResult.Mode = "subprocess-error: " + err.Error()
		return defaultResult
	}
	// Validate the wire payload against the critic_output schema BEFORE
	// surfacing it as a CriticResult. A validation failure means the
	// subprocess emitted something the runner can't safely consume —
	// we fall back to a defaultResult with Mode="schema-invalid: …" so
	// the rubric thresholds reject all scores (→ RETRY/HALT path).
	var wire ExternalCriticResult
	if err := json.Unmarshal(out, &wire); err != nil {
		defaultResult.Mode = "decode-error"
		return defaultResult
	}
	errs, err := schema.ValidateFile(out, embed.CriticOutputSchema)
	if err != nil {
		defaultResult.Mode = "schema-validator-error: " + err.Error()
		return defaultResult
	}
	if len(errs) > 0 {
		defaultResult.Mode = "schema-invalid: " + shortSchemaErr(errs)
		defaultResult.Suggestions = append(
			defaultResult.Suggestions[:0],
			"critic output failed schema validation: "+strings.Join(errs, "; "),
		)
		return defaultResult
	}
	if wire.Scores == nil {
		defaultResult.Mode = "decode-error: missing scores"
		return defaultResult
	}

	// Default Mode to "unconfigured" when the wire payload omits it. The
	// critic_output schema only requires `scores`; mode is optional. Callers
	// (e.g. TestExternalCritic_TimeoutContext) assert Mode is populated
	// even on partial / no-timeout paths, so we must never return Mode="".
	result := CriticResult{
		Scores:      wire.Scores,
		Suggestions: wire.Suggestions,
		Blocking:    wire.Blocking,
		Mode:        wire.Mode,
	}
	if result.Mode == "" {
		result.Mode = "unconfigured"
	}
	if result.Suggestions == nil {
		result.Suggestions = []string{}
	}
	return result
}

// shortSchemaErr joins the schema validator's errors into one short,
// human-readable mode label. We cap the output so the trace Mode field
// stays readable; full diagnostics live in Suggestions.
func shortSchemaErr(errs []string) string {
	if len(errs) == 0 {
		return "unknown"
	}
	msg := errs[0]
	for _, e := range errs[1:] {
		msg += "; " + e
	}
	if len(msg) > 240 {
		msg = msg[:240] + "…"
	}
	return fmt.Sprintf("%d issue(s): %s", len(errs), msg)
}
