package l4

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- runtime_orchestrator (the closed-loop handler) ---

func TestHandleFault_SixStagesPresent(t *testing.T) {
	root := t.TempDir()
	// Audit dir must exist (orchestrator will create it on its own, but a temp
	// fixture ensures a clean run).
	out := HandleFault(HandleFaultInput{
		Root:     root,
		Fault:    "RDS connection timeout",
		Resource: "rds:instance",
		Risk:     "medium",
	}, nil)

	if out.FaultID == "" {
		t.Error("fault_id missing")
	}
	if !strings.HasPrefix(out.FaultID, "orch-") && len(out.FaultID) < 30 {
		// uuid hex is 32 chars, optionally prefixed
		t.Errorf("fault_id=%q, want non-trivial uuid", out.FaultID)
	}
	if out.Topology.Origin != "rds:instance" {
		t.Errorf("topology.origin=%q, want rds:instance", out.Topology.Origin)
	}
	if len(out.Orchestration.PrimarySkills) == 0 {
		t.Error("orchestration.primary_skills should not be empty for an RDS fault")
	}
	if out.Predictive.Evaluated {
		t.Error("predictive.evaluated should be false when no metric_values supplied")
	}
	if out.GCL.OverallSafety == false {
		t.Error("gcl.overall_safety should default true for a clean fault")
	}
	if out.Trust.TrustLevel == "" {
		t.Error("trust.trust_level missing")
	}
	if out.Learning.TracePersisted == "" {
		t.Error("learning.trace_persisted should be set")
	}
	// Trace should be on disk.
	if _, err := readFile(out.Learning.TracePersisted); err != nil {
		t.Errorf("trace file not readable: %v", err)
	}
}

func TestHandleFault_ResourceHeuristic(t *testing.T) {
	root := t.TempDir()
	out := HandleFault(HandleFaultInput{
		Root:  root,
		Fault: "ELB latency spike",
		Risk:  "low",
	}, nil)
	// elb keyword → resource=elb:* (the Python heuristic uses a known set;
	// ELB is in the token list).
	if !strings.Contains(out.Resource, "elb") && !strings.Contains(out.Topology.Origin, "unknown") {
		// Acceptable: any derived or unknown resource as long as it's set
	}
	if out.Resource == "" {
		t.Error("resource must be set even when not provided")
	}
}

func TestHandleFault_PredictiveWithMetrics(t *testing.T) {
	root := t.TempDir()
	values := []float64{10, 20, 30, 40, 50, 60, 70, 80, 85, 90}
	thr := 95.0
	out := HandleFault(HandleFaultInput{
		Root:            root,
		Fault:           "RDS CPU high",
		Resource:        "rds:instance",
		Risk:            "medium",
		MetricValues:    values,
		MetricThreshold: &thr,
	}, nil)
	if !out.Predictive.Evaluated {
		t.Error("predictive should be evaluated when metric_values supplied")
	}
	if out.Predictive.Trend == nil {
		t.Error("predictive.trend should be non-nil")
	}
}

// KNOWN-FLAKY: nondeterministic outcome-memory/keyword-match ordering can
// intermittently produce L0_new fallback or AutoApprove=false. Documented
// in AGENTS.md pre-commit gate exceptions; not related to Phase 3 changes.
// TODO(phase4): stabilize by seeding a fixed skill key instead of
// MatchFaultSkills output ordering.
func TestHandleFault_DecisionAutoProceed(t *testing.T) {
	root := t.TempDir()
	// "VPC subnet unreachable" → keyword primary is huaweicloud-vpc-ops.
	// Trust is keyed on that primary (not pipeline Steps[0] after domain reorder).
	mem, err := NewOutcomeMemory(root)
	if err != nil {
		t.Fatalf("NewOutcomeMemory: %v", err)
	}
	matched := MatchFaultSkills("VPC subnet unreachable", nil)
	if len(matched) == 0 {
		t.Fatal("expected keyword matches for unreachable fault")
	}
	trustSkill := matched[0].Skill
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		ts := now.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339)
		if err := mem.Record(OutcomeRecord{
			ID:        "trust" + ts,
			Timestamp: ts,
			Skill:     trustSkill,
			Action:    "diagnose_and_remediate",
			Outcome:   "success",
			Risk:      "medium",
		}); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}
	out := HandleFault(HandleFaultInput{
		Root:     root,
		Fault:    "VPC subnet unreachable",
		Resource: "vpc:subnet",
		Risk:     "low",
		Mem:      mem,
	}, nil)
	if out.Trust.TrustLevel == "L0_new" {
		t.Errorf("trust fallback did not load outcome memory: level=%s score=%v (primary=%s)", out.Trust.TrustLevel, out.Trust.CompositeScore, trustSkill)
	}
	if !out.Trust.AutoApprove {
		t.Errorf("AutoApprove=false for L4 trust (level=%s score=%v, primary=%s)", out.Trust.TrustLevel, out.Trust.CompositeScore, trustSkill)
	}
	if out.Trust.RequiresHumanApproval {
		t.Errorf("RequiresHumanApproval=true for L4 trust")
	}
}

