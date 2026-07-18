package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// GCL_SKILLS is the set of skills that must provide GCL artifacts.
var GCL_SKILLS = map[string]bool{
	"huaweicloud-billing-ops":       true,
	"huaweicloud-cbr-ops":           true,
	"huaweicloud-cce-ops":           true,
	"huaweicloud-ces-ops":           true,
	"huaweicloud-css-ops":           true,
	"huaweicloud-cts-ops":           true,
	"huaweicloud-cdn-ops":           true,
	"huaweicloud-dcs-ops":           true,
	"huaweicloud-dms-ops":           true,
	"huaweicloud-dns-ops":           true,
	"huaweicloud-ecs-ops":           true,
	"huaweicloud-eip-ops":           true,
	"huaweicloud-elb-ops":           true,
	"huaweicloud-functiongraph-ops": true,
	"huaweicloud-gaussdb-ops":       true,
	"huaweicloud-hss-ops":           true,
	"huaweicloud-kms-ops":           true,
	"huaweicloud-iam-ops":           true,
	"huaweicloud-lts-ops":           true,
	"huaweicloud-obs-ops":           true,
	"huaweicloud-rds-ops":           true,
	"huaweicloud-swr-ops":           true,
	"huaweicloud-vpc-ops":           true,
	"huaweicloud-waf-ops":           true,
}

// runValidateGCL dispatches `skillcheck validate gcl-conformance` and
// `skillcheck validate alarm-wire-contract`.
func runValidateGCL(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("validate: missing subcommand (gcl-conformance|alarm-wire-contract)")
	}
	switch args[0] {
	case "gcl-conformance":
		return runValidateGCLConformance(args[1:])
	case "alarm-wire-contract":
		return runValidateAlarmWireContract(args[1:])
	case "-h", "--help", "help":
		fmt.Fprintln(os.Stdout, "skillcheck validate <gcl-conformance|alarm-wire-contract> --root <dir>")
		return nil
	default:
		return fmt.Errorf("validate: unknown subcommand %q", args[0])
	}
}

// ---------------------------------------------------------------------------
// validate gcl-conformance
// ---------------------------------------------------------------------------

type gclConformanceResult struct {
	Skill                       string `json:"skill"`
	RubricSections              int    `json:"rubric_sections"`
	PromptSections              int    `json:"prompt_sections"`
	HasQualityGate              bool   `json:"has_quality_gate"`
	PromptHasOperationIntent    bool   `json:"prompt_has_operation_intent"`
	PromptHasNoBarePlaceholders bool   `json:"prompt_has_no_bare_placeholders"`
	RubricOK                    bool   `json:"rubric_ok"`
	PromptOK                    bool   `json:"prompt_ok"`
	SkillOK                     bool   `json:"skill_ok"`
	OK                          bool   `json:"ok"`
}

var rubricSectionRe = regexp.MustCompile(`(?m)^## (\d+)\. `)
var qualityGateRe = regexp.MustCompile(`(?m)^## Quality Gate \(GCL\)$`)

// barePlaceholderRe matches {word} placeholders.
// {{word}} escaped ones are stripped before matching.
var barePlaceholderRe = regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_.-]*\}`)

// doubleBraceRe matches {{...}} escaped placeholders to strip before detection.
var doubleBraceRe = regexp.MustCompile(`\{\{[^}]+\}\}`)

// codeBlockRe removes fenced code blocks so bare placeholder detection
// does not fire on examples.
var codeBlockRe = regexp.MustCompile("```[\\s\\S]*?```")

func countNumberedSections(text string, target int) int {
	for number := 1; number <= target; number++ {
		pattern := fmt.Sprintf(`(?m)^## %d\. `, number)
		if !regexp.MustCompile(pattern).MatchString(text) {
			return 0
		}
	}
	return target
}

func hasBarePlaceholders(text string) bool {
	// Strip fenced code blocks and escaped {{...}} placeholders first
	text = codeBlockRe.ReplaceAllString(text, "")
	text = doubleBraceRe.ReplaceAllString(text, "")
	// Strip inline comments so {placeholders} in comments don't trigger false positives
	text = stripComments(text)
	return barePlaceholderRe.MatchString(text)
}

