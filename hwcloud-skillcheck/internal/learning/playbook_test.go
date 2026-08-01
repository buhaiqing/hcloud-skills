package learning

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadPlaybooks_ParsesRemediation verifies the loader reads execute/
// verification/rollback/auto_execute_threshold from a playbook fixture.
func TestLoadPlaybooks_ParsesRemediation(t *testing.T) {
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
			"name": "Alarm Re-Enable",
			"trigger": {"metric": "alarm_status"},
			"diagnosis": {"steps": ["hcloud CES listAlarms"]},
			"remediation": {
				"risk_level": "low",
				"auto_execute_threshold": 0.7,
				"preconditions": ["hcloud CES getAlarm --alarm_id {{output.alarm_id}}"],
				"dry_run": "hcloud CES listAlarms --dry-run",
				"execute": "hcloud CES enableAlarm --alarm_id {{output.alarm_id}}",
				"verification": "hcloud CES getAlarm --alarm_id {{output.alarm_id}} | grep -q ENABLED",
				"rollback": "hcloud CES disableAlarm --alarm_id {{output.alarm_id}}",
				"timeout_seconds": 30
			},
			"escalation": {"condition": "fails"},
			"metadata": {"success_rate": 0.98}
		}]
	}`
	if err := os.WriteFile(filepath.Join(dir, "remediation-playbooks.json"), []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}

	pbs, err := LoadPlaybooks(root, "huaweicloud-ecs-ops")
	if err != nil {
		t.Fatalf("LoadPlaybooks: %v", err)
	}
	if len(pbs) != 1 {
		t.Fatalf("expected 1 playbook, got %d", len(pbs))
	}
	pb := pbs[0]
	if pb.ID != "ECS-R001" {
		t.Errorf("id=%q, want ECS-R001", pb.ID)
	}
	if pb.Remediation.RiskLevel != "low" {
		t.Errorf("risk_level=%q, want low", pb.Remediation.RiskLevel)
	}
	if pb.Remediation.AutoExecuteThreshold != 0.7 {
		t.Errorf("auto_execute_threshold=%v, want 0.7", pb.Remediation.AutoExecuteThreshold)
	}
	if pb.Remediation.Execute == "" || pb.Remediation.Verification == "" || pb.Remediation.Rollback == "" {
		t.Errorf("execute/verification/rollback must be non-empty: %#v", pb.Remediation)
	}
	if pb.Remediation.TimeoutSeconds != 30 {
		t.Errorf("timeout_seconds=%d, want 30", pb.Remediation.TimeoutSeconds)
	}
}

// TestLoadPlaybooks_MissingFile verifies graceful empty result on absent file.
func TestLoadPlaybooks_MissingFile(t *testing.T) {
	root := t.TempDir()
	pbs, err := LoadPlaybooks(root, "huaweicloud-ecs-ops")
	if err != nil {
		t.Fatalf("LoadPlaybooks on missing file: %v", err)
	}
	if len(pbs) != 0 {
		t.Fatalf("expected 0 playbooks on missing file, got %d", len(pbs))
	}
}

// TestLoadPlaybooks_InvalidJSON verifies empty result (not error) on bad JSON.
func TestLoadPlaybooks_InvalidJSON(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "huaweicloud-ecs-ops", "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "remediation-playbooks.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	pbs, err := LoadPlaybooks(root, "huaweicloud-ecs-ops")
	if err != nil {
		t.Fatalf("LoadPlaybooks on invalid JSON: %v", err)
	}
	if len(pbs) != 0 {
		t.Fatalf("expected 0 playbooks on invalid JSON, got %d", len(pbs))
	}
}
