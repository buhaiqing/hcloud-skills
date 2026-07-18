// Package security implements a secret-leak scanner ported from
// scripts/gcl_security_scan.py and scripts/gcl_runner.py. It scans raw
// content for credential patterns (access keys, bearer tokens, private
// keys, password/api-key assignments) and reports each finding with its
// location and a masked snippet so traces/summaries can be shipped without
// leaking live secrets.
//
// The pattern set and the masking rules mirror the Python reference exactly:
//   - SECRET_PATTERNS from gcl_runner.py (HW/Secret access keys, SK=...)
//   - EXTRA_PATTERNS from gcl_security_scan.py (bearer, authorization,
//     private-key block, password/api-key assignments)
//   - content already containing "<masked>" is skipped wholesale, matching
//     scan_text()'s early return.
package security

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// Finding describes a single suspected credential leak in scanned content.
type Finding struct {
	// Type is the stable identifier of the matched pattern
	// (e.g. "hw_secret_access_key", "bearer_token").
	Type string
	// Line is the 1-based line number of the match.
	Line int
	// Column is the 1-based rune column of the match start within its line.
	Column int
	// Snippet is the masked version of the matched region. The secret value
	// is replaced with "<masked>" so callers can log context without leaking.
	Snippet string
}

// secretPattern pairs a stable finding type with its compiled regex.
var secretPatterns = []struct {
	typ string
	re  *regexp.Regexp
}{
	{"hw_secret_access_key", regexp.MustCompile(`HW_SECRET_ACCESS_KEY\s*=\s*[^\s"']+`)},
	{"secret_access_key", regexp.MustCompile(`SECRET_ACCESS_KEY\s*=\s*[^\s"']+`)},
	{"secret_access_key_camel", regexp.MustCompile(`SecretAccessKey\s*[=:]\s*[^\s"']+`)},
	{"sk", regexp.MustCompile(`SK\s*[=:]\s*[A-Za-z0-9/+]{20,}`)},
}

// extraPatterns mirror gcl_security_scan.EXTRA_PATTERNS. They cover cases not
// expressed as KEY=VALUE assignment in SECRET_PATTERNS.
var extraPatterns = []struct {
	typ string
	re  *regexp.Regexp
}{
	{"bearer_token", regexp.MustCompile(`Bearer\s+[A-Za-z0-9._\-]{20,}`)},
	{"authorization_header", regexp.MustCompile(`Authorization\s*[:=]\s*['"]?[^\s'"]+`)},
	{"private_key_block", regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH |PGP )?PRIVATE KEY-----`)},
	{"password_assignment", regexp.MustCompile(`(?i)password\s*[:=]\s*['"]?[^'"\s]{6,}`)},
	{"api_key_assignment", regexp.MustCompile(`(?i)(?:api[_-]?key|secret[_-]?key)\s*[:=]\s*['"]?[A-Za-z0-9._\-/+=]{16,}`)},
}

// allPatterns returns the combined secret + extra pattern set in a stable
// order, used by both ScanContent and ScanJSON.
func allPatterns() []struct {
	typ string
	re  *regexp.Regexp
} {
	out := make([]struct {
		typ string
		re  *regexp.Regexp
	}, 0, len(secretPatterns)+len(extraPatterns))
	out = append(out, secretPatterns...)
	out = append(out, extraPatterns...)
	return out
}

// maskedSnippets returns the masked form of s for a single pattern type, by
// replacing the captured secret value with "<masked>". It mirrors
// gcl_runner.mask_secrets: the prefix (pattern up to the value) is kept, the
// value is replaced. For block markers (private key) the whole marker is kept.
func maskedSnippets(typ string, s string) string {
	switch typ {
	case "hw_secret_access_key":
		return regexp.MustCompile(`(HW_SECRET_ACCESS_KEY\s*=\s*)[^\s"']+`).ReplaceAllString(s, `$1<masked>`)
	case "secret_access_key":
		return regexp.MustCompile(`(SECRET_ACCESS_KEY\s*=\s*)[^\s"']+`).ReplaceAllString(s, `$1<masked>`)
	case "secret_access_key_camel":
		return regexp.MustCompile(`(SecretAccessKey\s*[=:]\s*)[^\s"']+`).ReplaceAllString(s, `$1<masked>`)
	case "sk":
		return regexp.MustCompile(`(SK\s*[=:]\s*)[A-Za-z0-9/+]{20,}`).ReplaceAllString(s, `$1<masked>`)
	case "bearer_token":
		// Keep the "Bearer " scheme, mask the token that follows.
		return regexp.MustCompile(`(Bearer\s+)[A-Za-z0-9._\-]+`).ReplaceAllString(s, `$1<masked>`)
	case "authorization_header":
		// Keep the header name, mask the credential that follows.
		return regexp.MustCompile(`(Authorization\s*[:=]\s*['"]?)[^\s'"]+`).ReplaceAllString(s, `$1<masked>`)
	default:
		// password/api-key assignments and any other EXTRA pattern: mask the
		// value after the first whitespace/quote boundary so the surrounding
		// label is retained for context.
		return regexp.MustCompile(`(\S+\s*[:=]\s*['"]?)[^\s'"]+`).ReplaceAllString(s, `$1<masked>`)
	}
}

// ScanContent scans raw content for credential patterns. It returns the list
// of findings (empty when none). When content already contains "<masked>" it
// is treated as pre-sanitized and skipped entirely, mirroring
// gcl_security_scan.scan_text's early return.
func ScanContent(data []byte) ([]Finding, error) {
	text := string(data)
	if strings.Contains(text, "<masked>") {
		return nil, nil
	}

	var findings []Finding
	all := make([]struct {
		typ string
		re  *regexp.Regexp
	}, 0, len(secretPatterns)+len(extraPatterns))
	all = append(all, secretPatterns...)
	all = append(all, extraPatterns...)

	for _, p := range all {
		loc := p.re.FindStringIndex(text)
		if loc == nil {
			continue
		}
		matched := text[loc[0]:loc[1]]
		line, col := lineColumn(text, loc[0])
		findings = append(findings, Finding{
			Type:    p.typ,
			Line:    line,
			Column:  col,
			Snippet: maskedSnippets(p.typ, matched),
		})
	}
	return findings, nil
}

// ScanJSON walks a decoded JSON payload (using json.Decoder with UseNumber so
// numbers are not coerced to float64) and applies the same credential patterns
// as ScanContent to every scanned string value. It mirrors
// gcl_security_scan.scan_payload: a string is scanned if its key is in
// scannedTextFields OR its length is <= 200000, and strings already containing
// "<masked>" are skipped. The field path (e.g. "iterations[0].critic") is
// reported so callers can locate the leak.
func ScanJSON(data []byte) ([]Finding, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		return nil, err
	}
	var findings []Finding
	walkValue(root, "", &findings)
	return findings, nil
}

