# SPEC: AIOps L5 — Autonomous Operations

> Version: 1.0.0
> Created: 2026-07-18
> Status: **FINAL_SPEC** — 18 个 Batch 全部合并 main (2026-07-18)，2026-07-26 补充代码实现引用
> Target: hcloud-skills AIOps L5 (Autonomous)

## 1. Overview

### 1.1 Vision

L5 AIOps represents the fully autonomous operations layer: the system can **detect**, **diagnose**, **decide**, **act**, and **learn** with minimal human intervention. Humans approve high-risk actions; routine operations execute automatically.

### 1.2 Scope

This spec defines the technical architecture and implementation requirements for L5 AIOps capabilities:
1. **Self-Healing Closed-Loop** — diagnosis → decision → action → verification
2. **Self-Learning** — historical pattern learning, threshold optimization
3. **Predictive Maintenance** — failure prediction, proactive intervention
4. **Root Cause Self-Discovery** — causal chain mining, knowledge graph

### 1.3 Dependencies on L3/L4

L5 builds upon L3/L4 foundations:
- **L3 Required**: SLO/SLI, Change Correlation, Capacity Forecasting
- **L4 Required**: Chaos Engineering, Resilience Scoring, Diagnosis Confidence

### 1.4 Exclusions

- L5 does NOT include fully unsupervised actions without human approval for high-risk operations
- Physical infrastructure changes remain manual
- Compliance-critical changes require human sign-off

---

## 2. System Architecture

### 2.1 Autonomous Operations Loop

```
┌─────────────────────────────────────────────────────────────────┐
│                    L5 Autonomous Operations Loop                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌─────────┐ │
│  │  DETECT  │───▶│ DIAGNOSE │───▶│  DECIDE  │───▶│   ACT   │ │
│  └──────────┘    └──────────┘    └──────────┘    └─────────┘ │
│       │                                  │               │      │
│       │         ┌───────────────────────┘               │      │
│       │         ▼                                       ▼      │
│       │    ┌──────────┐                          ┌──────────┐ │
│       │    │  LEARN   │◀────────────────────────│ VERIFY   │ │
│       │    └──────────┘                          └──────────┘ │
│       │         │                                       │      │
│       │         ▼                                       │      │
│       │    ┌──────────┐                                │      │
│       │    │  KNOWLEDGE│◀───────────────────────────────┘      │
│       │    │   GRAPH   │ (因果链 + 历史模式)                      │
│       │    └──────────┘                                        │
│       │                                                          │
│       ▼                                                          │
│  ┌──────────┐                                                   │
│  │  PREDICT │ (预测性维护)                                        │
│  └──────────┘                                                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Components

| Component | Responsibility | State | Implementation |
|-----------|---------------|-------|---------------|
| Detector | CES alarm monitoring, anomaly detection | Requires L3 | `huaweicloud-ces-ops` skill; CES alarm rules via `hcloud ces alarm` CLI |
| Diagnoser | Root cause analysis, confidence scoring | Requires L4 (Diagnosis Confidence) | `internal/l4/orchestration.go` — `matchFaultSkills()`, `evaluateOperation()` |
| Decider | Action selection, risk assessment, human approval gate | L5 New | `internal/l4/orchestrator.go` — progressive trust scoring, risk-level gating |
| Actor | Execute remediation via hcloud CLI / SDK | L5 New | `internal/l4/trust.go` — trust-tier gating; `internal/gcl/runner.go` — Generator command execution |
| Verifier | Verify action effectiveness, SLO impact | L5 New | `internal/gcl/runner.go` — GCL cycle verification; `internal/critic/critic.go` — 5-dim scoring |
| Learner | Historical pattern mining, threshold optimization | L5 New | `internal/learning/trace.go` — trace aggregation; `internal/learning/knowledge.go` — failure pattern knowledge base |
| Knowledge Graph | Causal chain storage, incident memory | L5 New | `internal/l4/topology.go` — topology graph; `internal/learning/knowledge.go` — pattern knowledge |
| Predictor | Failure prediction, capacity forecasting | Requires L3 Capacity Forecasting | `internal/l4/predictive.go` — breach prediction, trend analysis |

### 2.3 CLI Integration

All L5 capabilities are accessible via the `hwcloud-skillcheck` CLI:

| CLI Subcommand | L5 Component | File |
|----------------|-------------|------|
| `l4 handle --fault <text>` | Closed-Loop Orchestrator (Decider + Actor + Verifier) | `cmd/l4.go` |
| `learning trace aggregate` | Learner — trace → failure pattern aggregation | `cmd/learning.go` |
| `learning trace learn` | Learner — single trace learning | `cmd/learning.go` |
| `learning trace report` | Learner — knowledge base report | `cmd/learning.go` |
| `learning gen` | Knowledge Base — regenerate patterns + playbooks | `cmd/learning.go` |
| `gcl run` | Actor — GCL execution loop | `cmd/gcl_run.go` |
| `critic score` | Verifier — 5-dimension quality scoring | `cmd/critic.go` |

### 2.4 Data Flow

```
CES Alarms ──▶ Detector ──▶ Diagnoser ──▶ Decider ──▶ Actor ──▶ Verifier
                                      │                    │
                                      │ (if high-risk)     │
                                      ▼                    ▼
                                  Human              Knowledge Graph
                                  Approval              (update)
                                      │                    │
                                      ▼                    ▼
                                  Decider ◀─────────── Learner
