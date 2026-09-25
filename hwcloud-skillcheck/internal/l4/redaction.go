package l4

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	persistenceMask = "<masked>"
	// NUL delimiters make the internal sentinel non-typable in ordinary input
	// and prevent a user-supplied " MASKED " string from being rewritten.
	persistenceMarker = "\x00__L4_PERSISTENCE_MASK__\x00"
)

type persistenceReplacement struct {
	re       *regexp.Regexp
	repl     string
	category string
}

// Ordered from structured, high-confidence forms to generic credential
// assignments. Every value replacement is canonicalized to <masked>.
var persistenceReplacements = []persistenceReplacement{
	{
		re:       regexp.MustCompile(`(?s)-----BEGIN [^-\r\n]+-----.*?-----END [^-\r\n]+-----`),
		repl:     persistenceMask,
		category: "private-key",
	},
	{
		re:       regexp.MustCompile(`(?i)((?:HW_SECRET_ACCESS_KEY|SECRET_ACCESS_KEY|SECRET_KEY|ACCESS_KEY(?:_ID)?|CLIENT_SECRET|API[_-]?KEY|TOKEN|PASSWORD|PASSWD|PRIVATE_KEY|CREDENTIAL|SK|SecretAccessKey|AK)\s*[=:]\s*)(?:"[^"]*"|'[^']*'|[^\s,;]+)`),
		repl:     "${1}" + persistenceMask,
		category: "credential-assignment",
	},
	{
		re:       regexp.MustCompile(`(?i)((?:Authorization|Proxy-Authorization|WWW-Authenticate)\s*:\s*)(?:Bearer|Basic|Digest)?\s*[^\r\n]+`),
		repl:     "${1}" + persistenceMask,
		category: "authorization-header",
	},
	{
		re:       regexp.MustCompile(`(?i)((?:Set-Cookie|Cookie)\s*:\s*)[^\r\n]+`),
		repl:     "${1}" + persistenceMask,
		category: "cookie-header",
	},
	{
		re:       regexp.MustCompile(`(?i)(https?://)[^/\s:@]+:[^@\s/]+@`),
		repl:     "${1}" + persistenceMask + "@",
		category: "url-userinfo",
	},
	{
		re:       regexp.MustCompile(`(?i)([?&](?:access_token|refresh_token|session(?:id)?|signature|x-amz-signature|x-hw-signature|api_key|apikey|token|secret|password)=)[^&#\s]+`),
		repl:     "${1}" + persistenceMask,
		category: "url-secret-query",
	},
	{
		re:       regexp.MustCompile(`(?i)(--(?:token|access[-_]?token|api[-_]?key|secret|password|signature)(?:=|\s+))[^\s]+`),
		repl:     "${1}" + persistenceMask,
		category: "credential-flag",
	},
	{
		re:       regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{20,}|sk-[A-Za-z0-9]{20,})\b`),
		repl:     persistenceMask,
		category: "api-key",
	},
	{
		re:       regexp.MustCompile(`\bAK[A-Z0-9]{16,}\b`),
		repl:     persistenceMask,
		category: "huawei-access-key",
	},
	{
		re:       regexp.MustCompile(`(?i)\b(?:acs|arn):[^\s"',;]+`),
		repl:     persistenceMask,
		category: "arn",
	},
	{
		re:       regexp.MustCompile(`\b(?:AKIA|ASIA)[A-Z0-9]{16}\b`),
		repl:     persistenceMask,
		category: "access-key",
	},
	{
		re:       regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`),
		repl:     persistenceMask,
		category: "jwt",
	},
	{
		// A product short name is a resource-ID prefix only at the start of a
		// token.  A hyphen inside a skill slug (huaweicloud-vpc-ops) is
		// structural, not a resource identifier.
		re:       regexp.MustCompile(`(?i)(^|[^A-Za-z0-9-])(?:ecs|rds|vpc|dcs|elb|cce|kms|iam|sg|evs|as|sbs|csbs|ces|clb|ccs)-[A-Za-z0-9][A-Za-z0-9-]{3,}\b`),
		repl:     "${1}" + persistenceMask,
		category: "resource-id",
	},
	{
		re:       regexp.MustCompile(`(?i)((?:resource|account|project|user|domain|tenant)(?:[_-]?id)?\s*[=:]\s*)(?:"[^"]*"|'[^']*'|[^\s,;]+)`),
		repl:     "${1}" + persistenceMask,
		category: "tenant-identifier",
	},
}

