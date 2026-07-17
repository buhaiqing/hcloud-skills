# Change Correlation — CBR

> **Purpose**: Correlate CTS change events with anomalies for root cause analysis.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §Change Correlation
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. CTS Event Types → Fault Type Mapping

| CTS Event Type | Resource Type | Change Impact | Associated Fault Type |
|----------------|---------------|---------------|---------------------|
| CreateBackup | CBR Backup | Backup created | Storage usage spike |
| RestoreBackup | CBR Backup | Data restored | Data inconsistency |
| DeleteBackup | CBR Backup | Backup deleted | Data loss risk |
| CreateVault | CBR Vault | New vault created | Billing increase |
| DeleteVault | CBR Vault | Vault deleted | Backup loss |
| ModifyVault | CBR Vault | Vault properties changed | Retention policy change |
| CreatePolicy | CBR Policy | Backup policy created | Scheduled backup change |
| UpdatePolicy | CBR Policy | Policy updated | Backup schedule change |
| ExecutePolicy | CBR Policy | Manual backup triggered | Storage I/O spike |
| BindResource | CBR Vault | Resource bound to vault | Backup scope change |

### 1.1 Time Window Correlation

| Correlation Window | Use Case |
|-------------------|----------|
| 0-15 min before alarm | Immediate cause |
| 15-30 min before alarm | Contributing factor |
| 30-60 min before alarm | Indirect cause |
| 60-120 min before alarm | Background context |

---

## 2. Common Change-Triggered Alarm Patterns

### 2.1 Backup Restore Triggers Data Inconsistency

```
Pattern: RestoreBackup → Old data rolled in → Alarm
Detection:
  - CTS event: RestoreBackup
  - Time window: 0-30 min after restore
  - Expected impact: Application reads old data while backup restores
  - Alert if: Data checksum mismatch or application error rate > 5%
```

### 2.2 Large Backup Triggers Storage Alert

```
Pattern: CreateBackup → Large backup → Alarm
Detection:
  - CTS event: CreateBackup
  - Time window: 0-60 min after backup starts
  - Expected impact: Storage consumption spike
  - Alert if: Vault storage > 85% after large backup
```

### 2.3 Vault Delete Triggers Backup Loss Alert

```
Pattern: DeleteVault → All backups lost → Alarm
Detection:
  - CTS event: DeleteVault
  - Time window: 0-5 min after deletion
  - Expected impact: Complete loss of backup history
  - Alert if: Any restore request succeeds from deleted vault
```

### 2.4 Policy Update Triggers Backup Schedule Change

```
Pattern: UpdatePolicy → Backup window changed → Alarm
Detection:
  - CTS event: UpdatePolicy
  - Time window: 0-15 min after update
  - Expected impact: Next backup at unexpected time
  - Alert if: Backup not found at expected time window
```

### 2.5 Backup Delete Triggers Retention Gap

```
Pattern: DeleteBackup → Specific backup removed → Alarm
Detection:
  - CTS event: DeleteBackup
  - Time window: 0-5 min after deletion
  - Expected impact: Point-in-time recovery gap
  - Alert if: RPO gap detected after backup deletion
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
  name: "CTS-Based Root Cause Correlation — CBR"
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
