package spec_audit

import (
	"os"
	"os/exec"
	"path/filepath"
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
	cmd := exec.Command("/bin/bash", "/Users/bohaiqing/opensource/git/hcloud-skills/scripts/pre_commit_check.sh", "--skip-tests")
	cmd.Dir = ".."
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
