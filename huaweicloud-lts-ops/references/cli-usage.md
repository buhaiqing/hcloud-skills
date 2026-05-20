# CLI Usage — Huawei Cloud LTS

## Overview

KooCLI (`hcloud`) supports LTS via API Explorer operation names. Each Huawei Cloud API is exposed as `hcloud LTS <OperationName>`.

## Command Map

| Operation | CLI Command | Notes |
|-----------|-------------|-------|
| Create log group | `hcloud LTS CreateLogGroup --log_group_name="my-group" --ttl_in_days=30` | Project ID from profile or `--project_id` |
| List log groups | `hcloud LTS ListLogGroups` | Returns all groups in project |
| Update log group TTL | `hcloud LTS UpdateLogGroup --log_group_id="{{id}}" --ttl_in_days=60` | Uses PUT |
| Delete log group | `hcloud LTS DeleteLogGroup --log_group_id="{{id}}"` | Cannot be undone |
| Create log stream | `hcloud LTS CreateLogStream --log_group_id="{{id}}" --log_stream_name="my-stream"` | |
| List log streams | `hcloud LTS ListLogStreams --log_group_name="my-group"` | |
| Delete log stream | `hcloud LTS DeleteLogStream --log_group_id="{{id}}" --log_stream_id="{{stream_id}}"` | |
| Search logs | `hcloud LTS ListLogs --log_group_id="{{id}}" --log_stream_id="{{stream_id}}" --start_time=1700000000000 --end_time=1700086400000 --keywords="ERROR"` | Times in epoch ms |
| Create transfer | `hcloud LTS CreateTransfer --log_group_id="{{id}}" --log_stream_ids="[{{stream_id}}]" --obs_bucket_name="my-bucket"` | |
| List transfers | `hcloud LTS ListTransfers` | |
| Delete transfer | `hcloud LTS DeleteTransfer --log_transfer_id="{{transfer_id}}"` | |
| Create dashboard | `hcloud LTS CreateDashboard --title="My Dashboard"` | |
| List dashboards | `hcloud LTS ListDashboards` | |

## Common CLI Patterns

### Using a specific project/region
```bash
hcloud LTS ListLogGroups --cli-region="cn-north-4" --project_id="{{env.HW_PROJECT_ID}}"
```

### JSON output for parsing
```bash
hcloud LTS ListLogGroups --cli-region="cn-north-4" --cli-output=json
```

### Filtering output with jq
```bash
hcloud LTS ListLogGroups --cli-region="cn-north-4" --cli-output=json | jq '.log_groups[] | {name: .log_group_name, id: .log_group_id, ttl: .ttl_in_days}'
```

### Skeleton generation (write parameters to JSON file)
```bash
hcloud LTS CreateLogGroup --skeleton
# Writes skeleton JSON to ./skeleton.json
hcloud LTS CreateLogGroup --cli-jsonInput=./skeleton.json
```

## Coverage Gap Table

| Feature | CLI Support | SDK Support | Notes |
|---------|-------------|-------------|-------|
| Log Group CRUD | ✅ Full | ✅ Full | All operations matched |
| Log Stream CRUD | ✅ Full | ✅ Full | All operations matched |
| Log Search | ✅ Full | ✅ Full | Cursor-based pagination fully supported |
| Log Transfer | ✅ Full | ✅ Full | Create/List/Delete |
| Dashboard | ✅ Full | ✅ Full | Create/List/Delete |
| Quick Search | ✅ Full | ✅ Full | Create/List/Delete |
| Structured Parsing Config | ⚠️ Partial | ✅ Full | CLI via UpdateLogStream only |
| ICAgent Management | ❌ No | ⚠️ Partial | Use console or ECS commands |

## Error Handling in CLI

CLI errors return JSON with `error_code` and `error_msg`:

```bash
$ hcloud LTS CreateLogGroup --log_group_name="" --ttl_in_days=30
{
  "error_code": "LTS.0001",
  "error_msg": "Invalid parameter: log_group_name cannot be empty"
}
```

Use `--debug` flag to see raw request/response:
```bash
hcloud LTS ListLogGroups --debug
```
