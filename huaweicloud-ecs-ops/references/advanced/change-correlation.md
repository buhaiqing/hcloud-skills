# Change Correlation — ECS

> **Purpose**: Correlate CTS change events with anomalies for root cause analysis.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §Change Correlation
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. CTS Event Types → Fault Type Mapping

| CTS Event Type | Resource Type | Change Impact | Associated Fault Type |
|----------------|---------------|---------------|---------------------|
| CreateServer | ECS Instance | New resource created | Resource not reachable |
| DeleteServer | ECS Instance | Resource removed | Service disruption |
| ResizeServer | ECS Instance | Instance size changed | Performance degradation |
| RebootServer | ECS Instance | Instance restarted | Temporary outage |
| StartServer | ECS Instance | Instance started | Resource not reachable |
| StopServer | ECS Instance | Instance stopped | Service disruption |
| AttachSecurityGroup | Network | Security group changed | Connection blocked/allowed |
| DetachSecurityGroup | Network | Security group removed | Connection blocked |
| ResetPassword | ECS Instance | Password changed | Authentication failure |
| UpdateServerMetadata | ECS Instance | Metadata changed | Configuration-related anomaly |
| AttachVolume | ECS Instance | Volume attached | Disk space issue |
| DetachVolume | ECS Instance | Volume detached | Disk space issue |

### 1.1 Time Window Correlation

| Correlation Window | Use Case |
|-------------------|----------|
| 0-15 min before alarm | Immediate cause |
| 15-30 min before alarm | Contributing factor |
| 30-60 min before alarm | Indirect cause |
| 60-120 min before alarm | Background context |

---

## 2. Common Change-Triggered Alarm Patterns

### 2.1 Scale-Out Triggers CPU Alert

```
Pattern: ResizeServer (scale-up) → CPU spike → Alarm
Detection:
  - CTS event: ResizeServer
  - Time window: 0-10 min after scale-out
  - Expected behavior: CPU temporarily high during initialization
  - Alert if: CPU > 90% for > 15 min (not just initialization spike)
```

### 2.2 Security Group Change Triggers Connection Timeout

```
Pattern: Security group rule change → Connection failures → Alarm
Detection:
  - CTS event: AttachSecurityGroup or DetachSecurityGroup
  - Time window: 0-5 min after change
  - Expected impact: Specific port/IP blocked or opened
  - Alert if: Connection errors on affected port
```

### 2.3 Instance Restart Triggers Health Check Failure

```
Pattern: RebootServer → Health check failure → Alarm
Detection:
  - CTS event: RebootServer or StartServer or StopServer
  - Time window: 0-10 min after action
  - Expected behavior: Brief unavailability during boot/shutdown
  - Alert if: Health check fails for > 5 min after action
```

### 2.4 Configuration Change Triggers Performance Alert

```
Pattern: UpdateServerMetadata → Performance change → Alarm
Detection:
  - CTS event: UpdateServerMetadata
  - Time window: 0-30 min after change
  - Expected impact: Depends on change type
  - Alert if: Performance degrades beyond threshold
```

### 2.5 Volume Attach/Detach Triggers Disk Alert

```
Pattern: AttachVolume → Disk usage change → Alarm
Detection:
  - CTS event: AttachVolume or DetachVolume
  - Time window: 0-5 min after change
  - Expected impact: Disk capacity change
  - Alert if: Disk usage > 85% or < 15% after change
```

### 2.6 Password Reset Triggers Auth Failure

```
Pattern: ResetPassword → Auth failures → Alarm
Detection:
  - CTS event: ResetPassword
  - Time window: 0-5 min after reset
  - Expected impact: Brief auth failures during credential propagation
  - Alert if: Auth failure rate > 5% for > 2 min
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

# Query CTS for changes on this ECS instance
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
