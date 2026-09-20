package l4

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/gcl"
)

// primarySkillFromMatched returns the keyword-matched primary skill (not the
// pipeline-reordered first step). Trust and context attribution must use this
// so a high-trust monitoring delegate cannot auto-approve a low-trust primary.
func primarySkillFromMatched(matched []MatchedSkill) string {
	if len(matched) == 0 {
		return ""
	}
	return matched[0].Skill
}

// primarySkillFromPlan returns the first step's skill after plan ordering.
// Prefer primarySkillFromMatched for trust / fault attribution (ADR-0011).
func primarySkillFromPlan(p *ExecutionPlan) string {
	if p == nil || len(p.Steps) == 0 {
		return ""
	}
	return p.Steps[0].Skill
}

// HandleFaultInput is the input to HandleFault.
type HandleFaultInput struct {
	Root            string
	Fault           string
	Resource        string
	Risk            string
	Skills          []string
	TrustData       map[string]any
	MetricValues    []float64
	MetricThreshold *float64
	// ContextMem records the orchestrator's lifecycle events across
	// invocations. When nil, a fresh ContextMemory is created from
	// the resolved root directory.
	ContextMem *ContextMemory
	// Mem is the outcome-memory store for self-healing. When nil, a fresh
	// OutcomeMemory is created from the resolved root directory.
	Mem *OutcomeMemory
	// Policy configures self-healing behavior. Zero value = no healing
	// (same behavior as before this feature existed).
	Policy HealingPolicy
	// Autofix is an injectable autonomous-remediation hook. When non-nil,
	// it is threaded into RunExecutionLoopWithHealing and invoked on a step's
	// permanent failure. The CLI layer bridges it to internal/learning so
	// internal/l4 stays import-cycle-free.
	Autofix AutofixFunc
}

// TopologyResult is the public topology block in the orchestrator output.
type TopologyResult struct {
	Origin            string   `json:"origin"`
	TotalAffected     int      `json:"total_affected"`
	MaxDepthReached   int      `json:"max_depth_reached"`
	CriticalityScore  float64  `json:"criticality_score"`
	DomainsImpacted   []string `json:"domains_impacted"`
	AffectedResources []string `json:"affected_resources"`
}

// OrchestrationResult is the orchestration block.
type OrchestrationResult struct {
	PrimarySkills          []string `json:"primary_skills"`
	TransitiveSkills       []string `json:"transitive_skills"`
	Strategy               string   `json:"strategy"`
	PlanID                 string   `json:"plan_id"`
	StepCount              int      `json:"step_count"`
	MaxTotalTimeoutSeconds int      `json:"max_total_timeout_seconds"`
}

// PredictiveResult is the predictive block.
type PredictiveResult struct {
	Trend     *Trend          `json:"trend"`
	Breach    *BreachForecast `json:"breach"`
	Evaluated bool            `json:"evaluated"`
}

// GCLDecision is one step's GCL outcome.
type GCLDecision struct {
	Step int             `json:"step"`
	GCL  GCLDecisionBody `json:"gcl"`
}

// GCLDecisionBody is the per-step body.
type GCLDecisionBody struct {
	Scores           map[string]float64 `json:"scores"`
	Decision         string             `json:"decision"`
	PreExecutionRisk any                `json:"pre_execution_risk"`
}

// GCLResult is the GCL block.
type GCLResult struct {
	OverallSafety bool          `json:"overall_safety"`
	Decisions     []GCLDecision `json:"decisions"`
	PassedSteps   int           `json:"passed_steps"`
}

// TrustResult is the trust block.
type TrustResult struct {
	TrustLevel            string  `json:"trust_level"`
	CompositeScore        float64 `json:"composite_score"`
	AutoApprove           bool    `json:"auto_approve"`
	RequiresHumanApproval bool    `json:"requires_human_approval"`
}

// LearningResult is the learning block.
type LearningResult struct {
	TracePersisted          string   `json:"trace_persisted"`
	PatternsMatched         int      `json:"patterns_matched"`
	KnowledgeBaseSkillsUsed []string `json:"knowledge_base_skills_used"`
}

