package gcl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseThresholdsFromYAML(t *testing.T) {
	yaml := `
gcl_quality:
  pass_rate_warn: 0.80
  pass_rate_critical: 0.60
  max_iter_warn_count: 5
  safety_fail_alert: false
`
	cfg := ParseThresholdsFromYAML(yaml)
	if cfg.PassRateWarn != 0.80 {
		t.Errorf("PassRateWarn = %.2f, want 0.80", cfg.PassRateWarn)
	}
	if cfg.PassRateCritical != 0.60 {
		t.Errorf("PassRateCritical = %.2f, want 0.60", cfg.PassRateCritical)
	}
	if cfg.MaxIterWarnCount != 5 {
		t.Errorf("MaxIterWarnCount = %d, want 5", cfg.MaxIterWarnCount)
	}
	if cfg.SafetyFailAlert != false {
		t.Errorf("SafetyFailAlert = %v, want false", cfg.SafetyFailAlert)
	}
}

func TestParseThresholdsFromYAML_Defaults(t *testing.T) {
	yaml := `
some_other_key: value
`
	cfg := ParseThresholdsFromYAML(yaml)
	if cfg.PassRateWarn != DefaultThresholds.PassRateWarn {
		t.Errorf("PassRateWarn = %.2f, want default %.2f", cfg.PassRateWarn, DefaultThresholds.PassRateWarn)
	}
	if cfg.PassRateCritical != DefaultThresholds.PassRateCritical {
		t.Errorf("PassRateCritical = %.2f, want default %.2f", cfg.PassRateCritical, DefaultThresholds.PassRateCritical)
	}
}

func TestParseThresholdsFromYAML_InlineComment(t *testing.T) {
	yaml := `gcl_quality:
  pass_rate_warn: 0.90 # this is a comment
  safety_fail_alert: true
`
	cfg := ParseThresholdsFromYAML(yaml)
	if cfg.PassRateWarn != 0.90 {
		t.Errorf("PassRateWarn = %.2f, want 0.90", cfg.PassRateWarn)
	}
	if !cfg.SafetyFailAlert {
		t.Error("SafetyFailAlert = false, want true")
	}
}

func TestEvaluate_PassRateCritical(t *testing.T) {
	summary := QualitySummary{PassRate: 0.50, Totals: map[string]int{"SAFETY_FAIL": 0, "MAX_ITER": 0}}
	thresholds := DefaultThresholds
	result := Evaluate(summary, thresholds)
	if len(result.Breaches) == 0 {
		t.Error("expected breach at pass_rate below critical")
	}
	if result.Breaches[0].Severity != "CRITICAL" {
		t.Errorf("severity = %q, want CRITICAL", result.Breaches[0].Severity)
	}
}

func TestEvaluate_PassRateWarn(t *testing.T) {
	summary := QualitySummary{PassRate: 0.80, Totals: map[string]int{"SAFETY_FAIL": 0, "MAX_ITER": 0}}
	thresholds := DefaultThresholds
	result := Evaluate(summary, thresholds)
	if len(result.Breaches) == 0 {
		t.Error("expected breach at pass_rate below warn")
	}
	if result.Breaches[0].Severity != "WARN" {
		t.Errorf("severity = %q, want WARN", result.Breaches[0].Severity)
	}
}

func TestEvaluate_PassRateOK(t *testing.T) {
	summary := QualitySummary{PassRate: 0.95, Totals: map[string]int{"SAFETY_FAIL": 0, "MAX_ITER": 0}}
	result := Evaluate(summary, DefaultThresholds)
	if !result.OK {
		t.Error("expected OK=true when pass_rate above warn threshold")
	}
	if len(result.Breaches) != 0 {
		t.Errorf("breaches = %d, want 0", len(result.Breaches))
	}
}

func TestEvaluate_SafetyFailCritical(t *testing.T) {
	summary := QualitySummary{PassRate: 0.95, Totals: map[string]int{"SAFETY_FAIL": 1, "MAX_ITER": 0}}
	result := Evaluate(summary, DefaultThresholds)
	if result.OK {
		t.Error("SAFETY_FAIL > 0 should set OK=false (CRITICAL breach)")
	}
	found := false
	for _, b := range result.Breaches {
		if b.Metric == "safety_fail_count" {
			found = true
			if b.Severity != "CRITICAL" {
				t.Errorf("safety_fail severity = %q, want CRITICAL", b.Severity)
			}
		}
	}
	if !found {
		t.Error("expected safety_fail_count breach")
	}
}

func TestEvaluate_SafetyFailDisabled(t *testing.T) {
	summary := QualitySummary{PassRate: 0.95, Totals: map[string]int{"SAFETY_FAIL": 1, "MAX_ITER": 0}}
	thresholds := DefaultThresholds
	thresholds.SafetyFailAlert = false
	result := Evaluate(summary, thresholds)
	found := false
	for _, b := range result.Breaches {
		if b.Metric == "safety_fail_count" {
			found = true
		}
	}
	if found {
		t.Error("safety_fail_alert=false should not produce safety_fail breach")
	}
}

