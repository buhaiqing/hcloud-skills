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