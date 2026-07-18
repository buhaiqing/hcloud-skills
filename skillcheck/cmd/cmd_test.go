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
