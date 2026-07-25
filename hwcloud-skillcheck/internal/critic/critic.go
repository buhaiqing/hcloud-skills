// Package critic is the rule-based 5-dimension quality scorer for GCL
// traces. It mirrors scripts/critic_v1.py 1:1 so callers can swap the Python
// invocation for the Go binary with no behaviour change.
//
// Output schema (returned by Score) MUST match the contract gcl_runner
// validates:
//
//	{
//	  "scores": {"correctness": float, "safety": float, ...},
//	  "suggestions": [string],
//	  "blocking": bool,
//	  ...
//	}
//
// Each score dimension is one of {0, 0.5, 1}. FinOps cost is surfaced as a
// suggestion, not a score dimension (the rubric doesn't include FinOps).
package critic

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
)

// Verb classification.
var (
	readVerbs      = map[string]bool{"list": true, "show": true, "describe": true, "get": true, "fetch": true, "listquotas": true, "showquota": true}
	writeVerbs     = map[string]bool{"create": true, "delete": true, "terminate": true, "remove": true, "update": true, "modify": true, "attach": true, "detach": true, "reboot": true, "restart": true, "stop": true, "start": true}
	destructiveSet = []string{"delete", "terminate", "remove", "drop", "purge"}
)

// CostTable maps (skill_short, operation) → USD/call. Defaults to 0.001 when
// missing (see EstimateCost).
var CostTable = map[string]float64{
	"ecs,create-server":       0.05,
	"ecs,delete-server":       0.0,
	"rds,create-instance":     0.12,
	"rds,delete-instance":     0.0,
	"rds,enlarge-volume":      0.01,
	"elb,create-loadbalancer": 0.025,
	"elb,delete-loadbalancer": 0.0,
	"vpc,create-vpc":          0.001,
	"cce,create-cluster":      0.5,
	"cce,create-node":         0.08,
}

var (
	destructiveRe = regexp.MustCompile(`(?i)\b(` + joinOr(destructiveSet) + `)\b`)
	dryRunRe      = regexp.MustCompile(`(?i)--dry-run|-dry-run`)
	hcloudRe      = regexp.MustCompile(`(?i)\bhcloud\b`)
	goRunRe       = regexp.MustCompile(`(?i)\bgo run\b`)
	hcloudCmdRe   = regexp.MustCompile(`(?i)\bhcloud\s+([a-z0-9_-]+)`)
	opRe          = regexp.MustCompile(`(?i)\bhcloud\s+[a-z0-9_-]+\s+([a-zA-Z][\w-]*)`)
)

// secretPatterns mirror scripts/gcl_runner.py:SECRET_PATTERNS.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)HW_SECRET_ACCESS_KEY\s*=\s*[^\s"']+`),
	regexp.MustCompile(`(?i)SECRET_ACCESS_KEY\s*=\s*[^\s"']+`),
	regexp.MustCompile(`(?i)SecretAccessKey\s*[=:]\s*[^\s"']+`),
	regexp.MustCompile(`(?i)SK\s*[=:]\s*[A-Za-z0-9/+]{20,}`),
}

// HasCredentialLeak returns true if text contains an unmasked credential
// pattern. The presence of "<masked>" anywhere suppresses a leak finding so
// trace redaction works correctly.
func HasCredentialLeak(text string) bool {
	if text == "" {
		return false
	}
	if contains(text, "<masked>") {
		return false
	}
	for _, p := range secretPatterns {
		if p.MatchString(text) {
			return true
		}
	}
	return false
}

// Score runs all 5 rules against a generator payload and returns the
// critic-compatible result map.
func Score(generator map[string]any) map[string]any {
	command, _ := generator["command"].(string)
	safety, sSafety := scoreSafety(command)
	correctness, sCorrect := scoreCorrectness(command)
	idempotency := scoreIdempotency(command)
	traceability, sTrace := scoreTraceability(generator)
	spec, sSpec := scoreSpecCompliance(command, generator)

	scores := map[string]any{
		"correctness":     correctness,
		"safety":          safety,
		"idempotency":     idempotency,
		"traceability":    traceability,
		"spec_compliance": spec,
	}

	blocking := safety == 0.0 || correctness == 0.0
	suggestions := append([]string{}, sSafety...)
	suggestions = append(suggestions, sCorrect...)
	suggestions = append(suggestions, sTrace...)
	suggestions = append(suggestions, sSpec...)
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	cost, skillShort := EstimateCost(command)
	if cost >= 0.1 {
		suggestions = append(suggestions,
			fmt.Sprintf("FinOps: estimated cost $%.3f/call (skill=%s); consider cheaper alternative or batch", cost, skillShort))
	}
	if safety == 1.0 && destructiveRe.MatchString(command) {
		suggestions = append(suggestions,
			"SecOps: destructive command gated behind --dry-run; ensure human approval before live run")
	}

	return map[string]any{
		"scores":              scores,
		"suggestions":         suggestions,
		"blocking":            blocking,
		"_mode":               "critic-v1",
		"finops_estimate_usd": cost,
	}
}

