package gcl

import (
	"encoding/json"
	"testing"
)

// golden-trace fixtures used to verify Python parity.

func newGen(exit int, durationMs int, out string) GeneratorOutput {
	return GeneratorOutput{
		Command:       "hcloud ecs list-servers",
		ExitCode:      exit,
		ResultExcerpt: out,
		StdoutLen:     len(out),
		StderrLen:     0,
		HasLeak:       false,
		DurationMs:    durationMs,
	}
}

func TestComputeOpsEfficiency_Empty(t *testing.T) {
	got := ComputeOpsEfficiency(GCLTrace{})
	if got.RetryCount != 0 || got.TotalAPICalls != 0 {
		t.Fatalf("empty trace should give zero counters, got %+v", got)
	}
	if got.AutomationLevel != "assisted" {
		t.Fatalf("empty trace default automation = 'assisted', got %q", got.AutomationLevel)
	}
}

func TestComputeOpsEfficiency_SinglePass(t *testing.T) {
	trace := GCLTrace{
		DurationMs: 1000,
		Iterations: []Iteration{
			{Iter: 1, Generator: newGen(0, 1000, "ok"), Critic: CriticResult{Scores: map[string]float64{"safety": 1}}, Decision: "PASS"},
		},
		Final: &FinalResult{Status: "PASS", Iter: 1},
	}
	got := ComputeOpsEfficiency(trace)
	if got.RetryCount != 0 {
		t.Fatalf("retry_count should be 0, got %d", got.RetryCount)
	}
	if got.WastedTimeMs != 0 {
		t.Fatalf("wasted_time_ms should be 0, got %d", got.WastedTimeMs)
	}
	if got.FirstSuccessIter != 1 {
		t.Fatalf("first_success_iter should be 1, got %v", got.FirstSuccessIter)
	}
	if got.TotalAPICalls != 1 {
		t.Fatalf("total_api_calls should be 1, got %d", got.TotalAPICalls)
	}
	if got.AutomationLevel != "full" {
		t.Fatalf("single-PASS automation = 'full', got %q", got.AutomationLevel)
	}
	if got.TotalDurationMs != 1000 {
		t.Fatalf("total_duration_ms = 1000, got %d", got.TotalDurationMs)
	}
}

func TestComputeOpsEfficiency_RetryThenPass(t *testing.T) {
	trace := GCLTrace{
		DurationMs: 5000,
		Iterations: []Iteration{
			{Iter: 1, Generator: newGen(0, 2000, "fail-1"), Critic: CriticResult{Scores: map[string]float64{"safety": 0}}, Decision: "SAFETY_FAIL"},
			{Iter: 2, Generator: newGen(0, 3000, "fail-2"), Critic: CriticResult{Scores: map[string]float64{"safety": 0}}, Decision: "SAFETY_FAIL"},
			{Iter: 3, Generator: newGen(0, 1000, "ok"), Critic: CriticResult{Scores: map[string]float64{"safety": 1}}, Decision: "PASS"},
		},
		Final: &FinalResult{Status: "PASS", Iter: 3},
	}
	got := ComputeOpsEfficiency(trace)
	if got.RetryCount != 2 {
		t.Fatalf("retry_count = 2, got %d", got.RetryCount)
	}
	if got.WastedTimeMs != 5000 {
		t.Fatalf("wasted_time_ms = 2000+3000 = 5000, got %d", got.WastedTimeMs)
	}
	if got.FirstSuccessIter != 3 {
		t.Fatalf("first_success_iter = 3, got %v", got.FirstSuccessIter)
	}
	if got.AutomationLevel != "assisted" {
		t.Fatalf("3-iter PASS automation = 'assisted', got %q", got.AutomationLevel)
	}
}

func TestComputeOpsEfficiency_MaxIter(t *testing.T) {
	trace := GCLTrace{
		DurationMs: 4000,
		Iterations: []Iteration{
			{Iter: 1, Generator: newGen(0, 1000, "x"), Critic: CriticResult{Scores: map[string]float64{"safety": 0}}, Decision: "SAFETY_FAIL"},
			{Iter: 2, Generator: newGen(0, 1000, "x"), Critic: CriticResult{Scores: map[string]float64{"safety": 0}}, Decision: "SAFETY_FAIL"},
		},
		Final: &FinalResult{Status: "MAX_ITER", Iter: 2},
	}
	got := ComputeOpsEfficiency(trace)
	if got.RetryCount != 1 {
		t.Fatalf("retry_count = 1, got %d", got.RetryCount)
	}
	if got.FirstSuccessIter != nil {
		t.Fatalf("first_success_iter must be null on MAX_ITER, got %v", got.FirstSuccessIter)
	}
}

