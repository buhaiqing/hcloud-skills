package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMergeVerify_ParsesLog validates a parallel-execution log JSON is parsed
// into merge entries.
func TestMergeVerify_ParsesLog(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "parallel-log.json")
	log := `[
		{"agent":"sa-1","files":["huaweicloud-ecs-ops/SKILL.md"],"status":"success","result":"updated"},
		{"agent":"sa-2","files":["huaweicloud-rds-ops/SKILL.md"],"status":"success","result":"updated"}
	]`
	if err := os.WriteFile(logPath, []byte(log), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := loadMergeLog(logPath)
	if err != nil {
		t.Fatalf("loadMergeLog: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	if entries[0].Agent != "sa-1" || entries[0].Status != "success" {
		t.Errorf("entry[0] parsed wrong: %+v", entries[0])
	}
	if len(entries[1].Files) != 1 || entries[1].Files[0] != "huaweicloud-rds-ops/SKILL.md" {
		t.Errorf("entry[1].Files parsed wrong: %+v", entries[1])
	}
}

// TestMergeVerify_ValidatesFiles runs the cross-file link check on a fixture
// with valid and broken links, and verifies the broken one is caught.
func TestMergeVerify_ValidatesFiles(t *testing.T) {
	root := t.TempDir()
	// A good markdown file with a valid relative link to an existing file.
	good := filepath.Join(root, "good.md")
	os.WriteFile(good, []byte("# Good\n\nSee [x](./existing.md)\n"), 0o644)
	os.WriteFile(filepath.Join(root, "existing.md"), []byte("existing"), 0o644)
	// A bad markdown file with a broken link.
	bad := filepath.Join(root, "bad.md")
	os.WriteFile(bad, []byte("# Bad\n\nSee [x](./missing.md)\n"), 0o644)

	entries := []MergeEntry{
		{Agent: "sa-1", Files: []string{"good.md"}, Status: "success"},
		{Agent: "sa-2", Files: []string{"bad.md"}, Status: "success"},
	}
	report := runMergeVerify(root, entries)
	// sa-2's bad.md has a broken link → it must be flagged with missing.md.
	if !strings.Contains(report, "bad.md") || !strings.Contains(report, "missing.md") {
		t.Errorf("sa-2's bad.md broken link should be flagged, got:\n%s", report)
	}
	// sa-1's good.md must NOT be flagged as broken (its link is valid).
	if strings.Contains(report, "good.md:") && strings.Contains(report, "missing") {
		t.Errorf("sa-1's good.md must not have a broken-link finding:\n%s", report)
	}
}