// StageMarker records one phase of the L4 closed-loop pipeline. Phase 4
// (end-to-end autonomous test) asserts all five appear in order on the
// emitted trace — this is the objective "Detect→Diagnose→Execute→Verify→Learn"
// evidence contract from docs/superpowers/plans/2026-07-31-l4-maturity-upgrade.md §Phase 4.
type StageMarker struct {
	Stage string `json:"stage"` // detect | diagnose | execute | verify | learn
	Done  bool   `json:"done"`
}

// OrchestratorOutput is the top-level result.
type OrchestratorOutput struct {
	FaultID          string              `json:"fault_id"`
	StartedAt        string              `json:"started_at"`
	FinishedAt       string              `json:"finished_at"`
	FaultDescription string              `json:"fault_description"`
	Resource         string              `json:"resource"`
	RiskClass        string              `json:"risk_class"`
	Topology         TopologyResult      `json:"topology"`
	Orchestration    OrchestrationResult `json:"orchestration"`
	Predictive       PredictiveResult    `json:"predictive"`
	GCL              GCLResult           `json:"gcl"`
	Trust            TrustResult         `json:"trust"`
	Learning         LearningResult      `json:"learning"`
	Stages           []StageMarker       `json:"stages"`
	Decision         string              `json:"decision"`
}

// resourceHeuristic mirrors scripts/runtime_orchestrator.py:50-55 — extract
// a resource type from the fault text.
var resourceTokens = []string{"rds", "ecs", "elb", "vpc", "cce", "dcs", "gaussdb", "dms"}

func deriveResource(fault string) string {
	f := strings.ToLower(fault)
	for _, t := range resourceTokens {
		if strings.Contains(f, t) {
			return t + ":instance"
		}
	}
	return "unknown:resource"
}

