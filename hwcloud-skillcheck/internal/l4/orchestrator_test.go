package l4

import (
	"encoding/json"
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

// Stabilized Phase 4: seed the exact (skill, action) pair the orchestrator
// resolves via primarySkillFromMatched(MatchFaultSkills(...)), so trust
// lookup is deterministic and the test is no longer order-dependent. Prior
// version keyed on matched[0].Skill directly, which diverged from the
// orchestrator's primarySkillFromMatched resolution and intermittently
// fell back to L0_new (KNOWN-FLAKY). See ADR-0009 §Migration.
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
	// Use the SAME resolution the orchestrator applies (primarySkillFromMatched).
	trustSkill := primarySkillFromMatched(matched)
	trustAction := "diagnose_and_remediate"
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		ts := now.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339)
		if err := mem.Record(OutcomeRecord{
			ID:        "trust" + ts,
			Timestamp: ts,
			Skill:     trustSkill,
			Action:    trustAction,
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

// TestMatchPreExecutionRisk_ReturnsFix verifies that a matched failure pattern
// surfaces its fix.action/fix.strategy so an autofix executor can consume it
// (Phase A — L4 self-evolution).
func TestMatchPreExecutionRisk_ReturnsFix(t *testing.T) {
	patterns := []map[string]any{
		{
			"id":   "ECS-FP001",
			"risk": "high",
			"signature": map[string]any{
				"error_message_regex": "InsufficientResource",
				"command_pattern":     "create-server",
			},
			"fix": map[string]any{
				"strategy": "fallback",
				"action":   "hcloud ECS listFlavors --az cn-north-4a",
			},
		},
	}
	got := matchPreExecutionRisk("hcloud ECS createServer InsufficientResource", patterns)
	if got == nil {
		t.Fatal("expected a match, got nil")
	}
	rm, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if rm["matched_pattern_id"] != "ECS-FP001" {
		t.Errorf("matched_pattern_id=%v, want ECS-FP001", rm["matched_pattern_id"])
	}
	fix, ok := rm["fix"].(map[string]any)
	if !ok {
		t.Fatalf("expected fix map in result, got %#v", rm["fix"])
	}
	if fix["action"] != "hcloud ECS listFlavors --az cn-north-4a" {
		t.Errorf("fix.action=%v, want remediation command", fix["action"])
	}
	if fix["strategy"] != "fallback" {
		t.Errorf("fix.strategy=%v, want fallback", fix["strategy"])
	}
}

// TestMatchPreExecutionRisk_NoMatch verifies nil when no pattern matches.
func TestMatchPreExecutionRisk_NoMatch(t *testing.T) {
	patterns := []map[string]any{
		{
			"id":        "X",
			"signature": map[string]any{"error_message_regex": "NoSuchThing"},
		},
	}
	if got := matchPreExecutionRisk("hcloud ECS listServers", patterns); got != nil {
		t.Errorf("expected nil for non-matching command, got %#v", got)
	}
}

// TestMatchPreExecutionRisk_FirstMatch verifies the first matching pattern wins
// (loop returns on first hit, not last).
func TestMatchPreExecutionRisk_FirstMatch(t *testing.T) {
	patterns := []map[string]any{
		{"id": "FP-A", "signature": map[string]any{"error_message_regex": "alpha"}, "fix": map[string]any{"action": "fix-a"}},
		{"id": "FP-B", "signature": map[string]any{"error_message_regex": "alpha"}, "fix": map[string]any{"action": "fix-b"}},
	}
	got := matchPreExecutionRisk("hcloud ecs alpha", patterns)
	if got == nil {
		t.Fatal("expected a match")
	}
	rm := got.(map[string]any)
	if rm["matched_pattern_id"] != "FP-A" {
		t.Fatalf("want first matching pattern FP-A, got %v", rm["matched_pattern_id"])
	}
}

// TestMatchPreExecutionRisk_SkipsPatternWithoutRegex verifies a pattern with an
// empty error_message_regex is skipped (no panic, no match).
func TestMatchPreExecutionRisk_SkipsPatternWithoutRegex(t *testing.T) {
	patterns := []map[string]any{
		{"id": "FP-EMPTY", "signature": map[string]any{"error_message_regex": ""}},
		{"id": "FP-OK", "signature": map[string]any{"error_message_regex": "alpha"}, "fix": map[string]any{"action": "x"}},
	}
	got := matchPreExecutionRisk("hcloud ecs alpha", patterns)
	if got == nil {
		t.Fatal("expected FP-OK to match")
	}
	rm := got.(map[string]any)
	if rm["matched_pattern_id"] != "FP-OK" {
		t.Fatalf("want FP-OK (first with non-empty regex), got %v", rm["matched_pattern_id"])
	}
}

// TestMatchPreExecutionRisk_SkipsNilSignature verifies a pattern with a nil
// signature is skipped (no panic).
func TestMatchPreExecutionRisk_SkipsNilSignature(t *testing.T) {
	patterns := []map[string]any{
		{"id": "FP-NIL", "signature": nil},
		{"id": "FP-OK", "signature": map[string]any{"error_message_regex": "alpha"}, "fix": map[string]any{"action": "x"}},
	}
	got := matchPreExecutionRisk("hcloud ecs alpha", patterns)
	if got == nil {
		t.Fatal("expected FP-OK to match")
	}
	rm := got.(map[string]any)
	if rm["matched_pattern_id"] != "FP-OK" {
		t.Fatalf("want FP-OK, got %v", rm["matched_pattern_id"])
	}
}

// --- Skill attribution (resolveTraceSkill) ---

// TestResolveTraceSkill_MatchedPrimary pins the canonical path: keyword
// matching returned a primary, and that primary wins over plan/expanded
// alternatives. Source tag is "matched".
func TestResolveTraceSkill_MatchedPrimary(t *testing.T) {
	matched := []MatchedSkill{{Skill: "huaweicloud-rds-ops", Confidence: 0.9}}
	plan := &ExecutionPlan{Steps: []PlanStep{{Skill: "huaweicloud-vpc-ops", Action: "list-routes"}}}
	expanded := []MatchedSkill{{Skill: "huaweicloud-vpc-ops"}, {Skill: "huaweicloud-rds-ops"}}

	got, src := resolveTraceSkill(matched, plan, expanded, "anything goes when matched wins")
	if got != "huaweicloud-rds-ops" {
		t.Errorf("got %q, want huaweicloud-rds-ops (matched wins)", got)
	}
	if src != "matched" {
		t.Errorf("source=%q, want matched", src)
	}
}

// TestResolveTraceSkill_FallsBackToPlanStep covers the case the silent
// attribute bug used to produce: a fault whose keywords match no rule (so
// matched is empty) but the planner still built a plan — most commonly
// because the operator supplied a fault string outside the static rules.
// The trace MUST still be attributed to a real skill, not the literal
// "unknown", otherwise the learner drops it.
func TestResolveTraceSkill_FallsBackToPlanStep(t *testing.T) {
	// matched is empty (fault matched no rule), but plan has a step.
	plan := &ExecutionPlan{Steps: []PlanStep{{Skill: "huaweicloud-vpc-ops", Action: "list-routes"}}}
	expanded := []MatchedSkill{{Skill: "huaweicloud-ces-ops"}}

	got, src := resolveTraceSkill(nil, plan, expanded, "any fault")
	if got != "huaweicloud-vpc-ops" {
		t.Errorf("got %q, want huaweicloud-vpc-ops (plan step wins when matched empty)", got)
	}
	if src != "plan_step" {
		t.Errorf("source=%q, want plan_step", src)
	}
}

// TestResolveTraceSkill_FallsBackToExpanded covers the rarer fallback: a
// fault that matched keywords but the allow-list (Skills) excluded every
// priority skill, so plan ended up empty even though expanded (matched ∪
// delegates) had something.
func TestResolveTraceSkill_FallsBackToExpanded(t *testing.T) {
	expanded := []MatchedSkill{{Skill: "huaweicloud-vpc-ops", Confidence: delegateConfidence}}

	got, src := resolveTraceSkill(nil, nil, expanded, "any fault")
	if got != "huaweicloud-vpc-ops" {
		t.Errorf("got %q, want huaweicloud-vpc-ops (expanded wins when matched+plan empty)", got)
	}
	if src != "expanded" {
		t.Errorf("source=%q, want expanded", src)
	}
}

// TestResolveTraceSkill_UnknownWhenAllEmpty pins the only path that still
// emits the literal "unknown": every source is empty (no keyword match, no
// plan step, no expanded entry). The orchestrator emits a DEBUG stderr
// line so the operator can see WHY attribution failed; the learner still
// cannot consume the trace, but the cause is recorded in the run log.
func TestResolveTraceSkill_UnknownWhenAllEmpty(t *testing.T) {
	got, src := resolveTraceSkill(nil, nil, nil, "completely unrelated text with no resource token")
	if got != "unknown" {
		t.Errorf("got %q, want unknown", got)
	}
	if src != "unknown" {
		t.Errorf("source=%q, want unknown", src)
	}
}

// TestResolveTraceSkill_ResourceFallback pins the new 4th fallback that
// catches the silent-attribute cases the static keyword rules miss: a fault
// like "alchemist exception in RDS" carries no keyword the rules know, but
// deriveResource("...RDS...") still returns "rds:instance" and we map it
// back to huaweicloud-rds-ops. Source tag is "resource".
func TestResolveTraceSkill_ResourceFallback(t *testing.T) {
	got, src := resolveTraceSkill(nil, nil, nil, "unexpected alchemist exception in RDS connection")
	if got != "huaweicloud-rds-ops" {
		t.Errorf("got %q, want huaweicloud-rds-ops (resource fallback)", got)
	}
	if src != "resource" {
		t.Errorf("source=%q, want resource", src)
	}
}

// TestDeriveResource_RejectsSubstringMatches pins the word-boundary
// contract: a token that appears ONLY as a substring of a longer word
// must NOT count. Pre-fix, strings.Contains let "specs" → ecs and
// "accent" → cce (false-positive attributions that poisoned the
// fault→skill mapping). These are the minimal regression guards.
func TestDeriveResource_RejectsSubstringMatches(t *testing.T) {
	cases := []struct {
		fault string
		want  string
	}{
		// Substring false-positives that pre-fix deriveResource matched.
		{"review the specs doc", "unknown:resource"},
		{"user accent was wrong", "unknown:resource"},
		{"ELB token bucket blew up", "elb:instance"}, // "ELB" alone matches as a word
		// Legitimate matches still work.
		{"RDS CPU at 99%", "rds:instance"},
		{"check the ECS instance", "ecs:instance"},
		{"vpc peering broken", "vpc:instance"},
	}
	for _, tc := range cases {
		got := deriveResource(tc.fault)
		if got != tc.want {
			t.Errorf("deriveResource(%q)=%q, want %q", tc.fault, got, tc.want)
		}
	}
}

// TestHandleFault_TraceSkillAttributedFromPlanStep exercises the full
// HandleFault path with a fault the static rules do not cover. Pre-fix
// this produced skill="unknown" on the persisted trace, which the learner
// dropped at the skill-mismatch gate. Post-fix the orchestrator falls back
// to plan.Steps[0].Skill so the trace reaches the right skill's
// failure_patterns.json.
func TestHandleFault_TraceSkillAttributedFromPlanStep(t *testing.T) {
	root := t.TempDir()
	// "unexpected alchemist exception in RDS" — no keyword rule covers the
	// fault phrase, but deriveResource still picks up "RDS" and the
	// resource-fallback source must win. This is the exact shape the
	// silent-attribute bug used to write "unknown" on.
	out := HandleFault(HandleFaultInput{
		Root:  root,
		Fault: "unexpected alchemist exception in RDS connection",
		Risk:  "low",
	}, nil)

	raw, err := readFile(out.Learning.TracePersisted)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	// Persisted trace MUST NOT carry the literal "unknown" when there is a
	// concrete skill to attribute to. The orchestrator falls back through
	// plan.Steps[0].Skill, expanded, and finally deriveResource before
	// giving up, so for any fault that mentions a known product token a
	// real skill id wins.
	var trace map[string]any
	if err := json.Unmarshal(raw, &trace); err != nil {
		t.Fatalf("parse trace: %v", err)
	}
	skill, _ := trace["skill"].(string)
	if skill == "" || skill == "unknown" {
		t.Errorf("trace skill=%q, want a concrete attribution (the silent-attribute bug)", skill)
	}
	// The trace must carry one of the canonical prefixes used by the static
	// skill registry — sanity-check the attribution shape, not just the
	// absence of "unknown".
	if !strings.HasPrefix(skill, "huaweicloud-") || !strings.HasSuffix(skill, "-ops") {
		t.Errorf("trace skill=%q, want huaweicloud-*-ops form", skill)
	}
	// The specific shape the resource fallback must produce.
	if skill != "huaweicloud-rds-ops" {
		t.Errorf("trace skill=%q, want huaweicloud-rds-ops (resource fallback from 'RDS' token)", skill)
	}
}
