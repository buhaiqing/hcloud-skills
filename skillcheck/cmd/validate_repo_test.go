package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// --- validate frontmatter ---

const validSkillMD = `---
name: huaweicloud-ecs-ops
description: ECS ops runbook
compatibility: huaweicloud-sdk-go-v3 >= 3.0
license: Apache-2.0
metadata:
  version: 1.0.0
  last_updated: 2026-06-01
  cli_applicability: cli-first
---
# ECS operations
`

func TestValidateFrontmatterGood(t *testing.T) {
	errs := validateSkillFrontmatter([]byte(validSkillMD), "huaweicloud-ecs-ops")
	if len(errs) != 0 {
		t.Fatalf("expected no errors for valid frontmatter, got: %v", errs)
	}
}

func TestValidateFrontmatterBadName(t *testing.T) {
	md := `---
name: not-a-skill
description: x
compatibility: x
license: Apache-2.0
metadata:
  version: 1.0.0
  last_updated: 2026-06-01
  cli_applicability: cli-first
---
`
	errs := validateSkillFrontmatter([]byte(md), "huaweicloud-ecs-ops")
	if len(errs) == 0 {
		t.Fatal("expected error for name not starting with huaweicloud-")
	}
}

func TestValidateFrontmatterNameDirMismatch(t *testing.T) {
	md := `---
name: huaweicloud-rds-ops
description: x
compatibility: x
license: Apache-2.0
metadata:
  version: 1.0.0
  last_updated: 2026-06-01
  cli_applicability: cli-first
---
`
	errs := validateSkillFrontmatter([]byte(md), "huaweicloud-ecs-ops")
	found := false
	for _, e := range errs {
		if containsStr(e, "does not match directory") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected name/directory mismatch error, got: %v", errs)
	}
}

func TestValidateFrontmatterMissingFields(t *testing.T) {
	md := `---
name: huaweicloud-ecs-ops
---
`
	errs := validateSkillFrontmatter([]byte(md), "huaweicloud-ecs-ops")
	// At least missing description, compatibility, license, metadata.
	if len(errs) < 4 {
		t.Fatalf("expected >=4 missing-field errors, got: %v", errs)
	}
}

func TestValidateFrontmatterBadCLI(t *testing.T) {
	md := `---
name: huaweicloud-ecs-ops
description: x
compatibility: x
license: Apache-2.0
metadata:
  version: 1.0.0
  last_updated: 2026-06-01
  cli_applicability: bogus-value
---
`
	errs := validateSkillFrontmatter([]byte(md), "huaweicloud-ecs-ops")
	found := false
	for _, e := range errs {
		if containsStr(e, "invalid cli_applicability") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected invalid cli_applicability error, got: %v", errs)
	}
}

func TestValidateFrontmatterMissingCLI(t *testing.T) {
	// Skill not in OPTIONAL_NO_CLI and no cli_applicability => error.
	md := `---
name: huaweicloud-ecs-ops
description: x
compatibility: x
license: Apache-2.0
metadata:
  version: 1.0.0
  last_updated: 2026-06-01
---
`
	errs := validateSkillFrontmatter([]byte(md), "huaweicloud-ecs-ops")
	found := false
	for _, e := range errs {
		if containsStr(e, "missing metadata.cli_applicability") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected missing cli_applicability error, got: %v", errs)
	}
}

func TestValidateFrontmatterOptionalNoCLI(t *testing.T) {
	// huaweicloud-billing-ops is in OPTIONAL_NO_CLI, so missing cli is OK.
	md := `---
name: huaweicloud-billing-ops
description: x
compatibility: x
license: Apache-2.0
metadata:
  version: 1.0.0
  last_updated: 2026-06-01
---
`
	errs := validateSkillFrontmatter([]byte(md), "huaweicloud-billing-ops")
	if len(errs) != 0 {
		t.Fatalf("billing-ops may omit cli_applicability, got: %v", errs)
	}
}

func TestRunValidateFrontmatterCLI(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-ecs-ops")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(validSkillMD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runValidateFrontmatter([]string{"--root", root}); err != nil {
		t.Fatalf("valid skill tree should pass, got: %v", err)
	}
}

