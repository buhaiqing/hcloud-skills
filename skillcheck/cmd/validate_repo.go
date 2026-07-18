package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/embed"
	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/schema"
	"github.com/buhaiqing/hcloud-skills/skillcheck/internal/yaml"
)

// cliApplicability is the set of legal cli_applicability values, mirroring
// scripts/validate_skills_frontmatter.CLI_APPLICABILITY.
var cliApplicability = map[string]bool{
	"dual-path": true,
	"cli-first": true,
	"cli-only":  true,
	"sdk-only":  true,
}

// optionalNoCLI lists skills permitted to omit cli_applicability, mirroring
// scripts/validate_skills_frontmatter.OPTIONAL_NO_CLI.
var optionalNoCLI = map[string]bool{
	"huaweicloud-billing-ops":     true,
	"huaweicloud-skill-generator": true,
}

// --- validate frontmatter ---

// validateSkillFrontmatter validates the YAML frontmatter of a SKILL.md
// document against the contract in scripts/validate_skills_frontmatter.py.
// skillDir is the containing directory name (used to verify the name field).
// It returns a list of human-readable errors (empty when valid).
func validateSkillFrontmatter(content []byte, skillDir string) []string {
	fm, err := yaml.ExtractFrontmatter(content)
	if err != nil {
		return []string{fmt.Sprintf("%s: missing or invalid YAML frontmatter (%v)", skillDir, err)}
	}

	var errs []string
	name, _ := fm["name"].(string)
	if name == "" || !strings.HasPrefix(name, "huaweicloud-") {
		errs = append(errs, fmt.Sprintf("%s: missing or invalid 'name' (must start with huaweicloud-)", skillDir))
	} else if name != skillDir {
		errs = append(errs, fmt.Sprintf("%s: name %q does not match directory %q", skillDir, name, skillDir))
	}

	for _, key := range []string{"description", "compatibility", "license"} {
		if _, ok := fm[key]; !ok {
			errs = append(errs, fmt.Sprintf("%s: missing '%s'", skillDir, key))
		}
	}

	meta, ok := fm["metadata"].(map[string]any)
	if !ok {
		errs = append(errs, fmt.Sprintf("%s: missing 'metadata'", skillDir))
		return errs
	}
	if _, ok := meta["version"]; !ok {
		errs = append(errs, fmt.Sprintf("%s: missing metadata.version", skillDir))
	}
	if _, ok := meta["last_updated"]; !ok {
		errs = append(errs, fmt.Sprintf("%s: missing metadata.last_updated", skillDir))
	}

	cli, _ := meta["cli_applicability"].(string)
	if cli == "" {
		if top, ok := fm["cli_applicability"].(string); ok {
			cli = top
		}
	}
	if cli != "" {
		if !cliApplicability[cli] {
			errs = append(errs, fmt.Sprintf("%s: invalid cli_applicability %q", skillDir, cli))
		}
	} else if !optionalNoCLI[skillDir] {
		errs = append(errs, fmt.Sprintf("%s: missing metadata.cli_applicability", skillDir))
	}

	return errs
}

