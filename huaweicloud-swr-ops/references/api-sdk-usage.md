# API & SDK Usage — Huawei Cloud SWR

## API Service Profile

| Item | Value |
|------|-------|
| Service | SWR (Software Repository for Container) |
| API Version | v2 |
| Base URL | `https://swr-api.{{region}}.myhuaweicloud.com/v2` |
| SDK Package | `github.com/huaweicloud/huaweicloud-sdk-go-v3/services/swr/v2` |
| Authentication | AK/SK via IAM |

## Operation Map

| Resource | Operation | API Name | SDK Function | CLI Command |
|----------|-----------|----------|-------------|-------------|
| Organization | Create | `CreateOrganization` | `CreateOrganization` | `hcloud SWR CreateOrganization` |
| Organization | List | `ListOrganizations` | `ListOrganizations` | `hcloud SWR ListOrganizations` |
| Organization | Delete | `DeleteOrganization` | `DeleteOrganization` | `hcloud SWR DeleteOrganization` |
| Repository | Create | `CreateRepository` | `CreateRepository` | `hcloud SWR CreateRepository` |
| Repository | List | `ListRepositories` | `ListRepositories` | `hcloud SWR ListRepositories` |
| Repository | Delete | `DeleteRepository` | `DeleteRepository` | `hcloud SWR DeleteRepository` |
| Image | List | `ListImages` | `ListImages` | `hcloud SWR ListImages` |
| Image | Delete | `DeleteImageTag` | `DeleteImageTag` | `hcloud SWR DeleteImageTag` |
| Policy | Create | `CreateRetentionPolicy` | `CreateRetentionPolicy` | `hcloud SWR CreateRetentionPolicy` |
| Policy | List | `ListRetentionPolicies` | `ListRetentionPolicies` | `hcloud SWR ListRetentionPolicies` |
| Sync | Create | `CreateImageSync` | `CreateImageSync` | `hcloud SWR CreateImageSync` |
| Sync | List | `ListImageSync` | `ListImageSync` | `hcloud SWR ListImageSync` |
| Auth | Token | `GenerateLoginToken` | `GenerateLoginToken` | `hcloud SWR GenerateLoginToken` |

## Required Fields Per Operation

### CreateRepository

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `namespace` | string | Yes | Organization name |
| `name` | string | Yes | Repository name, lowercase |
| `category` | string | Yes | `app_server` or `linux` |
| `description` | string | No | Repository description |
| `is_public` | bool | No | Default: false |

### Response Schema (CreateRepository)

```json
{
  "id": 12345,
  "name": "nginx",
  "category": "app_server",
  "description": "Production nginx images",
  "size": 0,
  "num_images": 0,
  "created_at": "2026-05-21T10:00:00Z"
}
```

### ListImages Response

```json
{
  "body": [
    {
      "name": "1.25",
      "size": "128456789",
      "digest": "sha256:abc...",
      "pushed_at": "2026-05-21T10:00:00Z",
      "pull_count": 150,
      "is_vulnerable": false
    }
  ]
}
```

## JIT Go SDK Runtime

```go
import (
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/swr/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/swr/v2/model"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/swr/v2/region"
)

auth := basic.NewCredentialsBuilder().
    WithAk(os.Getenv("HW_ACCESS_KEY_ID")).
    WithSk(os.Getenv("HW_SECRET_ACCESS_KEY")).
    WithProjectId(os.Getenv("HW_PROJECT_ID")).
    Build()

client := swr.NewSwrClient(
    swr.SwrClientBuilder().
        WithRegion(region.ValueOf(os.Getenv("HW_REGION_ID"))).
        WithCredential(auth).
        Build(),
)
```
