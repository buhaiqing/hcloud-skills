# Troubleshooting — Huawei Cloud CBR

## Error Code Reference

| Code | Message | Cause | Diagnostic | Resolution |
|------|---------|-------|-----------|------------|
| `CBR.0001` | Vault quota exceeded | Account vault limit reached | `ListVaults` to count vaults | Request quota increase or delete unused vaults |
| `CBR.0002` | Invalid vault type | `object_type` doesn't match resource | Verify resource type | Use correct `object_type` (`server`/`disk`/`turbo`) |
| `CBR.0003` | Billing account error | Account has insufficient balance | Check billing center | Recharge or enable auto-pay |
| `CBR.0004` | Vault not found | Vault ID is incorrect | `ListVaults` to verify | Use correct vault ID |
| `CBR.0005` | Resource not found | Protected resource doesn't exist | Verify resource ID with ECS/EVS API | Use correct resource ID |
| `CBR.0006` | Backup not found | Backup ID is incorrect | `ListBackups` to verify | Use correct backup ID |
| `CBR.0007` | Insufficient vault capacity | Vault storage is full | Check `vault_used_percent` metric | Resize vault or delete old backups |
| `CBR.0008` | Backup in progress | Another backup is running | `ListBackups` to check status | Wait for completion (5-30 min) |
| `CBR.0009` | Policy schedule conflict | Two policies overlap | List policies and check schedules | Adjust schedules to avoid overlap |
| `CBR.0010` | Resource already associated | Resource already in another vault | Check resource vault association | Disassociate first |
| `CBR.0011` | Destination region unavailable | CBR not enabled in target region | Verify CBR availability | Enable CBR in target region first |
| `CBR.0012` | Replication bandwidth exceeded | Too many concurrent replications | Monitor replication queue | Reduce concurrent replication tasks |

## Diagnostic Procedures

### Scenario 1: Backup Failed

```bash
# 1. Check backup status
hcloud CBR ShowBackup --backup_id="{{user.backup_id}}"

# 2. Check vault capacity
hcloud CBR ShowVault --vault_id="{{user.vault_id}}"
# Look for: storage_used, storage_size, status

# 3. Check CES metrics for backup failures
hcloud CES ShowMetricData \
  --namespace="SYS.CBR" \
  --metric_name="backup_failure_count" \
  --dim="vault_id={{user.vault_id}}" \
  --period="3600" --from="-24h" --to="now"
```

### Scenario 2: Restore Failed

```bash
# 1. Verify backup is available
hcloud CBR ShowBackup --backup_id="{{user.backup_id}}"
# Status should be "available"

# 2. Verify target resource exists
hcloud ECS ShowInstance --server_id="{{user.resource_id}}"

# 3. Check for capacity constraints on target
# Ensure target resource has enough free space
```

### Scenario 3: Vault Capacity Full

```bash
# 1. Check current usage
hcloud CBR ShowVault --vault_id="{{user.vault_id}}"

# 2. List old backups to identify cleanup candidates
hcloud CBR ListBackups --vault_id="{{user.vault_id}}" --format=json | \
  jq '.backups | sort_by(.created_at) | .[0:5]'

# 3. Delete expired backups
hcloud CBR DeleteBackup --backup_id="{{backup_id}}"

# 4. Or resize vault
hcloud CBR UpdateVault --vault_id="{{user.vault_id}}" --storage_size="{{new_size}}"
```

## Known Issues

| Issue | Symptom | Workaround | Fix Version |
|-------|---------|-----------|-------------|
| Backup window contention | Backup starts during peak hours | Adjust policy schedule to off-peak | N/A (config) |
| Incremental backup chain too long | Restore takes longer | Create periodic full backups | N/A (design) |
| Cross-region replication delay | Backup not available in DR region | Check network bandwidth and latency | N/A |
