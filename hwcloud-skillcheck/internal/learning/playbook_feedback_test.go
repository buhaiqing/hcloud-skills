package learning

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRecordPlaybookOutcome_Success verifies a successful fix bumps the
// playbook's metadata.success_rate (closed loop at the playbook level).
func TestRecordPlaybookOutcome_Success(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-ecs-ops", "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := `{
		"$schema": "remediation-playbooks/v1",
		"skill_id": "huaweicloud-ecs-ops",
		"playbooks": [{
			"id": "ECS-R001",
			"name": "Fix",
			"trigger": {},
			"diagnosis": {},
			"remediation": {"risk_level": "low", "auto_execute_threshold": 0.7, "execute": "x", "verification": "y"},
			"metadata": {"success_rate": 0.98, "avg_execution_seconds": 10}
		}]
	}`
	if err := os.WriteFile(filepath.Join(dir, "remediation-playbooks.json"), []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RecordPlaybookOutcome(root, "huaweicloud-ecs-ops", "ECS-R001", true); err != nil {
		t.Fatalf("RecordPlaybookOutcome(success): %v", err)
	}
	// Reload and assert success_rate rose.
	pbs, err := LoadPlaybooks(root, "huaweicloud-ecs-ops")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(pbs) != 1 {
		t.Fatalf("expected 1 playbook, got %d", len(pbs))
	}
	rate := toFloat(pbs[0].Metadata["success_rate"])
	if rate <= 0.98 {
		t.Errorf("success_rate=%v, want > 0.98 after success", rate)
	}
}

// TestRecordPlaybookOutcome_UnknownPlaybook verifies graceful no-op (no error)
// when the playbook ID is not found.
func TestRecordPlaybookOutcome_UnknownPlaybook(t *testing.T) {
	root := t.TempDir()
	if err := RecordPlaybookOutcome(root, "huaweicloud-ecs-ops", "NOPE", true); err != nil {
		t.Fatalf("unknown playbook must not error, got %v", err)
	}
}
