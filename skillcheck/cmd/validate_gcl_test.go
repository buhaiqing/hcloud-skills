package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper: scaffold a minimal GCL-conformant skill directory.
// ---------------------------------------------------------------------------

func scaffoldGCLSkill(t *testing.T, root, skill, rubricBody, promptBody, skillMD string) {
	t.Helper()
	refDir := filepath.Join(root, skill, "references")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if rubricBody != "" {
		if err := os.WriteFile(filepath.Join(refDir, "rubric.md"), []byte(rubricBody), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if promptBody != "" {
		if err := os.WriteFile(filepath.Join(refDir, "prompt-templates.md"), []byte(promptBody), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if skillMD != "" {
		if err := os.WriteFile(filepath.Join(root, skill, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func gclRubricOK() string {
	return `## 1. Correctness
## 2. Safety
## 3. Completeness
## 4. Consistency
## 5. Clarity
## 6. Efficiency
## 7. Testability
## 8. Maintainability
`
}

func gclPromptOK() string {
	return `## 1. System prompt
## 2. Task description
## 3. Input specification
## 4. Output specification
## 5. Constraints
## 6. Examples
## 7. Quality Gate (GCL)
{{output.operation_intent}}
`
}

func gclSkillMDOK() string {
	return `# SKILL.md for huaweicloud-ecs-ops

## Quality Gate (GCL)
Some quality gate text here.
`
}

// ---------------------------------------------------------------------------
// Tests: runValidateGCLConformance
// ---------------------------------------------------------------------------

func TestGCLConformance_AllConformant(t *testing.T) {
	tmp := t.TempDir()
	scaffoldGCLSkill(t, tmp, "huaweicloud-ecs-ops", gclRubricOK(), gclPromptOK(), gclSkillMDOK())
	err := runValidateGCLConformance([]string{"--root", tmp})
	if err != nil {
		t.Fatalf("expected conformant skill to pass, got: %v", err)
	}
}

func TestGCLConformance_RubricSectionsWrong(t *testing.T) {
	tmp := t.TempDir()
	// Only 3 rubric sections instead of 8
	scaffoldGCLSkill(t, tmp, "huaweicloud-ecs-ops",
		"## 1. Correctness\n## 2. Safety\n## 3. Completeness\n",
		gclPromptOK(), gclSkillMDOK())
	err := runValidateGCLConformance([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure due to wrong rubric sections, got nil")
	}
}

func TestGCLConformance_MissingQualityGate(t *testing.T) {
	tmp := t.TempDir()
	// SKILL.md without ## Quality Gate (GCL) heading
	scaffoldGCLSkill(t, tmp, "huaweicloud-ecs-ops", gclRubricOK(), gclPromptOK(),
		"# SKILL.md\nSome content without quality gate heading.\n")
	err := runValidateGCLConformance([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure due to missing quality gate, got nil")
	}
}

func TestGCLConformance_BarePlaceholder(t *testing.T) {
	tmp := t.TempDir()
	// prompt-templates.md has bare {instance_id} placeholder
	scaffoldGCLSkill(t, tmp, "huaweicloud-ecs-ops", gclRubricOK(),
		"## 1. System prompt\n## 2. Task\n## 3. Input\n## 4. Output\n## 5. Constraints\n## 6. Examples\n## 7. Quality Gate (GCL)\n{{output.operation_intent}}\n\nList {instance_id} here.\n",
		gclSkillMDOK())
	err := runValidateGCLConformance([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure due to bare placeholder, got nil")
	}
}

func TestGCLConformance_EscapedPlaceholderOK(t *testing.T) {
	tmp := t.TempDir()
	// {{instance_id}} is escaped (doubled braces) — should NOT count as bare
	prompt := gclPromptOK() + "\nUse {{instance_id}} for the identifier.\n"
	scaffoldGCLSkill(t, tmp, "huaweicloud-ecs-ops", gclRubricOK(), prompt, gclSkillMDOK())
	err := runValidateGCLConformance([]string{"--root", tmp})
	if err != nil {
		t.Fatalf("escaped {{placeholder}} should not trigger bare-placeholder failure: %v", err)
	}
}

func TestGCLConformance_BareInCodeBlockIgnored(t *testing.T) {
	tmp := t.TempDir()
	// Bare {instance_id} is inside a fenced code block — should be ignored
	prompt := gclPromptOK() + "\n```json\n{ \"id\": {instance_id} }\n```\n"
	scaffoldGCLSkill(t, tmp, "huaweicloud-ecs-ops", gclRubricOK(), prompt, gclSkillMDOK())
	err := runValidateGCLConformance([]string{"--root", tmp})
	if err != nil {
		t.Fatalf("bare placeholder inside code block should be ignored: %v", err)
	}
}

func TestGCLConformance_JSONOutput(t *testing.T) {
	tmp := t.TempDir()
	scaffoldGCLSkill(t, tmp, "huaweicloud-ecs-ops", gclRubricOK(), gclPromptOK(), gclSkillMDOK())
	err := runValidateGCLConformance([]string{"--root", tmp, "--json"})
	if err != nil {
		t.Fatalf("expected JSON output to pass for conformant skill, got: %v", err)
	}
}

func TestGCLConformance_NoSkills(t *testing.T) {
	tmp := t.TempDir()
	// No skill directories present
	err := runValidateGCLConformance([]string{"--root", tmp})
	if err != nil {
		t.Fatalf("expected pass with no skills, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: runValidateAlarmWireContract
// ---------------------------------------------------------------------------

func scaffoldCESConfig(t *testing.T, tmp, configYAML string) {
	t.Helper()
	cesDir := filepath.Join(tmp, cesSkill, "assets")
	if err := os.MkdirAll(cesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cesDir, "example-config.yaml"), []byte(configYAML), 0o644); err != nil {
		t.Fatal(err)
	}
}

func scaffoldGCLErrorSpec(t *testing.T, tmp, specMD string) {
	t.Helper()
	docsDir := filepath.Join(tmp, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "gcl-spec.md"), []byte(specMD), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAlarmWire_OK(t *testing.T) {
	tmp := t.TempDir()
	cfg := `gcl_quality:
  pass_rate_warn: 0.85
  pass_rate_critical: 0.70
  max_iter_warn_count: 3
  safety_fail_alert: true
`
	scaffoldCESConfig(t, tmp, cfg)
	spec := `## GCL Thresholds
- pass_rate_warn: 0.85
- pass_rate_critical: 0.70
- max_iter_warn_count: 3
- safety_fail_alert: true
`
	scaffoldGCLErrorSpec(t, tmp, spec)
	err := runValidateAlarmWireContract([]string{"--root", tmp})
	if err != nil {
		t.Fatalf("expected clean config to pass, got: %v", err)
	}
}

func TestAlarmWire_MissingConfig(t *testing.T) {
	tmp := t.TempDir()
	err := runValidateAlarmWireContract([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure when CES config missing, got nil")
	}
}

func TestAlarmWire_MissingBlock(t *testing.T) {
	tmp := t.TempDir()
	scaffoldCESConfig(t, tmp, `other_key: value\n`)
	err := runValidateAlarmWireContract([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure when gcl_quality block missing, got nil")
	}
}

func TestAlarmWire_PassRateDrift(t *testing.T) {
	tmp := t.TempDir()
	// pass_rate_warn=0.80 drifts from expected 0.85
	cfg := `gcl_quality:
  pass_rate_warn: 0.80
  pass_rate_critical: 0.70
  max_iter_warn_count: 3
  safety_fail_alert: true
`
	scaffoldCESConfig(t, tmp, cfg)
	spec := `pass_rate_warn: 0.80
pass_rate_critical: 0.70
max_iter_warn_count: 3
safety_fail_alert: true
`
	scaffoldGCLErrorSpec(t, tmp, spec)
	err := runValidateAlarmWireContract([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure due to pass_rate drift, got nil")
	}
}

func TestAlarmWire_InvalidOrdering(t *testing.T) {
	tmp := t.TempDir()
	// warn < critical violates 0 <= critical <= warn constraint
	cfg := `gcl_quality:
  pass_rate_warn: 0.60
  pass_rate_critical: 0.80
  max_iter_warn_count: 3
  safety_fail_alert: true
`
	scaffoldCESConfig(t, tmp, cfg)
	spec := `pass_rate_warn: 0.60
pass_rate_critical: 0.80
max_iter_warn_count: 3
safety_fail_alert: true
`
	scaffoldGCLErrorSpec(t, tmp, spec)
	err := runValidateAlarmWireContract([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure due to invalid pass_rate ordering, got nil")
	}
}

func TestAlarmWire_MissingSafetyFailAlert(t *testing.T) {
	tmp := t.TempDir()
	cfg := `gcl_quality:
  pass_rate_warn: 0.85
  pass_rate_critical: 0.70
  max_iter_warn_count: 3
`
	scaffoldCESConfig(t, tmp, cfg)
	spec := `pass_rate_warn: 0.85
pass_rate_critical: 0.70
max_iter_warn_count: 3
`
	scaffoldGCLErrorSpec(t, tmp, spec)
	err := runValidateAlarmWireContract([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure due to missing safety_fail_alert, got nil")
	}
}

func TestAlarmWire_MissingSpecFragment(t *testing.T) {
	tmp := t.TempDir()
	cfg := `gcl_quality:
  pass_rate_warn: 0.85
  pass_rate_critical: 0.70
  max_iter_warn_count: 3
  safety_fail_alert: true
`
	scaffoldCESConfig(t, tmp, cfg)
	// docs/gcl-spec.md missing pass_rate_critical fragment
	spec := `pass_rate_warn: 0.85
max_iter_warn_count: 3
safety_fail_alert: true
`
	scaffoldGCLErrorSpec(t, tmp, spec)
	err := runValidateAlarmWireContract([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure due to missing spec fragment, got nil")
	}
}

func TestAlarmWire_JSONOutput(t *testing.T) {
	tmp := t.TempDir()
	cfg := `gcl_quality:
  pass_rate_warn: 0.85
  pass_rate_critical: 0.70
  max_iter_warn_count: 3
  safety_fail_alert: true
`
	scaffoldCESConfig(t, tmp, cfg)
	spec := `pass_rate_warn: 0.85
pass_rate_critical: 0.70
max_iter_warn_count: 3
safety_fail_alert: true
`
	scaffoldGCLErrorSpec(t, tmp, spec)
	err := runValidateAlarmWireContract([]string{"--root", tmp, "--json"})
	if err != nil {
		t.Fatalf("expected JSON output to pass for clean config, got: %v", err)
	}
}