func TestEvaluate_MaxIter(t *testing.T) {
	summary := QualitySummary{PassRate: 0.95, Totals: map[string]int{"SAFETY_FAIL": 0, "MAX_ITER": 5}}
	result := Evaluate(summary, DefaultThresholds) // warn=3
	found := false
	for _, b := range result.Breaches {
		if b.Metric == "max_iter_count" {
			found = true
			if b.Severity != "WARN" {
				t.Errorf("severity = %q, want WARN", b.Severity)
			}
		}
	}
	if !found {
		t.Error("expected max_iter_count breach when MAX_ITER > warn threshold")
	}
}

func TestEvaluate_OK_CriticalBreachPresent(t *testing.T) {
	summary := QualitySummary{PassRate: 0.50, Totals: map[string]int{"SAFETY_FAIL": 0, "MAX_ITER": 0}}
	result := Evaluate(summary, DefaultThresholds)
	if result.OK {
		t.Error("OK should be false when CRITICAL breach exists")
	}
}

func TestRenderPlan(t *testing.T) {
	evaluation := Evaluate(QualitySummary{PassRate: 0.50, Totals: map[string]int{"SAFETY_FAIL": 0, "MAX_ITER": 0}}, DefaultThresholds)
	plan := RenderPlan(evaluation, 0.85, 0.70, 3)
	if len(plan) != 4 {
		t.Fatalf("len(plan) = %d, want 4", len(plan))
	}
	// Verify alarm names are distinct.
	names := make(map[string]bool)
	for _, e := range plan {
		if names[e.Name] {
			t.Errorf("duplicate alarm name: %s", e.Name)
		}
		names[e.Name] = true
	}
}

func TestBuildReport(t *testing.T) {
	// Create a temp summary file.
	tmpDir := t.TempDir()
	summaryPath := filepath.Join(tmpDir, "summary.json")
	summaryJSON := `{"pass_rate":0.75,"totals":{"SAFETY_FAIL":0,"MAX_ITER":0,"PASS":5}}`
	if err := os.WriteFile(summaryPath, []byte(summaryJSON), 0644); err != nil {
		t.Fatalf("write summary: %v", err)
	}

	report, err := BuildReport(summaryPath, "")
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	if report.Cloud != "huaweicloud" {
		t.Errorf("Cloud = %q, want huaweicloud", report.Cloud)
	}
	if report.MetricNamespace != "CUSTOM.GCL" {
		t.Errorf("MetricNamespace = %q, want CUSTOM.GCL", report.MetricNamespace)
	}
	if len(report.AlarmPlan) != 4 {
		t.Errorf("len(AlarmPlan) = %d, want 4", len(report.AlarmPlan))
	}
}

func TestBuildReport_FileNotFound(t *testing.T) {
	_, err := BuildReport("/nonexistent/path/summary.json", "")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestWritePlan(t *testing.T) {
	tmpDir := t.TempDir()
	report := &AlarmPlanReport{
		GeneratedAt:     "2026-07-18T00:00:00Z",
		Cloud:           "huaweicloud",
		MetricNamespace: "CUSTOM.GCL",
		SummaryPath:     "/tmp/summary.json",
		Thresholds:      DefaultThresholds,
		Evaluation:      EvaluationResult{OK: true, Breaches: nil},
		AlarmPlan:       []AlarmPlanEntry{},
	}
	path, err := WritePlan(report, tmpDir, "test")
	if err != nil {
		t.Fatalf("WritePlan: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("plan file not written: %v", err)
	}
}

func TestAlarmPlanReport_JSONRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	report := &AlarmPlanReport{
		GeneratedAt:     "2026-07-18T00:00:00Z",
		Cloud:           "huaweicloud",
		MetricNamespace: "CUSTOM.GCL",
		SummaryPath:     "/tmp/summary.json",
		Thresholds:      DefaultThresholds,
		Evaluation:      EvaluationResult{PassRate: 0.90, OK: true, Breaches: nil},
		AlarmPlan:       RenderPlan(EvaluationResult{PassRate: 0.90, OK: true}, 0.85, 0.70, 3),
	}
	path, err := WritePlan(report, tmpDir, "roundtrip")
	if err != nil {
		t.Fatalf("WritePlan: %v", err)
	}
	loaded, err := ParseAlarmPlanFromJSON(path)
	if err != nil {
		t.Fatalf("ParseAlarmPlanFromJSON: %v", err)
	}
	if loaded.Cloud != "huaweicloud" {
		t.Errorf("Cloud = %q, want huaweicloud", loaded.Cloud)
	}
	if len(loaded.AlarmPlan) != 4 {
		t.Errorf("len(AlarmPlan) = %d, want 4", len(loaded.AlarmPlan))
	}
}
