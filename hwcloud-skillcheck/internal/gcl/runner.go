// Package gcl provides the Generator-Critic-Loop runtime components for
// hwcloud-skillcheck: sanitizer (safety_class enum + resource ID masking) and
// runner (GCL loop orchestration).
//
// This file implements the L1-B runner layer, ported from scripts/gcl_runner.py
// (cmd_run, decide, structural_critic, run_command, persist_trace,
// extract_failure_pattern, RUBRIC_THRESHOLDS, SKILL_MAX_ITER).
package gcl

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Exit codes (UNIX conventions).
const (
	ExitOK      = 0   // PASS
	ExitMaxIter = 1   // MAX_ITER: loop exhausted
	ExitUsage   = 2   // usage / internal error
	ExitSafety  = 3   // SAFETY_FAIL: credential leak or safety violation
	ExitTimeout = 124 // TIMEOUT: command exceeded timeout
)

// RUBRIC_THRESHOLDS are the minimum passing scores for each quality dimension.
// Mirrors RUBRIC_THRESHOLDS in gcl_runner.py.
var RUBRIC_THRESHOLDS = map[string]float64{
	"correctness":     0.5,
	"safety":          1.0, // strict: any leak is a SAFETY_FAIL
	"idempotency":     0.5,
	"traceability":    0.5,
	"spec_compliance": 0.5,
}

// SKILL_MAX_ITER is the default maximum GCL loop iterations per skill.
// Mirrors SKILL_MAX_ITER in gcl_runner.py.
var SKILL_MAX_ITER = map[string]int{
	"huaweicloud-ecs-ops":           2,
	"huaweicloud-iam-ops":           2,
	"huaweicloud-rds-ops":           2,
	"huaweicloud-gaussdb-ops":       2,
	"huaweicloud-dcs-ops":           2,
	"huaweicloud-dms-ops":           2,
	"huaweicloud-css-ops":           2,
	"huaweicloud-cce-ops":           2,
	"huaweicloud-cbr-ops":           2,
	"huaweicloud-vpc-ops":           2,
	"huaweicloud-obs-ops":           2,
	"huaweicloud-swr-ops":           2,
	"huaweicloud-functiongraph-ops": 2,
	"huaweicloud-waf-ops":           2,
	"huaweicloud-hss-ops":           2,
	"huaweicloud-elb-ops":           3,
	"huaweicloud-ces-ops":           3,
	"huaweicloud-lts-ops":           3,
	"huaweicloud-cts-ops":           3,
	"huaweicloud-billing-ops":       5,
	"huaweicloud-skill-generator":   3,
}

// ---- Types ---------------------------------------------------------------

// GeneratorOutput is the result of running a Generator command.
// Mirrors the "generator" dict in a GCL trace iteration.
type GeneratorOutput struct {
	Command         string `json:"command"`
	ExitCode        int    `json:"exit_code"`
	ResultExcerpt   string `json:"result_excerpt"` // masked, max 2000 chars
	StdoutLen       int    `json:"stdout_len"`
	StderrLen       int    `json:"stderr_len"`
	StdoutDropped   int64  `json:"stdout_dropped_bytes,omitempty"` // bytes past the capture cap, dropped
	StderrDropped   int64  `json:"stderr_dropped_bytes,omitempty"`
	StdoutTruncated bool   `json:"stdout_truncated,omitempty"`
	StderrTruncated bool   `json:"stderr_truncated,omitempty"`
	DurationMs      int    `json:"duration_ms"`
	HasLeak         bool   `json:"has_leak"` // true if raw output contained a credential leak (before masking)
}

// maxCaptureBytes caps the per-stream capture buffer in runCommand. Past
// this size, runCommand stops appending to the buffer (recording the dropped
// byte count on GeneratorOutput) and the downstream MaskSecrets scan sees
// only the head — usually where leaks and failure messages live.
const maxCaptureBytes = 1 << 20 // 1 MiB

