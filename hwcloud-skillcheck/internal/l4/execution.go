package l4

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/gcl"
)

// ExecutionResult captures the outcome of running one step.
type ExecutionResult struct {
	Step         int             `json:"step"`
	Command      string          `json:"command"`
	RBACDecision RBACDecision    `json:"rbac_decision"`
	GCLDecision  GCLDecisionBody `json:"gcl_decision"`
	Success      bool            `json:"success"`
	Error        string          `json:"error,omitempty"`
	StartedAt    string          `json:"started_at"`
	FinishedAt   string          `json:"finished_at"`
}

// Executor runs a candidate command and returns the result.
// StubExecutor returns a controlled outcome for tests.
// RealExecutor (in ADR-0010) wraps os/exec.CommandContext.
type Executor interface {
	Run(candidate string, timeout time.Duration) (exitCode int, stdout string, err error)
}

// StubExecutor returns a preconfigured outcome; used by E2E tests.
type StubExecutor struct {
	Outcomes []StubStep // index = step number in the plan
	mu       sync.Mutex
	cursor   int
}

// StubStep is one preconfigured executor outcome.
type StubStep struct {
	ExitCode int
	Stdout   string
	Err      error
}

// Run returns the next preconfigured outcome. After all outcomes are consumed
// it returns (0, "", nil) — used for the step after the last configured one.
func (s *StubExecutor) Run(candidate string, timeout time.Duration) (int, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cursor >= len(s.Outcomes) {
		return 0, "", nil
	}
	out := s.Outcomes[s.cursor]
	s.cursor++
	return out.ExitCode, out.Stdout, out.Err
}

// RealExecutor shells out via os/exec. Captures stdout/stderr (capped),
// returns exit code. Default timeout 60s; configurable. Captures capped at
// MaxBytes (default 1 MiB) per stream to prevent OOM on runaway output.
type RealExecutor struct {
	Env      []string      // os.Environ() by default; override for tests
	Timeout  time.Duration // per-step timeout default when caller passes 0
	MaxBytes int           // stdout/stderr capture cap (default 1 << 20)
}

// NewRealExecutor constructs a RealExecutor with production defaults.
func NewRealExecutor() *RealExecutor {
	return &RealExecutor{
		Env:      os.Environ(),
		Timeout:  60 * time.Second,
		MaxBytes: 1 << 20,
	}
}

// limitedBuffer is a bytes.Buffer wrapper that drops writes past its limit,
// matching the protection io.LimitWriter would give us. Added as a local
// helper so both stdout and stderr cap independently.
type limitedBuffer struct {
	buf bytes.Buffer
	max int64
	n   int64
}

func newLimitedBuffer(max int64) *limitedBuffer {
	return &limitedBuffer{max: max}
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := b.max - b.n
	if remaining <= 0 {
		return len(p), nil // drop, but pretend success so cmd.Run doesn't error
	}
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err := b.buf.Write(p)
	b.n += int64(n)
	return n, err
}

func (b *limitedBuffer) String() string { return b.buf.String() }

// Compile-time guard: limitedBuffer must implement io.Writer for exec.Cmd.
var _ io.Writer = (*limitedBuffer)(nil)

// Run executes candidate via `bash -c` to support multi-arg commands.
// Returns (exitCode, mergedStdoutAndStderr, err). err is non-nil for non-zero
// exits or timeouts. timeout=0 uses the receiver's Timeout.
func (r *RealExecutor) Run(candidate string, timeout time.Duration) (int, string, error) {
	if timeout <= 0 {
		timeout = r.Timeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", candidate)
	if r.Env != nil {
		cmd.Env = r.Env
	}

	max := r.MaxBytes
	if max <= 0 {
		max = 1 << 20
	}
	stdoutBuf := newLimitedBuffer(int64(max))
	stderrBuf := newLimitedBuffer(int64(max))
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	err := cmd.Run()
	output := stdoutBuf.String() + stderrBuf.String()

	// Distinguish timeout from a regular non-zero exit.
	if ctx.Err() == context.DeadlineExceeded {
		return 0, output, ctx.Err()
	}

	exitCode := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
		return exitCode, output, err
	}
	return exitCode, output, err
}

