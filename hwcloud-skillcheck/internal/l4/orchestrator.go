package l4

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/gcl"
	"golang.org/x/sync/errgroup"
)

// primarySkillFromPlan returns the first step's skill (the "primary" skill
// for trust-history lookup), or "" when the plan is empty.
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
	strategy := SelectStrategy(len(discovered), len(discovered) > 1)
	plan := BuildExecutionPlan(in.Fault, matched, strategy)
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
	for _, s := range matched {
		knownSkills[s.Skill] = true
	}

	// Pre-fetch failure patterns for every unique step.Skill concurrently
	// BEFORE entering the step loop. With many steps referencing the same
	// few skills, this collapses N sequential file reads into one fan-out
	// (capped at NumCPU). Each read is a disjoint file (per-skill path),
	// so the work is embarrassingly parallel.
	patternCache := map[string][]map[string]any{}
	if len(plan.Steps) > 0 {
		skillSet := map[string]struct{}{}
		for _, step := range plan.Steps {
			if knownSkills[step.Skill] {
				skillSet[step.Skill] = struct{}{}
			}
		}
		if len(skillSet) > 0 {
			skills := make([]string, 0, len(skillSet))
			for s := range skillSet {
				skills = append(skills, s)
			}
			var patternMu sync.Mutex
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
						patternMu.Lock()
						patternCache[skill] = patterns
						patternMu.Unlock()
					}
					return nil
				})
			}
			// errors are best-effort; an empty cache for a skill means "no
			// known patterns" and falls back to no pre-execution risk.
			_ = g.Wait()
		}
	}

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

	// Step 5 — Trust
	trustHistory := []OpHistory{}
	if in.TrustData != nil {
		if ops, ok := in.TrustData["history"].([]any); ok {
			for _, x := range ops {
				if m, ok := x.(map[string]any); ok {
					h := OpHistory{}
					if s, ok := m["outcome"].(string); ok {
						h.Outcome = s
					}
					if s, ok := m["risk_level"].(string); ok {
						h.RiskLevel = s
					}
					if s, ok := m["timestamp"].(string); ok {
						h.Timestamp = s
					}
					if b, ok := m["had_retry"].(bool); ok {
						h.HadRetry = b
					}
					trustHistory = append(trustHistory, h)
				}
			}
		}
	} else if primary := primarySkillFromPlan(plan); primary != "" {
		// Fallback: load trust_history.json from disk for the primary skill
		// (mirrors Python's cmd_handle: trust_data defaults to <root>/<skill>/assets/trust_history.json).
		if data := LoadTrustData(root, primary); data != nil {
			if ops, ok := data["operations"].(map[string]any); ok {
				for _, list := range ops {
					if arr, ok := list.([]any); ok {
						for _, x := range arr {
							if m, ok := x.(map[string]any); ok {
								h := OpHistory{}
								if s, ok := m["outcome"].(string); ok {
									h.Outcome = s
								}
								if s, ok := m["risk_level"].(string); ok {
									h.RiskLevel = s
								}
								if s, ok := m["timestamp"].(string); ok {
									h.Timestamp = s
								}
								if b, ok := m["had_retry"].(bool); ok {
									h.HadRetry = b
								}
								trustHistory = append(trustHistory, h)
							}
						}
					}
				}
			}
		}
	}
	score := ComputeTrustScore(trustHistory)
	eval := EvaluateOperation(score, risk, in.Fault)
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
	if gclRes.OverallSafety && trustRes.AutoApprove {
		decision = "auto_proceed"
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
		Decision:         decision,
	}
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// readFailurePatternsForSkill is a thin shim over the learning package. It
// returns the patterns slice for a skill, or an error if absent.
func readFailurePatternsForSkill(root, skill string) ([]map[string]any, error) {
	// Lazy import: defer the dependency to avoid an import cycle.
	// The learning package already imports nothing from l4.
	_ = root
	_ = skill
	// Inline loader: open <root>/<skill>/assets/failure_patterns.json.
	skillID := skill
	if !strings.HasPrefix(skill, "huaweicloud-") {
		skillID = "huaweicloud-" + skill + "-ops"
	}
	path := filepath.Join(root, skillID, "assets", "failure_patterns.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data struct {
		Patterns []map[string]any `json:"patterns"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return data.Patterns, nil
}

// matchPreExecutionRisk is a structural-critic pre-execution risk check. It
// scans the command for tokens in any pattern's error_message_regex.
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
			}
		}
	}
	return nil
}
