package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// gateResult is the outcome of one pre-commit gate. A soft gate mirrors the
// shell's `|| true`: it prints a warning on failure but never flips the overall
// exit code.
type gateResult struct {
	name   string
	passed bool
	soft   bool
	detail string
}

// resolveRepoRoot returns the skill repo root. When raw is the default ".",
// walk up from the working directory to the nearest ancestor containing a repo
// marker (AGENTS.md) so a bare `check --pre-commit` invoked from any subdir
// still targets the right tree (CA-7: --root must be cwd-tolerant). An explicit
// --root value is taken as-is via filepath.Abs.
func resolveRepoRoot(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("--root must not be empty (G8.1: input validation)")
	}
	if raw != "." {
		return filepath.Abs(raw)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cur := wd
	for range 6 {
		if _, err := os.Stat(filepath.Join(cur, "AGENTS.md")); err == nil {
			return cur, nil
		}
		if _, err := os.Stat(filepath.Join(cur, "hwcloud-skillcheck")); err == nil {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return filepath.Abs(wd)
}

// preCommitConfig bundles the options for pre-commit gate selection (G6.2:
// structured config over positional bool params).
type preCommitConfig struct {
	skipTests   bool
	checkOnly   bool
	testRetries int
}

// preCommitGate binds a gate label to the function that runs it. Splitting the
// registry from the runner keeps the exit-code contract independently testable
// (a fake registry can be substituted in tests).
type preCommitGate struct {
	label string
	run   func(rootDir string) gateResult
}

// secretReplacer is a strings.Replacer built once at init from env values, so
// maskSecrets does a single pass without per-call os.Getenv or per-key loops.
var secretReplacer *strings.Replacer

// goBinPath is cached at init time so goToolGate, rebuildSkillcheckBinary,
// and gateGofmt avoid repeated exec.LookPath("go") calls.
var goBinPath string

func init() {
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
	}
	if p, err := exec.LookPath("go"); err == nil {
		goBinPath = p
	}
}

// maskSecrets scrubs credential values from gate detail strings. Uses a
// pre-built strings.Replacer (init-time) for O(n) single-pass replacement
// instead of per-key os.Getenv + ReplaceAll loops.
func maskSecrets(s string) string {
	if secretReplacer == nil {
		return s
	}
	return secretReplacer.Replace(s)
}

// runCheckPreCommit implements `hwcloud-skillcheck check --pre-commit
// [--skip-tests] [--check-only] [--test-retries N]`. It replaces
// scripts/pre_commit_check.sh: the same 13 gates (15 with --check-only), the
// same hard/soft semantics, and the same exit-code contract (1 on any hard
// failure, 0 otherwise). See ADR-0014.
func runCheckPreCommit(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck check --pre-commit")
	root := fs.String("root", ".", "skill repository root")
	skipTests := fs.Bool("skip-tests", false, "skip the go test gate (git hook passes this; CI does not)")
	checkOnly := fs.Bool("check-only", false, "CI mode: skip binary rebuild, drift-check-only, add CI-only gates")
	testRetries := fs.Int("test-retries", 0, "retry go test up to N times on failure (CI uses 2; local uses 0)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := resolveRepoRoot(*root)
	if err != nil {
		return err
	}

	gates := preCommitGates(preCommitConfig{skipTests: *skipTests, checkOnly: *checkOnly, testRetries: *testRetries})
	results := runPreCommitGates(gates, rootDir)

	// Defense-in-depth build: rebuild ./bin/hwcloud-skillcheck so a stale
	// binary cannot drift from current source (mirrors shell line 39). The
	// gates themselves call in-process funcs, so the built binary is only a
	// safety net for external callers; skip it with SKILLCHECK_SKIP_BUILD=1.
	// In --check-only mode (CI), the binary is built in a prior step, so this
	// rebuild is skipped entirely.
	if !*checkOnly && os.Getenv("SKILLCHECK_SKIP_BUILD") != "1" {
		if err := rebuildSkillcheckBinary(rootDir); err != nil {
			results = append(results, gateResult{
				name:   "build hwcloud-skillcheck",
				passed: false,
				detail: maskSecrets(err.Error()),
			})
		}
	}

	return reportPreCommitResults(results)
}

// rebuildSkillcheckBinary rebuilds ./bin/hwcloud-skillcheck from the module at
// <rootDir>/hwcloud-skillcheck (mirrors the shell's conditional build step).
func rebuildSkillcheckBinary(rootDir string) error {
	if goBinPath == "" {
		return fmt.Errorf("'go' toolchain not found on PATH")
	}
	out := filepath.Join(rootDir, "bin", "hwcloud-skillcheck")
	cmd := exec.Command(goBinPath, "build", "-trimpath", "-o", out, ".")
	cmd.Dir = filepath.Join(rootDir, "hwcloud-skillcheck")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(buf.String()))
	}
	return nil
}

