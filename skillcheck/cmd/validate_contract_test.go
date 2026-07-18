package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper: scaffold files for generator-contract
// ---------------------------------------------------------------------------

func scaffoldGeneratorContractFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for relPath, content := range files {
		fullPath := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: generator-contract
// ---------------------------------------------------------------------------

func TestGeneratorContract_AllPatternsOK(t *testing.T) {
	tmp := t.TempDir()
	scaffoldGeneratorContractFiles(t, tmp, map[string]string{
		"huaweicloud-skill-generator/references/huaweicloud-skill-template.md": "metadata:\n  name: generator\n  gcl:\n    required: true\n    default_max_iter: 2\n    rubric_version: \"v1\"\n    trace_path: audit-results/gcl-trace-YYYYMMDD-HHMMSS.json\n## Quality Gate (GCL)\nreferences/rubric.md\nreferences/prompt-templates.md\noperation_intent\ngcl-prompt-backbone.md\n",
		"huaweicloud-skill-generator/SKILL.md": "# Generator Skill\n`references/rubric.md`\n`references/prompt-templates.md`\n`metadata.gcl`\nreferences/gcl-prompt-backbone.md\n",
		"huaweicloud-skill-generator/references/gcl-prompt-backbone.md": "## 1. Generator prompt template\n## 2. Critic prompt template\n## 3. Orchestrator prompt template\nPRIMARY: hcloud\nhuaweicloud-sdk-go-v3\n{{output.operation_intent}}\nDo NOT consider the original user request\nread-only\naudit-results/gcl-trace-YYYYMMDD-HHMMSS.json\n",
	})
	err := runValidateGeneratorContract([]string{"--root", tmp})
	if err != nil {
		t.Fatalf("expected all patterns to match, got: %v", err)
	}
}

func TestGeneratorContract_MissingFile(t *testing.T) {
	tmp := t.TempDir()
	// Only template file present; generator_skill and backbone missing
	scaffoldGeneratorContractFiles(t, tmp, map[string]string{
		"huaweicloud-skill-generator/references/huaweicloud-skill-template.md": "metadata:\n  gcl:\n    default_max_iter: 2\n",
	})
	err := runValidateGeneratorContract([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure when required files missing, got nil")
	}
}

func TestGeneratorContract_PatternMissing(t *testing.T) {
	tmp := t.TempDir()
	// Files exist but rubric_version pattern missing from template
	scaffoldGeneratorContractFiles(t, tmp, map[string]string{
		"huaweicloud-skill-generator/references/huaweicloud-skill-template.md": "metadata:\n  name: generator\n  gcl:\n    default_max_iter: 2\n",
		"huaweicloud-skill-generator/SKILL.md": "`references/rubric.md`\n",
		"huaweicloud-skill-generator/references/gcl-prompt-backbone.md": "## 1. Generator\n",
	})
	err := runValidateGeneratorContract([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure due to missing pattern, got nil")
	}
}

func TestGeneratorContract_BarePlaceholder(t *testing.T) {
	tmp := t.TempDir()
	scaffoldGeneratorContractFiles(t, tmp, map[string]string{
		"huaweicloud-skill-generator/references/huaweicloud-skill-template.md": "metadata:\n  gcl:\n    default_max_iter: 2\n    rubric_version: \"v1\"\n    trace_path: audit-results/gcl-trace-YYYYMMDD-HHMMSS.json\n## Quality Gate (GCL)\n## 1. Generator\n{unsupported_placeholder}\n",
		"huaweicloud-skill-generator/SKILL.md": "`references/rubric.md`\n",
		"huaweicloud-skill-generator/references/gcl-prompt-backbone.md": "## 1. Generator prompt template\n",
	})
	err := runValidateGeneratorContract([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure due to bare placeholder in template, got nil")
	}
}

func TestGeneratorContract_JSONOutput(t *testing.T) {
	tmp := t.TempDir()
	scaffoldGeneratorContractFiles(t, tmp, map[string]string{
		"huaweicloud-skill-generator/references/huaweicloud-skill-template.md": "metadata:\n  gcl:\n    required: true\n    default_max_iter: 2\n    rubric_version: \"v1\"\n    trace_path: audit-results/gcl-trace-YYYYMMDD-HHMMSS.json\n## Quality Gate (GCL)\nreferences/rubric.md\nreferences/prompt-templates.md\noperation_intent\ngcl-prompt-backbone.md\n",
		"huaweicloud-skill-generator/SKILL.md": "`references/rubric.md`\n`references/prompt-templates.md`\n`metadata.gcl`\nreferences/gcl-prompt-backbone.md\n",
		"huaweicloud-skill-generator/references/gcl-prompt-backbone.md": "## 1. Generator prompt template\n## 2. Critic prompt template\n## 3. Orchestrator prompt template\nPRIMARY: hcloud\nhuaweicloud-sdk-go-v3\n{{output.operation_intent}}\nDo NOT consider the original user request\nread-only\naudit-results/gcl-trace-YYYYMMDD-HHMMSS.json\n",
	})
	err := runValidateGeneratorContract([]string{"--root", tmp, "--json"})
	if err != nil {
		t.Fatalf("expected JSON output to pass, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: safety-class enum
// ---------------------------------------------------------------------------

func TestSafetyClass_AllValuesConformant(t *testing.T) {
	tmp := t.TempDir()
	scaffoldGeneratorContractFiles(t, tmp, map[string]string{
		// checkSafetyClassSchema reads huaweicloud-ces-ops/assets/gcl-trace.schema.json
		// and expects operation_intent.properties.safety_class.enum
		"huaweicloud-ces-ops/assets/gcl-trace.schema.json": `{
  "properties": {
    "operation_intent": {
      "type": "object",
      "properties": {
        "safety_class": {
          "type": "string",
          "enum": ["read-only","mutating","destructive"]
        },
        "operation": {"type": "string"},
        "resource_scope": {"type": "string"},
        "expected_state": {"type": "object"}
      },
      "required": ["safety_class","operation","resource_scope","expected_state"]
    }
  }
}`,
		// checkSafetyClassCode reads skillcheck/internal/gcl/sanitizer.go from cwd
		"skillcheck/internal/gcl/sanitizer.go": "package gcl\nconst SAFETY_CLASS_VALUES = []string{\"read-only\", \"mutating\", \"destructive\"}\n",
		// checkSafetyClassDocs reads docs/gcl-spec.md and backbone
		"docs/gcl-spec.md": "## Safety Class\nread-only; mutating; destructive\n",
		"huaweicloud-skill-generator/references/gcl-prompt-backbone.md": "## Safety\nread-only; mutating; destructive\n",
		// trace with valid safety_class value (nested inside operation_intent)
		"huaweicloud-ces-ops/assets/audit-results/gcl-trace-20260101-000000.json": `{"operation_intent":{"safety_class":"read-only","operation":"list","resource_scope":"*","expected_state":{}}}`,
	})
	err := runValidateSafetyClass([]string{"--root", tmp})
	if err != nil {
		t.Fatalf("expected conformant safety_class to pass, got: %v", err)
	}
}

func TestSafetyClass_UnknownValue(t *testing.T) {
	tmp := t.TempDir()
	scaffoldGeneratorContractFiles(t, tmp, map[string]string{
		"huaweicloud-ces-ops/assets/gcl-trace.schema.json": `{
  "properties": {
    "operation_intent": {
      "type": "object",
      "properties": {
        "safety_class": {
          "type": "string",
          "enum": ["read-only","mutating"]
        },
        "operation": {"type": "string"},
        "resource_scope": {"type": "string"},
        "expected_state": {"type": "object"}
      },
      "required": ["safety_class","operation","resource_scope","expected_state"]
    }
  }
}`,
		"skillcheck/internal/gcl/sanitizer.go": "package gcl\nconst SAFETY_CLASS_VALUES = []string{\"read-only\", \"mutating\"}\n",
		"docs/gcl-spec.md": "read-only; mutating; destructive\n",
		"huaweicloud-skill-generator/references/gcl-prompt-backbone.md": "read-only; mutating; destructive\n",
		"huaweicloud-ces-ops/assets/audit-results/gcl-trace-20260101-000000.json": `{"operation_intent":{"safety_class":"read-only","operation":"list","resource_scope":"*","expected_state":{}}}`,
	})
	err := runValidateSafetyClass([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure when schema missing 'destructive', got nil")
	}
}

func TestSafetyClass_InvalidValueInTrace(t *testing.T) {
	tmp := t.TempDir()
	scaffoldGeneratorContractFiles(t, tmp, map[string]string{
		"huaweicloud-ces-ops/assets/gcl-trace.schema.json": `{
  "properties": {
    "operation_intent": {
      "type": "object",
      "properties": {
        "safety_class": {
          "type": "string",
          "enum": ["read-only","mutating","destructive"]
        },
        "operation": {"type": "string"},
        "resource_scope": {"type": "string"},
        "expected_state": {"type": "object"}
      },
      "required": ["safety_class","operation","resource_scope","expected_state"]
    }
  }
}`,
		"skillcheck/internal/gcl/sanitizer.go": "package gcl\nconst SAFETY_CLASS_VALUES = []string{\"read-only\", \"mutating\", \"destructive\"}\n",
		"docs/gcl-spec.md": "read-only; mutating; destructive\n",
		"huaweicloud-skill-generator/references/gcl-prompt-backbone.md": "read-only; mutating; destructive\n",
		// trace with invalid safety_class value "super-secret"
		"audit-results/gcl-trace-20260101-000000.json": `{"operation_intent":{"safety_class":"super-secret","operation":"list","resource_scope":"*","expected_state":{}}}`,
	})
	err := runValidateSafetyClass([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure due to invalid safety_class value in trace, got nil")
	}
}

func TestSafetyClass_JSONOutput(t *testing.T) {
	tmp := t.TempDir()
	scaffoldGeneratorContractFiles(t, tmp, map[string]string{
		"huaweicloud-ces-ops/assets/gcl-trace.schema.json": `{
  "properties": {
    "operation_intent": {
      "type": "object",
      "properties": {
        "safety_class": {
          "type": "string",
          "enum": ["read-only","mutating","destructive"]
        },
        "operation": {"type": "string"},
        "resource_scope": {"type": "string"},
        "expected_state": {"type": "object"}
      },
      "required": ["safety_class","operation","resource_scope","expected_state"]
    }
  }
}`,
		"skillcheck/internal/gcl/sanitizer.go": "package gcl\nconst SAFETY_CLASS_VALUES = []string{\"read-only\", \"mutating\", \"destructive\"}\n",
		"docs/gcl-spec.md": "read-only; mutating; destructive\n",
		"huaweicloud-skill-generator/references/gcl-prompt-backbone.md": "read-only; mutating; destructive\n",
		"huaweicloud-ces-ops/assets/audit-results/gcl-trace-20260101-000000.json": `{"operation_intent":{"safety_class":"read-only","operation":"list","resource_scope":"*","expected_state":{}}}`,
	})
	err := runValidateSafetyClass([]string{"--root", tmp, "--json"})
	if err != nil {
		t.Fatalf("expected JSON output to pass, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: resource-scope PII masking
// ---------------------------------------------------------------------------

func TestResourceScope_ValidMaskedValues(t *testing.T) {
	tmp := t.TempDir()
	scaffoldGeneratorContractFiles(t, tmp, map[string]string{
		// Use raw string so JSON pattern contains literal backslash: pattern value is ^\*+$
		"huaweicloud-ces-ops/assets/gcl-trace.schema.json": `{"properties":{"operation_intent":{"properties":{"resource_scope":{"type":"array","items":{"anyOf":[{"pattern":"^\\*+$"},{"pattern":"^<masked>$"},{"pattern":"^[A-Za-z][A-Za-z0-9-]*-\\*+$"}]}}}}}}`,
		"skillcheck/internal/gcl/runner.go": "package gcl\nconst maskedFields = []string{\"resource_id\", \"user_id\"}\nfunc MaskResourceID(v string) string { return \"***\" }\n",
		"docs/gcl-spec.md": "## Resource Scope\nresource_scope masking: use *** or <masked> or prefix-*** for sensitive values.\n",
		"huaweicloud-skill-generator/references/gcl-prompt-backbone.md": "resource_scope\nmasking\n",
		"audit-results/gcl-trace-20260101-000000.json": `{"operation_intent":{"resource_scope": ["***"], "user": "<masked>", "account_id": "hw-***"}}`,
	})
	err := runValidateResourceScope([]string{"--root", tmp})
	if err != nil {
		t.Fatalf("expected valid masked values to pass, got: %v", err)
	}
}

func TestResourceScope_RawIDRejected(t *testing.T) {
	tmp := t.TempDir()
	scaffoldGeneratorContractFiles(t, tmp, map[string]string{
		"huaweicloud-ces-ops/assets/gcl-trace.schema.json": `{"properties":{"operation_intent":{"properties":{"resource_scope":{"type":"array","items":{"anyOf":[{"pattern":"^\\*+$"},{"pattern":"^<masked>$"},{"pattern":"^[A-Za-z][A-Za-z0-9-]*-\\*+$"}]}}}}}}`,
		"skillcheck/internal/gcl/runner.go": "package gcl\nfunc MaskResourceID(v string) string { return \"***\" }\n",
		"docs/gcl-spec.md": "## Resource Scope\nresource_scope masking required\n",
		"huaweicloud-skill-generator/references/gcl-prompt-backbone.md": "resource_scope\nmasking\n",
		"audit-results/gcl-trace-20260101-000000.json": `{"operation_intent":{"resource_scope": ["hw-abcd-1234-def"]}}`,
	})
	err := runValidateResourceScope([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure due to raw ID in trace, got nil")
	}
}

func TestResourceScope_PrefixStarOK(t *testing.T) {
	tmp := t.TempDir()
	scaffoldGeneratorContractFiles(t, tmp, map[string]string{
		// Use raw string: pattern values ^\*+$ (asterisks-only) and ^[A-Za-z][A-Za-z0-9-]*-\*+$ (prefix-asterisks)
		"huaweicloud-ces-ops/assets/gcl-trace.schema.json": `{"properties":{"operation_intent":{"properties":{"resource_scope":{"type":"array","items":{"anyOf":[{"pattern":"^\\*+$"},{"pattern":"^<masked>$"},{"pattern":"^[A-Za-z][A-Za-z0-9-]*-\\*+$"}]}}}}}}`,
		"skillcheck/internal/gcl/runner.go": "package gcl\nfunc MaskResourceID(v string) string { return \"***\" }\n",
		"docs/gcl-spec.md": "## Resource Scope\nresource_scope masking: prefix-*** allowed\n",
		"huaweicloud-skill-generator/references/gcl-prompt-backbone.md": "resource_scope\nmasking\n",
		"audit-results/gcl-trace-20260101-000000.json": `{"operation_intent":{"resource_scope": ["hw-***"]}}`,
	})
	err := runValidateResourceScope([]string{"--root", tmp})
	if err != nil {
		t.Fatalf("prefix-*** should be allowed: %v", err)
	}
}

func TestResourceScope_JSONOutput(t *testing.T) {
	tmp := t.TempDir()
	scaffoldGeneratorContractFiles(t, tmp, map[string]string{
		// Schema must have operation_intent nested structure for schema check to pass
		"huaweicloud-ces-ops/assets/gcl-trace.schema.json": `{"properties":{"operation_intent":{"properties":{"resource_scope":{"type":"array","items":{"anyOf":[{"pattern":"^\\*+$"},{"pattern":"^<masked>$"},{"pattern":"^[A-Za-z][A-Za-z0-9-]*-\\*+$"}]}}}}}}`,
		"skillcheck/internal/gcl/runner.go": "package gcl\nfunc MaskResourceID(v string) string { return \"***\" }\n",
		"docs/gcl-spec.md": "## Resource Scope\nresource_scope masking: ***, <masked>, prefix-***\n",
		"huaweicloud-skill-generator/references/gcl-prompt-backbone.md": "resource_scope\nmasking\n",
		"audit-results/gcl-trace-20260101-000000.json": `{"operation_intent":{"resource_scope": ["***"]}}`,
	})
	err := runValidateResourceScope([]string{"--root", tmp, "--json"})
	if err != nil {
		t.Fatalf("expected JSON output to pass, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: runCheckAuditResults (L2-C)
// ---------------------------------------------------------------------------

func TestCheckAuditResults_AllClean(t *testing.T) {
	tmp := t.TempDir()
	// Init real git repo (required for git ls-files check)
	if out, err := exec.Command("git", "init").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
	// .gitignore with all required patterns (use **/ for patterns that require double-asterisk)
	gitignore := "audit-results/\n**/audit-results/\ngcl-trace-*.json\n**/gcl-trace-*.json\ngcl-quality-summary-*.json\n**/gcl-quality-summary-*.json\ngcl-alarm-plan-*.json\n**/gcl-alarm-plan-*.json\n"
	if err := os.WriteFile(filepath.Join(tmp, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		t.Fatal(err)
	}
	// audit-results/ directory with mode 0700 (owner-only)
	auditDir := filepath.Join(tmp, "audit-results")
	if err := os.MkdirAll(auditDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// docs/gcl-spec.md with required fragments
	docsDir := filepath.Join(tmp, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "gcl-spec.md"),
		[]byte("audit-results/\nGCL\ngitignore\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runCheckAuditResults([]string{"--root", tmp})
	if err != nil {
		t.Fatalf("expected clean audit-results to pass, got: %v", err)
	}
}

func TestCheckAuditResults_MissingGitignoreEntry(t *testing.T) {
	tmp := t.TempDir()
	if out, err := exec.Command("git", "init").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
	// .gitignore without audit-results entries
	if err := os.WriteFile(filepath.Join(tmp, ".gitignore"), []byte("*.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	auditDir := filepath.Join(tmp, "audit-results")
	if err := os.MkdirAll(auditDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "docs", "gcl-spec.md"),
		[]byte("audit-results/\nGCL\ngitignore\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runCheckAuditResults([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure due to missing gitignore entries, got nil")
	}
}

func TestCheckAuditResults_TrackedFilesInAuditDir(t *testing.T) {
	tmp := t.TempDir()
	// git init must use Dir=tmp so the repo is created in tmp, not CWD
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmp
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".gitignore"),
		[]byte("audit-results/\n**/audit-results/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	auditDir := filepath.Join(tmp, "audit-results")
	if err := os.MkdirAll(auditDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// A tracked file inside audit-results (would normally be gitignored)
	if err := os.WriteFile(filepath.Join(auditDir, "gcl-trace-20260101-000000.json"),
		[]byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// git add -f (force) so the file is tracked despite .gitignore entry
	// This simulates a file that was forcefully added despite being gitignored
	addCmd := exec.Command("git", "add", "-f", "audit-results/gcl-trace-20260101-000000.json")
	addCmd.Dir = tmp
	if out, err := addCmd.CombinedOutput(); err != nil {
		t.Fatalf("git add -f failed: %v\n%s", err, out)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "docs", "gcl-spec.md"),
		[]byte("audit-results/\nGCL\ngitignore\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runCheckAuditResults([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure when tracked files found in audit-results, got nil")
	}
}

func TestCheckAuditResults_MissingDocsFragments(t *testing.T) {
	tmp := t.TempDir()
	if out, err := exec.Command("git", "init").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".gitignore"),
		[]byte("audit-results/\n**/audit-results/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	auditDir := filepath.Join(tmp, "audit-results")
	if err := os.MkdirAll(auditDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// docs/gcl-spec.md missing GCL fragment
	if err := os.MkdirAll(filepath.Join(tmp, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "docs", "gcl-spec.md"),
		[]byte("audit-results/\ngitignore\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runCheckAuditResults([]string{"--root", tmp})
	if err == nil {
		t.Fatal("expected failure due to missing GCL fragment in docs, got nil")
	}
}
