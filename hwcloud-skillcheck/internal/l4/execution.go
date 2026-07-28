package l4

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/gcl"
	"golang.org/x/sync/errgroup"
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

// RunExecutionLoop executes the planned steps with persistence and RBAC checks.
// Returns the task state after execution completes.
func RunExecutionLoop(root string, task *TaskState, plan *ExecutionPlan, matched []MatchedSkill) *TaskState {
	// Build known skills map.
	knownSkills := map[string]bool{}
	for _, m := range matched {
		knownSkills[m.Skill] = true
	}

	// Pre-fetch failure patterns concurrently.
	patternCache := preFetchPatterns(root, plan, knownSkills)

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

		// Step 3: Record step as pending (will be updated after actual execution).
		// For now, mark as passed the GCL check.
		result := StepResult{
			Step:         step.Step,
			Command:      candidate,
			StartedAt:    NowISO(),
			FinishedAt:   NowISO(),
			Success:      gclBody.Decision == "PASS" || gclBody.Decision == "ACCEPT",
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
// bypassed when mem is nil, p.IsZero() is true, or exec is nil (no executor
// wired). Otherwise it consults OutcomeMemory pre-exec (skip-on-bad-history)
// and post-failure (retry-on-transient / escalate-on-permanent).
func RunExecutionLoopWithHealing(root string, task *TaskState, plan *ExecutionPlan, matched []MatchedSkill, mem *OutcomeMemory, p HealingPolicy, exec Executor) *TaskState {
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
			Command:      candidate,
			StartedAt:    NowISO(),
			FinishedAt:   NowISO(),
			Success:      gclBody.Decision == "PASS" || gclBody.Decision == "ACCEPT",
			RBACApproved: rbacDec.Allowed,
			RBACReason:   rbacDec.Reason,
			GCLDecision:  gclBody.Decision,
			GCLScores:    crit.Scores,
		}

		// If an executor is wired, actually run it and capture exit code / err.
		if exec != nil {
			code, stdout, runErr := exec.Run(candidate, time.Duration(plan.MaxTotalTimeoutSeconds)*time.Second)
			result.ExitCode = code
			if runErr != nil {
				result.Error = runErr.Error()
				result.Success = false
			}
			result.Output = stdout
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

// hashContext returns the first 16 hex chars of sha256(command).
func hashContext(command string) string {
	sum := sha256.Sum256([]byte(command))
	return hex.EncodeToString(sum[:])[:16]
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

// preFetchPatterns loads failure patterns for all skills in the plan concurrently.
func preFetchPatterns(root string, plan *ExecutionPlan, knownSkills map[string]bool) map[string][]map[string]any {
	cache := map[string][]map[string]any{}

	skillSet := map[string]struct{}{}
	for _, step := range plan.Steps {
		if knownSkills[step.Skill] {
			skillSet[step.Skill] = struct{}{}
		}
	}

	if len(skillSet) == 0 {
		return cache
	}

	skills := make([]string, 0, len(skillSet))
	for s := range skillSet {
		skills = append(skills, s)
	}

	var mu sync.Mutex
	g, gCtx := errgroup.WithContext(context.Background())
	g.SetLimit(runtime.NumCPU())

	for _, skill := range skills {
		skill := skill
		g.Go(func() error {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			default:
			}
			patterns, err := readFailurePatternsForSkill(root, skill)
			if err == nil {
				mu.Lock()
				cache[skill] = patterns
				mu.Unlock()
			}
			return nil
		})
	}

	_ = g.Wait()
	return cache
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
