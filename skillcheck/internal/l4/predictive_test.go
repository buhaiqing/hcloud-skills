package l4

import (
	"testing"
)

// --- predictive_ops ---

func TestLinearRegression_Simple(t *testing.T) {
	pts := []struct{ X, Y float64 }{{0, 0}, {1, 2}, {2, 4}, {3, 6}}
	slope, intercept := LinearRegression(pts)
	if absDiff(slope, 2.0) > 1e-9 {
		t.Errorf("slope=%v, want 2.0", slope)
	}
	if absDiff(intercept, 0.0) > 1e-9 {
		t.Errorf("intercept=%v, want 0.0", intercept)
	}
}

func TestLinearRegression_Empty(t *testing.T) {
	slope, intercept := LinearRegression(nil)
	if slope != 0 || intercept != 0 {
		t.Errorf("empty input should yield (0,0), got (%v,%v)", slope, intercept)
	}
}

func TestExponentialSmoothing_Alpha(t *testing.T) {
	in := []float64{10, 20, 30}
	got := ExponentialSmoothing(in, 0.5)
	if got[0] != 10 {
		t.Errorf("got[0]=%v, want 10", got[0])
	}
	if got[1] != 15 {
		t.Errorf("got[1]=%v, want 15", got[1])
	}
	if got[2] != 22.5 {
		t.Errorf("got[2]=%v, want 22.5", got[2])
	}
}

func TestDetectTrend_Increasing(t *testing.T) {
	values := []float64{10, 20, 30, 40, 50}
	trend := DetectTrend(values)
	if trend.Direction != "increasing" {
		t.Errorf("direction=%q, want increasing", trend.Direction)
	}
	if trend.Slope <= 0 {
		t.Errorf("slope=%v, want >0", trend.Slope)
	}
}

func TestDetectTrend_Stable(t *testing.T) {
	values := []float64{50, 50.1, 49.9, 50, 50}
	trend := DetectTrend(values)
	if trend.Direction != "stable" {
		t.Errorf("direction=%q, want stable (slope ~0)", trend.Direction)
	}
}

func TestDetectTrend_InsufficientData(t *testing.T) {
	trend := DetectTrend([]float64{50})
	if trend.Direction != "insufficient_data" {
		t.Errorf("direction=%q, want insufficient_data", trend.Direction)
	}
}

func TestPredictBreachTime_WillBreach(t *testing.T) {
	values := []float64{40, 50, 60, 70, 80}
	got := PredictBreachTime(values, 100, 1.0)
	if !got.WillBreach {
		t.Fatal("should predict breach with positive slope")
	}
	if got.HoursToBreach == nil || *got.HoursToBreach <= 0 {
		t.Errorf("hours_to_breach should be positive, got %v", got.HoursToBreach)
	}
}

func TestPredictBreachTime_AlreadyBreached(t *testing.T) {
	values := []float64{50, 60, 70, 80, 200}
	got := PredictBreachTime(values, 100, 1.0)
	if !got.WillBreach {
		t.Fatal("should mark already_breached")
	}
	if !got.AlreadyBreached {
		t.Error("already_breached flag missing")
	}
}

func TestPredictBreachTime_Never(t *testing.T) {
	values := []float64{100, 90, 80, 70}
	got := PredictBreachTime(values, 200, 1.0)
	if got.WillBreach {
		t.Error("decreasing trend should not predict breach of higher threshold")
	}
}

func TestGenerateForecast_RiskScore(t *testing.T) {
	values := []float64{50, 60, 70, 80, 85, 90}
	forecast := GenerateForecast("cpu_utilization", values, 1.0)
	if forecast.Metric != "cpu_utilization" {
		t.Errorf("metric=%q", forecast.Metric)
	}
	if forecast.RiskScore < 0 || forecast.RiskScore > 1 {
		t.Errorf("risk_score=%v, out of [0,1]", forecast.RiskScore)
	}
	if forecast.Urgency == "" {
		t.Error("urgency missing")
	}
}

func TestGenerateForecast_UnknownMetric(t *testing.T) {
	forecast := GenerateForecast("custom_metric", []float64{1, 2, 3, 4, 5}, 1.0)
	if forecast.Unit != "unknown" {
		t.Errorf("unit=%q, want 'unknown' for unlisted metric", forecast.Unit)
	}
}

func TestGetRecommendations_Critical(t *testing.T) {
	forecast := Forecast{
		Metric:    "cpu_utilization",
		Urgency:   "critical_now",
		RiskScore: 0.95,
		Trend:     Trend{Direction: "increasing"},
	}
	recs := GetRecommendations(forecast)
	if len(recs.Recommendations) == 0 {
		t.Fatal("critical_now should produce recommendations")
	}
	if recs.Recommendations[0].Priority != "P0" {
		t.Errorf("priority=%q, want P0 for critical", recs.Recommendations[0].Priority)
	}
}

func TestGetRecommendations_Normal(t *testing.T) {
	forecast := Forecast{
		Metric:    "cpu_utilization",
		Urgency:   "normal",
		RiskScore: 0.3,
		Trend:     Trend{Direction: "stable"},
	}
	recs := GetRecommendations(forecast)
	if len(recs.Recommendations) != 0 {
		t.Errorf("normal urgency with stable trend should yield no recs, got %d", len(recs.Recommendations))
	}
}

func absDiff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}
