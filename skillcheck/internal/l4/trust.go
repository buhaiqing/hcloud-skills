package l4

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TrustLevel describes a single trust tier.
type TrustLevel struct {
	MinScore     float64
	Confirmation string
	Description  string
	MaxAutoRisk  string
}

// TrustLevels is the static trust-tier table, ordered by MinScore desc.
var TrustLevels = []struct {
	Key string
	Def TrustLevel
}{
	{"L4_autonomous", TrustLevel{MinScore: 0.95, Confirmation: "never", Description: "Proven autonomous capability — full auto including destructive with rollback", MaxAutoRisk: "critical"}},
	{"L3_trusted", TrustLevel{MinScore: 0.8, Confirmation: "never", Description: "Excellent track record — auto-execute all except destructive", MaxAutoRisk: "high"}},
	{"L2_established", TrustLevel{MinScore: 0.6, Confirmation: "critical_only", Description: "Good track record — confirm only critical risk operations", MaxAutoRisk: "medium"}},
	{"L1_provisional", TrustLevel{MinScore: 0.3, Confirmation: "high_risk_only", Description: "Some success history — confirm only high/critical risk operations", MaxAutoRisk: "low"}},
	{"L0_new", TrustLevel{MinScore: 0.0, Confirmation: "always", Description: "New operation, no history — always require human confirmation", MaxAutoRisk: "none"}},
}

// RiskOrder is the numeric ranking for risk levels.
var RiskOrder = map[string]int{"none": 0, "low": 1, "medium": 2, "high": 3, "critical": 4}

// TrustWeights is the component weight table.
var TrustWeights = map[string]float64{
	"success_rate":       0.35,
	"consistency":        0.20,
	"recency":            0.20,
	"complexity_mastery": 0.15,
	"error_recovery":     0.10,
}

// RecencyHalfLifeDays is the exponential decay half-life.
const RecencyHalfLifeDays = 30.0

// OpHistory is a single record in the trust history.
type OpHistory struct {
	Outcome   string `json:"outcome"`
	Timestamp string `json:"timestamp,omitempty"`
	RiskLevel string `json:"risk_level,omitempty"`
	HadRetry  bool   `json:"had_retry,omitempty"`
}

// ComputeSuccessRate is successes / total.
func ComputeSuccessRate(h []OpHistory) float64 {
	if len(h) == 0 {
		return 0
	}
	s := 0
	for _, x := range h {
		if x.Outcome == "success" {
			s++
		}
	}
	return float64(s) / float64(len(h))
}

// ComputeConsistency is 1 - sqrt(variance of 0/1 outcomes).
func ComputeConsistency(h []OpHistory) float64 {
	if len(h) < 3 {
		return 0.5
	}
	out := make([]float64, len(h))
	for i, x := range h {
		if x.Outcome == "success" {
			out[i] = 1
		}
	}
	mean := 0.0
	for _, v := range out {
		mean += v
	}
	mean /= float64(len(out))
	var variance float64
	for _, v := range out {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(out))
	val := 1.0 - math.Sqrt(variance)
	if val < 0 {
		val = 0
	}
	return val
}

// ComputeRecency is time-decayed success rate.
func ComputeRecency(h []OpHistory) float64 {
	if len(h) == 0 {
		return 0
	}
	now := time.Now().UTC()
	wsum := 0.0
	wtot := 0.0
	for _, x := range h {
		var ageDays float64
		if x.Timestamp != "" {
			if t, err := parseISO(x.Timestamp); err == nil {
				ageDays = math.Max(0, now.Sub(t).Hours()/24.0)
			} else {
				ageDays = RecencyHalfLifeDays
			}
		} else {
			ageDays = RecencyHalfLifeDays
		}
		w := math.Exp(-0.693 * ageDays / RecencyHalfLifeDays)
		score := 0.0
		if x.Outcome == "success" {
			score = 1.0
		}
		wsum += w * score
		wtot += w
	}
	return wsum / math.Max(wtot, 1e-10)
}

func parseISO(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, os.ErrInvalid
}

// ComputeComplexityMastery is success rate over high/critical operations.
func ComputeComplexityMastery(h []OpHistory) float64 {
	var complex []OpHistory
	for _, x := range h {
		if x.RiskLevel == "high" || x.RiskLevel == "critical" {
			complex = append(complex, x)
		}
	}
	if len(complex) == 0 {
		return 0.5
	}
	s := 0
	for _, x := range complex {
		if x.Outcome == "success" {
			s++
		}
	}
	return float64(s) / float64(len(complex))
}

// ComputeErrorRecovery is fraction of failures followed by a success.
func ComputeErrorRecovery(h []OpHistory) float64 {
	recoveries := 0
	opportunities := 0
	for i, x := range h {
		if x.Outcome == "failure" && x.HadRetry {
			opportunities++
			if i+1 < len(h) && h[i+1].Outcome == "success" {
				recoveries++
			}
		}
	}
	if opportunities == 0 {
		return 0.7
	}
	return float64(recoveries) / float64(opportunities)
}

// TrustScore is the result of ComputeTrustScore.
type TrustScore struct {
	Score              float64            `json:"score"`
	Level              string             `json:"level"`
	LevelDescription   string             `json:"level_description"`
	ConfirmationPolicy string             `json:"confirmation_policy"`
	MaxAutoRisk        string             `json:"max_auto_risk"`
	Components         map[string]float64 `json:"components"`
	HistorySize        int                `json:"history_size"`
	ComputedAt         string             `json:"computed_at"`
}

