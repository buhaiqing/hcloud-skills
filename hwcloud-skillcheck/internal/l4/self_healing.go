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
