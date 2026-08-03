package gcl

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/schema"
)

// checkJSONStructure validates the Generator output against the skill's
// OpenAPI schema (if present).
func (d *DefaultHallucinationDetector) checkJSONStructure(resultExcerpt string) (*L2Result, error) {
	result := &L2Result{}

	if resultExcerpt == "" {
		result.Details = "no output to validate"
		return result, nil
	}

	// Try to parse as JSON.
	var parsed any
	if err := json.Unmarshal([]byte(resultExcerpt), &parsed); err != nil {
		result.Details = "output is not JSON: " + err.Error()
		return result, nil
	}

	// Load the skill's OpenAPI schema if present.
	schemaPath := filepath.Join(d.SkillRoot, "references", "openapi-schema.json")
	schemaData, err := readFileBytes(schemaPath)
	if err != nil {
		result.Details = "could not read openapi-schema.json: " + err.Error()
		return result, nil
	}
	if schemaData == nil {
		result.Details = "no openapi-schema.json found; skipping L2"
		return result, nil
	}

	// Validate against schema.
	if verrs, verr := schema.ValidateFile([]byte(resultExcerpt), schemaData); verr != nil {
		result.Details = "schema validation error: " + verr.Error()
		return result, nil
	} else if len(verrs) > 0 {
		var l2errs []L2Err
		for _, e := range verrs {
			parts := strings.SplitN(e, ":", 3)
			path := ""
			msg := e
			if len(parts) >= 2 {
				path = strings.TrimSpace(parts[0])
				msg = strings.TrimSpace(parts[1])
				if len(parts) == 3 {
					msg += ":" + parts[2]
				}
			}
			l2errs = append(l2errs, L2Err{
				Path:    path,
				Message: msg,
			})
		}
		result.Errors = l2errs
		result.Blocked = true
		result.Details = strconv.Itoa(len(l2errs)) + " schema violations"
		return result, nil
	}

	result.Details = "schema valid"
	return result, nil
}
