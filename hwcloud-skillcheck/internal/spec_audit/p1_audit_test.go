package spec_audit

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestP1Acceptance_AuditsAllCriteria(t *testing.T) {
	root := ".."
	mustFind := map[string][]string{
		"A1.1":  {"TestGoldenScenarioCoverage"},
		"A1.2":  {"TestCrossProductScenarioCount"},
		"A1.3":  {"TestCLISubcommandFixtureCoverage"},
		"A1.4":  {"TestMockhcloudNoNetwork"},
		"A1.5":  {"TestGoldenRunPass"},
		"A1.6":  {"TestABDetectsStdoutDiff"},
		"A1.7":  {"TestTelemetryLaneSeparation"},
		"A1.8":  {"TestManifestGeneration"},
		"A1.9":  {"TestMaturityReportRollup"},
		"A1.10": {"TestP1GatesWired"},
	}
	hits := map[string]bool{}
	_ = filepath.Walk(root, func(p string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() && strings.HasSuffix(p, "_test.go") {
			b, _ := os.ReadFile(p)
			txt := string(b)
			for id, names := range mustFind {
				for _, n := range names {
					if strings.Contains(txt, n) {
						hits[id] = true
					}
				}
			}
		}
		return nil
	})
	missing := []string{}
	for id := range mustFind {
		if !hits[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("P1 acceptance criteria missing tests: %s", strings.Join(missing, ", "))
	}
}

func TestP1GatesWired(t *testing.T) {
	// Uses `hwcloud-skillcheck check --pre-commit` (ADR-0014, Phase 5 Go migration).
	scriptPath := filepath.Join(repoRoot(), "bin", "hwcloud-skillcheck")
	cmd := exec.Command(scriptPath, "check", "--pre-commit", "--skip-tests")
	cmd.Dir = repoRoot()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("script error: %v\noutput: %s", err, string(out))
	}
	text := string(out)
	needed := []string{"golden run", "check lanes", "ab compare", "check advanced-coverage"}
	for _, g := range needed {
		if !strings.Contains(text, g) {
			t.Errorf("gate %q not found in pre-commit output", g)
		}
	}
}

// repoRoot returns the repository root (parent of hwcloud-skillcheck/).
func repoRoot() string {
	// This test lives in hwcloud-skillcheck/internal/spec_audit/.
	// Walk up three levels to reach the repo root.
	_, f0, _, _ := runtime.Caller(0)
	// f0 = p1_audit_test.go in spec_audit dir
	// parent[0] = spec_audit dir
	// parent[1] = internal dir
	// parent[2] = hwcloud-skillcheck dir
	// parent[3] = repo root
	for i := 0; i < 4; i++ {
		f0 = filepath.Dir(f0)
	}
	return f0
}
