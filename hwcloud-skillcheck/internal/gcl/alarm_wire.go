// Package gcl provides the Generator-Critic-Loop runtime components for
// hwcloud-skillcheck. alarm_wire.go implements CES alarm plan generation from
// gcl-quality-summary JSON, ported from scripts/gcl_alarm_wire.py.
package gcl

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Default thresholds mirroring gcl_alarm_wire.DEFAULT_THRESHOLDS.
var DefaultThresholds = ThresholdConfig{
	PassRateWarn:     0.85,
	PassRateCritical: 0.70,
	MaxIterWarnCount: 3,
	SafetyFailAlert:  true,
}

const (
	GCLNamespace        = "CUSTOM.GCL"
	GCLPassRateMetric   = "gcl_overall_pass_rate"
	GCLSafetyFailMetric = "gcl_safety_fail_count"
	GCLMaxIterMetric    = "gcl_max_iter_count"
)

// ThresholdConfig holds the SLO threshold values for GCL quality evaluation.
// JSON tags match the Python gcl_alarm_wire.py snake_case contract.
type ThresholdConfig struct {
	PassRateWarn     float64 `json:"pass_rate_warn"`
	PassRateCritical float64 `json:"pass_rate_critical"`
	MaxIterWarnCount int     `json:"max_iter_warn_count"`
	SafetyFailAlert  bool    `json:"safety_fail_alert"`
}

// QualitySummary represents the aggregated GCL quality data from gcl_trace_aggregate.
type QualitySummary struct {
	Totals   map[string]int `json:"totals"`
	PassRate float64        `json:"pass_rate"`
	Skills   map[string]any `json:"skills,omitempty"`
}

// EvaluationResult holds the evaluated SLO status and detected breaches.
type EvaluationResult struct {
	PassRate   float64  `json:"pass_rate"`
	SafetyFail int      `json:"safety_fail"`
	MaxIter    int      `json:"max_iter"`
	Breaches   []Breach `json:"breaches"`
	OK         bool     `json:"ok"`
}

// Breach represents a single SLO threshold breach.
type Breach struct {
	Severity  string `json:"severity"` // "CRITICAL" or "WARN"
	Metric    string `json:"metric"`
	Value     string `json:"value"`
	Threshold string `json:"threshold"`
	Message   string `json:"message"`
}

// AlarmPlanEntry represents one CES alarm rule in the plan.
type AlarmPlanEntry struct {
	Op                 string  `json:"op"`
	Name               string  `json:"name"`
	Namespace          string  `json:"namespace"`
	MetricName         string  `json:"metric_name"`
	ComparisonOperator string  `json:"comparison_operator"`
	Threshold          float64 `json:"threshold"`
	Period             int     `json:"period"`
	EvaluationPeriods  int     `json:"evaluation_periods"`
	Severity           string  `json:"severity"`
	Description        string  `json:"description"`
}

// AlarmPlanReport is the top-level output of the plan command.
type AlarmPlanReport struct {
	GeneratedAt     string           `json:"generated_at"`
	Cloud           string           `json:"cloud"`
	MetricNamespace string           `json:"metric_namespace"`
	SummaryPath     string           `json:"summary_path"`
	Thresholds      ThresholdConfig  `json:"thresholds"`
	Evaluation      EvaluationResult `json:"evaluation"`
	AlarmPlan       []AlarmPlanEntry `json:"alarm_plan"`
}

// LoadThresholdsFromConfig parses gcl_quality thresholds from a YAML config file.
// Mirrors load_thresholds_from_config / load_thresholds_from_config_for_check.
func LoadThresholdsFromConfig(configPath string) (ThresholdConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultThresholds, nil
		}
		return ThresholdConfig{}, fmt.Errorf("read config: %w", err)
	}
	return ParseThresholdsFromYAML(string(data)), nil
}

// ParseThresholdsFromYAML extracts gcl_quality thresholds from arbitrary YAML
// text. Absent keys leave the DefaultThresholds value untouched; unknown keys
// and malformed input are ignored (defaults preserved).
// Mirrors load_thresholds_from_config_for_check in gcl_alarm_wire.py.
func ParseThresholdsFromYAML(text string) ThresholdConfig {
	cfg := DefaultThresholds
	var doc struct {
		GCLQuality struct {
			PassRateWarn     *float64 `yaml:"pass_rate_warn"`
			PassRateCritical *float64 `yaml:"pass_rate_critical"`
			MaxIterWarnCount *int     `yaml:"max_iter_warn_count"`
			SafetyFailAlert  *bool    `yaml:"safety_fail_alert"`
		} `yaml:"gcl_quality"`
	}
	if err := yaml.Unmarshal([]byte(text), &doc); err != nil {
		// Mirror the hand-rolled parser: malformed input silently keeps defaults.
		return DefaultThresholds
	}
	q := doc.GCLQuality
	if q.PassRateWarn != nil {
		cfg.PassRateWarn = *q.PassRateWarn
	}
	if q.PassRateCritical != nil {
		cfg.PassRateCritical = *q.PassRateCritical
	}
	if q.MaxIterWarnCount != nil {
		cfg.MaxIterWarnCount = *q.MaxIterWarnCount
	}
	if q.SafetyFailAlert != nil {
		cfg.SafetyFailAlert = *q.SafetyFailAlert
	}
	return cfg
}

