package l4

import (
	"math"
	"os"
	"sync/atomic"
	"time"
)

// trustOutcomeMaxRecords bounds the read window for outcome-memory backed
// trust scoring (covers ~MinSamples*10 without scanning unbounded history).
const trustOutcomeMaxRecords = 100

// ComputeTrustScoreFromOutcome is the Phase 4 canonical scoring entry
// (ADR-0009 §Migration). It reads OutcomeRecord history for (skill,
// action) and returns the composite TrustScore computed directly from
// those records — no OpHistory shim, no curator back-fill. policyHash
// is reserved for cache invalidation across HealingPolicy changes
// (ADR-0009 §Consequences); ignored today.
func ComputeTrustScoreFromOutcome(skill, action string, mem *OutcomeMemory, policyHash string) TrustScore {
	if mem == nil {
		return zeroTrustScore()
	}
	recent, err := mem.RecentOutcomes(skill, action, trustOutcomeMaxRecords)
	if err != nil || len(recent) < 1 {
		return zeroTrustScore()
	}
	return computeTrustScore(recent)
}

// computeTrustScore is the per-record scoring core. Single-pass-ish:
// five components share the same input slice, but each needs a
// different accumulation pattern (variance needs the mean first; the
// error_recovery scan needs the original index). Keep them as plain
// loops instead of helper functions — this is the only caller and
// inlining saves one slice header + one frame per component.
func computeTrustScore(records []OutcomeRecord) TrustScore {
	if len(records) == 0 {
		return zeroTrustScore()
	}

	// success_rate + first pass for consistency's mean.
	successes := 0
	for _, r := range records {
		if isSuccess(r) {
			successes++
		}
	}
	successRate := float64(successes) / float64(len(records))

	// consistency: 1 - sqrt(variance of 0/1 outcomes). Two passes by
	// design — variance needs the mean first.
	consistency := 0.5
	if len(records) >= 3 {
		mean := successRate
		var variance float64
		for _, r := range records {
			d := -mean
			if isSuccess(r) {
				d = 1.0 - mean
			}
			variance += d * d
		}
		variance /= float64(len(records))
		val := 1.0 - math.Sqrt(variance)
		if val < 0 {
			val = 0
		}
		consistency = val
	}

	// recency: time-decayed success rate (30-day half-life per
	// RecencyHalfLifeDays).
	now := time.Now().UTC()
	wsum := 0.0
	wtot := 0.0
	for _, r := range records {
		var ageDays float64
		if r.Timestamp != "" {
			if t, err := parseISO(r.Timestamp); err == nil {
				ageDays = math.Max(0, now.Sub(t).Hours()/24.0)
			} else {
				ageDays = RecencyHalfLifeDays
			}
		} else {
			ageDays = RecencyHalfLifeDays
		}
		w := math.Exp(-0.693 * ageDays / RecencyHalfLifeDays)
		score := 0.0
		if isSuccess(r) {
			score = 1.0
		}
		wsum += w * score
		wtot += w
	}
	recency := wsum / math.Max(wtot, 1e-10)

	// complexity_mastery: success rate over high/critical records.
	n := 0
	s := 0
	for _, r := range records {
		if r.Risk == "high" || r.Risk == "critical" {
			n++
			if isSuccess(r) {
				s++
			}
		}
	}
	complexityMastery := 0.5
	if n > 0 {
		complexityMastery = float64(s) / float64(n)
	}

	// error_recovery: fraction of (failed + retried) records that are
	// immediately followed by a success.
	recoveries := 0
	opportunities := 0
	for i, r := range records {
		if !isSuccess(r) && r.RetryCount > 0 {
			opportunities++
			if i+1 < len(records) && isSuccess(records[i+1]) {
				recoveries++
			}
		}
	}
	errorRecovery := 0.7
	if opportunities > 0 {
		errorRecovery = float64(recoveries) / float64(opportunities)
	}

	components := map[string]float64{
		"success_rate":       successRate,
		"consistency":        consistency,
		"recency":            recency,
		"complexity_mastery": complexityMastery,
		"error_recovery":     errorRecovery,
	}
	weighted := 0.0
	for k, w := range TrustWeights {
		weighted += components[k] * w
	}
	return finalizeTrustScore(weighted, components, len(records))
}

