package learning

import (
	"fmt"
	"os"
	"path/filepath"
)

// readFile is a thin wrapper used by tests to surface test-only reads without
// importing "os" from the test files directly.
func readFile(p string) ([]byte, error) {
	return os.ReadFile(p)
}

// writeJSONFile is the test-only JSON writer (small wrapper around os.WriteFile).
func writeJSONFile(p string, v any) error {
	buf, err := marshal(v)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, buf, 0o644)
}

func marshal(v any) ([]byte, error) {
	// Local indirection so we don't pull encoding/json into the test helper.
	return jsonMarshal(v)
}

// writeTraceJSON is used by tests to plant a trace fixture.
func writeTraceJSON(p string, v any) error {
	return writeJSONFile(p, v)
}

func mkdirAll(p string) error {
	if p == "" {
		return fmt.Errorf("empty path")
	}
	return os.MkdirAll(p, 0o755)
}

func writeFileImpl(p, content string) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(content), 0o644)
}
