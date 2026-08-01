package l4

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/gcl"
)

// TestL4AutonomousClosedLoop is the Phase 4 (Batch L4-D) acceptance guard:
// an end-to-end, zero-human-intervention run of the L4 closed loop over a
// non-destructive fault, with all five pipeline stages present on the
// emitted trace. It also drives the GCL runner directly to prove the
// runtime quality-gate trace (gcl-trace-*.json) is produced, PASS, and
// credential-safe.
//
// Determinism note: we seed >=5 consecutive successes for the matched
// (skill, action) so trust auto-approves (Phase 3 exploration window
// satisfied), avoiding the nondeterministic MatchFaultSkills-ordering
// flakiness that affects TestHandleFault_DecisionAutoProceed.
func TestL4AutonomousClosedLoop(t *testing.T) {
	root := t.TempDir()
	mem, err := NewOutcomeMemory(root)
	if err != nil {
		t.Fatalf("NewOutcomeMemory: %v", err)
	}
	// Seed the skill/action the VPC fault will resolve to (see
	// TestHandleFault_DecisionAutoProceed for the keyword match).
	const skill, action = "huaweicloud-vpc-ops", "diagnose_and_remediate"
	seedConsecutiveSuccesses(t, mem, skill, action, 6) // > ExplorationWindow(5)

	out := HandleFault(HandleFaultInput{
		Root:     root,
		Fault:    "VPC subnet unreachable",
		Resource: "vpc:subnet",
		Risk:     "low",
		Mem:      mem,
	}, nil)

	// 1) Trace file persisted and parseable.
	if out.Learning.TracePersisted == "" {
		t.Fatal("orchestrator did not persist a trace")
	}
	raw, err := os.ReadFile(out.Learning.TracePersisted)
	if err != nil {
		t.Fatalf("read trace %s: %v", out.Learning.TracePersisted, err)
	}
	var trace map[string]any
	if err := json.Unmarshal(raw, &trace); err != nil {
		t.Fatalf("parse trace: %v", err)
	}

	// 2) All five closed-loop stages present, in order: Detect→Diagnose→
	//    Execute→Verify→Learn (Phase 4 evidence contract).
	stagesRaw, ok := trace["stages"].([]any)
	if !ok {
		t.Fatalf("trace missing stages field: %#v", trace)
	}
	wantStages := []string{"detect", "diagnose", "execute", "verify", "learn"}
	if len(stagesRaw) != len(wantStages) {
		t.Fatalf("expected %d stages, got %d: %#v", len(wantStages), len(stagesRaw), stagesRaw)
	}
	for i, w := range wantStages {
		stage, _ := stagesRaw[i].(map[string]any)
		if stage["stage"] != w {
			t.Errorf("stage[%d] = %v, want %q", i, stage["stage"], w)
		}
		if done, _ := stage["done"].(bool); !done {
			t.Errorf("stage %q must be Done=true in autonomous path", w)
		}
	}

	// 3) GCL safety passed (no SAFETY_FAIL) — the loop is safe to run.
	if !out.GCL.OverallSafety {
		t.Errorf("GCL OverallSafety=false; loop should be safe for low-risk fault (decision=%s)", out.Decision)
	}

	// 4) Autonomous path: trust auto-approved -> decision reached without
	//    a human-review block.
	if !out.Trust.AutoApprove {
		t.Errorf("trust did not auto-approve after 6 seeded successes (level=%s)", out.Trust.TrustLevel)
	}

	// 5) GCL runner integration: a real closed loop through gcl.Run must
	//    persist a PASS trace with masking applied (no leak).
	runGCLClosedLoop(t)
}

// runGCLClosedLoop drives the GCL Generator-Critic loop directly and asserts
// the runtime quality-gate trace contract: PASS, persisted, masked.
func runGCLClosedLoop(t *testing.T) {
	t.Helper()
	cfg := gcl.RunConfig{
		Skill:   "huaweicloud-vpc-ops",
		Request: "verify vpc-abc12345 subnet reachability",
		Command: "echo ok", // deterministic, non-destructive Generator
		MaxIter: 1,
		Timeout: 5,
		Root:    t.TempDir(),
	}
	result := gcl.Run(cfg)
	if result.ExitCode != 0 {
		t.Fatalf("gcl run expected PASS (exit 0), got %d (path=%s)", result.ExitCode, result.TracePath)
	}
	gtrace, err := readGCLTrace(t, result.TracePath)
	if err != nil {
		t.Fatalf("read gcl trace: %v", err)
	}
	if gtrace.Final == nil || gtrace.Final.Status != "PASS" {
		t.Errorf("gcl trace final status not PASS: %#v", gtrace.Final)
	}
	if len(gtrace.Iterations) < 1 {
		t.Errorf("gcl trace has no iterations")
	}
	// Credential/resource safety: the on-disk trace must mask the request.
	if !contains(gtrace.SanitizedRequest, "<id>") && !contains(gtrace.SanitizedRequest, "<masked>") {
		t.Errorf("gcl SanitizedRequest should mask resource id; got %q", gtrace.SanitizedRequest)
	}
	if contains(gtrace.SanitizedRequest, "vpc-abc12345") {
		t.Error("raw resource id leaked into gcl SanitizedRequest")
	}
	// Trace lands under audit-results/ per gcl-spec.
	if filepath.Base(filepath.Dir(result.TracePath)) != "audit-results" {
		t.Errorf("gcl trace not under audit-results/: %s", result.TracePath)
	}
}

// readGCLTrace parses a gcl-trace-*.json file using the exported GCLTrace
// type (gcl is a safe one-directional import: l4 already imports gcl).
func readGCLTrace(t *testing.T, path string) (*gcl.GCLTrace, error) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tr gcl.GCLTrace
	if err := json.Unmarshal(data, &tr); err != nil {
		return nil, err
	}
	return &tr, nil
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
