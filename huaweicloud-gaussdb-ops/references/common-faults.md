# GaussDB Common Faults

## Fault 1: Instance Creation Fails
**Symptom**: `CreateInstance` returns 400 or 500.

**Causes**:
- Incorrect `flavor_ref` (wrong spec code)
- Subnet/VPC misconfiguration
- Quota exceeded (instance or storage)
- Unsupported region/availability zone combination

**Resolution**:
1. Verify `ListFlavors()` output for correct `flavor_ref`:
   ```bash
   hcloud GaussDB ListFlavors --cli-region="{{env.REGION}}" \
     --cli-query="flavors[?engine_version=='V2.0-3.2'].{spec:spec_code}"
   ```
2. Check project quotas via `hcloud GaussDB ShowQuotas`.
3. Validate VPC/Subnet IDs exist and are in the same region.
4. Retry with `--debug` flag for more detail.

---

## Fault 2: Instance Stuck in "CREATING" / "BACKING UP"
**Symptom**: Instance stays in transitional status for >30 minutes.

**Causes**:
- Cloud resource provisioning delay
- Storage allocation backlog in the AZ

**Resolution**:
1. Check task list:
   ```bash
   hcloud GaussDB ListTasks --cli-region="{{env.REGION}}" \
     --cli-query="tasks[?status=='Running']"
   ```
2. If status persists >1 hour, open a Huawei Cloud support ticket.
3. Do not attempt `DeleteInstance` on a CREATING instance — wait or contact support.

---

## Fault 3: Password Reset Not Taking Effect
**Symptom**: `ResetPwd()` succeeds but old password still works / new password rejected.

**Causes**:
- Password does not meet GaussDB complexity policy (< 8 chars, missing mixed case)
- Propagation delay (30 seconds)
- Application connection pool caching old credentials

**Resolution**:
1. Validate password format: minimum 8 characters, at least one uppercase, one lowercase, one digit, one special char.
2. Wait 60 seconds and retry login.
3. Restart application pods to refresh connection pool.

---

## Fault 4: Backup Failure
**Symptom**: `ListBackups` shows status `FAILED`.

**Causes**:
- Insufficient instance storage during backup
- Instance in non-ACTIVE state
- Concurrent DDL operations blocking backup

**Resolution**:
1. Check instance status: `hcloud GaussDB ShowInstanceDetail --instance_id=...`
2. Ensure `disk_usage` is <95% and storage is sufficient.
3. Retry backup during low-load window.
4. If recurring, increase `keep_days` or switch to longer interval.

---

## Fault 5: Cannot Connect — Connection Refused
**Symptom**: Application fails to connect to GaussDB endpoint.

**Causes**:
- Security group not allowing inbound traffic on port 8000
- EIP not bound or misconfigured
- Instance in FAULT/RESTARTING status
- SSL mismatch (client requires SSL but server not configured)

**Resolution**:
```bash
# Check instance status and endpoint
hcloud GaussDB ShowInstanceDetail --instance_id="{{env.GAUSSDB_INSTANCE_ID}}" \
  --cli-region="{{env.REGION}}" \
  --cli-query="{status:status,private_ips:private_ips[0]}"

# Verify security group rules (via VPC console or support)
# For SSL: download cert and configure client
hcloud GaussDB ShowSslCertDownloadLink --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"
```

---

## Fault 6: Slow Query Performance
**Symptom**: Query execution time spikes suddenly.

**Causes**:
- Missing or stale table statistics
- Inefficient query plan (seq scan on large table)
- Resource contention with other workloads
- Insufficient instance specifications

**Resolution**:
1. Run `ANALYZE` to update statistics.
2. Check `EXPLAIN ANALYZE` for full table scans.
3. Consider `ResizeInstanceFlavor()` to upgrade specs.
4. Review slow query logs via CloudEye.

---

## Fault 7: Template Application Fails
**Symptom**: `ApplyConfiguration()` returns error.

**Causes**:
- Template is incompatible with instance version
- Instance is not ACTIVE
- Template contains parameters not supported by instance engine version

**Resolution**:
1. Verify template `datastore_version` matches instance:
   ```bash
   hcloud GaussDB ShowConfigurationSetting --config_id="{{env.GAUSSDB_CONFIG_ID}}"
   ```
2. Verify instance is ACTIVE.
3. Use `ListDiffDetails()` to preview changes before applying.

---

## Fault 8: Insufficient Disk Space
**Symptom**: Operations fail with "disk full" errors.

**Causes**:
- Binlog / WAL accumulation
- Unexpected data growth
- Backup files filling local disk

**Resolution**:
1. Monitor with:
   ```bash
   hcloud GaussDB ShowInstanceDetail --instance_id="{{env.GAUSSDB_INSTANCE_ID}}" \
     --cli-query="{disk_usage:disk_usage,volume_type:volume.type,volume_size:volume.size}"
   ```
2. Scale storage: use `ResizeInstanceFlavor()` for storage expansion.
3. Clean up old backups: `DeleteManualBackup(backup_id=...)`.
4. Set disk space alarm at 85% threshold via CloudEye.
