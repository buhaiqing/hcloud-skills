package learning

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

// LoadPlaybooks reads the merged seed+overlay view for a skill (seed/runtime
// KB split, spec #T3): definitions come from the tracked seed
// (remediation-playbooks.seed.json), metadata (success_rate etc.) from the
// runtime overlay (remediation-playbooks.json). Without a seed file the
// overlay alone is the document — legacy skills keep their exact pre-split
// behavior. A missing overlay+seed or an invalid overlay returns an empty
// slice (never an error), matching LoadFailurePatterns' graceful contract.
func LoadPlaybooks(root, skill string) ([]RemediationPlaybook, error) {
	dir := filepath.Join(root, skillAssetID(skill), "assets")
	type envT struct {
		Playbooks []RemediationPlaybook `json:"playbooks"`
	}
	readDoc := func(name string) (*envT, bool) { // (parsed-or-nil, file present)
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, false
		}
		var env envT
		if json.Unmarshal(raw, &env) != nil {
			return nil, true // present but invalid
		}
		return &env, true
	}
	overlay, overlayPresent := readDoc("remediation-playbooks.json")
	if overlayPresent && overlay == nil {
		return nil, nil // invalid overlay JSON → empty, no error (contract)
	}
	seed, _ := readDoc("remediation-playbooks.seed.json")
	if seed == nil { // no seed (or corrupt seed → fail open to overlay, its last good view)
		if overlay == nil {
			return nil, nil
		}
		return overlay.Playbooks, nil
	}
	ovByID := map[string]RemediationPlaybook{}
	if overlay != nil {
		for _, ov := range overlay.Playbooks {
			ovByID[ov.ID] = ov
		}
	}
	seedIDs := map[string]bool{}
	out := make([]RemediationPlaybook, 0, len(seed.Playbooks))
	for _, sp := range seed.Playbooks {
		seedIDs[sp.ID] = true
		if ov, ok := ovByID[sp.ID]; ok && ov.Metadata != nil {
			sp.Metadata = ov.Metadata // stats from overlay, defs from seed
		}
		out = append(out, sp)
	}
	if overlay != nil {
		for _, ov := range overlay.Playbooks {
			if !seedIDs[ov.ID] {
				out = append(out, ov)
			}
		}
	}
	return out, nil
}

// RecordPlaybookOutcome updates a remediation playbook's metadata.success_rate
// after an autonomous fix attempt (L4 self-evolution closed loop). A successful
// fix nudges success_rate up via EWMA; a failed fix de-ranks it (so the autofix
// executor's auto_execute_threshold eventually blocks it). The write goes to
// the runtime overlay only (seed/runtime split, spec #T4): entries present in
// the overlay are updated in place (structure preserved); a seed-known ID with
// no overlay entry gets an {id, metadata} partial appended; an ID unknown to
// both seed and overlay is a graceful no-op.
func RecordPlaybookOutcome(root, skill, playbookID string, success bool) error {
	skillID := skillAssetID(skill)
	dir := filepath.Join(root, skillID, "assets")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "remediation-playbooks.json")
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	env := struct {
		Playbooks []map[string]any `json:"playbooks"`
	}{Playbooks: []map[string]any{}}
	if raw, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(raw, &env) != nil {
			return nil // invalid overlay → no-op (contract)
		}
	}
	var target map[string]any
	for _, pb := range env.Playbooks {
		if id, _ := pb["id"].(string); id == playbookID {
			target = pb
			break
		}
	}
	if target == nil {
		if !seedKnowsPlaybook(dir, playbookID) {
			return nil // unknown ID → graceful no-op
		}
		target = map[string]any{"id": playbookID}
		env.Playbooks = append(env.Playbooks, target)
	}
	md, _ := target["metadata"].(map[string]any)
	if md == nil {
		md = map[string]any{}
		target["metadata"] = md
	}
	rate := toFloat(md["success_rate"])
	if success {
		rate = rate*0.9 + 0.1
	} else {
		rate *= 0.9
	}
	md["success_rate"] = min(rate, 1.0)
	md["last_updated"] = NowISO()
	return writePlaybookOverlay(path, map[string]any{
		"$schema":   "remediation-playbooks/v1",
		"skill_id":  skillID,
		"playbooks": env.Playbooks,
	})
}

func writePlaybookOverlay(path string, value any) error {
	buf, err := marshal(value)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".remediation-playbooks-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(buf); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// seedKnowsPlaybook reports whether the tracked seed file lists playbookID.
func seedKnowsPlaybook(dir, playbookID string) bool {
	raw, err := os.ReadFile(filepath.Join(dir, "remediation-playbooks.seed.json"))
	if err != nil {
		return false
	}
	var env struct {
		Playbooks []struct {
			ID string `json:"id"`
		} `json:"playbooks"`
	}
	if json.Unmarshal(raw, &env) != nil {
		return false
	}
	for _, pb := range env.Playbooks {
		if pb.ID == playbookID {
			return true
		}
	}
	return false
}
