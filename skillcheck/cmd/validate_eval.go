package cmd

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// evalFormat detection mirrors validate_eval_queries_schema.py. For an array
// document it returns the entry $def name; for an object document it returns
// the object $def name; otherwise an error.
func detectEvalFormat(content, schemaData []byte) (string, any, error) {
	dec := json.NewDecoder(strings.NewReader(string(content)))
	dec.UseNumber()
	var parsed any
	if err := dec.Decode(&parsed); err != nil {
		return "", nil, fmt.Errorf("parse instance: %w", err)
	}
	switch v := parsed.(type) {
	case []any:
		if len(v) == 0 {
			return "", nil, fmt.Errorf("$: expected non-empty array")
		}
		first, ok := v[0].(map[string]any)
		if !ok {
			return "", nil, fmt.Errorf("$: every array item must be an object")
		}
		def := evalArrayDefFor(first)
		if def == "" {
			return "", nil, fmt.Errorf("$: unrecognized array entry shape")
		}
		return def, v, nil
	case map[string]any:
		def := evalObjectDefFor(v)
		if def == "" {
			return "", nil, fmt.Errorf("$: unrecognized object shape; expected evaluation_queries, should_match, or should_trigger")
		}
		return def, v, nil
	default:
		return "", nil, fmt.Errorf("$: expected array or object")
	}
}

var (
	productAssessmentBlockRe = regexp.MustCompile("(?s)```json\\s*\\n(\\{.*?\\})\\n```")
	productPillars           = map[string]bool{"reliability": true, "security": true, "cost": true, "efficiency": true}
	productStatuses          = map[string]bool{"OK": true, "PARTIAL": true, "ERROR": true}
	pillarStatuses           = map[string]bool{"assessed": true, "not_assessed": true, "skipped": true}
)

// requiredAssessmentTop lists the top-level keys required by the Worker Output
// Contract (scripts/validate_product_assessment.py:REQUIRED_TOP).
var requiredAssessmentTop = []string{
	"skill_id", "product", "region", "scope", "assessment_date", "status",
	"partial", "resource_count", "pillars", "recommendations", "trace", "errors",
}

// validateProductAssessment validates the Worker Output Contract JSON example
// embedded in a well-architected-assessment.md document. It mirrors the core of
// scripts/validate_product_assessment.py:validate_assessment plus the
// skill_id==skillName consistency enforced in validate_file.
//
// Alignment scope (documented for reviewers):
//   - Covered: "Worker Output Contract" presence; JSON parse; required top
//     fields; status enum; pillars object + per-pillar status enum; skill_id
//     == skillName consistency.
//   - Not covered (deliberate, self-contained): the PRODUCT_BY_SKILL registry
//     cross-check (external to the binary) and the deep finding/recommendation
//     sub-field validation. Those are additive and do not affect the contract
//     shape validated here.
func validateProductAssessment(content []byte, skillName string) []string {
	text := string(content)
	if !strings.Contains(text, "Worker Output Contract") {
		return []string{"missing 'Worker Output Contract' section"}
	}
	blocks := productAssessmentBlockRe.FindAllSubmatch(content, -1)
	if len(blocks) == 0 {
		return []string{"no product_assessment JSON example found"}
	}

	var all []string
	for _, m := range blocks {
		raw := string(m[1])
		if !strings.Contains(raw, `"product"`) || !strings.Contains(raw, `"pillars"`) {
			continue // Python skips blocks lacking product/pillars markers
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			all = append(all, fmt.Sprintf("JSON parse error: %v", err))
			continue
		}
		all = append(all, validateAssessmentObject(data, skillName)...)
	}
	return all
}

func validateAssessmentObject(data map[string]any, skillName string) []string {
	var errs []string

	for _, key := range requiredAssessmentTop {
		if _, ok := data[key]; !ok {
			errs = append(errs, fmt.Sprintf("missing top-level field %q", key))
		}
	}

	if status, _ := data["status"].(string); status != "" && !productStatuses[status] {
		errs = append(errs, fmt.Sprintf("invalid status %q", status))
	}
	if product, _ := data["product"].(string); product == "" {
		errs = append(errs, "product must be non-empty string")
	}
	if skillID, _ := data["skill_id"].(string); skillID != "" && skillID != skillName {
		errs = append(errs, fmt.Sprintf("skill_id %q does not match skill directory %q", skillID, skillName))
	}

	pillars, ok := data["pillars"].(map[string]any)
	if !ok {
		errs = append(errs, "pillars must be an object")
		return errs
	}
	for pkey, pval := range pillars {
		if !productPillars[pkey] {
			errs = append(errs, fmt.Sprintf("unknown pillar key %q", pkey))
			continue
		}
		pobj, ok := pval.(map[string]any)
		if !ok {
			errs = append(errs, fmt.Sprintf("pillars.%s must be an object", pkey))
			continue
		}
		if pstatus, _ := pobj["status"].(string); pstatus != "" && !pillarStatuses[pstatus] {
			errs = append(errs, fmt.Sprintf("pillars.%s.status invalid %q", pkey, pstatus))
		}
	}

	// trace.commands must not carry an unmasked secret reference.
	if trace, ok := data["trace"].(map[string]any); ok {
		if cmds, ok := trace["commands"].([]any); ok {
			for _, c := range cmds {
				if s, ok := c.(string); ok && strings.Contains(strings.ToUpper(s), "SECRET") && !strings.Contains(s, "<masked>") {
					errs = append(errs, "trace.commands contains unmasked secret reference")
					break
				}
			}
		}
	}

	return errs
}
