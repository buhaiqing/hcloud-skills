# Decider Component Design — L5 Autonomous Operations

> **Purpose**: Decision engine that maps diagnosis results to executable action plans with risk assessment.
> **Extends**: `diagnosis-confidence-template.md` + `action-catalog.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Overview

The Decider is the "brain" of L5 autonomous operations. It receives diagnosis results with confidence scores and outputs an executable action plan.

```
┌─────────────────┐     ┌──────────────┐     ┌─────────────────┐
│  Diagnosis      │ ──▶ │   Decider    │ ──▶ │   Action Plan   │
│  Result         │     │              │     │   + Approval    │
│  + Confidence   │     │  Risk Assess │     │   Gate          │
└─────────────────┘     └──────────────┘     └─────────────────┘
```

### 1.1 Input Contract

```yaml
diagnosis_input:
  alarm_id: string
  resource_id: string
  skill: string                    # e.g. "ecs", "rds"
  root_cause:
    category: string               # e.g. "resource_pressure"
    description: string
    confidence: float              # 0.0 - 1.0
  evidence: list[Evidence]
  uncertainty_declaration: dict
  timestamp: string
```

### 1.2 Output Contract

```yaml
action_plan:
  plan_id: string                  # UUID for tracking
  alarm_id: string
  actions: list[Action]
  overall_risk: string             # Low/Medium/High/Critical
  requires_approval: bool
  approval_channel: string | null  # "slack" / "pagerduty" / "email" / null
  estimated_duration: int          # seconds
  preconditions_met: bool
  preconditions_check: list[dict]  # {check: string, passed: bool, error: string}
  dry_run_supported: bool
  rollback_available: bool
  timestamp: string
```

---

## 2. Decision Logic (Pseudocode)

```python
def decide(diagnosis_input, action_catalog):
    plan_id = generate_uuid()
    actions = []
    overall_risk = "Low"

    # Step 1: Get applicable actions for this skill + scenario
    applicable = match_actions(
        action_catalog,
        skill=diagnosis_input.skill,
        scenario=diagnosis_input.root_cause.category
    )

    if not applicable:
        return ActionPlan(
            plan_id=plan_id,
            status="no_matching_action",
            actions=[],
            requires_approval=True,
            approval_channel="slack",
            message="No auto-remediation available. Manual investigation required."
        )

    # Step 2: Score and rank actions by confidence-adjusted risk
    scored = []
    for action in applicable:
        risk_score = calculate_risk_score(action, diagnosis_input.confidence)
        scored.append((action, risk_score))

    scored.sort(key=lambda x: x[1], reverse=True)
    best_action = scored[0][0]

    # Step 3: Check preconditions
    preconditions_check = check_preconditions(best_action.preconditions)

    # Step 4: Determine approval requirement
    requires_approval = (
        best_action.risk_level in ("High", "Critical") or
        diagnosis_input.confidence < get_min_confidence(best_action) or
        not all(passed for p in preconditions_check)
    )

    # Step 5: Select approval channel
    approval_channel = None
    if requires_approval:
        approval_channel = get_approval_channel(best_action.risk_level)

    # Step 6: Build action plan
    actions.append(Action(
        action_id=best_action.id,
        description=best_action.action,
        risk_level=best_action.risk_level,
        auto_executable=not requires_approval,
        preconditions=preconditions_check,
        rollback=best_action.rollback,
        dry_run_command=best_action.dry_run_command
    ))

    overall_risk = best_action.risk_level

    return ActionPlan(
        plan_id=plan_id,
        alarm_id=diagnosis_input.alarm_id,
        actions=actions,
        overall_risk=overall_risk,
        requires_approval=requires_approval,
        approval_channel=approval_channel,
        preconditions_met=all(passed for p in preconditions_check),
        preconditions_check=preconditions_check,
        estimated_duration=best_action.estimated_duration,
        dry_run_supported=best_action.dry_run_supported,
        rollback_available=best_action.rollback != "Irreversible"
    )
```

---

## 3. Risk Score Calculation

```
RiskScore = BaseRisk × ConfidenceFactor × PreconditionFactor

Where:
  BaseRisk        = Risk level numeric (Low=1, Medium=2, High=3, Critical=4)
  ConfidenceFactor = 1.0 - diagnosis_input.confidence  # lower confidence = higher effective risk
  PreconditionFactor = 1.5 if any precondition failed else 1.0