// cappedWriter is an io.Writer that forwards into a bytes.Buffer up to a
// byte cap, silently drops the rest, and reports whether truncation happened.
// Used by runCommand to keep generator stdout/stderr bounded.
type cappedWriter struct {
	buf          bytes.Buffer
	cap          int
	truncated    bool
	droppedBytes int64
}

func (w *cappedWriter) Write(p []byte) (int, error) {
	remaining := w.cap - w.buf.Len()
	switch {
	case remaining > 0:
		n := len(p)
		if n > remaining {
			n = remaining
		}
		w.buf.Write(p[:n])
		if len(p) > remaining {
			w.truncated = true
			w.droppedBytes += int64(len(p) - remaining)
		}
	case w.buf.Len() >= w.cap:
		// First dropped write — flip the flag once.
		w.truncated = true
		w.droppedBytes += int64(len(p))
	}
	// We always claim the whole slice was accepted so the cmd side
	// does not see a short-write error and abort on overflow.
	return len(p), nil
}

func (w *cappedWriter) String() string { return w.buf.String() }

// CriticResult holds the Critic's quality assessment of a Generator output.
type CriticResult struct {
	Scores      map[string]float64 `json:"scores"`
	Suggestions []string           `json:"suggestions"`
	Blocking    bool               `json:"blocking"`
	Mode        string             `json:"mode,omitempty"`  // e.g. "structural-only"
	Model       string             `json:"model,omitempty"` // LLM model used for scoring (optional; "unknown" if unavailable)
}

// GCLTrace is the full record of a GCL loop execution.
// Mirrors the trace schema in gcl_runner.py.
type GCLTrace struct {
	TraceSchemaVersion string           `json:"trace_schema_version"`
	Skill              string           `json:"skill"`
	Model              string           `json:"model,omitempty"` // LLM model (e.g. "anthropic/claude-3-5-sonnet"); "unknown" if unavailable
	Request            string           `json:"request"`
	OperationIntent    map[string]any   `json:"operation_intent,omitempty"`
	RubricVersion      string           `json:"rubric_version"`
	MaskedFields       []string         `json:"masked_fields"`
	DurationMs         int              `json:"duration_ms"`
	TokenUsage         map[string]any   `json:"token_usage,omitempty"`
	ResourceContext    map[string]any   `json:"resource_context,omitempty"`
	OpsEfficiency      *OpsEfficiency   `json:"ops_efficiency,omitempty"`
	CostAttribution    *CostAttribution `json:"cost_attribution,omitempty"`
	Iterations         []Iteration      `json:"iterations"`
	Final              *FinalResult     `json:"final,omitempty"`
}

// Iteration records one pass through the GCL loop.
type Iteration struct {
	Iter      int             `json:"iter"`
	Generator GeneratorOutput `json:"generator"`
	Critic    CriticResult    `json:"critic"`
	Decision  string          `json:"decision"`
}

// FinalResult describes the terminal state of a GCL loop.
type FinalResult struct {
	Status         string          `json:"status"` // PASS, SAFETY_FAIL, MAX_ITER
	Iter           int             `json:"iter"`
	Output         string          `json:"output,omitempty"`
	FailurePattern *FailurePattern `json:"failure_pattern,omitempty"`
	Unresolved     []string        `json:"unresolved,omitempty"`
}

// FailurePattern categorizes a recurring failure for事后 analysis.
type FailurePattern struct {
	Category string `json:"category"`
	Skill    string `json:"skill"`
	Command  string `json:"command,omitempty"`
	Error    string `json:"error"`
	Fix      string `json:"fix"`
	Count    int    `json:"count"`
	Reusable bool   `json:"reusable"`
}

