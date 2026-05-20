# API & SDK Usage — Huawei Cloud DCS v2

## Operation Map

### Instance Lifecycle

| OperationId | Method | URI | Description |
|-------------|--------|-----|-------------|
| CreateInstance | POST | `/v2/{project_id}/instances` | Create a new DCS instance |
| ShowInstance | GET | `/v2/{project_id}/instances/{instance_id}` | Get instance details |
| UpdateInstance | PUT | `/v2/{project_id}/instances/{instance_id}` | Modify instance config |
| DeleteInstance | DELETE | `/v2/{project_id}/instances/{instance_id}` | Delete instance |
| ListInstances | GET | `/v2/{project_id}/instances` | List instances (paginated) |
| RestartInstance | POST | `/v2/{project_id}/instances/{instance_id}/restart` | Restart instance |
| StopInstance | POST | `/v2/{project_id}/instances/{instance_id}/stop` | Stop instance (pay-per-use) |
| ResizeInstance | POST | `/v2/{project_id}/instances/{instance_id}/resize` | Resize spec/capacity |
| ExtendInstance | POST | `/v2/{project_id}/instances/{instance_id}/extend` | Extend subscription period |
| ShowInstanceStatus | GET | `/v2/{project_id}/instances/{instance_id}/status` | Get instance status |

### Backup & Restore

| OperationId | Method | URI | Description |
|-------------|--------|-----|-------------|
| CreateBackup | POST | `/v2/{project_id}/instances/{instance_id}/backups` | Create manual backup |
| ListBackups | GET | `/v2/{project_id}/instances/{instance_id}/backups` | List backup records |
| DeleteBackup | DELETE | `/v2/{project_id}/instances/{instance_id}/backups/{backup_id}` | Delete backup |
| RestoreInstance | POST | `/v2/{project_id}/instances/{instance_id}/restore` | Restore from backup |

### Password & Whitelist

| OperationId | Method | URI | Description |
|-------------|--------|-----|-------------|
| ResetPassword | POST | `/v2/{project_id}/instances/{instance_id}/password` | Reset Redis AUTH password |
| ShowWhitelist | GET | `/v2/{project_id}/instances/{instance_id}/whitelist` | Get IP whitelist config |
| UpdateWhitelist | PUT | `/v2/{project_id}/instances/{instance_id}/whitelist` | Update IP whitelist |

### Other

| OperationId | Method | URI | Description |
|-------------|--------|-----|-------------|
| ModifyInstanceName | PUT | `/v2/{project_id}/instances/{instance_name}` | Modify instance name |
| BatchStopOrStartInstances | POST | `/v2/{project_id}/instances/batch` | Batch start/stop |
| ListStatistics | GET | `/v2/{project_id}/instances/statistics` | Get account-level instance statistics |
| DeleteMigrationTask | DELETE | `/v2/{project_id}/migration-task/{task_id}` | Delete migration task |

## Required Fields per Operation

### CreateInstance

| Parameter | Type | Required | Default | Notes |
|-----------|------|---------|-------|-------|
| name | string | Yes | — | 4–64 chars, letters/digits/-/_ |
| engine | string | Yes | — | "redis" or "memcached" |
| engine_version | string | Yes | — | Redis: "4.0"/"5.0"/"6.0", Memcached: "1.x" |
| capacity | number | Yes | — | In GB units (capacity_gb) |
| instance_mode | string | Yes | — | "single", "ha", "cluster", "rw" |
| vpc_id | string | Yes | — | Target VPC ID |
| subnet_id | string | Yes | — | Target subnet ID (must be in VPC) |
| security_group_id | string | Yes | — | Security group ID |
| password | string | No (Redis) | — | 8–64 chars, letters+digits+symbols |
| availability_zones | string[] | Yes | — | One or more AZ IDs |
| description | string | No | "" | Instance description |
| enable_ssl | boolean | No | false | Enable TLS for Redis 6.0 |
| port | number | No | 6379 | Custom port (6379, 6380) |
| backup_policy | object | No | — | Auto-backup config (schedule, retention) |
| whitelists | object[] | No | — | IP whitelist entries |