var (
	canonicalMarkerPattern = regexp.MustCompile(`<id>|<arn>|<redacted>|<masked>|\*\*\*`)
	opaqueTokenPattern     = regexp.MustCompile(`\b[A-Za-z0-9_./:-]{16,}\b`)
	generatedIDPattern     = regexp.MustCompile(`^[0-9a-fA-F]{16}(?:[0-9a-fA-F]{16,48})?$`)
	uuidPattern            = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	timestampPattern       = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$`)
)

// sanitizeSecretForPersistence and sanitizeRequestForPersistence share one
// implementation. The former name remains at execution-domain call sites; no
// persistence surface may use a weaker sanitizer.
func sanitizeSecretForPersistence(field, value string) (string, error) {
	return sanitizeStringForPersistence(field, value)
}

func sanitizeRequestForPersistence(field, value string) (string, error) {
	return sanitizeStringForPersistence(field, value)
}

func sanitizeStringForPersistence(field, value string) (string, error) {
	if value == "" {
		return value, nil
	}
	if passthroughFieldValue(field, value) {
		return value, nil
	}
	normalized := canonicalMarkerPattern.ReplaceAllStringFunc(value, func(string) string { return persistenceMask })
	for _, replacement := range persistenceReplacements {
		normalized = replacement.re.ReplaceAllString(normalized, replacement.repl)
	}
	if firstUnsafeOpaque(normalized) != "" && !safeOperationalValue(normalized) {
		if strings.Contains(field, "preference") || strings.Contains(field, "output") {
			return opaqueTokenPattern.ReplaceAllString(normalized, persistenceMask), nil
		}
		return "", fmt.Errorf("redaction refused field %s: category opaque-token", field)
	}
	return normalized, nil
}

// Typed-field passthrough patterns: IDs, skill slugs and schema identifiers.
// Values in typed fields are shape-validated; prose that fails the shape falls
// through to the opaque-token check (fail-closed).
var (
	idTokenPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{3,63}$`)
	skillSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{2,63}$`)
	schemaIDPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{1,63}$`)
)

// passthroughFieldValue classifies structured operational values by field role.
// Caller field names are role-qualified ("lookup skill", "task updated",
// "context.schema"), so matching is by exact name, suffix or role substring.
func passthroughFieldValue(field, value string) bool {
	switch {
	case field == "task root", strings.HasSuffix(field, "trace_persisted"):
		return true
	case field == "id" || field == "task id" || field == "gcl experiment id" || strings.HasSuffix(field, "_id") || strings.HasSuffix(field, ".id"):
		return idTokenPattern.MatchString(value)
	case field == "timestamp" || strings.Contains(field, "timestamp") || strings.Contains(field, "created") || strings.Contains(field, "updated") || strings.Contains(field, "started") || strings.Contains(field, "finished") || strings.HasSuffix(field, "_at"):
		return timestampPattern.MatchString(value)
	case field == "skill" || strings.HasSuffix(field, " skill") || strings.HasSuffix(field, "_skill"):
		return skillSlugPattern.MatchString(value)
	case strings.HasSuffix(field, " schema"):
		return schemaIDPattern.MatchString(value)
	case strings.HasSuffix(field, " session"):
		return uuidPattern.MatchString(value)
	}
	return false
}

// sanitizeMapKey treats map keys as names, not secret payloads. Known secret
// shapes are still replaced, but the opaque catch-all is not applied: it
// would mask schema keys ("patterns_matched", "knowledge_base_skills_used")
// to <masked>, collide distinct keys, and break JSON round-trip on unmarshal.
func sanitizeMapKey(key string) (string, error) {
	safe := canonicalMarkerPattern.ReplaceAllStringFunc(key, func(string) string { return persistenceMask })
	for _, replacement := range persistenceReplacements {
		safe = replacement.re.ReplaceAllString(safe, replacement.repl)
	}
	if strings.ContainsAny(safe, "\x00\n\r") {
		return "", fmt.Errorf("redaction refused map key: category control-char")
	}
	return safe, nil
}

