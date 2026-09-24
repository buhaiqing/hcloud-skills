package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/learning"
)

// TestGateLearningGen_CheckMode pins the P0-3 gate wiring: the pre-commit gate
// must run `gen --check` (zero writes) and enforce the tri-state contract —
// vacuous pass without the repo marker, hard fail on missing/drifted seeds,
// pass when seeds match.
func TestGateLearningGen_CheckMode(t *testing.T) {
	// No repo marker → vacuous pass (state-tolerant contract on empty roots).
	root := t.TempDir()
	_ = captureStdout(t, func() {
		if got := gateLearningGen(root); !got.passed {
			t.Errorf("marker-less root must pass, got detail=%q", got.detail)
		}
	})

	// Marker present but seeds absent → hard fail.
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "gcl-spec.md"), []byte("# gcl"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = captureStdout(t, func() {
		if got := gateLearningGen(root); got.passed {
			t.Error("marker root without seeds must fail gen --check")
		}
	})

	// Seeds generated and matching → pass.
	if _, err := learning.GenerateAll(root); err != nil {
		t.Fatalf("GenerateAll: %v", err)
	}
	_ = captureStdout(t, func() {
		if got := gateLearningGen(root); !got.passed {
			t.Errorf("matching seeds must pass, got detail=%q", got.detail)
		}
	})

	// Drifted seed → fail.
	p := filepath.Join(root, "huaweicloud-rds-ops", "assets", "failure_patterns.seed.json")
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, append(raw, ' '), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = captureStdout(t, func() {
		if got := gateLearningGen(root); got.passed {
			t.Error("drifted seed must fail gen --check")
		}
	})
}

// TestGateLearningGen_CheckNeverWrites pins the other half of the P0-3 fix: a
// failing gen --check must report drift, never repair it. The old write-mode
// gate rewrote the runtime KB on every commit.
func TestGateLearningGen_CheckNeverWrites(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "gcl-spec.md"), []byte("# gcl"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := learning.GenerateAll(root); err != nil {
		t.Fatalf("GenerateAll: %v", err)
	}
	drifted := filepath.Join(root, "huaweicloud-elb-ops", "assets", "remediation-playbooks.seed.json")
	raw, err := os.ReadFile(drifted)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(drifted, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	before := snapshotTree(t, root)
	_ = captureStdout(t, func() {
		if got := gateLearningGen(root); got.passed {
			t.Error("drifted seed must fail gen --check")
		}
	})
	after := snapshotTree(t, root)
	for path, want := range before {
		if got := after[path]; got != want {
			t.Errorf("gen --check mutated %s", path)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			t.Errorf("gen --check created %s", path)
		}
	}
}

// snapshotTree maps every file under root (relative path) to its contents.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[rel] = string(raw)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	return out
}
