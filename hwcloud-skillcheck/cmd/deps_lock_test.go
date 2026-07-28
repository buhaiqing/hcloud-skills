package cmd

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDependencyLock(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "../.."))
	lockPath := filepath.Join(root, "deps", "embed", "SHA256SUMS")
	file, err := os.Open(lockPath)
	if err != nil {
		t.Fatalf("open dependency lock: %v", err)
	}
	defer file.Close()
	wanted := map[string]bool{"all-MiniLM-L6-v2.onnx": false, "tokenizer.json": false}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 {
			if _, ok := wanted[fields[1]]; ok {
				wanted[fields[1]] = true
			}
		}
	}
	for name, found := range wanted {
		if !found {
			t.Errorf("dependency lock does not cover %s", name)
		}
	}
}
