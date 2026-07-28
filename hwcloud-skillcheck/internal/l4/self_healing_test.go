package l4

import (
	"strings"
	"testing"
	"time"
)

func TestExtractHighRiskVerbs(t *testing.T) {
	verbs := ExtractHighRiskVerbs()
	want := map[string]bool{"delete": true, "terminate": true, "destroy": true, "drop": true, "remove": true, "rm": true, "del": true}
	if len(verbs) != len(want) {
		t.Fatalf("len mismatch: got %d, want %d (%v)", len(verbs), len(want), verbs)
	}
	for _, v := range verbs {
		if !want[v] {
			t.Errorf("unexpected verb %q", v)
		}
	}
	// regex alternation must include every extracted verb.
	for v := range want {
		ok := false
		for _, re := range HighRiskVerbs {
			if re.MatchString(v) {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("HighRiskVerbs regex does not match extracted verb %q", v)
		}
	}
}

func TestHealingPolicy_IsZero(t *testing.T) {
	if !(HealingPolicy{}).IsZero() {
		t.Error("zero-value HealingPolicy should be IsZero")
	}
	p := HealingPolicy{MinSamples: 5}
	if p.IsZero() {
		t.Error("non-zero HealingPolicy should not be IsZero")
	}
}

func TestIsTransient(t *testing.T) {
	cases := map[string]bool{
		"connection reset by peer":     true,
		"timeout waiting for response": true,
		"token expired":                true,
		"HTTP 401 Unauthorized":        true,
		"rate limit 429":               true,
		"503 service unavailable":      true,
		"permission denied":            false,
		"instance not found":           false,
	}
	for in, want := range cases {
		if got := isTransient(in); got != want {
			t.Errorf("isTransient(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestDefaultHealingPolicy(t *testing.T) {
	p := defaultHealingPolicy()
	if p.MaxRetries != 0 {
		t.Errorf("MaxRetries default = %d, want 0 (no auto-retry until configured)", p.MaxRetries)
	}
	for _, verb := range []string{"delete", "terminate", "drop", "remove"} {
		found := false
		for _, v := range p.DestructiveVerbs {
			if v == verb {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DestructiveVerbs missing %q", verb)
		}
	}
}

func TestHealingDecision_String(t *testing.T) {
	d := HealingDecision{Action: "retry", Reason: "transient"}
	if !strings.Contains(d.Reason, "transient") {
		t.Fatal("reason should mention transient")
	}
}

func TestPreExecHook_EmptyMemoryReturnsProceed(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewOutcomeMemory(dir)
	step := TaskStep{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "list"}
	p := HealingPolicy{FailureRateSkipThreshold: 0.5, MinSamples: 2, LookbackWindow: time.Hour}
	d := PreExecHook(step, mem, p)
	if d.Action != "proceed" {
		t.Fatalf("empty memory: want proceed, got %+v", d)
	}
}

func TestPreExecHook_HighFailureRateSkips(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewOutcomeMemory(dir)
	now := time.Now().UTC().Format(time.RFC3339)
	// 4 failures, 1 success: 80% failure rate, above 0.5 threshold.
	for i, outcome := range []string{"failure", "failure", "failure", "success", "failure"} {
		_ = mem.Record(OutcomeRecord{
			ID: string(rune('a' + i)), Timestamp: now, Skill: "huaweicloud-ecs-ops", Action: "delete-instances", Outcome: outcome,
		})
	}
	step := TaskStep{Step: 1, Skill: "huaweicloud-ecs-ops", Action: "delete-instances"}
	p := HealingPolicy{FailureRateSkipThreshold: 0.5, MinSamples: 5, LookbackWindow: time.Hour}
	d := PreExecHook(step, mem, p)
	if d.Action != "skip" {
		t.Fatalf("want skip, got %+v", d)
	}
	if !strings.Contains(d.Reason, "failure") {
		t.Fatalf("reason should mention failure, got %q", d.Reason)
	}
}

func TestPreExecHook_BelowMinSamplesProceeds(t *testing.T) {
	dir := t.TempDir()
	mem, _ := NewOutcomeMemory(dir)
	now := time.Now().UTC().Format(time.RFC3339)
	for i := 0; i < 2; i++ {
		_ = mem.Record(OutcomeRecord{
			ID: string(rune('a' + i)), Timestamp: now, Skill: "s", Action: "x", Outcome: "failure",
		})
	}
	step := TaskStep{Step: 1, Skill: "s", Action: "x"}
	p := HealingPolicy{FailureRateSkipThreshold: 0.5, MinSamples: 5, LookbackWindow: time.Hour}
	if d := PreExecHook(step, mem, p); d.Action != "proceed" {
		t.Fatalf("want proceed (below min samples), got %+v", d)
	}
}