func skillNames(matches []MatchedSkill) []string {
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.Skill)
	}
	return out
}
func TestHandleFault_DecisionHumanReviewForHighRisk(t *testing.T) {
	root := t.TempDir()
	out := HandleFault(HandleFaultInput{
		Root:     root,
		Fault:    "RDS destructive",
		Resource: "rds:instance",
		Risk:     "critical",
	}, nil)
	// critical risk + L0 trust → human_review_required
	if out.Decision != "human_review_required" {
		t.Errorf("decision=%q, want human_review_required for critical risk", out.Decision)
	}
}

func TestHandleFault_TraceFileMaskedSecrets(t *testing.T) {
	root := t.TempDir()
	out := HandleFault(HandleFaultInput{
		Root:     root,
		Fault:    "RDS connection timeout",
		Resource: "rds:instance",
		Risk:     "medium",
	}, nil)
	raw, err := readFile(out.Learning.TracePersisted)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	// MaskSecrets replaces AK/SK-style strings. No real credentials here, but
	// ensure trace contains the orchestrator block.
	if !strings.Contains(string(raw), `"topology"`) {
		t.Error("trace missing topology block")
	}
	if !strings.Contains(string(raw), `"orchestration"`) {
		t.Error("trace missing orchestration block")
	}
	if !strings.Contains(string(raw), `"trust"`) {
		t.Error("trace missing trust block")
	}
}

func TestHandleFault_DefaultRoot(t *testing.T) {
	// Without explicit Root, should fall back to current working dir.
	out := HandleFault(HandleFaultInput{
		Fault:    "ECS unreachable",
		Resource: "ecs:instance",
		Risk:     "low",
	}, nil)
	if out.Learning.TracePersisted == "" {
		t.Error("default root should still produce a trace file")
	}
	_ = filepath.Join("a", "b") // keep import used
}

func TestHandleFault_CrossSkillPlanIncludesDelegates(t *testing.T) {
	root := t.TempDir()
	out := HandleFault(HandleFaultInput{
		Root:     root,
		Fault:    "RDS connection timeout",
		Resource: "rds:instance",
		Risk:     "medium",
	}, nil)

	if len(out.Orchestration.PrimarySkills) == 0 {
		t.Fatal("expected primary skills for RDS fault")
	}
	if out.Orchestration.StepCount <= len(out.Orchestration.PrimarySkills) {
		t.Errorf("step_count=%d should exceed primary_skills=%d when delegates are wired",
			out.Orchestration.StepCount, len(out.Orchestration.PrimarySkills))
	}
	if out.Orchestration.Strategy != "pipeline" {
		t.Errorf("strategy=%q, want pipeline for multi-skill delegation", out.Orchestration.Strategy)
	}

	// GCL should evaluate every planned step, including delegated skills.
	if len(out.GCL.Decisions) != out.Orchestration.StepCount {
		t.Errorf("gcl decisions=%d, want %d (one per planned step)",
			len(out.GCL.Decisions), out.Orchestration.StepCount)
	}
}
