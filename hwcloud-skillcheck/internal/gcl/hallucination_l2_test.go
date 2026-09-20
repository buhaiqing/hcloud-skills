package gcl

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// l2SkillRoot creates a skill directory fixture. schemaJSON == "" leaves the
// skill without references/openapi-schema.json (the shipped-corpus default);
// asDirectory=true makes the schema path a directory so reading it fails with
// a non-"not exist" error.
func l2SkillRoot(t *testing.T, schemaJSON string, asDirectory bool) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "huaweicloud-ecs-ops")
	refs := filepath.Join(root, "references")
	if err := os.MkdirAll(refs, 0o755); err != nil {
		t.Fatal(err)
	}
	schemaPath := filepath.Join(refs, "openapi-schema.json")
	switch {
	case asDirectory:
		if err := os.MkdirAll(schemaPath, 0o755); err != nil {
			t.Fatal(err)
		}
	case schemaJSON != "":
		if err := os.WriteFile(schemaPath, []byte(schemaJSON), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const l2ValidSchema = `{"type":"object","required":["server_id"],"properties":{"server_id":{"type":"string"}}}`

// l2DefsSchema mirrors the shape documented for generated skills in
// huaweicloud-skill-generator/references/openapi-schema-asset.md: a
// root-level `$defs` map plus a root `$ref` into it.
const l2DefsSchema = `{"$defs":{"server":{"type":"object","required":["server_id"],"properties":{"server_id":{"type":"string"}}}},"$ref":"#/$defs/server"}`

// TestL2StatusOutcomes pins the status contract: a check that could not run
// must be distinguishable from one that passed, and a violation must still
// block.
func TestL2StatusOutcomes(t *testing.T) {
	tests := []struct {
		name        string
		schemaJSON  string
		asDirectory bool
		excerpt     string
		wantStatus  L2Status
		wantBlocked bool
		wantSkipped bool
	}{
		{
			name:        "no output",
			excerpt:     "",
			wantStatus:  L2StatusSkippedNoOutput,
			wantSkipped: true,
		},
		{
			name:       "output is not JSON",
			excerpt:    "server_id: ecs-1",
			wantStatus: L2StatusInvalidOutput,
		},
		{
			name:        "skill ships no schema",
			excerpt:     `{"server_id":"ecs-1"}`,
			wantStatus:  L2StatusSkippedNoSchema,
			wantSkipped: true,
		},
		{
			name:        "schema path unreadable",
			asDirectory: true,
			excerpt:     `{"server_id":"ecs-1"}`,
			wantStatus:  L2StatusSchemaUnreadable,
		},
		{
			name:       "schema is not valid JSON",
			schemaJSON: "not-a-schema",
			excerpt:    `{"server_id":"ecs-1"}`,
			wantStatus: L2StatusValidatorError,
		},
		{
			name:       "output conforms",
			schemaJSON: l2ValidSchema,
			excerpt:    `{"server_id":"ecs-1"}`,
			wantStatus: L2StatusPass,
		},
		{
			name:       "documented $defs shape conforms",
			schemaJSON: l2DefsSchema,
			excerpt:    `{"server_id":"ecs-1"}`,
			wantStatus: L2StatusPass,
		},
		{
			name:        "documented $defs shape violates",
			schemaJSON:  l2DefsSchema,
			excerpt:     `{"id":"ecs-1"}`,
			wantStatus:  L2StatusViolation,
			wantBlocked: true,
		},
		{
			name:       "unknown $defs ref is a loud error",
			schemaJSON: `{"$defs":{},"$ref":"#/$defs/server"}`,
			excerpt:    `{"server_id":"ecs-1"}`,
			wantStatus: L2StatusValidatorError,
		},
		{
			name:        "output violates schema",
			schemaJSON:  l2ValidSchema,
			excerpt:     `{"status":"SHUTOFF"}`,
			wantStatus:  L2StatusViolation,
			wantBlocked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skillRoot := l2SkillRoot(t, tt.schemaJSON, tt.asDirectory)
			detector := NewHallucinationDetector(skillRoot)

			result, err := asDefault(detector).checkJSONStructure(tt.excerpt)
			if err != nil {
				t.Fatalf("checkJSONStructure: %v", err)
			}
			if got := result.Status(); got != tt.wantStatus {
				t.Errorf("Status() = %q, want %q (details: %s)", got, tt.wantStatus, result.Details)
			}
			if result.Blocked != tt.wantBlocked {
				t.Errorf("Blocked = %v, want %v", result.Blocked, tt.wantBlocked)
			}
			if result.Skipped() != tt.wantSkipped {
				t.Errorf("Skipped() = %v, want %v", result.Skipped(), tt.wantSkipped)
			}
			if tt.wantStatus == L2StatusViolation && len(result.Errors) == 0 {
				t.Error("violation status without recorded errors")
			}
		})
	}
}

// TestL2StatusNilResult covers the trace-wiring call path, where L2 may be
// absent from the combined result.
func TestL2StatusNilResult(t *testing.T) {
	var result *L2Result
	if got := result.Status(); got != "" {
		t.Errorf("nil Status() = %q, want empty", got)
	}
	if result.Skipped() {
		t.Error("nil result must not report Skipped")
	}
}

// TestL2StatusSurvivesTraceJSON pins that the status token travels with the
// persisted L2 block (trace JSON), so readers can tell a skip from a pass.
func TestL2StatusSurvivesTraceJSON(t *testing.T) {
	skillRoot := l2SkillRoot(t, "", false)
	detector := NewHallucinationDetector(skillRoot)

	result, err := asDefault(detector).checkJSONStructure(`{"server_id":"ecs-1"}`)
	if err != nil {
		t.Fatalf("checkJSONStructure: %v", err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var restored L2Result
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := restored.Status(); got != L2StatusSkippedNoSchema {
		t.Errorf("round-tripped Status() = %q, want %q", got, L2StatusSkippedNoSchema)
	}
	if !strings.Contains(string(data), string(L2StatusSkippedNoSchema)) {
		t.Errorf("persisted L2 JSON %s does not carry the status token", data)
	}
}

// TestL2SkippedNoSchemaWarnsOnce proves the capability gap is reported instead
// of passing silently, and that repeated iterations of the same skill do not
// flood the log.
func TestL2SkippedNoSchemaWarnsOnce(t *testing.T) {
	var logs bytes.Buffer
	prevLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(prevLogger)
	// sync.Map must not be copied (go vet): clear the dedup table in place and
	// leave it empty afterwards. A leak into another test would only suppress
	// its warning, and tests that care clear it themselves.
	l2SkipWarned.Range(func(k, _ any) bool { l2SkipWarned.Delete(k); return true })
	defer func() { l2SkipWarned.Range(func(k, _ any) bool { l2SkipWarned.Delete(k); return true }) }()

	skillRoot := l2SkillRoot(t, "", false)
	detector := asDefault(NewHallucinationDetector(skillRoot))

	for i := 0; i < 3; i++ {
		if _, err := detector.checkJSONStructure(`{"server_id":"ecs-1"}`); err != nil {
			t.Fatalf("checkJSONStructure: %v", err)
		}
	}

	out := logs.String()
	if got := strings.Count(out, "L2 hallucination check skipped"); got != 1 {
		t.Fatalf("want exactly 1 skip warning for 3 iterations, got %d:\n%s", got, out)
	}
	if !strings.Contains(out, "level=WARN") {
		t.Errorf("warning not emitted at WARN level:\n%s", out)
	}
	if !strings.Contains(out, "missing references/openapi-schema.json") {
		t.Errorf("warning does not name the missing asset:\n%s", out)
	}
	if !strings.Contains(out, "huaweicloud-ecs-ops") {
		t.Errorf("warning does not name the skill:\n%s", out)
	}
}

// TestL2StatusThroughDetectorRun proves the status is reachable through the
// existing detector API (HallucinationResult.L2), not only via the unexported
// check helper.
func TestL2StatusThroughDetectorRun(t *testing.T) {
	skillRoot := l2SkillRoot(t, "", false)
	detector := NewHallucinationDetector(skillRoot)

	res, err := detector.Run(context.Background(), GeneratorOutput{
		Command:       "hcloud ecs show-server --server-id ecs-1",
		ResultExcerpt: `{"server_id":"ecs-1"}`,
	}, &GCLTrace{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.L2 == nil {
		t.Fatal("Run returned no L2 result")
	}
	if got := res.L2.Status(); got != L2StatusSkippedNoSchema {
		t.Errorf("L2.Status() = %q, want %q", got, L2StatusSkippedNoSchema)
	}
	if !res.L2.Skipped() {
		t.Error("L2 should report Skipped for a skill without an OpenAPI schema")
	}
	if res.L2.Blocked {
		t.Error("a skipped L2 check must not block the run")
	}
}
