package ab

import "testing"

func TestDiff_DetectsStdoutDrift(t *testing.T) {
	old := Result{PerScenario: map[string]string{"a": "x\n"}}
	cur := Result{PerScenario: map[string]string{"a": "y\n"}}
	d := Compare(old, cur)
	if !d.HasDrift("a") {
		t.Errorf("expected drift for scenario 'a', got false")
	}
}

func TestDiff_AllowsAllowlisted(t *testing.T) {
	old := Result{PerScenario: map[string]string{"a": "x\n"}}
	cur := Result{PerScenario: map[string]string{"a": "y\n"}}
	allow := map[string]bool{"a": true}
	d := CompareWith(old, cur, allow)
	if d.HasDrift("a") {
		t.Errorf("expected no drift for allowlisted scenario 'a', got true")
	}
}
func TestABDetectsStdoutDiff(t *testing.T) {
	old := Result{PerScenario: map[string]string{"a": "x\n"}}
	cur := Result{PerScenario: map[string]string{"a": "y\n"}}
	d := Compare(old, cur)
	if !d.HasDrift("a") {
		t.Error("drift should be detected")
	}
}
