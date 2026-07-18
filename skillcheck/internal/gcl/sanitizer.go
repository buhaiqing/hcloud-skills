// Package gcl provides the Generator-Critic-Loop runtime components for
// skillcheck: sanitizer (safety_class enum + resource ID masking) and
// runner (GCL loop orchestration).
//
// This file implements the L1-A sanitizer layer, ported from
// scripts/gcl_runner.py (sanitize_operation_intent, mask_resource_id,
// _enforce_safety_class_enum, _mask_resource_scope).
package gcl

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// SAFETY_CLASS_VALUES is the canonical enum for operation_intent.safety_class.
// It must stay in sync with:
//   - gcl-trace.schema.json  (operation_intent.safety_class enum)
//   - docs/gcl-spec.md       (§operation_intent)
//   - huaweicloud-skill-generator/references/gcl-prompt-backbone.md
//
// Safety class values: "read-only" | "mutating" | "destructive"
var SAFETY_CLASS_VALUES = [3]string{"read-only", "mutating", "destructive"}

// IsValidSafetyClass returns true when v is one of the canonical enum values.
func IsValidSafetyClass(v string) bool {
	for _, ok := range SAFETY_CLASS_VALUES {
		if ok == v {
			return true
		}
	}
	return false
}

// SanitizeError wraps a sanitation failure with context.
type SanitizeError struct {
	Field   string
	Value   any
	Message string
}

func (e *SanitizeError) Error() string {
	return fmt.Sprintf("operation_intent.%s=%v; %s", e.Field, e.Value, e.Message)
}

// SanitizeOperationIntent JSON-unmarshals raw and returns a sanitized copy.
//
// It applies two transformations mirroring the Python reference:
//  1. resource_scope IDs are masked via MaskResourceID.
//  2. safety_class is validated against SAFETY_CLASS_VALUES; invalid values
//     cause a *SanitizeError (fail-closed).
//
// If raw is empty or pure whitespace, SanitizeOperationIntent returns nil,
// nil (equivalent to the Python "if not raw: return None" early-exit).
//
// If raw is not valid JSON, a partial sanitized map with a "summary" key
// containing masked content is returned so the trace can still be persisted.
func SanitizeOperationIntent(raw string) (map[string]any, error) {
	if raw == "" {
		return nil, nil
	}
	var intent map[string]any
	if err := json.Unmarshal([]byte(raw), &intent); err != nil {
		// Not valid JSON — mask secrets in the raw string and return partial.
		return map[string]any{"summary": MaskSecrets([]byte(raw))}, nil
	}
	sanitized, ok := maskJSON(intent).(map[string]any)
	if !ok {
		return nil, nil
	}
	if scope, ok := sanitized["resource_scope"]; ok {
		sanitized["resource_scope"] = maskResourceScope(scope)
	}
	if err := enforceSafetyClass(sanitized); err != nil {
		return nil, err
	}
	return sanitized, nil
}

// maskJSON recursively walks a decoded JSON value and masks secret-named keys
// and embedded secret strings, mirroring _mask_json in gcl_runner.py.
func maskJSON(v any) any {
	switch val := v.(type) {
	case map[string]any:
		masked := make(map[string]any, len(val))
		for k, item := range val {
			if secretKeyRe.MatchString(k) {
				masked[k] = "<masked>"
			} else {
				masked[k] = maskJSON(item)
			}
		}
		return masked
	case []any:
		out := make([]any, len(val))
		for i := range val {
			out[i] = maskJSON(val[i])
		}
		return out
	case string:
		return maskSecretsInString(val)
	default:
		return val
	}
}

// secretKeyRe matches field names that look like secrets.
var secretKeyRe = regexp.MustCompile(`(?i)(?:secret|password|token|credential|ak|sk)`)

// maskResourceScope applies MaskResourceID to each string element in scope,
// mirroring _mask_resource_scope in gcl_runner.py.
func maskResourceScope(value any) any {
	switch val := value.(type) {
	case []any:
		out := make([]any, len(val))
		for i := range val {
			if s, ok := val[i].(string); ok {
				out[i] = MaskResourceID(s)
			} else {
				out[i] = masked
			}
		}
		return out
	case string:
		return MaskResourceID(val)
	default:
		return masked
	}
}

