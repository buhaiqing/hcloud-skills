# API & SDK Usage — Huawei Cloud LTS

## API Service Profile

| Item | Value |
|------|-------|
| Service | LTS (Log Tank Service) |
| API Version | v2 |
| Base URL | `https://lts.{{region}}.myhuaweicloud.com/v2` |
| SDK Package | `github.com/huaweicloud/huaweicloud-sdk-go-v3/services/lts/v2` |
| Authentication | AK/SK via IAM |

## Operation Map

| Resource | Operation | API Name | SDK Function | CLI Command |
|----------|-----------|----------|-------------|-------------|
| Log Group | Create | `CreateLogGroup` | `CreateLogGroup` | `hcloud LTS CreateLogGroup` |
| Log Group | List | `ListLogGroups` | `ListLogGroups` | `hcloud LTS ListLogGroups` |
| Log Group | Update (TTL) | `UpdateLogGroup` | `UpdateLogGroup` | `hcloud LTS UpdateLogGroup` |
| Log Group | Delete | `DeleteLogGroup` | `DeleteLogGroup` | `hcloud LTS DeleteLogGroup` |
| Log Stream | Create | `CreateLogStream` | `CreateLogStream` | `hcloud LTS CreateLogStream` |
| Log Stream | List (all) | `ListLogStreams` | `ListLogStreams` | `hcloud LTS ListLogStreams` |
| Log Stream | List (by group) | `ListLogStream` | `ListLogStream` | `hcloud LTS ListLogStream` |
| Log Stream | Delete | `DeleteLogStream` | `DeleteLogStream` | `hcloud LTS DeleteLogStream` |
| Log | Search/Query | `ListLogs` | `ListLogs` | `hcloud LTS ListLogs` |
| Log | High-precision ingest | `CreateLogDumpObs` | `CreateLogDumpObs` | `hcloud LTS CreateLogDumpObs` |
| Transfer | Create | `CreateTransfer` | `CreateTransfer` | `hcloud LTS CreateTransfer` |
| Transfer | List | `ListTransfers` | `ListTransfers` | `hcloud LTS ListTransfers` |
| Transfer | Delete | `DeleteTransfer` | `DeleteTransfer` | `hcloud LTS DeleteTransfer` |
| Dashboard | Create | `CreateDashboard` | `CreateDashboard` | `hcloud LTS CreateDashboard` |
| Dashboard | List | `ListDashboards` | `ListDashboards` | `hcloud LTS ListDashboards` |
| Dashboard | Delete | `DeleteDashboard` | `DeleteDashboard` | `hcloud LTS DeleteDashboard` |
| Quick Search | Create | `CreateQuickSearch` | `CreateQuickSearch` | `hcloud LTS CreateQuickSearch` |
| Quick Search | List | `ListQuickSearch` | `ListQuickSearch` | `hcloud LTS ListQuickSearch` |

## Pagination

List operations support pagination via `limit` and `offset` parameters:

```json
// Request: ListLogGroups with pagination
GET /v2/{project_id}/groups?limit=10&offset=0

// Response
{
  "log_groups": [ ... ],
  "total_count": 47
}
```

For `ListLogs` (search), pagination uses cursor-based approach via `line_num`, `is_desc`, and `search_type`:

```json
// First query (init)
{
  "start_time": 1700000000000,
  "end_time": 1700086400000,
  "keywords": "ERROR",
  "limit": 100,
  "is_count": true
}

// Subsequent page (forward)
{
  "start_time": 1700000000000,
  "end_time": 1700086400000,
  "keywords": "ERROR",
  "line_num": "1700050000000123456",
  "is_desc": "false",
  "search_type": "forwards",
  "limit": 100
}
```

## Required vs Optional Fields

| API | Required Fields | Optional Fields |
|-----|----------------|-----------------|
| `CreateLogGroup` | `log_group_name`, `ttl_in_days` | `tags`, `enterprise_project_id` |
| `CreateLogStream` | `log_group_id` (path), `log_stream_name` | — |
| `ListLogs` | `log_group_id`, `log_stream_id`, `start_time`, `end_time` | `keywords`, `labels`, `line_num`, `is_desc`, `search_type`, `limit`, `is_count` |
| `CreateTransfer` | `log_group_id`, `log_stream_ids`, `obs_bucket_name` | `obs_period`, `obs_dir_prefix`, `dms_transfer_detail` |

## Async Operations

LTS operations are synchronous (immediate response). No polling required.

## Error Response Format

```json
{
  "error_code": "LTS.0101",
  "error_msg": "The number of log groups has reached the maximum limit."
}
```

## Log Ingestion SDK

For production log ingestion, use the dedicated LTS Go SDK:

```
go get github.com/huaweicloud/huaweicloud-lts-sdk-go
```

```go
import "github.com/huaweicloud/huaweicloud-lts-sdk-go/producer"

config := producer.GetDefaultConfig()
config.ProjectId = os.Getenv("HW_PROJECT_ID")
config.AccessKeyId = os.Getenv("HW_ACCESS_KEY_ID")
config.AccessKeySecret = os.Getenv("HW_SECRET_ACCESS_KEY")
config.RegionName = os.Getenv("HW_REGION_ID")

producerInstance := producer.NewProducer(config)
log := &producer.Log{
    LogTime: time.Now().UnixMilli(),
    Content: "{\"level\":\"ERROR\",\"message\":\"Connection timeout\"}",
}
err := producerInstance.SendLog(groupId, streamId, log)
```