// RunConfig configures a single GCL Run.
// Root is the repository root (defaults to the hwcloud-skillcheck module root).
type RunConfig struct {
	Skill           string // skill id, e.g. "huaweicloud-ecs-ops"
	Request         string // sanitized user request
	Command         string // shell command for the Generator
	OperationIntent string // optional JSON operation intent
	MaxIter         int    // maximum loop iterations (0 = use SKILL_MAX_ITER default)
	Timeout         int    // command timeout in seconds (default 120)
	Critic          Critic // optional Critic override; nil falls back to StructuralCriticAdapter
	Root            string // repository root for audit-results/
	Model           string // LLM model for the Generator (optional; "unknown" if unavailable)
}

// RunResult is the output of a GCL Run.
type RunResult struct {
	ExitCode  int    // 0=PASS, 1=MAX_ITER, 2=usage, 3=SAFETY_FAIL, 124=timeout
	TracePath string // absolute path to the persisted trace JSON
}

// ---- Public API ----------------------------------------------------------

// Run executes the GCL loop: Generator → Critic → Orchestrator.
// It returns when a PASS or SAFETY_FAIL decision is reached, or MAX_ITER is exhausted.
//
// Exit codes:
//   - 0: PASS
//   - 1: MAX_ITER: loop exhausted without PASS
//   - 2: usage / internal error (e.g. invalid operation_intent)
//   - 3: SAFETY_FAIL: credential leak or safety violation detected
//
// Note: cfg.Root must be set to the hwcloud-skillcheck module root directory
// (where audit-results/ will be created). Tests should pass t.TempDir().
//
// If Root is empty, PersistTrace writes to process cwd — which is
// correct for tests but incorrect in production.
func Run(cfg RunConfig) RunResult {
	startTime := time.Now()

	// Resolve Critic once. Callers that leave cfg.Critic nil get
	// the in-process Structural critic (the historical default).
	critic := cfg.Critic
	if critic == nil {
		critic = StructuralCriticAdapter{}
	}

	// Determine max iterations.
	maxIter := cfg.MaxIter
	if maxIter == 0 {
		maxIter = 2 // default fallback
		if skillDefault, ok := SKILL_MAX_ITER[cfg.Skill]; ok {
			maxIter = skillDefault
		}
	}

	// Sanitize operation intent.
	var opIntent map[string]any
	if cfg.OperationIntent != "" {
		var err error
		opIntent, err = SanitizeOperationIntent(cfg.OperationIntent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			return RunResult{ExitCode: ExitUsage}
		}
	}

	model := cfg.Model
	if model == "" {
		model = "unknown"
	}
	trace := GCLTrace{
		TraceSchemaVersion: "v1",
		Skill:              cfg.Skill,
		Model:              model,
		Request:            cfg.Request,
		OperationIntent:    opIntent,
		RubricVersion:      "v1",
		MaskedFields:       []string{"request", "operation_intent", "generator.command", "generator.result_excerpt"},
		Iterations:         []Iteration{},
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120
	}

	for iteration := 1; iteration <= maxIter; iteration++ {
		generatorStart := time.Now()
		generator := runCommand(cfg.Command, timeout)
		generator.DurationMs = int(time.Since(generatorStart).Milliseconds())

		// Critic evaluates the Generator output. Default is the
		// in-process Structural critic; callers may inject a custom
		// Critic (e.g. an ExternalCritic subprocess) via RunConfig.
		scored := critic.Score(context.Background(), generator)
		if scored.Scores == nil {
			scored.Scores = map[string]float64{}
		}
		criticResult := CriticResult{
			Scores:      scored.Scores,
			Suggestions: scored.Suggestions,
			Blocking:    scored.Blocking,
			Mode:        scored.Mode,
		}

		decision := Decide(criticResult.Scores)

		trace.Iterations = append(trace.Iterations, Iteration{
			Iter:      iteration,
			Generator: generator,
			Critic:    criticResult,
			Decision:  decision,
		})

		switch decision {
		case "SAFETY_FAIL":
			trace.Final = &FinalResult{
				Status:         "SAFETY_FAIL",
				Iter:           iteration,
				Output:         "",
				FailurePattern: extractFailurePattern(cfg.Skill, cfg.Command, generator, criticResult),
			}
			trace.DurationMs = int(time.Since(startTime).Milliseconds())
			FinalizeFinopsAiops(&trace)
			path, err := PersistTrace(&trace, cfg.Root)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: PersistTrace failed: %v\n", err)
			}
			fmt.Fprintf(os.Stderr, "SAFETY_FAIL — trace: %s\n", path)
			return RunResult{ExitCode: ExitSafety, TracePath: path}

		case "PASS":
			trace.Final = &FinalResult{
				Status: "PASS",
				Iter:   iteration,
				Output: generator.ResultExcerpt,
			}
			trace.DurationMs = int(time.Since(startTime).Milliseconds())
			FinalizeFinopsAiops(&trace)
			path, err := PersistTrace(&trace, cfg.Root)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: PersistTrace failed: %v\n", err)
			}
			fmt.Printf("PASS (iter %d) — trace: %s\n", iteration, path)
			return RunResult{ExitCode: ExitOK, TracePath: path}
		}

	}

	// MAX_ITER exhausted.
	last := trace.Iterations[len(trace.Iterations)-1]
	var unresolved []string
	for dim, threshold := range RUBRIC_THRESHOLDS {
		if last.Critic.Scores[dim] < threshold {
			unresolved = append(unresolved, dim)
		}
	}
	trace.DurationMs = int(time.Since(startTime).Milliseconds())
	FinalizeFinopsAiops(&trace)
	path, err := PersistTrace(&trace, cfg.Root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: PersistTrace failed: %v\n", err)
	}
	fmt.Fprintf(os.Stderr, "MAX_ITER — trace: %s\n", path)
	return RunResult{ExitCode: ExitMaxIter, TracePath: path}
}

