# Change Correlation — RDS

> **Purpose**: Correlate CTS change events with anomalies for root cause analysis.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §Change Correlation
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. CTS Event Types → Fault Type Mapping

| CTS Event Type | Resource Type | Change Impact | Associated Fault Type |
|----------------|---------------|---------------|---------------------|
| CreateInstance | RDS Instance | New instance created | Instance unreachable |
| DeleteInstance | RDS Instance | Instance removed | Service disruption |
| RestartInstance | RDS Instance | Instance restarted | Temporary outage |
| ResizeInstance | RDS Instance | Instance size changed | Performance degradation |
| BackupInstance | RDS Instance | Backup started | I/O performance impact |
| RestoreInstance | RDS Instance | Data restored | Data inconsistency risk |
| CreateDatabase | RDS Database | Database created | Resource configuration issue |
| DeleteDatabase | RDS Database | Database removed | Service disruption |
| CreateAccount | RDS Account | Account created | Authentication failure |
| DeleteAccount | RDS Account | Account removed | Authentication failure |
| GrantPrivilege | RDS Account | Permissions changed | Security/connectivity issue |
| RevokePrivilege | RDS Account | Permissions removed | Authentication/authorization failure |
| UpdateSecurityGroup | RDS Security Group | Security group changed | Connection blocked/allowed |
| ModifyParameterGroup | RDS Parameters | Parameters changed | Performance degradation |
| CreateReadReplica | RDS Replica | Read replica created | Replication lag |
| DeleteReadReplica | RDS Replica | Read replica removed | Query load increase |

### 1.1 Time Window Correlation

| Correlation Window | Use Case |
|-------------------|----------|
| 0-15 min before alarm | Immediate cause |
| 15-30 min before alarm | Contributing factor |
| 30-60 min before alarm | Indirect cause |
| 60-120 min before alarm | Background context |

---

## 2. Common Change-Triggered Alarm Patterns

### 2.1 Instance Restart Triggers Connection Spike

```
Pattern: RestartInstance → DB restart → Connection pool reconnect → Alarm
Detection:
  - CTS event: RestartInstance
  - Time window: 0-15 min after restart
  - Expected behavior: Brief connection failures during restart
  - Alert if: Connection errors > 100 or connection latency > 5s for > 5 min
```

### 2.2 Instance Resize Triggers Performance Alert

```
Pattern: ResizeInstance → New resource specs → Performance shift → Alarm
Detection:
  - CTS event: ResizeInstance
  - Time window: 0-30 min after resize
  - Expected behavior: Performance normalization after resize completes
  - Alert if: QPS < baseline * 0.8 or latency > baseline * 1.5
```

### 2.3 Backup Start Triggers I/O Alert

```
Pattern: BackupInstance → I/O pressure → Query latency → Alarm
Detection:
  - CTS event: BackupInstance
  - Time window: 0-60 min during backup (varies by DB size)
  - Expected impact: I/O usage increase, query latency increase
  - Alert if: I/O utilization > 80% or query latency > 500ms
```

### 2.4 Parameter Group Change Triggers Query Performance Alert

```
Pattern: ModifyParameterGroup → Parameter applied → Performance change → Alarm
Detection:
  - CTS event: ModifyParameterGroup
  - Time window: 0-30 min after change
  - Expected impact: Depends on parameter changed
  - Alert if: Performance degrades or error rate increases
```

### 2.5 Security Group Change Triggers Connection Timeout

```
Pattern: UpdateSecurityGroup → Rules changed → Connection blocked → Alarm
Detection:
  - CTS event: UpdateSecurityGroup
  - Time window: 0-5 min after change
  - Expected impact: Specific IP/port blocked or allowed
  - Alert if: Connection failures on affected port
```

### 2.6 Read Replica Deletion Triggers Replication Lag

```
Pattern: DeleteReadReplica → Replica removed → Query load shift → Alarm
Detection:
  - CTS event: DeleteReadReplica
  - Time window: 0-10 min after deletion
  - Expected impact: Write node sees increased query load
  - Alert if: Replication lag > 10s or CPU > 85%
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

# Query CTS for changes on this RDS instance
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