// HandleFault runs the full L4 closed-loop pipeline.
// Mirrors scripts/runtime_orchestrator.py:handle_fault().
func HandleFault(in HandleFaultInput, _ *struct{}) *OrchestratorOutput {
	root := in.Root
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			root = "."
		}
	}
	faultID := randomHex(16) // matches the Python uuid hex (no dashes)
	startedAt := NowISO()
	resource := in.Resource
	if resource == "" {
		resource = deriveResource(in.Fault)
	}
	risk := in.Risk
	if risk == "" {
		risk = "medium"
	}

	// Resolve outcome-memory: caller-supplied or fresh under root.
	// If creation fails, log and fall back to nil (healing is bypassed).
	mem := in.Mem
	if mem == nil {
		var err error
		mem, err = NewOutcomeMemory(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "orchestrator: outcome memory: %v\n", err)
			mem = nil
		}
	}

	// Step 1 — Topology
	graph := BuildGraphFromSkills(root, nil, false)
	br := graph.BlastRadius(resource, 3)
	topo := TopologyResult{
		Origin:            br.Origin,
		TotalAffected:     br.TotalAffected,
		MaxDepthReached:   br.MaxDepthReached,
		CriticalityScore:  graph.Criticality(resource),
		DomainsImpacted:   br.DomainsImpacted,
		AffectedResources: br.AffectedResources,
	}
	if len(topo.AffectedResources) > 5 {
		topo.AffectedResources = topo.AffectedResources[:5]
	}

	// Step 2 — Orchestration
	matched := MatchFaultSkills(in.Fault, in.Skills)
	primarySkills := make([]string, 0, len(matched))
	for _, m := range matched {
		primarySkills = append(primarySkills, m.Skill)
	}
	discovered := DiscoverTransitiveSkills(primarySkills)
	expanded := ExpandMatchedWithDelegates(matched, discovered, in.Skills)
	hasDelegates := len(expanded) > len(matched)
	strategy := SelectStrategy(len(expanded), hasDelegates)
	plan := BuildExecutionPlan(in.Fault, expanded, strategy)
	orch := OrchestrationResult{
		PrimarySkills:          primarySkills,
		TransitiveSkills:       discovered,
		Strategy:               strategy,
		PlanID:                 plan.PlanID,
		StepCount:              len(plan.Steps),
		MaxTotalTimeoutSeconds: plan.MaxTotalTimeoutSeconds,
	}

	// Step 3 — Predictive
	pred := PredictiveResult{Evaluated: false}
	if len(in.MetricValues) >= 3 {
		trend := DetectTrend(in.MetricValues)
		pred.Trend = &trend
		if in.MetricThreshold != nil {
			b := PredictBreachTime(in.MetricValues, *in.MetricThreshold, 1.0)
			pred.Breach = &b
		}
		pred.Evaluated = true
	}

	// Step 4 — GCL structural critic on the planned steps
	gclRes := GCLResult{OverallSafety: true, Decisions: []GCLDecision{}}
	passCount := 0
	knownSkills := map[string]bool{}
	for _, s := range expanded {
		knownSkills[s.Skill] = true
	}

	// Pre-fetch failure patterns for every unique plan skill concurrently
	// BEFORE entering the step loop (Eng-M2 / T-7 shared helper).
	skills := make([]string, 0, len(knownSkills))
	for s := range knownSkills {
		skills = append(skills, s)
	}
	patternCache := preFetchFailurePatterns(root, skills)

	// Real structural-critic results, one per planned step. Folded into the
	// persisted trace's critic scores below (the trace previously carried
	// hardcoded literals instead — see the P0 audit finding).
	critics := make([]gcl.CriticResult, 0, len(plan.Steps))

	for _, step := range plan.Steps {
		short := step.SkillShort
		if short == "" {
			short = strings.ReplaceAll(strings.ReplaceAll(step.Skill, "huaweicloud-", ""), "-ops", "")
		}
		candidate := fmt.Sprintf("hcloud %s %s", short, step.Action)
		genPayload := gcl.GeneratorOutput{
			Command:       candidate,
			ExitCode:      0,
			ResultExcerpt: "dry-run",
		}
		crit := gcl.StructuralCritic(genPayload)
		// Match pre-execution risk from failure patterns (best-effort).
		var preRisk any
		if knownSkills[step.Skill] {
			if patterns, ok := patternCache[step.Skill]; ok && len(patterns) > 0 {
				preRisk = matchPreExecutionRisk(candidate, patterns)
			}
		}
		body := GCLDecisionBody{
			Scores:           crit.Scores,
			Decision:         gcl.Decide(crit.Scores),
			PreExecutionRisk: preRisk,
		}
		gclRes.Decisions = append(gclRes.Decisions, GCLDecision{Step: step.Step, GCL: body})
		critics = append(critics, crit)
		if crit.Scores["safety"] == 0.0 {
			gclRes.OverallSafety = false
		}
		if body.Decision == "PASS" || body.Decision == "ACCEPT" {
			passCount++
		}
	}
	gclRes.PassedSteps = passCount

	// Step 5 — Trust (Phase 4: outcome-memory only, per ADR-0009 §Migration).
	// Key by keyword-matched primary, NOT pipeline Steps[0]: delegates may
	// reorder monitoring ahead of the fault's primary skill (code-review HIGH).
	trustSkill := primarySkillFromMatched(matched)
	trustAction := "diagnose_and_remediate"
	if plan != nil {
		for _, s := range plan.Steps {
			if s.Skill == trustSkill && s.Action != "" {
				trustAction = s.Action
				break
			}
		}
	}
	score := LookupTrust(trustSkill, trustAction, mem)
	eval := EvaluateOperationWithHistory(score, trustSkill, trustAction, risk, in.Fault, mem)
	trustRes := TrustResult{
		TrustLevel:            score.Level,
		CompositeScore:        score.Score,
		AutoApprove:           eval.AutoApproved,
		RequiresHumanApproval: eval.RequiresConfirmation,
	}

	// Step 6 — Learning: synthesize + persist trace
	primary := "unknown"
	if len(matched) > 0 {
		primary = matched[0].Skill
	}
	primaryCmd := ""
	if len(plan.Steps) > 0 {
		primaryCmd = plan.Steps[0].Action
	}
	patternsMatched := 0
	for _, d := range gclRes.Decisions {
		if d.GCL.PreExecutionRisk != nil {
			patternsMatched++
		}
	}
	usedSkills := []string{}
	for s := range knownSkills {
		usedSkills = append(usedSkills, s)
	}
	learning := LearningResult{
		TracePersisted:          "", // filled after write
		PatternsMatched:         patternsMatched,
		KnowledgeBaseSkillsUsed: usedSkills,
	}
	// Phase 4 evidence contract: the closed loop always Detects (topology)
	// and Diagnoses (skill match); it Executes+Verifies only when trust
	// auto-approves (the autonomous path); Learn (trace persist) always runs.
	executed := gclRes.OverallSafety && trustRes.AutoApprove
	stages := []StageMarker{
		{Stage: "detect", Done: true},
		{Stage: "diagnose", Done: true},
		{Stage: "execute", Done: executed},
		{Stage: "verify", Done: executed},
		{Stage: "learn", Done: true},
	}

	// The persisted critic block is the REAL structural-critic output folded
	// from step 4 — the P0 audit finding was that this site wrote hardcoded
	// literals (0.9/0.85/0.95/0.8) and no `final` block, so every
	// orchestrator trace was unreadable by cmd/aggregate.go and
	// internal/learning (both require `final`).
	traceCritic := traceCriticResult(critics, primaryCmd)
	finalStatus := gclFinalStatus(traceCritic.Scores, gclRes.OverallSafety, len(gclRes.Decisions))

	// The dry-run pipeline makes exactly one pass over the plan, so the
	// folded critic result is recorded as iteration 1 (the gcl.Iteration
	// shape: iter/generator/critic/decision). Traces with no planned steps
	// carry an empty iterations array — there was nothing to iterate on.
	iterations := []any{}
	if len(gclRes.Decisions) > 0 {
		iterations = append(iterations, map[string]any{
			"iter": 1,
			"generator": map[string]any{
				"command":        primaryCmd,
				"exit_code":      0,
				"result_excerpt": "dry-run",
			},
			"critic":   traceCritic,
			"decision": gcl.Decide(traceCritic.Scores),
		})
	}

	auditRoot := filepath.Join(root, "audit-results")
	_ = os.MkdirAll(auditRoot, 0o700)
	tracePath := filepath.Join(auditRoot, fmt.Sprintf("orchestrator-trace-%s.json", faultID))
	learning.TracePersisted = tracePath

	trace := map[string]any{
		"trace_id":         faultID,
		"skill":            primary,
		"request":          in.Fault,
		"fault":            in.Fault, // smoke marker for l4 traces (learning.IsSmokeTrace)
		"source":           "l4",     // readers treat an absent source as "gcl"
		"command":          primaryCmd,
		"started_at":       startedAt,
		"finished_at":      NowISO(),
		"status":           "pass",
		"exit_code":        0,
		"stdout":           "",
		"stderr":           "",
		"iteration":        1,
		"max_iterations":   1,
		"decision":         "pass",
		"resource_scope":   map[string]any{"resource_id": resource, "type": strings.SplitN(resource, ":", 2)[0]},
		"operation_intent": map[string]any{"goal": in.Fault, "risk_class": risk},
		"critic_scores":    traceCritic.Scores,
		"iterations":       iterations,
		"final": map[string]any{
			"status":          finalStatus, // PASS | SAFETY_FAIL | MAX_ITER
			"iter":            1,
			"output":          primaryCmd,
			"dimensions":      traceCritic.Scores,
			"overall":         meanCriticScore(traceCritic.Scores),
			"critic_type":     "structural", // the l4 plan critic is the structural dry-run critic
			"failure_pattern": l4FailurePattern(finalStatus, primary, primaryCmd, traceCritic),
		},
		"trust":         trustRes,
		"topology":      topo,
		"predictive":    pred,
		"orchestration": orch,
		"gcl":           gclRes,
		"learning":      learning,
		"stages":        stages,
	}
	if finalStatus != "PASS" {
		trace["status"] = "fail"
		trace["exit_code"] = 1
		trace["decision"] = "halt"
	}
	raw, _ := json.MarshalIndent(trace, "", "  ")
	_ = os.WriteFile(tracePath, append(raw, '\n'), 0o600)

	decision := "human_review_required"
	executionTask := (*TaskState)(nil)

	// Context memory: instantiate from input or default to <root>/.l4-memory.
	cm := in.ContextMem
	if cm == nil {
		var err error
		cm, err = NewContextMemory(root)
		if err != nil {
			// Don't fail the whole run for context-memory init failure;
			// skip recording for this invocation.
			cm = nil
		}
	}
	// Persist healing/trust counters so `metrics` scrape (separate process)
	// can observe them (code-review HIGH).
	SetMetricsPersistRoot(root)

	faultPrimary := primarySkillFromMatched(matched)

	if gclRes.OverallSafety && trustRes.AutoApprove {
		decision = "auto_proceed"
		// Build task from plan and run execution loop with persistence + RBAC.
		task := BuildTaskFromPlan(plan, in.Fault, root)
		persistTaskChecked(root, task)

		// Record task creation in context memory.
		if cm != nil {
			_ = cm.RecordTask(TaskSummary{
				TaskID:       task.ID,
				Fault:        task.Fault,
				StartedAt:    task.CreatedAt,
				Status:       string(TaskStatusRunning),
				PrimarySkill: faultPrimary,
			})
		}

		executionTask = RunExecutionLoopWithHealing(root, task, plan, expanded, mem, in.Policy, nil, in.Autofix)

		// Record final task status and record each failed step as an error.
		if cm != nil {
			_ = cm.RecordTask(TaskSummary{
				TaskID:       executionTask.ID,
				Fault:        executionTask.Fault,
				StartedAt:    executionTask.CreatedAt,
				FinishedAt:   executionTask.UpdatedAt,
				Status:       string(executionTask.Status),
				PrimarySkill: faultPrimary,
			})
			_ = cm.CloseTask(executionTask.ID)
			for _, r := range executionTask.Results {
				if !r.Success && r.Error != "" {
					errSkill := r.Skill
					if errSkill == "" {
						errSkill = faultPrimary
					}
					_ = cm.RecordError(ErrorSummary{
						Timestamp:  r.FinishedAt,
						Skill:      errSkill,
						Action:     r.Command,
						ErrorClass: "unknown",
						ErrorMsg:   r.Error,
					})
				}
			}
		}

		// Update decision based on execution result.
		switch executionTask.Status {
		case TaskStatusCompleted:
			decision = "completed"
		case TaskStatusFailed:
			decision = "failed"
		case TaskStatusAborted:
			decision = "aborted"
		}
	} else {
		// Not auto-approved; still record the request so future runs have
		// context for similar faults.
		if cm != nil {
			_ = cm.RecordTask(TaskSummary{
				TaskID:       faultID,
				Fault:        in.Fault,
				StartedAt:    startedAt,
				FinishedAt:   NowISO(),
				Status:       "human_review_required",
				PrimarySkill: faultPrimary,
			})
		}
	}
	// Eng-T5: mutations queue in-memory; one Flush at task-finalize.
	if cm != nil {
		if err := cm.Flush(); err != nil {
			fmt.Fprintf(os.Stderr, "orchestrator: context memory flush: %v\n", err)
		}
	}
	return &OrchestratorOutput{
		FaultID:          faultID,
		StartedAt:        startedAt,
		FinishedAt:       NowISO(),
		FaultDescription: in.Fault,
		Resource:         resource,
		RiskClass:        risk,
		Topology:         topo,
		Orchestration:    orch,
		Predictive:       pred,
		GCL:              gclRes,
		Trust:            trustRes,
		Learning:         learning,
		Stages:           stages,
		Decision:         decision,
	}
}