// RunExecutionLoop executes the planned steps with persistence, RBAC, GCL,
// and the supplied Executor. Pass nil for exec to default to NewRealExecutor()
// (which shells out via os/exec). Returns the task state after all steps.
//
// ADR-0010: GCL dry-run gate short-circuits to a SKIPPED result when the
// structural critic does not PASS / ACCEPT; otherwise the executor is run
// and its exit code determines Success.
func RunExecutionLoop(root string, task *TaskState, plan *ExecutionPlan, matched []MatchedSkill, exec Executor) *TaskState {
	if exec == nil {
		exec = NewRealExecutor()
	}

	// Build known skills map.
	knownSkills := map[string]bool{}
	for _, m := range matched {
		knownSkills[m.Skill] = true
	}

	// Pre-fetch failure patterns concurrently.
	skills := make([]string, 0, len(knownSkills))
	for s := range knownSkills {
		skills = append(skills, s)
	}
	patternCache := preFetchFailurePatterns(root, skills)

	// Main execution loop.
	for {
		// Check if all steps are done.
		if task.CurrentStep >= len(task.Steps) {
			CompleteTask(task)
			_ = PersistTask(root, task.ID, task)
			return task
		}

		// Get next step.
		step := task.Steps[task.CurrentStep]

		// Build command.
		short := step.SkillShort
		if short == "" {
			short = strings.ReplaceAll(strings.ReplaceAll(step.Skill, "huaweicloud-", ""), "-ops", "")
		}
		candidate := "hcloud " + short + " " + step.Action

		// Step 1: RBAC check.
		risk := step.Risk
		if risk == "" {
			risk = "medium"
		}
		trustLevel := "L2_established" // default
		score := 0.65
		rbacDec := CheckCommandPermission(candidate, trustLevel, score)

		// Check immutable constraints first.
		if !rbacDec.Allowed {
			// Record failed step and abort.
			result := StepResult{
				Step:         step.Step,
				Skill:        step.Skill,
				Command:      candidate,
				StartedAt:    NowISO(),
				FinishedAt:   NowISO(),
				Success:      false,
				Error:        rbacDec.Reason,
				RBACApproved: false,
				RBACReason:   rbacDec.Reason,
				GCLDecision:  "blocked_by_rbac",
			}
			task.Results = append(task.Results, result)
			FailTask(task, rbacDec.Reason)
			_ = PersistTask(root, task.ID, task)
			return task
		}

		// Step 2: GCL structural critic.
		genPayload := gcl.GeneratorOutput{
			Command:       candidate,
			ExitCode:      0,
			ResultExcerpt: "dry-run",
		}
		crit := gcl.StructuralCritic(genPayload)

		// Match pre-execution risk from failure patterns.
		var preRisk any
		if knownSkills[step.Skill] {
			if patterns, ok := patternCache[step.Skill]; ok && len(patterns) > 0 {
				preRisk = matchPreExecutionRisk(candidate, patterns)
			}
		}

		gclBody := GCLDecisionBody{
			Scores:           crit.Scores,
			Decision:         gcl.Decide(crit.Scores),
			PreExecutionRisk: preRisk,
		}

		// Check safety gate.
		if crit.Scores["safety"] == 0.0 {
			result := StepResult{
				Step:         step.Step,
				Skill:        step.Skill,
				Command:      candidate,
				StartedAt:    NowISO(),
				FinishedAt:   NowISO(),
				Success:      false,
				Error:        "safety check failed",
				RBACApproved: true,
				GCLDecision:  "SAFETY_FAIL",
				GCLScores:    crit.Scores,
			}
			task.Results = append(task.Results, result)
			FailTask(task, "safety check failed")
			_ = PersistTask(root, task.ID, task)
			return task
		}

		// Step 3: Dry-run gate (GCL structural critic).
		// If the gate denies the step, record SKIPPED and continue.
		if gclBody.Decision != "PASS" && gclBody.Decision != "ACCEPT" {
			task.Results = append(task.Results, StepResult{
				Step:         step.Step,
				Skill:        step.Skill,
				Command:      candidate,
				StartedAt:    NowISO(),
				FinishedAt:   NowISO(),
				Success:      false,
				Error:        "skipped: GCL=" + gclBody.Decision,
				RBACApproved: rbacDec.Allowed,
				RBACReason:   rbacDec.Reason,
				GCLDecision:  gclBody.Decision,
				GCLScores:    crit.Scores,
			})
			task.CurrentStep++
			_ = PersistTask(root, task.ID, task)
			continue
		}

		// Step 4: Real execution via os/exec.
		startedAt := NowISO()
		exitCode, output, execErr := exec.Run(candidate, 0)
		finishedAt := NowISO()
		var errStr string
		if execErr != nil {
			errStr = execErr.Error()
		}
		result := StepResult{
			Step:         step.Step,
			Skill:        step.Skill,
			Command:      candidate,
			StartedAt:    startedAt,
			FinishedAt:   finishedAt,
			ExitCode:     exitCode,
			Success:      execErr == nil && exitCode == 0,
			Output:       output,
			Error:        errStr,
			RBACApproved: rbacDec.Allowed,
			RBACReason:   rbacDec.Reason,
			GCLDecision:  gclBody.Decision,
			GCLScores:    crit.Scores,
		}
		task.Results = append(task.Results, result)
		task.CurrentStep++

		// Persist checkpoint after each step.
		_ = PersistTask(root, task.ID, task)
	}

	// Never reached.
}