func checkGCLSkillConformance(root, skill string) gclConformanceResult {
	skillDir := filepath.Join(root, skill)
	rubricPath := filepath.Join(skillDir, "references", "rubric.md")
	promptPath := filepath.Join(skillDir, "references", "prompt-templates.md")
	skillPath := filepath.Join(skillDir, "SKILL.md")

	rubricText := readFileString(rubricPath)
	promptText := readFileString(promptPath)
	skillText := readFileString(skillPath)

	rubricSections := 0
	if rubricText != "" {
		rubricSections = countNumberedSections(rubricText, 8)
	}
	promptSections := 0
	if promptText != "" {
		promptSections = countNumberedSections(promptText, 7)
	}
	hasQualityGate := qualityGateRe.MatchString(skillText)
	promptHasOpIntent := strings.Contains(promptText, "{{output.operation_intent}}") || strings.Contains(promptText, "operation_intent")
	promptNoBare := !hasBarePlaceholders(promptText)

	rubricOK := rubricSections == 8
	promptOK := promptSections == 7 && promptHasOpIntent && promptNoBare
	skillOK := hasQualityGate

	return gclConformanceResult{
		Skill:                       skill,
		RubricSections:              rubricSections,
		PromptSections:              promptSections,
		HasQualityGate:              hasQualityGate,
		PromptHasOperationIntent:    promptHasOpIntent,
		PromptHasNoBarePlaceholders: promptNoBare,
		RubricOK:                    rubricOK,
		PromptOK:                    promptOK,
		SkillOK:                     skillOK,
		OK:                          rubricOK && promptOK && skillOK,
	}
}

func readFileString(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func runValidateGCLConformance(args []string) error {
	fs := newFlagSet("skillcheck validate gcl-conformance")
	root := fs.String("root", ".", "skill repository root")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	var skills []string
	for skill := range GCL_SKILLS {
		if dirExists(filepath.Join(rootDir, skill)) {
			skills = append(skills, skill)
		}
	}
	sort.Strings(skills)

	var reports []gclConformanceResult
	for _, skill := range skills {
		reports = append(reports, checkGCLSkillConformance(rootDir, skill))
	}

	passing := 0
	for _, r := range reports {
		if r.OK {
			passing++
		}
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]any{
			"summary": map[string]int{
				"total":   len(reports),
				"passing": passing,
				"failing": len(reports) - passing,
			},
			"reports": reports,
		})
	} else {
		fmt.Printf("GCL conformance: %d/%d skills conform.\n", passing, len(reports))
		for _, r := range reports {
			if r.OK {
				continue
			}
			var reasons []string
			if !r.RubricOK {
				reasons = append(reasons, fmt.Sprintf("rubric_sections=%d/8", r.RubricSections))
			}
			if !r.PromptOK {
				reasons = append(reasons, fmt.Sprintf("prompt_sections=%d/7", r.PromptSections))
				if !r.PromptHasOperationIntent {
					reasons = append(reasons, "missing operation_intent in prompt templates")
				}
				if !r.PromptHasNoBarePlaceholders {
					reasons = append(reasons, "bare placeholder detected")
				}
			}
			if !r.SkillOK {
				reasons = append(reasons, "missing ## Quality Gate (GCL) heading in SKILL.md")
			}
			fmt.Printf("  FAIL %s: %s\n", r.Skill, strings.Join(reasons, ", "))
		}
	}

	if passing < len(reports) {
		return fmt.Errorf("gcl-conformance: %d/%d passed", passing, len(reports))
	}
	return nil
}

// ---------------------------------------------------------------------------
// validate alarm-wire-contract
// ---------------------------------------------------------------------------

const cesSkill = "huaweicloud-ces-ops"
const cesConfigRelative = "huaweicloud-ces-ops/assets/example-config.yaml"

func runValidateAlarmWireContract(args []string) error {
	fs := newFlagSet("skillcheck validate alarm-wire-contract")
	root := fs.String("root", ".", "skill repository root")
	jsonOut := fs.Bool("json", false, "emit JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootDir, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	errors := checkAlarmWireContract(rootDir)
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]any{"ok": len(errors) == 0, "errors": errors})
	} else {
		for _, e := range errors {
			fmt.Printf("  FAIL: %s\n", e)
		}
		if len(errors) == 0 {
			fmt.Println("[alarm-wire-contract] OK")
		} else {
			fmt.Printf("[alarm-wire-contract] FAIL: %d issue(s)\n", len(errors))
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("alarm-wire-contract: %d issue(s)", len(errors))
	}
	return nil
}

