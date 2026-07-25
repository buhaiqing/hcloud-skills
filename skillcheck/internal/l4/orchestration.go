// Package l4 contains the L4 runtime engines: orchestration, topology,
// predictive, trust, plus the closed-loop orchestrator that wires them.
// Each engine is a thin port of the matching scripts/*.py file.
//
// All engines return pure Go structs; no I/O happens inside the engine
// itself. The orchestrator (orchestrator.go) owns file + subprocess concerns.
package l4

import (
	"sort"
	"strings"
)

// SkillCap describes a single skill's domain, capabilities, and delegation edges.
type SkillCap struct {
	Domain       string
	Capabilities []string
	DelegatesTo  []string
}

// SkillCapabilities is the static registry of skill → capability mapping.
var SkillCapabilities = map[string]SkillCap{
	"huaweicloud-ecs-ops":     {Domain: "compute", Capabilities: []string{"instance_lifecycle", "diagnostics", "resize", "migration"}, DelegatesTo: []string{"huaweicloud-vpc-ops", "huaweicloud-ces-ops", "huaweicloud-elb-ops"}},
	"huaweicloud-vpc-ops":     {Domain: "network", Capabilities: []string{"connectivity", "security_group", "route_table", "subnet"}, DelegatesTo: []string{"huaweicloud-ecs-ops", "huaweicloud-elb-ops"}},
	"huaweicloud-ces-ops":     {Domain: "monitoring", Capabilities: []string{"metrics_query", "alarm_management", "dashboard"}},
	"huaweicloud-elb-ops":     {Domain: "network", Capabilities: []string{"health_check", "backend_manage", "listener"}, DelegatesTo: []string{"huaweicloud-ecs-ops", "huaweicloud-vpc-ops"}},
	"huaweicloud-rds-ops":     {Domain: "database", Capabilities: []string{"instance_lifecycle", "backup", "performance", "parameter"}, DelegatesTo: []string{"huaweicloud-ecs-ops", "huaweicloud-ces-ops", "huaweicloud-vpc-ops"}},
	"huaweicloud-iam-ops":     {Domain: "identity", Capabilities: []string{"permission_check", "policy_manage", "credential_rotate"}},
	"huaweicloud-cbr-ops":     {Domain: "backup", Capabilities: []string{"backup_create", "restore", "policy_manage"}, DelegatesTo: []string{"huaweicloud-ecs-ops", "huaweicloud-rds-ops"}},
	"huaweicloud-dns-ops":     {Domain: "network", Capabilities: []string{"record_manage", "zone_manage", "health_check"}, DelegatesTo: []string{"huaweicloud-vpc-ops"}},
	"huaweicloud-billing-ops": {Domain: "cost", Capabilities: []string{"cost_analysis", "budget_alert", "right_sizing"}},
	"huaweicloud-cts-ops":     {Domain: "audit", Capabilities: []string{"event_query", "trail_manage", "compliance_check"}},
}

// FaultRule is a fault keyword → required capabilities mapping.
type FaultRule struct {
	Pattern        []string
	RequiredCaps   []string
	PrioritySkills []string
}

// FaultRules is the static list of routing rules.
var FaultRules = []FaultRule{
	{Pattern: []string{"unreachable", "connectivity", "timeout", "connection refused"}, RequiredCaps: []string{"connectivity", "diagnostics"}, PrioritySkills: []string{"huaweicloud-vpc-ops", "huaweicloud-ecs-ops"}},
	{Pattern: []string{"disk full", "storage", "no space", "capacity"}, RequiredCaps: []string{"instance_lifecycle", "metrics_query"}, PrioritySkills: []string{"huaweicloud-ecs-ops", "huaweicloud-ces-ops"}},
	{Pattern: []string{"permission denied", "403", "access denied", "unauthorized"}, RequiredCaps: []string{"permission_check"}, PrioritySkills: []string{"huaweicloud-iam-ops"}},
	{Pattern: []string{"high cpu", "high memory", "performance", "slow"}, RequiredCaps: []string{"metrics_query", "diagnostics"}, PrioritySkills: []string{"huaweicloud-ces-ops", "huaweicloud-ecs-ops"}},
	{Pattern: []string{"database", "rds", "connection pool", "deadlock"}, RequiredCaps: []string{"instance_lifecycle", "performance"}, PrioritySkills: []string{"huaweicloud-rds-ops", "huaweicloud-ces-ops"}},
	{Pattern: []string{"backup", "restore", "recovery", "data loss"}, RequiredCaps: []string{"backup_create", "restore"}, PrioritySkills: []string{"huaweicloud-cbr-ops"}},
	{Pattern: []string{"dns", "resolve", "domain"}, RequiredCaps: []string{"record_manage", "health_check"}, PrioritySkills: []string{"huaweicloud-dns-ops", "huaweicloud-vpc-ops"}},
	{Pattern: []string{"cost", "billing", "budget", "overspend"}, RequiredCaps: []string{"cost_analysis"}, PrioritySkills: []string{"huaweicloud-billing-ops"}},
}

// MatchedSkill is the public return shape from MatchFaultSkills.
type MatchedSkill struct {
	Skill           string
	Confidence      float64
	Domain          string
	Capabilities    []string
	MatchedKeywords []string
}

