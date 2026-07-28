package router

import (
	"context"
	"testing"
	"time"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/registry"
)

func TestRouterONNXRerankLatency(t *testing.T) {
	entries := make([]registry.Entry, 0, 20)
	for i := 0; i < 20; i++ {
		entries = append(entries, registry.Entry{Skill: "skill-" + string(rune('a'+i)), Name: "ECS", Description: "ECS server"})
	}
	started := time.Now()
	// LocalReranker was removed in spec v0.5.0; pass nil to use the package default.
	decision := Route(context.Background(), entries, "ECS server", Intent{SafetyClass: "read-only"}, nil)
	if len(decision.Candidates) != 5 || decision.Chosen == "" {
		t.Fatalf("incomplete decision: %+v", decision)
	}
	if elapsed := time.Since(started); elapsed > 50*time.Millisecond {
		t.Fatalf("stage 2 took %s", elapsed)
	}
	if decision.Candidates[0].ONNXCosine == 0 {
		t.Fatal("stage 2 score was not emitted")
	}
}