// RunExecutionLoopWithHealing is the production call path. Self-healing is
// bypassed when mem is nil or p.IsZero() is true. exec may be nil — when so,
// it defaults to NewRealExecutor() (ADR-0010). Otherwise it consults
// OutcomeMemory pre-exec (skip-on-bad-history) and post-failure
// (retry-on-transient / escalate-on-permanent).
// AutofixFunc is an injectable autonomous-remediation hook. When provided,
// RunExecutionLoopWithHealing calls it after a step fails permanently
// (PostFailureHook escalates). The hook is bridged by the CLI layer to
// internal/learning (loads playbooks, renders, executes, records outcome), so
// internal/l4 stays import-cycle-free.
type AutofixFunc func(skill, command string) AutofixResult

func RunExecutionLoopWithHealing(root string, task *TaskState, plan *ExecutionPlan, matched []MatchedSkill, mem *OutcomeMemory, p HealingPolicy, exec Executor, autofix ...AutofixFunc) *TaskState {
	if exec == nil {
		exec = NewRealExecutor()
	}
	for {
		if task.CurrentStep >= len(task.Steps) {
			CompleteTask(task)
			_ = PersistTask(root, task.ID, task)
			return task
		}
		step := task.Steps[task.CurrentStep]

		// PRE-EXEC HOOK: skip step if history says it'll fail.
		if mem != nil && !p.IsZero() {
			if d := PreExecHook(step, mem, p); d.Action == "skip" {
				task.Results = append(task.Results, StepResult{
					Step:        step.Step,
					Skill:       step.Skill,
					Command:     "hcloud " + skillShortOrDerived(step.Skill, step.SkillShort) + " " + step.Action,
					StartedAt:   NowISO(),
					FinishedAt:  NowISO(),
					Success:     false,
					Error:       "skipped: " + d.Reason,
					GCLDecision: "SKIPPED_BY_HEALING",
				})
				task.CurrentStep++
				_ = PersistTask(root, task.ID, task)
				continue
			}
		}

		short := skillShortOrDerived(step.Skill, step.SkillShort)
		candidate := "hcloud " + short + " " + step.Action

		risk := step.Risk
		if risk == "" {
			risk = "medium"
		}
		trustLevel := "L2_established"
		score := 0.65
		rbacDec := CheckCommandPermission(candidate, trustLevel, score)

		if !rbacDec.Allowed {
			task.Results = append(task.Results, StepResult{
				Step:         step.Step,
				Skill:        step.Skill,
				Command:      candidate,
				StartedAt:    NowISO(),
				FinishedAt:   NowISO(),
				Success:      false,
				Error:        rbacDec.Reason,
				RBACApproved: false,
				RBACReason:   rbacDec.Reason,
				GCLDecision:  "blocked_by_rbac",
			})
			FailTask(task, rbacDec.Reason)
			_ = PersistTask(root, task.ID, task)
			return task
		}

		genPayload := gcl.GeneratorOutput{Command: candidate, ExitCode: 0, ResultExcerpt: "dry-run"}
		crit := gcl.StructuralCritic(genPayload)
		gclBody := GCLDecisionBody{Scores: crit.Scores, Decision: gcl.Decide(crit.Scores)}

		if crit.Scores["safety"] == 0.0 {
			task.Results = append(task.Results, StepResult{
				Step:         step.Step,
				Skill:        step.Skill,
				Command:      candidate,
				StartedAt:    NowISO(),
				FinishedAt:   NowISO(),
				Success:      false,
				Error:        "safety check failed",
				RBACApproved: true,
				GCLDecision:  "SAFETY_FAIL",
				GCLScores:    crit.Scores,
			})
			FailTask(task, "safety check failed")
			_ = PersistTask(root, task.ID, task)
			return task
		}

		result := StepResult{
			Step:         step.Step,
			Skill:        step.Skill,
			Command:      candidate,
			StartedAt:    NowISO(),
			FinishedAt:   NowISO(),
			Success:      gclBody.Decision == "PASS" || gclBody.Decision == "ACCEPT",
			RBACApproved: rbacDec.Allowed,
			RBACReason:   rbacDec.Reason,
			GCLDecision:  gclBody.Decision,
			GCLScores:    crit.Scores,
		}

		// Dry-run gate (GCL structural critic). Skip-on-deny.
		if gclBody.Decision != "PASS" && gclBody.Decision != "ACCEPT" {
			result.Success = false
			result.Error = "skipped: GCL=" + gclBody.Decision
		} else {
			// Run for real via Executor (defaults to RealExecutor when nil).
			code, stdout, runErr := exec.Run(candidate, time.Duration(plan.MaxTotalTimeoutSeconds)*time.Second)
			result.ExitCode = code
			result.Output = stdout
			if runErr != nil {
				result.Error = runErr.Error()
				result.Success = false
			} else if code != 0 {
				result.Success = false
			} else {
				result.Success = true
			}
		}

		// POST-FAILURE HOOK: retry transient errors, escalate permanent.
		if mem != nil && !p.IsZero() && exec != nil && !result.Success {
			if d := PostFailureHook(step, result, 0, mem, p); d.Action == "retry" {
				if p.RetryBackoff > 0 {
					time.Sleep(p.RetryBackoff)
				}
				_ = PersistTask(root, task.ID, task)
				continue // re-run the same step
			}
			// Permanent failure → attempt autonomous remediation (L4 autofix).
			if len(autofix) > 0 && autofix[0] != nil {
				candidate := "hcloud " + skillShortOrDerived(step.Skill, step.SkillShort) + " " + step.Action
				fixRes := autofix[0](step.Skill, candidate)
				if fixRes.Executed {
					// The autofix hook ran a remediation; record it as a step outcome
					// so the audit trace reflects the autonomous action.
					result.Error = "escalated→autofix[" + fixRes.PlaybookID + "]: " + fixRes.Action
					if fixRes.Success {
						result.Success = true
					}
				}
			}
		}

		task.Results = append(task.Results, result)

		// Record outcome for future pre-exec decisions.
		if mem != nil && exec != nil {
			_ = mem.Record(OutcomeRecord{
				ID:           newOutcomeID(),
				Timestamp:    NowISO(),
				TaskID:       task.ID,
				Skill:        step.Skill,
				Action:       step.Action,
				ContextHash:  hashContext(candidate),
				Outcome:      outcomeString(result.Success),
				ErrorClass:   errorClass(result.Error),
				ErrorMsg:     truncate(result.Error, 200),
				Risk:         risk,
				RBACDecision: rbacDecisionString(rbacDec.Allowed),
				GCLDecision:  gclBody.Decision,
			})
		}

		task.CurrentStep++
		_ = PersistTask(root, task.ID, task)
	}
}

