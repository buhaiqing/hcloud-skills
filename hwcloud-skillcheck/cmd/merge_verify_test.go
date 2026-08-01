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

// TestMergeVerify_OrchestratorMultiAgent verifies a full parallel-execution
// collection: several subagents each change a file; the merge-verify gate
// validates all cross-file references and passes when none are broken.
func TestMergeVerify_OrchestratorMultiAgent(t *testing.T) {
	root := t.TempDir()
	// Build a small fake skill tree with cross-references.
	os.MkdirAll(filepath.Join(root, "huaweicloud-ecs-ops"), 0o755)
	os.WriteFile(filepath.Join(root, "huaweicloud-ecs-ops", "SKILL.md"), []byte("ecs skill"), 0o644)
	os.WriteFile(filepath.Join(root, "a.md"), []byte("# a\nSee [ecs](./huaweicloud-ecs-ops/SKILL.md)\n"), 0o644)
	os.WriteFile(filepath.Join(root, "b.md"), []byte("# b\nSee [a](./a.md)\n"), 0o644)

	entries := []MergeEntry{
		{Agent: "sa-1", Files: []string{"a.md"}, Status: "success"},
		{Agent: "sa-2", Files: []string{"b.md"}, Status: "success"},
	}
	report := runMergeVerify(root, entries)
	// Both agents must appear with ok files.
	if !strings.Contains(report, "sa-1") || !strings.Contains(report, "sa-2") {
		t.Errorf("report must list all agents, got:\n%s", report)
	}
	if strings.Contains(report, "missing markdown") {
		t.Errorf("no broken links expected, got:\n%s", report)
	}
	if !strings.Contains(report, "cross-file link validation: PASS") {
		t.Errorf("want PASS summary, got:\n%s", report)
	}
}

// TestMergeVerify_OrchestratorDetectsConflict verifies that when a parallel
// subagent introduces a broken cross-file link, merge-verify flags that agent.
func TestMergeVerify_OrchestratorDetectsConflict(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "ok.md"), []byte("# ok\n"), 0o644)
	os.WriteFile(filepath.Join(root, "broken.md"), []byte("# broken\nSee [x](./ghost.md)\n"), 0o644)

	entries := []MergeEntry{
		{Agent: "sa-ok", Files: []string{"ok.md"}, Status: "success"},
		{Agent: "sa-broken", Files: []string{"broken.md"}, Status: "success"},
	}
	report := runMergeVerify(root, entries)
	// The broken file's agent must be associated with the finding.
	if !strings.Contains(report, "broken.md") || !strings.Contains(report, "ghost.md") {
		t.Errorf("broken cross-file link must be flagged, got:\n%s", report)
	}
	// A validly-linked agent's file must not appear as broken.
	if strings.Contains(report, "ok.md:") && strings.Contains(report, "missing") {
		t.Errorf("ok.md must not have a finding:\n%s", report)
	}
}

// TestMergeVerify_OrchestratorFailedAgentFailsGate verifies that a subagent
// reporting status=failed causes runCheckMergeVerify to return an error
// (the gate fails the whole parallel merge).
func TestMergeVerify_OrchestratorFailedAgentFailsGate(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "parallel-log.json")
	os.WriteFile(logPath, []byte(`[
		{"agent":"sa-1","files":[],"status":"success"},
		{"agent":"sa-2","files":[],"status":"failed","result":"bug"}
	]`), 0o644)

	err := runCheckMergeVerify([]string{"--root", root, "--log", logPath})
	if err == nil {
		t.Fatal("merge-verify gate must fail when a subagent reports failed")
	}
	if !strings.Contains(err.Error(), "1 subagent(s) reported failure") {
		t.Errorf("unexpected error text: %v", err)
	}
}

// TestMergeVerify_OrchestratorAllSuccessPassesGate verifies a clean parallel
// merge (all agents success, no broken links) passes the gate.
func TestMergeVerify_OrchestratorAllSuccessPassesGate(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "parallel-log.json")
	os.WriteFile(logPath, []byte(`[
		{"agent":"sa-1","files":[],"status":"success"},
		{"agent":"sa-2","files":[],"status":"success"}
	]`), 0o644)

	_ = captureStdout(t, func() {
		if err := runCheckMergeVerify([]string{"--root", root, "--log", logPath}); err != nil {
			t.Errorf("clean parallel merge should pass gate, got err=%v", err)
		}
	})
}
