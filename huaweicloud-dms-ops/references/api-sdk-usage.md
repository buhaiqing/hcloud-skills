# API & SDK Usage — Huawei Cloud DMS

## API Service Profile

| Item | Value |
|------|-------|
| Service | DMS (Distributed Message Service) |
| API Version | v2 (Kafka + RabbitMQ) |
| Base URL | `https://dms.{{region}}.myhuaweicloud.com/v2` |
| SDK Package | `github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dms/v2` |
| Authentication | AK/SK via IAM |

## Operation Map

| Resource | Operation | API Name | SDK Function | CLI Command |
|----------|-----------|----------|-------------|-------------|
| Instance | Create | `CreateInstance` | `CreateInstance` | `hcloud DMS CreateInstance` |
| Instance | Show | `ShowInstance` | `ShowInstance` | `hcloud DMS ShowInstance` |
| Instance | List | `ListInstances` | `ListInstances` | `hcloud DMS ListInstances` |
| Instance | Update | `UpdateInstance` | `UpdateInstance` | `hcloud DMS UpdateInstance` |
| Instance | Delete | `DeleteInstance` | `DeleteInstance` | `hcloud DMS DeleteInstance` |
| Topic | Create | `CreateTopic` | `CreateTopic` | `hcloud DMS CreateTopic` |
| Topic | List | `ListTopics` | `ListTopics` | `hcloud DMS ListTopics` |
| Topic | Delete | `DeleteTopic` | `DeleteTopic` | `hcloud DMS DeleteTopic` |
| Queue | Create | `CreateQueue` | `CreateQueue` | `hcloud DMS CreateQueue` |
| Queue | List | `ListQueues` | `ListQueues` | `hcloud DMS ListQueues` |
| Consumer Group | List | `ListConsumerGroups` | `ListConsumerGroups` | `hcloud DMS ListConsumerGroups` |
| Consumer Group | Show Lag | `ShowConsumerGroupLag` | `ShowConsumerGroupLag` | `hcloud DMS ShowConsumerGroupLag` |
| Backup | Create | `CreateBackup` | `CreateBackup` | `hcloud DMS CreateBackup` |
| Backup | List | `ListBackups` | `ListBackups` | `hcloud DMS ListBackups` |
| Backup | Restore | `RestoreInstance` | `RestoreInstance` | `hcloud DMS RestoreInstance` |

## Pagination

All list operations support pagination via `limit` and `offset` parameters:

```json
// Request
{
  "limit": 10,
  "offset": 0
}

// Response
{
  "instances": [...],
  "total_count": 45,
  "limit": 10,
  "offset": 0
}
```

## Required Fields Per Operation

### CreateInstance

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `name` | string | Yes | 4-64 chars, unique per account per region |
| `engine` | string | Yes | `kafka` or `rabbitmq` |
| `engine_version` | string | Yes | e.g., `2.7` for Kafka, `3.x.x` for RabbitMQ |
| `specification` | string | Yes | Spec code from instance types API |
| `storage_space` | int | Yes | Per broker, in GB |
| `broker_num` | int | Yes | Number of brokers (3-30) |
| `vpc_id` | string | Yes | VPC to deploy into |
| `subnet_id` | string | Yes | Subnet within VPC |
| `security_group_id` | string | Yes | SG controlling network access |
| `maintain_begin` | string | No | Maintenance window start: `HH:mm` |
| `maintain_end` | string | No | Maintenance window end: `HH:mm` |
| `enable_auto_topic` | bool | No | Kafka: auto-create topics |

### Response Schema (CreateInstance)

```json
{
  "instance_id": "dms-abc123",
  "name": "prod-kafka-cluster",
  "status": "CREATING",
  "engine": "kafka",
  "specification": "kafka.4u8g.cluster",
  "broker_num": 3,
  "storage_space": 500,
  "vpc_id": "vpc-abc123",
  "created_at": "2026-05-21T10:00:00Z"
}
```

## JIT Go SDK Runtime

```go
// Setup
import (
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dms/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dms/v2/model"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dms/v2/region"
)

auth := basic.NewCredentialsBuilder().
    WithAk(os.Getenv("HW_ACCESS_KEY_ID")).
    WithSk(os.Getenv("HW_SECRET_ACCESS_KEY")).
    WithProjectId(os.Getenv("HW_PROJECT_ID")).
    Build()

client := dms.NewDmsClient(
    dms.DmsClientBuilder().
        WithRegion(region.ValueOf(os.Getenv("HW_REGION_ID"))).
        WithCredential(auth).
        Build(),
)
```