// Decide evaluates critic scores against rubric thresholds and returns a GCL
// decision string: PASS | RETRY | SAFETY_FAIL.
//
// Mirrors decide() in gcl_runner.py.
func Decide(scores map[string]float64) string {
	if scores["safety"] < 1.0 {
		return "SAFETY_FAIL"
	}
	for dim, threshold := range RUBRIC_THRESHOLDS {
		if scores[dim] < threshold {
			return "RETRY"
		}
	}
	return "PASS"
}

// StructuralCritic performs a rule-based quality assessment of a Generator output.
// This is a lightweight critic intended for CI/local smoke tests only —
// production GCL must use an externally supplied isolated Critic.
//
// Mirrors structural_critic() in gcl_runner.py.
func StructuralCritic(gen GeneratorOutput) CriticResult {
	scores := map[string]float64{}
	suggestions := []string{}

	exitCode := gen.ExitCode
	scores["correctness"] = 1.0
	if exitCode != 0 {
		scores["correctness"] = 0.0
		suggestions = append(suggestions, fmt.Sprintf("Generator exit_code=%d; fix command, credentials, or endpoint", exitCode))
	}

	leak := gen.HasLeak || hasCredentialLeak(gen.Command)
	scores["safety"] = 1.0
	if leak {
		scores["safety"] = 0.0
		suggestions = append(suggestions, "Credential leak in trace — mask HW_SECRET_ACCESS_KEY and re-run")
	}

	scores["idempotency"] = 0.5
	scores["traceability"] = 0.5
	if gen.Command != "" && gen.ResultExcerpt != "" {
		scores["traceability"] = 1.0
	}
	if gen.ResultExcerpt == "" {
		suggestions = append(suggestions, "Empty generator output — capture stdout/stderr in trace")
	}

	scores["spec_compliance"] = 1.0
	if exitCode == 0 && !strings.Contains(gen.Command, "hcloud") && !strings.Contains(strings.ToLower(gen.Command), "go run") {
		scores["spec_compliance"] = 0.5
	}

	// Limit suggestions to 3.
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return CriticResult{
		Scores:      scores,
		Suggestions: suggestions,
		Blocking:    scores["safety"] == 0.0 || scores["correctness"] == 0.0,
		Mode:        "structural-only",
		Model:       "structural-only",
	}
}

