package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDevExSingleEntry(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "../.."))
	if _, err := os.Stat(filepath.Join(root, "Taskfile.yml")); err != nil {
		t.Fatalf("Taskfile.yml missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "Makefile")); !os.IsNotExist(err) {
		t.Fatal("Makefile must not remain after P2 DevEx migration")
	}
}
