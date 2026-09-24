package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validCampaignMetrics = `{
  "post_fix_rereview": "reviewed",
  "dispatches": {"failed": 7, "total": 16},
  "critic": {"findings": 20, "acknowledged_false_positive": 1},
  "adjudicator": {"overturns": 2},
  "fix": {"touched_in_later_rounds": 6, "first_round_files": 25},
  "rounds": {"rereviewed": 2, "total": 2}
}`

func writeMetrics(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "metrics.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write metrics: %v", err)
	}
	return path
}

func recordArgs(root, metricsPath string, extra ...string) []string {
	args := []string{"record", "--root", root, "--id", "dec-test-001",
		"--outcome", "success", "--metrics", metricsPath}
	return append(args, extra...)
}

func TestLearningCampaignRecord_WritesCampaignAndOutcome(t *testing.T) {
	root := t.TempDir()
	metrics := writeMetrics(t, validCampaignMetrics)

	if err := runLearningCampaign(recordArgs(root, metrics)); err != nil {
		t.Fatalf("record campaign: %v", err)
	}

	campaignPath := filepath.Join(root, "audit-results", "gcl-campaigns", "dec-test-001.json")
	body, err := os.ReadFile(campaignPath)
	if err != nil {
		t.Fatalf("campaign artifact missing: %v", err)
	}
	var record map[string]any
	if err := json.Unmarshal(body, &record); err != nil {
		t.Fatalf("campaign artifact is not JSON: %v", err)
	}
	// The artifact keeps raw counts, not derived rates: `campaign report`
	// recomputes them so an edited artifact cannot disagree with its own report.
	if _, derived := record["rates"]; derived {
		t.Fatalf("campaign artifact must not store derived rates: %v", record["rates"])
	}
	raw, ok := record["raw"].(map[string]any)
	if !ok {
		t.Fatalf("campaign artifact must keep raw counts, got: %v", record["raw"])
	}
	if critic, _ := raw["critic"].(map[string]any); critic["findings"] != float64(20) {
		t.Fatalf("raw critic counts must survive, got: %v", raw["critic"])
	}

	row, err := os.ReadFile(filepath.Join(root, ".l4-memory", "outcomes.jsonl"))
	if err != nil {
		t.Fatalf("outcome row missing: %v", err)
	}
	var outcome map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(row))), &outcome); err != nil {
		t.Fatalf("outcome row is not JSON: %v", err)
	}
	if outcome["gcl_experiment_id"] != "dec-test-001" {
		t.Fatalf("outcome row must carry the experiment id, got: %v", outcome["gcl_experiment_id"])
	}
	for _, key := range []string{"dispatch_failure_rate", "fix_rework_rate", "post_fix_rereview_coverage"} {
		v, present := outcome[key]
		if !present {
			t.Fatalf("outcome row missing metric %q: %v", key, outcome)
		}
		if f, isFloat := v.(float64); !isFloat || f < 0 || f > 1 {
			t.Fatalf("metric %q must be a rate in [0,1], got %v", key, v)
		}
	}
	// The dropped critic-signal rate must not reappear under any name.
	for key := range outcome {
		if strings.Contains(key, "critic") {
			t.Fatalf("critic-signal rate must stay raw in the artifact, found %q on the row", key)
		}
	}
	// 7/16 dispatches failed — the row carries the computed rate, not the raw count.
	if got := outcome["dispatch_failure_rate"].(float64); got < 0.43 || got > 0.44 {
		t.Fatalf("dispatch_failure_rate = %v, want ~0.4375", got)
	}
	// retry_count means per-step retries in the schema; dispatch failures must
	// not leak into it or trust.go infers the wrong thing.
	if got := outcome["retry_count"].(float64); got != 0 {
		t.Fatalf("retry_count = %v, want 0 (dispatch failures live in the artifact)", got)
	}
	if hash, _ := outcome["context_hash"].(string); !strings.HasPrefix(hash, "sha256:") || len(hash) != len("sha256:")+64 {
		t.Fatalf("context_hash must be a real sha256 of the metrics input, got %q", outcome["context_hash"])
	}
}

func TestLearningCampaignRecord_ZeroRateIsPreserved(t *testing.T) {
	// A campaign with no failed dispatches measures 0.0 — omitempty on a plain
	// float would silently drop it and read as "not measured".
	root := t.TempDir()
	metrics := writeMetrics(t, `{
      "post_fix_rereview": "reviewed",
      "dispatches": {"failed": 0, "total": 12},
      "critic": {"findings": 8, "acknowledged_false_positive": 0},
      "adjudicator": {"overturns": 0},
      "fix": {"touched_in_later_rounds": 0, "first_round_files": 10},
      "rounds": {"rereviewed": 1, "total": 1}
    }`)

	if err := runLearningCampaign(recordArgs(root, metrics)); err != nil {
		t.Fatalf("record campaign: %v", err)
	}
	row, err := os.ReadFile(filepath.Join(root, ".l4-memory", "outcomes.jsonl"))
	if err != nil {
		t.Fatalf("outcome row missing: %v", err)
	}
	if !strings.Contains(string(row), `"dispatch_failure_rate":0`) {
		t.Fatalf("a measured 0.0 rate must be serialized, got: %s", row)
	}
}