```

---

## 3. Implementation Status

### 3.1 Reference Documents (18 Batch, 25 files)

All L5 design documents have been created and merged to `main`:

| Phase | Batch | Document | Status |
|-------|-------|----------|--------|
| 1 Foundation | L5-A | `huaweicloud-skill-generator/references/action-catalog.md` | ✅ |
| 1 Foundation | L5-B | `huaweicloud-skill-generator/references/decider-design.md` | ✅ |
| 1 Foundation | L5-C | `huaweicloud-skill-generator/references/actor-framework.md` | ✅ |
| 2 Closed-Loop | L5-D | `huaweicloud-ces-ops/references/advanced/autonomous-loop.md` | ✅ |
| 2 Closed-Loop | L5-E | `huaweicloud-ecs-ops/references/advanced/auto-remediate.md` | ✅ |
| 2 Closed-Loop | L5-E | `huaweicloud-rds-ops/references/advanced/auto-remediate.md` | ✅ |
| 2 Closed-Loop | L5-E | `huaweicloud-dcs-ops/references/advanced/auto-remediate.md` | ✅ |
| 2 Closed-Loop | L5-F | `huaweicloud-skill-generator/references/human-approval-workflow.md` | ✅ |
| 2 Closed-Loop | L5-G | `huaweicloud-ces-ops/references/advanced/verification-logic.md` | ✅ |
| 3 Self-Learning | L5-H | `huaweicloud-skill-generator/references/self-learning-framework.md` | ✅ |
| 3 Self-Learning | L5-I | `huaweicloud-ces-ops/references/advanced/threshold-optimization.md` | ✅ |
| 3 Self-Learning | L5-J | `huaweicloud-skill-generator/references/pattern-mining.md` | ✅ |
| 4 Predictive | L5-K | `huaweicloud-skill-generator/references/prediction-models.md` | ✅ |
| 4 Predictive | L5-L | `huaweicloud-ces-ops/references/advanced/prediction-service.md` | ✅ |
| 4 Predictive | L5-M | `huaweicloud-ces-ops/references/advanced/prediction-alerts.md` | ✅ |
| 4 Predictive | L5-N | `huaweicloud-ces-ops/references/advanced/prediction-dashboard.md` | ✅ |
| 5 Knowledge Graph | L5-O | `huaweicloud-skill-generator/references/knowledge-graph-schema.md` | ✅ |
| 5 Knowledge Graph | L5-P | `huaweicloud-skill-generator/references/causal-discovery-algorithm.md` | ✅ |
| 5 Knowledge Graph | L5-Q | `huaweicloud-ces-ops/references/advanced/knowledge-graph.md` | ✅ |
| 5 Knowledge Graph | L5-R | `huaweicloud-ces-ops/references/advanced/causal-chain-update.md` | ✅ |

### 3.2 Go Runtime Implementation

| L5 Capability | Go Package | Key Functions |
|---------------|-----------|---------------|
| Closed-Loop Orchestrator | `internal/l4/orchestrator.go` | `HandleFault()` — detect → diagnose → decide → act → verify → learn |
| Dynamic Orchestration | `internal/l4/orchestration.go` | `matchFaultSkills()`, `evaluateOperation()`, `buildExecutionPlan()` |
| Progressive Trust | `internal/l4/trust.go` | Trust scoring (L0_new → L4_autonomous), risk-level gating |
| Topology Graph | `internal/l4/topology.go` | Resource blast radius, criticality scoring, cross-skill delegation |
| Predictive Maintenance | `internal/l4/predictive.go` | Breach prediction, trend analysis, threshold scanning |
| Learning Engine | `internal/learning/trace.go` | Trace aggregation, pattern learning |
| Knowledge Base | `internal/learning/knowledge.go` | Failure pattern management, playbook storage |
| GCL Runner | `internal/gcl/runner.go` | Generator → Critic → Loop → Trace execution |
| Critic Scoring | `internal/critic/critic.go` | 5-dimension (correctness, safety, idempotency, traceability, spec_compliance) |

### 3.3 CLI Surface

```bash
# L4/L5 Orchestrator — closed-loop fault handler
hwcloud-skillcheck l4 handle --fault "RDS CPU > 90%" --risk medium

