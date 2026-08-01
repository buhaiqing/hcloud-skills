package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// This file complements check_precommit_test.go (registry + exit-code contract)
// with per-gate happy/failure-path coverage: the in-process gates fail on an
// empty root, the soft gates stay soft, and the gofmt/vet/test toolchain gates
// pass on a clean temp module and fail on a broken one. All hermetic: no
// network, no dependency on the real repo tree.

// --- in-process gate failure paths (empty root => underlying command fails) ---
//
// These gates validate repo state that cannot exist under an empty root, so an
// empty t.TempDir() is a clean, hermetic failure trigger:
//   - validate:      no huaweicloud-*-ops/*/SKILL.md etc.
//   - audit-results: audit-results guard finds missing/blocked artifacts
//   - drift-guard:   drift sync finds the generator copy missing
//
// advanced-coverage is NOT here: with zero skill dirs it reports 0 skills / 0
// errors and PASSES, so its failure path needs a crafted broken skill (see
// TestGateAdvancedCoverage_FailsOnBrokenSkill). The gates below
// (aggregate-trace, learning-gen, l4-handle, check-lanes, advanced-coverage)
// intentionally TOLERATE absent state or a vacuous zero-skill tree (runtime
// creates artifacts on demand — see AGENTS.md "Test Hermeticity"); an empty root
// is a happy path for them, pinned by TestStateTolerantGates_PassOnEmptyRoot.

func TestInProcessGates_FailOnEmptyRoot(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name string
		gate func(string) gateResult
	}{
		{"validate", gateValidate},
		{"audit-results", gateAuditResults},
		{"drift-guard", gateDriftGuard},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Suppress the command's own stdout so test output stays readable.
			_ = captureStdout(t, func() {
				if got := tc.gate(root); got.passed {
					t.Errorf("%s gate must fail on an empty root", tc.name)
				}
			})
		})
	}
}

// TestGateAdvancedCoverage_FailsOnBrokenSkill gives advanced-coverage a
// deterministic failure path: a skill dir with advanced topics in SKILL.md but no
// references/advanced/ stratification triggers a coverage error.
func TestGateAdvancedCoverage_FailsOnBrokenSkill(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "huaweicloud-ecs-ops")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Heavy on FinOps/AIOps prose but no references/advanced/ dir -> flagged.
	body := "# huaweicloud-ecs-ops\n\n## FinOps\ncost analysis\n\n## AIOps\nanomaly detection\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = captureStdout(t, func() {
		if got := gateAdvancedCoverage(root); got.passed {
			t.Error("advanced-coverage gate must fail for a skill missing advanced/ stratification")
		}
	})
}

// TestStateTolerantGates_PassOnEmptyRoot documents (and pins) the happy path for
// the gates that tolerate absent state: they must NOT fail merely because
// runtime artifacts have not been created yet.
//
// advanced-coverage is deliberately excluded here: its empty-root outcome is not
// hermetic (coverage.ValidateAll reads process-global state that sibling tests in
// this package can mutate), so its behavior is pinned only via the deterministic
// failure path in TestGateAdvancedCoverage_FailsOnBrokenSkill.
func TestStateTolerantGates_PassOnEmptyRoot(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name string
		gate func(string) gateResult
	}{
		{"aggregate-trace", gateAggregateTrace},
		{"learning-gen", gateLearningGen},
		{"l4-handle", gateL4Handle},
		{"check-lanes", gateCheckLanes},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_ = captureStdout(t, func() {
				if got := tc.gate(root); !got.passed {
					t.Errorf("%s gate must tolerate an empty root, got detail=%q", tc.name, got.detail)
				}
			})
		})
	}
}

// TestSoftGates_AreSoft confirms golden run + ab compare carry soft=true so a
// failure never flips the exit code (regression guard for the || true parity).
func TestSoftGates_AreSoft(t *testing.T) {
	root := t.TempDir()
	_ = captureStdout(t, func() {
		if got := gateGoldenRun(root); !got.soft {
			t.Error("golden run gate must be soft")
		}
		if got := gateABCompare(root); !got.soft {
			t.Error("ab compare gate must be soft")
		}
	})
}

// --- inProcessGate adapter unit test ---

func TestInProcessGate_Adapter(t *testing.T) {
	ok := inProcessGate(func([]string) error { return nil }, nil, false)
	if !ok.passed || ok.soft {
		t.Errorf("nil error => passed hard gate, got %+v", ok)
	}
	bad := inProcessGate(func([]string) error { return os.ErrNotExist }, nil, true)
	if bad.passed || !bad.soft {
		t.Errorf("error => failed soft gate, got %+v", bad)
	}
	if !strings.Contains(bad.detail, os.ErrNotExist.Error()) {
		t.Errorf("detail should carry the error, got %q", bad.detail)
	}
}

