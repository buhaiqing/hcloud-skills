# Chaos Engineering — RDS

> **Purpose**: Document fault injection experiments for RDS (Relational Database Service) resilience verification.
> **Extends**: `huaweicloud-skill-generator/references/chaos-engineering-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Fault Injection Experiment Design

| Experiment Type | Injection Method | Observed Metrics | Expected Behavior | Termination Condition |
|----------------|-----------------|-----------------|------------------|---------------------|
| Primary failure | Failover to standby | RTO, RPO, connection recovery | ≤ 120s RTO, zero data loss | RTO >5min |
| AZ failure | Stop primary in one AZ | Cross-AZ failover, read availability | Standby promoted, read replicas unaffected | Write unavailability >3min |
| Disk pressure | Fill data disk to 90% | Write latency, alert trigger, readonly | Alert at 80%, read-only at 95% | DB enters read-only >1min |
| Connection exhaustion | Exhaust max connections | New connection rejection, error rate | Connection pool fallback, retry logic | Application error rate >20% for >2min |
| Replication lag | Introduce delay on replica | Replication lag metric, read consistency | Lag <300s under normal, alert at >60s | Replication lag >600s for >5min |
| Backup failure | Simulate backup job failure | Backup status, recovery point | Alert triggered, manual intervention logged | Backup missed >24h window |

## 2. Resilience Score Model

### Scoring Dimensions (0-10 each)

| Dimension | Scoring Criteria | Weight |
|-----------|----------------|--------|
| Fault detection speed | Time from fault to CES alarm | 20% |
| Fault isolation ability | Explosion radius (single DB vs cluster) | 20% |
| Recovery automation | Auto-failover success rate, MTTR | 25% |
| Degradation quality | Read availability during failover | 15% |
| Data consistency | RPO = 0 verified via point-in-time recovery | 20% |

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
  name: "rds-primary-failure"
  objective: "Verify RDS failover ≤ 120s with zero data loss"

  preconditions:
    - "RDS instance with HA enabled (primary + standby)"
    - "CES alarm on instance health status"
    - "Automated backup configured"
    - "Connection string uses read/write endpoint"

  steps:
    - inject_fault: "Trigger manual failover via RDS console"
    - observe_metrics: "Monitor RTO via CES, check connection status"
    - verify_behavior: "Confirm standby promoted ≤ 120s, RPO = 0"
    - rollback_fault: "Verify original primary restored as standby"

  success_criteria:
    - "RTO ≤ 120s (measured via connection drop to recovery)"
    - "RPO = 0 (no transaction loss, verified via log sequence)"
    - "Connection string automatically routes to new primary"

  emergency_rollback:
    - "Force switchback to original primary"
    - "Restore from latest backup if data inconsistency detected"
    - "Scale connection pool if connection leak detected"
```

## 4. RDS-Specific Experiment Details

### 4.1 Primary Failure (Primary Scenario)

**Objective**: Verify HA failover RTO ≤ 120s, RPO = 0.

**Injection**:
```bash
# Trigger manual failover
hcloud RDS SwitchMasterInstance --instance-id <rds-id>
```

**Metrics to Monitor**:
- `rds004_sql_server_health` via CES
- `rds005_transaction_log_size`
- DNS TTL propagation time for endpoint

**Expected**: Standby promoted, connections reconnect via DNS, zero transaction loss.

### 4.2 Connection Exhaustion

**Objective**: Verify connection pool fallback and retry behavior.

**Injection**:
```bash
# Set max_connections to 1 (extreme case)
# In practice: simulate connection leak via long-running transaction
hcloud RDS ModifyParameter --instance-id <rds-id> --param "max_connections=50"
```

**Metrics**: Active connections, connection wait time, application error rate.

### 4.3 Disk Pressure & Read-Only

**Objective**: Verify storage alert and read-only mode behavior.

**Injection**:
```bash
# Fill data directory (requires shell access via ECS in same AZ)
# Monitor threshold alerts via CES
```

**Metrics**: `rds003_storage_usage`, write latency, read-only event log.

### 4.4 Replication Lag

**Objective**: Verify read replica lag monitoring and consistency.

**Injection**:
```bash
# Introduce replication delay via RDS parameter group
hcloud RDS ModifyParameter --instance-id <rds-id> --param "replication_delay=300"
```

**Metrics**: `rds_replication_lag`, read-after-write consistency test.

## 5. Emergency Rollback Procedures

| Scenario | Rollback Action |
|----------|----------------|
| Failover timeout > 5min | Force switchback, engage DBA on-call |
| Data inconsistency | Point-in-time recovery to last known good state |
| Read-only mode persists | Scale storage immediately, check disk fragmentation |
| Connection leak | Restart DB instance to clear connections |
| Replication break | Reinitialize replication from primary |

## 6. Compliance Checklist

- [x] ≥5 fault injection experiments designed (6 experiments)
- [x] Resilience scoring model defined (5 dimensions, 20% weights)
- [x] Experiment workflow documented (YAML format)
- [x] Emergency rollback procedures defined (5 scenarios)
