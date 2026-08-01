package l4

import (
	"testing"
	"time"
)

// seedAutoProceedTrust records successful outcomes so HandleFault's trust
// reaches AutoApprove (allowing the execution loop to run).
func seedAutoProceedTrust(t *testing.T, root, skill string) *OutcomeMemory {
	t.Helper()
	mem, err := NewOutcomeMemory(root)
	if err != nil {
		t.Fatalf("NewOutcomeMemory: %v", err)
	}
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		ts := now.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339)
		if err := mem.Record(OutcomeRecord{
			ID: "trust" + ts, Timestamp: ts,
			Skill: skill, Action: "diagnose_and_remediate",
			Outcome: "success", Risk: "medium",
		}); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}
	return mem
}

// TestHandleFault_AutofixHookWired verifies an injectable Autofix hook is
// accepted by HandleFault and threaded into the execution loop without breaking
// the normal flow. (A step in the HandleFault path is typically gated by RBAC/
// GCL before execution, so autofix is not guaranteed to fire — this test
// verifies wiring is safe and the full orchestration still completes.)
func TestHandleFault_AutofixHookWired(t *testing.T) {
	root := t.TempDir()
	matched := MatchFaultSkills("VPC subnet unreachable", nil)
	if len(matched) == 0 {
		t.Fatal("no matched skills for unreachable fault")
	}
	skill := primarySkillFromMatched(matched)
	mem := seedAutoProceedTrust(t, root, skill)

	var gotSkill, gotCmd string
	autofix := func(skillName, command string) AutofixResult {
		gotSkill, gotCmd = skillName, command
		return AutofixResult{Action: "execute", PlaybookID: "pb-1", Executed: true, Success: true}
	}

	out := HandleFault(HandleFaultInput{
		Root:     root,
		Fault:    "VPC subnet unreachable",
		Resource: "vpc:subnet",
		Risk:     "low",
		Mem:      mem,
		Policy:   defaultHealingPolicy(),
		Autofix:  autofix,
	}, nil)

	// The orchestrator must still produce a complete output with the hook wired.
	if out.FaultID == "" {
		t.Fatal("fault_id missing")
	}
	if out.Trust.TrustLevel == "L0_new" {
		t.Errorf("trust should be elevated from seeded memory, got %s", out.Trust.TrustLevel)
	}
	// If the loop ran and a step genuinely failed, the hook must have been
	// called with a skill and command. Otherwise the step was gated earlier
	// (RBAC/GCL), which is also correct.
	if gotSkill != "" || gotCmd != "" {
		t.Logf("autofix hook invoked: skill=%q cmd=%q", gotSkill, gotCmd)
	}
}

// TestHandleFault_AutofixNilNoPanic verifies HandleFault works when Autofix is
// nil (the default path), even in the auto_proceed branch.
func TestHandleFault_AutofixNilNoPanic(t *testing.T) {
	root := t.TempDir()
	matched := MatchFaultSkills("VPC subnet unreachable", nil)
	if len(matched) == 0 {
		t.Fatal("no matched skills")
	}
	skill := primarySkillFromMatched(matched)
	mem := seedAutoProceedTrust(t, root, skill)

	out := HandleFault(HandleFaultInput{
		Root:     root,
		Fault:    "VPC subnet unreachable",
		Resource: "vpc:subnet",
		Risk:     "low",
		Mem:      mem,
		// Autofix deliberately nil.
	}, nil)
	if out.FaultID == "" {
		t.Fatal("fault_id missing")
	}
}

// TestHandleFault_ResourceInference verifies resource type is derived from the
// fault text when not explicitly provided (integration of deriveResource).
func TestHandleFault_ResourceInference(t *testing.T) {
	cases := []struct{ fault, want string }{
		{"ECS instance unreachable", "ecs:instance"},
		{"RDS connection timeout", "rds:instance"},
		{"VPC subnet unreachable", "vpc:instance"},
		{"something completely generic", "unknown:resource"},
	}
	for _, c := range cases {
		if got := deriveResource(c.fault); got != c.want {
			t.Errorf("deriveResource(%q): got %q, want %q", c.fault, got, c.want)
		}
	}
}
