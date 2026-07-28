package l4

import (
	"bytes"
	"fmt"
	"io"
	"sync/atomic"
)

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
			bumpPersisted(func(s *CounterSnapshot) { s.PreExecSkip++ })
		}
	case "retry":
		m.PostFailureRetry.Add(1)
		bumpPersisted(func(s *CounterSnapshot) { s.PostFailureRetry++ })
	case "escalate":
		m.PostFailureEscalate.Add(1)
		bumpPersisted(func(s *CounterSnapshot) { s.PostFailureEscalate++ })
	}
}

// WritePrometheus emits healing and trust counters in Prometheus text exposition
// format. Nil collectors are treated as zero-valued.
func WritePrometheus(w io.Writer) error {
	if err := writeHealingPrometheus(w, DefaultHealingMetrics); err != nil {
		return err
	}
	return writeTrustPrometheus(w, DefaultTrustSource)
}

// PrometheusText returns the Prometheus exposition text for process-wide metrics.
func PrometheusText() string {
	var buf bytes.Buffer
	_ = WritePrometheus(&buf)
	return buf.String()
}

func writeHealingPrometheus(w io.Writer, m *HealingMetrics) error {
	var pre, retry, escalate uint64
	if m != nil {
		pre = m.PreExecSkip.Load()
		retry = m.PostFailureRetry.Load()
		escalate = m.PostFailureEscalate.Load()
	}
	metrics := []struct {
		name, help string
		value      uint64
	}{
		{"l4_healing_pre_exec_skip_total", "PreExecHook skip decisions", pre},
		{"l4_healing_post_failure_retry_total", "PostFailureHook retry decisions", retry},
		{"l4_healing_post_failure_escalate_total", "PostFailureHook escalate decisions", escalate},
	}
	for _, metric := range metrics {
		if _, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n%s %d\n",
			metric.name, metric.help, metric.name, metric.name, metric.value); err != nil {
			return err
		}
	}
	return nil
}

func writeTrustPrometheus(w io.Writer, t *TrustSourceCounter) error {
	var mem uint64
	if t != nil {
		mem = t.FromOutcomeMemory.Load()
	}
	_, err := fmt.Fprintf(w,
		"# HELP l4_trust_source_lookups_total Trust lookup count by source\n"+
			"# TYPE l4_trust_source_lookups_total counter\n"+
			`l4_trust_source_lookups_total{from="outcome_memory"} %d`+"\n",
		mem)
	return err
}
