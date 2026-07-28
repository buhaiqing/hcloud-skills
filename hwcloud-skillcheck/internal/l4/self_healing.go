package l4

import (
	"strings"
	"time"
)

// HealingPolicy configures pre-exec and post-failure hooks.
// Zero value disables auto-retry and skip-on-bad-history (safe default).
type HealingPolicy struct {
	MaxRetries               int
	RetryBackoff             time.Duration
	DestructiveVerbs         []string
	FailureRateSkipThreshold float64
	MinSamples               int
	LookbackWindow           time.Duration
}

// HealingDecision is the return value of pre/post hooks.
type HealingDecision struct {
	Action string // proceed | skip | retry | escalate
	Reason string
}

// transientPatterns are the substrings we treat as transient failures.
// Match is case-insensitive substring.
var transientPatterns = []string{
	"timeout",
	"token expired",
	"401",
	"429",
	"503",
	"connection reset",
}

// isTransient reports whether errMsg matches any transient pattern.
func isTransient(errMsg string) bool {
	lower := strings.ToLower(errMsg)
	for _, p := range transientPatterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// defaultHealingPolicy returns the safe baseline: no auto-retry, but the
// destructive-verb list is populated from the canonical RBAC high-risk
// verb regex in rbac.go (HighRiskVerbs). Single source of truth — if
// RBAC adds "rm" or "del" tomorrow, healing picks it up automatically.
func defaultHealingPolicy() HealingPolicy {
	return HealingPolicy{
		DestructiveVerbs: ExtractHighRiskVerbs(),
		MinSamples:       5,
		LookbackWindow:   time.Hour,
	}
}

// IsZero reports whether p has no values set across any field. Used by
// runExecutionLoopInner to short-circuit healing when the caller passed
// the zero-value HealingPolicy.
func (p HealingPolicy) IsZero() bool {
	return p.MaxRetries == 0 &&
		p.RetryBackoff == 0 &&
		len(p.DestructiveVerbs) == 0 &&
		p.FailureRateSkipThreshold == 0 &&
		p.MinSamples == 0 &&
		p.LookbackWindow == 0
}

// PreExecHook returns the action to take before executing step.
// Default: proceed. Skips only when:
//   - p.FailureRateSkipThreshold > 0
//   - at least p.MinSamples recent records exist for (step.Skill, step.Action)
//   - failure rate >= threshold
//   - the most recent record is within p.LookbackWindow (when set)
func PreExecHook(step TaskStep, mem *OutcomeMemory, p HealingPolicy) HealingDecision {
	if p.FailureRateSkipThreshold <= 0 || p.MinSamples <= 0 {
		return HealingDecision{Action: "proceed"}
	}
	recent, err := mem.RecentOutcomes(step.Skill, step.Action, p.MinSamples)
	if err != nil || len(recent) < p.MinSamples {
		return HealingDecision{Action: "proceed"}
	}
	failures := 0
	for _, r := range recent {
		if r.Outcome == "failure" {
			failures++
		}
	}
	rate := float64(failures) / float64(len(recent))
	if rate < p.FailureRateSkipThreshold {
		return HealingDecision{Action: "proceed"}
	}
	if p.LookbackWindow > 0 {
		last, err := time.Parse(time.RFC3339, recent[0].Timestamp)
		if err != nil || time.Since(last) > p.LookbackWindow {
			return HealingDecision{Action: "proceed"}
		}
	}
	return HealingDecision{Action: "skip", Reason: "high historical failure rate"}
}

// PostFailureHook returns the action to take after a step has failed.
//   - "retry"     when error is transient AND retry budget remains
//     AND step verb is not destructive (verb matched via EqualFold)
//   - "escalate"  in every other case (including MaxRetries=0)
func PostFailureHook(step TaskStep, result StepResult, retryCount int, mem *OutcomeMemory, p HealingPolicy) HealingDecision {
	if retryCount >= p.MaxRetries {
		return HealingDecision{Action: "escalate", Reason: "max retries reached"}
	}
	// Match by step.Verb (pre-extracted by inferRiskFromAction in execution.go),
	// NOT by substring of step.Action — "undelete-restore" must not be
	// classified as destructive just because "delete" is a substring.
	for _, verb := range p.DestructiveVerbs {
		if strings.EqualFold(step.Verb, verb) {
			return HealingDecision{Action: "escalate", Reason: "destructive op: no auto-retry"}
		}
	}
	if isTransient(result.Error) {
		return HealingDecision{Action: "retry", Reason: "transient error: " + result.Error}
	}
	return HealingDecision{Action: "escalate", Reason: "non-transient error: " + result.Error}
}
