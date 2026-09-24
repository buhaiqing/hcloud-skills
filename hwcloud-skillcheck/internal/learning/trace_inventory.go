package learning

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// TraceInventory counts every trace file by the frozen classification order
// (parse → schema-invalid → smoke → evidence).
//
// It exists because `trend report` is silent about what it dropped: when the
// north-star metric reads 0 traces, the operator cannot tell an empty campaign
// history apart from 117 legacy files that predate the writer fix. That
// ambiguity is how a dead metric stays trusted for months.
type TraceInventory struct {
	TotalFiles    int `json:"total_files"`
	Evidence      int `json:"evidence"`
	Smoke         int `json:"smoke"`
	SchemaInvalid int `json:"schema_invalid"`
	Unparsable    int `json:"unparsable"`
	// ByFamily counts files per trace family ("gcl" / "orchestrator"), so a
	// single-family writer regression is visible.
	ByFamily map[string]int `json:"by_family"`
	// Attributable counts evidence traces whose skill attribution is not the
	// literal "unknown" — the other reason trend findings stay empty.
	Attributable int `json:"attributable"`
}

func traceFamily(name string) string {
	if strings.HasPrefix(name, "orchestrator-trace-") {
		return "orchestrator"
	}
	if strings.HasPrefix(name, "gcl-trace-") {
		return "gcl"
	}
	return "other"
}

// ComputeTraceInventory classifies every trace file under
// <root>/audit-results. A missing directory is an empty inventory, not an
// error: runtime state may legitimately not exist yet.
func ComputeTraceInventory(root string) (*TraceInventory, error) {
	inv := &TraceInventory{ByFamily: map[string]int{}}
	tracesDir := filepath.Join(root, "audit-results")
	entries, err := os.ReadDir(tracesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return inv, nil
		}
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() || !IsTraceFileName(e.Name()) {
			continue
		}
		inv.TotalFiles++
		inv.ByFamily[traceFamily(e.Name())]++

		raw, readErr := os.ReadFile(filepath.Join(tracesDir, e.Name()))
		if readErr != nil {
			inv.Unparsable++
			continue
		}
		var trace map[string]any
		if uErr := json.Unmarshal(raw, &trace); uErr != nil {
			inv.Unparsable++
			continue
		}
		switch class, _ := ClassifyTrace(raw, trace); class {
		case TraceEvidence:
			inv.Evidence++
			if skill, _ := trace["skill"].(string); skill != "" && skill != "unknown" {
				inv.Attributable++
			}
		case TraceSmoke:
			inv.Smoke++
		default:
			inv.SchemaInvalid++
		}
	}
	return inv, nil
}
