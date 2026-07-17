# Change Correlation — DMS

> **Purpose**: Correlate CTS change events with anomalies for root cause analysis.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §Change Correlation
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. CTS Event Types → Fault Type Mapping

| CTS Event Type | Resource Type | Change Impact | Associated Fault Type |
|----------------|---------------|---------------|---------------------|
| CreateQueue | DMS Queue | New queue created | Configuration error |
| DeleteQueue | DMS Queue | Queue removed | Message delivery failure |
| UpdateQueue | DMS Queue | Queue properties changed | Consumer disruption |
| CreateConsumerGroup | DMS Consumer Group | Group created | Consumer rebalance |
| DeleteConsumerGroup | DMS Consumer Group | Group removed | Consumer rebalance |
| ResetConsumerGroup | DMS Consumer Group | Offset reset | Duplicate message delivery |
| ModifyInstance | DMS Instance | Instance changed | Performance anomaly |
| CreatePartition | DMS Queue | Partition added | Throughput change |
| DeletePartition | DMS Queue | Partition removed | Throughput change |

### 1.1 Time Window Correlation

| Correlation Window | Use Case |
|-------------------|----------|
| 0-15 min before alarm | Immediate cause |
| 15-30 min before alarm | Contributing factor |
| 30-60 min before alarm | Indirect cause |
| 60-120 min before alarm | Background context |

---

## 2. Common Change-Triggered Alarm Patterns

### 2.1 Queue Delete Triggers Message Delivery Failure

```
Pattern: DeleteQueue → Messages cannot be delivered → Alarm
Detection:
  - CTS event: DeleteQueue
  - Time window: 0-5 min after deletion
  - Expected impact: Produce failures, consumer errors
  - Alert if: Production failure rate > 5% after queue deletion
```

### 2.2 Queue Update Triggers Consumer Rebalance

```
Pattern: UpdateQueue → Consumer rejoin → Alarm
Detection:
  - CTS event: UpdateQueue
  - Time window: 0-15 min after update
  - Expected impact: Brief rebalancing of consumers
  - Alert if: Consumer lag > threshold for > 10 min
```

### 2.3 Consumer Group Reset Triggers Duplicate Delivery

```
Pattern: ResetConsumerGroup → Offset rewound → Alarm
Detection:
  - CTS event: ResetConsumerGroup
  - Time window: 0-30 min after reset
  - Expected impact: Duplicate message delivery
  - Alert if: Duplicate rate > 10% within 30 min after reset
```

### 2.4 Partition Change Triggers Throughput Drop

```
Pattern: DeletePartition → Reduced throughput → Alarm
Detection:
  - CTS event: DeletePartition
  - Time window: 0-15 min after deletion
  - Expected impact: Reduced parallel processing capacity
  - Alert if: Throughput drop > 30% for > 5 min
```

### 2.5 Instance Modification Triggers Latency Spike

```
Pattern: ModifyInstance → Instance reconfig → Alarm
Detection:
  - CTS event: ModifyInstance
  - Time window: 0-20 min after modification
  - Expected impact: Queue discovery and connection re-establishment
  - Alert if: Latency p99 > 500ms for > 10 min
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
  name: "CTS-Based Root Cause Correlation — DMS"
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