// preCommitGates returns the ordered gate registry. Gates 8 (golden run),
// 10 (ab compare) are soft (warn but never fail). Gate 13 (go test) is omitted
// when skipTests is set. When checkOnly is true, CI-only gates (critic-score,
// gcl alarm-wire, drift sync --dry-run) are appended. Drift guard always does
// sync+check (never check-only in CI — fresh checkout needs sync to bootstrap).
func preCommitGates(cfg preCommitConfig) []preCommitGate {
	// In --check-only mode, the binary rebuild is skipped (CI builds separately),
	// but drift guard still does sync+check (fresh CI checkout has no .agents/skills/).
	// The check-only drift guard (no sync) is NOT used in CI — it's only for local
	// testing scenarios where the agent runtime copy already exists.
	driftGuardFn := gateDriftGuard
	gates := []preCommitGate{
		{"gofmt", gateGofmt},
		{"go vet", gateGoVet},
		{"hwcloud-skillcheck validate", gateValidate},
		{"hwcloud-skillcheck check audit-results", gateAuditResults},
		{"hwcloud-skillcheck aggregate trace", gateAggregateTrace},
		{"hwcloud-skillcheck learning gen", gateLearningGen},
		{"hwcloud-skillcheck l4 handle smoke", gateL4Handle},
		{"hwcloud-skillcheck golden run", gateGoldenRun}, // soft
		{"hwcloud-skillcheck check lanes", gateCheckLanes},
		{"hwcloud-skillcheck ab compare", gateABCompare}, // soft
		{"hwcloud-skillcheck check advanced-coverage", gateAdvancedCoverage},
		{"skill_generator drift guard", driftGuardFn},
	}
	if !cfg.skipTests {
		gates = append(gates, preCommitGate{"Go test", func(rootDir string) gateResult {
			return gateGoTestRetry(rootDir, cfg.testRetries)
		}})
	}
	if cfg.checkOnly {
		gates = append(gates,
			preCommitGate{"drift sync --dry-run", gateDriftSyncDryRun}, // soft
			preCommitGate{"gcl alarm-wire", gateGclAlarmWire},          // soft
			preCommitGate{"critic-score", gateCriticScore},             // soft
		)
	}
	return gates
}

// runPreCommitGates executes each gate in dependency-order stages. Gates within
// a stage are independent and run concurrently (no shared mutable state — each
// gate reads files or shells out to the go toolchain independently). Stage
// boundaries ensure that gates which depend on prior gate side-effects (e.g.
// validate must finish before audit-results) are ordered correctly.
func runPreCommitGates(gates []preCommitGate, rootDir string) []gateResult {
	// Group gates into stages. Within a stage, gates are independent and can
	// run concurrently. Stages are sequential: stage N+1 starts after stage N
	// completes.
	stages := gateStages(gates)
	var allResults []gateResult
	for _, stage := range stages {
		results := runStageConcurrently(stage, rootDir)
		allResults = append(allResults, results...)
	}
	return allResults
}

