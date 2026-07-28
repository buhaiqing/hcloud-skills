package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRouterEmbedTestLocalPreflight(t *testing.T) {
	root := t.TempDir()
	registry := `{"embedding":{"mode":"local","provider_name":"local-fasttext","dim":128,"timeout_ms":500}}`
	if err := os.WriteFile(filepath.Join(root, "capability-registry.json"), []byte(registry), 0o600); err != nil {
		t.Fatal(err)
	}
	var callErr error
	output := captureStdout(t, func() {
		callErr = runRouterEmbedTest([]string{"--root", root, "--text", "list ecs"})
	})
	if callErr != nil {
		t.Fatal(callErr)
	}
	for _, want := range []string{"sandbox preflight: PASS", "provider: local-fasttext", "embedding smoke test: PASS", "vector_dim: 128"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestRouterEmbedTestCloudFriendlyPreflight(t *testing.T) {
	root := t.TempDir()
	registry := `{"embedding":{"mode":"cloud","provider_name":"huaweicloud-modelarts","endpoint":"http://example.com","auth_env":"MISSING_MODELARTS_AUTH","dim":384,"timeout_ms":700}}`
	if err := os.WriteFile(filepath.Join(root, "capability-registry.json"), []byte(registry), 0o600); err != nil {
		t.Fatal(err)
	}
	var callErr error
	output := captureStdout(t, func() {
		callErr = runRouterEmbedTest([]string{"--root", root})
	})
	if callErr == nil {
		t.Fatal("invalid cloud config unexpectedly passed")
	}
	for _, want := range []string{"sandbox preflight: FAIL", "error [endpoint]", "error [auth_env]", "error [timeout_ms]", "Fix:"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}
