package gcl

import (
	"regexp"
	"strconv"
	"strings"
)

// checkWAF performs offline security, cost, and stability checks.
func (d *DefaultHallucinationDetector) checkWAF(gen GeneratorOutput, trace *GCLTrace) (*L3Result, error) {
	result := &L3Result{}

	// --- Security: credential exposure in raw output ---
	if violations := d.checkCredentialExposure(gen.ResultExcerpt); len(violations) > 0 {
		result.Violations = append(result.Violations, violations...)
	}

	// --- Security: dangerous verbs without guard ---
	if violations := d.checkDangerousVerbs(gen.Command); len(violations) > 0 {
		result.Violations = append(result.Violations, violations...)
	}

	// --- Cost: high-cost operation ---
	if violations := d.checkHighCostOperation(gen.Command, trace); len(violations) > 0 {
		result.Violations = append(result.Violations, violations...)
	}

	// --- Stability: multi-resource mutation without rollback ---
	if violations := d.checkStability(gen.Command, trace); len(violations) > 0 {
		result.Violations = append(result.Violations, violations...)
	}

	if len(result.Violations) > 0 {
		result.Details = "WAF violations: " + strconv.Itoa(len(result.Violations))
	} else {
		result.Details = "WAF passed"
	}

	return result, nil
}

// checkCredentialExposure scans for plain-text secrets in the output.
// Anything flagged here triggers SAFETY_FAIL in the parent.
func (d *DefaultHallucinationDetector) checkCredentialExposure(output string) []L3Violation {
	var violations []L3Violation
	patterns := []struct {
		name string
		re   *regexp.Regexp
	}{
		{"HW_ACCESS_KEY_ID", regexp.MustCompile(`(?i)(HW_ACCESS_KEY_ID|hcloud_access_key)\s*[=: ]\s*[A-Z0-9]{20,}`)},
		{"HW_SECRET_ACCESS_KEY", regexp.MustCompile(`(?i)(HW_SECRET_ACCESS_KEY|hcloud_secret_key)\s*[=: ]\s*[A-Za-z0-9/+=]{30,}`)},
		{"generic-AK", regexp.MustCompile(`(?i)\bAK[=: ]\s*[A-Z0-9]{20,}`)},
		{"generic-SK", regexp.MustCompile(`(?i)\bSK[=: ]\s*[A-Za-z0-9/+=]{30,}`)},
		{"AK/SK pair", regexp.MustCompile(`(?i)AK[=: ]\s*[A-Z0-9]{20,}.{0,50}SK[=: ]\s*[A-Za-z0-9/+=]{30,}`)},
		{"password in plain text", regexp.MustCompile(`(?i)password\s*[=:]\s*["']?[A-Za-z0-9@#$%]{8,}`)},
		{"Bearer token", regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+`)},
	}

	for _, p := range patterns {
		if p.re.MatchString(output) {
			violations = append(violations, L3Violation{
				Dimension: "security",
				Rule:      "credential-exposure",
				Detail:    "possible plain-text credential: " + p.name,
			})
		}
	}
	return violations
}

// checkDangerousVerbs detects risky operations without --dry-run or --no-act.
func (d *DefaultHallucinationDetector) checkDangerousVerbs(command string) []L3Violation {
	var violations []L3Violation
	dangerous := []string{"delete", "drop", "terminate", "purge", "destroy", "remove"}
	hasGuard := regexp.MustCompile(`(?i)(--dry-run|--no-act|--confirm|--force)`).MatchString(command)

	for _, verb := range dangerous {
		if strings.Contains(strings.ToLower(command), verb) && !hasGuard {
			violations = append(violations, L3Violation{
				Dimension: "stability",
				Rule:      "dangerous-verb-without-guard",
				Detail:    "command contains '" + verb + "' without --dry-run or --confirm guard",
			})
			break
		}
	}
	return violations
}

// checkHighCostOperation flags scale-out or high-frequency mutations.
func (d *DefaultHallucinationDetector) checkHighCostOperation(command string, trace *GCLTrace) []L3Violation {
	var violations []L3Violation

	// Check resource_context from FinOps context.
	if trace.ResourceContext != nil {
		if cost, ok := trace.ResourceContext["monthly_cost_usd"].(float64); ok && cost > 1000 {
			violations = append(violations, L3Violation{
				Dimension: "cost",
				Rule:      "high-cost-resource",
				Detail:    "target resource monthly cost $" + strconv.FormatFloat(cost, 'f', 2, 64),
			})
		}
	}

	// Flag bulk operations.
	if regexp.MustCompile(`(?i)(--batch|--all|--instances\s+\*|--count\s+[5-9]|[5-9]+ instances)`).MatchString(command) {
		violations = append(violations, L3Violation{
			Dimension: "cost",
			Rule:      "bulk-mutation",
			Detail:    "bulk operation detected in command",
		})
	}

	return violations
}

// checkStability ensures multi-resource changes have a rollback plan.
func (d *DefaultHallucinationDetector) checkStability(command string, trace *GCLTrace) []L3Violation {
	var violations []L3Violation

	// If operation_intent says multi-resource but no rollback plan → warning.
	if trace.OperationIntent != nil {
		if blastRadius, ok := trace.OperationIntent["blast_radius"].(string); ok {
			multiResource := blastRadius == "multi-resource" || blastRadius == "service-wide" || blastRadius == "region-wide"
			hasRollback := false
			if rollback, ok := trace.OperationIntent["rollback_plan"].(string); ok && rollback != "" {
				hasRollback = true
			}
			if multiResource && !hasRollback {
				violations = append(violations, L3Violation{
					Dimension: "stability",
					Rule:      "no-rollback-plan",
					Detail:    "blast_radius=" + blastRadius + " without rollback_plan",
				})
			}
		}
	}

	return violations
}