// skillShortOrDerived returns step.SkillShort or derives a short name.
func skillShortOrDerived(skill, short string) string {
	if short != "" {
		return short
	}
	return strings.ReplaceAll(strings.ReplaceAll(skill, "huaweicloud-", ""), "-ops", "")
}

// hashContext returns the first 16 hex chars of sha256 over the *stable*
// form of command (see stripVolatileArgs).
//
// Stable tokens (kept): CLI binary, product short-name, action/verb, and
// non-volatile flags/values (resource IDs, names, specs).
// Volatile tokens (stripped): time windows, pagination markers, client/
// request IDs — otherwise every call hashes uniquely and MatchOutcomes
// never correlates repeats (Eng-m1 / T-6).
func hashContext(command string) string {
	sum := sha256.Sum256([]byte(stripVolatileArgs(command)))
	return hex.EncodeToString(sum[:])[:16]
}

// volatileFlagNames are CLI flag names (without leading dashes) whose
// values change every invocation and must not enter ContextHash.
var volatileFlagNames = map[string]bool{
	"query-window": true,
	"start-time":   true,
	"end-time":     true,
	"start_time":   true,
	"end_time":     true,
	"since":        true,
	"until":        true,
	"from-time":    true,
	"to-time":      true,
	"marker":       true,
	"page":         true,
	"offset":       true,
	"client-token": true,
	"request-id":   true,
	"x-request-id": true,
	"timestamp":    true,
}

