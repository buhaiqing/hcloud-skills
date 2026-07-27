package maturity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func plantManifest(t *testing.T, root, skill string, golden int, te7, complete, lane bool) {
	t.Helper()
	dir := filepath.Join(root, skill)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	mf := map[string]interface{}{
		"maturity": map[string]interface{}{
			"golden_scenarios":     golden,
			"te7_pass":             te7,
			"manifest_complete":    complete,
			"telemetry_lane_clean": lane,
		},
	}
	b, err := json.Marshal(mf)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "capability_manifest.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestReport_RollsUpScores(t *testing.T) {
	root := t.TempDir()
	plantManifest(t, root, "skill-a", 5, true, true, true)
	plantManifest(t, root, "skill-b", 6, true, true, true)

	r, err := Rollup(root)
	if err != nil {
		t.Fatal(err)
	}
	for skill, score := range r.PerSkill {
		if score < 0.99 {
			t.Errorf("skill %s score = %.2f, want >= 0.99", skill, score)
		}
	}
}
func TestMaturityReportRollup(t *testing.T) {
	root := t.TempDir()
	plantManifest(t, root, "huaweicloud-ecs-ops", 5, true, true, true)
	r, err := Rollup(root)
	if err != nil {
		t.Fatalf("Rollup: %v", err)
	}
	if got := r.PerSkill["huaweicloud-ecs-ops"]; got < 0.99 {
		t.Errorf("score=%f, want >= 0.99", got)
	}
}
