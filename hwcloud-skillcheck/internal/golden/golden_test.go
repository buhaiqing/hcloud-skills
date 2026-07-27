package golden

import (
	"os"
	"os/exec"
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
func TestRun_ReplaysScenarioAndPasses(t *testing.T) {
	root := t.TempDir()
	// Build the mockhcloud binary.
	bin := filepath.Join(root, "bin", "mockhcloud")
	_ = os.MkdirAll(filepath.Dir(bin), 0o700)
	build(t, bin)

	// Build a script JSON: match "hcloud ecs list" → response "[]", exit 0.
	ecsDir := filepath.Join(root, "testdata", "ecs-ops")
	_ = os.MkdirAll(ecsDir, 0o700)
	scriptPath := filepath.Join(ecsDir, "ecs-list-empty.script.json")
	_ = os.WriteFile(scriptPath, []byte(`{"entries":[{"match":"hcloud ecs list","response":"[]","exit_code":0}]}`), 0o600)

	// Write the scenario JSON.
	scenarioPath := filepath.Join(ecsDir, "ecs-list-empty.json")
	_ = os.WriteFile(scenarioPath, []byte(`{"name":"ecs-list-empty","command":"hcloud ecs list","expected_stdout_excerpt":"[]","expected_stderr_excerpt":"","expected_exit_code":0,"tags":["read-only"]}`), 0o600)

	sc := Scenario{
		Name:             "ecs-list-empty",
		Command:          "hcloud ecs list",
		ExpectedStdout:   "[]",
		ExpectedStderr:   "",
		ExpectedExitCode: 0,
	}
	ok, err := runScenario(root, scriptPath, sc)
	if err != nil {
		t.Fatalf("runScenario returned error: %v", err)
	}
	if !ok {
		t.Error("expected runScenario to pass")
	}
}

func build(t *testing.T, dst string) {
	t.Helper()
	out, err := exec.Command("go", "build", "-o", dst, "../../cmd/mockhcloud").CombinedOutput()
	if err != nil {
		t.Skipf("mockhcloud not buildable: %v\n%s", err, string(out))
	}
	_ = os.Chmod(dst, 0o755)
}
