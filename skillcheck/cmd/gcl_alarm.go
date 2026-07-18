package cmd

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/gcl"
)

// runGCLAlarmWire implements `skillcheck gcl alarm-wire`.
// It evaluates GCL trace quality against SLO thresholds and optionally
// generates and applies a CES alarm plan.
func runGCLAlarmWire(args []string) error {
	fs := newFlagSet("skillcheck gcl alarm-wire")
	root := fs.String("root", ".", "repository root")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	quiet := fs.Bool("quiet", false, "suppress stdout except summary")
	planFile := fs.String("plan-file", "", "write alarm plan JSON to path (implies --write-plan)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil // help was shown; exit cleanly
		}
		return err
	}

	repoRoot, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	// Load CES example-config for threshold defaults.
	cesConfigPath := filepath.Join(repoRoot, cesConfigRelative)
	thresholds, err := gcl.LoadThresholdsFromConfig(cesConfigPath)
	if err != nil {
		return fmt.Errorf("load thresholds: %w", err)
	}

	// Find most recent gcl-trace-*.json in audit-results/.
	auditDir := filepath.Join(repoRoot, "audit-results")
	paths, err := filepath.Glob(filepath.Join(auditDir, "gcl-trace-*.json"))
	if err != nil {
		return fmt.Errorf("glob traces: %w", err)
	}
	sort.Strings(paths)

	var tracePath string
	if len(paths) > 0 {
		tracePath = paths[len(paths)-1]
	}

	var summary gcl.QualitySummary
	if tracePath != "" {
		summary, err = loadQualitySummaryFromTrace(tracePath)
		if err != nil {
			return fmt.Errorf("parse trace %s: %w", tracePath, err)
		}
	} else {
		// No traces: use zero values.
		summary = gcl.QualitySummary{
			Totals:   map[string]int{"PASS": 0, "SAFETY_FAIL": 0, "MAX_ITER": 0, "total_runs": 0},
			PassRate: 0.0,
		}
	}

	evaluation := gcl.Evaluate(summary, thresholds)

	// Build alarm plan entries.
	alarmPlan := gcl.RenderPlan(
		evaluation,
		thresholds.PassRateWarn,
		thresholds.PassRateCritical,
		thresholds.MaxIterWarnCount,
	)

	var alarmPlanPath string
	if *planFile != "" {
		report := &gcl.AlarmPlanReport{
			GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
			Cloud:           "huaweicloud",
			MetricNamespace: gcl.GCLNamespace,
			SummaryPath:     tracePath,
			Thresholds:      thresholds,
			Evaluation:      evaluation,
			AlarmPlan:       alarmPlan,
		}
		alarmPlanPath, err = gcl.WritePlan(report, auditDir, "alarm-wire")
		if err != nil {
			return fmt.Errorf("write plan: %w", err)
		}
		if !*quiet {
			fmt.Printf("Wrote alarm plan to %s\n", alarmPlanPath)
		}

		// Apply the alarm plan (dry-run by default; remove --dry-run to apply for real).
		if err := gcl.ApplyAlarmPlan(alarmPlan, true); err != nil {
			return fmt.Errorf("apply alarm plan: %w", err)
		}
	}

	if *quiet {
		return nil
	}

	if *jsonOut {
		printGCLAlarmJSON(evaluation, alarmPlan, alarmPlanPath)
	} else {
		printGCLAlarmHuman(evaluation, alarmPlan, alarmPlanPath, tracePath)
	}

	// Exit 0 on OK (no critical breaches), exit 1 on alert/warning.
	if evaluation.OK {
		return nil
	}
	os.Exit(1)
	return nil // unreachable
}

// loadQualitySummaryFromTrace parses a single gcl-trace-*.json file and
// returns a QualitySummary with totals and pass_rate.
func loadQualitySummaryFromTrace(tracePath string) (gcl.QualitySummary, error) {
	data, err := os.ReadFile(tracePath)
	if err != nil {
		return gcl.QualitySummary{}, err
	}
	var trace map[string]any
	if err := json.Unmarshal(data, &trace); err != nil {
		return gcl.QualitySummary{}, fmt.Errorf("decode JSON: %w", err)
	}

	totals := map[string]int{"PASS": 0, "SAFETY_FAIL": 0, "MAX_ITER": 0, "total_runs": 1}
	status := "UNKNOWN"
	if fin, ok := trace["final"].(map[string]any); ok {
		if s, ok := fin["status"].(string); ok {
			status = s
		}
	}
	if _, ok := totals[status]; ok {
		totals[status] = 1
	} else {
		// Treat unknown status as MAX_ITER.
		totals["MAX_ITER"] = 1
	}

	passRate := 0.0
	if status == "PASS" {
		passRate = 1.0
	}

	return gcl.QualitySummary{
		Totals:   totals,
		PassRate: passRate,
	}, nil
}

func printGCLAlarmHuman(evaluation gcl.EvaluationResult, plan []gcl.AlarmPlanEntry, planPath, tracePath string) {
	if tracePath != "" {
		fmt.Printf("Latest trace: %s\n", tracePath)
	}
	if evaluation.OK {
		fmt.Println("OK: no critical breaches")
	} else {
		fmt.Println("ALERT: threshold breaches detected:")
		for _, b := range evaluation.Breaches {
			fmt.Printf("  [%s] %s — %s (threshold: %s)\n", b.Severity, b.Metric, b.Message, b.Threshold)
		}
	}
	fmt.Printf("Pass rate: %.2f  Safety fails: %d  Max iter: %d\n",
		evaluation.PassRate, evaluation.SafetyFail, evaluation.MaxIter)
	if planPath != "" {
		fmt.Printf("Alarm plan: %s (%d rules)\n", planPath, len(plan))
	}
}

func printGCLAlarmJSON(evaluation gcl.EvaluationResult, plan []gcl.AlarmPlanEntry, planPath string) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(map[string]any{
		"ok":              evaluation.OK,
		"pass_rate":       evaluation.PassRate,
		"safety_fail":     evaluation.SafetyFail,
		"max_iter":        evaluation.MaxIter,
		"breaches":        evaluation.Breaches,
		"alarm_plan":      plan,
		"alarm_plan_path": planPath,
	})
}
