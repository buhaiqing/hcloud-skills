// Smoke test: hcloud-skillcheck Go binary self-checks against embedded fixtures.
//
// Python source scripts have been migrated to Go. This test validates that
// hcloud-skillcheck's core functionality works correctly using embedded
// fixtures (--self-check) and deterministic inputs.
//
// This does NOT test against the live repo — the repo may have pre-existing
// issues that hcloud-skillcheck correctly detects (those are validated by
// `make self-check`).
//
// Run with `go test ./internal/equivcheck/...` from the hwcloud-skillcheck
// module root. Exit code 0 = all checks pass; non-zero = at least one failure.
package equivcheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// resolveBinary returns the built binary path, building it on demand.
func resolveBinary(t *testing.T, root string) string {
	t.Helper()
	bin := filepath.Join(root, "bin", "hcloud-skillcheck")
	if _, err := os.Stat(bin); err == nil {
		return bin
	}
	cmd := exec.Command("go", "build", "-C", root, "-trimpath", "-o", "bin/hcloud-skillcheck", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build hcloud-skillcheck: %v\n%s", err, out)
	}
	return bin
}

// runSkillcheck invokes the binary with the given args and returns its exit
// code (0 when it succeeds).
func runSkillcheck(binary string, args ...string) int {
	cmd := exec.Command(binary, args...)
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		return -1 // failed to run (e.g. binary not found)
	}
	return 0
}

// SmokeCase describes one subcommand to exercise.
type SmokeCase struct {
	Name          string
	Args          []string
	ExpectSuccess bool
}

func TestEquivalence(t *testing.T) {
	root := projectRoot(t)
	binary := resolveBinary(t, root)
	fixtures := filepath.Join(root, "internal", "embed", "fixtures")

	cases := []SmokeCase{
		{"validate schema trace (fixture)",
			[]string{"validate", "schema", "trace", "--file", filepath.Join(fixtures, "gcl-trace-healthy.json")}, true},
		{"validate schema summary (fixture)",
			[]string{"validate", "schema", "summary", "--file", filepath.Join(fixtures, "gcl-quality-summary-healthy.json")}, true},
		{"validate schema alarm-plan (fixture)",
			[]string{"validate", "schema", "alarm-plan", "--file", filepath.Join(fixtures, "gcl-alarm-plan-healthy.json")}, true},
		{"scan secret trace --self-check",
			[]string{"scan", "secret", "trace", "--self-check"}, true},
		{"scan secret summary --self-check",
			[]string{"scan", "secret", "summary", "--self-check"}, true},
		{"scan secret alarm-plan --self-check",
			[]string{"scan", "secret", "alarm-plan", "--self-check"}, true},
		{"--help", []string{"--help"}, true},
	}

	var failures []string
	for _, c := range cases {
		exit := runSkillcheck(binary, c.Args...)
		success := exit == 0
		if success != c.ExpectSuccess {
			failures = append(failures, c.Name)
			t.Errorf("[FAIL] %s: expected success=%v but got exit=%d", c.Name, c.ExpectSuccess, exit)
		} else {
			t.Logf("[OK]   %s", c.Name)
		}
	}

	if len(failures) > 0 {
		t.Fatalf("%d smoke test(s) failed: %v", len(failures), failures)
	}
}

// projectRoot walks up from this source file to the module root, identified
// by the presence of go.mod (cwd-tolerant, per CA-7).
func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not locate module root (go.mod)")
	return ""
}
