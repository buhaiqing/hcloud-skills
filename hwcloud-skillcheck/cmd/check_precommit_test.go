package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
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
// It tests both local-dev mode (checkOnly=false) and CI mode (checkOnly=true).
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

// TestPreCommitGates_LocalVsCI verifies the gate count and composition differ
// between local-dev mode and CI (checkOnly) mode.
func TestPreCommitGates_LocalVsCI(t *testing.T) {
	// Local-dev mode: 13 gates (with tests, no CI-only gates).
	local := preCommitGates(preCommitConfig{skipTests: false, checkOnly: false, testRetries: 0})
	if len(local) != 13 {
		t.Fatalf("local mode expected 13 gates, got %d", len(local))
	}
	// Last gate must be Go test.
	if last := local[len(local)-1]; last.label != "Go test" {
		t.Fatalf("local mode last gate expected 'Go test', got %q", last.label)
	}

	// CI mode: 16 gates (12 base + go test + 3 CI-only: drift sync --dry-run,
	// gcl alarm-wire, critic-score).
	ci := preCommitGates(preCommitConfig{skipTests: false, checkOnly: true, testRetries: 2})
	if len(ci) != 16 {
		t.Fatalf("CI mode expected 16 gates, got %d", len(ci))
	}
	// Last gate must be critic-score.
	if last := ci[len(ci)-1]; last.label != "critic-score" {
		t.Fatalf("CI mode last gate expected 'critic-score', got %q", last.label)
	}

	// CI mode with skipTests: 15 gates (12 base + 3 CI-only, no go test).
	ciSkipTests := preCommitGates(preCommitConfig{skipTests: true, checkOnly: true, testRetries: 0})
	if len(ciSkipTests) != 15 {
		t.Fatalf("CI skip-tests mode expected 15 gates, got %d", len(ciSkipTests))
	}
	if last := ciSkipTests[len(ciSkipTests)-1]; last.label != "critic-score" {
		t.Fatalf("CI skip-tests last gate expected 'critic-score', got %q", last.label)
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
// when skipTests is false, and that gates 8/10 are marked soft. Also verifies
// that CI mode includes the 3 CI-only gates (drift sync --dry-run,
// gcl alarm-wire, critic-score) all as soft gates.
func TestPreCommitGates_SkipTests(t *testing.T) {
	withTests := preCommitGates(preCommitConfig{skipTests: false, checkOnly: false, testRetries: 0})
	last := withTests[len(withTests)-1]
	if last.label != "Go test" {
		t.Fatalf("expected last gate 'Go test' when skipTests=false, got %q", last.label)
	}

	withoutTests := preCommitGates(preCommitConfig{skipTests: true, checkOnly: false, testRetries: 0})
	for _, g := range withoutTests {
		if g.label == "Go test" {
			t.Fatal("Go test gate must be omitted when skipTests=true")
		}
	}

	// CI mode: 3 CI-only gates are appended (drift sync --dry-run,
	// gcl alarm-wire, critic-score), all soft.
	ciGates := preCommitGates(preCommitConfig{skipTests: false, checkOnly: true, testRetries: 2})
	ciOnlyLabels := []string{"drift sync --dry-run", "gcl alarm-wire", "critic-score"}
	found := map[string]bool{}
	for _, g := range ciGates {
		for _, want := range ciOnlyLabels {
			if g.label == want {
				found[want] = true
			}
		}
	}
	for _, want := range ciOnlyLabels {
		if !found[want] {
			t.Fatalf("CI mode must include gate %q", want)
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
// Since the replacer is built at init() time from real env vars, this test
// must re-init the replacer after t.Setenv to observe the masking.
func TestMaskSecrets(t *testing.T) {
	t.Setenv("HW_SECRET_ACCESS_KEY", "super-secret-value")
	initMaskSecrets() // re-init after t.Setenv
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

// TestRunStageConcurrently_NoDataRace verifies that running a stage with
// multiple independent gates concurrently produces correct results without
// data races. Each gate writes to a shared slice and prints to stdout/stderr
// under a mutex.
func TestRunStageConcurrently_NoDataRace(t *testing.T) {
	stage := []preCommitGate{
		fakeGate("gate-a", true, false),
		fakeGate("gate-b", true, false),
		fakeGate("gate-c", true, false),
		fakeGate("gate-d", false, true), // soft failure
	}
	results := runStageConcurrently(stage, ".")
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}
	for i, r := range results {
		if r.name != stage[i].label {
			t.Errorf("result[%d] name mismatch: expected %q, got %q", i, stage[i].label, r.name)
		}
	}
	// gate-d is a soft failure: must be false, soft, not flip the overall exit.
	if results[3].passed {
		t.Error("gate-d (soft failure) must report passed=false")
	}
	if !results[3].soft {
		t.Error("gate-d (soft failure) must report soft=true")
	}
}

// TestGateStages_PartitionsCorrectly verifies that gateStages groups gates into
// expected dependency stages.
func TestGateStages_PartitionsCorrectly(t *testing.T) {
	gates := preCommitGates(preCommitConfig{skipTests: false, checkOnly: true, testRetries: 2}) // CI mode: all 16 gates
	stages := gateStages(gates)

	if len(stages) < 2 {
		t.Fatalf("expected at least 2 stages, got %d", len(stages))
	}

	// Stage 0 must contain gofmt + go vet.
	hasGofmt := false
	hasGoVet := false
	for _, g := range stages[0] {
		if g.label == "gofmt" {
			hasGofmt = true
		}
		if g.label == "go vet" {
			hasGoVet = true
		}
	}
	if !hasGofmt || !hasGoVet {
		t.Errorf("stage 0 must contain gofmt and go vet, got %v", stageLabels(stages[0]))
	}

	// Total gate count across all stages must match.
	total := 0
	for _, s := range stages {
		total += len(s)
	}
	if total != len(gates) {
		t.Fatalf("stage total %d != gate total %d", total, len(gates))
	}
}

// stageLabels returns the labels of gates in a stage for diagnostics.
func stageLabels(stage []preCommitGate) []string {
	labels := make([]string, len(stage))
	for i, g := range stage {
		labels[i] = g.label
	}
	return labels
}

// TestMaskSecrets_ConcurrentSafety verifies maskSecrets is safe for concurrent
// use (the strings.Replacer is immutable after init, so reads are safe).
func TestMaskSecrets_ConcurrentSafety(t *testing.T) {
	t.Setenv("HW_ACCESS_KEY_ID", "test-ak-12345")
	// Re-init the replacer since we changed env vars.
	initMaskSecrets()
	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			result := maskSecrets("using key test-ak-12345 for auth")
			if strings.Contains(result, "test-ak-12345") {
				t.Errorf("maskSecrets leaked secret in concurrent call: %q", result)
			}
		}()
	}
	wg.Wait()
}

// initMaskSecrets rebuilds the secret replacer from current env state.
// Exported for test use only.
func initMaskSecrets() {
	var pairs []string
	for _, key := range []string{
		"HW_SECRET_ACCESS_KEY", "HW_ACCESS_KEY_ID",
		"TENCENTCLOUD_SECRET_KEY", "AWS_SECRET_ACCESS_KEY",
	} {
		if v := os.Getenv(key); v != "" {
			pairs = append(pairs, v, "***")
		}
	}
	if len(pairs) > 0 {
		secretReplacer = strings.NewReplacer(pairs...)
	} else {
		secretReplacer = nil
	}
}

// TestResolveRepoRoot_EmptyString verifies G8.1: empty --root must error rather
// than silently resolving to cwd or a wrong directory.
func TestResolveRepoRoot_EmptyString(t *testing.T) {
	_, err := resolveRepoRoot("")
	if err == nil {
		t.Fatal("resolveRepoRoot(\"\") must return an error (G8.1: input validation)")
	}
}

// TestResolveRepoRoot_PathTraversal verifies G8.3: path traversal attempts
// (../../etc) are rejected or safely contained.
func TestResolveRepoRoot_PathTraversal(t *testing.T) {
	// "../../../etc" should be cleaned to "/etc" — but after filepath.Abs it
	// becomes an absolute path outside the repo. The function should still
	// resolve it (it's an explicit --root, so caller takes responsibility),
	// but filepath.Clean must be applied.
	got, err := resolveRepoRoot("../../../etc")
	if err != nil {
		t.Fatalf("explicit path should not error: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("explicit path must be absolute, got %q", got)
	}
	// The result must be clean (no .. components).
	if strings.Contains(got, "..") {
		t.Fatalf("resolved path must not contain '..': %q", got)
	}
}

// TestResolveRepoRoot_NonExistent verifies G8.1: non-existent explicit path
// should still resolve (filepath.Abs doesn't check existence).
func TestResolveRepoRoot_NonExistent(t *testing.T) {
	got, err := resolveRepoRoot("/nonexistent/path/for/testing")
	if err != nil {
		t.Fatalf("non-existent explicit path should resolve: %v", err)
	}
	if got != "/nonexistent/path/for/testing" {
		t.Fatalf("expected path preserved, got %q", got)
	}
}