# Learning — trace aggregation and pattern mining
hwcloud-skillcheck learning trace aggregate --skill huaweicloud-ecs-ops --since-hours 168
hwcloud-skillcheck learning trace learn --skill huaweicloud-ecs-ops --trace audit-results/gcl-trace-*.json
hwcloud-skillcheck learning trace report --skill huaweicloud-ecs-ops

# Knowledge base — regenerate patterns and playbooks
hwcloud-skillcheck learning gen

# GCL execution loop (Actor + Verifier)
hwcloud-skillcheck gcl run --skill huaweicloud-billing-ops --command 'hcloud ...' --structural-critic-only

# Critic scoring (Verifier)
hwcloud-skillcheck critic score --generator /path/to/generator-trace.json
```

---

### 4.1 Definition

Self-healing closed-loop enables automatic remediation of detected anomalies without human intervention for routine failures. High-risk actions require human approval.

### 4.2 Action Classification

| Risk Level | Criteria | Example | Action |
|------------|----------|---------|--------|
| **Low** | No data loss, no cost impact, reversible | Restart process, clear cache | Auto-execute |
| **Medium** | Minor cost impact, low data risk | Scale up instance, adjust threshold | Auto-execute + notify |
| **High** | Significant cost, data risk, service impact | Delete resource, change security group | Human approval required |
| **Critical** | Irreversible, compliance-relevant | Drop table, disable multi-AZ | Manual only |

### 4.3 Pre-Approved Action Catalog

#### Low-Risk (Auto-Execute)

| Scenario | Action | Skill |
|----------|--------|-------|
| Alarm disabled after deployment | Re-enable alarm | CES |
| Threshold too sensitive | Auto-adjust to P95+10% | CES |
| Cache exhaustion | Clear cache | Redis/DCS |
| Connection pool饱和 | Connection pool reset | RDS |

#### Medium-Risk (Auto + Notify)

| Scenario | Action | Skill |
|----------|--------|-------|
| CPU持续高 | Scale up instance | ECS |
| 磁盘即将满 | Expand disk | ECS/EVS |
| 内存泄漏 | Restart instance | ECS |
| 负载高触发扩容 | Execute AS scaling | ECS/CCE |

#### High-Risk (Human Approval)

| Scenario | Action | Skill |
|----------|--------|-------|
| 重启生产实例 | Instance reboot | ECS |
| 安全组变更 | Security group rule change | VPC |
| 数据库主备切换 | RDS failover | RDS |

#### Critical (Manual Only)

| Scenario | Action | Skill |
|----------|--------|-------|
| 删除数据 | Data deletion | RDS/OBS |
| 关闭多AZ | Disable multi-AZ | RDS |
| 删除EIP | EIP release | VPC |

### 4.4 Closed-Loop Workflow

```yaml
closed_loop:
  name: "Auto-Heal Workflow"
  trigger:
    - alarm_severity: critical
    - alarm_severity: high
    - diagnosis_confidence: ">= 0.8"

  steps:
    - name: detect
      component: Detector
      output: alarm_event

    - name: diagnose
      component: Diagnoser
      input: alarm_event
      output: diagnosis_result
      confidence_threshold: 0.6

    - name: decide
      component: Decider
      input: diagnosis_result
      output: action_plan
      risk_classification: automatic

    - name: act
      component: Actor
      input: action_plan
      output: action_result
      dry_run: false  # Set true for first-time actions

    - name: verify
      component: Verifier
      input: action_result
      output: verification_result
      slo_impact_check: true

    - name: learn
      component: Learner
      input: verification_result
      output: updated_knowledge
