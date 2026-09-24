package l4

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadFailurePatternsForSkill_SeedOnlyFreshClone pins spec #T6: fresh
// clones carry only the tracked seed file (failure_patterns.json is gitignored),
// so the pre-execution risk gate must read through learning.LoadFailurePatterns
// and still see the curated patterns.
func TestReadFailurePatternsForSkill_SeedOnlyFreshClone(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-ecs-ops", "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{"$schema":"failure-patterns/v1","skill_id":"huaweicloud-ecs-ops","patterns":[{"id":"ECS-FP001","category":"permission","signature":{"error_code":"","error_message_regex":"Quota exceeded","command_pattern":"*"},"root_cause":"quota"}],"meta":{"total_patterns":1}}`
	if err := os.WriteFile(filepath.Join(dir, "failure_patterns.seed.json"), []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readFailurePatternsForSkill(root, "huaweicloud-ecs-ops")
	if err != nil {
		t.Fatalf("readFailurePatternsForSkill: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("patterns = %d, want 1 served from seed (fresh-clone shape)", len(got))
	}
	if id, _ := got[0]["id"].(string); id != "ECS-FP001" {
		t.Errorf("pattern id = %q, want ECS-FP001", id)
	}

	// The skill-id guard must still fail closed before any path join.
	if _, err := readFailurePatternsForSkill(root, "../escape"); err == nil {
		t.Error("path-traversal skill id must be rejected")
	}
	if _, err := readFailurePatternsForSkill(root, ""); err == nil {
		t.Error("empty skill id must be rejected")
	}
}