// runValidateFrontmatter handles:
//
//	skillcheck validate frontmatter --root <dir>
//
// It walks <dir>/huaweicloud-*-ops/*/SKILL.md and validates each frontmatter.
// Exit 0 = all valid, 1 = any invalid (or no SKILL.md found).
func runValidateFrontmatter(args []string) error {
	fs := newFlagSet("validate frontmatter")
	root := fs.String("root", ".", "skill repository root (default: current directory)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	skills, err := discoverSkills(rootDir)
	if err != nil {
		return err
	}
	if len(skills) == 0 {
		return fmt.Errorf("validate frontmatter: no huaweicloud-*-ops/*/SKILL.md found under %s", rootDir)
	}

	var all []string
	okCount := 0
	for _, path := range skills {
		content, err := os.ReadFile(path)
		if err != nil {
			all = append(all, fmt.Sprintf("%s: read error: %v", path, err))
			continue
		}
		skillDir := skillNameFromPath(path)
		errs := validateSkillFrontmatter(content, skillDir)
		if len(errs) == 0 {
			okCount++
			continue
		}
		all = append(all, errs...)
	}

	if len(all) > 0 {
		for _, e := range all {
			fmt.Fprintln(os.Stderr, "FAIL:", e)
		}
		return fmt.Errorf("validate frontmatter: %d error(s) across %d skill(s)", len(all), len(skills))
	}
	fmt.Printf("OK: %d SKILL.md frontmatter file(s) validated\n", okCount)
	return nil
}

// discoverSkills returns all huaweicloud-*/SKILL.md paths under root, sorted
// for deterministic reporting. Mirrors validate_skills_frontmatter.SKILL_GLOB.
func discoverSkills(root string) ([]string, error) {
	pattern := filepath.Join(root, "huaweicloud-*", "SKILL.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

// --- validate eval-queries ---

// runValidateEvalQueries handles:
//
//	skillcheck validate eval-queries --root <dir>
//
// It walks <dir>/huaweicloud-*-ops/*/assets/eval_queries.json and validates
// each against the embedded eval-queries union schema, mirroring
// scripts/validate_eval_queries_schema.py (format auto-detection plus the
// skill-name consistency check for matchArrayEntry / structuredObject).
func runValidateEvalQueries(args []string) error {
	fs := newFlagSet("validate eval-queries")
	root := fs.String("root", ".", "skill repository root (default: current directory)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	files, err := discoverEvalQueries(rootDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("validate eval-queries: no huaweicloud-*-ops/*/assets/eval_queries.json found under %s", rootDir)
	}

	var all []string
	okCount := 0
	for _, path := range files {
		skillName := filepath.Base(filepath.Dir(filepath.Dir(path)))
		content, err := os.ReadFile(path)
		if err != nil {
			all = append(all, fmt.Sprintf("%s: read error: %v", path, err))
			continue
		}
		errs := validateEvalQueriesFile(content, embed.EvalQueriesSchema, skillName)
		if len(errs) == 0 {
			okCount++
			continue
		}
		for _, e := range errs {
			all = append(all, fmt.Sprintf("%s: %s", path, e))
		}
	}

	if len(all) > 0 {
		for _, e := range all {
			fmt.Fprintln(os.Stderr, "FAIL:", e)
		}
		return fmt.Errorf("validate eval-queries: %d error(s) across %d file(s)", len(all), len(files))
	}
	fmt.Printf("OK: %d eval_queries.json file(s) validated\n", okCount)
	return nil
}

func discoverEvalQueries(root string) ([]string, error) {
	pattern := filepath.Join(root, "huaweicloud-*", "assets", "eval_queries.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

// validateEvalQueriesFile validates a single eval_queries.json byte slice,
// dispatching on its format and enforcing skill-name consistency with the
// owning skill directory.
func validateEvalQueriesFile(content, schemaData []byte, skillName string) []string {
	def, parsed, err := detectEvalFormat(content, schemaData)
	if err != nil {
		return []string{err.Error()}
	}

	// Per-element validation for the array format.
	if arr, ok := parsed.([]any); ok {
		var all []string
		for i, item := range arr {
			itemBytes, mErr := marshalJSON(item)
			if mErr != nil {
				all = append(all, fmt.Sprintf("$[%d]: %v", i, mErr))
				continue
			}
			errs, vErr := schema.ValidateDef(schemaData, def, itemBytes)
			if vErr != nil {
				all = append(all, vErr.Error())
				continue
			}
			for _, e := range errs {
				all = append(all, fmt.Sprintf("$[%d]%s", i, e))
			}
			if def == "matchArrayEntry" {
				if m, ok := item.(map[string]any); ok {
					if declared, _ := m["skill"].(string); declared != "" && declared != skillName {
						all = append(all, fmt.Sprintf("$[%d].skill: expected %q, got %q", i, skillName, declared))
					}
				}
			}
		}
		return all
	}

	// Object format: validate the whole document against its def.
	errs, vErr := schema.ValidateDef(schemaData, def, content)
	if vErr != nil {
		return []string{vErr.Error()}
	}
	if def == "structuredObject" {
		if m, ok := parsed.(map[string]any); ok {
			if declared, _ := m["skill_name"].(string); declared != skillName {
				errs = append(errs, fmt.Sprintf(".skill_name: expected %q, got %q", skillName, declared))
			}
		}
	}
	return errs
}

// skillNameFromPath derives the skill directory name from a discovered file
// path (parent directory of the file), e.g.
// root/huaweicloud-ecs-ops/SKILL.md -> huaweicloud-ecs-ops.
func skillNameFromPath(path string) string {
	return filepath.Base(filepath.Dir(path))
}

// --- validate product-assessment ---

// runValidateProductAssessment handles:
//
//	skillcheck validate product-assessment --root <dir>
//
// It walks <dir>/huaweicloud-*-ops/*/references/well-architected-assessment.md
// and validates the Worker Output Contract JSON example(s), mirroring the core
// contract in scripts/validate_product_assessment.py (top-level required
// fields, status enum, pillar structure, skill_id consistency). See the test
// comments for the exact alignment scope.
func runValidateProductAssessment(args []string) error {
	fs := newFlagSet("validate product-assessment")
	root := fs.String("root", ".", "skill repository root (default: current directory)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	files, err := discoverAssessment(rootDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("validate product-assessment: no huaweicloud-*-ops/*/references/well-architected-assessment.md found under %s", rootDir)
	}

	var all []string
	okCount := 0
	for _, path := range files {
		skillName := filepath.Base(filepath.Dir(filepath.Dir(path)))
		content, err := os.ReadFile(path)
		if err != nil {
			all = append(all, fmt.Sprintf("%s: read error: %v", path, err))
			continue
		}
		errs := validateProductAssessment(content, skillName)
		if len(errs) == 0 {
			okCount++
			continue
		}
		for _, e := range errs {
			all = append(all, fmt.Sprintf("%s: %s", path, e))
		}
	}

	if len(all) > 0 {
		for _, e := range all {
			fmt.Fprintln(os.Stderr, "FAIL:", e)
		}
		return fmt.Errorf("validate product-assessment: %d error(s) across %d file(s)", len(all), len(files))
	}
	fmt.Printf("OK: %d well-architected-assessment.md file(s) validated\n", okCount)
	return nil
}

func discoverAssessment(root string) ([]string, error) {
	pattern := filepath.Join(root, "huaweicloud-*", "references", "well-architected-assessment.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}
