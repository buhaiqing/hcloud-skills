// Package cmd: hwcloud-skillcheck l4 autofix — autonomous remediation executor.
//
//	hwcloud-skillcheck l4 autofix --skill <id> --command <cmd> [--root <dir>] [--dry-run]
//
// This is the ONLY package that imports both internal/learning and internal/l4,
// bridging the pure-data layer (learning) with the orchestration layer (l4)
// without creating an import cycle.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/l4"
	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/learning"
)

func runL4Autofix(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck l4 autofix")
	root := fs.String("root", ".", "repo root")
	skill := fs.String("skill", "", "skill id (e.g. huaweicloud-ecs-ops)")
	command := fs.String("command", "", "the failed command to remediate")
	dryRun := fs.Bool("dry-run", false, "print the would-be remediation without executing")
	output := fs.String("output", "", "write autofix result JSON to this path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *skill == "" || *command == "" {
		return fmt.Errorf("--skill and --command are required")
	}

	// Load remediation playbooks for the skill.
	playbooks, err := learning.LoadPlaybooks(*root, *skill)
	if err != nil {
		return fmt.Errorf("load playbooks: %w", err)
	}

	outputs := map[string]string{} // captured {{output.*}} values (from trace; empty for CLI)

	// Bridge learning.RemediationPlaybook → l4.PlaybookSpec.
	specs := make([]l4.PlaybookSpec, 0, len(playbooks))
	for _, pb := range playbooks {
		specs = append(specs, l4.PlaybookSpec{
			ID:            pb.ID,
			RiskLevel:     pb.Remediation.RiskLevel,
			Threshold:     pb.Remediation.AutoExecuteThreshold,
			SuccessRate:   playbookSuccessRate(pb),
			Preconditions: pb.Remediation.Preconditions,
			Execute:       pb.Remediation.Execute,
			Verification:  pb.Remediation.Verification,
			Rollback:      pb.Remediation.Rollback,
			Timeout:       pb.Remediation.TimeoutSeconds,
		})
	}

	cfg := l4.AutofixConfig{
		AutoExecute:     !*dryRun,
		DestructiveHITL: true,
		Exec:            l4.NewRealExecutor(),
		Outputs:         outputs,
		RenderOutput:    learning.RenderOutput,
		// Adapter: l4 passes a single joined string; learning expects []string.
		EvalPrecond: func(joined string, outs map[string]string, run func(string) (int, string, error)) (bool, []string) {
			var preconds []string
			if joined != "" {
				preconds = strings.Split(joined, "\n")
			}
			return learning.EvalPreconditions(preconds, outs, run)
		},
		RecordOutcome: func(r l4.OutcomeRecord) error {
			// r.Skill carries the PLAYBOOK ID; write the fix outcome back to the
			// playbook's metadata.success_rate (playbook-level closed loop).
			return learning.RecordPlaybookOutcome(*root, *skill, r.Skill, r.Outcome == "success")
		},
	}

	res := l4.AutoFix(specs, *command, cfg)

	// Emit result (JSON if --output, else human).
	if *output != "" {
		raw, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(*output, raw, 0o644)
	}
	if *dryRun {
		fmt.Printf("DRY-RUN autofix for %s\n  command: %s\n  would:    %s (threshold %.2f)\n",
			*skill, *command, res.Action, playbookThreshold(specs))
		return nil
	}
	fmt.Printf("autofix[%s]: action=%s playbook=%s success=%v\n", *skill, res.Action, res.PlaybookID, res.Success)
	if res.Error != "" {
		fmt.Fprintf(os.Stderr, "  %s\n", res.Error)
	}
	return nil
}

// playbookSuccessRate returns a playbook's learned success_rate from its
// metadata (0.0 = unlearned → autofix blocks).
func playbookSuccessRate(pb learning.RemediationPlaybook) float64 {
	if pb.Metadata == nil {
		return 0.0
	}
	if r, ok := pb.Metadata["success_rate"].(float64); ok {
		return r
	}
	return 0.0
}

// playbookThreshold reports the max auto_execute_threshold among playbooks
// (for dry-run display).
func playbookThreshold(specs []l4.PlaybookSpec) float64 {
	var max float64
	for _, s := range specs {
		if s.Threshold > max {
			max = s.Threshold
		}
	}
	return max
}
