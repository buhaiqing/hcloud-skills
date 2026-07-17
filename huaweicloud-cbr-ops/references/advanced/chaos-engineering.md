# Chaos Engineering — CBR

> **Purpose**: Document fault injection experiments for CBR resilience verification.
> **Extends**: `huaweicloud-skill-generator/references/chaos-engineering-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Fault Injection Experiment Design

| Experiment Type | Injection Method | Observed Metrics | Expected Behavior | Termination Condition |
|----------------|-----------------|-----------------|------------------|---------------------|
| Backup task failure | Cancel running backup job | Backup job status, failure reason | Retry with backoff | Backup missed >1 cycle |
| Restore timeout | Execute slow restore operation | Restore duration, success rate | Timeout + retry | Timeout >30min |
| Storage vault full | Fill vault to capacity | Vault usage, write success | Lifecycle trigger | Write failure >1min |
| Cross-region replication failure | Block replication network | Replication status, lag | Retry with backoff | Replication lag >24h |
| Agent heartbeat loss | Stop CBR agent on VM | Agent status, backup scheduling | Agent restart, reschedule | Backup missed >1 cycle |
| Vault deletion | Delete backup vault | Backup availability, alert | Protection disabled alert | Data loss if unprotected |

## 2. Resilience Score Model

### Scoring Dimensions (0-10 each)

| Dimension | Scoring Criteria | Weight |
|-----------|----------------|--------|
| Fault detection speed | Time from fault to CES alert | 20% |
| Fault isolation ability | Explosion radius (affected backups) | 20% |
| Recovery automation | Auto-healing success rate, MTTR | 25% |
| Degradation quality | Backup availability during degradation | 15% |
| Data consistency | Backup integrity after recovery | 20% |

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
  name: "cbr-backup-failure"
  objective: "Verify CBR handles backup task failure gracefully"

  preconditions:
    - "CBR vault with active backup policies"
    - "CES alarm configured for backup job status"
    - "Backup target VM with CBR agent"

  steps:
    - inject_fault: "Cancel running backup job via API"
    - observe_metrics: "Monitor backup retry, alert trigger"
    - verify_behavior: "Confirm retry with backoff, alert fires"
    - rollback_fault: "Verify next scheduled backup succeeds"

  success_criteria:
    - "Backup retry within exponential backoff"
    - "Alert triggered for failed backup"
    - "Subsequent backup completes successfully"

  emergency_rollback:
    - "Manual backup trigger"
    - "Increase retry frequency"
    - "Expand vault capacity if needed"
```

## 4. CBR-Specific Experiment Details

### 4.1 Backup Task Failure (Primary Scenario)

**Objective**: Verify backup retry mechanism and alert firing.

**Injection**:
```bash
# Cancel running backup job
hcloud CBR CancelBackupJob --job_id <job-id>
```

**Metrics to Monitor**:
- `CBR.BackupJobStatus` via CES
- Backup retry count
- Vault usage

**Expected**: Retry with exponential backoff, alert fires.

### 4.2 Restore Timeout

**Objective**: Verify restore timeout handling and retry.

**Injection**:
```bash
# Start restore with large data volume (may take long time)
hcloud CBR RestoreBackup --backup_id <backup-id> --target_vm <vm-id>
```

**Metrics**: Restore duration, timeout rate, success rate.

### 4.3 Storage Vault Full

**Objective**: Verify vault quota enforcement and lifecycle.

**Injection**:
```bash
# Fill vault by creating many backups
for i in {1..50}; do
  hcloud CBR CreateBackup --vault_id <vault-id> --resource_id <resource-id>
done
```

**Metrics**: Vault usage, write success rate, lifecycle trigger.

## 5. Emergency Rollback Procedures

| Scenario | Rollback Action |
|----------|-----------------|
| Backup retry exhaustion | Manual backup trigger, investigate root cause |
| Restore failure | Retry with smaller dataset, check target VM |
| Vault full | Expand vault, enable lifecycle policy |
| Cross-region replication failure | Manual replication retry, verify destination |

## 6. Compliance Checklist

- [x] ≥5 fault injection experiments designed (6 experiments)
- [x] Resilience scoring model defined (5 dimensions, 20% weights)
- [x] Experiment workflow documented (YAML format)
- [x] Emergency rollback procedures defined (4 scenarios)
