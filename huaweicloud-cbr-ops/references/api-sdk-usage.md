# API & SDK Usage — Huawei Cloud CBR

## API Service Profile

| Item | Value |
|------|-------|
| Service | CBR (Cloud Backup and Recovery) |
| API Version | v3 |
| Base URL | `https://cbr.{{region}}.myhuaweicloud.com/v3` |
| SDK Package | `github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cbr/v3` |
| Authentication | AK/SK via IAM |

## Operation Map

| Resource | Operation | API Name | SDK Function | CLI Command |
|----------|-----------|----------|-------------|-------------|
| Vault | Create | `CreateVault` | `CreateVault` | `hcloud CBR CreateVault` |
| Vault | Show | `ShowVault` | `ShowVault` | `hcloud CBR ShowVault` |
| Vault | List | `ListVaults` | `ListVaults` | `hcloud CBR ListVaults` |
| Vault | Update | `UpdateVault` | `UpdateVault` | `hcloud CBR UpdateVault` |
| Vault | Delete | `DeleteVault` | `DeleteVault` | `hcloud CBR DeleteVault` |
| Policy | Create | `CreatePolicy` | `CreatePolicy` | `hcloud CBR CreatePolicy` |
| Policy | List | `ListPolicies` | `ListPolicies` | `hcloud CBR ListPolicies` |
| Policy | Update | `UpdatePolicy` | `UpdatePolicy` | `hcloud CBR UpdatePolicy` |
| Policy | Delete | `DeletePolicy` | `DeletePolicy` | `hcloud CBR DeletePolicy` |
| Backup | Create | `CreateBackup` | `CreateBackup` | `hcloud CBR CreateBackup` |
| Backup | List | `ListBackups` | `ListBackups` | `hcloud CBR ListBackups` |
| Backup | Show | `ShowBackup` | `ShowBackup` | `hcloud CBR ShowBackup` |
| Backup | Delete | `DeleteBackup` | `DeleteBackup` | `hcloud CBR DeleteBackup` |
| Backup | Restore | `RestoreBackup` | `RestoreBackup` | `hcloud CBR RestoreBackup` |
| Replication | Create | `ReplicateBackup` | `ReplicateBackup` | `hcloud CBR ReplicateBackup` |

## Required Fields Per Operation

### CreateVault

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `name` | string | Yes | 4-64 chars, unique per account |
| `object_type` | string | Yes | `server`, `disk`, `turbo`, or `workspace` |
| `size` | int | Yes | Vault capacity in GB |
| `charging_mode` | string | No | `prePaid` or `postPaid` (default) |
| `period_type` | string | No | `month` or `year` (if prePaid) |
| `period_num` | int | No | 1-36 (if prePaid) |
| `auto_renew` | bool | No | Auto-renew for prePaid |

### Response Schema (CreateVault)

```json
{
  "vault": {
    "id": "vault-abc123",
    "name": "prod-ecs-backup",
    "billing": {
      "object_type": "server",
      "size": 1000,
      "used": 0,
      "status": "available",
      "charging_mode": "postPaid"
    },
    "resources": [],
    "created_at": "2026-05-21T10:00:00Z"
  }
}
```

## JIT Go SDK Runtime

```go
import (
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cbr/v3"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cbr/v3/model"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cbr/v3/region"
)

auth := basic.NewCredentialsBuilder().
    WithAk(os.Getenv("HW_ACCESS_KEY_ID")).
    WithSk(os.Getenv("HW_SECRET_ACCESS_KEY")).
    WithProjectId(os.Getenv("HW_PROJECT_ID")).
    Build()

client := cbr.NewCbrClient(
    cbr.CbrClientBuilder().
        WithRegion(region.ValueOf(os.Getenv("HW_REGION_ID"))).
        WithCredential(auth).
        Build(),
)
```
