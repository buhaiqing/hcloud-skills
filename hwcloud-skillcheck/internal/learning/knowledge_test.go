// Package knowledge provides a Go port of scripts/gen_skill_knowledge.py.
// It regenerates the failure_patterns.json + remediation-playbooks.json for
// the high-frequency skills (RDS/VPC/ELB/CCE). The byte-level output must
// match the Python baseline so downstream tooling (e.g. gcl_runner) sees
// no diff.
package learning

import (
	"encoding/json"
	"os"
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
func TestGeneratePitfallReport(t *testing.T) {
	root := t.TempDir()

	// Write two minimal failure_patterns.json files in fake skill dirs.
	for _, name := range []string{"huaweicloud-ecs-ops", "huaweicloud-vpc-ops"} {
		dir := filepath.Join(root, name, "assets")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		fp := map[string]any{
			"$schema":  "failure-patterns/v1",
			"skill_id": name,
			"patterns": []any{
				map[string]any{
					"id":         "TEST-FP001",
					"category":   "resource_state",
					"root_cause": "Instance not found",
					"prevention": "Verify instance exists before delete",
				},
				map[string]any{
					"id":         "TEST-FP002",
					"category":   "permission",
					"root_cause": "Quota exceeded",
					"prevention": "Check quota before create",
				},
			},
		}
		raw, err := json.Marshal(fp)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "failure_patterns.json"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	count, err := GeneratePitfallReport(root)
	if err != nil {
		t.Fatalf("GeneratePitfallReport error: %v", err)
	}
	if count != 4 {
		t.Errorf("count=%d, want 4 (2 skills × 2 patterns)", count)
	}

	// Verify the markdown report was written.
	reportPath := filepath.Join(root, "huaweicloud-skill-generator", "references", "common-pitfalls.md")
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("report not written: %v", err)
	}
	if !strings.Contains(string(raw), "resource_state") {
		t.Error("report missing resource_state category")
	}
	if !strings.Contains(string(raw), "permission") {
		t.Error("report missing permission category")
	}
	if !strings.Contains(string(raw), "Verify instance exists before delete") {
		t.Error("report missing first prevention text")
	}
}
