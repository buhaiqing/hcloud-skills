package gcl

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestPersistTrace_AppliesMaskedFields_Request: build a GCLTrace whose
// Request contains a resource ID and whose MaskedFields lists "request";
// after PersistTrace, the on-disk trace must have Request == "<masked>"
// (no raw resource ID).
func TestPersistTrace_AppliesMaskedFields_Request(t *testing.T) {
	tmp := t.TempDir()
	trace := &GCLTrace{
		TraceSchemaVersion: "v1",
		Skill:              "test-skill",
		Request:            "delete ecs-abc12345 server",
		OperationIntent:    nil,
		RubricVersion:      "v1",
		MaskedFields:       []string{"request"},
		Iterations:         nil,
		Final: &FinalResult{
			Status: "PASS",
			Iter:   1,
			Output: "ok",
		},
	}

	path, err := PersistTrace(trace, tmp)
	if err != nil {
		t.Fatalf("PersistTrace error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}

	// Raw resource ID must NOT appear in the file.
	if strings.Contains(string(data), "ecs-abc12345") {
		t.Errorf("raw resource ID leaked into persisted trace:\n%s", data)
	}

	// Round-trip: unmarshal and assert Request == "<masked>".
	var roundTripped GCLTrace
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal trace: %v", err)
	}
	if roundTripped.Request != "<masked>" {
		t.Errorf("Request = %q, want <masked>", roundTripped.Request)
	}
}

// TestPersistTrace_AppliesMaskedFields_GeneratorCommand: when MaskedFields
// lists "generator.command", each iteration's Generator.Command must be
// "<masked>" on the persisted trace.
func TestPersistTrace_AppliesMaskedFields_GeneratorCommand(t *testing.T) {
	tmp := t.TempDir()
	trace := &GCLTrace{
		TraceSchemaVersion: "v1",
		Skill:              "test-skill",
		Request:            "list",
		RubricVersion:      "v1",
		MaskedFields:       []string{"generator.command"},
		Iterations: []Iteration{
			{
				Iter: 1,
				Generator: GeneratorOutput{
					Command:       "ecs-abc12345",
					ExitCode:      0,
					ResultExcerpt: "ok",
				},
				Critic:   CriticResult{Scores: map[string]float64{}},
				Decision: "PASS",
			},
		},
		Final: &FinalResult{Status: "PASS", Iter: 1, Output: "ok"},
	}

	path, err := PersistTrace(trace, tmp)
	if err != nil {
		t.Fatalf("PersistTrace error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}

	var roundTripped GCLTrace
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal trace: %v", err)
	}
	if roundTripped.Iterations[0].Generator.Command != "<masked>" {
		t.Errorf("Generator.Command = %q, want <masked>",
			roundTripped.Iterations[0].Generator.Command)
	}
}

// TestPersistTrace_AppliesMaskedFields_GeneratorResultExcerpt: when
// MaskedFields lists "generator.result_excerpt", each iteration's
// Generator.ResultExcerpt must be "<masked>" on the persisted trace.
func TestPersistTrace_AppliesMaskedFields_GeneratorResultExcerpt(t *testing.T) {
	tmp := t.TempDir()
	trace := &GCLTrace{
		TraceSchemaVersion: "v1",
		Skill:              "test-skill",
		Request:            "list",
		RubricVersion:      "v1",
		MaskedFields:       []string{"generator.result_excerpt"},
		Iterations: []Iteration{
			{
				Iter: 1,
				Generator: GeneratorOutput{
					Command:       "ls",
					ExitCode:      0,
					ResultExcerpt: "secret-leak-abc12345",
				},
				Critic:   CriticResult{Scores: map[string]float64{}},
				Decision: "PASS",
			},
		},
		Final: &FinalResult{Status: "PASS", Iter: 1, Output: "ok"},
	}

	path, err := PersistTrace(trace, tmp)
	if err != nil {
		t.Fatalf("PersistTrace error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}

	var roundTripped GCLTrace
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal trace: %v", err)
	}
	if roundTripped.Iterations[0].Generator.ResultExcerpt != "<masked>" {
		t.Errorf("Generator.ResultExcerpt = %q, want <masked>",
			roundTripped.Iterations[0].Generator.ResultExcerpt)
	}
}

// TestPersistTrace_DoesNotMaskUnlistedFields: when MaskedFields does NOT
// contain "request", the raw Request must remain in the persisted trace.
func TestPersistTrace_DoesNotMaskUnlistedFields(t *testing.T) {
	tmp := t.TempDir()
	trace := &GCLTrace{
		TraceSchemaVersion: "v1",
		Skill:              "test-skill",
		Request:            "untouched sentence",
		RubricVersion:      "v1",
		MaskedFields:       []string{}, // nothing masked
		Iterations:         nil,
		Final: &FinalResult{
			Status: "PASS",
			Iter:   1,
			Output: "ok",
		},
	}

	path, err := PersistTrace(trace, tmp)
	if err != nil {
		t.Fatalf("PersistTrace error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}

	var roundTripped GCLTrace
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal trace: %v", err)
	}
	if roundTripped.Request != "untouched sentence" {
		t.Errorf("Request = %q, want untouched", roundTripped.Request)
	}
}
