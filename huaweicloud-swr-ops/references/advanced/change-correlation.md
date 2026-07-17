# Change Correlation — SWR

> **Purpose**: Correlate CTS change events with anomalies for root cause analysis.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §Change Correlation
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. CTS Event Types → Fault Type Mapping

| CTS Event Type | Resource Type | Change Impact | Associated Fault Type |
|----------------|---------------|---------------|---------------------|
| CreateOrganization | SWR Organization | New org created | Access configuration |
| CreateRepository | SWR Repository | New repo created | Image not found |
| DeleteRepository | SWR Repository | Repo deleted | Image pull failure |
| UpdateRepository | SWR Repository | Repo updated | Image tag configuration |
| CreateAccessDomain | SWR Repo Access | Access granted | Security exposure |
| DeleteAccessDomain | SWR Repo Access | Access revoked | Application outage |
| UpdateRepositoryPermission | SWR Permission | Permission changed | Access denied |
| CreateTrigger | SWR Trigger | Trigger created | Unexpected builds |
| DeleteTrigger | SWR Trigger | Trigger deleted | CI/CD pipeline broken |
| UpdateTrigger | SWR Trigger | Trigger updated | Build schedule change |

### 1.1 Time Window Correlation

| Correlation Window | Use Case |
|-------------------|----------|
| 0-15 min before alarm | Immediate cause |
| 15-30 min before alarm | Contributing factor |
| 30-60 min before alarm | Indirect cause |
| 60-120 min before alarm | Background context |

---

## 2. Common Change-Triggered Alarm Patterns

### 2.1 Repository Delete Triggers Image Pull Failure

```
Pattern: DeleteRepository → Image not found → Alarm
Detection:
  - CTS event: DeleteRepository
  - Time window: 0-5 min after deletion
  - Expected impact: Pod/image pull failures in K8s
  - Alert if: Image pull failure rate > 10% after repo deletion
```

### 2.2 Access Domain Remove Triggers Permission Denied

```
Pattern: DeleteAccessDomain → Access revoked → Alarm
Detection:
  - CTS event: DeleteAccessDomain
  - Time window: 0-5 min after removal
  - Expected impact: Specific user/service access denied
  - Alert if: 403 error rate > 20% for affected users
```

### 2.3 Repository Update Triggers Image Tag Mismatch

```
Pattern: UpdateRepository → Tag changed → Alarm
Detection:
  - CTS event: UpdateRepository
  - Time window: 0-15 min after update
  - Expected impact: Old tag no longer points to expected image
  - Alert if: Deployment uses old tag that returns different image
```

### 2.4 Trigger Delete Breaks CI/CD Pipeline

```
Pattern: DeleteTrigger → Build not triggered → Alarm
Detection:
  - CTS event: DeleteTrigger
  - Time window: 0-60 min after deletion
  - Expected impact: No new images built on code push
  - Alert if: Build count drops to 0 for > 2 build cycles
```

### 2.5 Permission Update Triggers Access Denied

```
Pattern: UpdateRepositoryPermission → Scope changed → Alarm
Detection:
  - CTS event: UpdateRepositoryPermission
  - Time window: 0-5 min after update
  - Expected impact: Broader or narrower access granted
  - Alert if: 403 error spike or security exposure detected
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
  name: "CTS-Based Root Cause Correlation — SWR"
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
