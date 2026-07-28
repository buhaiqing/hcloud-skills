package embedder

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreflightReportFriendlyError(t *testing.T) {
	report := PreflightReport{Provider: "cloud", Errors: []Issue{{Field: "endpoint", Message: "missing", Fix: "set embedding.endpoint", DocURL: "docs/deployment-guide.md"}}}
	got := report.Error()
	for _, want := range []string{"sandbox preflight failed", "endpoint: missing", "Fix: set embedding.endpoint", "See: docs/deployment-guide.md"} {
		if !strings.Contains(got, want) {
			t.Fatalf("error missing %q: %s", want, got)
		}
	}
}

func TestFactoryDefaultsAndRejectsConflicts(t *testing.T) {
	emb, err := PreflightAndInit(context.Background(), ProviderConfig{})
	if err != nil || emb.Name() != "local-fasttext" {
		t.Fatalf("default provider: emb=%v err=%v", emb, err)
	}
	_, err = PreflightAndInit(context.Background(), ProviderConfig{Mode: "local", ProviderName: "huaweicloud-modelarts"})
	if err == nil || !strings.Contains(err.Error(), "conflicts") || !strings.Contains(err.Error(), "Fix:") {
		t.Fatalf("want friendly conflict error, got %v", err)
	}
}

func TestLoadConfigEnvironmentOverridesRegistry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "capability-registry.json")
	body := `{"embedding":{"mode":"cloud","provider_name":"huaweicloud-modelarts","endpoint":"https://example.com","auth_env":"TEST_AUTH","dim":128}}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HC_CAPABILITY_REGISTRY", path)
	t.Setenv("HC_EMBED_PROVIDER", "local-fasttext")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProviderName != "local-fasttext" || cfg.Mode != "local" {
		t.Fatalf("env override not applied cleanly: %+v", cfg)
	}
}
