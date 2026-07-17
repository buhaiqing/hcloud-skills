# SPEC: AIOps L5 — Autonomous Operations

> Version: 1.0.0
> Created: 2026-07-18
> Status: **DRAFT** — for review
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

| Component | Responsibility | State |
|-----------|---------------|-------|
| Detector | CES alarm monitoring, anomaly detection | Requires L3 |
| Diagnoser | Root cause analysis, confidence scoring | Requires L4 (Diagnosis Confidence) |
| Decider | Action selection, risk assessment, human approval gate | L5 New |
| Actor | Execute remediation via hcloud CLI / SDK | L5 New |
| Verifier | Verify action effectiveness, SLO impact | L5 New |
| Learner | Historical pattern mining, threshold optimization | L5 New |
| Knowledge Graph | Causal chain storage, incident memory | L5 New |
| Predictor | Failure prediction, capacity forecasting | Requires L3 Capacity Forecasting |

### 2.3 Data Flow

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

## 3. Self-Healing Closed-Loop

### 3.1 Definition

Self-healing closed-loop enables automatic remediation of detected anomalies without human intervention for routine failures. High-risk actions require human approval.

### 3.2 Action Classification

| Risk Level | Criteria | Example | Action |
|------------|----------|---------|--------|
| **Low** | No data loss, no cost impact, reversible | Restart process, clear cache | Auto-execute |
| **Medium** | Minor cost impact, low data risk | Scale up instance, adjust threshold | Auto-execute + notify |
| **High** | Significant cost, data risk, service impact | Delete resource, change security group | Human approval required |
| **Critical** | Irreversible, compliance-relevant | Drop table, disable multi-AZ | Manual only |

### 3.3 Pre-Approved Action Catalog

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

### 3.4 Closed-Loop Workflow

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

## 4. Self-Learning

### 4.1 Definition

Self-learning enables the system to improve from historical incidents, optimizing thresholds and patterns based on past outcomes.

### 4.2 Learning Sources

| Source | Data | Frequency |
|--------|------|-----------|
| Incident History | Alarm logs, diagnosis results, actions taken | Per incident |
| SLO Violations | Error budget consumption, burn rate | Monthly |
| Threshold Adjustments | Before/after alarm behavior | Weekly |
| Chaos Engineering Results | Resilience scores, failure modes | Quarterly |

### 4.3 Learning Algorithms

#### 4.3.1 Threshold Optimization

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

#### 4.3.2 Pattern Mining

```
# From incident history, extract:
1. Co-occurrence patterns: Which alarms frequently occur together?
2. Causal patterns: Which alarm precedes another?
3. Time patterns: When do certain alarms peak?
4. Resolution patterns: Which actions resolve which alarms?
```

### 4.4 Learning Workflow

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

## 5. Predictive Maintenance

### 5.1 Definition

Predictive maintenance forecasts potential failures before they occur, enabling proactive intervention.

### 5.2 Prediction Targets

| Target | Prediction Horizon | Required Data |
|--------|-------------------|---------------|
| CPU exhaustion | 7-30 days | CPU history + trend |
| Memory leak | 7-14 days | Memory usage pattern |
| Disk full | 14-30 days | Disk growth rate |
| Connection limit | 7-14 days | Connection usage pattern |
| Quota exhaustion | 30-90 days | Resource creation rate |
| Service outage | 1-24 hours | Multi-metric anomaly |

### 5.3 Prediction Models

#### 5.3.1 Linear Regression (Short-term)

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

#### 5.3.2 Seasonal Decomposition (Periodic)

```
y(t) = Trend(t) + Seasonal(t) + Residual(t)

Where:
- Trend = long-term growth direction
- Seasonal = periodic pattern (weekly/monthly)
- Residual = noise/anomalies
```

**Use case**: Load patterns with clear seasonality (business hours, weekly cycles)

#### 5.3.3 Anomaly Detection (Outlier-based)

```
z = (x - μ) / σ

Where:
- μ = historical mean
- σ = historical standard deviation
- z = standard score

Alert when: z > 3 (3-sigma rule) OR trend acceleration detected
```

**Use case**: Sudden changes, DDoS detection, traffic spikes

### 5.4 Prediction Output Schema

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

## 6. Root Cause Self-Discovery

### 6.1 Definition

Root cause self-discovery automatically builds causal graphs from incidents, enabling faster diagnosis of future problems.

### 6.2 Knowledge Graph Schema

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

### 6.3 Causal Discovery Algorithm

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

### 6.4 Knowledge Graph Update Flow

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

## 7. Implementation Phases

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

## 8. Acceptance Criteria

### 8.1 Self-Healing

| Criteria | Target | Measurement |
|----------|--------|-------------|
| Auto-resolution rate | ≥ 60% of low-risk incidents | % resolved without human |
| False positive rate | ≤ 10% | Incorrect remediations / total |
| MTTR improvement | ≥ 50% reduction | Mean time to resolve |
| Human approval accuracy | ≥ 95% | Correct approve/reject decisions |

### 8.2 Self-Learning

| Criteria | Target | Measurement |
|----------|--------|-------------|
| Threshold optimization coverage | ≥ 80% of alarms | Alarms with auto-tuned thresholds |
| Learning cycle | Weekly | Frequency of threshold updates |
| Pattern accuracy | ≥ 85% | Correctly predicted next alarm |
| False threshold adjustment | ≤ 5% | Overly aggressive tuning |

### 8.3 Predictive Maintenance

| Criteria | Target | Measurement |
|----------|--------|-------------|
| Prediction accuracy | ≥ 80% | Predictions that actually occurred |
| Prediction horizon | ≥ 7 days | Average advance warning |
| False positive rate | ≤ 20% | Predicted but didn't occur |
| Coverage | ≥ 60% of critical resources | Resources with active predictions |

### 8.4 Knowledge Graph

| Criteria | Target | Measurement |
|----------|--------|-------------|
| Root cause accuracy | ≥ 90% | Correct root cause in top-3 |
| Coverage | ≥ 70% of incidents | Incidents with causal chain |
| Query latency | ≤ 1 second | Per query |
| Graph update frequency | Real-time | Per incident resolution |

---

## 9. Out of Scope

- Physical infrastructure automation
- Compliance-critical changes (always manual)
- Zero-downtime deployment strategies
- Multi-cloud scenarios
- Cross-region coordination

---

## 10. Open Questions

1. **Knowledge Graph Storage**: Neo4j vs. PostgreSQL with graph extension?
2. **Human Approval UX**: How to surface approval requests efficiently?
3. **Rollback Strategy**: How to automatically rollback failed remediations?
4. **Learning Rate**: What α value for threshold optimization? (suggest 0.1)
5. **Prediction Model Selection**: Which models per metric type?

---

## 11. References

- `huaweicloud-skill-generator/references/aiops-best-practices.md` — L1-L4 spec
- `huaweicloud-ces-ops/references/advanced/self-healing.md` — existing self-healing pattern
- `huaweicloud-skill-generator/references/well-architected-assessment.md` §7 — Maturity Model
