package yaml

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestExtractFrontmatter verifies the --- fenced block is extracted and
// parsed into a map, mirroring validate_skills_frontmatter.py's contract
// (frontmatter must be a top-level YAML object).
func TestExtractFrontmatter(t *testing.T) {
	md := "---\nname: huaweicloud-ecs-ops\ndescription: ECS ops\ncli_applicability: cli-first\n---\n# Body\n"
	fm, err := ExtractFrontmatter([]byte(md))
	if err != nil {
		t.Fatalf("ExtractFrontmatter error: %v", err)
	}
	if fm["name"] != "huaweicloud-ecs-ops" {
		t.Errorf("name = %v, want huaweicloud-ecs-ops", fm["name"])
	}
	if fm["cli_applicability"] != "cli-first" {
		t.Errorf("cli_applicability = %v, want cli-first", fm["cli_applicability"])
	}
}

func TestExtractFrontmatterMissing(t *testing.T) {
	_, err := ExtractFrontmatter([]byte("# No frontmatter\nbody"))
	if err == nil {
		t.Error("missing frontmatter should error")
	}
}

func TestExtractFrontmatterInvalidYAML(t *testing.T) {
	md := "---\nname: : : bad\n---\n"
	_, err := ExtractFrontmatter([]byte(md))
	if err == nil {
		t.Error("invalid YAML should error")
	}
}

func TestExtractYAMLBlock(t *testing.T) {
	md := "text\n```yaml\nkey: value\nlist:\n  - a\n  - b\n```\nmore"
	block, err := ExtractYAMLBlock([]byte(md))
	if err != nil {
		t.Fatalf("ExtractYAMLBlock error: %v", err)
	}
	var data map[string]any
	if err := yaml.Unmarshal([]byte(block), &data); err != nil {
		t.Fatalf("unmarshal block: %v", err)
	}
	if data["key"] != "value" {
		t.Errorf("key = %v, want value", data["key"])
	}
	list, ok := data["list"].([]any)
	if !ok || len(list) != 2 {
		t.Errorf("list = %v, want [a,b]", list)
	}
}

func TestExtractYAMLBlockMissing(t *testing.T) {
	_, err := ExtractYAMLBlock([]byte("no code fence"))
	if err == nil {
		t.Error("missing yaml block should error")
	}
}

func TestDetectAnchors(t *testing.T) {
	lines := strings.Split("defaults: &def\n  a: 1\nuse:\n  <<: *def", "\n")
	defined, referenced, errors := DetectAnchors(lines)
	if len(defined) != 1 || !defined["def"] {
		t.Errorf("defined anchors = %v, want {def}", defined)
	}
	if len(referenced) != 1 || !referenced["def"] {
		t.Errorf("referenced anchors = %v, want {def}", referenced)
	}
	if len(errors) != 0 {
		t.Errorf("errors = %v, want none", errors)
	}
}

func TestDetectAnchorsUndefinedRef(t *testing.T) {
	lines := strings.Split("use:\n  <<: *missing", "\n")
	_, _, errors := DetectAnchors(lines)
	if len(errors) != 1 {
		t.Errorf("undefined anchor should yield 1 error, got %v", errors)
	}
}