// PersistTrace writes trace to <root>/audit-results/gcl-trace-<timestamp>.json
// and returns the path. Directory is created with mode 0700 (owner-only).
//
// Mirrors persist_trace() in gcl_runner.py.
func PersistTrace(trace *GCLTrace, root string) (string, error) {
	outDir := filepath.Join(root, "audit-results")
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return "", fmt.Errorf("PersistTrace mkdir: %w", err)
	}

	ts := time.Now().UTC().Format("20060102-150405")
	suffix := uniqueShortID()
	filename := fmt.Sprintf("gcl-trace-%s-%s.json", ts, suffix)
	path := filepath.Join(outDir, filename)

	data, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return "", fmt.Errorf("PersistTrace json: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("PersistTrace write: %w", err)
	}
	return path, nil
}

// uniqueShortID returns 8 hex chars from crypto/rand; the Python port
// appended uuid.uuid4().hex[:8] to the timestamp to avoid filename collisions
// when two traces finish in the same second.
func uniqueShortID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000"
	}
	return hex.EncodeToString(b[:])
}

// ---- Internal helpers ----------------------------------------------------

// secretPatterns matches embedded credential strings in command output.
// Mirrors SECRET_PATTERNS in gcl_runner.py.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)HW_SECRET_ACCESS_KEY\s*=\s*[^\s"']+`),
	regexp.MustCompile(`(?i)SECRET_ACCESS_KEY\s*=\s*[^\s"']+`),
	regexp.MustCompile(`(?i)SecretAccessKey\s*[=:]\s*[^\s"']+`),
	regexp.MustCompile(`(?i)SK\s*[=:]\s*[A-Za-z0-9/+]{20,}`),
}

// hasCredentialLeak reports true if text contains an unmasked credential pattern.
// Mirrors has_credential_leak() in gcl_runner.py.
func hasCredentialLeak(text string) bool {
	if strings.Contains(text, "<masked>") {
		return false
	}
	for _, pat := range secretPatterns {
		if pat.MatchString(text) {
			return true
		}
	}
	return false
}