// firstUnsafeOpaque returns the first 16+ char token that is neither a URL
// structure nor an assignment key position. Key names and URLs are metadata:
// the secret-shaped content beside them (the value after '='/':' , the query
// string) is scanned as its own token or masked by persistenceReplacements.
func firstUnsafeOpaque(s string) string {
	for _, loc := range opaqueTokenPattern.FindAllStringIndex(s, -1) {
		run := s[loc[0]:loc[1]]
		if strings.Contains(run, "://") {
			continue
		}
		if isAssignmentKeyName(s[loc[1]:]) {
			continue
		}
		// Operational action names are vocabulary, not opaque credentials.
		// Keep the existing whole-value exception for free-form messages, and
		// additionally permit these exact tokens inside a command/reason.
		if safeOperationalValue(run) {
			continue
		}
		return run
	}
	return ""
}

func isAssignmentKeyName(rest string) bool {
	rest = strings.TrimLeft(rest, " \t")
	return strings.HasPrefix(rest, "=") || strings.HasPrefix(rest, ":")
}

func safeOperationalValue(value string) bool {
	switch value {
	case "diagnose_and_remediate", "connection reset", "delete-instances", "list-servers", "restart-instance", "describe-instance", "human_review_required", "auto_proceed", "completed", "failed", "aborted":
		return true
	default:
		return false
	}
}

// sanitizeAnyForPersistence deep-copies maps and slices before sanitizing all
// string keys and values. Map keys are included because arbitrary overlays and
// preferences can carry sensitive data in either position.
func sanitizeAnyForPersistence(field string, value any) (any, []string, error) {
	switch v := value.(type) {
	case string:
		if strings.HasSuffix(field, "trace_persisted") {
			return v, []string{field}, nil
		}
		out, err := sanitizeStringForPersistence(field, v)
		return out, []string{field}, err
	case []string:
		out := make([]string, len(v))
		var paths []string
		for i, item := range v {
			safe, err := sanitizeStringForPersistence(fmt.Sprintf("%s[%d]", field, i), item)
			if err != nil {
				return nil, nil, err
			}
			out[i] = safe
			paths = append(paths, fmt.Sprintf("%s[%d]", field, i))
		}
		return out, paths, nil
	case map[string]string:
		out := make(map[string]string, len(v))
		var paths []string
		for key, item := range v {
			safeKey, err := sanitizeMapKey(key)
			if err != nil {
				return nil, nil, err
			}
			safeValue, err := sanitizeStringForPersistence(field+"."+safeKey, item)
			if err != nil {
				return nil, nil, err
			}
			if _, exists := out[safeKey]; exists {
				return nil, nil, fmt.Errorf("redaction refused field %s: category key-collision", field)
			}
			out[safeKey] = safeValue
			paths = append(paths, field+"."+safeKey)
		}
		return out, paths, nil
	case []any:
		out := make([]any, len(v))
		var paths []string
		for i, item := range v {
			child, childPaths, err := sanitizeAnyForPersistence(fmt.Sprintf("%s[%d]", field, i), item)
			if err != nil {
				return nil, nil, err
			}
			out[i] = child
			paths = append(paths, childPaths...)
		}
		return out, paths, nil
	case map[string]any:
		out := make(map[string]any, len(v))
		var paths []string
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			safeKey, err := sanitizeMapKey(key)
			if err != nil {
				return nil, nil, err
			}
			if _, exists := out[safeKey]; exists {
				return nil, nil, fmt.Errorf("redaction refused field %s: category key-collision", field)
			}
			child, childPaths, err := sanitizeAnyForPersistence(field+"."+safeKey, v[key])
			if err != nil {
				return nil, nil, err
			}
			out[safeKey] = child
			paths = append(paths, field+"."+safeKey)
			paths = append(paths, childPaths...)
		}
		return out, paths, nil
	default:
		return value, nil, nil
	}
}

