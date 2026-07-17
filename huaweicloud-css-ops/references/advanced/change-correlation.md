# Change Correlation — CSS

> **Purpose**: Correlate CTS change events with anomalies for root cause analysis.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §Change Correlation
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. CTS Event Types → Fault Type Mapping

| CTS Event Type | Resource Type | Change Impact | Associated Fault Type |
|----------------|---------------|---------------|---------------------|
| CreateCluster | CSS Cluster | New cluster created | Cluster not reachable |
| RestartCluster | CSS Cluster | Cluster restarted | Temporary outage |
| UpgradeCluster | CSS Cluster | Version upgraded | Version compatibility |
| CreateSnapshot | CSS Snapshot | Snapshot created | Storage usage increase |
| DeleteSnapshot | CSS Snapshot | Snapshot deleted | Data loss risk |
| RestoreSnapshot | CSS Snapshot | Snapshot restored | Data inconsistency |
| AddNode | CSS Node | Node added | Shard rebalancing |
| DeleteNode | CSS Node | Node removed | Shard rebalancing |
| ModifyCluster | CSS Cluster | Cluster config changed | Performance anomaly |
| CreateIndex | CSS Index | Index created | Storage/cpu spike |

### 1.1 Time Window Correlation

| Correlation Window | Use Case |
|-------------------|----------|
| 0-15 min before alarm | Immediate cause |
| 15-30 min before alarm | Contributing factor |
| 30-60 min before alarm | Indirect cause |
| 60-120 min before alarm | Background context |

---

## 2. Common Change-Triggered Alarm Patterns

### 2.1 Cluster Restart Triggers Red Health Status

```
Pattern: RestartCluster → Replica recovery → Alarm
Detection:
  - CTS event: RestartCluster
  - Time window: 0-15 min after restart
  - Expected behavior: Brief red cluster status during recovery
  - Alert if: Cluster status red for > 10 min after restart
```

### 2.2 Cluster Upgrade Triggers JVM GC Pause

```
Pattern: UpgradeCluster → New version deployed → Alarm
Detection:
  - CTS event: UpgradeCluster
  - Time window: 0-30 min after upgrade
  - Expected impact: GC pauses and heap reallocation
  - Alert if: JVM GC pause > 500ms for > 15 min after upgrade
```

### 2.3 Node Add/Remove Triggers Shard Rebalance

```
Pattern: AddNode → Shard rebalance → Alarm
Detection:
  - CTS event: AddNode or DeleteNode
  - Time window: 0-30 min after change
  - Expected impact: Increased indexing load and network traffic
  - Alert if: CPU > 85% or disk I/O > 80% for > 20 min during rebalance
```

### 2.4 Snapshot Restore Triggers Data Inconsistency

```
Pattern: RestoreSnapshot → Old data rolled in → Alarm
Detection:
  - CTS event: RestoreSnapshot
  - Time window: 0-60 min after restore
  - Expected impact: Index version mismatch and query divergence
  - Alert if: Query error rate > 5% after restore
```

### 2.5 Index Creation Triggers Storage Spike

```
Pattern: CreateIndex → New shards allocated → Alarm
Detection:
  - CTS event: CreateIndex
  - Time window: 0-30 min after index creation
  - Expected impact: Increased disk usage and CPU
  - Alert if: Disk usage > 85% within 30 min of index creation
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
  name: "CTS-Based Root Cause Correlation — CSS"
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
