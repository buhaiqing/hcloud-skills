package l4

import (
	"context"
	"runtime"
	"strings"
	"sync"

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