```

---

## 5. Self-Learning

### 5.1 Definition

Self-learning enables the system to improve from historical incidents, optimizing thresholds and patterns based on past outcomes.

### 5.2 Learning Sources

| Source | Data | Frequency |
|--------|------|-----------|
| Incident History | Alarm logs, diagnosis results, actions taken | Per incident |
| SLO Violations | Error budget consumption, burn rate | Monthly |
| Threshold Adjustments | Before/after alarm behavior | Weekly |
| Chaos Engineering Results | Resilience scores, failure modes | Quarterly |

### 5.3 Learning Algorithms

#### 5.3.1 Threshold Optimization

```
New_Threshold = α × Historical_P95 + (1-α) × Current_Threshold

Where:
- α = learning_rate (0.0 ~ 1.0)
- Historical_P95 = 95th percentile of metric over past 30 days
- Current_Threshold = currently configured alarm threshold
```

**Constraints**:
- New threshold must be within ±20% of current threshold (prevent wild swings)
- Only learn from stable periods (no incidents in past 7 days)
- Minimum 30 data points required

#### 5.3.2 Pattern Mining

```
# From incident history, extract:
1. Co-occurrence patterns: Which alarms frequently occur together?
2. Causal patterns: Which alarm precedes another?
3. Time patterns: When do certain alarms peak?
4. Resolution patterns: Which actions resolve which alarms?
```

### 5.4 Learning Workflow

```yaml
learning:
  name: "Self-Learning Workflow"
  schedule: "weekly"

  inputs:
    - incident_history: LTS log query
    - alarm_thresholds: CES alarm config
    - action_outcomes: skill execution logs
    - chaos_results: resilience scores

  process:
    - name: analyze_incidents
      algorithm: pattern_mining
      output: incident_patterns

    - name: optimize_thresholds
      algorithm: threshold_optimization
      input: incident_patterns + alarm_thresholds
      output: recommended_thresholds

    - name: validate_recommendations
      method: simulation
      output: validated_recommendations

    - name: apply_recommendations
      condition: confidence >= 0.8 AND risk_level == low
      action: auto_apply
      else: human_review
```

---

## 6. Predictive Maintenance

### 6.1 Definition

Predictive maintenance forecasts potential failures before they occur, enabling proactive intervention.

### 6.2 Prediction Targets

| Target | Prediction Horizon | Required Data |
|--------|-------------------|---------------|
| CPU exhaustion | 7-30 days | CPU history + trend |
| Memory leak | 7-14 days | Memory usage pattern |
| Disk full | 14-30 days | Disk growth rate |
| Connection limit | 7-14 days | Connection usage pattern |
| Quota exhaustion | 30-90 days | Resource creation rate |
| Service outage | 1-24 hours | Multi-metric anomaly |

### 6.3 Prediction Models

#### 6.3.1 Linear Regression (Short-term)

```
y = mx + b

Where:
- y = predicted metric value
- m = slope (growth rate)
- x = time
- b = current value

Exhaustion_Date = (Quota_Limit - Current_Value) / m
```

**Use case**: Stable, linear growth patterns (disk usage, connection count)

#### 6.3.2 Seasonal Decomposition (Periodic)

```
y(t) = Trend(t) + Seasonal(t) + Residual(t)

Where:
- Trend = long-term growth direction
- Seasonal = periodic pattern (weekly/monthly)
- Residual = noise/anomalies
```

**Use case**: Load patterns with clear seasonality (business hours, weekly cycles)

#### 6.3.3 Anomaly Detection (Outlier-based)

```
z = (x - μ) / σ

Where:
- μ = historical mean
- σ = historical standard deviation
- z = standard score

Alert when: z > 3 (3-sigma rule) OR trend acceleration detected
```

**Use case**: Sudden changes, DDoS detection, traffic spikes

### 6.4 Prediction Output Schema

```yaml
prediction:
  resource_id: "ecs-xxxxx"
  metric: "cpu_usage"
  predicted_value: 0.95  # 95% at prediction_date
  prediction_date: "2026-08-15"
  confidence: 0.85
  horizon_days: 30
  risk_level: "critical"  # critical/high/medium/low
  recommended_action:
    - type: "scale_up"
      target: "ecs.xlarge"
      estimated_cost: "¥500/month"
  created_at: "2026-07-18T10:00:00Z"
