package router

import (
	"context"
	"strings"
	"testing"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/embedder"
)

func samplePolicy(primary, fallback string) *Policy {
	return &Policy{
		RouterPolicyVersion: "v1.0.0-test",
		ConfidenceGate:      GateThresholds{Top1ScoreMin: 100, MarginMin: 0, EntityMatch: []string{"strong", "weak", "absent"}},
		Embedding: embedder.ProviderConfig{
			Mode:          "",
			ProviderName:  primary,
			Endpoint:      "https://invalid.example.invalid/v1/infers/test",
			AuthEnv:       "MISSING_AUTH",
			Dim:           64,
			TimeoutMs:     500,
			FallbackChain: []string{fallback},
		},
	}
}

func TestRouteWithPolicyFallbackToLocal(t *testing.T) {
	policy := samplePolicy("huaweicloud-modelarts", "local-fasttext")
	decision, err := RouteWithPolicy(context.Background(), sampleEntries(), "list ecs servers", Intent{SafetyClass: "read-only"}, policy)
	if err != nil {
		t.Fatalf("fallback failed: %v", err)
	}
	if !decision.FallbackUsed {
		t.Fatalf("expected fallback_used=true; got %+v", decision.EmbeddingProviderMeta)
	}
	if decision.EmbeddingProviderMeta.ActiveProvider != "local-fasttext" || decision.EmbeddingProviderMeta.Primary != "huaweicloud-modelarts" {
		t.Fatalf("unexpected meta: %+v", decision.EmbeddingProviderMeta)
	}
}

func TestRouteWithPolicyNoneSkipsStage2(t *testing.T) {
	policy := samplePolicy("none", "local-fasttext")
	decision, err := RouteWithPolicy(context.Background(), sampleEntries(), "list ecs servers", Intent{SafetyClass: "read-only"}, policy)
	if err != nil {
		t.Fatalf("none: %v", err)
	}
	if decision.RerankMode != "skipped" {
		t.Fatalf("expected rerank_mode=skipped, got %q", decision.RerankMode)
	}
	if decision.EmbeddingProviderMeta.ActiveProvider != "none" {
		t.Fatalf("expected active=none, got %+v", decision.EmbeddingProviderMeta)
	}
}

func TestRouteWithPolicyReportsWhenAllProvidersFail(t *testing.T) {
	policy := samplePolicy("huaweicloud-modelarts", "huaweicloud-modelarts")
	_, err := RouteWithPolicy(context.Background(), sampleEntries(), "list ecs", Intent{SafetyClass: "read-only"}, policy)
	if err == nil {
		t.Fatal("expected error when primary and all fallbacks fail")
	}
	if !strings.Contains(err.Error(), "primary embedder") {
		t.Fatalf("error should explain which provider failed: %v", err)
	}
}
