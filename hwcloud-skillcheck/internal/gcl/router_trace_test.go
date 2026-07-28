package gcl

import (
	"encoding/json"
	"os"
	"testing"
)

func TestRouterEmitsDecisionBlock(t *testing.T) {
	result := Run(RunConfig{
		Skill:          "huaweicloud-ecs-ops",
		Request:        "smoke",
		Command:        "echo ok",
		MaxIter:        1,
		Root:           t.TempDir(),
		RouterDecision: map[string]any{"chosen": "huaweicloud-ecs-ops", "candidates": []any{"x"}},
	})
	data, err := os.ReadFile(result.TracePath)
	if err != nil {
		t.Fatal(err)
	}
	var trace struct {
		RouterDecision map[string]any `json:"router_decision"`
	}
	if err := json.Unmarshal(data, &trace); err != nil {
		t.Fatal(err)
	}
	if trace.RouterDecision["chosen"] != "huaweicloud-ecs-ops" {
		t.Fatalf("router decision missing: %+v", trace.RouterDecision)
	}
}
