package l4

import (
	"strings"
	"testing"
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
