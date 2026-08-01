package learning

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// RemediationPlaybook is the strongly-typed runtime shape of a
// remediation-playbooks/v1 entry, consumed by the L4 autofix executor. It is
// distinct from the weak-typed Playbook (used by `learning gen` to write
// assets): here remediation.execute / verification / rollback /
// auto_execute_threshold are first-class, so the autofix loop can gate, run,
// verify, and roll back an autonomous fix.
type RemediationPlaybook struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Trigger     map[string]any `json:"trigger"`
	Diagnosis   map[string]any `json:"diagnosis"`
	Remediation struct {
		RiskLevel            string   `json:"risk_level"`
		AutoExecuteThreshold float64  `json:"auto_execute_threshold"`
		Preconditions        []string `json:"preconditions"`
		DryRun               string   `json:"dry_run"`
		Execute              string   `json:"execute"`
		Verification         string   `json:"verification"`
		Rollback             string   `json:"rollback"`
		TimeoutSeconds       int      `json:"timeout_seconds"`
	} `json:"remediation"`
	Escalation map[string]any `json:"escalation"`
	Metadata   map[string]any `json:"metadata"`
}

// skillAssetID normalizes a bare shortname ("ecs") or full id
// ("huaweicloud-ecs-ops") to the on-disk skill directory name.
func skillAssetID(skill string) string {
	if strings.HasPrefix(skill, "huaweicloud-") {
		return skill
	}
	return "huaweicloud-" + skill + "-ops"
}

// LoadPlaybooks reads remediation-playbooks.json for a skill. A missing file or
// malformed JSON returns an empty slice (never an error), matching the graceful
// behavior of LoadFailurePatterns.
func LoadPlaybooks(root, skill string) ([]RemediationPlaybook, error) {
	path := filepath.Join(root, skillAssetID(skill), "assets", "remediation-playbooks.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil // missing → empty, no error
	}
	var env struct {
		Playbooks []RemediationPlaybook `json:"playbooks"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, nil // invalid JSON → empty, no error
	}
	return env.Playbooks, nil
}

// RecordPlaybookOutcome updates a remediation playbook's metadata.success_rate
// after an autonomous fix attempt (L4 self-evolution closed loop). A successful
// fix nudges success_rate up via EWMA; a failed fix de-ranks it (so the autofix
// executor's auto_execute_threshold eventually blocks it). Unknown playbook ID
// is a graceful no-op.
func RecordPlaybookOutcome(root, skill, playbookID string, success bool) error {
	path := filepath.Join(root, skillAssetID(skill), "assets", "remediation-playbooks.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var env struct {
		Playbooks []map[string]any `json:"playbooks"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil
	}
	for _, pb := range env.Playbooks {
		if id, _ := pb["id"].(string); id != playbookID {
			continue
		}
		md, _ := pb["metadata"].(map[string]any)
		if md == nil {
			md = map[string]any{}
			pb["metadata"] = md
		}
		rate := toFloat(md["success_rate"])
		if success {
			// EWMA nudge toward 1.0.
			rate = rate*0.9 + 0.1
		} else {
			// EWMA de-rank toward 0.0.
			rate *= 0.9
		}
		if rate > 1.0 {
			rate = 1.0
		}
		md["success_rate"] = rate
		md["last_updated"] = NowISO()
		// Persist back.
		payload := map[string]any{
			"$schema":   "remediation-playbooks/v1",
			"skill_id":  skillAssetID(skill),
			"playbooks": env.Playbooks,
		}
		return writeJSON(path, payload)
	}
	return nil
}
