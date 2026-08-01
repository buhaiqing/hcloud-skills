package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStdout redirects stdout to a pipe, runs fn, restores stdout, and returns
// what was printed. Uses t.Fatal on any pipe/setup error so tests abort cleanly.
func captureStdout(t *testing.T, fn func()) string {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	os.Stdout = old
	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// --- check example-config ---

func writeExampleConfig(t *testing.T, root, skill, content string) {
	t.Helper()
	dir := filepath.Join(root, skill, "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "example-config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckExampleConfigClean(t *testing.T) {
	root := t.TempDir()
	writeExampleConfig(t, root, "huaweicloud-ecs-ops", "region: cn-north-4\nimage_id: {{env.HW_IMAGE_ID}}\n")
	if err := runCheck([]string{"example-config", "--root", root}); err != nil {
		t.Fatalf("clean example-config should pass, got: %v", err)
	}
}

func TestDiscoverSkillDirsMissingRoot(t *testing.T) {
	// WR-03: ReadDir failure must surface (not silently succeed with 0 skills).
	_, err := discoverSkillDirs(filepath.Join(t.TempDir(), "no-such-dir"))
	if err == nil {
		t.Fatal("expected error for missing root")
	}
	if err := runCheck([]string{"example-config", "--root", filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Fatal("example-config on missing root must fail")
	}
}

func TestCheckExampleConfigPlaintextSecret(t *testing.T) {
	root := t.TempDir()
	// A real secret literal (secret: "value") must fail.
	writeExampleConfig(t, root, "huaweicloud-ecs-ops", "secret: \"supersecret123\"\n")
	if err := runCheck([]string{"example-config", "--root", root}); err == nil {
		t.Fatal("plaintext secret should fail example-config")
	}
}

func TestCheckExampleConfigAnchorBeforeDefined(t *testing.T) {
	root := t.TempDir()
	// *base referenced before &base defined.
	writeExampleConfig(t, root, "huaweicloud-ecs-ops", "defaults: *base\nbase: &base\n  region: cn-north-4\n")
	if err := runCheck([]string{"example-config", "--root", root}); err == nil {
		t.Fatal("anchor referenced before defined should fail")
	}
}

func TestCheckExampleConfigWarnOnly(t *testing.T) {
	root := t.TempDir()
	// A plaintext secret is an error; --warn-only must downgrade it to a
	// warning so the command still exits 0.
	writeExampleConfig(t, root, "huaweicloud-ecs-ops", "secret: \"supersecret123\"\n")
	if err := runCheck([]string{"example-config", "--root", root, "--warn-only"}); err != nil {
		t.Fatalf("--warn-only should downgrade failure to warning (exit 0), got: %v", err)
	}
}

func TestCheckExampleConfigMissingFile(t *testing.T) {
	root := t.TempDir()
	// Skill dir without assets/example-config.yaml => error (missing file).
	if err := os.MkdirAll(filepath.Join(root, "huaweicloud-ecs-ops"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := runCheck([]string{"example-config", "--root", root}); err == nil {
		t.Fatal("missing example-config.yaml should fail")
	}
}

// --- check markdown-links ---

func writeMD(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckMarkdownLinksGood(t *testing.T) {
	root := t.TempDir()
	writeMD(t, root, "README.md", "# Title\n[doc](docs/guide.md)\n")
	writeMD(t, root, "docs/guide.md", "# Guide\n")
	if err := runCheck([]string{"markdown-links", "--root", root}); err != nil {
		t.Fatalf("valid relative link should pass, got: %v", err)
	}
}

func TestCheckMarkdownLinksBroken(t *testing.T) {
	root := t.TempDir()
	writeMD(t, root, "README.md", "# Title\n[doc](docs/missing.md)\n")
	if err := runCheck([]string{"markdown-links", "--root", root}); err == nil {
		t.Fatal("broken relative link should fail")
	}
}

func TestCheckMarkdownLinksExternalIgnored(t *testing.T) {
	root := t.TempDir()
	writeMD(t, root, "README.md", "# Title\n[ext](https://example.com)\n")
	if err := runCheck([]string{"markdown-links", "--root", root}); err != nil {
		t.Fatalf("external link should be ignored, got: %v", err)
	}
}

// --- check references-links ---

func writeRefMD(t *testing.T, root, skill, name, content string) {
	t.Helper()
	path := filepath.Join(root, skill, "references", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckReferencesLinksGood(t *testing.T) {
	root := t.TempDir()
	writeRefMD(t, root, "huaweicloud-ecs-ops", "a.md", "# Section One\n[link](b.md#section-two)\n")
	writeRefMD(t, root, "huaweicloud-ecs-ops", "b.md", "## Section Two\n")
	if err := runCheck([]string{"references-links", "--root", root}); err != nil {
		t.Fatalf("valid anchor link should pass, got: %v", err)
	}
}

func TestCheckReferencesLinksBadAnchor(t *testing.T) {
	root := t.TempDir()
	writeRefMD(t, root, "huaweicloud-ecs-ops", "a.md", "[link](b.md#nonexistent)\n")
	writeRefMD(t, root, "huaweicloud-ecs-ops", "b.md", "## Section Two\n")
	if err := runCheck([]string{"references-links", "--root", root}); err == nil {
		t.Fatal("missing anchor should fail")
	}
}

func TestCheckReferencesLinksJSON(t *testing.T) {
	root := t.TempDir()
	writeRefMD(t, root, "huaweicloud-ecs-ops", "a.md", "# Section One\n")
	// JSON output must succeed and report ok.
	if err := runCheck([]string{"references-links", "--root", root, "--json"}); err != nil {
		t.Fatalf("references-links --json should pass, got: %v", err)
	}
}

// TestCheckAdvancedCoverageJSON verifies that --json flag produces valid JSON
// with expected top-level fields and a non-empty reports array.
func TestCheckAdvancedCoverageJSON(t *testing.T) {
	root := t.TempDir()
	scaffoldSkillTree(t, root, "huaweicloud-ecs-ops",
		map[string]string{"aiops-patterns.md": "finops content"},
		map[string]string{"runbook.md": "some doc"})

	err := runCheckAdvancedCoverage([]string{"--root", root, "--json"})
	if err != nil {
		t.Fatalf("advanced-coverage --json should not fail, got: %v", err)
	}

	output := captureStdout(t, func() {
		_ = runCheckAdvancedCoverage([]string{"--root", root, "--json"})
	})

	for _, field := range []string{`"ok"`, `"skills_checked"`, `"skills_with_advanced"`, `"reports"`} {
		if !strings.Contains(output, field) {
			t.Errorf("JSON output missing field %q; got:\n%s", field, output)
		}
	}
	if !strings.Contains(output, `"skill"`) {
		t.Errorf("JSON output missing skill entry; got:\n%s", output)
	}
	output = strings.TrimSpace(output)
	if !strings.HasPrefix(output, "{") {
		t.Errorf("JSON output should start with '{{'; got: %s", output[:min(10, len(output))])
	}
}

// TestCheckAdvancedCoverageJSONFail verifies that when a skill is missing
// advanced/, the JSON still prints all required top-level fields (even though OK=true
// due to warn-only suppressing all errors).
func TestCheckAdvancedCoverageJSONFail(t *testing.T) {
	root := t.TempDir()
	scaffoldSkillTree(t, root, "huaweicloud-ecs-ops", nil,
		map[string]string{"runbook.md": "some doc"})

	output := captureStdout(t, func() {
		_ = runCheckAdvancedCoverage([]string{"--root", root, "--json", "--warn-only"})
	})

	for _, field := range []string{`"ok"`, `"skills_checked"`, `"errors"`, `"warnings"`, `"reports"`} {
		if !strings.Contains(output, field) {
			t.Errorf("JSON output missing field %q; got:\n%s", field, output)
		}
	}
	if !strings.Contains(output, `"skills_checked": 1`) {
		t.Errorf("skills_checked should be 1; got:\n%s", output)
	}
}

// --- looksLikeRepoPath (Go SDK module path false-positive fix) ---

func TestLooksLikeRepoPath(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"sdk module path", "huaweicloud-sdk-go-v3/services/ecs/v2", false},
		{"obs sibling module", "huaweicloud-sdk-go-obs", false},
		{"bare sdk module no slash", "huaweicloud-sdk-go-v3", false},
		{"real repo path", "huaweicloud-ecs-ops/SKILL.md", true},
		{"full module path", "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cbr/v3", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := looksLikeRepoPath(tc.text); got != tc.want {
				t.Errorf("looksLikeRepoPath(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}

func TestCheckMarkdownFileSkipsSDKPath(t *testing.T) {
	root := t.TempDir()
	md := filepath.Join(root, "SKILL.md")
	content := "import `huaweicloud-sdk-go-v3/services/ecs/v2` in code\n"
	if err := os.WriteFile(md, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	findings := checkMarkdownFile(root, md)
	for _, f := range findings {
		if strings.Contains(f, "missing backtick path target") {
			t.Errorf("SDK path flagged as missing backtick path target: %s", f)
		}
	}
}
