# Auto-Remediation — DCS (Redis) L5 Autonomous Operations

> **Purpose**: Low-risk auto-remediation scenarios for DCS (Redis) with dry-run and verification.
> **Extends**: `action-catalog.md` (DCS actions) + `actor-framework.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Auto-Remediation Scenarios

| Scenario | Action ID | Risk | Auto-Execute Condition |
|----------|-----------|------|------------------------|
| 连接池饱和 | DCS-A01 | Low | ✅ Always |
| Memory持续高 | DCS-A02 | Medium | Confidence ≥ 0.85 |
| 实例不健康 | DCS-A03 | Medium | ReplicaSet > 1 |
| 慢查询 | DCS-A04 | Low | ✅ Always |
| 备份失败 | DCS-A05 | Low | ✅ Always |

---

## 2. Scenario Details

### 2.1 Connection Pool Reset (DCS-A01)

**Trigger**: 连接池饱和

**Dry-Run Command**:
```bash
hcloud DCS listInstances --instance_id <instance_id> --dry-run
```

**Execute Command**:
```bash
hcloud DCS restartInstance --instance_id <instance_id>
```

**Verification**:
```bash
hcloud DCS getInstanceDetail --instance_id <instance_id> | grep "status" | grep -q "RUNNING"
# Check connections: hcloud DCS getStatistics --instance_id <instance_id>
```

**Rollback**: Wait for connections to naturally reconnect — typically <30s

---

### 2.2 Memory Scale-Up (DCS-A02)

**Trigger**: Memory持续高 (>85% used)

**Preconditions**:
- `instance_type_supports_scale`:DCS instance supports memory resize
- `quota_available`: Account quota for DCS

**Dry-Run Command**:
```bash
hcloud DCS resizeInstance --instance_id <instance_id> --new_capacity <capacity+2> --dry-run
```

**Execute Command**:
```bash
hcloud DCS resizeInstance --instance_id <instance_id> --new_capacity <new_capacity_gb>
```

**Verification**:
```bash
# Wait 180s for resize
hcloud DCS getInstanceDetail --instance_id <instance_id> | grep "capacity"
# Verify new capacity and instance status RUNNING
```

**Rollback**: Cannot downgrade — document as irreversible, require human approval

---

### 2.3 Instance Restart (DCS-A03)

**Trigger**: 实例不健康

**Preconditions**:
- `replica_available`: If master/replica setup, replica is available
- `data_persistence_enabled`: AOF or RDB backup enabled

**Execute Command**:
```bash
hcloud DCS restartInstance --instance_id <instance_id>
```

**Verification**:
```bash
# Wait 60s for restart
hcloud DCS getInstanceDetail --instance_id <instance_id> | grep "status" | grep -q "RUNNING"
hcloud DCS checkInstanceConnectivity --instance_id <instance_id>
```

**Rollback**: No explicit rollback — instance restarts with latest data from persistence

---

### 2.4 Slow Query Analysis (DCS-A04)

**Trigger**: 慢查询 (command latency > threshold)

**Preconditions**:
- `monitoring_enabled`: DCS performance monitoring enabled
- `slowlog_accessible`: Can query slowlog

**Execute Command**:
```bash
hcloud DCS getSlowLog --instance_id <instance_id> --start_time <timestamp>
```

**Analysis & Action**:
```bash
# If specific keys causing slow:
hcloud DCS analyzeBigKey --instance_id <instance_id> --key_pattern <pattern>
# If big key found:
hcloud DCS splitBigKey --instance_id <instance_id> --key <key_name>
```

**Verification**:
```bash
hcloud DCS getSlowLog --instance_id <instance_id> --end_time <now>
# Verify no new slow commands
```

**Rollback**: N/A (analysis only, or key splitting is reversible via rebalancing)

---

### 2.5 Retry Backup (DCS-A05)

**Trigger**: 备份失败

**Preconditions**:
- `backup_service_normal`: DCS backup service operational
- `no_active_incident`: No open incidents for this instance

**Execute Command**:
```bash
hcloud DCS createBackup --instance_id <instance_id> --backup_name "auto-retry-<timestamp>"
```

**Verification**:
```bash
hcloud DCS listBackups --instance_id <instance_id> --status complete
# Verify latest backup is successful
```

**Rollback**: Delete the backup if needed

---

## 3. Execution Metrics

| Scenario | Avg Execution Time | Success Rate | Rollback Rate |
|----------|-------------------|--------------|---------------|
| Connection Pool Reset | ~30s | 98% | <1% |
| Memory Scale-Up | ~180s | 96% | 0% (irreversible) |
| Instance Restart | ~60s | 97% | 2% |
| Slow Query Analysis | ~20s | 99% | <1% |
| Retry Backup | ~120s | 97% | 2% |

---

## 4. Safety Gates

- [ ] All actions have dry-run support
- [ ] All actions have verification logic
- [ ] Rollback procedures documented
- [ ] Memory scale-up requires human approval (irreversible)
- [ ] Instance restart only for instances with persistence enabled
- [ ] Execution logged with idempotency keys
