# GaussDB AIOps Patterns

## Pattern 1: Storage Exhaustion Detection

**Trigger**: `disk_usage` exceeds configurable threshold.

```bash
#!/bin/bash
# check-gaussdb-storage.sh — Run daily via cron
THRESHOLD=85
REGION="{{env.REGION}}"

hcloud GaussDB ListInstances --cli-region="$REGION" \
  --cli-query="instances[?disk_usage>='$THRESHOLD'].{id:id,name:name,disk_usage:disk_usage}" \
  --cli-output-format=json | jq -c '.[]' | while read -r instance; do
  id=$(echo "$instance" | jq -r '.id')
  name=$(echo "$instance" | jq -r '.name')
  usage=$(echo "$instance" | jq -r '.disk_usage')
  echo "[WARN] GaussDB $name ($id): disk_usage=${usage}% exceeds ${THRESHOLD}%"
  # Auto-remediate: scale storage by 20%
  # hcloud GaussDB ResizeInstanceFlavor --instance_id="$id" ...
done
```

**Escalation**: If storage >95% → P0 incident.

---

## Pattern 2: Backup Health Anomaly

**Trigger**: Backup failures or missing scheduled backups.

```bash
#!/bin/bash
# check-gaussdb-backups.sh
REGION="{{env.REGION}}"
MAX_BACKUP_AGE_HOURS=48

hcloud GaussDB ListInstances --cli-region="$REGION" \
  --cli-query="instances[].id" -o json | jq -r '.[]' | while read -r instance_id; do
  
  # Check latest backup
  latest=$(hcloud GaussDB ListBackups \
    --instance_id="$instance_id" \
    --backup_type="auto" \
    --limit=1 \
    --cli-region="$REGION" \
    --cli-query="backups[0].{end_time:end_time,status:status}" \
    --cli-output-format=json 2>/dev/null)
  
  status=$(echo "$latest" | jq -r '.status')
  end_time=$(echo "$latest" | jq -r '.end_time')
  
  if [ "$status" = "FAILED" ]; then
    echo "[WARN] GaussDB $instance_id: last backup FAILED"
    # Auto-remediate: retry backup
    hcloud GaussDB CreateManualBackup \
      --instance_id="$instance_id" \
      --name="auto-retry-$(date +%Y%m%d-%H%M)" \
      --description="Auto-retry after failure" \
      --cli-region="$REGION"
  fi
done
```

---

## Pattern 3: Connection Saturation Anomaly

**Trigger**: Sudden spike in connections exhausting the `max_connections` pool.

**Detection**:
```bash
# Query via CloudEye metrics API or GaussDB system tables
# Step 1: Check active connections count
gaussdb_connections=$(hcloud GaussDB ShowInstanceDetail \
  --instance_id="{{env.GAUSSDB_INSTANCE_ID}}" \
  --cli-region="{{env.REGION}}" \
  --cli-query="connection_count")

# Step 2: Compare with instance max_connections
max_conn=$(hcloud GaussDB ShowConfigurationSetting \
  --config_id="{{env.GAUSSDB_CONFIG_ID}}" \
  --cli-region="{{env.REGION}}" \
  --cli-query="parameters[?name=='max_connections'].value | [0]")

ratio=$(echo "scale=2; $gaussdb_connections * 100 / $max_conn" | bc)
echo "Connection ratio: ${ratio}%"
```

**Remediation**:
- `ratio > 90%` → Apply parameter template with higher `max_connections`
- `ratio > 95%` → Resize instance flavor (more vCPU/memory)
- Investigate app-side connection leaks

---

## Pattern 4: Long-Running Query Detection

**Trigger**: Queries running longer than SLA threshold (e.g., >5 minutes).

**Detection approach**:
```bash
#!/bin/bash
# check-gaussdb-long-queries.sh — Run every 5 minutes
REGION="{{env.REGION}}"
THRESHOLD_SECONDS=300

# List all instances and their tasks
hcloud GaussDB ListTasks --cli-region="$REGION" \
  --cli-query="tasks[?status=='Running'].{id:id,name:name,created:created}" \
  --cli-output-format=json | jq -c '.[]' | while read -r task; do
  created=$(echo "$task" | jq -r '.created')
  now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  # Calculate duration — if > THRESHOLD, flag
  echo "[WARN] Long-running task: $(echo "$task" | jq -r '.name') since $created"
done
```

**Remediation**:
- For known slow operations (backup, resize): extend timeout expectations
- For stuck queries: check `pg_cancel_backend()` or contact support
- For recurring slow queries: optimize indexes, update statistics

---

## Pattern 5: Cross-Product Correlation

**Combined detection**: GaussDB + ECS + CloudEye

```bash
# Step 1: Find all GaussDB instances
# Step 2: For each, check underlying ECS metrics (CPU, memory, IO)
# Step 3: Correlation: high IOPS + high disk_usage + increased query latency
#         → likely storage bottleneck, recommend ResizeInstanceFlavor
```

---

## Auto-Remediation Script Structure

```python
#!/usr/bin/env python3
"""gaussdb-auto-heal.py — sample auto-remediation logic"""
import json
import subprocess
import sys

INSTANCE_ID = sys.argv[1]

def run_cli(cmd: str) -> dict:
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    return json.loads(result.stdout)

def check_and_remediate():
    # 1. Get instance detail
    detail = run_cli(f"hcloud GaussDB ShowInstanceDetail --instance_id={INSTANCE_ID} --cli-output-format=json")
    
    # 2. Check disk_usage
    disk_usage = float(detail.get("disk_usage", "0").rstrip("%"))
    if disk_usage > 90:
        print(f"[ACTION] disk_usage={disk_usage}% > 90% → triggering storage resize")
        # run_cli(f"hcloud GaussDB ResizeInstanceFlavor --instance_id={INSTANCE_ID} ...")
    
    # 3. Check backup health
    backups = run_cli(f"hcloud GaussDB ListBackups --instance_id={INSTANCE_ID} --limit=3 --cli-output-format=json")
    failed = [b for b in backups.get("backups", []) if b.get("status") == "FAILED"]
    if failed:
        print(f"[ACTION] {len(failed)} failed backups detected → triggering retry")
    
    # 4. Check tasks
    tasks = run_cli(f"hcloud GaussDB ListTasks --cli-output-format=json")
    running = [t for t in tasks.get("tasks", []) if t.get("status") == "Running"]
    print(f"[INFO] {len(running)} running tasks")

if __name__ == "__main__":
    check_and_remediate()
```
