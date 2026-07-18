// Package gcl provides the Generator-Critic-Loop runtime components for
// skillcheck. alarm_wire.go implements CES alarm plan generation from
// gcl-quality-summary JSON, ported from scripts/gcl_alarm_wire.py.
package gcl

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Default thresholds mirroring gcl_alarm_wire.DEFAULT_THRESHOLDS.
var DefaultThresholds = ThresholdConfig{
	PassRateWarn:      0.85,
	PassRateCritical:  0.70,
	MaxIterWarnCount:  3,
	SafetyFailAlert:   true,
}

const (
	GCLNamespace         = "CUSTOM.GCL"
	GCLPassRateMetric    = "gcl_overall_pass_rate"
	GCLSafetyFailMetric  = "gcl_safety_fail_count"
	GCLMaxIterMetric     = "gcl_max_iter_count"
)

// ThresholdConfig holds the SLO threshold values for GCL quality evaluation.
type ThresholdConfig struct {
	PassRateWarn      float64
	PassRateCritical  float64
	MaxIterWarnCount  int
	SafetyFailAlert   bool
}

// QualitySummary represents the aggregated GCL quality data from gcl_trace_aggregate.
type QualitySummary struct {
	Totals   map[string]int     `json:"totals"`
	PassRate float64            `json:"pass_rate"`
	Skills   map[string]any     `json:"skills,omitempty"`
}

// EvaluationResult holds the evaluated SLO status and detected breaches.
type EvaluationResult struct {
	PassRate  float64     `json:"pass_rate"`
	SafetyFail int        `json:"safety_fail"`
	MaxIter   int         `json:"max_iter"`
	Breaches  []Breach    `json:"breaches"`
	OK        bool        `json:"ok"`
}

// Breach represents a single SLO threshold breach.
type Breach struct {
	Severity  string `json:"severity"`   // "CRITICAL" or "WARN"
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
	GeneratedAt   string            `json:"generated_at"`
	Cloud         string            `json:"cloud"`
	MetricNamespace string          `json:"metric_namespace"`
	SummaryPath   string            `json:"summary_path"`
	Thresholds    ThresholdConfig   `json:"thresholds"`
	Evaluation    EvaluationResult  `json:"evaluation"`
	AlarmPlan     []AlarmPlanEntry  `json:"alarm_plan"`
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

// ParseThresholdsFromYAML extracts gcl_quality thresholds from arbitrary YAML text.
// Mirrors load_thresholds_from_config_for_check in gcl_alarm_wire.py.
func ParseThresholdsFromYAML(text string) ThresholdConfig {
	cfg := DefaultThresholds
	inBlock := false
	for _, line := range strings.Split(text, "\n") {
		stripped := strings.TrimSpace(line)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		if strings.HasPrefix(stripped, "gcl_quality:") {
			inBlock = true
			continue
		}
		if inBlock {
			// Block ends when we hit a top-level key that's not indented.
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && strings.Contains(stripped, ":") {
				break
			}
			if !strings.Contains(stripped, ":") {
				continue
			}
			parts := strings.SplitN(stripped, ":", 2)
			if len(parts) < 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			raw := strings.TrimSpace(parts[1])
			// Strip inline comments.
			if idx := strings.Index(raw, "#"); idx >= 0 {
				raw = strings.TrimSpace(raw[:idx])
			}
			switch key {
			case "pass_rate_warn":
				if f, err := strconv.ParseFloat(raw, 64); err == nil {
					cfg.PassRateWarn = f
				}
			case "pass_rate_critical":
				if f, err := strconv.ParseFloat(raw, 64); err == nil {
					cfg.PassRateCritical = f
				}
			case "max_iter_warn_count":
				if i, err := strconv.Atoi(raw); err == nil {
					cfg.MaxIterWarnCount = i
				}
			case "safety_fail_alert":
				cfg.SafetyFailAlert = raw == "true"
			}
		}
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

// RenderPlan generates CES alarm rule entries from an evaluation result.
// Mirrors render_plan() in gcl_alarm_wire.py.
func RenderPlan(evaluation EvaluationResult, passRateWarn, passRateCritical float64, maxIterWarnCount int) []AlarmPlanEntry {
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
	alarmPlan := RenderPlan(evaluation, thresholds.PassRateWarn, thresholds.PassRateCritical, thresholds.MaxIterWarnCount)

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
	filename := fmt.Sprintf("gcl-alarm-plan-%s-%s.json", stamp, suffix)
	path := filepath.Join(auditDir, filename)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write plan: %w", err)
	}
	return path, nil
}

// ApplyAlarmPlan executes a list of alarm plan entries via hcloud ces CLI.
// dryRun=true only writes the plan without executing. Mirrors cmd_apply in gcl_alarm_wire.py.
func ApplyAlarmPlan(plan []AlarmPlanEntry, dryRun bool) error {
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
		cmd := exec.Command("hcloud", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[apply] FAILED %s: %s\n", entry.Name, string(out))
			return fmt.Errorf("apply %s: %w", entry.Name, err)
		}
		fmt.Printf("[apply] OK: %s\n", entry.Name)
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

// ValidateThresholdsContract checks that hard-coded constants in alarm_wire.go
// match the values declared in example-config.yaml and docs/gcl-spec.md.
var _thresholdDocRe = regexp.MustCompile(`(?i)(?:pass_rate_warn|pass_rate_critical|max_iter_warn_count|safety_fail_alert)\s*[:=]\s*(\d+\.?\d*)`)