func TestComputeCostAttribution(t *testing.T) {
	trace := GCLTrace{
		DurationMs: 3_600_000, // 1 hour
		Iterations: []Iteration{
			{Iter: 1, Generator: newGen(0, 0, "out-1")},
			{Iter: 2, Generator: newGen(0, 0, "out-2")},
		},
	}
	tok := map[string]any{"estimated_cost_usd": 0.42}
	res := map[string]any{"monthly_cost_usd": 720.0}

	got := ComputeCostAttribution(trace, tok, res)
	if got.CloudAPICalls != 2 {
		t.Fatalf("cloud_api_calls = 2, got %d", got.CloudAPICalls)
	}
	if got.AICostUSD != 0.42 {
		t.Fatalf("ai_cost_usd = 0.42, got %v", got.AICostUSD)
	}
	if got.ResourceCostUSD < 0.99 || got.ResourceCostUSD > 1.01 {
		t.Fatalf("resource_cost_usd ~= 1.0 (720/720 * 1h), got %v", got.ResourceCostUSD)
	}
	expectedTotal := 0.42 + 1.0
	if got.TotalCostUSD < expectedTotal-0.001 || got.TotalCostUSD > expectedTotal+0.001 {
		t.Fatalf("total_cost_usd ~= 1.42, got %v", got.TotalCostUSD)
	}
	if got.CostPerAPICallUSD < 0.20 || got.CostPerAPICallUSD > 0.22 {
		t.Fatalf("cost_per_api_call ~= 0.21, got %v", got.CostPerAPICallUSD)
	}
}

func TestComputeCostAttribution_NoResource(t *testing.T) {
	trace := GCLTrace{DurationMs: 1000}
	got := ComputeCostAttribution(trace, map[string]any{"estimated_cost_usd": 0.5}, nil)
	if got.ResourceCostUSD != 0 {
		t.Fatalf("resource cost without monthly_cost_usd should be 0, got %v", got.ResourceCostUSD)
	}
	if got.TotalCostUSD != 0.5 {
		t.Fatalf("total = 0.5, got %v", got.TotalCostUSD)
	}
}

func TestEnhanceTokenUsage_SingleIter(t *testing.T) {
	tok := map[string]any{"total_tokens": 1000, "estimated_cost_usd": 0.10}
	EnhanceTokenUsage(tok, 1)
	if tok["retry_waste_tokens"] != 0 {
		t.Fatalf("single iter should have 0 retry_waste, got %v", tok["retry_waste_tokens"])
	}
	if tok["retry_waste_cost_usd"] != 0.0 {
		t.Fatalf("single iter retry_waste_cost = 0.0, got %v", tok["retry_waste_cost_usd"])
	}
	if tok["cost_per_iteration_usd"] != 0.10 {
		t.Fatalf("cost_per_iter = 0.10, got %v", tok["cost_per_iteration_usd"])
	}
}

func TestEnhanceTokenUsage_MultiIter(t *testing.T) {
	tok := map[string]any{"total_tokens": 1200, "estimated_cost_usd": 0.30}
	EnhanceTokenUsage(tok, 3)
	// retry_waste_tokens = 1200 * (3-1) / 3 = 800
	if tok["retry_waste_tokens"] != 800 {
		t.Fatalf("retry_waste_tokens = 800, got %v", tok["retry_waste_tokens"])
	}
	// retry_waste_cost = 0.30 * 2/3 = 0.2
	if tok["retry_waste_cost_usd"] != 0.2 {
		t.Fatalf("retry_waste_cost_usd = 0.2, got %v", tok["retry_waste_cost_usd"])
	}
	// cost_per_iter = 0.30 / 3 = 0.1
	if tok["cost_per_iteration_usd"] != 0.1 {
		t.Fatalf("cost_per_iteration_usd = 0.1, got %v", tok["cost_per_iteration_usd"])
	}
}

func TestEnhanceTokenUsage_NilSafe(t *testing.T) {
	EnhanceTokenUsage(nil, 5) // must not panic
}

func TestFinalizeFinopsAiops_EndToEnd(t *testing.T) {
	trace := GCLTrace{
		DurationMs: 1000,
		Iterations: []Iteration{
			{Iter: 1, Generator: newGen(0, 500, "ok")},
		},
		Final:           &FinalResult{Status: "PASS", Iter: 1},
		TokenUsage:      map[string]any{"total_tokens": 100, "estimated_cost_usd": 0.05},
		ResourceContext: map[string]any{"monthly_cost_usd": 100.0},
	}
	FinalizeFinopsAiops(&trace)

	if trace.OpsEfficiency == nil {
		t.Fatal("ops_efficiency should be populated")
	}
	if trace.CostAttribution == nil {
		t.Fatal("cost_attribution should be populated")
	}
	if trace.TokenUsage["retry_waste_tokens"] != 0 {
		t.Fatalf("single-iter retry_waste_tokens = 0, got %v", trace.TokenUsage["retry_waste_tokens"])
	}
	if trace.OpsEfficiency.AutomationLevel != "full" {
		t.Fatalf("single-iter PASS should be 'full', got %q", trace.OpsEfficiency.AutomationLevel)
	}

	// Round-trip JSON to confirm schema is JSON-stable.
	out, err := json.Marshal(trace.OpsEfficiency)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("ops_efficiency JSON marshal produced empty output")
	}
}
