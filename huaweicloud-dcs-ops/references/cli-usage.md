# CLI Usage — Huawei Cloud DCS

## CLI Command Map

| Operation | CLI Command | Required Flags | Output JSON Path |
|-----------|-------------|----------------|------------------|
| Create Instance | `hcloud dcs create-instance` | `--region`, `--name`, `--engine`, `--engine-version`, `--capacity`, `--vpc-id`, `--subnet-id`, `--security-group-id` | `.instance_id` |
| Show Instance | `hcloud dcs show-instance` | `--instance-id` | `.status` |
| List Instances | `hcloud dcs list-instances` | `--region` | `.instances[]` |
| Delete Instance | `hcloud dcs delete-instance` | `--instance-id` | — |
| Resize Instance | `hcloud dcs resize-instance` | `--instance-id`, `--new-spec-code` | `.instance_id` |
| Restart Instance | `hcloud dcs restart-instance` | `--instance-id` | — |
| Reset Password | `hcloud dcs reset-password` | `--instance-id`, `--new-password` | — |
| Create Backup | `hcloud dcs create-backup` | `--instance-id`, `--backup-name` | `.backup_id` |
| List Backups | `hcloud dcs list-backups` | `--instance-id` | `.backups[]` |
| Restore Instance | `hcloud dcs restore-instance` | `--instance-id`, `--backup-id` | `.instance_id` |
| Show Whitelist | `hcloud dcs show-whitelist` | `--instance-id` | `.whitelist` |
| Update Whitelist | `hcloud dcs update-whitelist` | `--instance-id`, `--whitelist` | — |

## Coverage Gap Table

| Operation | CLI Support | Fallback |
|-----------|-------------|----------|
| CreateInstance | ✅ Full | Not needed |
| ShowInstance | ✅ Full | Not needed |
| ListInstances | ✅ Full | Not needed |
| DeleteInstance | ✅ Full | Not needed |
| ResizeInstance | ✅ Full | Not needed |
| RestartInstance | ✅ Full | Not needed |
| ResetPassword | ✅ Full | Not needed |
| CreateBackup | ✅ Full | Not needed |
| ListBackups | ✅ Full | Not needed |
| RestoreInstance | ✅ Full | Not needed |
| ShowWhitelist | ✅ Full | Not needed |
| UpdateWhitelist | ✅ Full | Not needed |
| BatchStopOrStart | ⚠️ Partial (start/stop only) | JIT Go SDK |
| ModifyInstanceName | ✅ Full | Not needed |
| GetInstanceStatus | ✅ Full | Not needed |
| ListStatistics | ✅ Full | Not needed |

## JSON Output Parsing with jq

```bash
# Extract instance_id from create output
INSTANCE_ID=$(hcloud dcs create-instance ... | jq -r '.instance_id')

# Check instance status
STATUS=$(hcloud dcs show-instance --instance-id "$INSTANCE_ID" | jq -r '.status')

# Get instance IP and port
IP=$(hcloud dcs show-instance --instance-id "$INSTANCE_ID" | jq -r '.ip')
PORT=$(hcloud dcs show-instance --instance-id "$INSTANCE_ID" | jq -r '.port')

# List only instance names and statuses
hcloud dcs list-instances --region cn-north-4 | jq -r '.instances[] | "\(.name) \t \(.status)"'

# Count instances by status
hcloud dcs list-instances --region cn-north-4 | jq -r '.instances[].status' | sort | uniq -c

# Get backup_id from create-backup
BACKUP_ID=$(hcloud dcs create-backup --instance-id "$ID" --name "test" | jq -r '.backup_id')
```

## Common Invocation Patterns

```bash
# Quick: List all instances in region
hcloud dcs list-instances --region cn-north-4

# Create Redis 6.0 HA instance
hcloud dcs create-instance \
  --region cn-north-4 \
  --name "my-redis-ha" \
  --engine redis \
  --engine-version 6.0 \
  --capacity 4 \
  --instance-mode ha \
  --vpc-id "vpc-abc123" \
  --subnet-id "subnet-def456" \
  --security-group-id "sg-ghi789" \
  --password "SecureP@ss2024"

# Delete with no backup
hcloud dcs delete-instance --instance-id "dcs-0a1b2c3d" --delete-backup=false

# Backup with auto-naming
hcloud dcs create-backup \
  --instance-id "dcs-0a1b2c3d" \
  --backup-name "manual-$(date +%Y%m%d-%H%M%S)"

# Update whitelist
hcloud dcs update-whitelist \
  --instance-id "dcs-0a1b2c3d" \
  --whitelist-enable true \
  --whitelist "192.168.1.0/24,10.0.0.0/8"

# Reset password (will disconnect all clients!)
hcloud dcs reset-password \
  --instance-id "dcs-0a1b2c3d" \
  --new-password "NewSecureP@ss2024"
```

## CLI Error Handling

- CLI returns **exit code 0** on success, **non-zero** on failure
- Errors are written to **stderr** with JSON error object:
  ```json
  {"error_code":"DCS.0002","error_msg":"Instance not found","request_id":"xxx"}
  ```
- Map to SDK error codes:
  - `DCS.0002` → `InstanceNotFound`
  - `DCS.0007` → `QuotaExceeded`
  - `DCS.0011` → `InsufficientBalance`
- HTTP 429 → Throttling — implement exponential backoff in retry logic