// Evaluate inspects summary against thresholds and returns breaches.
// Mirrors evaluate() in gcl_alarm_wire.py.
func Evaluate(summary QualitySummary, thresholds ThresholdConfig) EvaluationResult {
	passRate := summary.PassRate
	if passRate < 0 {
		passRate = 0
	}
	// A pass rate above 100% is data corruption; clamp to 1.0 (perfect) so a
	// garbage value can't silently skew threshold comparisons.
	if passRate > 1 {
		passRate = 1
	}

	safetyFail := 0
	if t, ok := summary.Totals["SAFETY_FAIL"]; ok {
		safetyFail = t
	}
	maxIter := 0
	if t, ok := summary.Totals["MAX_ITER"]; ok {
		maxIter = t
	}

	var breaches []Breach

	if passRate < thresholds.PassRateCritical {
		breaches = append(breaches, Breach{
			Severity:  "CRITICAL",
			Metric:    "pass_rate",
			Value:     fmt.Sprintf("%.2f", passRate),
			Threshold: fmt.Sprintf("< %.2f", thresholds.PassRateCritical),
			Message:   fmt.Sprintf("GCL pass_rate %.2f below critical %.2f", passRate, thresholds.PassRateCritical),
		})
	} else if passRate < thresholds.PassRateWarn {
		breaches = append(breaches, Breach{
			Severity:  "WARN",
			Metric:    "pass_rate",
			Value:     fmt.Sprintf("%.2f", passRate),
			Threshold: fmt.Sprintf("< %.2f", thresholds.PassRateWarn),
			Message:   fmt.Sprintf("GCL pass_rate %.2f below warn %.2f", passRate, thresholds.PassRateWarn),
		})
	}

	if thresholds.SafetyFailAlert && safetyFail > 0 {
		breaches = append(breaches, Breach{
			Severity:  "CRITICAL",
			Metric:    "safety_fail_count",
			Value:     strconv.Itoa(safetyFail),
			Threshold: "== 0",
			Message:   fmt.Sprintf("GCL observed %d SAFETY_FAIL trace(s)", safetyFail),
		})
	}

	if maxIter > thresholds.MaxIterWarnCount {
		breaches = append(breaches, Breach{
			Severity:  "WARN",
			Metric:    "max_iter_count",
			Value:     strconv.Itoa(maxIter),
			Threshold: fmt.Sprintf("<= %d", thresholds.MaxIterWarnCount),
			Message:   fmt.Sprintf("GCL hit MAX_ITER %d time(s)", maxIter),
		})
	}

	hasCritical := false
	for _, b := range breaches {
		if b.Severity == "CRITICAL" {
			hasCritical = true
			break
		}
	}

	return EvaluationResult{
		PassRate:   passRate,
		SafetyFail: safetyFail,
		MaxIter:    maxIter,
		Breaches:   breaches,
		OK:         !hasCritical,
	}
}

// RenderPlan generates the static SLO alarm rule entries for a GCL deployment.
// The rules are fixed thresholds, not breach-driven — the evaluation result is
// never consulted (callers run Evaluate separately).
// Mirrors render_plan() in gcl_alarm_wire.py.
func RenderPlan(passRateWarn, passRateCritical float64, maxIterWarnCount int) []AlarmPlanEntry {
	return []AlarmPlanEntry{
		{
			Op:                 "create-or-update-alarm-rule",
			Name:               "gcl-overall-pass-rate-critical",
			Namespace:          GCLNamespace,
			MetricName:         GCLPassRateMetric,
			ComparisonOperator: "LT",
			Threshold:          passRateCritical,
			Period:             300,
			EvaluationPeriods:  3,
			Severity:           "CRITICAL",
			Description:        "Fires when GCL pass_rate is below critical threshold.",
		},
		{
			Op:                 "create-or-update-alarm-rule",
			Name:               "gcl-overall-pass-rate-warn",
			Namespace:          GCLNamespace,
			MetricName:         GCLPassRateMetric,
			ComparisonOperator: "LT",
			Threshold:          passRateWarn,
			Period:             300,
			EvaluationPeriods:  3,
			Severity:           "WARN",
			Description:        "Fires when GCL pass_rate is below warning threshold.",
		},
		{
			Op:                 "create-or-update-alarm-rule",
			Name:               "gcl-safety-fail-critical",
			Namespace:          GCLNamespace,
			MetricName:         GCLSafetyFailMetric,
			ComparisonOperator: "GT",
			Threshold:          0,
			Period:             60,
			EvaluationPeriods:  1,
			Severity:           "CRITICAL",
			Description:        "Fires on any GCL SAFETY_FAIL.",
		},
		{
			Op:                 "create-or-update-alarm-rule",
			Name:               "gcl-max-iter-warning",
			Namespace:          GCLNamespace,
			MetricName:         GCLMaxIterMetric,
			ComparisonOperator: "GT",
			Threshold:          float64(maxIterWarnCount),
			Period:             300,
			EvaluationPeriods:  2,
			Severity:           "WARN",
			Description:        "Fires when GCL MAX_ITER count exceeds threshold.",
		},
	}
}