```

---

## 7. Root Cause Self-Discovery

### 7.1 Definition

Root cause self-discovery automatically builds causal graphs from incidents, enabling faster diagnosis of future problems.

### 7.2 Knowledge Graph Schema

```yaml
knowledge_graph:
  node_types:
    - name: "alarm"
      properties:
        - alarm_id
        - alarm_name
        - severity
        - metric
        - threshold
        - resource_id

    - name: "change"
      properties:
        - change_id
        - change_type
        - resource_id
        - timestamp
        - actor

    - name: "symptom"
      properties:
        - symptom_id
        - description
        - impact
        - duration

    - name: "root_cause"
      properties:
        - cause_id
        - category
        - description
        - fix_action

  edge_types:
    - name: "causes"
      from: "root_cause"
      to: "symptom"

    - name: "triggers"
      from: "change"
      to: "alarm"

    - name: "correlates_with"
      from: "alarm"
      to: "alarm"

    - name: "resolves"
      from: "change"
      to: "symptom"
```

### 7.3 Causal Discovery Algorithm

```python
def discover_causal_chain(incident):
    # Step 1: Find correlated alarms
    correlated_alarms = find_correlated_alarms(incident.alarm, time_window="30min")

    # Step 2: Find preceding changes
    preceding_changes = find_changes(
        resource=incident.resource_id,
        time_range=(incident.time - 30min, incident.time)
    )

    # Step 3: Build candidate causal graph
    causal_graph = build_graph(correlated_alarms, preceding_changes)

    # Step 4: Score causal paths
    scored_paths = score_paths(causal_graph, historical_incidents)

    # Step 5: Return highest-confidence root cause
    return scored_paths[0]  # Highest confidence path
```

### 7.4 Knowledge Graph Update Flow

```
Incident Resolved
      │
      ▼
┌─────────────┐
│ Extract     │
│ Root Cause  │
└─────────────┘
      │
      ▼
┌─────────────┐     ┌─────────────────┐
│ Update      │────▶│ Knowledge Graph │
│ Causal Path │     │ (Neo4j/etc)     │
└─────────────┘     └─────────────────┘
      │
      ▼