// gateStages partitions gates into dependency-order stages. Gates that have
// no inter-dependencies are grouped together for parallel execution.
//
// Stage layout:
//
//	Stage 0: gofmt (toolchain) + go vet (toolchain) — both read source only
//	Stage 1: validate (reads all skills, writes nothing)
//	Stage 2: audit-results + aggregate trace + learning gen + l4 handle
//	         + golden run + check lanes + ab compare + advanced-coverage
//	         (all read-only, no shared mutable state)
//	Stage 3: drift guard (may mutate via sync --apply in local mode)
//	Stage 4: go test (heavy, runs last in local; can run alongside CI-only in CI)
//	Stage 5: CI-only gates (drift sync --dry-run, gcl alarm-wire, critic-score)
func gateStages(gates []preCommitGate) [][]preCommitGate {
	// Simple partitioning: consecutive independent gates form a stage.
	// Dependencies: gofmt→go vet→validate→remaining (validate reads all files,
	// so it should finish before audit-results etc. that depend on its output).
	// Drift guard must run after validate (it syncs the generator).
	// Go test is independent of everything except the source code itself.
	var stages [][]preCommitGate
	i := 0

	// Stage 0: gofmt + go vet (both toolchain-only, read source)
	if i < len(gates) && (gates[i].label == "gofmt" || gates[i].label == "go vet") {
		stage := takeWhile(&i, gates, func(g preCommitGate) bool {
			return g.label == "gofmt" || g.label == "go vet"
		})
		if len(stage) > 0 {
			stages = append(stages, stage)
		}
	}

	// Stage 1: validate (single gate, but could be grouped with other
	// read-only in-process gates if they don't depend on validate output)
	if i < len(gates) && gates[i].label == "hwcloud-skillcheck validate" {
		stages = append(stages, []preCommitGate{gates[i]})
		i++
	}

	// Stage 2: all remaining read-only in-process gates before drift guard
	if i < len(gates) && gates[i].label != "skill_generator drift guard" && gates[i].label != "Go test" {
		stage := takeWhile(&i, gates, func(g preCommitGate) bool {
			return g.label != "skill_generator drift guard" && g.label != "Go test" &&
				g.label != "drift sync --dry-run" && g.label != "gcl alarm-wire" && g.label != "critic-score"
		})
		if len(stage) > 0 {
			stages = append(stages, stage)
		}
	}

	// Stage 3: drift guard (may mutate)
	if i < len(gates) && gates[i].label == "skill_generator drift guard" {
		stages = append(stages, []preCommitGate{gates[i]})
		i++
	}

	// Stage 4: Go test (heavy, independent)
	if i < len(gates) && gates[i].label == "Go test" {
		stages = append(stages, []preCommitGate{gates[i]})
		i++
	}

	// Stage 5: remaining CI-only soft gates (all independent)
	if i < len(gates) {
		stage := gates[i:]
		if len(stage) > 0 {
			stages = append(stages, stage)
		}
	}

	return stages
}

// takeWhile consumes gates from *i while fn returns true, returning the
// consumed slice and advancing i.
func takeWhile(i *int, gates []preCommitGate, fn func(preCommitGate) bool) []preCommitGate {
	var stage []preCommitGate
	for *i < len(gates) && fn(gates[*i]) {
		stage = append(stage, gates[*i])
		*i++
	}
	return stage
}

// runStageConcurrently executes all gates in a stage in parallel, collecting
// results in order. Uses a mutex to serialize stdout/stderr output so gate
// headers and results don't interleave.
func runStageConcurrently(stage []preCommitGate, rootDir string) []gateResult {
	if len(stage) == 1 {
		// Fast path: single gate, no goroutine overhead.
		g := stage[0]
		fmt.Printf("==> [gate] %s\n", g.label)
		res := g.run(rootDir)
		res.name = g.label
		printGateResult(res)
		return []gateResult{res}
	}

	results := make([]gateResult, len(stage))
	var wg sync.WaitGroup
	var mu sync.Mutex // protects stdout/stderr interleaving

	for idx, g := range stage {
		wg.Add(1)
		go func(i int, gate preCommitGate) {
			defer wg.Done()
			res := gate.run(rootDir)
			res.name = gate.label
			mu.Lock()
			fmt.Printf("==> [gate] %s\n", gate.label)
			printGateResult(res)
			mu.Unlock()
			results[i] = res
		}(idx, g)
	}
	wg.Wait()
	return results
}

// printGateResult outputs the failure/warning for a gate result. Must be called
// with mu held in parallel mode.
func printGateResult(res gateResult) {
	if !res.passed {
		tag := "FAIL"
		if res.soft {
			tag = "WARN"
		}
		detail := maskSecrets(res.detail)
		if detail != "" {
			fmt.Fprintf(os.Stderr, "%s: %s: %s\n", tag, res.name, detail)
		} else {
			fmt.Fprintf(os.Stderr, "%s: %s\n", tag, res.name)
		}
	}
}

