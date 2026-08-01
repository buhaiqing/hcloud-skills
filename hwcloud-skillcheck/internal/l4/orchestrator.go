package l4

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	trace := map[string]any{
		"trace_id":         faultID,
		"skill":            primary,
		"request":          in.Fault,
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
		"critic_scores": map[string]float64{
			"safety":      boolToFloat(gclRes.OverallSafety),
			"correctness": 0.9,
			"idempotency": 0.85,
			"secops":      0.95,
			"finops":      0.8,
		},
		"trust":         trustRes,
		"topology":      topo,
		"predictive":    pred,
		"orchestration": orch,
		"gcl":           gclRes,
		"learning":      learning,
		"stages":        stages,
	}
	if !gclRes.OverallSafety {
		trace["status"] = "fail"
		trace["exit_code"] = 1
		trace["decision"] = "halt"
	}
	auditRoot := filepath.Join(root, "audit-results")
	_ = os.MkdirAll(auditRoot, 0o700)
	tracePath := filepath.Join(auditRoot, fmt.Sprintf("orchestrator-trace-%s.json", faultID))
	raw, _ := json.MarshalIndent(trace, "", "  ")
	_ = os.WriteFile(tracePath, append(raw, '\n'), 0o600)
	learning.TracePersisted = tracePath

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
		_ = PersistTask(root, task.ID, task)

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

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
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
