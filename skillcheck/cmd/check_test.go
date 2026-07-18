package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

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
