# Diagnosis Confidence Scoring — Template

> **Purpose**: Standardized confidence scoring for AI diagnosis results.
> **Usage**: Copy to skill's `references/advanced/diagnosis-confidence.md`,
>   customizing evidence types and weights for your product.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Confidence Calculation Model

```
Confidence = Σ(Evidence_i × Weight_i) / Σ(Weight_i)

Where Evidence sources and weights:
| Evidence Type       | Weight | Description                                           |
|---------------------|--------|-------------------------------------------------------|
| Direct metric anomaly | 0.30 | CES metric exceeds threshold, anomaly confirmed        |
| Correlated metrics  | 0.20 | Multi-metric joint anomaly, diagnosis confidence增强   |
| Change correlation  | 0.15 | Anomaly preceded by change, causal clue provided      |
| Knowledge base match | 0.15 | Historical fault pattern matched, experience evidence  |
| Dependency anomaly  | 0.10 | Downstream/upstream anomaly, propagation path          |
| Log evidence        | 0.10 | LTS logs show errors/exceptions, direct evidence      |
```

---

## 2. Confidence Levels & Actions

| Confidence | Level | Diagnosis Behavior | Report Wording |
|------------|-------|-------------------|----------------|
| **0.8–1.0** | High | Direct root cause + fix recommendation | "Root cause confirmed: ..." |
| **0.5–0.8** | Medium | Most likely root cause + alternatives + steps | "Most likely root cause: ... (confidence: XX%), suggested verification: ..." |
| **0.2–0.5** | Low | Multiple hypotheses + investigation steps | "Suspected root cause (needs verification): 1)... 2)... 3)..." |
| **0–0.2** | Very Low | Describe anomaly only + manual investigation | "Anomaly confirmed but root cause unknown, suggested manual investigation: ..." |

---

## 3. Uncertainty Declaration

All diagnosis reports **MUST** include uncertainty declaration:

```yaml
diagnosis_report:
  root_cause: "<primary cause>"
  confidence: 0.75
  uncertainty_declaration:
    confirmed:
      - "<definitely confirmed facts>"
    suspected:
      - "<suspected but not confirmed>"
      - confidence: 0.75
    unverified:
      - "<possible causes not yet investigated>"
  data_blindspots:
    - "<missing data sources if any>"
    - "<impact of missing data on confidence>"
```

---

## 4. Evidence Collection Checklist

| Evidence Type | Collection Method | Required | Weight |
|--------------|-------------------|----------|--------|
| Direct metric anomaly | CES query: `cpu_usage > 90%` | Yes | 0.30 |
| Correlated metrics | CES multi-metric query | Yes | 0.20 |
| Change correlation | CTS query: recent changes | Yes | 0.15 |
| Knowledge base match | knowledge-base.md lookup | Yes | 0.15 |
| Dependency anomaly | Delegate to dependency skill | No | 0.10 |
| Log evidence | LTS query: error logs | No | 0.10 |

---

## 5. Minimum Confidence Threshold

| Action Type | Minimum Confidence | Rationale |
|-------------|-------------------|-----------|
| Auto-remediation (low-risk) | 0.80 | High confidence needed for automatic action |
| Auto-remediation (medium-risk) | 0.90 | Higher threshold for riskier actions |
| Suggest fix to user | 0.50 | Medium confidence sufficient for suggestions |
| List hypotheses | 0.20 | Low confidence OK for investigation prompts |

---

## 6. Example Diagnosis Report

```yaml
diagnosis_report:
  timestamp: "2026-07-18T10:00:00Z"
  alarm_id: "alarm-xxxxx"
  resource_id: "ecs-xxxxx"

  root_cause:
    category: "resource_pressure"
    description: "CPU持续高导致实例响应变慢"
    confidence: 0.82

  evidence:
    - type: "direct_metric_anomaly"
      metric: "cpu_usage"
      value: 0.95
      threshold: 0.80
      weight_applied: 0.30
    - type: "correlated_metrics"
      metrics: ["mem_usedPercent", "diskUsage_percent"]
      observation: "内存和磁盘同时升高"
      weight_applied: 0.20
    - type: "change_correlation"
      change_type: "scale-out"
      time_delta: "10m before alarm"
      weight_applied: 0.15
    - type: "knowledge_base_match"
      pattern: "ECS-002 CPU spike after scaling"
      weight_applied: 0.15

  uncertainty_declaration:
    confirmed:
      - "CPU使用率超过阈值"
      - "内存使用同步升高"
    suspected:
      - "应用负载增加导致"
      - confidence: 0.82
    unverified:
      - "是否存在内存泄漏"
      - "应用代码问题"

  recommended_action:
    - type: "investigate"
      description: "检查应用负载和内存使用"
      confidence_required: 0.50
    - type: "auto_remediate"
      action: "如果CPU持续超过90%超过15分钟，自动扩容"
      confidence_required: 0.80
```

---

## 7. Implementation Notes

### 7.1 How to Use This Template

1. Copy to your skill's `references/advanced/diagnosis-confidence.md`
2. Adjust evidence types to match your product's metrics
3. Adjust weights based on your product's characteristics
4. Implement evidence collection functions in your skill's operations

### 7.2 Integration with Diagnosis Loop

```python
def diagnose_with_confidence(alarm):
    evidence = collect_evidence(alarm)
    confidence = calculate_confidence(evidence)
    report = build_diagnosis_report(alarm, evidence, confidence)
    return report
```

---

## 8. Compliance Checklist

- [ ] Confidence calculation formula documented
- [ ] Evidence types match product metrics
- [ ] All four confidence levels have clear actions
- [ ] Uncertainty declaration template included
- [ ] Minimum thresholds defined for auto-remediation
