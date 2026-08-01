package l4

import (
	"strings"
	"testing"
	"time"
)

// fakeExecutor records Run calls and returns scripted outcomes.
type fakeExecutor struct {
	calls []string
	outs  []fakeOutcome
	idx   int
}

type fakeOutcome struct {
	code int
	out  string
	err  error
}

func (f *fakeExecutor) Run(candidate string, timeout time.Duration) (int, string, error) {
	f.calls = append(f.calls, candidate)
	var o fakeOutcome
	if f.idx < len(f.outs) {
		o = f.outs[f.idx]
		f.idx++
	}
	return o.code, o.out, o.err
}

// noopRender passes the template through unchanged (no placeholders in fixtures).
func noopRender(tmpl string, outputs map[string]string) (string, bool, error) {
	return tmpl, true, nil
}

// autoExec builds an AutofixConfig with execute wired and render stubbed.
func autoExec(exec Executor) AutofixConfig {
	return AutofixConfig{
		AutoExecute:   true,
		Exec:          exec,
		RenderOutput:  noopRender,
		RecordOutcome: func(r OutcomeRecord) error { return nil },
	}
}

// helper playbooks for autofix tests, with a given success_rate applied to all.
func testPlaybooks(rate float64) []PlaybookSpec {
	return []PlaybookSpec{
		{
			ID:            "ECS-R001",
			RiskLevel:     "low",
			Threshold:     0.7,
			SuccessRate:   rate,
			Preconditions: []string{},
			Execute:       "hcloud ECS listFlavors",
			Verification:  "hcloud ECS listFlavors | grep -q ok",
			Rollback:      "hcloud ECS rollback",
			Timeout:       30,
		},
		{
			ID:            "RDS-R004",
			RiskLevel:     "high",
			Threshold:     0.95,
			SuccessRate:   rate,
			Preconditions: []string{},
			Execute:       "hcloud RDS failover",
			Verification:  "hcloud RDS show | grep ACTIVE",
			Rollback:      "hcloud RDS failback",
			Timeout:       60,
		},
	}
}

func TestAutoFix_ExecutesLowRiskPlaybook(t *testing.T) {
	exec := &fakeExecutor{outs: []fakeOutcome{{code: 0, out: "ok"}}}
	res := AutoFix(testPlaybooks(1.0), "command", autoExec(exec))
	if res.Action != "execute" {
		t.Fatalf("action=%q, want execute", res.Action)
	}
	if !res.Success {
		t.Fatalf("expected success, err=%v", res.Error)
	}
	if len(exec.calls) != 2 { // execute + verification
		t.Fatalf("expected 2 exec calls (execute+verify), got %d: %v", len(exec.calls), exec.calls)
	}
}

func TestAutoFix_ThresholdBlocksFirstRun(t *testing.T) {
	// success_rate unset (0.0) < 0.7 threshold → skip (conservative bootstrap).
	exec := &fakeExecutor{}
	res := AutoFix(testPlaybooks(0.0), "command", autoExec(exec))
	if res.Action != "skip_threshold" {
		t.Fatalf("action=%q, want skip_threshold for unset success_rate", res.Action)
	}
	if len(exec.calls) != 0 {
		t.Fatalf("expected 0 exec calls when blocked by threshold, got %v", exec.calls)
	}
}

func TestAutoFix_HighRiskNeedsHighThreshold(t *testing.T) {
	// Only a high-risk playbook (0.95 threshold) is available; its success_rate
	// 0.8 is below it → skip (no auto-failover at 80% confidence).
	exec := &fakeExecutor{}
	playbooks := []PlaybookSpec{
		{
			ID:            "RDS-R004",
			RiskLevel:     "high",
			Threshold:     0.95,
			SuccessRate:   0.8,
			Preconditions: []string{},
			Execute:       "hcloud RDS failover",
			Verification:  "hcloud RDS show | grep ACTIVE",
			Rollback:      "hcloud RDS failback",
			Timeout:       60,
		},
	}
	res := AutoFix(playbooks, "command", autoExec(exec))
	if res.Action != "skip_threshold" {
		t.Fatalf("action=%q, want skip_threshold for high-risk low-confidence", res.Action)
	}
	if len(exec.calls) != 0 {
		t.Fatalf("high-risk below threshold must not exec, got %v", exec.calls)
	}
}

func TestAutoFix_DryRunDoesNotExecute(t *testing.T) {
	exec := &fakeExecutor{}
	cfg := autoExec(exec)
	cfg.AutoExecute = false
	res := AutoFix(testPlaybooks(1.0), "command", cfg)
	if res.Executed {
		t.Fatal("dry-run must not execute")
	}
	if len(exec.calls) != 0 {
		t.Fatalf("dry-run must not call exec, got %v", exec.calls)
	}
}

func TestAutoFix_VerificationFailureRollsBack(t *testing.T) {
	exec := &fakeExecutor{outs: []fakeOutcome{
		{code: 0, out: "ok"},          // execute succeeds
		{code: 1, out: "not enabled"}, // verification fails
		{code: 0, out: "rolled back"}, // rollback
	}}
	res := AutoFix(testPlaybooks(1.0), "command", autoExec(exec))
	if res.Action != "rollback" {
		t.Fatalf("action=%q, want rollback on verification failure", res.Action)
	}
	if res.Success {
		t.Fatal("success must be false when verification fails")
	}
	if len(exec.calls) != 3 {
		t.Fatalf("expected 3 exec calls (execute+verify+rollback), got %d: %v", len(exec.calls), exec.calls)
	}
	if !strings.Contains(exec.calls[2], "rollback") {
		t.Errorf("rollback command not executed: %v", exec.calls)
	}
}