// MatchFaultSkills scores every skill against the fault using the static
// rules. Returns matches sorted by descending confidence, deduped by skill.
func MatchFaultSkills(fault string, available []string) []MatchedSkill {
	faultLower := strings.ToLower(fault)
	avail := map[string]bool{}
	for _, s := range available {
		avail[s] = true
	}
	haveAvail := len(available) > 0
	var matched []MatchedSkill
	for _, rule := range FaultRules {
		score := 0
		var kws []string
		for _, kw := range rule.Pattern {
			if strings.Contains(faultLower, kw) {
				score++
				kws = append(kws, kw)
			}
		}
		if score == 0 {
			continue
		}
		conf := float64(score) / float64(len(rule.Pattern))
		if conf > 1.0 {
			conf = 1.0
		}
		for _, skill := range rule.PrioritySkills {
			if haveAvail && !avail[skill] {
				continue
			}
			cap, ok := SkillCapabilities[skill]
			if !ok {
				continue
			}
			matched = append(matched, MatchedSkill{
				Skill:           skill,
				Confidence:      round2(conf),
				Domain:          cap.Domain,
				Capabilities:    cap.Capabilities,
				MatchedKeywords: kws,
			})
		}
	}
	best := map[string]MatchedSkill{}
	for _, m := range matched {
		if cur, ok := best[m.Skill]; !ok || m.Confidence > cur.Confidence {
			best[m.Skill] = m
		}
	}
	out := make([]MatchedSkill, 0, len(best))
	for _, m := range best {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Confidence > out[j].Confidence })
	return out
}

// DiscoverTransitiveSkills does a BFS through SkillCapabilities.DelegatesTo
// from the given primary skills.
func DiscoverTransitiveSkills(primary []string) []string {
	discovered := map[string]bool{}
	q := append([]string{}, primary...)
	for _, s := range primary {
		discovered[s] = true
	}
	for len(q) > 0 {
		skill := q[0]
		q = q[1:]
		cap, ok := SkillCapabilities[skill]
		if !ok {
			continue
		}
		for _, d := range cap.DelegatesTo {
			if !discovered[d] {
				discovered[d] = true
				q = append(q, d)
			}
		}
	}
	out := make([]string, 0, len(discovered))
	for s := range discovered {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// SelectStrategy mirrors the Python policy: single → sequential, with deps →
// pipeline, ≤3 → parallel, >3 → fan_out_collect.
func SelectStrategy(skillCount int, hasDependency bool) string {
	if skillCount == 1 {
		return "sequential"
	}
	if hasDependency {
		return "pipeline"
	}
	if skillCount <= 3 {
		return "parallel"
	}
	return "fan_out_collect"
}

// PlanStep is a single step in an execution plan.
type PlanStep struct {
	Step           int     `json:"step"`
	Skill          string  `json:"skill"`
	SkillShort     string  `json:"skill_short"`
	Action         string  `json:"action"`
	DependsOn      []int   `json:"depends_on"`
	Confidence     float64 `json:"confidence"`
	TimeoutSeconds int     `json:"timeout_seconds"`
}

// ExecutionPlan is the result of BuildExecutionPlan.
type ExecutionPlan struct {
	PlanID                 string     `json:"plan_id"`
	CreatedAt              string     `json:"created_at"`
	FaultDescription       string     `json:"fault_description"`
	Strategy               string     `json:"strategy"`
	TotalSkills            int        `json:"total_skills"`
	Steps                  []PlanStep `json:"steps"`
	RollbackPolicy         string     `json:"rollback_policy"`
	MaxTotalTimeoutSeconds int        `json:"max_total_timeout_seconds"`
	Status                 string     `json:"status"`
}

// BuildExecutionPlan constructs the ordered plan from matched skills + strategy.
func BuildExecutionPlan(fault string, skills []MatchedSkill, strategy string) *ExecutionPlan {
	planID := "orch-" + randomHex(12)
	steps := []PlanStep{}

	if strategy == "pipeline" {
		order := map[string]int{"monitoring": 0, "identity": 1, "network": 2, "compute": 3, "database": 4, "backup": 5, "cost": 6, "audit": 7}
		ordered := append([]MatchedSkill{}, skills...)
		sort.SliceStable(ordered, func(i, j int) bool { return order[ordered[i].Domain] < order[ordered[j].Domain] })
		for i, s := range ordered {
			deps := []int{}
			if i > 0 {
				deps = []int{i}
			}
			steps = append(steps, PlanStep{
				Step:           i + 1,
				Skill:          s.Skill,
				SkillShort:     skillShort(s.Skill),
				Action:         "diagnose_and_remediate",
				DependsOn:      deps,
				Confidence:     s.Confidence,
				TimeoutSeconds: 300,
			})
		}
	} else if strategy == "parallel" {
		for i, s := range skills {
			steps = append(steps, PlanStep{
				Step:           i + 1,
				Skill:          s.Skill,
				SkillShort:     skillShort(s.Skill),
				Action:         "diagnose_and_remediate",
				DependsOn:      []int{},
				Confidence:     s.Confidence,
				TimeoutSeconds: 300,
			})
		}
	} else {
		for i, s := range skills {
			deps := []int{}
			if i > 0 && strategy == "sequential" {
				deps = []int{i}
			}
			steps = append(steps, PlanStep{
				Step:           i + 1,
				Skill:          s.Skill,
				SkillShort:     skillShort(s.Skill),
				Action:         "diagnose_and_remediate",
				DependsOn:      deps,
				Confidence:     s.Confidence,
				TimeoutSeconds: 300,
			})
		}
	}

	totalTimeout := 0
	for _, s := range steps {
		totalTimeout += s.TimeoutSeconds
	}
	return &ExecutionPlan{
		PlanID:                 planID,
		CreatedAt:              NowISO(),
		FaultDescription:       fault,
		Strategy:               strategy,
		TotalSkills:            len(skills),
		Steps:                  steps,
		RollbackPolicy:         "reverse_order",
		MaxTotalTimeoutSeconds: totalTimeout,
		Status:                 "pending",
	}
}

func skillShort(skill string) string {
	s := strings.TrimPrefix(skill, "huaweicloud-")
	s = strings.TrimSuffix(s, "-ops")
	return s
}
