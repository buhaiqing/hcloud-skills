# Alarm Storm Handling — LTS

> **Purpose**: Guidelines for handling alarm storms in Huawei Cloud LTS.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

| Condition | Threshold | Detection Window |
|-----------|-----------|------------------|
| High frequency | > 10 alarms | 5 minutes |
| Wide impact | > 50% of log groups affected | 10 minutes |
| Cascading failure | Multiple services impacted | 5 minutes |

---

## 2. Suppression Rules

### 2.1 Aggregation

- Group identical `resource_id + metric` alarms in a 5-minute window
- Emit one notification per group with severity worst-of
- Include alarm count in notification content

### 2.2 Suppression Hierarchy

| Priority | Condition | Action |
|----------|-----------|--------|
| P0 | Upstream P0 alarm firing | Suppress all downstream alarms |
| P1 | > 50% log groups affected | Aggregate into single alarm |
| P2 | > 10 alarms in 5 min | Enable aggregation mode |
| P3 | Normal state | Process all alarms individually |

### 2.3 Cooldown Period

| Alarm Type | Cooldown |
|------------|----------|
| Resource pressure | 15 minutes |
| Trend anomaly | 30 minutes |
| Sudden change | 5 minutes |
| Correlation anomaly | 10 minutes |

---

## 3. Recovery Actions

### 3.1 Resource Pressure Alarm Storm

1. Identify most severely impacted log group
2. Check current quota usage
3. Execute storage cleanup or quota expansion
4. Verify alarm rate decreases after action

### 3.2 Ingestion Spike Alarm Storm

1. Identify source of spike (which log group/agent)
2. Check for legitimate burst vs anomalous traffic
3. If anomalous: investigate potential attack or misconfiguration
4. If legitimate: confirm scaling is appropriate

### 3.3 Query Latency Alarm Storm

1. Check LTS service health status
2. Identify affected log groups
3. Execute query optimization (partition, index)
4. If persistent: check for service degradation

---

## 4. Notification Templates

### 4.1 Aggregated Alarm

```
[LTS Alarm Storm Detected]
- Alarm Count: {count}
- Impacted Log Groups: {percentage}%
- Primary Pattern: {pattern_type}
- Severity: {max_severity}
- Time Window: {window_start} to {window_end}
- Recommended Action: {action}
```

### 4.2 Suppressed Alarm Notice

```
[Alarm Suppressed - Upstream P0 Active]
- Original Alarm: {alarm_type}
- Resource: {resource_id}
- Suppressed Until: {suppress_until}
- Root Alarm: {root_alarm_id}
```
