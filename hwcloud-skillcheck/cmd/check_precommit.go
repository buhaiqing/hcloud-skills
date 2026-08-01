package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// preCommitGate binds a gate label to the function that runs it. Splitting the
// registry from the runner keeps the exit-code contract independently testable
// (a fake registry can be substituted in tests).
type preCommitGate struct {
	label string
	run   func(rootDir string) gateResult
}

// maskSecrets scrubs anything resembling a Huawei Cloud credential from gate
// detail strings so no secret can leak into stdout/stderr. Defense-in-depth:
// gates shell out to `go` / call in-process funcs that should never surface
// secrets, but a rogue error message must still be masked.
var secretEnvKeys = []string{
	"HW_SECRET_ACCESS_KEY", "HW_ACCESS_KEY_ID",
	"TENCENTCLOUD_SECRET_KEY", "AWS_SECRET_ACCESS_KEY",
}

func maskSecrets(s string) string {
	for _, key := range secretEnvKeys {
		if v := os.Getenv(key); v != "" {
			s = strings.ReplaceAll(s, v, "***")
		}
	}
	return s
}

// runCheckPreCommit implements `hwcloud-skillcheck check --pre-commit
// [--skip-tests]`. It replaces scripts/pre_commit_check.sh: the same 13 gates,
// the same hard/soft semantics, and the same exit-code contract (1 on any hard
// failure, 0 otherwise). See ADR-0014.
func runCheckPreCommit(args []string) error {
	fs := newFlagSet("hwcloud-skillcheck check --pre-commit")
	root := fs.String("root", ".", "skill repository root")
	skipTests := fs.Bool("skip-tests", false, "skip the go test gate (git hook passes this; CI does not)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := resolveRepoRoot(*root)
	if err != nil {
		return err
	}

	// Defense-in-depth build: rebuild ./bin/hwcloud-skillcheck so a stale
	// binary cannot drift from current source (mirrors shell line 39). The
	// gates themselves call in-process funcs, so the built binary is only a
	// safety net for external callers; skip it with SKILLCHECK_SKIP_BUILD=1.
	gates := preCommitGates(*skipTests)
	results := runPreCommitGates(gates, rootDir)

	// Defense-in-depth build: rebuild ./bin/hwcloud-skillcheck so a stale
	// binary cannot drift from current source (mirrors shell line 39). The
	// gates themselves call in-process funcs, so the built binary is only a
	// safety net for external callers; skip it with SKILLCHECK_SKIP_BUILD=1.
	// A build failure is surfaced as a synthetic hard gate result so the
	// summary and exit code stay consistent with the gate contract.
	if os.Getenv("SKILLCHECK_SKIP_BUILD") != "1" {
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
	goBin, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("'go' toolchain not found on PATH")
	}
	out := filepath.Join(rootDir, "bin", "hwcloud-skillcheck")
	cmd := exec.Command(goBin, "build", "-trimpath", "-o", out, ".")
	cmd.Dir = filepath.Join(rootDir, "hwcloud-skillcheck")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(buf.String()))
	}
	return nil
}

// preCommitGates returns the ordered 13-gate registry. Gates 8 (golden run)
// and 10 (ab compare) are soft (warn but never fail). Gate 13 (go test) is
// omitted when skipTests is set.
func preCommitGates(skipTests bool) []preCommitGate {
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
		{"skill_generator drift guard", gateDriftGuard},
	}
	if !skipTests {
		gates = append(gates, preCommitGate{"Go test", gateGoTest})
	}
	return gates
}

// runPreCommitGates executes each gate in order, printing the shell-compatible
// `==> [gate] <label>` header before running it.
func runPreCommitGates(gates []preCommitGate, rootDir string) []gateResult {
	results := make([]gateResult, 0, len(gates))
	for _, g := range gates {
		fmt.Printf("==> [gate] %s\n", g.label)
		res := g.run(rootDir)
		res.name = g.label
		if !res.passed {
			tag := "FAIL"
			if res.soft {
				tag = "WARN"
			}
			detail := maskSecrets(res.detail)
			if detail != "" {
				fmt.Fprintf(os.Stderr, "%s: %s: %s\n", tag, g.label, detail)
			} else {
				fmt.Fprintf(os.Stderr, "%s: %s\n", tag, g.label)
			}
		}
		results = append(results, res)
	}
	return results
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

// goToolGate shells out to the `go` toolchain in <rootDir>/hwcloud-skillcheck,
// treating a non-zero exit as a gate failure with the captured output as detail.
func goToolGate(rootDir, failMsg string, goArgs ...string) gateResult {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return gateResult{passed: false, detail: "'go' toolchain not found on PATH"}
	}
	cmd := exec.Command(goBin, goArgs...)
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
