# GaussDB Safety Gates

## Gate Protocol

Every high-risk GaussDB operation follows this pre-flight procedure:

```
1. PRE-CHECK  → Validate instance status, backup existence, and dependencies
2. CONFIRM    → Require explicit `--confirm` flag or interactive "yes"
3. EXECUTE    → Perform the operation
4. VERIFY     → Confirm the result and instance health
5. ROLLBACK   → If failure, restore from backup
```

---

## Operation: Delete Instance (`DeleteInstance`)

**Risk**: CRITICAL — permanent data loss

**Pre-flight**:
```bash
#!/bin/bash
INSTANCE_ID="{{env.GAUSSDB_INSTANCE_ID}}"

# 1. Check instance exists and is ACTIVE
status=$(hcloud GaussDB ShowInstanceDetail \
  --instance_id="$INSTANCE_ID" \
  --cli-region="{{env.REGION}}" \
  --cli-query="status" --cli-output-format=json | jq -r '.')
[ "$status" != "ACTIVE" ] && { echo "[ERROR] Instance not ACTIVE (status=$status)"; exit 1; }

# 2. Verify latest backup is valid
latest_backup=$(hcloud GaussDB ListBackups \
  --instance_id="$INSTANCE_ID" \
  --limit=1 --cli-query="backups[0].status" --cli-output-format=json | jq -r '.')
[ "$latest_backup" != "COMPLETED" ] && { echo "[ERROR] No valid backup found"; exit 1; }

# 3. Check for dependent applications
echo "[CHECK] Are there applications connecting to this instance? (yes/no)"
read -r ans; [ "$ans" != "yes" ] && { echo "[ABORT] Verify application dependencies first"; exit 1; }

# 4. Final confirmation
echo "CONFIRM delete of instance $INSTANCE_ID? Type the instance name to confirm:"
read -r confirm
[ "$confirm" != "$INSTANCE_ID" ] && { echo "[ABORT] Confirmation mismatch"; exit 1; }

# 5. Execute
hcloud GaussDB DeleteInstance --instance_id="$INSTANCE_ID" --cli-region="{{env.REGION}}"
```

---

## Operation: Reset Password (`ResetPwd`)

**Risk**: HIGH — service disruption during credential rotation

**Pre-flight**:
```bash
INSTANCE_ID="{{env.GAUSSDB_INSTANCE_ID}}"

# 1. Verify instance status
status=$(hcloud GaussDB ShowInstanceDetail --instance_id="$INSTANCE_ID" \
  --cli-query="status" --cli-query="status" --cli-output-format=json | jq -r '.')
[ "$status" != "ACTIVE" ] && { echo "[ERROR] Instance not ACTIVE"; exit 1; }

# 2. Notify team
echo "[CHECK] Have you notified the application team about password rotation? (yes/no)"
read -r ans; [ "$ans" != "yes" ] && { echo "[ABORT] Notify team first"; exit 1; }

# 3. Execute
hcloud GaussDB ResetPwd \
  --instance_id="$INSTANCE_ID" \
  --password="{{env.NEW_DB_PASSWORD}}" \
  --cli-region="{{env.REGION}}"

# 4. Verify connectivity (post-change smoke test)
# psql -h "$ENDPOINT" -p 8000 -U root -W "$NEW_PASSWORD" -c "SELECT 1;" && echo "[OK] Password updated"
```

---

## Operation: Resize Instance Flavor (`ResizeInstanceFlavor`)

**Risk**: HIGH — brief connection interruption (30-120 seconds)