// BuildReport produces a full AlarmPlanReport from a quality summary file path
// and optional config path. Mirrors build_report() + cmd_plan in gcl_alarm_wire.py.
func BuildReport(summaryPath string, configPath string) (*AlarmPlanReport, error) {
	summaryData, err := os.ReadFile(summaryPath)
	if err != nil {
		return nil, fmt.Errorf("read summary: %w", err)
	}
	var summary QualitySummary
	if err := json.Unmarshal(summaryData, &summary); err != nil {
		return nil, fmt.Errorf("parse summary JSON: %w", err)
	}

	thresholds := DefaultThresholds
	if configPath != "" {
		cfg, err := LoadThresholdsFromConfig(configPath)
		if err != nil {
			return nil, fmt.Errorf("load thresholds: %w", err)
		}
		thresholds = cfg
	}

	evaluation := Evaluate(summary, thresholds)
	alarmPlan := RenderPlan(thresholds.PassRateWarn, thresholds.PassRateCritical, thresholds.MaxIterWarnCount)

	return &AlarmPlanReport{
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		Cloud:           "huaweicloud",
		MetricNamespace: GCLNamespace,
		SummaryPath:     summaryPath,
		Thresholds:      thresholds,
		Evaluation:      evaluation,
		AlarmPlan:       alarmPlan,
	}, nil
}

// WritePlan persists an AlarmPlanReport to audit-results/gcl-alarm-plan-<stamp>_<suffix>.json.
// Mirrors write_plan() in gcl_alarm_wire.py.
func WritePlan(report *AlarmPlanReport, auditDir, suffix string) (string, error) {
	if err := os.MkdirAll(auditDir, 0o700); err != nil {
		return "", fmt.Errorf("create audit dir: %w", err)
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal report: %w", err)
	}
	stamp := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("gcl-alarm-plan-%s-%s-%s.json", stamp, suffix, uniqueShortID())
	path := filepath.Join(auditDir, filename)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write plan: %w", err)
	}
	return path, nil
}

// execCommand is the seam ApplyAlarmPlan shells out through. Tests replace it
// with a stub so failure accounting runs hermetically regardless of whether a
// real hcloud CLI is on PATH.
var execCommand = exec.Command

// ApplyAlarmPlan executes a list of alarm plan entries via hcloud ces CLI.
// dryRun=true only writes the plan without executing. Mirrors cmd_apply in gcl_alarm_wire.py.
func ApplyAlarmPlan(plan []AlarmPlanEntry, dryRun bool) error {
	failedCount := 0
	// Keep the first failure's name + output so the aggregate error carries a
	// representative detail snippet for the CLI caller to surface.
	firstFailName, firstFailOut := "", ""
	for _, entry := range plan {
		if dryRun {
			fmt.Printf("[dry-run] would: hcloud ces create-alarm-rule --name %s ...\n", entry.Name)
			continue
		}
		args := []string{
			"ces", "create-alarm-rule",
			"--name", entry.Name,
			"--namespace", entry.Namespace,
			"--metric-name", entry.MetricName,
			"--comparison-operator", entry.ComparisonOperator,
			"--threshold", fmt.Sprintf("%.0f", entry.Threshold),
			"--period", strconv.Itoa(entry.Period),
			"--evaluation-periods", strconv.Itoa(entry.EvaluationPeriods),
		}
		cmd := execCommand("hcloud", args...)
		// Bound a hung hcloud CLI; without this the alarm wire blocks
		// indefinitely. Mirrors the 60s guard in gcl_alarm_wire.py:cmd_apply.
		//
		// The callback must guard against nil Process — cmd.Start can fail
		// (e.g. hcloud not in PATH), in which case cmd.Process is nil and
		// .Kill() panics. The nil check is the difference between a clean
		// "FAILED hcloud-missing" log and a goroutine-driven panic that
		// takes down the orchestrator.
		timer := time.AfterFunc(60*time.Second, func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		})
		out, err := cmd.CombinedOutput()
		timer.Stop()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[apply] FAILED %s: %s\n", entry.Name, string(out))
			if failedCount == 0 {
				firstFailName = entry.Name
				firstFailOut = strings.TrimSpace(string(out))
				if len(firstFailOut) > 200 {
					firstFailOut = firstFailOut[:200] + "..."
				}
			}
			failedCount++
			continue
		}
		fmt.Printf("[apply] OK: %s\n", entry.Name)
	}
	if failedCount > 0 {
		return fmt.Errorf("%d of %d alarm rule(s) failed to apply (e.g. %s: %s)", failedCount, len(plan), firstFailName, firstFailOut)
	}
	return nil
}

// ParseAlarmPlanFromJSON parses a persisted alarm plan JSON file.
func ParseAlarmPlanFromJSON(path string) (*AlarmPlanReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var r AlarmPlanReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return &r, nil
}
