# Change Correlation — CES (Cloud Eye Service)

> **Purpose**: Correlate CTS change events with CES alarm anomalies for root cause analysis.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §Change Correlation
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. CTS Event Types → CES Alarm Mapping

| CTS Event Type | Resource Type | Change Impact | Associated Fault Type |
|----------------|---------------|---------------|----------------------|
| UpdateAlarm | CES Alarm Rule | Alarm rule configuration changed | Monitoring anomaly / Alert storm |
| CreateMetricData | CES Custom Metric | Metric data ingestion anomaly | Data gap / Incorrect thresholds |
| DeleteAlarm | CES Alarm Rule | Monitoring blind spot created | Unmonitored resource |
| UpdateNotification | CES Notification | Notification channel changed | Alert not delivered |
| UpdateAlarmAction | CES Alarm Action | Alarm action configuration changed | Auto-remediation failure |
| CreateAlarm | CES Alarm Rule | New monitoring coverage added | False positive from new rule |
| EnableAlarm | CES Alarm Rule | Previously disabled alarm enabled | Alert noise increase |
| DisableAlarm | CES Alarm Rule | Alarm suppressed | Detection gap |

### 1.1 Time Window Correlation

| Correlation Window | Use Case |
|--------------------|----------|
| 0-15 min before alarm | Immediate cause |
| 15-30 min before alarm | Contributing factor |
| 30-60 min before alarm | Indirect cause |
| 60-120 min before alarm | Background context |

---

## 2. Common Change-Triggered Alarm Patterns

### 2.1 Alarm Rule Change Triggers Monitoring Anomaly

```
Pattern: UpdateAlarm → Alarm triggers unexpectedly → Alert storm
Detection:
  - CTS event: UpdateAlarm
  - Time window: 0-15 min after rule change
  - Expected impact: Alarm threshold or evaluation period modified
  - Alert if: Alarm fires within 15 min of rule modification
```

### 2.2 Metric Data Anomaly Triggers Data Quality Alert

```
Pattern: CreateMetricData → Abnormal metric value → False alarm
Detection:
  - CTS event: CreateMetricData
  - Time window: 0-5 min after data ingestion
  - Expected impact: Custom metric receives anomalous data point
  - Alert if: Metric value > 3σ from rolling mean
```

### 2.3 Alarm Deletion Creates Monitoring Blind Spot

```
Pattern: DeleteAlarm → Resource becomes unmonitored → Detection gap
Detection:
  - CTS event: DeleteAlarm
  - Time window: 0-60 min after deletion
  - Expected impact: No active alarm for previously monitored resource
  - Alert if: Resource CPU/memory exceeds safe threshold with no active alarm
```

### 2.4 Notification Channel Change Causes Alert Delivery Failure

```
Pattern: UpdateNotification → Alerts not reaching recipients → On-call miss
Detection:
  - CTS event: UpdateNotification
  - Time window: 0-10 min after change
  - Expected impact: Notification topic ARN or endpoint modified
  - Alert if: Alarm fires but no notification delivered within SLA
```

### 2.5 Alarm Action Change Disrupts Auto-Remediation

```
Pattern: UpdateAlarmAction → Auto-remediation not triggered → Incident prolongs
Detection:
  - CTS event: UpdateAlarmAction
  - Time window: 0-30 min after change
  - Expected impact: Alarm action (SMS/webhook/SMN) configuration altered
  - Alert if: Alarm fires but remediation action not executed
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

# Query CTS for changes on this CES alarm rule
hcloud cts list-traces \
  --region "$REGION" \
  --resource_id "$RESOURCE_ID" \
  --start_time "$WINDOW_START" \
  --end_time "$WINDOW_END" \
  --output json
```

### 3.2 Query CES Alarm History

```bash
REGION="{{env.HW_REGION_ID}}"
ALARM_NAME="{{output.alarm_name}}"
FROM_TIME=$(date -d '7 days ago' +%Y-%m-%dT%H:%M:%SZ)

# List alarm history for correlation
hcloud ces list-alarm-history \
  --region "$REGION" \
  --alarm_name "$ALARM_NAME" \
  --start_time "$FROM_TIME" \
  --output json
```

### 3.3 Correlate Events with Alarm

```python
def correlate_change_with_alarm(alarm, cts_events):
    """
    Correlate CTS change events with alarm occurrence.
    Returns list of (event, correlation_score) tuples.
    """
    results = []
    for event in cts_events:
        time_delta = alarm.timestamp - event.timestamp

        # Calculate correlation score
        score = 0.0
        if 0 <= time_delta.minutes <= 15:
            score = 1.0  # High confidence: immediate cause
        elif 15 < time_delta.minutes <= 30:
            score = 0.7  # Medium confidence
        elif 30 < time_delta.minutes <= 60:
            score = 0.4  # Lower confidence
        elif 60 < time_delta.minutes <= 120:
            score = 0.2  # Background context

        results.append((event, score))

    return sorted(results, key=lambda x: x[1], reverse=True)
```

---

## 4. Change Correlation Workflow

```yaml
change_correlation:
  name: "CTS-Based Root Cause Correlation for CES"

  steps:
    - name: collect_alarm_context
      input: alarm_id
      output:
        - alarm_time
        - alarm_name
        - resource_id
        - metric_name

    - name: query_cts_events
      input: resource_id + alarm_time
      output: cts_events[]
      params:
        window_before: 60m
        window_after: 5m
        event_types:
          - UpdateAlarm
          - CreateMetricData
          - DeleteAlarm
          - UpdateNotification
          - UpdateAlarmAction

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

- [x] At least 5 CTS event types mapped to CES alarm impacts
- [x] Time window correlation rules defined (0-15/15-30/30-60/60-120 min)
- [x] At least 3 change-triggered alarm patterns documented
- [x] CTS query examples provided
- [x] Correlation workflow implemented
- [x] Product-specific event types (CES alarm operations) mapped
