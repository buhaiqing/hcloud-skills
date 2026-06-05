# GaussDB Cost Optimization (FinOps)

## 1. Right-Sizing Instance Specifications

GaussDB flavors are priced based on vCPU + memory + storage. Use the `ListFlavors()` API to audit current selections.

**Check current spec distribution**:
```bash
hcloud GaussDB ListInstances --cli-region="{{env.REGION}}" \
  --cli-query="instances[].{name:name,flavor:flavor_ref,disk_usage:disk_usage,status:status}"
```

**Strategy**:
| Workload Profile | Recommended Spec | Cost Factor |
|-----------------|-----------------|-------------|
| Dev/Test / low-TPS | `gaussdb.opengauss.2xlarge.x864.4` (4vCPU 16G) | Minimal |
| Production medium | `gaussdb.opengauss.4xlarge.x864.8` (8vCPU 32G) | Balanced |
| Production high-TPS | `gaussdb.opengauss.8xlarge.x864.16` (16vCPU 64G) | Performance |
| Analytics / batch | `gaussdb.opengauss.16xlarge.x864.32` (32vCPU 128G) | Throughput |

## 2. Storage Cost Management

GaussDB uses `ULTRAHIGH` (SSD) storage by default.

**Monitor storage utilization**:
```bash
hcloud GaussDB ShowInstanceDetail --instance_id="{{env.GAUSSDB_INSTANCE_ID}}" \
  --cli-query="{name:name,disk_usage:disk_usage,volume_size:volume.size,backup_used_space:backup_used_space}"
```

**Actions**:
- Over-provisioned (disk_usage < 30% for 30d) → Downsize via migration
- Under-provisioned (disk_usage > 85%) → Scale up before reaching limits
- Remove unused old manual backups → `DeleteManualBackup()`

## 3. Backup Cost Optimization

Automated backups consume storage at no additional charge up to instance storage size, but overage is billed.

**Current backup usage**:
```bash
hcloud GaussDB ListBackups --instance_id="{{env.GAUSSDB_INSTANCE_ID}}" \
  --cli-query="backups[?type=='auto'].{name:name,size:size,status:status,begin_time:begin_time}"
```

**Optimization**:
- Set `keep_days=7` (default 7) — longer retention increases cost
- Delete manual backups older than required compliance window
- For large datasets, schedule backups during off-peak hours

```bash
hcloud GaussDB SetBackupPolicy \
  --instance_id="{{env.GAUSSDB_INSTANCE_ID}}" \
  --keep_days=7 \
  --start_time="02:00:00" \
  --period="1,3,5" \
  --cli-region="{{env.REGION}}"
```

## 4. Idle / Underutilized Instance Detection

**Query instances with low activity**:
```bash
hcloud GaussDB ListInstances --cli-region="{{env.REGION}}" \
  --cli-query="instances[?disk_usage<='20'].{id:id,name:name,disk_usage:disk_usage}"
```

**Action Plan**:
1. Verify if instance is still needed (contact project team)
2. Delete unused instances: `DeleteInstance(instance_id=...)`
3. For temporarily idle instances, consider downscaling flavor

## 5. Enterprise Project Cost Allocation

Tag GaussDB instances by enterprise project for cost tracking:

```bash
# List instances by enterprise project
hcloud GaussDB ListEpsQuotas --cli-region="{{env.REGION}}" \
  --cli-query="quotas[].{eps_id:enterprise_project_id,volume:volume_quota,instance:instance_quota}"
```

**Best Practice**:
- Use separate enterprise projects for prod/staging/dev
- Set quota limits per project to prevent runaway spending
- Review monthly cost report by enterprise project

## 6. Scheduled Deletion of Test Instances

For CI/CD or ephemeral environments:

```bash
# Delete instances older than N days with tag "ephemeral=true"
expiry_date=$(date -d "-30 days" +%Y-%m-%d)
hcloud GaussDB ListInstances --cli-region="{{env.REGION}}" \
  --cli-query="instances[?starts_with(created, '$expiry_date')].{id:id,name:name}" \
  | jq -r '.[].id' | while read id; do
    hcloud GaussDB DeleteInstance --instance_id="$id" --cli-region="{{env.REGION}}"
  done
```

## 7. Cost Dashboard Monitoring

Set up CloudEye alarms:
- **Disk usage > 85%** → storage scaling alert
- **Backup size > 500 GB** → review retention policy
- **Idle instance (no connections for 7d)** → cleanup trigger
