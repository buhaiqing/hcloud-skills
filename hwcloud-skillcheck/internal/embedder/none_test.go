package embedder

import (
	"context"
	"strings"
	"testing"
)

func TestNonePreflightFriendlyAndEmbeddingSkipped(t *testing.T) {
	report := (&None{}).Preflight(ProviderConfig{Endpoint: "https://unused", AuthEnv: "UNUSED", FallbackChain: []string{"local-fasttext"}})
	if !report.OK || len(report.Warnings) != 3 {
		t.Fatalf("unexpected report: %+v", report)
	}
	emb, err := PreflightAndInit(context.Background(), ProviderConfig{ProviderName: "none"})
	if err != nil {
		t.Fatalf("init none: %v", err)
	}
	vec, err := emb.Embed(context.Background(), "list ecs")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(vec) != 0 {
		t.Fatalf("none must return nil vector, got %v", vec)
	}
	if _, err := emb.Score(context.Background(), "q", "d"); err == nil || !strings.Contains(err.Error(), "rerank_mode=skipped") {
		t.Fatalf("expected friendly score error, got %v", err)
	}
}