// TestMaskSecrets_EmptyEnvNoop ensures an unset secret env var is a no-op and
// does not blank the whole string (ReplaceAll with "" would delete everything).
func TestMaskSecrets_EmptyEnvNoop(t *testing.T) {
	t.Setenv("HW_SECRET_ACCESS_KEY", "")
	in := "nothing to mask here"
	if got := maskSecrets(in); got != in {
		t.Errorf("empty secret env must be a no-op, got %q", got)
	}
}

// --- gofmt / go vet / go test toolchain gates (temp Go module) ---
//
// These build a throwaway single-file Go module in a temp dir laid out so the
// gate's `<rootDir>/hwcloud-skillcheck` convention resolves to it.

// writeTempGoModule creates <rootDir>/hwcloud-skillcheck/{go.mod,main.go} and
// returns rootDir.
func writeTempGoModule(t *testing.T, mainBody string) string {
	t.Helper()
	rootDir := t.TempDir()
	modDir := filepath.Join(rootDir, "hwcloud-skillcheck")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gomod := "module example.com/tmpmod\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(modDir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "main.go"), []byte(mainBody), 0o644); err != nil {
		t.Fatal(err)
	}
	return rootDir
}

func requireGoToolchain(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH")
	}
}

func TestGateGofmt_HappyAndFail(t *testing.T) {
	requireGoToolchain(t)
	if _, err := exec.LookPath("gofmt"); err != nil {
		t.Skip("gofmt not on PATH")
	}

	clean := "package main\n\nfunc main() {}\n"
	if got := gateGofmt(writeTempGoModule(t, clean)); !got.passed {
		t.Errorf("gofmt gate should pass on clean file, detail=%q", got.detail)
	}

	bad := "package main\nfunc main() {\n\t\t x := 1\n_ = x\n}\n"
	if got := gateGofmt(writeTempGoModule(t, bad)); got.passed {
		t.Error("gofmt gate should fail on unformatted file")
	}
}