┌─────────────┐
│ Propagate to│
│ Similar     │
│ Incidents   │
└─────────────┘
```

---

## 8. Implementation Phases

### Phase 1: Foundation (Weeks 1-4)

| Task | Deliverable | Dependency |
|------|-------------|------------|
| Action Catalog | Pre-approved action list per skill | L3/L4 complete |
| Risk Classification | Risk level matrix | None |
| Decider Component | Decision logic for action selection | Action Catalog |
| Actor Enhancement | Safe action execution framework | L3/L4 skills |

### Phase 2: Closed-Loop (Weeks 5-8)

| Task | Deliverable | Dependency |
|------|-------------|------------|
| Closed-Loop Framework | Detect → Diagnose → Act → Verify loop | Phase 1 |
| Low-Risk Auto-Execute | Automatic remediation for low-risk | Phase 1 |
| Verification Logic | SLO impact check after actions | L3 SLO/SLI |
| Human Approval Gate | High-risk action approval workflow | Phase 1 |

### Phase 3: Self-Learning (Weeks 9-12)

| Task | Deliverable | Dependency |
|------|-------------|------------|
| Learning Framework | Historical data ingestion | LTS integration |
| Threshold Optimization | Automated threshold tuning | Phase 2 |
| Pattern Mining | Co-occurrence, causal pattern extraction | Phase 2 |
| Learning Validation | Simulation before apply | None |

### Phase 4: Predictive Maintenance (Weeks 13-16)

| Task | Deliverable | Dependency |
|------|-------------|------------|
| Prediction Models | Linear/Seasonal/Anomaly models | L3 Capacity Forecasting |
| Prediction API | Prediction service endpoints | None |
| Alert Integration | Push predictions to CES | Prediction Models |
| Dashboard | Prediction summary view | Prediction API |

### Phase 5: Knowledge Graph (Weeks 17-20)

| Task | Deliverable | Dependency |
|------|-------------|------------|
| Graph Schema | Node/Edge type definitions | None |
| Causal Discovery | Algorithm implementation | Phase 2 |
| Graph Storage | Neo4j or equivalent | Graph Schema |
| Query Interface | Root cause lookup by symptom | Graph Storage |

---

## 9. Acceptance Criteria

### 9.1 Self-Healing

| Criteria | Target | Measurement |
|----------|--------|-------------|
| Auto-resolution rate | ≥ 60% of low-risk incidents | % resolved without human |
| False positive rate | ≤ 10% | Incorrect remediations / total |
| MTTR improvement | ≥ 50% reduction | Mean time to resolve |
| Human approval accuracy | ≥ 95% | Correct approve/reject decisions |

### 9.2 Self-Learning

| Criteria | Target | Measurement |
|----------|--------|-------------|
| Threshold optimization coverage | ≥ 80% of alarms | Alarms with auto-tuned thresholds |
| Learning cycle | Weekly | Frequency of threshold updates |
| Pattern accuracy | ≥ 85% | Correctly predicted next alarm |
| False threshold adjustment | ≤ 5% | Overly aggressive tuning |

### 9.3 Predictive Maintenance

| Criteria | Target | Measurement |
|----------|--------|-------------|
| Prediction accuracy | ≥ 80% | Predictions that actually occurred |
| Prediction horizon | ≥ 7 days | Average advance warning |
| False positive rate | ≤ 20% | Predicted but didn't occur |
| Coverage | ≥ 60% of critical resources | Resources with active predictions |

### 9.4 Knowledge Graph

| Criteria | Target | Measurement |
|----------|--------|-------------|
| Root cause accuracy | ≥ 90% | Correct root cause in top-3 |
| Coverage | ≥ 70% of incidents | Incidents with causal chain |
| Query latency | ≤ 1 second | Per query |
| Graph update frequency | Real-time | Per incident resolution |

---

## 10. Out of Scope

- Physical infrastructure automation
- Compliance-critical changes (always manual)
- Zero-downtime deployment strategies
- Multi-cloud scenarios
- Cross-region coordination

---

## 11. Open Questions & Resolutions

> **Status**: All questions resolved in current implementation.

| # | Question | Resolution | Rationale |
|---|----------|-----------|-----------|
| 1 | **Knowledge Graph Storage**: Neo4j vs. PostgreSQL? | **In-memory topology graph** (`internal/l4/topology.go`) + JSON persistence. No external DB. | L5 scope is single-repo operational intelligence; external DB adds deployment complexity with no immediate benefit. The topology graph parses `references/integration.md` delegation tables at runtime. |
| 2 | **Human Approval UX**: How to surface approval requests efficiently? | **Trust-tier gating** (`internal/l4/trust.go`): `human_review_required` flag in orchestrator output. Destructive ops on high-criticality resources with low trust auto-escalate. | Integrated into the existing `l4 handle` CLI output rather than building a separate notification system. |
| 3 | **Rollback Strategy**: How to auto-rollback failed remediations? | **Per-step verification in GCL loop** (`internal/gcl/runner.go`). Failed steps → `RETRY` or `SAFETY_FAIL` (non-retryable). No automatic rollback; escalation to human for destructive reversals. | Automatic rollback is itself a destructive operation. Safety-first: verify → fail → notify. |
| 4 | **Learning Rate α**: What α value for threshold optimization? | **α = 0.1** (as suggested). Configurable via `assets/remediation-playbooks.json` per-skill settings. | Conservative rate prevents wild threshold swings; 0.1 means 90% weight on current threshold, 10% on historical P95. |
| 5 | **Prediction Model Selection**: Which models per metric type? | **Linear regression** for monotonic growth (disk, connections); **z-score anomaly** for sudden changes (CPU spikes). Implemented in `internal/l4/predictive.go`. | Seasonal decomposition deferred — most hcloud metrics lack clear weekly seasonality patterns. |

---

## 12. References

- `huaweicloud-skill-generator/references/aiops-best-practices.md` — L1-L4 spec
- `huaweicloud-ces-ops/references/advanced/self-healing.md` — existing self-healing pattern
- `huaweicloud-skill-generator/references/well-architected-assessment.md` §7 — Maturity Model
- `docs/superpowers/plans/aiops-l5-autonomous.md` — 18 Batch plan (COMPLETE)
- `docs/superpowers/specs/aiops-optimization.md` — L4 maturity spec (prerequisite)
- `hwcloud-skillcheck/internal/l4/` — Go runtime implementation of L5 components
- `hwcloud-skillcheck/internal/learning/` — Learning engine implementation
- `hwcloud-skillcheck/cmd/l4.go` — CLI entry point for L5 operations
