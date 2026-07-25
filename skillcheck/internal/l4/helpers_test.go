package l4

import (
	"os"
	"path/filepath"
)

// readFile / writeFileImpl / mkdirAll are shared test helpers used across
// the *_test.go files in this package. They are package-private so they
// don't pollute the public API.
func readFile(p string) ([]byte, error) { return os.ReadFile(p) }
func writeFileImpl(p, content string) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(content), 0o644)
}
func mkdirAll(p string) error { return os.MkdirAll(p, 0o755) }
