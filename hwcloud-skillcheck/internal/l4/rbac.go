package l4

import (
	"fmt"
	"regexp"
	"strings"
)

// RBACRisk is the risk tier for an operation.
type RBACRisk string

const (
	RBACRiskNone     RBACRisk = "none"
	RBACRiskLow      RBACRisk = "low"
	RBACRiskMedium   RBACRisk = "medium"
	RBACRiskHigh     RBACRisk = "high"
	RBACRiskCritical RBACRisk = "critical"
)

// RBACDecision is the result of an RBAC permission check.
type RBACDecision struct {
	Allowed     bool     `json:"allowed"`
	Risk        RBACRisk `json:"risk"`
	Reason      string   `json:"reason"`
	RequiresEnv []string `json:"requires_env,omitempty"`
	Approvers   []string `json:"approvers,omitempty"`
	// Immutable constraints that can never be overridden.
	ImmutableConstraints []string `json:"immutable_constraints,omitempty"`
}

// OperationPermission defines the permission requirements for one operation type.
type OperationPermission struct {
	RiskLevel        RBACRisk `json:"risk_level"`
	RequiresEnv      []string `json:"requires_env,omitempty"`
	RequiredEnvMatch []string `json:"required_env_match,omitempty"` // e.g. ["HW_REGION_ID"]
	Approvers        []string `json:"approvers,omitempty"`          // usernames for high/critical
	Immutable        bool     `json:"immutable"`                    // if true, cannot be auto-approved
}

// DefaultOperationPermissions is the built-in permission table.
// Operations not listed default to RBACRiskHigh and require approval.
var DefaultOperationPermissions = map[string]OperationPermission{
	// Destructive operations — always require human approval regardless of trust level.
	"delete":                {RiskLevel: RBACRiskCritical, Immutable: true},
	"terminate":             {RiskLevel: RBACRiskCritical, Immutable: true},
	"destroy":               {RiskLevel: RBACRiskCritical, Immutable: true},
	"drop":                  {RiskLevel: RBACRiskCritical, Immutable: true},
	"remove":                {RiskLevel: RBACRiskHigh, Immutable: false},
	"delete-security-group": {RiskLevel: RBACRiskCritical, Immutable: true},
	"delete-subnet":         {RiskLevel: RBACRiskHigh, Immutable: false},
	"delete-vpc":            {RiskLevel: RBACRiskCritical, Immutable: true},
	"delete-instance":       {RiskLevel: RBACRiskCritical, Immutable: true},
	"delete-database":       {RiskLevel: RBACRiskCritical, Immutable: true},
	"terminate-instance":    {RiskLevel: RBACRiskCritical, Immutable: true},

	// High-risk operations — require approval for non-L4 trust levels.
	"create":            {RiskLevel: RBACRiskMedium},
	"update":            {RiskLevel: RBACRiskMedium},
	"modify":            {RiskLevel: RBACRiskMedium},
	"attach":            {RiskLevel: RBACRiskMedium},
	"detach":            {RiskLevel: RBACRiskHigh},
	"associate-eip":     {RiskLevel: RBACRiskMedium},
	"disassociate-eip":  {RiskLevel: RBACRiskHigh},
	"failover-instance": {RiskLevel: RBACRiskHigh},
	"restart-instance":  {RiskLevel: RBACRiskMedium},
	"resize":            {RiskLevel: RBACRiskMedium},
	"scale":             {RiskLevel: RBACRiskMedium},
	"restore":           {RiskLevel: RBACRiskHigh},
	"backup":            {RiskLevel: RBACRiskLow},
	"snapshot":          {RiskLevel: RBACRiskLow},
	"grant":             {RiskLevel: RBACRiskHigh},
	"revoke":            {RiskLevel: RBACRiskHigh},

	// Read-only operations — auto-approved for all trust levels.
	"list":            {RiskLevel: RBACRiskLow},
	"list-instances":  {RiskLevel: RBACRiskLow},
	"list-servers":    {RiskLevel: RBACRiskLow},
	"describe":        {RiskLevel: RBACRiskLow},
	"show":            {RiskLevel: RBACRiskLow},
	"get":             {RiskLevel: RBACRiskLow},
	"search":          {RiskLevel: RBACRiskLow},
	"query":           {RiskLevel: RBACRiskLow},
	"count":           {RiskLevel: RBACRiskLow},
	"get-metric-data": {RiskLevel: RBACRiskLow},
	"get-metrics":     {RiskLevel: RBACRiskLow},
}