// isSuccess: only literal "success" counts; everything else (failure,
// blocked, unknown) is treated as not-success. Per ADR-0009 §"Mapping
// OutcomeRecord → trust inputs", RBAC-denied ("blocked") is a bad
// outcome and is excluded from success counts here.
func isSuccess(r OutcomeRecord) bool {
	return r.Outcome == "success"
}

// zeroTrustScore returns the empty-history score (was
// ComputeTrustScore(nil) pre-Phase-4).
func zeroTrustScore() TrustScore {
	components := map[string]float64{
		"success_rate":       0,
		"consistency":        0.5,
		"recency":            0,
		"complexity_mastery": 0.5,
		"error_recovery":     0.7,
	}
	// 0.35*0 + 0.20*0.5 + 0.20*0 + 0.15*0.5 + 0.10*0.7 = 0.245
	weighted := 0.0
	for k, w := range TrustWeights {
		weighted += components[k] * w
	}
	return finalizeTrustScore(weighted, components, 0)
}

// finalizeTrustScore clips, rounds, and assigns the tier. Shared by
// both zero and populated paths so the level-lookup rules live in
// exactly one place.
func finalizeTrustScore(weighted float64, components map[string]float64, n int) TrustScore {
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
		HistorySize:        n,
		ComputedAt:         NowISO(),
	}
}

// TrustSourceCounter tracks outcome-memory trust lookups. Phase 4
// removed FromOpHistory and DeprecationCount — the legacy paths are
// gone, so there is only one source left. Exposed via
// `hwcloud-skillcheck trust stats`.
type TrustSourceCounter struct {
	FromOutcomeMemory atomic.Uint64
}

// DefaultTrustSource is the process-wide counter for trust lookups.
var DefaultTrustSource = &TrustSourceCounter{}

// LookupTrust is the canonical trust lookup. Always routes through
// outcome memory; no legacy fall-back (ADR-0009 §Migration Phase 4).
// Stamps the last-lookup time and increments FromOutcomeMemory when
// the memory had records to score; a nil mem or empty result yields
// the zero-history score without bumping the counter.
func LookupTrust(skill, action string, mem *OutcomeMemory) TrustScore {
	if mem != nil && skill != "" && action != "" {
		if recent, err := mem.RecentOutcomes(skill, action, trustOutcomeMaxRecords); err == nil && len(recent) > 0 {
			MarkLastOutcomeLookup()
			if DefaultTrustSource != nil {
				DefaultTrustSource.Record("outcome_memory")
			}
		}
	}
	return ComputeTrustScoreFromOutcome(skill, action, mem, "")
}

// Record increments the counter for the named source. Unknown sources
// are ignored (no panic) so callers can pass through free-form labels
// safely. Phase 4 only recognizes "outcome_memory".
func (t *TrustSourceCounter) Record(from string) {
	if t == nil {
		return
	}
	switch from {
	case "outcome_memory":
		t.FromOutcomeMemory.Add(1)
	}
}

var LastOutcomeLookup atomic.Pointer[string]

// MarkLastOutcomeLookup stamps NowISO() as the most recent outcome-memory
// trust lookup time.
func MarkLastOutcomeLookup() {
	ts := NowISO()
	LastOutcomeLookup.Store(&ts)
}

// SnapshotLastOutcomeLookup returns the last lookup time or "" if never
// recorded. Thread-safe.
func SnapshotLastOutcomeLookup() string {
	p := LastOutcomeLookup.Load()
	if p == nil {
		return ""
	}
	return *p
}

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

func parseISO(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, os.ErrInvalid
}

// TrustScore is the result of ComputeTrustScoreFromOutcome.
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