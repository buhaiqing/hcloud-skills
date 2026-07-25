package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAggregateTraceGood(t *testing.T) {
	root := t.TempDir()
	writeTraceJSON(t, root, "gcl-trace-20260701-000000.json", traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0))
	writeTraceJSON(t, root, "gcl-trace-20260701-000001.json", traceFixture("huaweicloud-ecs-ops", "PASS", 2, 1.0))
	writeTraceJSON(t, root, "gcl-trace-20260701-000002.json", traceFixture("huaweicloud-rds-ops", "SAFETY_FAIL", 3, 0.0))

	out := filepath.Join(root, "summary-out.json")
	if err := runAggregate([]string{"trace", "--root", root, "--output", out}); err != nil {
		t.Fatalf("aggregate should succeed, got: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var summary map[string]any
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	totals, ok := summary["totals"].(map[string]any)
	if !ok {
		t.Fatalf("missing totals in summary: %v", summary)
	}
	if numOf(totals["total_runs"]) != 3 {
		t.Fatalf("expected total_runs=3, got %v", totals["total_runs"])
	}
	if numOf(totals["PASS"]) != 2 {
		t.Fatalf("expected PASS=2, got %v", totals["PASS"])
	}
	if numOf(totals["SAFETY_FAIL"]) != 1 {
		t.Fatalf("expected SAFETY_FAIL=1, got %v", totals["SAFETY_FAIL"])
	}
	if diff := summary["pass_rate"].(float64) - 2.0/3.0; diff > 0.0001 || diff < -0.0001 {
		t.Fatalf("expected pass_rate~=%.4f, got %v", 2.0/3.0, summary["pass_rate"])
	}
	bySkill, ok := summary["by_skill"].(map[string]any)
	if !ok {
		t.Fatalf("missing by_skill: %v", summary)
	}
	if _, ok := bySkill["huaweicloud-ecs-ops"]; !ok {
		t.Fatal("expected huaweicloud-ecs-ops in by_skill")
	}
}

func TestAggregateTraceNoFiles(t *testing.T) {
	root := t.TempDir()
	// No audit-results/gcl-trace-*.json => WARN and skip, exit 0 (Spec §4).
	if err := runAggregate([]string{"trace", "--root", root}); err != nil {
		t.Fatalf("no-trace aggregate should WARN+exit 0, got: %v", err)
	}
}

func TestAggregateTraceSinceHours(t *testing.T) {
	root := t.TempDir()
	// A fresh trace should be picked up by --since-hours 1.
	writeTraceJSON(t, root, "gcl-trace-recent.json", traceFixture("huaweicloud-ecs-ops", "PASS", 1, 1.0))
	if err := runAggregate([]string{"trace", "--root", root, "--since-hours", "1"}); err != nil {
		t.Fatalf("recent trace should aggregate, got: %v", err)
	}
}

func TestAggregateTraceSelfCheck(t *testing.T) {
	// --self-check aggregates the embedded healthy trace fixture and must
	// produce a well-formed summary (total_runs>=1, pass_rate in [0,1]).
	out := filepath.Join(t.TempDir(), "self-summary.json")
	if err := runAggregate([]string{"trace", "--self-check", "--output", out}); err != nil {
		t.Fatalf("self-check should succeed, got: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var summary map[string]any
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	totals, ok := summary["totals"].(map[string]any)
	if !ok {
		t.Fatalf("missing totals in self-check summary: %v", summary)
	}
	if numOf(totals["total_runs"]) < 1 {
		t.Fatalf("expected total_runs>=1, got %v", totals["total_runs"])
	}
	passRate, ok := summary["pass_rate"].(float64)
	if !ok || passRate < 0 || passRate > 1 {
		t.Fatalf("expected pass_rate in [0,1], got %v", summary["pass_rate"])
	}
}

func traceFixture(skill, status string, iters int, score float64) string {
	// Build a minimal valid trace with `iters` iterations, each PASS at score.
	type iter struct {
		Iter   int `json:"iter"`
		Critic struct {
			Scores map[string]float64 `json:"scores"`
		} `json:"critic"`
		Decision string `json:"decision"`
	}
	iterations := make([]iter, 0, iters)
	for i := 1; i <= iters; i++ {
		var c struct {
			Scores map[string]float64 `json:"scores"`
		}
		c.Scores = map[string]float64{
			"correctness": score, "safety": score, "idempotency": score,
			"traceability": score, "spec_compliance": score,
		}
		iterations = append(iterations, iter{Iter: i, Critic: c, Decision: status})
	}
	b, _ := json.Marshal(map[string]any{
		"skill":      skill,
		"request":    "smoke",
		"iterations": iterations,
		"final":      map[string]any{"status": status, "iter": iters},
	})
	return string(b)
}

func writeTraceJSON(t *testing.T, root, name, content string) {
	t.Helper()
	dir := filepath.Join(root, "audit-results")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// numOf normalizes JSON number values (int or float64) to int for assertions.
func numOf(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}
