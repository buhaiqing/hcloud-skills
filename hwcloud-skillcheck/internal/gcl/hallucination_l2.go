package gcl

import (
	"encoding/json"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/buhaiqing/hcloud-skills/hwcloud-skillcheck/internal/schema"
)

// L2Status classifies how the L2 (OpenAPI schema conformance) check concluded.
// L2Result (hallucination.go) has no status field, so the outcome is encoded in
// Details by setDetails and recovered by Status. Without it, "the skill ships
// no schema, so nothing was checked" was indistinguishable from "the output
// conformed" — the silent-no-op that this status exists to prevent.
type L2Status string

const (
	// L2StatusPass: the skill ships references/openapi-schema.json and the
	// Generator output conforms to it.
	L2StatusPass L2Status = "pass"
	// L2StatusViolation: the output violates the skill's OpenAPI schema, so
	// the result blocks the run.
	L2StatusViolation L2Status = "violation"
	// L2StatusSkippedNoSchema: the skill ships no references/openapi-schema.json,
	// so no schema check could run. This is a capability gap, not a pass.
	L2StatusSkippedNoSchema L2Status = "skipped_no_schema"
	// L2StatusSkippedNoOutput: there was no Generator output to validate.
	L2StatusSkippedNoOutput L2Status = "skipped_no_output"
	// L2StatusInvalidOutput: the Generator excerpt is not JSON, so no schema
	// comparison was possible.
	L2StatusInvalidOutput L2Status = "invalid_output"
	// L2StatusSchemaUnreadable: the schema file exists but could not be read.
	L2StatusSchemaUnreadable L2Status = "schema_unreadable"
	// L2StatusValidatorError: the schema itself could not be parsed or resolved.
	L2StatusValidatorError L2Status = "validator_error"
)

// l2SchemaRelPath is the per-skill asset the L2 check validates against.
const l2SchemaRelPath = "references/openapi-schema.json"

// l2StatusSep separates the machine-readable status token from the
// human-readable detail inside L2Result.Details.
const l2StatusSep = ": "

// setDetails records the outcome and its human-readable explanation together,
// so Status, Outcome, and Details cannot disagree.
func (r *L2Result) setDetails(status L2Status, detail string) {
	r.Outcome = string(status)
	r.Details = string(status) + l2StatusSep + detail
}

// Status reports how the L2 check concluded: one of the L2Status values, or ""
// when no L2 check has run. Outcome is the persisted form; Details is parsed as
// a fallback for traces written before Outcome existed.
func (r *L2Result) Status() L2Status {
	if r == nil {
		return ""
	}
	if r.Blocked || len(r.Errors) > 0 {
		return L2StatusViolation
	}
	if r.Outcome != "" {
		return L2Status(r.Outcome)
	}
	token, _, ok := strings.Cut(r.Details, l2StatusSep)
	if !ok {
		return ""
	}
	return L2Status(token)
}

// Skipped reports whether L2 produced no verdict because it had nothing to
// work with: the skill ships no OpenAPI schema, or there was no output to
// validate.
func (r *L2Result) Skipped() bool {
	switch r.Status() {
	case L2StatusSkippedNoSchema, L2StatusSkippedNoOutput:
		return true
	default:
		return false
	}
}

// l2SkipWarned remembers skill roots already warned about a missing OpenAPI
// schema. The L2 check runs once per GCL iteration, so an undeduped warning
// would repeat on every retry; keying on the skill root warns once per skill.
var l2SkipWarned sync.Map

// warnL2SkippedNoSchema logs, at most once per skill root, that the L2 check
// cannot run because the skill ships no references/openapi-schema.json.
func warnL2SkippedNoSchema(skillRoot string) {
	if _, seen := l2SkipWarned.LoadOrStore(skillRoot, struct{}{}); seen {
		return
	}
	slog.Warn("L2 hallucination check skipped: missing "+l2SchemaRelPath,
		"skill", filepath.Base(skillRoot),
		"skill_root", skillRoot,
		"impact", "generator output is not checked against an OpenAPI schema",
		"fix", "add "+l2SchemaRelPath+" to the skill (see huaweicloud-skill-generator/references/openapi-schema-asset.md)")
}

// checkJSONStructure validates the Generator output against the skill's
// OpenAPI schema (if present). The returned L2Result records why the check
// concluded as it did: (*L2Result).Status reports pass, violation, or one of
// the skipped/error outcomes.
func (d *DefaultHallucinationDetector) checkJSONStructure(resultExcerpt string) (*L2Result, error) {
	result := &L2Result{}

	if resultExcerpt == "" {
		result.setDetails(L2StatusSkippedNoOutput, "no output to validate")
		return result, nil
	}

	// Try to parse as JSON.
	var parsed any
	if err := json.Unmarshal([]byte(resultExcerpt), &parsed); err != nil {
		result.setDetails(L2StatusInvalidOutput, "output is not JSON: "+err.Error())
		return result, nil
	}

	// Load the skill's OpenAPI schema if present.
	schemaPath := filepath.Join(d.SkillRoot, "references", "openapi-schema.json")
	schemaData, err := readFileBytes(schemaPath)
	if err != nil {
		result.setDetails(L2StatusSchemaUnreadable, "could not read openapi-schema.json: "+err.Error())
		return result, nil
	}
	if schemaData == nil {
		warnL2SkippedNoSchema(d.SkillRoot)
		result.setDetails(L2StatusSkippedNoSchema, "no openapi-schema.json found; skipping L2")
		return result, nil
	}

	// Validate against schema.
	if verrs, verr := schema.ValidateFile([]byte(resultExcerpt), schemaData); verr != nil {
		result.setDetails(L2StatusValidatorError, "schema validation error: "+verr.Error())
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
		result.setDetails(L2StatusViolation, strconv.Itoa(len(l2errs))+" schema violations")
		return result, nil
	}

	result.setDetails(L2StatusPass, "schema valid")
	return result, nil
}
