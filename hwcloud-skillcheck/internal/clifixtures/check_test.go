package clifixtures

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheck_MatchesFixture(t *testing.T) {
	root := t.TempDir()
	fixDir := filepath.Join(root, "internal", "clifixtures", "fixtures")
	_ = os.MkdirAll(fixDir, 0o700)
	_ = os.WriteFile(filepath.Join(fixDir, "gcl__run__smoke.json"),
		[]byte(`{"args":["gcl","run","smoke"],"stdout_excerpt":"PASS\n","stderr_excerpt":"","exit_code":0}`), 0o600)
	ok, err := Check(root, []string{"gcl", "run", "smoke"}, "PASS\n", "", 0)
	if err != nil || !ok {
		t.Errorf("Check returned ok=%v err=%v, want true nil", ok, err)
	}
}

func TestCheck_DetectsDrift(t *testing.T) {
	root := t.TempDir()
	fixDir := filepath.Join(root, "internal", "clifixtures", "fixtures")
	_ = os.MkdirAll(fixDir, 0o700)
	_ = os.WriteFile(filepath.Join(fixDir, "x.json"),
		[]byte(`{"args":["x"],"stdout_excerpt":"OLD\n","stderr_excerpt":"","exit_code":0}`), 0o600)
	ok, _ := Check(root, []string{"x"}, "NEW\n", "", 0)
	if ok {
		t.Error("drift should be detected")
	}
}
