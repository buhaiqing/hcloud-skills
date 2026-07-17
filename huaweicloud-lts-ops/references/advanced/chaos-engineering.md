# Chaos Engineering — LTS

> **Purpose**: Document fault injection experiments for LTS resilience verification.
> **Extends**: `huaweicloud-skill-generator/references/chaos-engineering-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Fault Injection Experiment Design

| Experiment Type | Injection Method | Observed Metrics | Expected Behavior | Termination Condition |
|----------------|-----------------|-----------------|------------------|---------------------|
| Log group deletion | Delete LTS log group via API | Log ingestion rate, alert trigger | Ingestion blocked, alert | Log gap >5min |
| Query timeout | Execute long-running query | Query latency, timeout rate | Query timeout, retry | Timeout rate >30% for >3min |
| Stream ingestion interruption | Block log source network | Ingestion success rate | Retry with backoff | Ingestion failure >10% for >5min |
| Storage quota exceeded | Fill storage to limit | Write success rate, quota alert | Write throttling | Write failure >1min |
| IAM permission revocation | Remove log group access | API call success rate | Access denied | Failure rate >50% for >2min |
| Region outage | Block cross-region API calls | Cross-region query success | Local-only mode | Query failure >20% for >5min |

## 2. Resilience Score Model

### Scoring Dimensions (0-10 each)

| Dimension | Scoring Criteria | Weight |
|-----------|----------------|--------|
| Fault detection speed | Time from fault to alert | 20% |
| Fault isolation ability | Explosion radius (affected log groups) | 20% |
| Recovery automation | Auto-healing success rate, MTTR | 25% |
| Degradation quality | Log availability during degradation | 15% |
| Data consistency | Log integrity after recovery | 20% |

### Resilience Grades

| Score | Grade | Recommendation |
|-------|-------|---------------|
| 8-10 | A (Excellent) | Regular chaos validation, maintain |
| 6-8 | B (Good) | Supplement missing fault scenarios |
| 4-6 | C (Fair) | Increase self-healing, improve degradation |
| 0-4 | D (Weak) | Prioritize critical resilience gaps |

## 3. Chaos Experiment Workflow

```yaml
chaos_experiment:
  name: "lts-log-group-deletion"
  objective: "Verify LTS handles log group deletion gracefully"

  preconditions:
    - "LTS log group with active ingestion"
    - "CES alarm configured for log group status"
    - "Log backup to OBS configured"

  steps:
    - inject_fault: "Delete LTS log group via API"
    - observe_metrics: "Monitor ingestion rate, alert trigger"
    - verify_behavior: "Confirm alert fires, ingestion fails gracefully"
    - rollback_fault: "Restore log group from backup"

  success_criteria:
    - "Alert triggered within 5min"
    - "No log data loss (backed up to OBS)"
    - "Ingestion resumes after restore"

  emergency_rollback:
    - "Restore log group from OBS backup"
    - "Reconfigure log sources"
    - "Verify log continuity"
```

## 4. LTS-Specific Experiment Details

### 4.1 Log Group Deletion (Primary Scenario)

**Objective**: Verify log group deletion detection and backup restore.

**Injection**:
```bash
# Delete log group
hcloud LTS DeleteLogGroup --log_group_id <log-group-id>
```

**Metrics to Monitor**:
- `LTS.LogGroupCount` via CES
- Log ingestion rate
- API error rate

**Expected**: Alert fires, log source fails gracefully.

### 4.2 Query Timeout

**Objective**: Verify long-running query timeout and retry.

**Injection**:
```bash
# Execute query with very large time range
hcloud LTS QueryLogs --log_group_id <log-group-id> \
  --start_time $(date -d '1 year ago' +%s) \
  --end_time $(date +%s) \
  --query "SELECT * FROM logs"
```

**Metrics**: Query latency, timeout rate, retry count.

### 4.3 Storage Quota Exceeded

**Objective**: Verify storage quota enforcement and lifecycle.

**Injection**:
```bash
# Fill storage by sending large volumes of logs
for i in {1..10000}; do
  echo "test log entry $i" | hcloud LTS UploadLogs --log_group_id <log-group-id>
done
```

**Metrics**: Storage usage, write success rate, quota alert.

## 5. Emergency Rollback Procedures

| Scenario | Rollback Action |
|----------|-----------------|
| Log group deletion | Restore from OBS backup, reconfigure sources |
| Query timeout | Cancel query, optimize query parameters |
| Storage overflow | Enable lifecycle policy, delete old logs |
| Permission revocation | Re-grant IAM permissions |

## 6. Compliance Checklist

- [x] ≥5 fault injection experiments designed (6 experiments)
- [x] Resilience scoring model defined (5 dimensions, 20% weights)
- [x] Experiment workflow documented (YAML format)
- [x] Emergency rollback procedures defined (4 scenarios)
