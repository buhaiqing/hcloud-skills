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
// These gates validate repo state that cannot exist under an empty root, so
// an empty t.TempDir() is a clean, hermetic failure trigger. advanced-coverage
// is NOT here: it is a coverage survey that returns OK on a zero-skill tree
// (no errors to report), so it tolerates an empty root like the gates below.
// aggregate-trace, learning-gen, l4-handle, check-lanes intentionally TOLERATE
// absent state (runtime creates artifacts on demand — see AGENTS.md
// "Test Hermeticity"); an empty root is a happy path for them.

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
	// advanced-coverage is intentionally excluded: coverage.ValidateAll's
	// behavior on an empty tree depends on process-level state (cwd/env), so an
	// empty t.TempDir() is not a stable hermetic trigger. It is covered by the
	// registry test (TestPreCommitGates_SkipTests) and the real repo in CI.
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

// TestStateTolerantGates_PassOnEmptyRoot documents (and pins) the happy path for
// the gates that tolerate absent state: they must NOT fail merely because
// runtime artifacts have not been created yet.
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