func TestRunValidateFrontmatterCLIFail(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-ecs-ops")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bad := `---
name: wrong-name
description: x
compatibility: x
license: Apache-2.0
metadata:
  version: 1.0.0
  last_updated: 2026-06-01
  cli_applicability: cli-first
---
`
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runValidateFrontmatter([]string{"--root", root}); err == nil {
		t.Fatal("invalid skill frontmatter should fail")
	}
}

// --- validate eval-queries ---

func TestRunValidateEvalQueriesCLI(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "huaweicloud-ecs-ops", "assets")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	good := `[
  {"query":"list ecs","should_match":true,"skill":"huaweicloud-ecs-ops","reason":"smoke"},
  {"query":"delete vpc","should_match":false,"skill":"huaweicloud-ecs-ops","reason":"negative"}
]`
	if err := os.WriteFile(filepath.Join(assets, "eval_queries.json"), []byte(good), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runValidateEvalQueries([]string{"--root", root}); err != nil {
		t.Fatalf("valid eval_queries should pass, got: %v", err)
	}
}

func TestRunValidateEvalQueriesCLIFail(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "huaweicloud-ecs-ops", "assets")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	// skill mismatch with directory name => error
	bad := `[{"query":"list ecs","should_match":true,"skill":"huaweicloud-rds-ops"}]`
	if err := os.WriteFile(filepath.Join(assets, "eval_queries.json"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runValidateEvalQueries([]string{"--root", root}); err == nil {
		t.Fatal("skill mismatch should fail")
	}
}

// --- validate product-assessment ---

const validAssessmentMD = `# Well-Architected

## Worker Output Contract (Read-Only Assessment Mode)

` + "```json" + `
{
  "skill_id": "huaweicloud-ecs-ops",
  "product": "ecs",
  "region": "cn-north-4",
  "scope": "account-wide",
  "assessment_date": "2026-06-19T10:00:00+08:00",
  "status": "OK",
  "partial": false,
  "resource_count": 1,
  "pillars": {
    "reliability": {"score": 80, "status": "assessed", "findings": []},
    "security": {"score": 80, "status": "assessed", "findings": []},
    "cost": {"score": 80, "status": "assessed", "findings": []},
    "efficiency": {"score": 80, "status": "assessed", "findings": []}
  },
  "recommendations": [],
  "trace": {"commands": [], "request_ids": []},
  "errors": []
}
` + "```" + `
`

func TestValidateProductAssessmentGood(t *testing.T) {
	errs := validateProductAssessment([]byte(validAssessmentMD), "huaweicloud-ecs-ops")
	if len(errs) != 0 {
		t.Fatalf("valid assessment should pass, got: %v", errs)
	}
}

func TestValidateProductAssessmentBadStatus(t *testing.T) {
	md := replaceStatus(validAssessmentMD, "OK", "BOGUS")
	errs := validateProductAssessment([]byte(md), "huaweicloud-ecs-ops")
	found := false
	for _, e := range errs {
		if containsStr(e, "invalid status") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected invalid status error, got: %v", errs)
	}
}

func TestValidateProductAssessmentMissingSection(t *testing.T) {
	errs := validateProductAssessment([]byte("# No contract here\n"), "huaweicloud-ecs-ops")
	found := false
	for _, e := range errs {
		if containsStr(e, "Worker Output Contract") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected missing-section error, got: %v", errs)
	}
}

func TestValidateProductAssessmentSkillMismatch(t *testing.T) {
	errs := validateProductAssessment([]byte(validAssessmentMD), "huaweicloud-rds-ops")
	found := false
	for _, e := range errs {
		if containsStr(e, "skill_id") && containsStr(e, "does not match") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected skill_id mismatch error, got: %v", errs)
	}
}

func TestRunValidateProductAssessmentCLI(t *testing.T) {
	root := t.TempDir()
	refs := filepath.Join(root, "huaweicloud-ecs-ops", "references")
	if err := os.MkdirAll(refs, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(refs, "well-architected-assessment.md")
	if err := os.WriteFile(path, []byte(validAssessmentMD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runValidateProductAssessment([]string{"--root", root}); err != nil {
		t.Fatalf("valid assessment should pass, got: %v", err)
	}
}

// --- helpers ---

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func replaceStatus(in, old, new string) string {
	for i := 0; i+len(old) <= len(in); i++ {
		if in[i:i+len(old)] == old {
			return in[:i] + new + in[i+len(old):]
		}
	}
	return in
}
