package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCheckLanes_DetectsCrossLaneWrite plants a self-test-tagged event inside
// audit-results/production/ and asserts the gate flags it. This is the M6
// enforcement: telemetry lane separation.
func TestCheckLanes_DetectsCrossLaneWrite(t *testing.T) {
	root := t.TempDir()
	// Plant an event in production/ that pretends to be self-test.
	bad := filepath.Join(root, "audit-results", "production", "20260101-000000-0000.json")
	if err := os.MkdirAll(filepath.Dir(bad), 0o700); err != nil {
		t.Fatalf("setup mkdir: %v", err)
	}
	if err := os.WriteFile(bad, []byte(`{"lane":"self-test","kind":"golden-run","ts":"x"}`+"\n"), 0o600); err != nil {
		t.Fatalf("setup write: %v", err)
	}
	if err := runCheckLanes([]string{"--root", root}); err == nil {
		t.Fatal("expected cross-lane write to be detected")
	}
}

// TestCheckLanes_CleanTree asserts that an empty production/ passes.
func TestCheckLanes_CleanTree(t *testing.T) {
	root := t.TempDir()
	if err := runCheckLanes([]string{"--root", root}); err != nil {
		t.Fatalf("clean tree should pass: %v", err)
	}
}

// TestCheckLanes_DetectsMalformedJSON plants a non-JSON file inside
// audit-results/production/ and asserts the gate flags it with the
// "(malformed json)" tag. An envelope we cannot parse cannot be defended as
// "production" by silence.
func TestCheckLanes_DetectsMalformedJSON(t *testing.T) {
	root := t.TempDir()
	bad := filepath.Join(root, "audit-results", "production", "broken.json")
	if err := os.MkdirAll(filepath.Dir(bad), 0o700); err != nil {
		t.Fatalf("setup mkdir: %v", err)
	}
	if err := os.WriteFile(bad, []byte("not json at all"), 0o600); err != nil {
		t.Fatalf("setup write: %v", err)
	}
	if err := runCheckLanes([]string{"--root", root}); err == nil {
		t.Fatal("expected malformed json to be detected")
	}
}
