package l4

import (
	"fmt"
	"strings"
	"time"
)

// PlaybookSpec is the minimal remediation-playbook shape the autofix executor
// needs. It is deliberately decoupled from internal/learning (no import cycle):
// the CLI layer bridges learning.RemediationPlaybook → PlaybookSpec.
type PlaybookSpec struct {
	ID            string
	RiskLevel     string // low | medium | high
	Threshold     float64
	SuccessRate   float64 // this playbook's own learned success_rate (metadata)
	Preconditions []string
	Execute       string
	Verification  string
	Rollback      string
	Timeout       int
}

// NewPlaybookSpec is the canonical constructor; keeps the zero-value unusable.
func NewPlaybookSpec(id string) PlaybookSpec { return PlaybookSpec{ID: id} }

// AutofixConfig carries the executor seam + safety knobs for AutoFix.
type AutofixConfig struct {
	AutoExecute     bool              // master switch; false = dry-run (never exec)
	DestructiveHITL bool              // destructive verb → always HITL
	Exec            Executor          // nil → NewRealExecutor()
	Outputs         map[string]string // captured {{output.*}} values
	EvalPrecond     func(string, map[string]string, func(string) (int, string, error)) (bool, []string)
	RenderOutput    func(string, map[string]string) (string, bool, error)
	RunPrecond      func(string) (int, string, error)
	RecordOutcome   func(OutcomeRecord) error
}

// AutofixResult reports what the autofix loop decided and did.
type AutofixResult struct {
	Action      string // execute | skip_threshold | skip_hitl | dry_run | rollback
	PlaybookID  string
	Executed    bool
	Success     bool
	VerifiedOut string
	Error       string
}

// AutoFix is the L4 autonomous-fix closed loop. Given a candidate set of
// remediation playbooks, it applies the safety gate chain, then (if cleared)
// renders + executes the fix, verifies it, and rolls back on failure.
//
// Gate order (each blocks execution):
//
//  1. dry-run (AutoExecute=false) → Action=dry_run, no exec
//  2. destructive verb (HITL)      → Action=skip_hitl
//  3. success_rate < threshold     → Action=skip_threshold (conservative; 0.0 blocked)
//
// Then: render execute → preconditions → exec → verify → success? record : rollback.
func AutoFix(playbooks []PlaybookSpec, command string, cfg AutofixConfig) AutofixResult {
	if !cfg.AutoExecute {
		return AutofixResult{Action: "dry_run"}
	}
	exec := cfg.Exec
	if exec == nil {
		exec = NewRealExecutor()
	}

	// Pick the first playbook whose own success_rate meets its threshold.
	pb, ok := pickPlaybook(playbooks)
	if !ok {
		return AutofixResult{Action: "skip_threshold"}
	}

	// Destructive verb → HITL (never auto-execute).
	if cfg.DestructiveHITL && isDestructiveCommand(command) {
		return AutofixResult{Action: "skip_hitl", PlaybookID: pb.ID}
	}

	// Render the execute command.
	if cfg.RenderOutput == nil {
		return AutofixResult{Action: "skip_hitl", PlaybookID: pb.ID, Error: "RenderOutput must be provided"}
	}
	rendered, ok, err := cfg.RenderOutput(pb.Execute, cfg.Outputs)
	if err != nil || !ok {
		return AutofixResult{Action: "skip_hitl", PlaybookID: pb.ID, Error: fmt.Sprintf("render execute: %v", err)}
	}

	// Preconditions.
	if len(pb.Preconditions) > 0 {
		if cfg.EvalPrecond == nil {
			return AutofixResult{Action: "skip_hitl", PlaybookID: pb.ID, Error: "EvalPrecond must be provided"}
		}
		run := cfg.RunPrecond
		if run == nil {
			run = func(c string) (int, string, error) { return exec.Run(c, time.Duration(pb.Timeout)*time.Second) }
		}
		ok, _ := cfg.EvalPrecond(strings.Join(pb.Preconditions, "\n"), cfg.Outputs, run)
		if !ok {
			return AutofixResult{Action: "skip_hitl", PlaybookID: pb.ID, Error: "preconditions not met"}
		}
	}

	// Execute the fix.
	timeout := time.Duration(pb.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	code, out, execErr := exec.Run(rendered, timeout)
	res := AutofixResult{Action: "execute", PlaybookID: pb.ID, Executed: true}
	if execErr != nil || code != 0 {
		res.Success = false
		res.Error = fmt.Sprintf("execute failed (code=%d): %v %s", code, execErr, out)
		res = rollbackIfAny(res, exec, pb, cfg)
		recordOutcome(cfg, false, pb.ID, command)
		return res
	}

	// Verify.
	verification := pb.Verification
	if verification == "" {
		res.Success = true
		recordOutcome(cfg, true, pb.ID, command)
		return res
	}
	vCode, vOut, vErr := exec.Run(verification, timeout)
	if vErr == nil && vCode == 0 {
		res.Success = true
		res.VerifiedOut = vOut
		recordOutcome(cfg, true, pb.ID, command)
		return res
	}
	// Verification failed → rollback + record failure.
	res.Success = false
	res.Error = fmt.Sprintf("verification failed (code=%d): %v %s", vCode, vErr, vOut)
	res = rollbackIfAny(res, exec, pb, cfg)
	recordOutcome(cfg, false, pb.ID, command)
	return res
}

// pickPlaybook returns the first playbook whose own success_rate meets its
// auto_execute_threshold. A success_rate of 0.0 (unlearned) blocks all
// (conservative bootstrap).
func pickPlaybook(playbooks []PlaybookSpec) (PlaybookSpec, bool) {
	for _, pb := range playbooks {
		if pb.SuccessRate <= 0 {
			continue
		}
		if pb.SuccessRate >= pb.Threshold {
			return pb, true
		}
	}
	return PlaybookSpec{}, false
}

// rollbackIfAny runs the playbook's rollback command (best-effort) after a
// failed execute or verify, and returns res with Action updated to "rollback"
// when a rollback was issued.
func rollbackIfAny(res AutofixResult, exec Executor, pb PlaybookSpec, cfg AutofixConfig) AutofixResult {
	if pb.Rollback == "" || cfg.RenderOutput == nil {
		return res
	}
	rendered, ok, _ := cfg.RenderOutput(pb.Rollback, cfg.Outputs)
	if !ok {
		return res
	}
	timeout := time.Duration(pb.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	exec.Run(rendered, timeout)
	res.Action = "rollback"
	return res
}

// recordOutcome feeds the learning loop with the fix result.
func recordOutcome(cfg AutofixConfig, success bool, playbookID, command string) {
	if cfg.RecordOutcome == nil {
		return
	}
	outcome := "failure"
	if success {
		outcome = "success"
	}
	_ = cfg.RecordOutcome(OutcomeRecord{
		ID:        newOutcomeID(),
		Timestamp: NowISO(),
		Skill:     playbookID,
		Action:    command,
		Outcome:   outcome,
	})
}

// isDestructiveCommand reports whether the command contains a destructive verb.
func isDestructiveCommand(command string) bool {
	for _, v := range ExtractHighRiskVerbs() {
		if strings.Contains(strings.ToLower(command), strings.ToLower(v)) {
			return true
		}
	}
	return false
}