// runCommand executes command with the given timeout (seconds) and returns
// a masked GeneratorOutput. On timeout, exit code is -1 and ResultExcerpt
// contains a TIMEOUT message.
//
// Mirrors run_command() in gcl_runner.py.
func runCommand(command string, timeoutSecs int) GeneratorOutput {
	maskedCmd := MaskSecrets([]byte(command))

	// Bound the captured stdout / stderr. A noisy generator (a hung
	// process looping, a verbose `hcloud` command, a misbehaving
	// child shell) can otherwise pin ~4x the actual output size in
	// memory through stdout+stderr buffers plus the
	// stdout.String()+stderr.String()+[]byte(combined) copies
	// below. 1 MiB per stream is comfortably larger than any
	// realistic hcloud output and gives leak/mask scanning the head
	// of the stream; bytes past the cap are discarded and flagged
	// on the returned GeneratorOutput for downstream awareness.
	const maxCaptureBytes = 1 << 20
	stdout := cappedWriter{cap: maxCaptureBytes}
	stderr := cappedWriter{cap: maxCaptureBytes}

	timeout := time.Duration(timeoutSecs) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Cancel runs INSIDE the goroutine that owns cmd, so cmd.Process is
	// read after Start has already written it — no race against cmd.Process
	// being set by Start.
	cmd.Cancel = func() error {
		return cmd.Process.Kill()
	}
	// WaitDelay forces the process to be killed if it ignores the cancel
	// signal. 100 ms is enough for SIGKILL to land.
	cmd.WaitDelay = 100 * time.Millisecond

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	// time.NewTimer + explicit Stop is preferable to time.After in a one-shot
	// select: the After channel is unstopped and persists until the configured
	// deadline, minorly wasteful across many iterations.
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var runErr error
	select {
	case runErr = <-done:
		// completed normally
	case <-timer.C:
		// Trigger Cancel + WaitDelay inside cmd's lifecycle. We don't
		// touch cmd.Process here — that's owned by the goroutine.
		cancel()
		<-done
		return GeneratorOutput{
			Command:       maskedCmd,
			ExitCode:      ExitTimeout, // 124 — UNIX convention for timeout
			ResultExcerpt: fmt.Sprintf("TIMEOUT after %ds", timeoutSecs),
			StdoutLen:     0,
			StderrLen:     0,
			HasLeak:       false,
		}
	}

	stdoutStr := stdout.String()
	stderrStr := stderr.String()
	// combined is bounded by maxCaptureBytes per stream, so the concat
	// costs at most 2 MiB instead of leaking the full stream size.
	combined := stdoutStr + stderrStr

	// Check for credential leaks BEFORE masking.
	leak := hasCredentialLeak(combined) || hasCredentialLeak(command)

	// Apply secret masking.
	masked := MaskSecrets([]byte(combined))
	excerpt := masked
	if len(excerpt) > 2000 {
		excerpt = masked[:2000] + "..."
	}

	exitCode := 0
	if runErr != nil {
		if exitError, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return GeneratorOutput{
		Command:         maskedCmd,
		ExitCode:        exitCode,
		ResultExcerpt:   excerpt,
		StdoutLen:       len(stdoutStr),
		StderrLen:       len(stderrStr),
		StdoutDropped:   stdout.droppedBytes,
		StderrDropped:   stderr.droppedBytes,
		StdoutTruncated: stdout.truncated,
		StderrTruncated: stderr.truncated,
		HasLeak:         leak,
	}
}

// failureSignatures maps pattern categories to regexps for extractFailurePattern.
// Mirrors _FAILURE_SIGNATURES in gcl_runner.py.
var failureSignatures = []struct {
	category string
	re       *regexp.Regexp
}{
	{"cli_parameter", regexp.MustCompile(`(?i)InvalidParameter|MissingParameter|APIGW\.|APIG\.`)},
	{"runtime", regexp.MustCompile(`(?i)TIMEOUT|RequestLimitExceeded|InternalError|ConnectionError|Throttling`)},
	{"cross_skill", regexp.MustCompile(`(?i)delegate-to|not found in target skill|cross-skill`)},
	{"token_efficiency", regexp.MustCompile(`(?i)token budget|exceeds.*token|too long|truncated`)},
	{"skill_generation", regexp.MustCompile(`(?i)frontmatter missing|missing rubric|broken link`)},
}

// extractFailurePattern identifies a known failure category from the GCL output.
// Returns nil if no known pattern matches.
//
// Mirrors extract_failure_pattern() in gcl_runner.py.
func extractFailurePattern(skill, command string, gen GeneratorOutput, critic CriticResult) *FailurePattern {
	corpus := command + "\n" + gen.ResultExcerpt + "\n" + strings.Join(critic.Suggestions, "\n")
	for _, fs := range failureSignatures {
		if !fs.re.MatchString(corpus) {
			continue
		}
		fix := "Investigate failure pattern and add fix"
		if len(critic.Suggestions) > 0 {
			fix = critic.Suggestions[0]
		}
		if len(fix) > 200 {
			fix = fix[:200]
		}
		cmd := command
		if len(cmd) > 200 {
			cmd = cmd[:200]
		}
		return &FailurePattern{
			Category: fs.category,
			Skill:    skill,
			Command:  MaskSecrets([]byte(cmd)),
			Error:    fs.re.FindString(corpus),
			Fix:      fix,
			Count:    1,
			Reusable: fs.category == "cli_parameter" || fs.category == "runtime",
		}
	}
	return nil
}
