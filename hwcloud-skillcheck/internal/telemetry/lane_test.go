package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestWrite_ProducesFileUnderLaneDir verifies that Write drops a JSON file at
// <repoRoot>/audit-results/<lane>/. To stay hermetic, the test makes the temp
// dir look like a repo root (AGENTS.md present) and chdir's into it.
func TestWrite_ProducesFileUnderLaneDir(t *testing.T) {
	dir := t.TempDir()
	// Anchor findRepoRoot at this tempdir by leaving an AGENTS.md marker and
	// running from there.
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# test\n"), 0o600); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	t.Chdir(dir)
	t.Setenv("HC_TELEMETRY_LANE", "self-test")

	if err := Write("self-test", "golden-run", map[string]any{"scenario": "x"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "audit-results", "self-test"))
	if err != nil {
		t.Fatalf("read lane dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected a file in audit-results/self-test, got none")
	}
	if len(entries) > 1 {
		t.Fatalf("expected exactly one file, got %d", len(entries))
	}
	// Verify the file is well-formed JSON with the expected envelope.
	b, err := os.ReadFile(filepath.Join(dir, "audit-results", "self-test", entries[0].Name()))
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	var doc struct {
		Lane    string         `json:"lane"`
		Kind    string         `json:"kind"`
		TS      string         `json:"ts"`
		Payload map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("written file is not valid JSON: %v\n%s", err, b)
	}
	if doc.Lane != "self-test" {
		t.Errorf("lane field: got %q want %q", doc.Lane, "self-test")
	}
	if doc.Kind != "golden-run" {
		t.Errorf("kind field: got %q want %q", doc.Kind, "golden-run")
	}
	if doc.Payload["scenario"] != "x" {
		t.Errorf("payload.scenario: got %v want %q", doc.Payload["scenario"], "x")
	}
	if doc.TS == "" {
		t.Errorf("ts field is empty")
	}
}