### ShowInstance Response Key Fields

| Field | Path | Type | Description |
|-------|------|------|-------------|
| instance_id | `.instance_id` | string | Unique instance identifier |
| name | `.name` | string | Instance name |
| status | `.status` | string | RUNNING / ERROR / CREATING / etc. |
| engine | `.engine` | string | "redis" or "memcached" |
| engine_version | `.engine_version` | string | Version string |
| capacity | `.capacity` | number | Memory capacity (GB) |
| ip | `.ip` | string | Instance IP address |
| port | `.port` | number | Redis port number |
| vpc_id | `.vpc_id` | string | VPC ID |
| subnet_id | `.subnet_id` | string | Subnet ID |
| security_group_id | `.security_group_id` | string | Security group ID |
| domain_name | `.domain_name` | string | DNS-resolvable hostname |
| created_at | `.created_at` | string | ISO 8601 creation timestamp |
| spec_code | `.spec_code` | string | Instance specification code |
| max_memory_mb | `.max_memory_mb` | number | Maximum memory in MB |

## Pagination Patterns

ListInstances supports pagination via:

```
GET /v2/{project_id}/instances?limit=50&offset=0&include_count=true
```

| Param | Type | Default | Max |
|-------|------|---------|-----|
| limit | int | 100 | 1000 |
| offset | int | 0 | — |
| include_count | bool | false | — |
| name (filter) | string | — | Exact/partial match |
| status (filter) | string | — | RUNNING, ERROR, etc. |

Response includes `instances` array and `instance_count` total.

## Async Operation Behavior

| Operation | Async | Poll Method | Terminal States |
|-----------|-------|-------------|----------------|
| CreateInstance | Yes | ShowInstance → check status | RUNNING (success), ERROR (failure) |
| DeleteInstance | Yes | ShowInstance → 404 = deleted | NotFound (success) |
| ResizeInstance | Yes | ShowInstance → check status | RUNNING (success), ERROR (failure) |
| RestoreInstance | Yes | ShowInstance → check status | RUNNING (success), ERROR (failure) |
| ResetPassword | Yes | ShowInstance → check status | RUNNING (success), ERROR (failure) |

**Polling recipe:**
```bash
for i in $(seq 1 60); do
  STATUS=$(hcloud dcs show-instance --instance-id "$ID" | jq -r '.status')
  printf "\r⏳ Processing... [%3ds] Status: %s" $((i*5)) "$STATUS"
  [ "$STATUS" = "RUNNING" ] && echo "" && break
  sleep 5
done
```

Default interval: **5s**. Max wait: **300s** (5 minutes).

## Request/Response Snippets

### CreateInstance Request Body

```json
{
  "name": "my-redis-instance",
  "engine": "redis",
  "engine_version": "6.0",
  "capacity": 4,
  "instance_mode": "ha",
  "vpc_id": "vpc-abc123",
  "subnet_id": "subnet-def456",
  "security_group_id": "sg-ghi789",
  "password": "SecureP@ss2024",
  "availability_zones": ["az1a", "az1b"],
  "enable_ssl": true,
  "backup_policy": {
    "save_days": 3,
    "period_type": "weekly",
    "backup_at": [1, 3, 5, 7],
    "begin_at": "02:00-04:00"
  }
}
```

### CreateInstance Response

```json
{
  "instance_id": "dcs-0a1b2c3d",
  "order_id": "CS2401011234567890"
}
```

## SDK Initialization

```go
cfg := config.DefaultHttpConfig()
client := dcs.NewDcsClient(
    dcs.DcsClientBuilder().
        WithEndpoint(fmt.Sprintf("dcs.%s.myhuaweicloud.com", region)).
        WithCredential(basic.NewCredentialsBuilder().
            WithAk(os.Getenv("HW_ACCESS_KEY_ID")).
            WithSk(os.Getenv("HW_SECRET_ACCESS_KEY")).
            Build()).
        WithHttpConfig(cfg).
        Build())
```

## Error Response Shape

```json
{
  "error_code": "DCS.0002",
  "error_msg": "The instance could not be found.",
  "request_id": "abc123-def456-ghi789"
}
```
