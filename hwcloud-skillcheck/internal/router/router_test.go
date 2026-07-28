package router

import (
	"testing"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/registry"
)

func TestRouterManifestFilterLatency(t *testing.T) {
	entries := make([]registry.Entry, 0, 7)
	for i := 0; i < 7; i++ {
		entries = append(entries, registry.Entry{
			Skill:           "huaweicloud-ecs-ops-" + string(rune('a'+i)),
			Name:            "ECS operations",
			Description:     "manage ECS servers and disks",
			SideEffectClass: "read-only",
		})
	}
	entries = append(entries, registry.Entry{
		Skill:           "huaweicloud-danger-ops",
		Name:            "Danger",
		Description:     "manage ECS servers",
		SideEffectClass: "destructive",
	})

	got := ManifestFilter(entries, "list ECS servers", Intent{SafetyClass: "read-only"})
	if len(got) != 5 {
		t.Fatalf("got %d candidates, want 5", len(got))
	}
	for _, candidate := range got {
		if candidate.Skill == "huaweicloud-danger-ops" {
			t.Fatal("destructive-only skill must not surface for read-only intent")
		}
	}
}

func TestManifestFilterIsDeterministicOnTies(t *testing.T) {
	entries := []registry.Entry{
		{Skill: "huaweicloud-z-ops", Description: "ECS server"},
		{Skill: "huaweicloud-a-ops", Description: "ECS server"},
	}
	got := ManifestFilter(entries, "ECS server", Intent{SafetyClass: "read-only"})
	if len(got) != 2 || got[0].Skill != "huaweicloud-a-ops" {
		t.Fatalf("tie ordering is unstable: %+v", got)
	}
}
