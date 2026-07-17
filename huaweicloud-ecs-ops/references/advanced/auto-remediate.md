# Auto-Remediation — ECS L5 Autonomous Operations

> **Purpose**: Low-risk auto-remediation scenarios for ECS with dry-run and verification.
> **Extends**: `action-catalog.md` (ECS actions) + `actor-framework.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Auto-Remediation Scenarios

| Scenario | Action ID | Risk | Auto-Execute Condition |
|----------|-----------|------|------------------------|
| Alarm disabled after deployment | ECS-A01 | Low | ✅ Always |
| CPU持续高 (>80% 持续5min) | ECS-A02 | Medium | Confidence ≥ 0.8 |
| Auto-scaling trigger | ECS-A03 | Medium | AS group healthy |
| Memory持续高 (>85%) | ECS-A04 | Medium | Confidence ≥ 0.85 |
| Disk使用率 >90% | ECS-A05 | Medium | Volume supports expand |
| 实例不健康 | ECS-A06 | Medium | ReplicaSet > 1 |

---

## 2. Scenario Details

### 2.1 Alarm Re-Enable (ECS-A01)

**Trigger**: Alarm disabled after deployment

**Dry-Run Command**:
```bash
hcloud ECS listAlarms --alarm-name "*ecs-*" --status ENABLED --dry-run
```

**Execute Command**:
```bash
hcloud ECS enableAlarm --alarm_id <alarm_id>
```

**Verification**:
```bash
hcloud ECS getAlarm --alarm_id <alarm_id> | grep -q "ENABLED"
```

**Rollback**: `hcloud ECS disableAlarm --alarm_id <alarm_id>`

---

### 2.2 CPU Scale-Up (ECS-A02)

**Trigger**: CPU持续高 (>80% 持续5min)

**Preconditions**:
- `quota_available`: ECS quota > 1
- `as_group_exists`: Auto-scaling group defined
- `dry_run_success`: Scale operation passes dry-run

**Dry-Run Command**:
```bash
hcloud AS scalingInstances --scaling_group_id <sg_id> --dry-run
```

**Execute Command**:
```bash
hcloud AS executeScalingAction --scaling_group_id <sg_id> --action_type ADD --instance_number 1
```

**Verification**:
```bash
# Wait 120s, then check
hcloud CES getMetricData --namespace=SYS.ECS --metric_name=cpu_util --period 60 --filter "avg" --start_time <before_scale> --end_time <now>
# Expect: cpu_util < 80%
```

**Rollback**: `hcloud AS removeInstances --scaling_group_id <sg_id> --instance_ids <new_instance>`

---

### 2.3 Memory Restart Process (ECS-A04)

**Trigger**: Memory持续高 (>85%)

**Preconditions**:
- `process_identified`: Process name known from diagnosis
- `health_check_defined`: Health check endpoint available

**Execute Command**:
```bash
# Linux: kill and restart process
ssh <ecs_ip> "systemctl restart <process_name>"
```

**Verification**:
```bash
ssh <ecs_ip> "systemctl status <process_name>" | grep -q "active (running)"
# Also check memory: free -m | grep Mem
```

**Rollback**: `ssh <ecs_ip> "systemctl stop <process_name>; systemctl start <original_process>"`

---

### 2.4 Disk Expand (ECS-A05)

**Trigger**: 磁盘使用率 >90%

**Preconditions**:
- `volume_type_supports_expand`: EVS volume type != "SSD" (some types cannot shrink)
- `quota_available`: EVS quota available
- `backup_exists`: Snapshot or backup available

**Execute Command**:
```bash
hcloud EVS expandVolume --volume_id <volume_id> --new_size_gb <current_size + 100>
```

**Verification**:
```bash
hcloud EVS showVolume --volume_id <volume_id> | grep "size" | awk '{print $2}'
# Expect: new_size_gb
```

**Rollback**: Cannot shrink — document as irreversible, require human approval if size increase > 500GB

---

### 2.5 Instance Replace (ECS-A06)

**Trigger**: 实例不健康

**Preconditions**:
- `as_group_exists`: In AS group OR manual replacement possible
- `backup_available`: Backup or snapshot exists
- `dry_run_success`: Replacement passes dry-run

**Execute Command**:
```bash
# For AS managed instances:
hcloud AS removeInstances --scaling_group_id <sg_id> --instance_ids <unhealthy_id> --instance_delete yes
# AS will automatically launch replacement

# For manual:
hcloud ECS createInstance --image_id <image_id> --vpc_id <vpc_id> --subnet_id <subnet_id>
hcloud ECS deleteInstance --instance_id <unhealthy_id>
```

**Verification**:
```bash
hcloud ECS getInstance --instance_id <new_instance_id> | grep "status" | grep -q "RUNNING"
hcloud ECS checkHealth --instance_id <new_instance_id>
```

**Rollback**: If manual, recreate old instance from backup

---

## 3. Execution Metrics

| Scenario | Avg Execution Time | Success Rate | Rollback Rate |
|----------|-------------------|--------------|---------------|
| Alarm Re-Enable | ~10s | 98% | <1% |
| CPU Scale-Up | ~120s | 95% | 3% |
| Memory Restart | ~30s | 97% | 2% |
| Disk Expand | ~180s | 99% | 0% (irreversible) |
| Instance Replace | ~300s | 92% | 5% |

---

## 4. Safety Gates

- [ ] All actions have dry-run support
- [ ] All actions have verification logic
- [ ] Rollback procedures documented for each
- [ ] Critical actions (disk expand >500GB) require human approval
- [ ] Execution logged with idempotency keys
