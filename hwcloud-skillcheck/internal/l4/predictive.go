package l4

import (
	"math"
)

// MetricThreshold defines warning/critical thresholds for a metric.
type MetricThreshold struct {
	Warning  float64
	Critical float64
	Unit     string
}

// MetricThresholds is the per-metric threshold table.
var MetricThresholds = map[string]MetricThreshold{
	"cpu_utilization":        {Warning: 70.0, Critical: 90.0, Unit: "%"},
	"mem_utilization":        {Warning: 75.0, Critical: 92.0, Unit: "%"},
	"disk_utilization":       {Warning: 80.0, Critical: 95.0, Unit: "%"},
	"network_in_rate":        {Warning: 800_000_000, Critical: 950_000_000, Unit: "bps"},
	"network_out_rate":       {Warning: 800_000_000, Critical: 950_000_000, Unit: "bps"},
	"iops_utilization":       {Warning: 70.0, Critical: 90.0, Unit: "%"},
	"connection_utilization": {Warning: 60.0, Critical: 85.0, Unit: "%"},
	"error_rate":             {Warning: 1.0, Critical: 5.0, Unit: "%"},
	"latency_p99":            {Warning: 500.0, Critical: 2000.0, Unit: "ms"},
}

// InterventionPlaybook describes preemptive actions for a metric.
type InterventionPlaybook struct {
	PreemptiveActions []string
	LeadTimeHours     int
}

// InterventionPlaybooks is the metric → playbook map.
var InterventionPlaybooks = map[string]InterventionPlaybook{
	"cpu_utilization":        {PreemptiveActions: []string{"Identify top CPU consumers: hcloud ecs list-server-monitoring --metric cpu", "Right-size recommendation: check if instance is over/under-provisioned", "Schedule scale-out before predicted breach"}, LeadTimeHours: 4},
	"mem_utilization":        {PreemptiveActions: []string{"Check for memory leaks in application processes", "Evaluate if swap configuration is adequate", "Plan instance resize to higher memory tier"}, LeadTimeHours: 6},
	"disk_utilization":       {PreemptiveActions: []string{"Identify large files: du -sh /var/log/* | sort -rh | head", "Rotate/compress old logs", "Expand volume: hcloud evs extend-volume", "Archive cold data to OBS"}, LeadTimeHours: 24},
	"connection_utilization": {PreemptiveActions: []string{"Check connection pool settings", "Identify idle connections for cleanup", "Scale connection limit or add read replicas"}, LeadTimeHours: 2},
	"error_rate":             {PreemptiveActions: []string{"Correlate with recent deployments (CTS event query)", "Check downstream dependency health", "Enable circuit breaker if not active"}, LeadTimeHours: 1},
}