// Topology uses symbolic roles for derived graph nodes.  These are not
// resource identifiers and must remain useful to the graph consumer; concrete
// IDs still go through the strict resource sanitizer.
func sanitizeTopologyResourceForPersistence(resource string) (string, string, error) {
	if resource == "rds:instance" || resource == "vpc:subnet" {
		return resource, strings.SplitN(resource, ":", 2)[0], nil
	}
	return sanitizeResourceForPersistence(resource)
}

func sanitizeResourceForPersistence(resource string) (string, string, error) {
	allowed := map[string]bool{
		"rds": true, "ecs": true, "vpc": true, "elb": true,
		"cce": true, "dcs": true, "gaussdb": true, "dms": true,
	}
	if resource == "" {
		return persistenceMask, "unknown", nil
	}
	colon := strings.IndexByte(resource, ':')
	if colon <= 0 || !allowed[strings.ToLower(resource[:colon])] {
		return persistenceMask, "unknown", nil
	}
	resourceType := strings.ToLower(resource[:colon])
	if len(resource) == colon+1 {
		return resourceType + ":" + persistenceMask, resourceType, nil
	}
	return resourceType + ":" + persistenceMask, resourceType, nil
}

// SanitizeOrchestratorOutput returns the only L4 OrchestratorOutput form
func SanitizeOrchestratorOutput(in *OrchestratorOutput) (*OrchestratorOutput, error) {
	if in == nil {
		return nil, fmt.Errorf("redaction refused field output: category nil")
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("redaction refused field output: category encode")
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, fmt.Errorf("redaction refused field output: category decode")
	}
	if resource, ok := generic["resource"].(string); ok {
		safeResource, _, err := sanitizeResourceForPersistence(resource)
		if err != nil {
			return nil, err
		}
		generic["resource"] = safeResource
	}
	if topology, ok := generic["topology"].(map[string]any); ok {
		if origin, ok := topology["origin"].(string); ok {
			safeOrigin, _, err := sanitizeTopologyResourceForPersistence(origin)
			if err != nil {
				return nil, err
			}
			topology["origin"] = safeOrigin
		}
		if affected, ok := topology["affected_resources"].([]any); ok {
			for i, raw := range affected {
				if value, ok := raw.(string); ok {
					safeValue, _, err := sanitizeResourceForPersistence(value)
					if err != nil {
						return nil, err
					}
					affected[i] = safeValue
				}
			}
		}
	}
	safe, _, err := sanitizeAnyForPersistence("output", generic)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(safe)
	if err != nil {
		return nil, fmt.Errorf("redaction refused field output: category encode-safe")
	}
	var out OrchestratorOutput
	if err := json.Unmarshal(encoded, &out); err != nil {
		return nil, fmt.Errorf("redaction refused field output: category decode-safe")
	}
	return &out, nil
}

// SanitizeAutofixResult returns a safe external projection. Raw executor
// output remains confined to AutoFix's verification branch.
func SanitizeAutofixResult(in AutofixResult) (AutofixResult, error) {
	out := in
	var err error
	if out.PlaybookID, err = sanitizeStringForPersistence("autofix.playbook_id", out.PlaybookID); err != nil {
		return AutofixResult{}, err
	}
	if out.VerifiedOut, err = sanitizeStringForPersistence("autofix.verified_output", out.VerifiedOut); err != nil {
		return AutofixResult{}, err
	}
	if out.Error, err = sanitizeStringForPersistence("autofix.error", out.Error); err != nil {
		return AutofixResult{}, err
	}
	return out, nil
}

// SanitizeExternalString is the boundary helper for human-oriented CLI text.
func SanitizeExternalString(value string) (string, error) {
	return sanitizeStringForPersistence("external", value)
}

// Small wrappers keep encoding details out of the sanitizer and make its
// package-level dependencies explicit.
var (
	jsonMarshal   = json.Marshal
	jsonUnmarshal = json.Unmarshal
)