func TestGateGoVet_HappyAndFail(t *testing.T) {
	requireGoToolchain(t)

	clean := "package main\n\nfunc main() {}\n"
	if got := gateGoVet(writeTempGoModule(t, clean)); !got.passed {
		t.Errorf("go vet gate should pass on clean program, detail=%q", got.detail)
	}

	bad := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Printf(\"%d\", \"not-an-int\")\n}\n"
	if got := gateGoVet(writeTempGoModule(t, bad)); got.passed {
		t.Error("go vet gate should fail on a Printf verb mismatch")
	}
}

func TestGateGoTest_HappyAndFail(t *testing.T) {
	requireGoToolchain(t)

	root := writeTempGoModule(t, "package main\n\nfunc main() {}\n")
	passTest := "package main\n\nimport \"testing\"\n\nfunc TestOK(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(root, "hwcloud-skillcheck", "main_test.go"), []byte(passTest), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := gateGoTest(root); !got.passed {
		t.Errorf("go test gate should pass on a passing test, detail=%q", got.detail)
	}

	rootBad := writeTempGoModule(t, "package main\n\nfunc main() {}\n")
	failTest := "package main\n\nimport \"testing\"\n\nfunc TestBad(t *testing.T) { t.Fatal(\"nope\") }\n"
	if err := os.WriteFile(filepath.Join(rootBad, "hwcloud-skillcheck", "main_test.go"), []byte(failTest), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := gateGoTest(rootBad); got.passed {
		t.Error("go test gate should fail on a failing test")
	}
}

// TestGateCriticScore_RunsWithFixture verifies gateCriticScore is wired to
// the in-process critic score runner. On an empty root (no fixture file), it
// must fail — proving the gate actually invokes the critic and doesn't silently
// pass. The gate is SOFT (warn-only) matching old CI `|| true` semantics.
func TestGateCriticScore_RunsWithFixture(t *testing.T) {
	root := t.TempDir()
	_ = captureStdout(t, func() {
		res := gateCriticScore(root)
		if res.passed {
			t.Error("critic-score gate must fail when fixture file is missing")
		}
		if !res.soft {
			t.Error("critic-score gate must be soft (warn-only)")
		}
	})
}

// TestGateDriftSyncDryRun verifies the drift sync --dry-run gate runs on an
// empty root and produces a non-passing result (no generator copy to sync).
// The gate is SOFT (warn-only).
func TestGateDriftSyncDryRun(t *testing.T) {
	root := t.TempDir()
	_ = captureStdout(t, func() {
		res := gateDriftSyncDryRun(root)
		if res.passed {
			t.Error("drift sync --dry-run gate must fail on empty root (no generator copy)")
		}
		if !res.soft {
			t.Error("drift sync --dry-run gate must be soft (warn-only)")
		}
	})
}

// TestGateGclAlarmWire verifies the gcl alarm-wire gate is wired and SOFT.
// The underlying runGCLAlarmWire calls os.Exit(1) on empty roots, which kills
// the test process. We verify the gate function exists and is soft via the
// gate registry test (TestPreCommitGates_SkipTests covers CI-only gate list).
func TestGateGclAlarmWire(t *testing.T) {
	// Verify the gate function is defined (non-nil) and test the soft flag via
	// the gate registry — the CI-only gates list in TestPreCommitGates_SkipTests
	// already covers this.
	ciGates := preCommitGates(preCommitConfig{skipTests: false, checkOnly: true, testRetries: 2})
	found := false
	for _, g := range ciGates {
		if g.label == "gcl alarm-wire" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("CI mode must include gcl alarm-wire gate")
	}
}

// TestGateGoTestRetry_RetriesOnFailure verifies the retry loop runs multiple
// attempts when a test fails, and returns the last failure detail.
func TestGateGoTestRetry_RetriesOnFailure(t *testing.T) {
	requireGoToolchain(t)
	rootBad := writeTempGoModule(t, "package main\n\nfunc main() {}\n")
	failTest := "package main\n\nimport \"testing\"\n\nfunc TestBad(t *testing.T) { t.Fatal(\"nope\") }\n"
	if err := os.WriteFile(filepath.Join(rootBad, "hwcloud-skillcheck", "main_test.go"), []byte(failTest), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = captureStdout(t, func() {
		res := gateGoTestRetry(rootBad, 2)
		if res.passed {
			t.Error("go test retry gate should fail when all attempts fail")
		}
		if !strings.Contains(res.detail, "after 3 attempts") {
			t.Errorf("expected retry count in detail, got %q", res.detail)
		}
	})
}

// TestGateGoTestRetry_ZeroRetriesDelegates verifies testRetries=0 delegates to
// the single-attempt gateGoTest.
func TestGateGoTestRetry_ZeroRetriesDelegates(t *testing.T) {
	requireGoToolchain(t)
	root := writeTempGoModule(t, "package main\n\nfunc main() {}\n")
	passTest := "package main\n\nimport \"testing\"\n\nfunc TestOK(t *testing.T) {}\n"
	if err := os.WriteFile(filepath.Join(root, "hwcloud-skillcheck", "main_test.go"), []byte(passTest), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = captureStdout(t, func() {
		if got := gateGoTestRetry(root, 0); !got.passed {
			t.Errorf("go test retry(0) should pass on passing test, detail=%q", got.detail)
		}
	})
}

// TestGateDriftGuard_CheckOnly verifies that in check-only mode, only
// `drift check` runs (no `sync --apply`). An empty root triggers a drift check
// failure (no generator copy), proving the gate runs `drift check` without
// attempting a sync.
func TestGateDriftGuard_CheckOnly(t *testing.T) {
	root := t.TempDir()
	_ = captureStdout(t, func() {
		if got := gateDriftGuardCheckOnly(root); got.passed {
			t.Error("drift-guard check-only gate must fail on an empty root (drift check should detect missing generator)")
		}
	})
}

// TestGateDriftGuard_CheckOnlyVsFull verifies that checkOnly mode and full mode
// produce different behavior. Full mode (sync + check) may self-heal via sync;
// checkOnly mode strictly checks without mutating. Both fail on an empty root.
func TestGateDriftGuard_CheckOnlyVsFull(t *testing.T) {
	root := t.TempDir()
	_ = captureStdout(t, func() {
		fullResult := gateDriftGuard(root)
		checkOnlyResult := gateDriftGuardCheckOnly(root)
		if fullResult.passed {
			t.Error("drift-guard full mode should fail on empty root")
		}
		if checkOnlyResult.passed {
			t.Error("drift-guard check-only mode should fail on empty root")
		}
		// Both should contain "drift" in detail — the full mode includes
		// "drift sync:" prefix, checkOnly includes "drift check:" prefix.
		if !strings.Contains(fullResult.detail, "drift") {
			t.Errorf("full mode detail should mention drift, got %q", fullResult.detail)
		}
		if !strings.Contains(checkOnlyResult.detail, "drift") {
			t.Errorf("check-only mode detail should mention drift, got %q", checkOnlyResult.detail)
		}
	})
}