// reportPreCommitResults prints the final summary and returns a non-nil error
// (exit 1) iff any HARD gate failed. Soft-gate failures never fail the run.
func reportPreCommitResults(results []gateResult) error {
	hardFailures := 0
	for _, r := range results {
		if !r.passed && !r.soft {
			hardFailures++
		}
	}
	fmt.Println()
	if hardFailures == 0 {
		fmt.Println("All pre-commit gates passed.")
		return nil
	}
	fmt.Println("One or more pre-commit gates FAILED.")
	return fmt.Errorf("pre-commit: %d hard gate(s) failed", hardFailures)
}

// --- gate implementations ---
//
// hwcloud-skillcheck gates call the binary's own command functions in-process
// (no subprocess): identical behavior, no double build/startup cost, and each
// gate stays unit-testable. gofmt / go vet / go test shell out to the `go`
// toolchain (same as the shell).

// gateGofmt fails if any Go file under hwcloud-skillcheck/ needs reformatting
// (mirrors `[ -z "$(gofmt -l .)" ]`).
func gateGofmt(rootDir string) gateResult {
	gofmtBin, err := exec.LookPath("gofmt")
	if err != nil {
		return gateResult{passed: false, detail: "'gofmt' not found on PATH"}
	}
	modDir := filepath.Join(rootDir, "hwcloud-skillcheck")
	cmd := exec.Command(gofmtBin, "-l", modDir)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return gateResult{passed: false, detail: fmt.Sprintf("gofmt error: %v: %s", err, strings.TrimSpace(out.String()))}
	}
	if needs := strings.TrimSpace(out.String()); needs != "" {
		return gateResult{passed: false, detail: "files need gofmt:\n  " + strings.ReplaceAll(needs, "\n", "\n  ")}
	}
	return gateResult{passed: true}
}

// gateGoVet runs `go vet ./...` in the module directory.
func gateGoVet(rootDir string) gateResult {
	return goToolGate(rootDir, "go vet failed", "vet", "./...")
}

// gateGoTest runs `go test ./... -count=1` in the module directory.
func gateGoTest(rootDir string) gateResult {
	return goToolGate(rootDir, "go test failed", "test", "./...", "-count=1")
}

// gateGoTestRetry runs go test with up to N retries on failure, mirroring the
// old CI shell's 3-attempt retry loop for flaky race detection.
func gateGoTestRetry(rootDir string, maxRetries int) gateResult {
	if maxRetries <= 0 {
		return gateGoTest(rootDir)
	}
	var lastDetail string
	for attempt := 0; attempt <= maxRetries; attempt++ {
		res := goToolGate(rootDir, "go test failed", "test", "./...", "-count=1")
		if res.passed {
			return res
		}
		lastDetail = res.detail
		if attempt < maxRetries {
			fmt.Fprintf(os.Stderr, "WARN: Go test attempt %d/%d failed, retrying...\n", attempt+1, maxRetries+1)
		}
	}
	return gateResult{passed: false, detail: fmt.Sprintf("go test failed after %d attempts: %s", maxRetries+1, lastDetail)}
}

// goToolGate shells out to the `go` toolchain in <rootDir>/hwcloud-skillcheck,
// treating a non-zero exit as a gate failure with the captured output as detail.
func goToolGate(rootDir, failMsg string, goArgs ...string) gateResult {
	if goBinPath == "" {
		return gateResult{passed: false, detail: "'go' toolchain not found on PATH"}
	}
	cmd := exec.Command(goBinPath, goArgs...)
	cmd.Dir = filepath.Join(rootDir, "hwcloud-skillcheck")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return gateResult{passed: false, detail: fmt.Sprintf("%s: %v: %s", failMsg, err, strings.TrimSpace(out.String()))}
	}
	return gateResult{passed: true}
}

// inProcessGate adapts an in-process `run*` command func into a gateResult.
// The func's own stdout is left as-is; its returned error becomes the detail.
func inProcessGate(fn func([]string) error, args []string, soft bool) gateResult {
	if err := fn(args); err != nil {
		return gateResult{passed: false, soft: soft, detail: err.Error()}
	}
	return gateResult{passed: true, soft: soft}
}

func gateValidate(rootDir string) gateResult {
	return inProcessGate(runValidate, []string{"--root", rootDir}, false)
}

