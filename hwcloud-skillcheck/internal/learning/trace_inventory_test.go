package learning

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTraceFile(t *testing.T, root, name string, body any) {
	t.Helper()
	dir := filepath.Join(root, "audit-results")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir audit-results: %v", err)
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal trace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
		t.Fatalf("write trace: %v", err)
	}
}

func TestComputeTraceInventory_MissingDirectoryIsEmpty(t *testing.T) {
	inv, err := ComputeTraceInventory(t.TempDir())
	if err != nil {
		t.Fatalf("a missing audit-results/ must not be an error: %v", err)
	}
	if inv.TotalFiles != 0 || inv.Evidence != 0 {
		t.Fatalf("expected an empty inventory, got %+v", inv)
	}
}

func TestComputeTraceInventory_ClassifiesEveryTrace(t *testing.T) {
	root := t.TempDir()
	// Reuse the schema-valid builders from trace_test.go: the inventory runs
	// the same ClassifyTrace the report does, so a hand-rolled trace that the
	// schema rejects would test nothing.
	evidenceFinal := map[string]any{
		"status": "PASS", "iter": 1, "output": nil,
		"failure_pattern": nil, "critic_type": "structural",
	}
	writeTraceFile(t, root, "gcl-trace-1.json", schemaValidTrace("huaweicloud-ecs-ops", evidenceFinal))
	// evidence but unattributed — the other reason findings stay empty
	writeTraceFile(t, root, "orchestrator-trace-2.json", schemaValidTrace("unknown", evidenceFinal))
	// schema-invalid legacy file: no final block
	writeTraceFile(t, root, "orchestrator-trace-3.json", map[string]any{"skill": "unknown"})

	inv, err := ComputeTraceInventory(root)
	if err != nil {
		t.Fatalf("inventory: %v", err)
	}
	if inv.TotalFiles != 3 {
		t.Fatalf("total_files = %d, want 3 (%+v)", inv.TotalFiles, inv)
	}
	if inv.Evidence != 2 || inv.Attributable != 1 {
		t.Fatalf("evidence/attributable = %d/%d, want 2/1 (%+v)", inv.Evidence, inv.Attributable, inv)
	}
	if inv.SchemaInvalid != 1 {
		t.Fatalf("schema_invalid = %d, want 1 (%+v)", inv.SchemaInvalid, inv)
	}
	if inv.ByFamily["gcl"] != 1 || inv.ByFamily["orchestrator"] != 2 {
		t.Fatalf("by_family = %v, want gcl=1 orchestrator=2", inv.ByFamily)
	}
}

func TestComputeTraceInventory_CountsUnparsableFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "audit-results")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gcl-trace-broken.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A non-trace file in the same directory must be ignored entirely.
	if err := os.WriteFile(filepath.Join(dir, "notes.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	inv, err := ComputeTraceInventory(root)
	if err != nil {
		t.Fatalf("inventory: %v", err)
	}
	if inv.Unparsable != 1 || inv.TotalFiles != 1 {
		t.Fatalf("unparsable/total = %d/%d, want 1/1 (%+v)", inv.Unparsable, inv.TotalFiles, inv)
	}
}
