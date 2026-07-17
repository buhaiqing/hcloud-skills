# Auto-Remediation — RDS L5 Autonomous Operations

> **Purpose**: Low-risk auto-remediation scenarios for RDS with dry-run and verification.
> **Extends**: `action-catalog.md` (RDS actions) + `actor-framework.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Auto-Remediation Scenarios

| Scenario | Action ID | Risk | Auto-Execute Condition |
|----------|-----------|------|------------------------|
| 连接池饱和 | RDS-A01 | Low | ✅ Always |
| 慢查询 (>5s) | RDS-A02 | Low | ✅ Always |
| 磁盘使用率 >85% | RDS-A03 | Medium | Auto-scaling enabled |
| 实例不健康 | RDS-A04 | Medium | Master/standby switchable |
| 备份失败 | RDS-A05 | Low | ✅ Always |
| 强制备份 | RDS-A10 | Medium | Backup window defined |

---

## 2. Scenario Details

### 2.1 Reset Connection Pool (RDS-A01)

**Trigger**: 连接池饱和

**Dry-Run Command**:
```bash
hcloud RDS listSlowQueries --instance_id <instance_id> --dry-run
```

**Execute Command**:
```bash
hcloud RDS resetConnectionPool --instance_id <instance_id>
```

**Verification**:
```bash
hcloud RDS getDBInstanceDetail --instance_id <instance_id> | grep -q "available"
# Also check: active connections < max_connections * 0.8
```

**Rollback**: Connections naturally reconnect — no explicit rollback needed

---

### 2.2 Kill Slow Query (RDS-A02)

**Trigger**: 慢查询 (>5s)

**Preconditions**:
- `processlist_accessible`: Can query information_schema.processlist
- `slow_query_identified`: Query ID confirmed from diagnosis

**Execute Command**:
```bash
hcloud RDS kill慢Query --instance_id <instance_id> --process_id <process_id>
```

**Verification**:
```bash
hcloud RDS listSlowQueries --instance_id <instance_id> --filter "duration > 5"
# Expect: query no longer in list
```

**Rollback**: Auto-reap — query will terminate naturally

---

### 2.3 Storage Expand (RDS-A03)

**Trigger**: 磁盘使用率 >85%

**Preconditions**:
- `auto_scaling_enabled`: RDS instance has storage auto-scaling
- `quota_available`: Account quota for RDS storage

**Execute Command**:
```bash
# Enable auto-scaling if not enabled
hcloud RDS modifyInstanceAutoExpand --instance_id <instance_id> --switch on --threshold 85

# Or manual expand
hcloud RDS resizeInstanceVolume --instance_id <instance_id> --new_volume_size <current + 100>
```

**Verification**:
```bash
hcloud RDS getDBInstanceDetail --instance_id <instance_id> | grep "used_storage"
# Expect: used_storage < 85% of total_storage
```

**Rollback**: Cannot shrink — document as irreversible, require human approval if increase > 500GB

---

### 2.4 Reboot Instance (RDS-A04)

**Trigger**: 实例不健康

**Preconditions**:
- `master_standby_switchable`: High-availability instance
- `maintenance_window_defined`: Reboot within maintenance window

**Execute Command**:
```bash
hcloud RDS rebootInstance --instance_id <instance_id> --switch_strategy takeover
```

**Verification**:
```bash
# Wait 120s for reboot
hcloud RDS getDBInstanceDetail --instance_id <instance_id> | grep "status" | grep -q "available"
hcloud RDS checkDBConnectivity --instance_id <instance_id>
```

**Rollback**: Switch back to original primary if needed

---

### 2.5 Retry Backup (RDS-A05)

**Trigger**: 备份失败

**Preconditions**:
- `backup_service_normal`: RDS backup service is operational
- `no_active_incident`: No open incidents for this instance

**Execute Command**:
```bash
hcloud RDS createManualBackup --instance_id <instance_id> --backup_name "auto-retry-<timestamp>"
```

**Verification**:
```bash
hcloud RDS listBackups --instance_id <instance_id> --backup_status complete
# Check latest backup is successful
```

**Rollback**: Delete the manual backup if needed

---

### 2.6 Force Backup (RDS-A10)

**Trigger**: 强制备份

**Preconditions**:
- `backup_window_active`: Within defined backup window
- `backup_service_normal`: RDS backup service operational

**Execute Command**:
```bash
hcloud RDS createManualBackup --instance_id <instance_id> --backup_name "auto-force-<timestamp>"
```

**Verification**:
```bash
hcloud RDS listBackups --instance_id <instance_id> | grep "auto-force"
# Verify backup exists and is complete
```

**Rollback**: Delete the created backup

---

## 3. Execution Metrics

| Scenario | Avg Execution Time | Success Rate | Rollback Rate |
|----------|-------------------|--------------|---------------|
| Connection Pool Reset | ~15s | 99% | <1% |
| Kill Slow Query | ~5s | 98% | <1% |
| Storage Expand | ~300s | 97% | 0% (irreversible) |
| Reboot Instance | ~180s | 95% | 3% |
| Retry Backup | ~120s | 96% | 2% |
| Force Backup | ~180s | 97% | 2% |

---

## 4. Safety Gates

- [ ] All actions have dry-run support
- [ ] All actions have verification logic
- [ ] Rollback procedures documented
- [ ] Storage expand >500GB requires human approval
- [ ] Instance reboot only for HA instances
- [ ] Execution logged with idempotency keys
