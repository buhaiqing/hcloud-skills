package l4

import "sync/atomic"

// HealingMetrics exposes in-process counters for self-healing decisions.
type HealingMetrics struct {
	PreExecSkip         atomic.Uint64
	PostFailureRetry    atomic.Uint64
	PostFailureEscalate atomic.Uint64
}

// DefaultHealingMetrics is the process-wide healing metrics collector.
var DefaultHealingMetrics = &HealingMetrics{}

// Record increments the counter corresponding to decision and hook.
func (m *HealingMetrics) Record(decision HealingDecision, hook string) {
	if m == nil {
		return
	}
	switch decision.Action {
	case "skip":
		if hook == "PreExecHook" {
			m.PreExecSkip.Add(1)
		}
	case "retry":
		m.PostFailureRetry.Add(1)
	case "escalate":
		m.PostFailureEscalate.Add(1)
	}
}
