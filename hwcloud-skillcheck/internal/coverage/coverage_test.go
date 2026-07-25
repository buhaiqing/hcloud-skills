package coverage

import (
	"os"
	"path/filepath"
	"testing"
)

// writeSkill scaffolds a minimal huaweicloud-<name>-ops tree for tests.
func writeSkill(t *testing.T, root, name string, advanced map[string]string, refs map[string]string) {
	t.Helper()
	refDir := filepath.Join(root, name, "references")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if len(advanced) > 0 {
		advDir := filepath.Join(refDir, "advanced")
		if err := os.MkdirAll(advDir, 0o755); err != nil {
			t.Fatal(err)
		}
		for f, body := range advanced {
			if err := os.WriteFile(filepath.Join(advDir, f), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	for f, body := range refs {
		if err := os.WriteFile(filepath.Join(refDir, f), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDiscoverSkills(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "huaweicloud-ecs-ops", nil, nil)
	writeSkill(t, root, "huaweicloud-rds-ops", nil, nil)
	if err := os.MkdirAll(filepath.Join(root, "not-a-skill"), 0o755); err != nil {
		t.Fatal(err)
	}
	skills, err := DiscoverSkills(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 2 || skills[0] != "huaweicloud-ecs-ops" || skills[1] != "huaweicloud-rds-ops" {
		t.Fatalf("unexpected skills: %v", skills)
	}
}

func TestValidateSkillMissingAdvanced(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "huaweicloud-ecs-ops", nil, map[string]string{
		"aiops-patterns.md": "some text",
	})
	rep := ValidateSkill(root, "huaweicloud-ecs-ops", false)
	if rep.OK {
		t.Fatalf("expected error for missing advanced/ dir, got OK")
	}
	if len(rep.Errors) == 0 {
		t.Fatalf("expected missing-advanced error")
	}
	if rep.Warnings == nil {
		t.Fatalf("expected security-marker warning (no markers present)")
	}
}

func TestValidateSkillWarnOnly(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "huaweicloud-ecs-ops", nil, nil)
	rep := ValidateSkill(root, "huaweicloud-ecs-ops", true)
	if !rep.OK {
		t.Fatalf("warn-only should demote missing-advanced to warning; got errors=%v", rep.Errors)
	}
	if len(rep.Warnings) == 0 {
		t.Fatalf("expected warning under warn-only")
	}
}

func TestValidateSkillExempt(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "huaweicloud-skill-generator", nil, nil)
	rep := ValidateSkill(root, "huaweicloud-skill-generator", false)
	if !rep.OK {
		t.Fatalf("exempt skill should not require advanced/; got errors=%v", rep.Errors)
	}
}

func TestValidateSkillSecurityMarkers(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "huaweicloud-ecs-ops",
		map[string]string{"aiops-patterns.md": "deep content"},
		map[string]string{"runbook.md": "Security-Sensitive: delete volumes"},
	)
	rep := ValidateSkill(root, "huaweicloud-ecs-ops", false)
	if !rep.OK {
		t.Fatalf("expected OK with advanced/ present; errors=%v", rep.Errors)
	}
	if rep.SecurityMarkers == 0 {
		t.Fatalf("expected at least one security marker counted")
	}
	if len(rep.AdvancedFiles) != 1 {
		t.Fatalf("expected 1 advanced file, got %v", rep.AdvancedFiles)
	}
}

func TestValidateAll(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "huaweicloud-ecs-ops",
		map[string]string{"aiops-patterns.md": "x"}, nil)
	writeSkill(t, root, "huaweicloud-rds-ops", nil, nil)
	rep, err := ValidateAll(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.SkillsChecked != 2 {
		t.Fatalf("expected 2 skills checked, got %d", rep.SkillsChecked)
	}
	if rep.SkillsWithAdvanced != 1 {
		t.Fatalf("expected 1 skill with advanced/, got %d", rep.SkillsWithAdvanced)
	}
	if rep.OK {
		t.Fatalf("expected overall not-OK because rds-ops lacks advanced/")
	}
}