// HighRiskVerbs matches command verbs that imply destructive or irreversible actions.
var HighRiskVerbs = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(delete|terminate|destroy|drop|remove|rm|del)\b`),
}

// HighRiskPattern matches dangerous resource patterns in commands.
var HighRiskPattern = regexp.MustCompile(`(?i)(security.?group|subnet|vpc|database|instance|cluster|load.?balancer).*delete`)

// ImmutableConstraints are rules that can never be overridden.
// These are checked BEFORE trust level evaluation.
var ImmutableConstraints = []string{
	// Never auto-delete security groups referenced by active resources.
	"Never auto-delete security group with active ENIs",
	// Never auto-delete default VPC.
	"Never delete default VPC",
	// Never auto-delete the last remaining subnet in an AZ.
	"Never delete last subnet in an availability zone",
	// Never auto-delete a database with replication configured.
	"Never delete DB instance with read replicas",
}

// CheckPermission evaluates whether an operation is permitted based on
// trust level, operation type, and risk tier.
func CheckPermission(opType string, trustLevel string, score float64) RBACDecision {
	risk := RBACRiskHigh // default to high risk
	if perm, ok := DefaultOperationPermissions[opType]; ok {
		risk = perm.RiskLevel
	} else {
		// Fall back: check if the verb matches a high-risk pattern.
		for _, re := range HighRiskVerbs {
			if re.MatchString(opType) {
				risk = RBACRiskHigh
				break
			}
		}
	}

	decision := RBACDecision{
		Risk:   risk,
		Reason: fmt.Sprintf("operation %q has risk level %s", opType, risk),
	}

	// Check immutable constraints first.
	for _, constraint := range ImmutableConstraints {
		if matchesImmutableConstraint(opType, constraint) {
			decision.Allowed = false
			decision.ImmutableConstraints = append(decision.ImmutableConstraints, constraint)
			decision.Reason = fmt.Sprintf("immutable constraint violated: %s", constraint)
			return decision
		}
	}

	// Check if operation is marked immutable.
	if perm, ok := DefaultOperationPermissions[opType]; ok && perm.Immutable {
		decision.Allowed = false
		decision.Reason = fmt.Sprintf("operation %q is immutable — requires human approval", opType)
		return decision
	}

	// Special case: read-only low-risk operations are always allowed.
	if risk == RBACRiskLow {
		decision.Allowed = true
		decision.Reason = fmt.Sprintf("read-only operation %q (risk=low) auto-approved", opType)
		return decision
	}

	// Map trust level to max auto-approved risk.
	maxAuto := trustToMaxRisk(trustLevel, score)
	allowed := riskOrder(risk) <= riskOrder(maxAuto)
	decision.Allowed = allowed

	if !allowed {
		decision.Reason = fmt.Sprintf(
			"operation %q (risk=%s) exceeds max auto-approval %s for trust level %s (score=%.2f)",
			opType, risk, maxAuto, trustLevel, score,
		)
	}

	// Add approvers for high/critical operations.
	if risk == RBACRiskCritical || risk == RBACRiskHigh {
		decision.Approvers = []string{"human_review_required"}
	}

	return decision
}

// trustToMaxRisk maps a trust level + score to the maximum risk that can be
// auto-approved without human confirmation.
// L0_new can auto-approve none (requires approval for all), but read-only
// operations with risk=low bypass this gate explicitly in CheckPermission.
func trustToMaxRisk(level string, score float64) RBACRisk {
	switch level {
	case "L4_autonomous":
		return RBACRiskCritical
	case "L3_trusted":
		return RBACRiskHigh
	case "L2_established":
		return RBACRiskMedium
	case "L1_provisional":
		return RBACRiskLow
	default: // L0_new — auto-approve nothing except low-risk read-only (handled separately)
		return RBACRiskNone
	}
}

// riskOrder returns the numeric rank of a risk level.
func riskOrder(r RBACRisk) int {
	switch r {
	case RBACRiskNone:
		return 0
	case RBACRiskLow:
		return 1
	case RBACRiskMedium:
		return 2
	case RBACRiskHigh:
		return 3
	case RBACRiskCritical:
		return 4
	default:
		return 3
	}
}

// matchesImmutableConstraint checks if an operation violates an immutable constraint.
// The logic is intentionally simple; expand as new constraints are discovered.
func matchesImmutableConstraint(opType, constraint string) bool {
	// All delete/terminate/destroy operations are inherently immutable.
	if opType == "delete" || opType == "terminate" || opType == "destroy" || opType == "drop" {
		return true
	}
	// Map constraint text to operation checks.
	switch {
	case strings.Contains(constraint, "security group"):
		return opType == "delete-security-group"
	case strings.Contains(constraint, "default VPC"):
		return opType == "delete-vpc"
	case strings.Contains(constraint, "last subnet"):
		return opType == "delete-subnet"
	case strings.Contains(constraint, "read replicas"):
		return opType == "delete-database"
	}
	return false
}

// ValidateEnv checks that required environment variables are set.
// Returns missing vars; empty slice if all present.
func ValidateEnv(required []string) []string {
	var missing []string
	for _, env := range required {
		if v := getEnv(env); v == "" {
			missing = append(missing, env)
		}
	}
	return missing
}

// getEnv is a thin shim over os.Getenv, extracted for testability.
var getEnv = func(key string) string {
	// Implemented in rbac_test.go via monkey-patching.
	return _getEnv(key)
}

func _getEnv(key string) string {
	// This will be replaced in tests.
	return ""
}

// CheckCommandPermission evaluates a full command string for permission.
// It extracts the action verb from the command and evaluates it.
func CheckCommandPermission(command, trustLevel string, score float64) RBACDecision {
	// Extract the action (first word after skill name).
	// Command format: "hcloud <skill> <action> [args...]"
	parts := strings.Fields(command)
	var action string
	if len(parts) >= 3 {
		action = parts[2]
	}

	if action == "" {
		return RBACDecision{
			Allowed: false,
			Risk:    RBACRiskCritical,
			Reason:  "cannot determine action from command",
		}
	}

	return CheckPermission(action, trustLevel, score)
}

// HighRiskCommands is a precompiled list of command regexes that are
// considered high-risk regardless of their action verb.
var HighRiskCommands = []*regexp.Regexp{
	regexp.MustCompile(`(?i)--force|--delete|--destroy|--purge`),
}

// RequiresApproval checks if the command itself signals high risk
// (e.g., --force flag present) regardless of action verb.
func RequiresApproval(command string) bool {
	for _, re := range HighRiskCommands {
		if re.MatchString(command) {
			return true
		}
	}
	return false
}
