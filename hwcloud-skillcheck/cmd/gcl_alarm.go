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

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/gcl"
)

// runGCLAlarmWire implements `hwcloud-skillcheck gcl alarm-wire`.
// It evaluates GCL trace quality against SLO thresholds and optionally
// generates and applies a CES alarm plan.
//
// Apply is GATED to prevent accidental production mutations:
//
//	--apply changes dry-run to real hcloud calls (still per-rule 60s)
//	--yes  bypasses the interactive confirmation prompt for CI
//	--target env  refuses the call unless HW_REGION_ID matches (CI safety)
func runGCLAlarmWire(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck gcl alarm-wire")
	root := fs.String("root", ".", "repository root")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	quiet := fs.Bool("quiet", false, "suppress stdout except summary")
	planFile := fs.String("plan-file", "", "write alarm plan JSON to path (implies --write-plan)")
	apply := fs.Bool("apply", false, "execute the alarm plan via `hcloud ces create-alarm-rule` instead of dry-run. Requires --yes if running interactively.")
	yes := fs.Bool("yes", false, "skip the interactive 'apply N alarm rules?' confirmation; mandatory for non-TTY CI usage with --apply.")
	applyRegion := fs.String("apply-target-region", os.Getenv("HW_REGION_ID"), "refuse --apply unless HW_REGION_ID matches this value. Defaults to $HW_REGION_ID.")
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

	// Apply safety gate. Without --apply, ApplyAlarmPlan is invoked
	// dryRun=true regardless of any other state. With --apply, we
	// require an explicit env-target match + non-TTY acknowledgement.
	if *apply {
		envRegion := os.Getenv("HW_REGION_ID")
		if envRegion == "" || *applyRegion == "" || envRegion != *applyRegion {
			return fmt.Errorf("--apply refused: HW_REGION_ID=%q does not match --apply-target-region=%q (set both to the same region to acknowledge the production blast radius)", envRegion, *applyRegion)
		}
		if !*yes && !isInteractive(os.Stdin) {
			return fmt.Errorf("--apply refuses to run non-interactively without --yes (would make `hcloud ces create-alarm-rule` calls against %s)", envRegion)
		}
		if !*quiet {
			fmt.Fprintf(os.Stderr, "APPLY: about to create %d alarm rule(s) against region %s via `hcloud ces create-alarm-rule`. Ctrl-C within 5s to abort.\n", len(alarmPlan), envRegion)
		}
		if !*yes {
			// Best-effort grace period; not safety-critical, just
			// gives the operator a moment.
			time.Sleep(5 * time.Second)
		}
	}

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

		// Apply the alarm plan. dryRun=false iff --apply was passed
		// (and survived the safety gate above).
		if err := gcl.ApplyAlarmPlan(alarmPlan, !*apply); err != nil {
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

// isInteractive reports whether the given file descriptor is connected
// to a real terminal. Used to decide whether --apply's confirmation
// gate can wait for a human keypress. CI is non-TTY → isInteractive
// returns false → --apply demands --yes regardless of stdin mode.
func isInteractive(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
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