func scoreSafety(command string) (float64, []string) {
	if destructiveRe.MatchString(command) && !dryRunRe.MatchString(command) {
		return 0.0, []string{"Destructive command without --dry-run; require explicit confirmation or dry-run gate"}
	}
	return 1.0, nil
}

func scoreCorrectness(command string) (float64, []string) {
	if command == "" {
		return 0.0, []string{"Empty command"}
	}
	if hcloudRe.MatchString(command) || goRunRe.MatchString(command) {
		return 1.0, nil
	}
	return 0.5, []string{"Command does not match hcloud or `go run` pattern; verify spelling"}
}

func scoreIdempotency(command string) float64 {
	op := ExtractOperation(command)
	if readVerbs[op] {
		return 1.0
	}
	// writeVerbs and unknown both yield 0.5 per the Python original.
	return 0.5
}

func scoreTraceability(generator map[string]any) (float64, []string) {
	excerpt, _ := generator["result_excerpt"].(string)
	command, _ := generator["command"].(string)
	if command != "" && excerpt != "" {
		return 1.0, nil
	}
	return 0.0, []string{"Missing result_excerpt or command; capture stdout/stderr in trace"}
}

func scoreSpecCompliance(command string, generator map[string]any) (float64, []string) {
	var s []string
	exitCode := numFromAny(generator["exit_code"], -1)
	excerpt, _ := generator["result_excerpt"].(string)
	if HasCredentialLeak(excerpt) || HasCredentialLeak(command) {
		return 0.0, []string{"Credential leak detected; mask HW_SECRET_ACCESS_KEY and re-run"}
	}
	if exitCode == 0 && (hcloudRe.MatchString(command) || goRunRe.MatchString(command)) {
		return 1.0, nil
	}
	if exitCode != 0 {
		s = append(s, fmt.Sprintf("Exit code %d; verify endpoint/credentials", exitCode))
	}
	s = append(s, "Spec mismatch: ensure command matches skill's CLI/SDK pattern")
	return 0.5, s
}

// EstimateCost returns (cost_usd, skill_short) for a command. Defaults to
// 0.001 when the (skill, op) pair is missing from CostTable.
func EstimateCost(command string) (float64, string) {
	skill := ExtractSkillShort(command)
	op := ExtractOperation(command)
	if c, ok := CostTable[skill+","+op]; ok {
		return c, skill
	}
	return 0.001, skill
}

// ExtractSkillShort returns the best-effort skill short name from a
// `hcloud <skill> ...` command.
func ExtractSkillShort(command string) string {
	m := hcloudCmdRe.FindStringSubmatch(command)
	if len(m) < 2 {
		return ""
	}
	return lowerASCII(m[1])
}

// ExtractOperation returns the best-effort operation verb from the command.
func ExtractOperation(command string) string {
	m := opRe.FindStringSubmatch(command)
	if len(m) < 2 {
		return ""
	}
	return lowerASCII(m[1])
}

// LoadGeneratorFile reads a generator trace from disk and returns its decoded
// payload. Used by the CLI; tests call Score directly.
func LoadGeneratorFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// ScoreFile is a convenience: read the file and run Score.
func ScoreFile(path string) (map[string]any, error) {
	gen, err := LoadGeneratorFile(path)
	if err != nil {
		return nil, err
	}
	return Score(gen), nil
}

func numFromAny(v any, fallback int) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		if x, err := strconv.Atoi(n); err == nil {
			return x
		}
	}
	return fallback
}

func joinOr(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "|"
		}
		out += p
	}
	return out
}

func lowerASCII(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