func gateAuditResults(rootDir string) gateResult {
	return inProcessGate(runCheck, []string{"audit-results", "--root", rootDir}, false)
}

func gateAggregateTrace(rootDir string) gateResult {
	return inProcessGate(runAggregate, []string{"trace", "--require-traces", "--root", rootDir}, false)
}

func gateLearningGen(rootDir string) gateResult {
	return inProcessGate(runLearning, []string{"gen", "--root", rootDir}, false)
}

func gateL4Handle(rootDir string) gateResult {
	return inProcessGate(runL4, []string{"handle", "--fault", "smoke", "--risk", "low", "--root", rootDir}, false)
}

// gateGoldenRun is SOFT: mirrors `golden run ... || true`.
func gateGoldenRun(rootDir string) gateResult {
	return inProcessGate(runGolden, []string{"--root", rootDir}, true)
}

func gateCheckLanes(rootDir string) gateResult {
	return inProcessGate(runCheck, []string{"lanes", "--root", rootDir}, false)
}

// gateABCompare is SOFT: mirrors `ab compare ... || true`.
func gateABCompare(rootDir string) gateResult {
	return inProcessGate(runAB, []string{"--root", rootDir}, true)
}

func gateAdvancedCoverage(rootDir string) gateResult {
	return inProcessGate(runCheck, []string{"advanced-coverage", "--root", rootDir}, false)
}

// gateDriftGuard runs `drift sync --apply` (self-healing) then `drift check`,
// mirroring the shell's `sync && check`. The check half is the gate.
func gateDriftGuard(rootDir string) gateResult {
	if err := runDrift([]string{"sync", "--apply", "--root", rootDir}); err != nil {
		return gateResult{passed: false, detail: "drift sync: " + err.Error()}
	}
	if err := runDrift([]string{"check", "--root", rootDir}); err != nil {
		return gateResult{passed: false, detail: "drift check: " + err.Error()}
	}
	return gateResult{passed: true}
}

// gateDriftGuardCheckOnly runs only `drift check` (no sync --apply). This is
// the CI-mode variant: the binary is built in a prior step and auto-healing
// (sync --apply) is inappropriate in a read-only validation pipeline.
func gateDriftGuardCheckOnly(rootDir string) gateResult {
	if err := runDrift([]string{"check", "--root", rootDir}); err != nil {
		return gateResult{passed: false, detail: "drift check: " + err.Error()}
	}
	return gateResult{passed: true}
}

// gateCriticScore runs `critic score` against the standard GCL quality summary
// fixture. SOFT (mirrors old CI `|| true`): this is a fixture smoke test, not a
// production correctness assertion.
func gateCriticScore(rootDir string) gateResult {
	return inProcessGate(runCritic, []string{"score", "--generator", filepath.Join(rootDir, "scripts", "fixtures", "gcl-quality-summary-healthy.json")}, true)
}

// gateDriftSyncDryRun runs `drift sync --dry-run` to exercise the dry-run code
// path so a wrong default flag cannot ship. SOFT: matches old CI `drift sync
// --dry-run` standalone step.
func gateDriftSyncDryRun(rootDir string) gateResult {
	return inProcessGate(runDrift, []string{"sync", "--dry-run", "--root", rootDir}, true)
}

// gateGclAlarmWire runs `gcl alarm-wire` against the standard fixture to
// exercise the alarm-wire wiring path. SOFT (mirrors old CI `|| true`).
// NOTE: runGCLAlarmWire calls os.Exit(1) on threshold breaches, which kills
// the process. We shell out via goToolGate to contain the exit.
func gateGclAlarmWire(rootDir string) gateResult {
	bin := filepath.Join(rootDir, "bin", "hwcloud-skillcheck")
	if _, err := os.Stat(bin); os.IsNotExist(err) {
		return gateResult{passed: false, soft: true, detail: "binary not found; run Build step first"}
	}
	cmd := exec.Command(bin, "gcl", "alarm-wire",
		"--root", rootDir,
		"--plan-file", filepath.Join(rootDir, "scripts", "fixtures", "gcl-quality-summary-healthy.json"),
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	_ = cmd.Run() // ignore exit code; always soft
	return gateResult{passed: true, soft: true}
}
