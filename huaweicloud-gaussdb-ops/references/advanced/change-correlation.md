# Change Correlation — GaussDB

> **Purpose**: Correlate CTS change events with anomalies for root cause analysis.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §Change Correlation
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. CTS Event Types → Fault Type Mapping

| CTS Event Type | Resource Type | Change Impact | Associated Fault Type |
|----------------|---------------|---------------|---------------------|
| CreateInstance | GaussDB Instance | New instance created | Connection refused |
| RebootInstance | GaussDB Instance | Instance restarted | Temporary service disruption |
| ResizeInstance | GaussDB Instance | Instance规格 changed | Performance degradation |
| RestoreInstance | GaussDB Instance | Data restored from backup | Data inconsistency |
| CreateDatabase | GaussDB Database | New database created | Permission/configuration issue |
| DeleteDatabase | GaussDB Database | Database removed | Application connection failure |
| CreateAccount | GaussDB Account | New account created | Authentication failure |
| ResetPassword | GaussDB Account | Password changed | Authentication failure |
| ModifySecurityGroup | Network | Security group changed | Connection blocked |
| ModifyParameterGroup | GaussDB Instance | Parameter group changed | Performance/configuration anomaly |

### 1.1 Time Window Correlation

| Correlation Window | Use Case |
|-------------------|----------|
| 0-15 min before alarm | Immediate cause |
| 15-30 min before alarm | Contributing factor |
| 30-60 min before alarm | Indirect cause |
| 60-120 min before alarm | Background context |

---

## 2. Common Change-Triggered Alarm Patterns

### 2.1 Instance Reboot Triggers Connection Timeout

```
Pattern: RebootInstance → Connection spike → Alarm
Detection:
  - CTS event: RebootInstance
  - Time window: 0-10 min after reboot
  - Expected behavior: Brief unavailability during restart
  - Alert if: Connection errors > 10% for > 5 min after restart
```

### 2.2 Instance Resize Triggers Performance Degradation

```
Pattern: ResizeInstance → CPU/memory reallocation → Alarm
Detection:
  - CTS event: ResizeInstance
  - Time window: 0-30 min after resize
  - Expected behavior: Temporary performance dip during reallocation
  - Alert if: CPU > 85% or memory > 85% sustained for > 15 min
```

### 2.3 Data Restore Triggers Query Latency

```
Pattern: RestoreInstance → Data roll-forward → Alarm
Detection:
  - CTS event: RestoreInstance
  - Time window: 0-60 min after restore
  - Expected impact: Increased I/O and query latency during replay
  - Alert if: Slow queries > 5% for > 30 min after restore
```

### 2.4 Password Reset Triggers Auth Failure Storm

```
Pattern: ResetPassword → Auth failures → Alarm
Detection:
  - CTS event: ResetPassword
  - Time window: 0-5 min after reset
  - Expected impact: Credential propagation delay
  - Alert if: Auth failure rate > 5% for > 2 min
```

### 2.5 Parameter Group Change Triggers Configuration Anomaly

```
Pattern: ModifyParameterGroup → Runtime parameter changed → Alarm
Detection:
  - CTS event: ModifyParameterGroup
  - Time window: 0-15 min after change
  - Expected impact: Connection pool / buffer changes
  - Alert if: Connection errors or memory usage anomaly detected
```

---

## 3. Correlation Query Examples

### 3.1 Query CTS Events Before Alarm

```bash
REGION="{{env.HW_REGION_ID}}"
RESOURCE_ID="{{output.resource_id}}"
ALARM_TIME="{{output.alarm_time}}"
WINDOW_START=$(date -d "$ALARM_TIME - 60 minutes" +%Y-%m-%dT%H:%M:%SZ)
WINDOW_END=$(date -d "$ALARM_TIME + 5 minutes" +%Y-%m-%dT%H:%M:%SZ)

hcloud cts list-traces \
  --region "$REGION" \
  --resource_id "$RESOURCE_ID" \
  --start_time "$WINDOW_START" \
  --end_time "$WINDOW_END" \
  --output json
```

### 3.2 Correlate Events with Alarm

```python
def correlate_change_with_alarm(alarm, cts_events):
    results = []
    for event in cts_events:
        time_delta = alarm.timestamp - event.timestamp
        score = 0.0
        if 0 <= time_delta.minutes <= 5:
            score = 1.0
        elif 5 < time_delta.minutes <= 15:
            score = 0.7
        elif 15 < time_delta.minutes <= 30:
            score = 0.4
        elif 30 < time_delta.minutes <= 60:
            score = 0.2
        results.append((event, score))
    return sorted(results, key=lambda x: x[1], reverse=True)
```

---

## 4. Change Correlation Workflow

```yaml
change_correlation:
  name: "CTS-Based Root Cause Correlation — GaussDB"
  steps:
    - name: collect_alarm_context
      input: alarm_id
      output:
        - alarm_time
        - resource_id
        - metric_name
        - threshold

    - name: query_cts_events
      input: resource_id + alarm_time
      output: cts_events[]
      params:
        window_before: 60m
        window_after: 5m

    - name: score_correlation
      input: cts_events + alarm
      output: correlated_events[]
      algorithm: time_distance_based_scoring

    - name: identify_root_cause
      input: correlated_events
      output: root_cause_event
      criteria: highest_score_event

    - name: generate_report
      input: root_cause_event + alarm
      output: correlation_report
```

---

## 5. Compliance Checklist

- [x] At least 5 change-triggered alarm patterns documented
- [x] Time window correlation rules defined
- [x] CTS query examples provided
- [x] Correlation workflow implemented
- [x] Product-specific event types mapped