// stripVolatileArgs drops known-volatile --flags (and their values) from a
// shell command string. Supports both `--flag value` and `--flag=value`.
// Resource IDs and non-listed flags are preserved.
func stripVolatileArgs(command string) string {
	tokens := strings.Fields(command)
	if len(tokens) == 0 {
		return command
	}
	out := make([]string, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		name, inlineVal, ok := splitFlag(tok)
		if !ok {
			out = append(out, tok)
			continue
		}
		if !volatileFlagNames[name] {
			out = append(out, tok)
			continue
		}
		// Drop --flag=value (inline) or --flag + next token (separate value).
		if inlineVal {
			continue
		}
		if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
			i++
		}
	}
	return strings.Join(out, " ")
}

// splitFlag parses a CLI token. Returns (name, hasInlineValue, isFlag).
// "--query-window=1h" → ("query-window", true, true)
// "--query-window"    → ("query-window", false, true)
// "ecs"               → ("", false, false)
func splitFlag(tok string) (name string, inlineVal bool, isFlag bool) {
	if !strings.HasPrefix(tok, "--") || len(tok) <= 2 {
		return "", false, false
	}
	body := tok[2:]
	if eq := strings.IndexByte(body, '='); eq >= 0 {
		return body[:eq], true, true
	}
	return body, false, true
}

// newOutcomeID returns a 16-char hex string for OutcomeRecord.ID.
func newOutcomeID() string {
	var b [8]byte
	_ = mustReadRandom(b[:])
	return hex.EncodeToString(b[:])
}

// outcomeString normalizes a bool to "success" / "failure".
func outcomeString(success bool) string {
	if success {
		return "success"
	}
	return "failure"
}

// errorClass returns "transient" for transient errors, "" otherwise.
func errorClass(errMsg string) string {
	if isTransient(errMsg) {
		return "transient"
	}
	return ""
}

// rbacDecisionString returns "allowed" / "denied" for RBAC results.
func rbacDecisionString(allowed bool) string {
	if allowed {
		return "allowed"
	}
	return "denied"
}

// truncate returns at most n bytes of s, with an ellipsis when truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n < 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

// BuildTaskFromPlan converts an ExecutionPlan into a TaskState for persistence.
func BuildTaskFromPlan(plan *ExecutionPlan, fault, root string) *TaskState {
	taskSteps := make([]TaskStep, len(plan.Steps))
	for i, s := range plan.Steps {
		taskSteps[i] = TaskStep{
			Step:       s.Step,
			Skill:      s.Skill,
			SkillShort: s.SkillShort,
			Action:     s.Action,
			Risk:       inferRiskFromAction(s.Action),
		}
	}

	return &TaskState{
		ID:          newTaskID(),
		Fault:       fault,
		Root:        root,
		CreatedAt:   NowISO(),
		UpdatedAt:   NowISO(),
		Status:      TaskStatusRunning,
		CurrentStep: 0,
		Steps:       taskSteps,
		Results:     nil,
	}
}

// inferRiskFromAction guesses the risk level from the action name.
// This is a heuristic; callers should override with actual risk if known.
func inferRiskFromAction(action string) string {
	actionLower := strings.ToLower(action)
	// Destructive actions are high risk.
	for _, d := range []string{"delete", "terminate", "destroy", "drop", "remove"} {
		if strings.Contains(actionLower, d) {
			return "high"
		}
	}
	// Read-only actions are low risk.
	for _, r := range []string{"list", "describe", "show", "get", "query", "search"} {
		if strings.Contains(actionLower, r) {
			return "low"
		}
	}
	// Default to medium.
	return "medium"
}