**Pre-flight**:
```bash
INSTANCE_ID="{{env.GAUSSDB_INSTANCE_ID}}"

# 1. Check current flavor
current=$(hcloud GaussDB ShowInstanceDetail --instance_id="$INSTANCE_ID" \
  --cli-query="flavor_ref" --cli-output-format=json | jq -r '.')
echo "[INFO] Current flavor: $current"

# 2. Validate target flavor exists
target_flavor="gaussdb.opengauss.8xlarge.x864.16"
matches=$(hcloud GaussDB ListFlavors --cli-region="{{env.REGION}}" \
  --cli-query="flavors[?spec_code=='$target_flavor'].spec_code | [0]" --cli-output-format=json | jq -r '.')
[ "$matches" = "null" ] && { echo "[ERROR] Target flavor not available in region"; exit 1; }

# 3. Schedule maintenance window
echo "[CHECK] Flavor resize causes ~60s downtime. Schedule during maintenance window? (yes/no)"
read -r ans; [ "$ans" != "yes" ] && { echo "[ABORT] Reschedule during maintenance window"; exit 1; }

# 4. Execute
hcloud GaussDB ResizeInstanceFlavor \
  --instance_id="$INSTANCE_ID" \
  --flavor_ref="$target_flavor" \
  --cli-region="{{env.REGION}}"

# 5. Monitor task
hcloud GaussDB ListTasks --cli-region="{{env.REGION}}" \
  --cli-query="tasks[?status=='Running'].{name:name,created:created}"
```

---

## Operation: Delete Manual Backup (`DeleteManualBackup`)

**Risk**: MEDIUM — potential recovery point loss

**Pre-flight**:
```bash
BACKUP_ID="{{env.GAUSSDB_BACKUP_ID}}"

# 1. Verify backup exists and is manual
backup=$(hcloud GaussDB ListBackups --backup_id="$BACKUP_ID" \
  --cli-region="{{env.REGION}}" --cli-query="backups[0].{name:name,type:type}" \
  --cli-output-format=json | jq -r '.')
backup_type=$(echo "$backup" | jq -r '.type')
[ "$backup_type" != "manual" ] && { echo "[ERROR] Only manual backups can be deleted"; exit 1; }

# 2. Ensure other valid backups remain
count=$(hcloud GaussDB ListBackups --cli-region="{{env.REGION}}" \
  --cli-query="length(backups)" --cli-output-format=json | jq -r '.')
[ "$count" -le 1 ] && { echo "[ERROR] Cannot delete last backup"; exit 1; }

# 3. Confirm
echo "Delete backup $(echo "$backup" | jq -r '.name')? (yes/no)"
read -r ans; [ "$ans" != "yes" ] && exit 1

# 4. Execute
hcloud GaussDB DeleteManualBackup --backup_id="$BACKUP_ID" --cli-region="{{env.REGION}}"
```

---

## Operation: Apply Configuration Template (`ApplyConfiguration`)

**Risk**: MEDIUM — some parameter changes require instance restart

**Pre-flight**:
```bash
CONFIG_ID="{{env.GAUSSDB_CONFIG_ID}}"
INSTANCE_ID="{{env.GAUSSDB_INSTANCE_ID}}"

# 1. Preview parameter differences
hcloud GaussDB ListDiffDetails \
  --config_id="$CONFIG_ID" \
  --instance_id="$INSTANCE_ID" \
  --cli-region="{{env.REGION}}" \
  --cli-query="differences[].{name:parameter_name,old:old_value,new:new_value}"

# 2. Check if restart required (parameters like shared_buffers, max_connections)
echo "[CHECK] Does this template require a restart? Some parameters apply dynamically."
echo "Proceed? (yes/no)"
read -r ans; [ "$ans" != "yes" ] && exit 1

# 3. Apply
hcloud GaussDB ApplyConfiguration \
  --instance_id="$INSTANCE_ID" \
  --configuration_id="$CONFIG_ID" \
  --cli-region="{{env.REGION}}"
```

---

## Operation: Bind/Unbind EIP

**Risk**: LOW — networking change

**Pre-flight for Bind**:
```bash
# Verify EIP is not already bound
public_ip=$(hcloud GaussDB ShowInstanceDetail --instance_id="{{env.GAUSSDB_INSTANCE_ID}}" \
  --cli-query="public_ips[0]" --cli-output-format=json | jq -r '.')
[ "$public_ip" != "null" ] && { echo "[WARN] EIP already bound: $public_ip"; exit 1; }
```

**Pre-flight for Unbind**:
```bash
# Ensure no active connections rely on public endpoint
echo "[CHECK] Confirm no application uses the EIP for connections? (yes/no)"
read -r ans; [ "$ans" != "yes" ] && exit 1
```
