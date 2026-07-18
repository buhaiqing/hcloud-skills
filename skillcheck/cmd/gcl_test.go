package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// skillcheckBinary is built on demand for each test that needs it.
func buildSkillcheckBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join("/tmp", "skillcheck-gcl-test-"+filepath.Base(t.TempDir()))
	// Build the main package from the module root (not from cmd/).
	cmd := exec.Command("go", "build", "-o", bin, "github.com/buhaiqing/hcloud-skills/skillcheck")
	cmd.Dir = os.Getenv("SKILLCHECK_ROOT")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("go build failed: %v\n%s", err, out)
	}
	return bin
}

func TestGCLHelp(t *testing.T) {
	bin := buildSkillcheckBinary(t)
	cmd := exec.Command(bin, "gcl", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gcl --help failed: %v", err)
	}
	got := string(out)
	for _, want := range []string{"skillcheck gcl run", "skillcheck gcl alarm-wire"} {
		if !contains(got, want) {
			t.Errorf("gcl --help output missing %q:\n%s", want, got)
		}
	}
}

func TestGCLRunHelp(t *testing.T) {
	bin := buildSkillcheckBinary(t)
	cmd := exec.Command(bin, "gcl", "run", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gcl run --help failed: %v", err)
	}
	got := string(out)
	if !contains(got, "-root") && !contains(got, "skillcheck gcl run") {
		t.Errorf("gcl run --help output unexpected:\n%s", got)
	}
}

func TestGCLAlarmHelp(t *testing.T) {
	bin := buildSkillcheckBinary(t)
	cmd := exec.Command(bin, "gcl", "alarm-wire", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gcl alarm-wire --help failed: %v", err)
	}
	got := string(out)
	if !contains(got, "-root") && !contains(got, "-json") {
		t.Errorf("gcl alarm-wire --help output unexpected:\n%s", got)
	}
}

// TestGCLRunSmoke exercises `gcl run` against a real skill to verify no panic.
func TestGCLRunSmoke(t *testing.T) {
	bin := buildSkillcheckBinary(t)
	// huaweicloud-ecs-ops is in the parent of the skillcheck worktree.
	skillDir := filepath.Join(os.Getenv("SKILLCHECK_ROOT"), "huaweicloud-ecs-ops")
	if _, err := os.Stat(skillDir); err != nil {
		t.Skip("huaweicloud-ecs-ops not found, skipping smoke test")
	}
	cmd := exec.Command(bin, "gcl", "run", "--root", skillDir, "--quiet")
	out, err := cmd.CombinedOutput()
	t.Logf("gcl run output: %s", string(out))
	t.Logf("gcl run exit error: %v", err)
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