// enforceSafetyClass validates intent["safety_class"] against the canonical
// enum and returns a *SanitizeError when the value is unknown (fail-closed).
// It mirrors _enforce_safety_class_enum in gcl_runner.py.
func enforceSafetyClass(intent map[string]any) error {
	if !isMapWith(intent, "safety_class") {
		return nil // safety_class absent — let schema gate catch it
	}
	v := intent["safety_class"]
	if v == nil {
		return nil
	}
	if s, ok := v.(string); ok && IsValidSafetyClass(s) {
		return nil
	}
	return &SanitizeError{
		Field:   "safety_class",
		Value:   v,
		Message: fmt.Sprintf("must be one of %v; see docs/gcl-spec.md §operation_intent", SAFETY_CLASS_VALUES),
	}
}

// isMapWith reports whether m is a map and contains key.
func isMapWith(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

// MaskResourceID returns a masked representation of a single resource identifier,
// mirroring mask_resource_id in gcl_runner.py.
//
// Masking rules (in priority order):
//   - Already-masked values (*** or <masked>) pass through unchanged (idempotent).
//   - ARNs (acs:...) get only the trailing ID segment replaced with ***.
//   - UUIDs (8-4-4-4-12 hex) become a plain ***.
//   - Bare single-character inputs fall back to *** (no type prefix).
//   - Everything else is normalized to <type>-*** where <type> is the alphabetic
//     prefix (segment before the first hyphen).
func MaskResourceID(value string) string {
	if value == "" {
		return masked
	}
	if alreadyMaskedRe.MatchString(value) {
		return value
	}
	if arn := arnRe.FindStringSubmatchIndex(value); len(arn) > 0 {
		prefix := value[arn[2]:arn[3]]
		return prefix + masked
	}
	if uuidRe.MatchString(value) {
		return masked
	}
	if len(value) == 1 {
		return masked
	}
	// Normal form: preserve type prefix, mask the rest.
	// e.g. "ecs-1234abc" → "ecs-***"
	if dash := indexOf(value, string('-')); dash > 0 {
		return value[:dash+1] + masked
	}
	// No dash — mask entirely.
	return masked
}

const masked = "***"

// ---- private helpers -------------------------------------------------------

var (
	// arnRe matches Huawei Cloud ARNs: acs:<service>:<region>:<uid>:<res>/<id>
	arnRe = regexp.MustCompile(`^(acs:[A-Za-z0-9-]+:[A-Za-z0-9-]+:[A-Za-z0-9-]+:[A-Za-z0-9-]+/)(.+)$`)

	// uuidRe matches bare UUIDs (with optional hyphens).
	uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

	// alreadyMaskedRe matches already-masked identifiers.
	alreadyMaskedRe = regexp.MustCompile(`\*+\)|<masked>`)
)

func indexOf(s, sep string) int {
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}

// MaskSecrets replaces every recognized secret value in data with "<masked>",
// mirroring gcl_runner.mask_secrets. It covers HW_SECRET_ACCESS_KEY,
// SECRET_ACCESS_KEY, SecretAccessKey, and SK=... patterns.
func MaskSecrets(data []byte) string {
	s := string(data)
	replacements := []struct {
		re *regexp.Regexp
		rp string
	}{
		{regexp.MustCompile(`(HW_SECRET_ACCESS_KEY\s*=\s*)[^\s"']+`), `$1<masked>`},
		{regexp.MustCompile(`(SECRET_ACCESS_KEY\s*=\s*)[^\s"']+`), `$1<masked>`},
		{regexp.MustCompile(`(SecretAccessKey\s*[=:]\s*)[^\s"']+`), `$1<masked>`},
		{regexp.MustCompile(`(SK\s*[=:]\s*)[A-Za-z0-9/+]{20,}`), `$1<masked>`},
	}
	for _, r := range replacements {
		s = r.re.ReplaceAllString(s, r.rp)
	}
	return s
}

// maskSecretsInString applies MaskSecrets to a single string value.
// If the string contains "<masked>" it is returned unchanged (pre-sanitized).
func maskSecretsInString(s string) string {
	if containsMasked(s) {
		return s
	}
	return MaskSecrets([]byte(s))
}

func containsMasked(s string) bool {
	return contains(s, "***") || contains(s, "<masked>")
}

func contains(s, substr string) bool {
	return indexOf(s, substr) >= 0
}
