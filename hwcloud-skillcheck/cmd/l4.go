// Package cmd: hwcloud-skillcheck l4 subcommand — closed-loop fault handler.
//
//	hwcloud-skillcheck l4 handle --fault <text> [--root <dir>] [--resource r] [--risk low|medium|high|critical]
//	                           [--skills s1,s2] [--trust-data @path|json] [--metric-values 1,2,3]
//	                           [--metric-threshold 95.0] [--output <path>] [--json]
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/l4"
	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/learning"
)

func runL4(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: hwcloud-skillcheck l4 <handle> ...")
	}
	switch args[0] {
	case "handle":
		return runL4Handle(args[1:])
	case "autofix":
		return runL4Autofix(args[1:])
	case "-h", "--help", "help":
		fmt.Fprint(os.Stderr, l4Help)
		return nil
	default:
		return fmt.Errorf("unknown l4 subcommand %q", args[0])
	}
}

const l4Help = `hwcloud-skillcheck l4 — closed-loop L4 orchestrator

Usage:
  hwcloud-skillcheck l4 handle --fault <text> [--root <dir>] [--resource r] [--risk low|medium|high|critical]
                            [--skills s1,s2] [--trust-data @path|json] [--metric-values 1,2,3]
                            [--metric-threshold 95.0] [--output <path>] [--autofix]
  hwcloud-skillcheck l4 autofix --skill <id> --command <cmd> [--root <dir>] [--dry-run]
`

func runL4Handle(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck l4 handle")
	root := fs.String("root", ".", "repo root")
	fault := fs.String("fault", "", "fault description (e.g. 'RDS connection timeout')")
	resource := fs.String("resource", "", "affected resource (e.g. rds:instance); auto-derived if empty")
	risk := fs.String("risk", "medium", "operation risk: low|medium|high|critical")
	skills := fs.String("skills", "", "comma-separated primary skills")
	trustData := fs.String("trust-data", "", "JSON string or @path with trust history")
	metricValues := fs.String("metric-values", "", "comma-separated numeric series for predictive trend")
	metricThreshold := fs.String("metric-threshold", "", "threshold for breach-time prediction (float)")
	output := fs.String("output", "", "write result to this path")
	autofix := fs.Bool("autofix", false, "attempt autonomous remediation on permanent step failure")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *fault == "" {
		return fmt.Errorf("--fault is required")
	}
	in := l4.HandleFaultInput{
		Root:     *root,
		Fault:    *fault,
		Resource: *resource,
		Risk:     *risk,
	}
	if *autofix {
		in.Autofix = autofixBridger(*root)
	}
	if *skills != "" {
		in.Skills = strings.Split(*skills, ",")
	}
	if *trustData != "" {
		if strings.HasPrefix(*trustData, "@") {
			p := strings.TrimPrefix(*trustData, "@")
			raw, err := os.ReadFile(p)
			if err != nil {
				return fmt.Errorf("read trust-data: %w", err)
			}
			var data map[string]any
			if err := json.Unmarshal(raw, &data); err != nil {
				return fmt.Errorf("parse trust-data: %w", err)
			}
			in.TrustData = data
		} else {
			var data map[string]any
			if err := json.Unmarshal([]byte(*trustData), &data); err != nil {
				return fmt.Errorf("parse trust-data: %w", err)
			}
			in.TrustData = data
		}
	}
	if *metricValues != "" {
		parts := strings.Split(*metricValues, ",")
		vals := make([]float64, 0, len(parts))
		for _, p := range parts {
			v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
			if err != nil {
				return fmt.Errorf("parse metric-values: %w", err)
			}
			vals = append(vals, v)
		}
		in.MetricValues = vals
	}
	if *metricThreshold != "" {
		v, err := strconv.ParseFloat(*metricThreshold, 64)
		if err != nil {
			return fmt.Errorf("parse metric-threshold: %w", err)
		}
		in.MetricThreshold = &v
	}
	out := l4.HandleFault(in, nil)
	buf, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	if *output != "" {
		if err := os.WriteFile(*output, append(buf, '\n'), 0o644); err != nil {
			return err
		}
		fmt.Printf("Wrote: %s\n", *output)
	}
	fmt.Println(string(buf))
	return nil
}

// autofixBridger builds the l4.AutofixFunc hook that connects the L4
// execution loop's permanent-failure branch to internal/learning remediation.
// It loads playbooks, reads the pattern success_rate, executes AutoFix, and
// records the fix outcome back to failure_patterns. Keeps internal/l4 free of
// internal/learning imports (no import cycle).
func autofixBridger(root string) l4.AutofixFunc {
	return func(skill, command string) l4.AutofixResult {
		playbooks, err := learning.LoadPlaybooks(root, skill)
		if err != nil {
			return l4.AutofixResult{Action: "skip_hitl", Error: err.Error()}
		}
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
			AutoExecute:     true,
			DestructiveHITL: true,
			Exec:            l4.NewRealExecutor(),
			RenderOutput:    learning.RenderOutput,
			EvalPrecond: func(joined string, outs map[string]string, run func(string) (int, string, error)) (bool, []string) {
				var preconds []string
				if joined != "" {
					preconds = strings.Split(joined, "\n")
				}
				return learning.EvalPreconditions(preconds, outs, run)
			},
			RecordOutcome: func(r l4.OutcomeRecord) error {
				// r.Skill carries the PLAYBOOK ID (autofix executes a playbook).
				// Write the outcome back to the playbook's metadata.success_rate
				// (playbook-level closed loop); failure de-ranks it so the
				// auto_execute_threshold eventually blocks a flaky fix.
				return learning.RecordPlaybookOutcome(root, skill, r.Skill, r.Outcome == "success")
			},
		}
		return l4.AutoFix(specs, command, cfg)
	}
}
