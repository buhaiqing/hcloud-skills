# Change Correlation — OBS

> **Purpose**: Correlate CTS change events with anomalies for root cause analysis.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §Change Correlation
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. CTS Event Types → Fault Type Mapping

| CTS Event Type | Resource Type | Change Impact | Associated Fault Type |
|----------------|---------------|---------------|---------------------|
| CreateBucket | OBS Bucket | New bucket created | Configuration error |
| DeleteBucket | OBS Bucket | Bucket deleted | Data loss / service disruption |
| SetBucketPolicy | OBS Bucket | Bucket policy changed | Access denied |
| SetBucketLifecycle | OBS Bucket | Lifecycle rule changed | Unexpected object deletion |
| SetBucketCors | OBS Bucket | CORS policy changed | Cross-origin request blocked |
| SetBucketQuota | OBS Bucket | Bucket quota changed | Upload failures |
| PutObject | OBS Object | Object uploaded | Storage increase |
| DeleteObject | OBS Object | Object deleted | Data loss |
| CopyObject | OBS Object | Object copied | Storage increase |
| SetObjectMetadata | OBS Object | Metadata changed | Application misconfiguration |

### 1.1 Time Window Correlation

| Correlation Window | Use Case |
|-------------------|----------|
| 0-15 min before alarm | Immediate cause |
| 15-30 min before alarm | Contributing factor |
| 30-60 min before alarm | Indirect cause |
| 60-120 min before alarm | Background context |

---

## 2. Common Change-Triggered Alarm Patterns

### 2.1 Bucket Policy Change Triggers Access Denied

```
Pattern: SetBucketPolicy → Unauthorized access → Alarm
Detection:
  - CTS event: SetBucketPolicy
  - Time window: 0-5 min after change
  - Expected impact: Access permissions altered
  - Alert if: 403 error rate > 10% for > 5 min after policy change
```

### 2.2 Lifecycle Rule Triggers Mass Object Deletion

```
Pattern: SetBucketLifecycle → Expiration triggered → Alarm
Detection:
  - CTS event: SetBucketLifecycle
  - Time window: 0-60 min after rule evaluation
  - Expected impact: Large number of objects deleted
  - Alert if: Delete event count > threshold or storage drop > 50%
```

### 2.3 Bucket Delete Triggers Data Loss Alert

```
Pattern: DeleteBucket → All objects lost → Alarm
Detection:
  - CTS event: DeleteBucket
  - Time window: 0-5 min after deletion
  - Expected impact: Complete data loss
  - Alert if: Any request to deleted bucket succeeds (should fail)
```

### 2.4 CORS Change Triggers Request Failures

```
Pattern: SetBucketCors → CORS rule restricted → Alarm
Detection:
  - CTS event: SetBucketCors
  - Time window: 0-5 min after change
  - Expected impact: Cross-origin requests blocked
  - Alert if: CORS preflight failure rate > 20%
```

### 2.5 Quota Change Triggers Upload Failures

```
Pattern: SetBucketQuota → Quota reduced → Alarm
Detection:
  - CTS event: SetBucketQuota
  - Time window: 0-5 min after change
  - Expected impact: Upload rejections when quota exceeded
  - Alert if: 503 error rate > 5% after quota reduction
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
  name: "CTS-Based Root Cause Correlation — OBS"
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
