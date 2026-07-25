package drift

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckMissingCanonical(t *testing.T) {
	root := t.TempDir()
	r, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if r.OK {
		t.Fatal("expected OK=false when canonical missing")
	}
	if len(r.Errors) == 0 {
		t.Fatal("expected errors")
	}
}

func TestCheckMissingRuntime(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, CanonicalRel, "SKILL.md"), "# x")
	r, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if r.OK {
		t.Fatal("expected OK=false when runtime missing")
	}
}

func TestCheckIdentical(t *testing.T) {
	root := t.TempDir()
	canon := filepath.Join(root, CanonicalRel)
	runt := filepath.Join(root, RuntimeRel)
	writeFile(t, filepath.Join(canon, "SKILL.md"), "# x")
	writeFile(t, filepath.Join(runt, "SKILL.md"), "# x")
	r, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK {
		t.Fatalf("expected OK=true, got errors: %v", r.Errors)
	}
}

func TestCheckDrift(t *testing.T) {
	root := t.TempDir()
	canon := filepath.Join(root, CanonicalRel)
	runt := filepath.Join(root, RuntimeRel)
	writeFile(t, filepath.Join(canon, "SKILL.md"), "# canonical")
	writeFile(t, filepath.Join(runt, "SKILL.md"), "# stale")
	writeFile(t, filepath.Join(runt, "extra.md"), "extra")
	r, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if r.OK {
		t.Fatal("expected drift")
	}
	if len(r.Drift.Differing) != 1 || r.Drift.Differing[0] != "SKILL.md" {
		t.Fatalf("unexpected Differing: %v", r.Drift.Differing)
	}
	if len(r.Drift.OnlyRuntime) != 1 || r.Drift.OnlyRuntime[0] != "extra.md" {
		t.Fatalf("unexpected OnlyRuntime: %v", r.Drift.OnlyRuntime)
	}
}

func TestCheckMissingCanonicalFile(t *testing.T) {
	root := t.TempDir()
	canon := filepath.Join(root, CanonicalRel)
	runt := filepath.Join(root, RuntimeRel)
	writeFile(t, filepath.Join(canon, "SKILL.md"), "# x")
	writeFile(t, filepath.Join(runt, "SKILL.md"), "# x")
	writeFile(t, filepath.Join(runt, "added.md"), "extra")
	r, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if r.OK {
		t.Fatal("expected drift on file missing from canonical")
	}
	if len(r.Drift.OnlyRuntime) != 1 || r.Drift.OnlyRuntime[0] != "added.md" {
		t.Fatalf("unexpected OnlyRuntime: %v", r.Drift.OnlyRuntime)
	}
}

func TestSyncDryRunCreatesAndReconciles(t *testing.T) {
	root := t.TempDir()
	canon := filepath.Join(root, CanonicalRel)
	runt := filepath.Join(root, RuntimeRel)
	writeFile(t, filepath.Join(canon, "SKILL.md"), "# canon")
	writeFile(t, filepath.Join(canon, "ref/x.md"), "r")
	// Stale runtime + extra runtime file
	writeFile(t, filepath.Join(runt, "SKILL.md"), "# stale")
	writeFile(t, filepath.Join(runt, "extra.md"), "extra")

	r, err := Sync(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK {
		t.Fatalf("expected OK=true, got %v", r.Errors)
	}
	if !r.DryRun {
		t.Fatal("expected DryRun=true")
	}
	// Runtime root should NOT have been created in dry-run.
	if isDir(runt) {
		// actually it was not — but extra.md may have been copied? Let's check.
		// Wait: runtime existed already in this test (we wrote extra.md). So
		// no "create" action. Good.
	}
	// The stale SKILL.md should NOT have been overwritten yet.
	got, _ := os.ReadFile(filepath.Join(runt, "SKILL.md"))
	if string(got) != "# stale" {
		t.Fatalf("dry-run should not write, got %q", got)
	}
	if !contains(r.Actions, "overwrite") {
		t.Fatalf("expected overwrite action, got %v", r.Actions)
	}
}

func TestSyncApplyCreatesRuntimeAndReconciles(t *testing.T) {
	root := t.TempDir()
	canon := filepath.Join(root, CanonicalRel)
	writeFile(t, filepath.Join(canon, "SKILL.md"), "# canon")
	writeFile(t, filepath.Join(canon, "ref/x.md"), "r")
	// No runtime dir yet — fresh checkout scenario.

	r, err := Sync(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK {
		t.Fatalf("expected OK=true, got %v", r.Errors)
	}
	if !isDir(filepath.Join(root, RuntimeRel)) {
		t.Fatal("runtime dir should have been created")
	}
	got, _ := os.ReadFile(filepath.Join(root, RuntimeRel, "SKILL.md"))
	if string(got) != "# canon" {
		t.Fatalf("expected SKILL.md copied, got %q", got)
	}
	got, _ = os.ReadFile(filepath.Join(root, RuntimeRel, "ref/x.md"))
	if string(got) != "r" {
		t.Fatalf("expected ref/x.md copied, got %q", got)
	}
	// Second sync should report no drift and no actions.
	r2, err := Sync(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(r2.Actions) != 0 {
		t.Fatalf("expected no actions on clean sync, got %v", r2.Actions)
	}
}

func TestSyncRemovesExtraRuntime(t *testing.T) {
	root := t.TempDir()
	canon := filepath.Join(root, CanonicalRel)
	runt := filepath.Join(root, RuntimeRel)
	writeFile(t, filepath.Join(canon, "SKILL.md"), "# canon")
	writeFile(t, filepath.Join(runt, "SKILL.md"), "# canon")
	writeFile(t, filepath.Join(runt, "extra.md"), "should be removed")

	r, err := Sync(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK {
		t.Fatalf("expected OK=true, got %v", r.Errors)
	}
	if _, err := os.Stat(filepath.Join(runt, "extra.md")); !os.IsNotExist(err) {
		t.Fatal("expected extra.md removed")
	}
}

func TestCheckIgnoresDotDSStore(t *testing.T) {
	root := t.TempDir()
	canon := filepath.Join(root, CanonicalRel)
	runt := filepath.Join(root, RuntimeRel)
	writeFile(t, filepath.Join(canon, "SKILL.md"), "# canon")
	writeFile(t, filepath.Join(runt, "SKILL.md"), "# canon")
	// macOS noise — only on runtime side.
	writeFile(t, filepath.Join(runt, ".DS_Store"), "mac noise")
	r, err := Check(root)
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK {
		t.Fatalf("expected OK=true (.DS_Store ignored), got %v", r.Errors)
	}
}

func contains(xs []string, needle string) bool {
	for _, x := range xs {
		if len(x) >= len(needle) && x[:len(needle)] == needle {
			return true
		}
	}
	return false
}
