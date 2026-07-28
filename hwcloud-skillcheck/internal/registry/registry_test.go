package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryBootDoesNotReadBody(t *testing.T) {
	root := t.TempDir()
	skill := filepath.Join(root, "huaweicloud-ecs-ops")
	if err := os.Mkdir(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: huaweicloud-ecs-ops\ndescription: ECS operations\nmetadata:\n  version: 1.2.3\n  cli_applicability: cli-first\n---\n{{body must not be parsed}}\n"
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	registry, err := Boot(root)
	if err != nil {
		t.Fatalf("Boot: %v", err)
	}
	entry, ok := registry.Lookup("huaweicloud-ecs-ops")
	if !ok {
		t.Fatal("expected ECS entry")
	}
	if entry.Description != "ECS operations" || entry.Version != "1.2.3" {
		t.Fatalf("frontmatter not indexed: %+v", entry)
	}
}

func TestBootSkipsMetaSkill(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"huaweicloud-ecs-ops", "huaweicloud-skill-generator"} {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		body := "---\nname: " + name + "\ndescription: test\nmetadata:\n  version: 1\n---\n"
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	registry, err := Boot(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Lookup("huaweicloud-skill-generator"); ok {
		t.Fatal("meta skill must not be executable registry entry")
	}
}