// LinearRegression does least-squares on (X,Y) pairs. Returns (slope, intercept).
// Empty / single point → (0, first_y or 0).
func LinearRegression(pts []struct{ X, Y float64 }) (float64, float64) {
	n := len(pts)
	if n < 2 {
		if n == 1 {
			return 0, pts[0].Y
		}
		return 0, 0
	}
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for _, p := range pts {
		sumX += p.X
		sumY += p.Y
		sumXY += p.X * p.Y
		sumX2 += p.X * p.X
	}
	denom := float64(n)*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-10 {
		return 0, sumY / float64(n)
	}
	slope := (float64(n)*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / float64(n)
	return slope, intercept
}

// ExponentialSmoothing applies single-exponential smoothing.
func ExponentialSmoothing(values []float64, alpha float64) []float64 {
	if len(values) == 0 {
		return nil
	}
	out := make([]float64, len(values))
	out[0] = values[0]
	for i := 1; i < len(values); i++ {
		out[i] = alpha*values[i] + (1-alpha)*out[i-1]
	}
	return out
}

// Trend is the result of DetectTrend.
type Trend struct {
	Direction     string  `json:"direction"`
	Strength      float64 `json:"strength"`
	Slope         float64 `json:"slope"`
	RelativeSlope float64 `json:"relative_slope"`
}

// DetectTrend computes direction + R²-like strength.
func DetectTrend(values []float64) Trend {
	if len(values) < 3 {
		return Trend{Direction: "insufficient_data", Strength: 0, Slope: 0, RelativeSlope: 0}
	}
	pts := make([]struct{ X, Y float64 }, len(values))
	for i, v := range values {
		pts[i] = struct{ X, Y float64 }{float64(i), v}
	}
	slope, _ := LinearRegression(pts)
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	relSlope := slope / math.Max(math.Abs(mean), 1e-10)
	direction := "stable"
	if math.Abs(relSlope) >= 0.01 {
		if relSlope > 0 {
			direction = "increasing"
		} else {
			direction = "decreasing"
		}
	}
	smoothed := ExponentialSmoothing(values, 0.3)
	ssRes, ssTot := 0.0, 0.0
	for i, v := range values {
		ssRes += (v - smoothed[i]) * (v - smoothed[i])
		ssTot += (v - mean) * (v - mean)
	}
	r2 := 1 - ssRes/math.Max(ssTot, 1e-10)
	if r2 < 0 {
		r2 = 0
	}
	return Trend{
		Direction:     direction,
		Strength:      round3(r2),
		Slope:         round4(slope),
		RelativeSlope: round4(relSlope),
	}
}

// BreachForecast is the result of PredictBreachTime.
type BreachForecast struct {
	WillBreach             bool     `json:"will_breach"`
	AlreadyBreached        bool     `json:"already_breached,omitempty"`
	HoursToBreach          *float64 `json:"hours_to_breach,omitempty"`
	Trend                  string   `json:"trend,omitempty"`
	PredictedValueAtBreach float64  `json:"predicted_value_at_breach,omitempty"`
	CurrentValue           float64  `json:"current_value,omitempty"`
	SlopePerHour           float64  `json:"slope_per_hour,omitempty"`
}

// PredictBreachTime returns hours-to-breach for the given threshold.
func PredictBreachTime(values []float64, threshold float64, intervalHours float64) BreachForecast {
	if len(values) == 0 {
		return BreachForecast{WillBreach: false}
	}
	pts := make([]struct{ X, Y float64 }, len(values))
	for i, v := range values {
		pts[i] = struct{ X, Y float64 }{float64(i) * intervalHours, v}
	}
	slope, intercept := LinearRegression(pts)
	current := values[len(values)-1]
	if current >= threshold {
		zero := 0.0
		return BreachForecast{WillBreach: true, AlreadyBreached: true, HoursToBreach: &zero}
	}
	if slope <= 0 {
		return BreachForecast{WillBreach: false, Trend: "stable_or_decreasing"}
	}
	currentTime := pts[len(pts)-1].X
	breachTime := (threshold - intercept) / slope
	htb := breachTime - currentTime
	if htb < 0 {
		return BreachForecast{WillBreach: false}
	}
	return BreachForecast{
		WillBreach:             true,
		HoursToBreach:          &htb,
		PredictedValueAtBreach: threshold,
		CurrentValue:           round2(current),
		SlopePerHour:           round4(slope),
	}
}

// Forecast is the top-level result of GenerateForecast.
type Forecast struct {
	Metric         string          `json:"metric"`
	ForecastedAt   string          `json:"forecasted_at"`
	DataPoints     int             `json:"data_points"`
	CurrentValue   float64         `json:"current_value"`
	Unit           string          `json:"unit"`
	Trend          Trend           `json:"trend"`
	Thresholds     MetricThreshold `json:"thresholds"`
	WarningBreach  BreachForecast  `json:"warning_breach"`
	CriticalBreach BreachForecast  `json:"critical_breach"`
	RiskScore      float64         `json:"risk_score"`
	Urgency        string          `json:"urgency"`
}

// GenerateForecast runs trend + breach + risk-score logic.
func GenerateForecast(metric string, values []float64, intervalHours float64) Forecast {
	th := MetricThresholds[metric]
	if th.Unit == "" {
		th = MetricThreshold{Warning: 80.0, Critical: 95.0, Unit: "unknown"}
	}
	trend := DetectTrend(values)
	wb := PredictBreachTime(values, th.Warning, intervalHours)
	cb := PredictBreachTime(values, th.Critical, intervalHours)
	current := 0.0
	if len(values) > 0 {
		current = values[len(values)-1]
	}
	proximity := 0.0
	if th.Critical > 0 {
		proximity = current / th.Critical
	}
	tf := 0.0
	if trend.Direction == "increasing" {
		tf = trend.Strength
	}
	risk := math.Min(1.0, proximity*0.6+tf*0.4)
	urgency := "normal"
	if cb.AlreadyBreached {
		urgency = "critical_now"
	} else if cb.WillBreach && cb.HoursToBreach != nil && *cb.HoursToBreach < 4 {
		urgency = "critical_imminent"
	} else if wb.WillBreach && wb.HoursToBreach != nil && *wb.HoursToBreach < 12 {
		urgency = "warning_soon"
	} else if risk > 0.7 {
		urgency = "elevated"
	}
	return Forecast{
		Metric:         metric,
		ForecastedAt:   NowISO(),
		DataPoints:     len(values),
		CurrentValue:   round2(current),
		Unit:           th.Unit,
		Trend:          trend,
		Thresholds:     th,
		WarningBreach:  wb,
		CriticalBreach: cb,
		RiskScore:      round3(risk),
		Urgency:        urgency,
	}
}

// Recommendation is one entry in Recommendations.
type Recommendation struct {
	Priority      string   `json:"priority"`
	Action        string   `json:"action"`
	Details       []string `json:"details"`
	LeadTimeHours int      `json:"lead_time_hours"`
}

// Recommendations is the result of GetRecommendations.
type Recommendations struct {
	Metric          string           `json:"metric"`
	Urgency         string           `json:"urgency"`
	RiskScore       float64          `json:"risk_score"`
	Recommendations []Recommendation `json:"recommendations"`
	GeneratedAt     string           `json:"generated_at"`
}

// GetRecommendations returns prioritized actions for a forecast.
func GetRecommendations(f Forecast) Recommendations {
	pb := InterventionPlaybooks[f.Metric]
	out := Recommendations{
		Metric:      f.Metric,
		Urgency:     f.Urgency,
		RiskScore:   f.RiskScore,
		GeneratedAt: NowISO(),
	}
	switch f.Urgency {
	case "critical_now", "critical_imminent":
		details := pb.PreemptiveActions
		if len(details) > 2 {
			details = details[:2]
		}
		if len(details) == 0 {
			details = []string{"Escalate to on-call engineer"}
		}
		out.Recommendations = append(out.Recommendations, Recommendation{
			Priority:      "P0",
			Action:        "Immediate intervention required",
			Details:       details,
			LeadTimeHours: 0,
		})
	case "warning_soon":
		details := pb.PreemptiveActions
		if len(details) == 0 {
			details = []string{"Monitor closely"}
		}
		lead := pb.LeadTimeHours
		if lead == 0 {
			lead = 4
		}
		out.Recommendations = append(out.Recommendations, Recommendation{
			Priority:      "P1",
			Action:        "Schedule proactive intervention",
			Details:       details,
			LeadTimeHours: lead,
		})
	case "elevated":
		out.Recommendations = append(out.Recommendations, Recommendation{
			Priority:      "P2",
			Action:        "Plan capacity review",
			Details:       []string{"Review resource utilization trends", "Evaluate right-sizing options"},
			LeadTimeHours: 24,
		})
	}
	if f.Trend.Direction == "increasing" {
		out.Recommendations = append(out.Recommendations, Recommendation{
			Priority: "P3",
			Action:   "Enhance monitoring",
			Details: []string{
				"Reduce alarm interval for " + f.Metric,
				"Add derivative-based alarm (rate of change)",
			},
			LeadTimeHours: 1,
		})
	}
	return out
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }
func round3(f float64) float64 { return math.Round(f*1000) / 1000 }
func round4(f float64) float64 { return math.Round(f*10000) / 10000 }
