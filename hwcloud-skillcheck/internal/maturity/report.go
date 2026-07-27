package maturity

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Report struct {
	PerSkill map[string]float64
}

func Rollup(root string) (*Report, error) {
	r := &Report{PerSkill: map[string]float64{}}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		mf, err := os.ReadFile(filepath.Join(root, e.Name(), "capability_manifest.json"))
		if err != nil {
			continue
		}
		var m struct {
			Maturity struct {
				GoldenScenarios    int  `json:"golden_scenarios"`
				TE7Pass            bool `json:"te7_pass"`
				ManifestComplete   bool `json:"manifest_complete"`
				TelemetryLaneClean bool `json:"telemetry_lane_clean"`
			} `json:"maturity"`
		}
		_ = json.Unmarshal(mf, &m)
		sc := score(m.Maturity)
		r.PerSkill[e.Name()] = sc
	}
	return r, nil
}

func score(m struct {
	GoldenScenarios    int  `json:"golden_scenarios"`
	TE7Pass            bool `json:"te7_pass"`
	ManifestComplete   bool `json:"manifest_complete"`
	TelemetryLaneClean bool `json:"telemetry_lane_clean"`
}) float64 {
	gr := 0.0
	if m.GoldenScenarios >= 5 {
		gr = 1.0
	}
	b := func(b bool) float64 {
		if b {
			return 1.0
		}
		return 0.0
	}
	return 0.3*gr + 0.3*b(m.TE7Pass) + 0.2*b(m.ManifestComplete) + 0.2*b(m.TelemetryLaneClean)
}