var scannedTextFields = map[string]bool{
	"request": true, "command": true, "result_excerpt": true, "operation": true,
	"user_request": true, "summary": true, "final_state": true, "raw_response": true,
}

func walkValue(v any, prefix string, out *[]Finding) {
	switch val := v.(type) {
	case map[string]any:
		for k, item := range val {
			child := k
			if prefix != "" {
				child = prefix + "." + k
			}
			if s, ok := item.(string); ok {
				scanStringField(s, child, out)
			} else {
				walkValue(item, child, out)
			}
		}
	case []any:
		for i, item := range val {
			child := prefix + "[" + strconv.Itoa(i) + "]"
			if s, ok := item.(string); ok {
				scanStringField(s, child, out)
			} else {
				walkValue(item, child, out)
			}
		}
	}
}

func scanStringField(s, field string, out *[]Finding) {
	if !scannedTextFields[field] && len(s) > 200000 {
		return
	}
	if strings.Contains(s, "<masked>") {
		return
	}
	for _, p := range allPatterns() {
		if p.re.MatchString(s) {
			*out = append(*out, Finding{
				Type:    p.typ,
				Line:    0,
				Column:  0,
				Snippet: maskedSnippets(p.typ, firstMatch(p.re, s)),
			})
		}
	}
}

func firstMatch(re *regexp.Regexp, s string) string {
	loc := re.FindStringIndex(s)
	if loc == nil {
		return s
	}
	return s[loc[0]:loc[1]]
}

// lineColumn converts a byte offset into 1-based line and rune-column numbers.
func lineColumn(text string, offset int) (int, int) {
	line := 1
	lastNL := -1
	for i := 0; i < offset && i < len(text); i++ {
		if text[i] == '\n' {
			line++
			lastNL = i
		}
	}
	col := 0
	for i := lastNL + 1; i < offset && i < len(text); i++ {
		col++
	}
	return line, col + 1
}

// MaskSecrets replaces every recognized secret value in data with "<masked>",
// mirroring gcl_runner.mask_secrets. It is a content-level helper used to
// sanitize operation_intent and other free-form fields.
func MaskSecrets(data []byte) []byte {
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
	return []byte(s)
}
