# Change Correlation — CCE

> **Purpose**: Correlate CTS change events with anomalies for root cause analysis.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §Change Correlation
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. CTS Event Types → Fault Type Mapping

| CTS Event Type | Resource Type | Change Impact | Associated Fault Type |
|----------------|---------------|---------------|---------------------|
| CreateCluster | CCE Cluster | New cluster created | Cluster unreachable |
| DeleteCluster | CCE Cluster | Cluster removed | Service disruption |
| UpdateCluster | CCE Cluster | Cluster config changed | Configuration-related anomaly |
| ScaleCluster | CCE Cluster | Node pool scaled | Performance degradation |
| CreateNodePool | CCE NodePool | Node pool added | Resource not reachable |
| DeleteNodePool | CCE NodePool | Node pool removed | Pod scheduling failure |
| ScaleNodePool | CCE NodePool | Node count changed | Resource capacity change |
| AddNode | CCE Node | Node added | Resource not reachable |
| DeleteNode | CCE Node | Node removed | Pod eviction/recreation |
| UpdateNodePool | CCE NodePool | Node pool config changed | Configuration-related anomaly |
| CreateWorkload | CCE Workload | Workload deployed | Resource not reachable |
| DeleteWorkload | CCE Workload | Workload removed | Service disruption |
| UpdateWorkload | CCE Workload | Workload spec changed | Performance degradation |
| ScaleWorkload | CCE Workload | Replica count changed | Service capacity change |
| RestartWorkload | CCE Workload | Pods restarted | Temporary outage |

### 1.1 Time Window Correlation

| Correlation Window | Use Case |
|-------------------|----------|
| 0-15 min before alarm | Immediate cause |
| 15-30 min before alarm | Contributing factor |
| 30-60 min before alarm | Indirect cause |
| 60-120 min before alarm | Background context |

---

## 2. Common Change-Triggered Alarm Patterns

### 2.1 Node Pool Scale-Out Triggers CPU Alert

```
Pattern: ScaleNodePool → New nodes joining → CPU spike → Alarm
Detection:
  - CTS event: ScaleNodePool or AddNode
  - Time window: 0-15 min after scale-out
  - Expected behavior: CPU temporarily high during pod scheduling
  - Alert if: CPU > 90% for > 15 min after scale-out completes
```

### 2.2 Workload Scale-Up Triggers Memory Alert

```
Pattern: ScaleWorkload → More replicas → Memory usage increase → Alarm
Detection:
  - CTS event: ScaleWorkload (replicas increase)
  - Time window: 0-10 min after scale-up
  - Expected behavior: Memory usage increases proportionally
  - Alert if: Memory > 85% after scale-up stabilizes
```

### 2.3 Cluster Config Change Triggers API Server Alert

```
Pattern: UpdateCluster → API server config changed → Latency → Alarm
Detection:
  - CTS event: UpdateCluster
  - Time window: 0-5 min after change
  - Expected impact: API server behavior change
  - Alert if: API latency > 500ms or error rate > 1%
```

### 2.4 Node Deletion Triggers Pod Eviction Alert

```
Pattern: DeleteNode → Pods evicted → Service degradation → Alarm
Detection:
  - CTS event: DeleteNode
  - Time window: 0-10 min after deletion
  - Expected behavior: Pods rescheduled to other nodes
  - Alert if: Pod restart count > threshold or service unavailable > 2 min
```

### 2.5 Workload Update Triggers CrashLoopBackOff

```
Pattern: UpdateWorkload → New pods failing → Crash → Alarm
Detection:
  - CTS event: UpdateWorkload
  - Time window: 0-15 min after update
  - Expected impact: Rolling update with brief unavailability
  - Alert if: CrashLoopBackOff detected or error rate > 10%
```

### 2.6 Node Pool Deletion Triggers Resource Pressure

```
Pattern: DeleteNodePool → Nodes removed → Resource pressure → Alarm
Detection:
  - CTS event: DeleteNodePool
  - Time window: 0-20 min after deletion
  - Expected impact: Pods evicted, resource freed
  - Alert if: Pending pods > 0 or CPU/memory pressure on remaining nodes
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

# Query CTS for changes on this CCE cluster
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
    """
    Correlate CTS change events with alarm occurrence.
    Returns list of (event, correlation_score) tuples.
    """
    results = []
    for event in cts_events:
        time_delta = alarm.timestamp - event.timestamp

        # Calculate correlation score
        score = 0.0
        if 0 <= time_delta.minutes <= 5:
            score = 1.0  # High confidence: immediate cause
        elif 5 < time_delta.minutes <= 15:
            score = 0.7  # Medium confidence
        elif 15 < time_delta.minutes <= 30:
            score = 0.4  # Lower confidence
        elif 30 < time_delta.minutes <= 60:
            score = 0.2  # Background context

        results.append((event, score))

    return sorted(results, key=lambda x: x[1], reverse=True)
```

---

## 4. Change Correlation Workflow

```yaml
change_correlation:
  name: "CTS-Based Root Cause Correlation"

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
