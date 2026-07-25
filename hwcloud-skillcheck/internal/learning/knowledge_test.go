// Package knowledge provides a Go port of scripts/gen_skill_knowledge.py.
// It regenerates the failure_patterns.json + remediation-playbooks.json for
// the high-frequency skills (RDS/VPC/ELB/CCE). The byte-level output must
// match the Python baseline so downstream tooling (e.g. gcl_runner) sees
// no diff.
package learning

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// TestProductsCoversTopSkills asserts the Go knowledge base covers the same
// products as the Python script. Adding a product requires updating this test.
func TestProductsCoversTopSkills(t *testing.T) {
	want := []string{"rds", "vpc", "elb", "cce"}
	for _, p := range want {
		_, ok := Products[p]
		if !ok {
			t.Errorf("Products map missing %q (must mirror gen_skill_knowledge.py PRODUCTS)", p)
		}
	}
	if len(Products) != len(want) {
		t.Errorf("Products has %d entries, want %d (no new products without test update)", len(Products), len(want))
	}
}

// TestWriteSkillAssets_GeneratesBothFiles asserts the generator emits the two
// required files with the correct envelope and content shape.
func TestWriteSkillAssets_GeneratesBothFiles(t *testing.T) {
	root := t.TempDir()
	// Construct a minimal fake skill dir to satisfy the writer's path layout.
	// The writer is expected to create huaweicloud-<short>-ops/assets/ under root.
	if err := WriteSkillAssets(root, "rds"); err != nil {
		t.Fatalf("WriteSkillAssets(rds) error: %v", err)
	}

	fpPath := filepath.Join(root, "huaweicloud-rds-ops", "assets", "failure_patterns.json")
	rpPath := filepath.Join(root, "huaweicloud-rds-ops", "assets", "remediation-playbooks.json")

	var fp map[string]any
	raw, err := readFile(fpPath)
	if err != nil {
		t.Fatalf("read failure_patterns.json: %v", err)
	}
	if err := json.Unmarshal(raw, &fp); err != nil {
		t.Fatalf("failure_patterns.json not valid JSON: %v", err)
	}
	if fp["$schema"] != "failure-patterns/v1" {
		t.Errorf("failure_patterns.json schema=%v, want failure-patterns/v1", fp["$schema"])
	}
	if fp["skill_id"] != "huaweicloud-rds-ops" {
		t.Errorf("failure_patterns.json skill_id=%v, want huaweicloud-rds-ops", fp["skill_id"])
	}
	patterns, _ := fp["patterns"].([]any)
	if len(patterns) != 10 {
		t.Errorf("rds has %d patterns, want 10", len(patterns))
	}
	meta, _ := fp["meta"].(map[string]any)
	if meta == nil || meta["total_patterns"].(float64) != 10 {
		t.Errorf("meta.total_patterns missing or wrong: %+v", meta)
	}

	var rp map[string]any
	raw, err = readFile(rpPath)
	if err != nil {
		t.Fatalf("read remediation-playbooks.json: %v", err)
	}
	if err := json.Unmarshal(raw, &rp); err != nil {
		t.Fatalf("remediation-playbooks.json not valid JSON: %v", err)
	}
	if rp["$schema"] != "remediation-playbooks/v1" {
		t.Errorf("remediation-playbooks.json schema=%v, want remediation-playbooks/v1", rp["$schema"])
	}
	playbooks, _ := rp["playbooks"].([]any)
	if len(playbooks) != 4 {
		t.Errorf("rds has %d playbooks, want 4", len(playbooks))
	}
}

// TestWriteSkillAssets_AllProducts verifies the generator runs for every
// product without panic and produces non-empty files.
func TestWriteSkillAssets_AllProducts(t *testing.T) {
	root := t.TempDir()
	for short := range Products {
		if err := WriteSkillAssets(root, short); err != nil {
			t.Errorf("WriteSkillAssets(%s) error: %v", short, err)
		}
		fpPath := filepath.Join(root, "huaweicloud-"+short+"-ops", "assets", "failure_patterns.json")
		raw, err := readFile(fpPath)
		if err != nil {
			t.Errorf("%s: %v", short, err)
			continue
		}
		if len(raw) < 100 {
			t.Errorf("%s: failure_patterns.json suspiciously small (%d bytes)", short, len(raw))
		}
		if !strings.Contains(string(raw), `"$schema": "failure-patterns/v1"`) {
			t.Errorf("%s: missing $schema header", short)
		}
	}
}
func TestPatternIDFormat(t *testing.T) {
	for short, payload := range Products {
		for i, p := range payload.Patterns {
			prefix := PatternIDPrefix(short)
			want := prefix + "-FP"
			if !strings.HasPrefix(p.ID, want) {
				t.Errorf("%s pattern[%d].id=%q, want prefix %q", short, i, p.ID, want)
			}
		}
	}
}