func checkAlarmWireContract(root string) []string {
	var errors []string

	configPath := filepath.Join(root, cesConfigRelative)
	cfgData, err := os.ReadFile(configPath)
	if err != nil {
		return []string{fmt.Sprintf("%s: missing canonical CES gcl_quality config", configPath)}
	}

	cfgText := string(cfgData)
	gclBlock, ok := extractGCLQualityBlock(cfgText)
	if !ok {
		return []string{fmt.Sprintf("%s: no gcl_quality block found", configPath)}
	}

	// Check numeric keys match defaults
	defaults := map[string]float64{
		"pass_rate_warn":      0.85,
		"pass_rate_critical":  0.70,
		"max_iter_warn_count": 3.0,
	}
	for key, defVal := range defaults {
		if val, ok := gclBlock[key]; ok {
			if fval, ok := val.(float64); ok && abs(fval-defVal) > 1e-9 {
				errors = append(errors, fmt.Sprintf("%s: gcl_quality.%s=%.2f drifts from default %.2f", configPath, key, fval, defVal))
			}
		} else {
			errors = append(errors, fmt.Sprintf("%s: gcl_quality.%s missing", configPath, key))
		}
	}

	// safety_fail_alert must be boolean
	if val, ok := gclBlock["safety_fail_alert"]; ok {
		if _, ok := val.(bool); !ok {
			errors = append(errors, fmt.Sprintf("%s: gcl_quality.safety_fail_alert=%v is not boolean", configPath, val))
		}
	} else {
		errors = append(errors, fmt.Sprintf("%s: gcl_quality.safety_fail_alert missing", configPath))
	}

	// pass_rate_warn >= pass_rate_critical
	warn := getFloat(gclBlock, "pass_rate_warn", 0.85)
	critical := getFloat(gclBlock, "pass_rate_critical", 0.70)
	if !(0 <= critical && critical <= warn && warn <= 1) {
		errors = append(errors, fmt.Sprintf("%s: invalid pass_rate ordering critical=%.2f warn=%.2f; require 0 <= critical <= warn <= 1", configPath, critical, warn))
	}

	// max_iter_warn_count >= 1
	maxIter := getFloat(gclBlock, "max_iter_warn_count", 3)
	if maxIter < 1 {
		errors = append(errors, fmt.Sprintf("%s: max_iter_warn_count must be >= 1, got %.0f", configPath, maxIter))
	}

	// docs/gcl-spec.md must document the thresholds
	specPath := filepath.Join(root, "docs", "gcl-spec.md")
	specText := readFileString(specPath)
	if specText == "" {
		errors = append(errors, fmt.Sprintf("%s: missing", specPath))
	} else {
		for _, fragment := range []string{"pass_rate_warn", "pass_rate_critical", "max_iter_warn_count", "safety_fail_alert"} {
			if !strings.Contains(specText, fragment) {
				errors = append(errors, fmt.Sprintf("%s: missing documented threshold %q", specPath, fragment))
			}
		}
	}

	return errors
}

func extractGCLQualityBlock(text string) (map[string]any, bool) {
	lines := strings.Split(text, "\n")
	inBlock := false
	blockLines := []string{}
	for _, line := range lines {
		stripped := strings.TrimSpace(line)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		if strings.HasPrefix(stripped, "gcl_quality:") {
			inBlock = true
			continue
		}
		if inBlock {
			// Block ends when we hit a top-level key (no leading space) that's not a continuation
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && strings.Contains(stripped, ":") {
				break
			}
			blockLines = append(blockLines, line)
		}
	}

	if len(blockLines) == 0 {
		return nil, false
	}

	result := map[string]any{}
	for _, bline := range blockLines {
		stripped := strings.TrimSpace(bline)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}
		parts := strings.SplitN(stripped, ":", 2)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		raw := strings.TrimSpace(parts[1])
		// Strip inline comments
		if idx := strings.Index(raw, "#"); idx >= 0 {
			raw = strings.TrimSpace(raw[:idx])
		}
		if raw == "" {
			continue
		}
		// Try bool first
		if raw == "true" {
			result[key] = true
			continue
		}
		if raw == "false" {
			result[key] = false
			continue
		}
		// Try float
		var f float64
		_, err := fmt.Sscanf(raw, "%f", &f)
		if err == nil {
			result[key] = f
		}
	}
	return result, len(result) > 0
}

func getFloat(m map[string]any, key string, def float64) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return def
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