// ComputeTrustScore returns the composite trust score + level.
func ComputeTrustScore(h []OpHistory) TrustScore {
	components := map[string]float64{
		"success_rate":       ComputeSuccessRate(h),
		"consistency":        ComputeConsistency(h),
		"recency":            ComputeRecency(h),
		"complexity_mastery": ComputeComplexityMastery(h),
		"error_recovery":     ComputeErrorRecovery(h),
	}
	weighted := 0.0
	for k, w := range TrustWeights {
		weighted += components[k] * w
	}
	if weighted < 0 {
		weighted = 0
	}
	if weighted > 1 {
		weighted = 1
	}
	score := math.Round(weighted*10000) / 10000
	level := "L0_new"
	for _, l := range TrustLevels {
		if score >= l.Def.MinScore {
			level = l.Key
			break
		}
	}
	rounded := map[string]float64{}
	for k, v := range components {
		rounded[k] = math.Round(v*10000) / 10000
	}
	return TrustScore{
		Score:              score,
		Level:              level,
		LevelDescription:   lookupTrust(level).Description,
		ConfirmationPolicy: lookupTrust(level).Confirmation,
		MaxAutoRisk:        lookupTrust(level).MaxAutoRisk,
		Components:         rounded,
		HistorySize:        len(h),
		ComputedAt:         NowISO(),
	}
}

func lookupTrust(key string) TrustLevel {
	for _, l := range TrustLevels {
		if l.Key == key {
			return l.Def
		}
	}
	return TrustLevel{}
}

// EvalResult is the output of EvaluateOperation.
type EvalResult struct {
	OperationType        string  `json:"operation_type"`
	OperationRisk        string  `json:"operation_risk"`
	TrustLevel           string  `json:"trust_level"`
	TrustScore           float64 `json:"trust_score"`
	AutoApproved         bool    `json:"auto_approved"`
	Reason               string  `json:"reason"`
	RequiresConfirmation bool    `json:"requires_confirmation"`
	EvaluatedAt          string  `json:"evaluated_at"`
}

// EvaluateOperation decides whether an operation is auto-approved.
func EvaluateOperation(score TrustScore, opRisk, opType string) EvalResult {
	maxAuto := score.MaxAutoRisk
	riskVal := RiskOrder[opRisk]
	if riskVal == 0 && opRisk != "none" {
		riskVal = 4 // unknown = critical
	}
	maxVal := RiskOrder[maxAuto]
	auto := riskVal <= maxVal
	reason := ""
	if auto {
		reason = "Risk '" + opRisk + "' ≤ max auto '" + maxAuto + "' at trust level " + score.Level
	} else {
		reason = "Risk '" + opRisk + "' > max auto '" + maxAuto + "' — requires confirmation"
	}
	override := ""
	if (opType == "delete" || opType == "terminate" || opType == "destroy") && score.Level != "L4_autonomous" {
		auto = false
		override = "Destructive operation requires L4_autonomous trust level"
	}
	if opRisk == "critical" && score.Score < 0.95 {
		auto = false
		override = "Critical risk requires score ≥ 0.95"
	}
	if override != "" {
		reason = override
	}
	return EvalResult{
		OperationType:        opType,
		OperationRisk:        opRisk,
		TrustLevel:           score.Level,
		TrustScore:           score.Score,
		AutoApproved:         auto,
		Reason:               reason,
		RequiresConfirmation: !auto,
		EvaluatedAt:          NowISO(),
	}
}

// LoadTrustData reads trust_history.json or returns a fresh scaffold.
func LoadTrustData(root, skill string) map[string]any {
	skillID := skill
	if !strings.HasPrefix(skill, "huaweicloud-") {
		skillID = "huaweicloud-" + skill + "-ops"
	}
	path := filepath.Join(root, skillID, "assets", "trust_history.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{
			"schema":     "trust-history/v1",
			"skill_id":   skillID,
			"operations": map[string]any{},
			"meta": map[string]any{
				"created_at":        NowISO(),
				"total_evaluations": 0,
			},
		}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{
			"schema":     "trust-history/v1",
			"skill_id":   skillID,
			"operations": map[string]any{},
			"meta": map[string]any{
				"created_at":        NowISO(),
				"total_evaluations": 0,
			},
		}
	}
	return out
}

// SaveTrustData persists trust_history.json back to disk.
func SaveTrustData(root, skill string, data map[string]any) (string, error) {
	skillID := skill
	if !strings.HasPrefix(skill, "huaweicloud-") {
		skillID = "huaweicloud-" + skill + "-ops"
	}
	dir := filepath.Join(root, skillID, "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(dir, "trust_history.json")
	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(p, append(buf, '\n'), 0o644); err != nil {
		return "", err
	}
	return p, nil
}

// Helpers exposed for orchestrator (alphabetical to avoid sort drift).

func opHistorySlice(v any) []OpHistory {
	raw, _ := v.([]any)
	if raw == nil {
		return nil
	}
	out := make([]OpHistory, 0, len(raw))
	for _, x := range raw {
		if m, ok := x.(map[string]any); ok {
			var h OpHistory
			if s, ok := m["outcome"].(string); ok {
				h.Outcome = s
			}
			if s, ok := m["timestamp"].(string); ok {
				h.Timestamp = s
			}
			if s, ok := m["risk_level"].(string); ok {
				h.RiskLevel = s
			}
			if b, ok := m["had_retry"].(bool); ok {
				h.HadRetry = b
			}
			out = append(out, h)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Timestamp < out[j].Timestamp })
	return out
}
