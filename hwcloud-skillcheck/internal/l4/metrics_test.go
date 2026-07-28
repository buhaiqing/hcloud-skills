package l4

import (
	"bytes"
	"strings"
	"testing"
)

func TestWritePrometheus_HealingCounters(t *testing.T) {
	prevHeal := DefaultHealingMetrics
	prevTrust := DefaultTrustSource
	DefaultHealingMetrics = &HealingMetrics{}
	DefaultTrustSource = &TrustSourceCounter{}
	defer func() {
		DefaultHealingMetrics = prevHeal
		DefaultTrustSource = prevTrust
	}()

	DefaultHealingMetrics.Record(HealingDecision{Action: "skip"}, "PreExecHook")
	DefaultHealingMetrics.Record(HealingDecision{Action: "retry"}, "PostFailureHook")
	DefaultHealingMetrics.Record(HealingDecision{Action: "escalate"}, "PostFailureHook")
	DefaultTrustSource.Record("outcome_memory")

	var buf bytes.Buffer
	if err := WritePrometheus(&buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()

	for _, name := range []string{
		"l4_healing_pre_exec_skip_total",
		"l4_healing_post_failure_retry_total",
		"l4_healing_post_failure_escalate_total",
		`l4_trust_source_lookups_total{from="outcome_memory"}`,
	} {
		if !strings.Contains(got, name) {
			t.Errorf("missing metric name %q in output:\n%s", name, got)
		}
	}
	for _, want := range []string{
		"l4_healing_pre_exec_skip_total 1",
		"l4_healing_post_failure_retry_total 1",
		"l4_healing_post_failure_escalate_total 1",
		`l4_trust_source_lookups_total{from="outcome_memory"} 1`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing value line %q in output:\n%s", want, got)
		}
	}
}

func TestPrometheusText_NilCollectors(t *testing.T) {
	prevHeal := DefaultHealingMetrics
	prevTrust := DefaultTrustSource
	DefaultHealingMetrics = nil
	DefaultTrustSource = nil
	defer func() {
		DefaultHealingMetrics = prevHeal
		DefaultTrustSource = prevTrust
	}()

	got := PrometheusText()
	if !strings.Contains(got, "l4_healing_pre_exec_skip_total 0") {
		t.Errorf("expected zero counters with nil collectors, got:\n%s", got)
	}
}
