package gcl

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"time"
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
	in, err := cmd.StdinPipe()
	if err != nil {
		defaultResult.Mode = "stdin-pipe-error"
		return defaultResult
	}
	defer in.Close()
	out, err := cmd.Output()
	if err != nil {
		defaultResult.Mode = "exec-error"
		return defaultResult
	}

	var wire ExternalCriticResult
	if err := json.Unmarshal(out, &wire); err != nil {
		defaultResult.Mode = "decode-error"
		return defaultResult
	}
	if wire.Scores == nil {
		defaultResult.Mode = "decode-error: missing scores"
		return defaultResult
	}

	result := CriticResult{
		Scores:      wire.Scores,
		Suggestions: wire.Suggestions,
		Blocking:    wire.Blocking,
		Mode:        wire.Mode,
	}
	if result.Suggestions == nil {
		result.Suggestions = []string{}
	}
	return result
}
