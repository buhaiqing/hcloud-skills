package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/l4"
)

func TestMetricsHTTPHandler(t *testing.T) {
	root := t.TempDir()
	prevHeal := l4.DefaultHealingMetrics
	prevTrust := l4.DefaultTrustSource
	l4.DefaultHealingMetrics = &l4.HealingMetrics{}
	l4.DefaultTrustSource = &l4.TrustSourceCounter{}
	l4.SetMetricsPersistRoot(root)
	defer func() {
		l4.DefaultHealingMetrics = prevHeal
		l4.DefaultTrustSource = prevTrust
		l4.SetMetricsPersistRoot("")
	}()

	l4.DefaultHealingMetrics.Record(l4.HealingDecision{Action: "skip"}, "PreExecHook")
	l4.DefaultTrustSource.Record("outcome_memory")

	// Persisted file must exist for a scrape-style process.
	if _, err := os.Stat(filepath.Join(root, ".l4-memory", "metrics.json")); err != nil {
		t.Fatalf("expected persisted metrics.json: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	metricsHTTPHandler(root)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("Content-Type = %q, want text/plain", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"l4_healing_pre_exec_skip_total 1",
		"l4_healing_post_failure_retry_total 0",
		"l4_healing_post_failure_escalate_total 0",
		`l4_trust_source_lookups_total{from="outcome_memory"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in body:\n%s", want, body)
		}
	}
}

func TestMetricsHTTPHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	rec := httptest.NewRecorder()
	metricsHTTPHandler(t.TempDir())(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestMetricsHTTPHandler_ReadsPersistedAcrossProcess(t *testing.T) {
	root := t.TempDir()
	// Simulate l4 handle process writing counters, then a fresh scrape with
	// zero in-process atomics (metrics subprocess).
	l4.SetMetricsPersistRoot(root)
	prevHeal := l4.DefaultHealingMetrics
	prevTrust := l4.DefaultTrustSource
	l4.DefaultHealingMetrics = &l4.HealingMetrics{}
	l4.DefaultTrustSource = &l4.TrustSourceCounter{}
	l4.DefaultHealingMetrics.Record(l4.HealingDecision{Action: "retry"}, "PostFailureHook")
	l4.DefaultHealingMetrics.Record(l4.HealingDecision{Action: "escalate"}, "PostFailureHook")
	l4.SetMetricsPersistRoot("")
	l4.DefaultHealingMetrics = &l4.HealingMetrics{} // zero in-process
	l4.DefaultTrustSource = &l4.TrustSourceCounter{}
	defer func() {
		l4.DefaultHealingMetrics = prevHeal
		l4.DefaultTrustSource = prevTrust
		l4.SetMetricsPersistRoot("")
	}()

	rec := httptest.NewRecorder()
	metricsHTTPHandler(root)(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "l4_healing_post_failure_retry_total 1") {
		t.Fatalf("scrape missed persisted retry counter:\n%s", body)
	}
	if !strings.Contains(body, "l4_healing_post_failure_escalate_total 1") {
		t.Fatalf("scrape missed persisted escalate counter:\n%s", body)
	}
}
