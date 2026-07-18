package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/embed"
)

// captureExit runs fn and returns its error (Execute() does not os.Exit in
// tests because we call the dispatch functions directly).
func TestValidateSchemaSummaryHealthy(t *testing.T) {
	// Write the embedded healthy summary fixture to a temp file, validate it.
	tmp := t.TempDir()
	path := filepath.Join(tmp, "summary.json")
	if err := os.WriteFile(path, embed.SummaryHealthy, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runValidateSchema([]string{"summary", "--file", path}); err != nil {
		t.Fatalf("healthy summary fixture should validate, got: %v", err)
	}
}

func TestValidateSchemaTraceHealthy(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "trace.json")
	// Minimal valid trace instance matching gcl-trace.schema.json required
	// fields (trace_schema_version, skill, request, rubric_version,
	// masked_fields, iterations, final).
	instance := []byte(`{
  "trace_schema_version": "v1",
  "skill": "huaweicloud-ecs-ops",
  "request": "smoke test",
  "rubric_version": "1.0.0",
  "masked_fields": [],
  "iterations": [],
  "final": {"status": "PASS", "iter": 1, "output": null, "unresolved": [], "failure_pattern": null}
}`)
	if err := os.WriteFile(path, instance, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runValidateSchema([]string{"trace", "--file", path}); err != nil {
		t.Fatalf("valid trace instance should validate, got: %v", err)
	}
}

func TestValidateSchemaInvalid(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.json")
	// summary schema requires "version" etc.; omitting it must fail.
	if err := os.WriteFile(path, []byte(`{"generated_at":"2026-07-18T10:00:00Z","cloud":"huaweicloud"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runValidateSchema([]string{"summary", "--file", path}); err == nil {
		t.Fatal("instance missing required fields should fail validation")
	}
}

func TestValidateSchemaStdin(t *testing.T) {
	old := osStdin
	osStdin = bytes.NewReader(embed.SummaryHealthy)
	defer func() { osStdin = old }()
	if err := runValidateSchema([]string{"summary", "--file", "-"}); err != nil {
		t.Fatalf("stdin healthy fixture should validate, got: %v", err)
	}
}

func TestValidateSchemaUnknownKind(t *testing.T) {
	if err := runValidateSchema([]string{"bogus", "--file", "-"}); err == nil {
		t.Fatal("unknown schema kind should error")
	}
}

func TestExecuteUnknownSubcommand(t *testing.T) {
	// Temporarily swap os.Args to simulate CLI invocation.
	old := os.Args
	os.Args = []string{"skillcheck", "frobnicate"}
	defer func() { os.Args = old }()
	if err := Execute(); err == nil {
		t.Fatal("unknown subcommand should error")
	}
}

// TestValidateSchemaAlarmPlanHealthy validates the embedded healthy
// alarm-plan fixture against the alarm-plan schema (T5 coverage for the
// 4th validate-schema kind).
func TestValidateSchemaAlarmPlanHealthy(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "alarm-plan.json")
	if err := os.WriteFile(path, embed.AlarmPlanHealthy, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runValidateSchema([]string{"alarm-plan", "--file", path}); err != nil {
		t.Fatalf("healthy alarm-plan fixture should validate, got: %v", err)
	}
}

// TestValidateSchemaEvalQueries validates a constructed eval_queries instance
// (matchArrayEntry array shape) against the eval-queries union schema (T5
// coverage for the 4th kind). The schema is a $defs-only union contract, so we
// validate the specific $def via format auto-detection.
func TestValidateSchemaEvalQueries(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "eval-queries.json")
	// matchArrayEntry requires query/should_match/skill; the whole file is an
	// array of entries (mirrors scripts/validate_eval_queries_schema.py).
	instance := []byte(`[
	  {"query":"list ecs instances","should_match":true,"skill":"huaweicloud-ecs-ops","reason":"smoke"},
	  {"query":"delete everything","should_match":false,"skill":"huaweicloud-ecs-ops","reason":"negative"}
	]`)
	if err := os.WriteFile(path, instance, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runValidateSchema([]string{"eval-queries", "--file", path}); err != nil {
		t.Fatalf("valid eval-queries instance should validate, got: %v", err)
	}
}

// TestValidateSchemaEvalQueriesInvalid confirms an invalid eval_queries
// instance (missing required fields, empty query) is rejected.
func TestValidateSchemaEvalQueriesInvalid(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "eval-queries-bad.json")
	instance := []byte(`[{"query":"","should_match":true}]`)
	if err := os.WriteFile(path, instance, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runValidateSchema([]string{"eval-queries", "--file", path}); err == nil {
		t.Fatal("invalid eval-queries instance should fail validation")
	}
}

// scaffoldSkillTree builds a minimal huaweicloud-<name>-ops tree for
// advanced-coverage and validate total-entry tests.
func scaffoldSkillTree(t *testing.T, root, name string, advanced map[string]string, refs map[string]string) {
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

// TestCheckAdvancedCoverageOK verifies a skill with references/advanced/ and a
// Security-Sensitive marker passes without error.
func TestCheckAdvancedCoverageOK(t *testing.T) {
	root := t.TempDir()
	scaffoldSkillTree(t, root, "huaweicloud-ecs-ops",
		map[string]string{"aiops-patterns.md": "deep content"},
		map[string]string{"runbook.md": "Security-Sensitive: delete volumes"})
	if err := runCheckAdvancedCoverage([]string{"--root", root}); err != nil {
		t.Fatalf("skill with advanced/ + marker should pass, got: %v", err)
	}
}

// TestCheckAdvancedCoverageMissingErrors verifies a skill lacking
// references/advanced/ fails (non-warn-only).
func TestCheckAdvancedCoverageMissingErrors(t *testing.T) {
	root := t.TempDir()
	scaffoldSkillTree(t, root, "huaweicloud-ecs-ops", nil, nil)
	if err := runCheckAdvancedCoverage([]string{"--root", root}); err == nil {
		t.Fatal("skill missing advanced/ should fail (non-warn-only)")
	}
}

// TestCheckAdvancedCoverageWarnOnly verifies --warn-only demotes the missing
// advanced/ finding to a warning and returns nil.
func TestCheckAdvancedCoverageWarnOnly(t *testing.T) {
	root := t.TempDir()
	scaffoldSkillTree(t, root, "huaweicloud-ecs-ops", nil, nil)
	if err := runCheckAdvancedCoverage([]string{"--root", root, "--warn-only"}); err != nil {
		t.Fatalf("warn-only should not fail, got: %v", err)
	}
}

// TestCheckAdvancedCoverageExempt verifies the generator skill is exempt from
// the advanced/ requirement.
func TestCheckAdvancedCoverageExempt(t *testing.T) {
	root := t.TempDir()
	scaffoldSkillTree(t, root, "huaweicloud-skill-generator", nil, nil)
	if err := runCheckAdvancedCoverage([]string{"--root", root}); err != nil {
		t.Fatalf("exempt skill should pass, got: %v", err)
	}
}

// TestValidateAllTotalEntryInvokesChecks verifies the `validate` total-entry
// (no subcommand) runs the A-class suite and reports failures when a skill
// violates advanced-coverage.
func TestValidateAllTotalEntryInvokesChecks(t *testing.T) {
	root := t.TempDir()
	scaffoldSkillTree(t, root, "huaweicloud-ecs-ops", nil, nil)
	// No advanced/ dir => advanced-coverage step fails => total-entry fails.
	if err := runValidateAll([]string{"--root", root}); err == nil {
		t.Fatal("total-entry should fail when a step fails")
	}
}

// TestValidateAllTotalEntryClean verifies the total-entry passes when every
// skill satisfies all 7 A-class checks. A fully-compliant skill fixture is
// scaffolded so each step (frontmatter, eval-queries, product-assessment,
// example-config, markdown-links, references-links, advanced-coverage) passes.
func TestValidateAllTotalEntryClean(t *testing.T) {
	root := t.TempDir()
	skill := "huaweicloud-ecs-ops"
	skillDir := filepath.Join(root, skill)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// SKILL.md with valid frontmatter (name matches dir, required keys,
	// closing --- fence so yaml.ExtractFrontmatter parses it).
	skillMD := "---\nname: huaweicloud-ecs-ops\ndescription: ecs ops\nlicense: MIT\ncompatibility: KooCLI\nmetadata:\n  version: \"1.0.0\"\n  last_updated: \"2026-06-04\"\n  cli_applicability: \"dual-path\"\n---\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		t.Fatal(err)
	}
	// assets/eval_queries.json (valid union instance).
	evalQ := `[{"query":"list ecs","should_match":true,"skill":"huaweicloud-ecs-ops","reason":"smoke"}]`
	if err := os.MkdirAll(filepath.Join(skillDir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "assets", "eval_queries.json"), []byte(evalQ), 0o644); err != nil {
		t.Fatal(err)
	}
	// assets/example-config.yaml using valid anchors + env placeholders.
	exampleCfg := "defaults: &defaults\n  region: \"{{env.HW_REGION_ID}}\"\ninstance:\n  <<: *defaults\n  name: demo\n"
	if err := os.WriteFile(filepath.Join(skillDir, "assets", "example-config.yaml"), []byte(exampleCfg), 0o644); err != nil {
		t.Fatal(err)
	}
	// references/well-architected-assessment.md with an IAM table and the
	// required "Worker Output Contract" section carrying a valid JSON contract.
	refDir := filepath.Join(skillDir, "references")
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wa := "# Well-Architected Assessment\n\n" +
		"## 1. Security\n\n" +
		"| Operation | IAM Action | Resource Scope |\n|-----------|-----------|---------------|\n| List | ecs:*List* | * |\n\n" +
		"## Worker Output Contract\n\n" +
		"```json\n" +
		"{\"skill_id\":\"huaweicloud-ecs-ops\",\"product\":\"ECS\",\"region\":\"cn-north-4\",\"scope\":\"read-only\"," +
		"\"assessment_date\":\"2026-06-04\",\"status\":\"OK\",\"partial\":false,\"resource_count\":1," +
		"\"pillars\":{\"security\":{\"status\":\"assessed\"}},\"recommendations\":[],\"trace\":{\"commands\":[]},\"errors\":[]}\n" +
		"```\n"
	if err := os.WriteFile(filepath.Join(skillDir, "references", "well-architected-assessment.md"), []byte(wa), 0o644); err != nil {
		t.Fatal(err)
	}
	// references/advanced/aiops-patterns.md (satisfies TE-7 + Security-Sensitive marker).
	advDir := filepath.Join(skillDir, "references", "advanced")
	if err := os.MkdirAll(advDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(advDir, "aiops-patterns.md"), []byte("Security-Sensitive: delete volumes"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A referenceable markdown file so markdown-links/references-links pass.
	if err := os.WriteFile(filepath.Join(skillDir, "references", "guide.md"), []byte("# Guide\nSee [runbook](../references/well-architected-assessment.md).\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := runValidateAll([]string{"--root", root}); err != nil {
		t.Fatalf("total-entry should pass for a clean skill, got: %v", err)
	}
}