func TestLearningCampaignRecord_WaiverRequiresEvidence(t *testing.T) {
	root := t.TempDir()
	metrics := writeMetrics(t, `{
      "post_fix_rereview": "waived",
      "waivers": [{"finding_id": "F1"}],
      "dispatches": {"failed": 1, "total": 4},
      "critic": {"findings": 4, "acknowledged_false_positive": 0},
      "adjudicator": {"overturns": 0},
      "fix": {"touched_in_later_rounds": 1, "first_round_files": 4},
      "rounds": {"rereviewed": 0, "total": 1}
    }`)

	err := runLearningCampaign(recordArgs(root, metrics))
	if err == nil || !strings.Contains(err.Error(), "waiver 0 incomplete") {
		t.Fatalf("an incomplete waiver must be rejected, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "audit-results", "gcl-campaigns", "dec-test-001.json")); statErr == nil {
		t.Fatal("a rejected campaign must not leave an artifact behind")
	}
}

func TestLearningCampaignRecord_AcceptsCompleteWaiver(t *testing.T) {
	root := t.TempDir()
	metrics := writeMetrics(t, `{
      "post_fix_rereview": "waived",
      "waivers": [{"finding_id": "F1", "file_line": "cmd/root.go:42", "command": "go test ./cmd/", "result": "pass"}],
      "dispatches": {"failed": 1, "total": 4},
      "critic": {"findings": 4, "acknowledged_false_positive": 0},
      "adjudicator": {"overturns": 1},
      "fix": {"touched_in_later_rounds": 1, "first_round_files": 4},
      "rounds": {"rereviewed": 0, "total": 1}
    }`)

	if err := runLearningCampaign(recordArgs(root, metrics)); err != nil {
		t.Fatalf("a fully evidenced waiver must be accepted, got: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, "audit-results", "gcl-campaigns", "dec-test-001.json"))
	if err != nil {
		t.Fatalf("campaign artifact missing: %v", err)
	}
	if !strings.Contains(string(body), "cmd/root.go:42") {
		t.Fatalf("waiver evidence must be persisted, got: %s", body)
	}
}

func TestLearningCampaignRecord_RejectsBadCounts(t *testing.T) {
	tests := map[string]struct{ metrics, want string }{
		"zero denominator": {
			`{"post_fix_rereview":"reviewed","dispatches":{"failed":0,"total":0},
			  "critic":{"findings":4,"acknowledged_false_positive":0},"adjudicator":{"overturns":0},
			  "fix":{"touched_in_later_rounds":0,"first_round_files":4},
			  "rounds":{"rereviewed":1,"total":1}}`,
			"denominator must be > 0",
		},
		"numerator above denominator": {
			`{"post_fix_rereview":"reviewed","dispatches":{"failed":9,"total":4},
			  "critic":{"findings":4,"acknowledged_false_positive":0},"adjudicator":{"overturns":0},
			  "fix":{"touched_in_later_rounds":0,"first_round_files":4},
			  "rounds":{"rereviewed":1,"total":1}}`,
			"out of range",
		},
		"negative numerator": {
			`{"post_fix_rereview":"reviewed","dispatches":{"failed":-1,"total":4},
			  "critic":{"findings":4,"acknowledged_false_positive":0},"adjudicator":{"overturns":0},
			  "fix":{"touched_in_later_rounds":0,"first_round_files":4},
			  "rounds":{"rereviewed":1,"total":1}}`,
			"out of range",
		},
		"missing post_fix_rereview": {
			`{"dispatches":{"failed":1,"total":4},"critic":{"findings":4,"acknowledged_false_positive":0},
			  "adjudicator":{"overturns":0},
			  "fix":{"touched_in_later_rounds":0,"first_round_files":4},
			  "rounds":{"rereviewed":1,"total":1}}`,
			"post_fix_rereview must be reviewed|waived",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			metrics := writeMetrics(t, tc.metrics)
			err := runLearningCampaign(recordArgs(root, metrics))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want error containing %q, got: %v", tc.want, err)
			}
		})
	}
}

func TestLearningCampaignRecord_RejectsBadOutcomeAndRequiresFlags(t *testing.T) {
	root := t.TempDir()
	metrics := writeMetrics(t, validCampaignMetrics)

	err := runLearningCampaign([]string{"record", "--root", root, "--id", "x",
		"--outcome", "partial", "--metrics", metrics})
	if err == nil || !strings.Contains(err.Error(), "success|failure|blocked") {
		t.Fatalf("an out-of-enum outcome must be rejected, got: %v", err)
	}

	err = runLearningCampaign([]string{"record", "--root", root, "--metrics", metrics})
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("missing --id/--outcome must be rejected, got: %v", err)
	}
}

