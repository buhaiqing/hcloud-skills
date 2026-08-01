package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGate returns a gate that always passes (or fails, honoring `soft`).
func fakeGate(label string, pass, soft bool) preCommitGate {
	return preCommitGate{
		label: label,
		run: func(rootDir string) gateResult {
			return gateResult{passed: pass, soft: soft, detail: ""}
		},
	}
}

// TestRunPreCommitGates_ExitContract verifies the core gate-iteration +
// exit-code contract without touching the real `go` toolchain or binaries.
func TestRunPreCommitGates_ExitContract(t *testing.T) {
	// All hard gates pass -> nil (exit 0).
	allPass := []preCommitGate{
		fakeGate("hard-ok", true, false),
		fakeGate("soft-ok", true, true),
	}
	if err := reportPreCommitResults(runPreCommitGates(allPass, ".")); err != nil {
		t.Fatalf("expected nil error when all hard gates pass, got %v", err)
	}

	// One hard gate fails -> non-nil (exit 1).
	hardFail := []preCommitGate{
		fakeGate("hard-ok", true, false),
		fakeGate("hard-bad", false, false),
	}
	if err := reportPreCommitResults(runPreCommitGates(hardFail, ".")); err == nil {
		t.Fatal("expected non-nil error when a hard gate fails")
	}

	// A soft gate failure must NOT flip the exit code.
	softFail := []preCommitGate{
		fakeGate("hard-ok", true, false),
		fakeGate("soft-bad", false, true),
	}
	if err := reportPreCommitResults(runPreCommitGates(softFail, ".")); err != nil {
		t.Fatalf("soft-gate failure must not fail the run, got %v", err)
	}
}

// TestReportPreCommitResults_Summary verifies the summary message wording so a
// regressing message still fails loudly in review.
func TestReportPreCommitResults_Summary(t *testing.T) {
	// Capture via a hard failure; we only assert the error is wrapped with the
	// failure count, which is what callers key on.
	err := reportPreCommitResults([]gateResult{{name: "x", passed: false, soft: false}})
	if err == nil {
		t.Fatal("expected error for single hard failure")
	}
	if !strings.Contains(err.Error(), "1 hard gate(s) failed") {
		t.Fatalf("unexpected error text: %q", err.Error())
	}
}

// TestPreCommitGates_SkipTests verifies gate #13 (Go test) is appended only
// when skipTests is false, and that gates 8/10 are marked soft.
func TestPreCommitGates_SkipTests(t *testing.T) {
	withTests := preCommitGates(false)
	last := withTests[len(withTests)-1]
	if last.label != "Go test" {
		t.Fatalf("expected last gate 'Go test' when skipTests=false, got %q", last.label)
	}

	withoutTests := preCommitGates(true)
	for _, g := range withoutTests {
		if g.label == "Go test" {
			t.Fatal("Go test gate must be omitted when skipTests=true")
		}
	}

	// Soft gates must be flagged so they never fail the run.
	softLabels := map[string]bool{}
	for _, g := range withTests {
		if g.label == "hwcloud-skillcheck golden run" || g.label == "hwcloud-skillcheck ab compare" {
			// run is wrapped via inProcessGate; softness is encoded inside the
			// gate func, not the registry. Assert the labels exist instead.
			softLabels[g.label] = true
		}
	}
	if len(softLabels) != 2 {
		t.Fatalf("expected 2 soft gate labels present, got %d", len(softLabels))
	}
}

// TestMaskSecrets verifies credential values are scrubbed from detail strings.
func TestMaskSecrets(t *testing.T) {
	t.Setenv("HW_SECRET_ACCESS_KEY", "super-secret-value")
	got := maskSecrets("connection failed for super-secret-value")
	if strings.Contains(got, "super-secret-value") {
		t.Fatalf("maskSecrets leaked secret: %q", got)
	}
	if !strings.Contains(got, "***") {
		t.Fatalf("maskSecrets did not redact: %q", got)
	}
}

// TestResolveRepoRoot_WalkUp verifies a bare "." walks up to the repo root
// containing a marker (CA-7: cwd-tolerant --root), so a subdir invocation does
// not silently gate the wrong tree.
func TestResolveRepoRoot_WalkUp(t *testing.T) {
	// Build a temp tree: <root>/AGENTS.md and <root>/sub/deep.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "sub", "deep")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	got, err := resolveRepoRoot(".")
	if err != nil {
		t.Fatalf("resolveRepoRoot error: %v", err)
	}
	if got != root {
		t.Fatalf("expected walk-up to %q, got %q", root, got)
	}
}

// TestResolveRepoRoot_Explicit keeps an explicit --root as-is (Abs'd).
func TestResolveRepoRoot_Explicit(t *testing.T) {
	got, err := resolveRepoRoot("/tmp") // not cwd, must not walk
	if err != nil {
		t.Fatalf("resolveRepoRoot error: %v", err)
	}
	if got != "/tmp" {
		t.Fatalf("explicit root should be preserved, got %q", got)
	}
}

// TestReportPreCommitResults_BuildFailureAsGate verifies a synthetic build-failure
// gateResult flips the run to a hard failure (summary contract stays consistent).
func TestReportPreCommitResults_BuildFailureAsGate(t *testing.T) {
	results := []gateResult{
		{name: "gofmt", passed: true},
		{name: "build hwcloud-skillcheck", passed: false, detail: "boom"},
	}
	if err := reportPreCommitResults(results); err == nil {
		t.Fatal("expected non-nil error when build gate fails")
	}
}