```

### 3.1 Confidence-Adjusted Risk Threshold

| Action Risk | Confidence ≥ 0.8 | Confidence 0.5-0.8 | Confidence < 0.5 |
|-------------|------------------|--------------------|--------------------|
| Low | Auto-execute | Auto-execute | Auto-execute (warn) |
| Medium | Auto-execute | Auto-execute | Approval |
| High | Approval | Approval | Approval |
| Critical | Approval | Approval | Reject + Escalate |

---

## 4. Human Approval Gate

### 4.1 Approval Triggers

| Trigger Condition | Risk Modifier | Action |
|-------------------|---------------|--------|
| Risk = High | +1 level | Requires approval |
| Risk = Critical | +2 levels | Emergency approval only |
| Confidence < min threshold | +1 level | Requires approval |
| Any precondition failed | +1 level | Requires approval |
| Blast radius = Multi-AZ | +1 level | Requires approval |
| Data impact = Production write | +1 level | Requires approval |

### 4.2 Approval Channels

| Risk Level | Channel | Timeout | Escalation |
|------------|---------|---------|------------|
| High | Slack #ops-approval | 5 min | On-call |
| Critical | PagerDuty + SMS | 2 min | On-call + Manager |
| Emergency | Direct phone call | 1 min | Manager |

### 4.3 Approval Request Format

```yaml
approval_request:
  plan_id: string
  alarm_id: string
  resource: string
  action: string
  risk_level: string
  reason: string                    # Why this action is needed
  confidence: float                 # Diagnosis confidence
  blast_radius: string
  preconditions_status: string
  estimated_duration: string
  rollback_available: bool
  link_to_alarm: string             # CES alarm link
  requested_at: timestamp
  decision_required_by: timestamp   # Timeout
```

---

## 5. Precondition Checks

### 5.1 Common Preconditions

| Check | Command | Pass Criteria |
|-------|---------|---------------|
| quota_available | `hcloud <skill> listQuotas` | Unused quota > 10% |
| health_check_ok | `hcloud <skill> checkHealth <id>` | Status = healthy |
| backup_exists | `hcloud <skill> listBackups <id>` | Backup count > 0 |
| no_active_incident | CES alarm query | No open Critical/Major |
| dry_run_success | `hcloud <skill> <action> --dry-run` | Exit code = 0 |

### 5.2 Precondition Failure Handling

| Failure Type | Decider Action |
|--------------|----------------|
| Non-critical check failed | Log warning, proceed with caution |
| Critical check failed | Reject action, mark as "precondition_failed" |
| Check timeout | Reject action, escalate to human |
| All checks failed | Reject + manual investigation required |

---

## 6. Action Selection Priority

When multiple actions match a diagnosis:

1. **Highest confidence match** — prefer action with most historical success
2. **Lowest risk** — if same confidence, prefer lower risk
3. **Fastest resolution** — if same risk, prefer faster action
4. **Reversibility** — prefer reversible actions

### 6.1 Tie-Breaking Rules

```python
priority_score = (
    confidence_weight * historical_success_rate +
    risk_weight * (5 - risk_level_numeric) +      # Lower risk = higher score
    speed_weight * (1 / estimated_duration) +
    reversibility_weight * (1 if reversible else 0)
)
```

Default weights: confidence=0.4, risk=0.3, speed=0.15, reversibility=0.15

---

## 7. Decision Outcomes

| Outcome | Condition | Next Step |
|---------|-----------|-----------|
| **AUTO_EXECUTE** | Low/Medium risk + confidence OK + preconditions met | → Actor.execute() |
| **PENDING_APPROVAL** | High risk OR confidence below threshold | → Send to approval channel |
| **REJECTED** | Critical risk OR preconditions failed | → Log + escalate + alert |
| **NO_ACTION** | No matching action in catalog | → Manual investigation |
| **ESCALATE** | Decision timeout OR multiple failures | → Human on-call |

---

## 8. Logging & Audit

### 8.1 Decision Log Entry

```yaml
decision_log:
  plan_id: string
  decision_type: string             # AUTO_EXECUTE / PENDING_APPROVAL / etc
  diagnosis_input_hash: string      # SHA256 of input for traceability
  selected_action: string
  risk_assessment:
    base_risk: string
    confidence_factor: float
    preconditions_met: bool
    final_risk: string
  approval_requested: bool
  timestamp: string
  latency_ms: int                   # Decision time
```

---

## 9. Integration with Other Components

```
Decider receives:
  ├── diagnosis_result (from Diagnose component)
  ├── confidence_score (from Diagnose component)
  └── action_catalog (from action-catalog.md)

Decider outputs to:
  ├── Actor (for auto-execute path)
  ├── Human Approval Workflow (for approval path)
  └── Knowledge Base (for learning)
```

---

## 10. Compliance Checklist

- [ ] Decision logic covers all 4 risk levels
- [ ] Human approval triggers are explicit
- [ ] Approval channels defined for each risk level
- [ ] Precondition checks implemented
- [ ] Confidence-adjusted risk scoring documented
- [ ] Tie-breaking rules defined
- [ ] Decision logging format specified
- [ ] All decision outcomes handled