func TestLearningCampaignRecord_DryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	metrics := writeMetrics(t, validCampaignMetrics)

	if err := runLearningCampaign(recordArgs(root, metrics, "--dry-run")); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "audit-results")); err == nil {
		t.Fatal("dry-run must not create audit-results")
	}
	if _, err := os.Stat(filepath.Join(root, ".l4-memory")); err == nil {
		t.Fatal("dry-run must not create outcome memory")
	}
}

func TestLearningCampaignRecord_RequiresCriticCounts(t *testing.T) {
	// Omitting the inconvenient numbers must not read as "0 findings found".
	root := t.TempDir()
	metrics := writeMetrics(t, `{
      "post_fix_rereview": "reviewed",
      "dispatches": {"failed": 1, "total": 4},
      "fix": {"touched_in_later_rounds": 0, "first_round_files": 4},
      "rounds": {"rereviewed": 1, "total": 1}
    }`)

	err := runLearningCampaign(recordArgs(root, metrics))
	if err == nil || !strings.Contains(err.Error(), "metrics.critic.findings is required") {
		t.Fatalf("missing critic counts must be rejected, got: %v", err)
	}
}

func TestLearningCampaignRecord_RejectsDuplicateExperimentID(t *testing.T) {
	// Re-recording the same id would overwrite the artifact's raw counts while
	// the first outcome row survives, leaving the two stores disagreeing.
	root := t.TempDir()
	metrics := writeMetrics(t, validCampaignMetrics)
	if err := runLearningCampaign(recordArgs(root, metrics)); err != nil {
		t.Fatalf("first record: %v", err)
	}
	err := runLearningCampaign(recordArgs(root, metrics))
	if err == nil || !strings.Contains(err.Error(), "already recorded") {
		t.Fatalf("a duplicate --id must be rejected, got: %v", err)
	}
	rows, err := os.ReadFile(filepath.Join(root, ".l4-memory", "outcomes.jsonl"))
	if err != nil {
		t.Fatalf("outcome rows: %v", err)
	}
	if got := len(strings.Split(strings.TrimSpace(string(rows)), "\n")); got != 1 {
		t.Fatalf("a rejected re-record must not append a second row, got %d rows", got)
	}
}

func TestLearningCampaignReport_RecomputesFromRawCounts(t *testing.T) {
	root := t.TempDir()
	metrics := writeMetrics(t, validCampaignMetrics)
	if err := runLearningCampaign(recordArgs(root, metrics)); err != nil {
		t.Fatalf("record campaign: %v", err)
	}

	if err := runLearningCampaignReport([]string{"--root", root, "--json"}); err != nil {
		t.Fatalf("report: %v", err)
	}

	// Second campaign so ordering and multi-row history are exercised.
	second := writeMetrics(t, strings.Replace(validCampaignMetrics,
		`"failed": 7, "total": 16`, `"failed": 0, "total": 4`, 1))
	if err := runLearningCampaign([]string{"record", "--root", root, "--id", "dec-test-002",
		"--outcome", "failure", "--metrics", second}); err != nil {
		t.Fatalf("record second campaign: %v", err)
	}

	// An artifact whose counts are internally broken must not hide the healthy
	// rows around it.
	broken := filepath.Join(root, "audit-results", "gcl-campaigns", "dec-broken.json")
	if err := os.WriteFile(broken, []byte(`{"experiment_id":"dec-broken","recorded_at":"2026-01-01T00:00:00Z",
      "raw":{"dispatches":{"failed":5,"total":1}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		if err := runLearningCampaignReport([]string{"--root", root, "--json"}); err != nil {
			t.Fatalf("report with a broken row: %v", err)
		}
	})
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("report is not JSON: %v\n%s", err, out)
	}
	if len(rows) != 3 {
		t.Fatalf("report must list all 3 campaigns, got %d: %s", len(rows), out)
	}
	first := rows[0]
	if first["experiment_id"] != "dec-broken" {
		t.Fatalf("rows must be sorted by recorded_at, got %v", first["experiment_id"])
	}
	if first["dispatch_failure_rate"].(float64) != -1 {
		t.Fatalf("an impossible count pair must report -1, got %v", first["dispatch_failure_rate"])
	}
	healthy := rows[1]
	if healthy["experiment_id"] != "dec-test-001" {
		t.Fatalf("unexpected second row: %v", healthy["experiment_id"])
	}
	if r := healthy["fix_rework_rate"].(float64); r < 0.23 || r > 0.25 {
		t.Fatalf("fix_rework_rate = %v, want 6/25 = 0.24", r)
	}
	if healthy["critic_findings"].(float64) != 20 || healthy["adjudicator_overturns"].(float64) != 2 {
		t.Fatalf("report must surface raw critic/adjudicator counts: %v", healthy)
	}
}

func TestLearningCampaignReport_NoCampaignsIsNotAnError(t *testing.T) {
	if err := runLearningCampaignReport([]string{"--root", t.TempDir()}); err != nil {
		t.Fatalf("an empty campaign history must not fail: %v", err)
	}
}
