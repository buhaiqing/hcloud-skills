package manifest_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/manifest"
)

func TestGenerate_ParsesFrontmatter(t *testing.T) {
	// Plant a temp skill dir with a SKILL.md containing frontmatter.
	skillDir := t.TempDir()
	skillName := "huaweicloud-test-ops"

	subDir := filepath.Join(skillDir, skillName)
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	skillMD := `---
name: Test Skill
description: A skill for testing manifest generation
version: "1.0"
---

# Test Skill
`
	if err := os.WriteFile(filepath.Join(subDir, "SKILL.md"), []byte(skillMD), 0644); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	if err := manifest.Generate(skillDir, outDir); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	manifestPath := filepath.Join(outDir, skillName, "capability_manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read output manifest: %v", err)
	}

	var m manifest.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if m.Name == "" {
		t.Error("Name field is empty, expected 'Test Skill' from frontmatter")
	}
	if m.SideEffectClass == "" {
		t.Error("SideEffectClass is empty, expected default 'destructive'")
	}
}
