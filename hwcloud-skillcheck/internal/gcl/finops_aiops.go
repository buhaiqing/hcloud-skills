package gcl

import (
	"encoding/json"
	"math"
)

// FinOps / AIOps derived fields injected into the trace before persistence.
//
// Mirrors scripts/gcl_runner.py:compute_ops_efficiency / compute_cost_attribution /
// enhance_token_usage / _finalize_finops_aiops 1:1 so the persisted trace JSON
// stays compatible with downstream consumers (skillcheck aggregate trace, CES
// alarm wire, historical audit-results/).

// OpsEfficiency is the AIOps efficiency summary written to trace["ops_efficiency"].
type OpsEfficiency struct {
	RetryCount       int    `json:"retry_count"`
	WastedTimeMs     int    `json:"wasted_time_ms"`
	FirstSuccessIter any    `json:"first_success_iter"`
	TotalAPICalls    int    `json:"total_api_calls"`
	AutomationLevel  string `json:"automation_level"`
	TotalDurationMs  int    `json:"total_duration_ms"`
}

// ComputeOpsEfficiency derives AIOps efficiency metrics from completed iterations.
func ComputeOpsEfficiency(trace GCLTrace) OpsEfficiency {
	totalIters := len(trace.Iterations)
	retryCount := totalIters - 1
	if retryCount < 0 {
		retryCount = 0
	}

	var (
		wastedTimeMs  int
		firstSuccessI any
		totalAPICalls int
	)
	for _, it := range trace.Iterations {
		totalAPICalls++
		if it.Decision == "PASS" {
			firstSuccessI = it.Iter
			break
		}
		wastedTimeMs += it.Generator.DurationMs
	}

	finalStatus := "UNKNOWN"
	if trace.Final != nil {
		finalStatus = trace.Final.Status
	}
	automation := "assisted"
	if totalIters == 1 && finalStatus == "PASS" {
		automation = "full"
	}

	return OpsEfficiency{
		RetryCount:       retryCount,
		WastedTimeMs:     wastedTimeMs,
		FirstSuccessIter: firstSuccessI,
		TotalAPICalls:    totalAPICalls,
		AutomationLevel:  automation,
		TotalDurationMs:  trace.DurationMs,
	}
}

// CostAttribution is the FinOps cost breakdown written to trace["cost_attribution"].
type CostAttribution struct {
	CloudAPICalls     int     `json:"cloud_api_calls"`
	AICostUSD         float64 `json:"ai_cost_usd"`
	ResourceCostUSD   float64 `json:"resource_cost_usd"`
	TotalCostUSD      float64 `json:"total_cost_usd"`
	CostPerAPICallUSD float64 `json:"cost_per_api_call_usd"`
}

// ComputeCostAttribution derives FinOps cost attribution from iterations +
// token_usage + resource_context. Pure function.
func ComputeCostAttribution(trace GCLTrace, tokenUsage map[string]any, resourceCtx map[string]any) CostAttribution {
	cloudAPICalls := len(trace.Iterations)

	aiCost := getFloat(tokenUsage, "estimated_cost_usd")

	resourceHourly := 0.0
	if monthlyCost := getFloat(resourceCtx, "monthly_cost_usd"); monthlyCost != 0 {
		resourceHourly = monthlyCost / 720.0
	}
	durationHours := float64(trace.DurationMs) / 3_600_000.0
	resourceCost := round8(resourceHourly * durationHours)
	totalCost := round6(aiCost + resourceCost)

	return CostAttribution{
		CloudAPICalls:     cloudAPICalls,
		AICostUSD:         aiCost,
		ResourceCostUSD:   resourceCost,
		TotalCostUSD:      totalCost,
		CostPerAPICallUSD: round6(aiCost / float64(maxInt(cloudAPICalls, 1))),
	}
}

// EnhanceTokenUsage adds retry_waste and per-iteration cost to token_usage.
// Mutates tokenUsage in place. Safe no-op when tokenUsage is nil.
func EnhanceTokenUsage(tokenUsage map[string]any, totalIters int) {
	if len(tokenUsage) == 0 {
		return
	}
	totalTokens := getFloat(tokenUsage, "total_tokens")
	estCost := getFloat(tokenUsage, "estimated_cost_usd")

	if totalIters > 1 {
		ratio := float64(totalIters-1) / float64(totalIters)
		tokenUsage["retry_waste_tokens"] = int(totalTokens * ratio)
		tokenUsage["retry_waste_cost_usd"] = round6(estCost * ratio)
	} else {
		tokenUsage["retry_waste_tokens"] = 0
		tokenUsage["retry_waste_cost_usd"] = 0.0
	}
	tokenUsage["cost_per_iteration_usd"] = round6(estCost / float64(maxInt(totalIters, 1)))
}

// FinalizeFinopsAiops computes and injects FinOps + AIOps derived fields
// before trace persistence. Mirrors scripts/gcl_runner.py:_finalize_finops_aiops.
//
// Cost attribution is only populated when the caller actually injected
// token_usage / resource_context — otherwise every persisted trace would
// carry a zero-valued cost_attribution block that downstream consumers
// (aggregate FinOps totals, alarm-wire cost breaches) would silently read
// as $0 spend instead of "unknown." Without this guard the migration to Go
// deleted the Python --token-json / --context-json flags without also
// dropping the CostAttribution output, turning the block from "expensive
// but accurate" into "always zero and misleading."
func FinalizeFinopsAiops(trace *GCLTrace) {
	if trace == nil {
		return
	}
	totalIters := len(trace.Iterations)

	// Ops efficiency is always derivable from iterations alone; populate
	// unconditionally so consumers can rely on its presence.
	ops := ComputeOpsEfficiency(*trace)
	trace.OpsEfficiency = &ops

	if len(trace.TokenUsage) > 0 {
		EnhanceTokenUsage(trace.TokenUsage, totalIters)
		if len(trace.ResourceContext) > 0 {
			cost := ComputeCostAttribution(*trace, trace.TokenUsage, trace.ResourceContext)
			trace.CostAttribution = &cost
		}
	}
}

// helpers

func getFloat(m map[string]any, key string) float64 {
	value, ok := m[key]
	if !ok {
		return 0
	}
	switch value := value.(type) {
	case int:
		return float64(value)
	case int8:
		return float64(value)
	case int16:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	case uint:
		return float64(value)
	case uint8:
		return float64(value)
	case uint16:
		return float64(value)
	case uint32:
		return float64(value)
	case uint64:
		return float64(value)
	case float32:
		return float64(value)
	case float64:
		return value
	case json.Number:
		parsed, err := value.Float64()
		if err == nil {
			return parsed
		}
	}
	return 0
}

func round6(value float64) float64 {
	return math.RoundToEven(value*1_000_000) / 1_000_000
}

func round8(value float64) float64 {
	return math.RoundToEven(value*100_000_000) / 100_000_000
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
