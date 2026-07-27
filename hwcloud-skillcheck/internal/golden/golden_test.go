package golden

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun_DetectsPerSkillCoverage(t *testing.T) {
	root := t.TempDir()
	// Plant 5 scenarios for one skill, 0 for another.
	plantSkill(t, root, "huaweicloud-ecs-ops", 5)
	plantSkill(t, root, "huaweicloud-rds-ops", 0)
	report, err := Run(root)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := report.PerSkill["huaweicloud-ecs-ops"]; got != 5 {
		t.Errorf("ecs-ops: got %d, want 5", got)
	}
	if !report.BelowThreshold("huaweicloud-rds-ops", 3) {
		t.Error("rds-ops should be below threshold")
	}
}

func plantSkill(t *testing.T, root, skill string, n int) {
	t.Helper()
	dir := filepath.Join(root, "internal", "golden", "testdata", skill)
	_ = os.MkdirAll(dir, 0o700)
	for i := 0; i < n; i++ {
		_ = os.WriteFile(filepath.Join(dir, "sc-"+skill+"-"+string(rune('a'+i))+".json"),
			[]byte(`{"name":"sc","command":"hcloud ecs list","expected_stdout_excerpt":"[]","expected_stderr_excerpt":"","expected_exit_code":0,"tags":["read-only"]}`), 0o600)
	}
}
