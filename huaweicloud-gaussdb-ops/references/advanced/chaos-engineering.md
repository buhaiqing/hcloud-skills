# Chaos Engineering — GaussDB

> **Purpose**: Document fault injection experiments for GaussDB resilience verification.
> **Extends**: `huaweicloud-skill-generator/references/chaos-engineering-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Fault Injection Experiment Design

| Experiment Type | Injection Method | Observed Metrics | Expected Behavior | Termination Condition |
|----------------|-----------------|-----------------|------------------|---------------------|
| Primary failure | Stop primary GaussDB instance | Failover time, connection recovery | ≤ 60s failover, transparent reconnect | Availability <99% for >5min |
| AZ failure | Block cross-AZ traffic via SG rule | Cross-AZ request success rate | Degraded but not failed | Success rate <50% for >2min |
| Transaction lock contention | Hold lock for extended period | Lock wait timeout, session count | Lock timeout + rollback | Session exhaustion |
| Storage space exhaustion | Fill data volume to 95% | Write success rate, alert trigger | Alert at 80%, write block at 95% | Write failure >1min |
| Backup interruption | Cancel ongoing backup job | Backup status, data integrity | Retry or resume backup | Backup integrity check fails |
| CPU overload | Stress test to 100% | Query latency, connection pool | Connection queuing, timeout | Latency >10s for >5min |

## 2. Resilience Score Model

### Scoring Dimensions (0-10 each)

| Dimension | Scoring Criteria | Weight |
|-----------|----------------|--------|
| Fault detection speed | Time from fault to CES alert | 20% |
| Fault isolation ability | Explosion radius (affected sessions) | 20% |
| Recovery automation | Auto-healing success rate, MTTR | 25% |
| Degradation quality | Availability during degradation | 15% |
| Data consistency | Data integrity after recovery (WAL) | 20% |

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
  name: "gaussdb-primary-failure"
  objective: "Verify GaussDB primary-standby failover within 60s"

  preconditions:
    - "GaussDB instance in HA configuration"
    - "CES alarm configured for instance health"
    - "Automated backup enabled"

  steps:
    - inject_fault: "Stop primary GaussDB instance via API"
    - observe_metrics: "Monitor failover time via CES"
    - verify_behavior: "Confirm standby promotes to primary ≤ 60s"
    - rollback_fault: "Restart original instance as standby"

  success_criteria:
    - "Failover completes ≤ 60s"
    - "Connection strings auto-update"
    - "No data loss (WAL integrity verified)"

  emergency_rollback:
    - "Force restart original instance"
    - "Manual failover if auto-failover fails"
    - "Restore from latest backup if needed"
```

## 4. GaussDB-Specific Experiment Details

### 4.1 Primary-Standby Failover (Primary Scenario)

**Objective**: Verify GaussDB handles primary instance failure gracefully.

**Injection**:
```bash
# Stop primary instance
hcloud GaussDB StopInstances --instance_id <primary-id> --force
```

**Metrics to Monitor**:
- `GaussDB.FailoverTime` via CES
- `GaussDB.ConnectionCount` state transitions
- DB instance status changes

**Expected**: Standby automatically promotes to primary, failover ≤ 60s.

### 4.2 Transaction Lock Contention

**Objective**: Verify lock wait timeout and session management.

**Injection**:
```sql
-- Session 1: Hold lock
BEGIN;
SELECT * FROM table WHERE id = 1 FOR UPDATE;
-- (keep transaction open)

-- Session 2: Wait for lock
SELECT * FROM table WHERE id = 1;
```

**Metrics**: Lock wait time, session count, timeout errors.

### 4.3 Storage Space Exhaustion

**Objective**: Verify disk alert triggers and write degradation.

**Injection**:
```bash
# Fill data volume to 95%
ssh <instance> "dd if=/dev/zero of=/data/gaussdb fill bs=1G count=$(( $(df /data | tail -1 | awk '{print $2}') * 95 / 100 / 1024 ))"
```

**Metrics**: `gaussdb.disk_usage`, write latency, alert firing time.

### 4.4 Backup Interruption

**Objective**: Verify backup job failure detection and retry.

**Injection**:
```bash
# Cancel ongoing backup
hcloud GaussDB CancelBackup --backup_id <backup-id>
```

**Metrics**: Backup job status, data integrity check.

## 5. Emergency Rollback Procedures

| Scenario | Rollback Action |
|----------|-----------------|
| Failover timeout | Force restart original instance, manual failover |
| Data corruption | Restore from latest automated backup |
| Session exhaustion | Kill idle sessions, increase connection pool |
| Lock deadlock | Rollback long-running transactions |

## 6. Compliance Checklist

- [x] ≥5 fault injection experiments designed (6 experiments)
- [x] Resilience scoring model defined (5 dimensions, 20% weights)
- [x] Experiment workflow documented (YAML format)
- [x] Emergency rollback procedures defined (4 scenarios)
