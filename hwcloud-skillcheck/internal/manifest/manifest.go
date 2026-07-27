package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/yaml"
)

// Manifest is the capability manifest for a skill.
type Manifest struct {
	SchemaVersion       string       `json:"schema_version"`
	Skill               string       `json:"skill"`
	Name                string       `json:"name"`
	Description         string       `json:"description"`
	Version             string       `json:"version,omitempty"`
	Inputs              []InputSpec  `json:"inputs"`
	Outputs             []OutputSpec `json:"outputs"`
	SideEffectClass     string       `json:"side_effect_class"`
	RequiredPermissions []string     `json:"required_permissions"`
	TelemetryEmitted    []string     `json:"telemetry_emitted"`
	Maturity            Maturity     `json:"maturity"`
}

// InputSpec describes a skill input parameter.
type InputSpec struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Required  bool   `json:"required"`
	Sensitive bool   `json:"sensitive"`
}

// OutputSpec describes a skill output.
type OutputSpec struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Maturity holds maturity signals for a skill.
type Maturity struct {
	GoldenScenarios    int  `json:"golden_scenarios"`
	TE7Pass            bool `json:"te7_pass"`
	ManifestComplete   bool `json:"manifest_complete"`
	TelemetryLaneClean bool `json:"telemetry_lane_clean"`
}

// Generate walks every huaweicloud-*-ops directory under root and writes a
// capability_manifest.json for each into out.
func Generate(root, out string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read root %q: %w", root, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "huaweicloud-") || !strings.HasSuffix(name, "-ops") {
			continue
		}
		skillDir := filepath.Join(root, name)
		m, err := generateOne(skillDir, name)
		if err != nil {
			return fmt.Errorf("generate %q: %w", name, err)
		}
		if err := writeManifest(out, name, m); err != nil {
			return fmt.Errorf("write %q: %w", name, err)
		}
	}
	return nil
}

// generateOne reads SKILL.md from skillDir, extracts frontmatter, and returns
// a Manifest.
func generateOne(skillDir, skillName string) (Manifest, error) {
	data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		return Manifest{}, fmt.Errorf("read SKILL.md: %w", err)
	}
	fm, err := yaml.ExtractFrontmatter([]byte(data))
	if err != nil {
		return Manifest{}, fmt.Errorf("extract frontmatter: %w", err)
	}
	return Manifest{
		SchemaVersion:       "1.0",
		Skill:               skillName,
		Name:                str(fm, "name"),
		Description:         str(fm, "description"),
		Version:             str(fm, "version"),
		Inputs:              nil,
		Outputs:             nil,
		SideEffectClass:     "destructive",
		RequiredPermissions: nil,
		TelemetryEmitted: []string{
			"gcl.trace.iteration",
			"gcl.critic.score",
			"l4.fault.handled",
		},
		Maturity: Maturity{
			GoldenScenarios:    0,
			TE7Pass:            false,
			ManifestComplete:   false,
			TelemetryLaneClean: false,
		},
	}, nil
}

// str safely extracts a string from the frontmatter map.
func str(fm map[string]any, k string) string {
	if v, ok := fm[k].(string); ok {
		return v
	}
	return ""
}

// writeManifest creates {out}/{skill}/ and writes the JSON manifest.
func writeManifest(out, skill string, m Manifest) error {
	dir := filepath.Join(out, skill)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "capability_manifest.json"), data, 0644)
}