// traceCriticResult folds the per-step structural-critic results into the
// single gcl.CriticResult persisted on the trace. `safety` is the AND across
// steps (one unsafe step fails the whole run, matching
// GCLResult.OverallSafety); every other dimension is the arithmetic mean.
// Suggestions are carried up (capped at 3, mirroring gcl.StructuralCritic) so
// a blocked run stays actionable.
//
// A run whose plan had no steps (e.g. a fault matching no skill) is scored by
// gcl.StructuralCritic on the empty dry-run payload, so the persisted scores
// are always real critic output — never invented literals.
func traceCriticResult(critics []gcl.CriticResult, fallbackCommand string) gcl.CriticResult {
	if len(critics) == 0 {
		return gcl.StructuralCritic(gcl.GeneratorOutput{
			Command:       fallbackCommand,
			ExitCode:      0,
			ResultExcerpt: "dry-run",
		})
	}
	out := gcl.CriticResult{
		Scores: foldCriticScores(critics),
		Mode:   "structural-only",
		Model:  "structural-only",
	}
	for _, c := range critics {
		out.Suggestions = append(out.Suggestions, c.Suggestions...)
		if c.Blocking {
			out.Blocking = true
		}
	}
	if len(out.Suggestions) > 3 {
		out.Suggestions = out.Suggestions[:3]
	}
	return out
}

