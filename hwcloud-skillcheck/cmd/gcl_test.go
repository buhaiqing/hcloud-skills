package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// testBinary is the shared GCL test binary. Building it used to happen once per
// test call (4+ times); sync.Once makes the whole cmd test binary a single go
// build. A fixed path under os.TempDir is deliberate: t.TempDir() is per-test,
// so it cannot be shared across tests.
var (
	testBinaryOnce sync.Once
	testBinary     string
	testBinaryErr  error
)

// buildSkillcheckBinary returns the shared test binary path, building it once
// on first call.
func buildSkillcheckBinary(t *testing.T) string {
	t.Helper()
	testBinaryOnce.Do(func() {
		bin := filepath.Join(os.TempDir(), "hwcloud-skillcheck-gcl-test-bin")
		// Build the main package from the module root (not from cmd/).
		cmd := exec.Command("go", "build", "-o", bin, "github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck")
		cmd.Dir = os.Getenv("SKILLCHECK_ROOT")
		if out, err := cmd.CombinedOutput(); err != nil {
			// Record the error instead of t.Skipf here: t.Skipf Goexits, which
			// unwinds sync.Once before it marks itself done, so the next test
			// would re-run the closure AND read an empty binary path.
			testBinaryErr = fmt.Errorf("go build failed: %w\n%s", err, out)
			return
		}
		testBinary = bin
	})
	if testBinaryErr != nil {
		t.Skipf("%v", testBinaryErr)
	}
	return testBinary
}

func TestGCLHelp(t *testing.T) {
	bin := buildSkillcheckBinary(t)
	cmd := exec.Command(bin, "gcl", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gcl --help failed: %v", err)
	}
	got := string(out)
	for _, want := range []string{"hwcloud-skillcheck gcl run", "hwcloud-skillcheck gcl alarm-wire"} {
		if !strings.Contains(got, want) {
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
	if !strings.Contains(got, "-root") && !strings.Contains(got, "hwcloud-skillcheck gcl run") {
		t.Errorf("gcl run --help output unexpected:\n%s", got)
	}
	for _, want := range []string{"-budget-tokens", "-budget-tool-calls", "-budget-wall-clock", "-max-iter", "-structural-critic-only"} {
		if !strings.Contains(got, want) {
			t.Errorf("gcl run help missing %q:\n%s", want, got)
		}
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
	if !strings.Contains(got, "-root") && !strings.Contains(got, "-json") {
		t.Errorf("gcl alarm-wire --help output unexpected:\n%s", got)
	}
}

// TestGCLRunSmoke exercises `gcl run` against a real skill to verify no panic.
func TestGCLRunSmoke(t *testing.T) {
	bin := buildSkillcheckBinary(t)
	// Skill dirs live at the repo root, siblings of the hwcloud-skillcheck
	// module (SKILLCHECK_ROOT points at the module root for the build above).
	skillDir := filepath.Join(os.Getenv("SKILLCHECK_ROOT"), "..", "huaweicloud-ecs-ops")
	if _, err := os.Stat(skillDir); err != nil {
		t.Skip("huaweicloud-ecs-ops not found, skipping smoke test")
	}
	cmd := exec.Command(bin, "gcl", "run", "--root", skillDir, "--quiet")
	out, err := cmd.CombinedOutput()
	// A non-zero exit (or a failed spawn) means the smoke run itself broke —
	// log-only output would let a broken `gcl run` silently pass.
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		t.Fatalf("gcl run failed: %v\n%s", err, out)
	}
}

// TestGCLRunSmoke_FailsOnNonZeroExit proves the smoke assertion is real, not
// dead code: `gcl run` against a nonexistent skill dir exits non-zero — the
// exact condition TestGCLRunSmoke's Fatalf guards on.
func TestGCLRunSmoke_FailsOnNonZeroExit(t *testing.T) {
	bin := buildSkillcheckBinary(t)
	cmd := exec.Command(bin, "gcl", "run", "--root", filepath.Join(t.TempDir(), "no-such-skill"), "--quiet")
	out, err := cmd.CombinedOutput()
	if err == nil || cmd.ProcessState.ExitCode() == 0 {
		t.Fatalf("gcl run on a missing root should exit non-zero, got err=%v exit=%v\n%s", err, cmd.ProcessState.ExitCode(), out)
	}
}
