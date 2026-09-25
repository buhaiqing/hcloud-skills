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
		AutoExecute:     true,
		DestructiveHITL: true,
		Exec:            exec,
		RenderOutput:    noopRender,
		RecordOutcome:   func(r OutcomeRecord) error { return nil },
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
			Trigger:       map[string]any{"command_pattern": "command"},
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
			Trigger:       map[string]any{"command_pattern": "command"},
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
			Trigger:       map[string]any{"command_pattern": "command"},
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
	if res.Action != "dry_run" || res.PlaybookID != "ECS-R001" {
		t.Fatalf("got action=%s playbook=%s, want dry_run/ECS-R001", res.Action, res.PlaybookID)
	}
	if res.Threshold != 0.7 || res.SuccessRate != 1.0 {
		t.Fatalf("dry-run metadata threshold=%v success_rate=%v", res.Threshold, res.SuccessRate)
	}
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

func TestAutoFix_UnrelatedQualifyingPlaybookIsNotSelected(t *testing.T) {
	playbooks := []PlaybookSpec{
		{ID: "other", Threshold: 0.5, SuccessRate: 1, Trigger: map[string]any{"command_pattern": "ecs list"}, Execute: "hcloud ECS delete", Verification: "hcloud ECS show", Rollback: "hcloud ECS restore"},
		{ID: "match", Threshold: 0.5, SuccessRate: 1, Trigger: map[string]any{"command_pattern": "rds show"}, Execute: "hcloud RDS show", Verification: "hcloud RDS show | grep ok", Rollback: "hcloud RDS restore"},
	}
	exec := &fakeExecutor{outs: []fakeOutcome{{code: 0}, {code: 0}}}
	res := AutoFix(playbooks, "hcloud RDS show", autoExec(exec))
	if res.PlaybookID != "match" || res.Action != "execute" {
		t.Fatalf("selected %s/%s, want match/execute", res.PlaybookID, res.Action)
	}
}

func TestAutoFix_NoMatchRejects(t *testing.T) {
	exec := &fakeExecutor{}
	playbooks := []PlaybookSpec{{ID: "other", Threshold: 0.5, SuccessRate: 1, Trigger: map[string]any{"command_pattern": "ecs list"}, Execute: "hcloud ECS show", Verification: "hcloud ECS show"}}
	res := AutoFix(playbooks, "hcloud RDS show", autoExec(exec))
	if res.Action != "skip_no_match" || len(exec.calls) != 0 {
		t.Fatalf("got action=%s calls=%v, want skip_no_match with no execution", res.Action, exec.calls)
	}
}

func TestAutoFix_LegacyPlaybookWithoutTriggerRemainsEligible(t *testing.T) {
	exec := &fakeExecutor{outs: []fakeOutcome{{code: 0}, {code: 0}}}
	playbooks := []PlaybookSpec{{
		ID: "legacy", Threshold: 0.5, SuccessRate: 1,
		Execute: "hcloud ECS repair", Verification: "hcloud ECS show | grep ok",
	}}
	res := AutoFix(playbooks, "hcloud ECS repair", autoExec(exec))
	if res.Action != "execute" || !res.Success {
		t.Fatalf("got action=%s success=%v, want legacy playbook execute", res.Action, res.Success)
	}
}

func TestAutoFix_RenderedDestructiveAndRollbackRequireHITL(t *testing.T) {
	playbooks := []PlaybookSpec{{ID: "p", Threshold: 0.5, SuccessRate: 1, Trigger: map[string]any{"command_pattern": "benign"}, Execute: "hcloud ECS {{output.action}}", Verification: "hcloud ECS show", Rollback: "hcloud ECS delete rollback"}}
	exec := &fakeExecutor{}
	cfg := autoExec(exec)
	cfg.Outputs = map[string]string{"action": "delete-server"}
	cfg.RenderOutput = func(tmpl string, outputs map[string]string) (string, bool, error) {
		return strings.ReplaceAll(tmpl, "{{output.action}}", outputs["action"]), true, nil
	}
	res := AutoFix(playbooks, "benign original", cfg)
	if res.Action != "skip_hitl" || len(exec.calls) != 0 {
		t.Fatalf("got action=%s calls=%v, want rendered execute blocked", res.Action, exec.calls)
	}
	playbooks[0].Execute = "hcloud ECS repair"
	playbooks[0].Verification = "hcloud ECS show | grep ok"
	playbooks[0].Rollback = "hcloud ECS {{output.rollback}}"
	exec = &fakeExecutor{outs: []fakeOutcome{{code: 1}, {code: 0}}}
	cfg = autoExec(exec)
	cfg.Outputs = map[string]string{"rollback": "delete-server"}
	cfg.RenderOutput = func(tmpl string, outputs map[string]string) (string, bool, error) {
		return strings.ReplaceAll(tmpl, "{{output.rollback}}", outputs["rollback"]), true, nil
	}
	res = AutoFix(playbooks, "benign original", cfg)
	if res.Action != "execute" || len(exec.calls) != 1 {
		t.Fatalf("got action=%s calls=%v, want destructive rollback blocked", res.Action, exec.calls)
	}
}

func TestAutoFix_MissingVerificationFailsClosed(t *testing.T) {
	playbooks := []PlaybookSpec{{ID: "p", Threshold: 0.5, SuccessRate: 1, Trigger: map[string]any{"command_pattern": "command"}, Execute: "hcloud ECS repair"}}
	exec := &fakeExecutor{}
	res := AutoFix(playbooks, "command", autoExec(exec))
	if res.Action != "skip_hitl" || len(exec.calls) != 0 {
		t.Fatalf("got action=%s calls=%v, want missing verification blocked", res.Action, exec.calls)
	}
}