// foldCriticScores reduces per-step critic scores to one dimension vector.
// Keys are the union of the steps' dimensions, iterated in sorted order so
// the persisted numbers are deterministic.
func foldCriticScores(critics []gcl.CriticResult) map[string]float64 {
	seen := map[string]bool{}
	keys := make([]string, 0, len(critics))
	for _, c := range critics {
		for k := range c.Scores {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)

	out := make(map[string]float64, len(keys))
	for _, k := range keys {
		if k == "safety" {
			min := 1.0
			for _, c := range critics {
				if v, ok := c.Scores[k]; ok && v < min {
					min = v
				}
			}
			out[k] = min
			continue
		}
		sum, n := 0.0, 0
		for _, c := range critics {
			if v, ok := c.Scores[k]; ok {
				sum += v
				n++
			}
		}
		if n > 0 {
			out[k] = sum / float64(n)
		}
	}
	return out
}

// meanCriticScore is the arithmetic mean of a dimension vector, reported as
// final.overall. Pass/fail is decided by gcl.Decide (rubric thresholds), not
// by this number — see gclFinalStatus.
func meanCriticScore(scores map[string]float64) float64 {
	if len(scores) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range scores {
		sum += v
	}
	return sum / float64(len(scores))
}

// gclFinalStatus maps the folded structural-critic verdict onto the GCL trace
// status enum (PASS | SAFETY_FAIL | MAX_ITER — internal/embed/schemas/
// trace.schema.json) so cmd/aggregate.go and internal/learning can bucket
// orchestrator traces exactly like gcl-trace files.
//
// A RETRY verdict is reported as MAX_ITER: this dry-run pipeline never re-runs
// a step, so "the loop exhausted without a PASS" is the accurate terminal
// state. Likewise a run with no planned steps verified nothing and is MAX_ITER
// even though no dimension scored low.
func gclFinalStatus(scores map[string]float64, overallSafety bool, steps int) string {
	if !overallSafety {
		return "SAFETY_FAIL"
	}
	if steps == 0 {
		return "MAX_ITER"
	}
	switch gcl.Decide(scores) {
	case "PASS":
		return "PASS"
	case "SAFETY_FAIL":
		return "SAFETY_FAIL"
	default:
		return "MAX_ITER"
	}
}

// l4FailurePattern builds the `final.failure_pattern` block for a non-PASS run
// so internal/learning can merge it into the skill's failure_patterns.json.
// The block is grounded in the critic's own output — the remediation text is
// the critic's suggestions, and the error string is the stable verdict (a
// stable string is what makes repeated occurrences merge into one pattern).
// PASS runs carry nil: the key must still be present because the canonical
// trace schema requires it.
func l4FailurePattern(status, skill, command string, critic gcl.CriticResult) any {
	if status == "PASS" {
		return nil
	}
	fix := "review the planned command and re-run the dry-run critic"
	if len(critic.Suggestions) > 0 {
		fix = strings.Join(critic.Suggestions, "; ")
	}
	category := "runtime"
	if status == "SAFETY_FAIL" {
		category = "permission" // credential leak / unsafe destructive command
	}
	return map[string]any{
		"category": category,
		"skill":    skill,
		"command":  command,
		"error":    "structural critic verdict: " + status,
		"fix":      fix,
		"count":    1,
		"reusable": true,
	}
}

// matchPreExecutionRisk is a structural-critic pre-execution risk check. It
// scans the command for tokens in any pattern's error_message_regex. On a match
// it returns the matched pattern's id/risk/signature PLUS the fix.action /
// fix.strategy so an autofix executor can consume the remediation (L4
// self-evolution closed loop).
func matchPreExecutionRisk(command string, patterns []map[string]any) any {
	for _, p := range patterns {
		sig, _ := p["signature"].(map[string]any)
		if sig == nil {
			continue
		}
		regex, _ := sig["error_message_regex"].(string)
		if regex == "" {
			continue
		}
		if strings.Contains(command, strings.Fields(regex)[0]) {
			return map[string]any{
				"matched_pattern_id": p["id"],
				"risk_level":         "high",
				"signature":          sig,
				"fix":                p["fix"],
			}
		}
	}
	return nil
}
