# CES Diagnosis Confidence — Huawei Cloud Cloud Eye Service

> **Purpose**: CES-specific implementation of diagnosis confidence scoring.
> **Extends**: `huaweicloud-skill-generator/references/diagnosis-confidence-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. CES-Specific Evidence Types

### 1.1 Direct Metric Anomalies

| Metric Namespace | Key Metrics | Anomaly Threshold | Weight |
|-----------------|-------------|-------------------|--------|
| `SYS.CES` | `alarm_status` | status = `alarm` | 0.30 |
| `SYS.CES` | `notification送达率` | rate < 0.99 | 0.25 |
| `SYS.CES` | `alarm_shield_count` | count > 0 | 0.20 |

### 1.2 CES-Specific Evidence Collection

```python
def collect_ces_evidence(alarm):
    evidence = []

    # Direct metric anomaly
    alarm_detail = hcloud ces describe-alarm --alarm-id alarm.id
    if alarm_detail.alarm_status == "alarm":
        evidence.append({
            "type": "direct_metric_anomaly",
            "metric": "alarm_status",
            "value": alarm_detail.alarm_status,
            "threshold": "normal",
            "weight": 0.30
        })

    # Notification delivery rate
    delivery_rate = query_ces_metric("notification送达率", alarm.resource_id)
    if delivery_rate < 0.99:
        evidence.append({
            "type": "direct_metric_anomaly",
            "metric": "notification送达率",
            "value": delivery_rate,
            "threshold": 0.99,
            "weight": 0.25
        })

    # Alarm shield active
    shield_count = alarm_detail.shield_count
    if shield_count > 0:
        evidence.append({
            "type": "direct_metric_anomaly",
            "metric": "alarm_shield_count",
            "value": shield_count,
            "threshold": 0,
            "weight": 0.20
        })

    return evidence
```

---

## 2. CES Confidence Levels

| Confidence | Level | Action |
|------------|-------|--------|
| **0.85–1.0** | High | 告警确实触发，通知异常，直接建议处理 |
| **0.60–0.85** | Medium | 可能通知异常，建议检查配置 |
| **0.30–0.60** | Low | 不确定，建议查看CES控制台 |
| **0–0.30** | Very Low | 数据不足，建议人工排查 |

---

## 3. CES-Specific Uncertainty Declaration

```yaml
uncertainty_declaration:
  confirmed:
    - "告警状态为触发"
    - "通知可能未送达"
  suspected:
    - "用户配置了告警屏蔽"
    - confidence: 0.75
  data_blindspots:
    - "LTS日志不可用时，无法获取详细错误信息"
    - "CTS变更历史不可用时，无法关联变更"
```

---

## 4. CES Diagnosis Examples

### 4.1 High Confidence Example

```yaml
diagnosis_report:
  alarm_id: "alarm-12345"
  resource_id: "ces-alarm-policy-xxxx"
  root_cause:
    category: "notification_failure"
    description: "告警通知送达失败"
    confidence: 0.92

  evidence:
    - type: "direct_metric_anomaly"
      metric: "notification送达率"
      value: 0.85
      threshold: 0.99
      weight_applied: 0.30
    - type: "log_evidence"
      source: "LTS"
      observation: "大量通知发送失败错误"
      weight_applied: 0.10

  recommended_action:
    - "检查SMN主题配置"
    - "验证订阅终端点"
    - "确认SMN服务状态"
```

### 4.2 Low Confidence Example

```yaml
diagnosis_report:
  alarm_id: "alarm-67890"
  resource_id: "ces-alarm-policy-yyyy"
  root_cause:
    category: "unknown"
    description: "无法确定根因"
    confidence: 0.28

  evidence:
    - type: "direct_metric_anomaly"
      metric: "alarm_status"
      value: "alarm"
      threshold: "normal"
      weight_applied: 0.30
    # 其他证据收集失败

  uncertainty_declaration:
    confirmed:
      - "告警状态为触发"
    suspected: []
    unverified:
      - "通知是否实际送达"
      - "用户是否配置了屏蔽规则"
    data_blindspots:
      - "LTS日志查询超时"
      - "CTS历史查询无权限"

  recommended_action:
    - "建议登录CES控制台查看详情"
    - "检查SMN配置"
    - "联系技术支持"
```

---

## 5. CES-Specific Remediation Actions

| Confidence Level | Action Type | Example |
|-----------------|-------------|---------|
| High (≥0.85) | Auto-fix | 重新启用告警、调整阈值 |
| Medium (0.60-0.85) | Suggest + Confirm | 建议检查SMN配置，等待用户确认 |
| Low (0.30-0.60) | Manual | 提供排查清单，用户自行处理 |
| Very Low (<0.30) | Escalate | 提交工单，人工介入 |

---

## 6. Implementation

```python
def ces_diagnose_with_confidence(alarm_id):
    alarm = get_alarm_detail(alarm_id)
    evidence = collect_ces_evidence(alarm)
    confidence = calculate_confidence(evidence)
    report = build_diagnosis_report(alarm, evidence, confidence)
    return report
```
