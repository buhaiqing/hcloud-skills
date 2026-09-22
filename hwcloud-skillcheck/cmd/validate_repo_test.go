package cmd

import (
	"io"
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

// --- validate frontmatter: delegates_to ---

// skillMDFixture renders a minimally valid SKILL.md whose frontmatter declares
// the given delegates_to targets (the key is omitted when there are none).
func skillMDFixture(name string, delegates ...string) string {
	md := "---\nname: " + name + "\n"
	if len(delegates) > 0 {
		md += "delegates_to:\n"
		for _, d := range delegates {
			md += "  - " + d + "\n"
		}
	}
	return md + "description: x\ncompatibility: x\nlicense: Apache-2.0\n" +
		"metadata:\n  version: 1.0.0\n  last_updated: 2026-06-01\n  cli_applicability: cli-first\n---\n# body\n"
}

// writeSkillFixture materialises <root>/<name>/SKILL.md for the CLI-level tests.
func writeSkillFixture(t *testing.T, root, name string, delegates ...string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillMDFixture(name, delegates...)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// captureStderr runs fn with os.Stderr redirected into a pipe and returns the
// captured text, so tests can assert on per-violation gate output (the gate
// returns only an aggregate count).
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestValidateDelegatesTo(t *testing.T) {
	skillDirs := map[string]bool{
		"huaweicloud-ecs-ops": true,
		"huaweicloud-vpc-ops": true,
		"huaweicloud-ces-ops": true,
	}
	tests := []struct {
		name     string
		skillDir string
		content  string
		want     []string
	}{
		{
			name:     "no delegates_to key",
			skillDir: "huaweicloud-ecs-ops",
			content:  skillMDFixture("huaweicloud-ecs-ops"),
		},
		{
			name:     "existing targets only",
			skillDir: "huaweicloud-ecs-ops",
			content:  skillMDFixture("huaweicloud-ecs-ops", "huaweicloud-vpc-ops", "huaweicloud-ces-ops"),
		},
		{
			name:     "empty list",
			skillDir: "huaweicloud-ecs-ops",
			content:  "---\nname: huaweicloud-ecs-ops\ndelegates_to:\ndescription: x\n---\n",
		},
		{
			name:     "missing skill reported",
			skillDir: "huaweicloud-ecs-ops",
			content:  skillMDFixture("huaweicloud-ecs-ops", "huaweicloud-evs-ops"),
			want:     []string{"huaweicloud-ecs-ops: delegates_to references missing skill huaweicloud-evs-ops"},
		},
		{
			name:     "only the missing target is reported",
			skillDir: "huaweicloud-ecs-ops",
			content:  skillMDFixture("huaweicloud-ecs-ops", "huaweicloud-vpc-ops", "huaweicloud-evs-ops", "huaweicloud-ces-ops"),
			want:     []string{"huaweicloud-ecs-ops: delegates_to references missing skill huaweicloud-evs-ops"},
		},
		{
			name:     "several missing targets each reported",
			skillDir: "huaweicloud-waf-ops",
			content:  skillMDFixture("huaweicloud-waf-ops", "huaweicloud-scm-ops", "huaweicloud-antiddos-ops", "huaweicloud-ecs-ops"),
			want: []string{
				"huaweicloud-waf-ops: delegates_to references missing skill huaweicloud-scm-ops",
				"huaweicloud-waf-ops: delegates_to references missing skill huaweicloud-antiddos-ops",
			},
		},
		{
			name:     "unparseable frontmatter is not double-reported",
			skillDir: "huaweicloud-ecs-ops",
			content:  "# no frontmatter here\n",
		},
		{
			name:     "scalar target accepted",
			skillDir: "huaweicloud-ecs-ops",
			content:  "---\nname: huaweicloud-ecs-ops\ndelegates_to: huaweicloud-vpc-ops\ndescription: x\n---\n",
		},
		{
			name:     "non-string entry rejected",
			skillDir: "huaweicloud-ecs-ops",
			content:  "---\nname: huaweicloud-ecs-ops\ndelegates_to:\n  - 42\ndescription: x\n---\n",
			want:     []string{"huaweicloud-ecs-ops: delegates_to entry 1 is not a skill name"},
		},
		{
			name:     "mapping rejected",
			skillDir: "huaweicloud-ecs-ops",
			content:  "---\nname: huaweicloud-ecs-ops\ndelegates_to:\n  target: huaweicloud-vpc-ops\ndescription: x\n---\n",
			want:     []string{"huaweicloud-ecs-ops: delegates_to must be a list of skill names"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateDelegatesTo([]byte(tt.content), tt.skillDir, skillDirs)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("error %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestRunValidateFrontmatterDelegatesCLI exercises the gate end to end on a
// tempdir repo: a valid delegate keeps the run green, a ghost entry fails it
// and names only the ghost.
func TestRunValidateFrontmatterDelegatesCLI(t *testing.T) {
	tests := []struct {
		name      string
		delegates []string
		wantFail  bool
		wantMsg   string
	}{
		{
			name:      "valid delegate passes",
			delegates: []string{"huaweicloud-vpc-ops"},
		},
		{
			name:      "ghost delegate fails",
			delegates: []string{"huaweicloud-vpc-ops", "huaweicloud-evs-ops"},
			wantFail:  true,
			wantMsg:   "huaweicloud-ecs-ops: delegates_to references missing skill huaweicloud-evs-ops",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeSkillFixture(t, root, "huaweicloud-ecs-ops", tt.delegates...)
			writeSkillFixture(t, root, "huaweicloud-vpc-ops", "huaweicloud-ecs-ops")

			var err error
			stderr := captureStderr(t, func() {
				err = runValidateFrontmatter([]string{"--root", root})
			})

			if !tt.wantFail {
				if err != nil {
					t.Fatalf("resolvable delegates should pass, got %v (stderr: %s)", err, stderr)
				}
				return
			}
			if err == nil {
				t.Fatal("dangling delegates_to entry should fail the gate")
			}
			if !containsStr(stderr, tt.wantMsg) {
				t.Errorf("stderr missing %q, got:\n%s", tt.wantMsg, stderr)
			}
			if containsStr(stderr, "missing skill huaweicloud-vpc-ops") {
				t.Errorf("existing delegate must not be reported, got:\n%s", stderr)
			}
		})
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
