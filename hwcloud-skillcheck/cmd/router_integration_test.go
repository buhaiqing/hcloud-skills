package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildRouterDecisionUsesRepositoryRegistry(t *testing.T) {
	root := t.TempDir()
	for _, skill := range []string{"huaweicloud-ecs-ops", "huaweicloud-rds-ops"} {
		dir := filepath.Join(root, skill)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		body := "---\nname: " + skill + "\ndescription: manage " + skill + " servers\nside_effect_class_max: read-only\nmetadata:\n  version: 1\n---\n"
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	decision, err := buildRouterDecision(filepath.Join(root, "huaweicloud-ecs-ops"), "manage ECS servers")
	if err != nil {
		t.Fatal(err)
	}
	if decision["chosen"] != "huaweicloud-ecs-ops" {
		t.Fatalf("unexpected chosen skill: %+v", decision)
	}
}

func TestBuildRouterDecisionMasksRequest(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-ecs-ops")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: huaweicloud-ecs-ops\ndescription: manage ECS servers\nmetadata:\n  version: 1\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	decision, err := buildRouterDecision(dir, "delete ecs-abc12345 server")
	if err != nil {
		t.Fatal(err)
	}
	if decision["request"] == "delete ecs-abc12345 server" {
		t.Fatal("router decision contains raw resource ID")
	}
}
