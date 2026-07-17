# Integration — Huawei Cloud LTS

## JIT Go SDK Setup

### Bootstrap

```go
package main

import (
    "fmt"
    "os"

    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/region"
    lts "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/lts/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/lts/v2/model"
)

func main() {
    auth := basic.NewCredentialsBuilder().
        WithAk(os.Getenv("HW_ACCESS_KEY_ID")).
        WithSk(os.Getenv("HW_SECRET_ACCESS_KEY")).
        Build()

    client := lts.NewLtsClient(
        lts.LtsClientBuilder().
            WithRegion(region.ValueOf(os.Getenv("HW_REGION_ID"))).
            WithCredential(auth).
            WithHttpConfig(config.DefaultHttpConfig().WithIgnoreSSLVerification(false)).
            Build())

    // Example: List Log Groups
    resp, err := client.ListLogGroups(&model.ListLogGroupsRequest{})
    if err != nil {
        fmt.Printf("[ERROR] %v\n", err)
        return
    }
    for _, g := range *resp.LogGroups {
        fmt.Printf("Group: %s (ID: %s, TTL: %d)\n",
            g.LogGroupName, g.LogGroupId, g.TtlInDays)
    }
}
```

### Dependencies

```
require github.com/huaweicloud/huaweicloud-sdk-go-v3 v0.1.187+
```

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `HW_ACCESS_KEY_ID` | Yes | Huawei Cloud AK |
| `HW_SECRET_ACCESS_KEY` | Yes | Huawei Cloud SK |
| `HW_REGION_ID` | Yes | Region (e.g., cn-north-4) |
| `HW_PROJECT_ID` | Yes | Project ID |

## LTS Log Ingestion SDK

For production log ingestion workloads, use the dedicated LTS production SDK:

```go
import "github.com/huaweicloud/huaweicloud-lts-sdk-go/producer"
```

## Cross-Skill Delegation Matrix

| Scenario | Trigger | Delegate To | Expected Outcome |
|----------|---------|-------------|-----------------|
| Log search timeout | User reports slow LTS query | `huaweicloud-ces-ops` | Check `lts_log_search_latency` metric |
| OBS transfer configured but no output | User says "logs not in OBS" | `huaweicloud-obs-ops` | Verify bucket exists and policy allows `PutObject` |
| ICAgent not collecting | User says "no logs in stream" | `huaweicloud-ecs-ops` | Check ECS host status and ICAgent process |
| K8s container logs missing | User says "no pod logs in LTS" | `huaweicloud-cce-ops` | Check CCE log collection addon/daemonset |
| Permission denied | User sees LTS.0202 | `huaweicloud-iam-ops` | Check IAM policies for `lts:*` permissions |
| Audit trail needed | User asks "who deleted the log group" | `huaweicloud-cts-ops` | Query CTS for LTS DeleteLogGroup events |
| DMS transfer target | User wants DMS as transfer target | `huaweicloud-dms-ops` | Verify DMS instance exists and is accessible |

## Security Integration

### Credential Masking

All sensitive values MUST be masked in output:
- AK: Show only last 4 chars (`****1234`)
- SK: Never displayed
- Project ID: Show full (not sensitive alone but treated as confidential)

### Network Security

- LTS API endpoint uses HTTPS only (TLS 1.2+)
- ICAgent → LTS communication is encrypted
- For private network scenarios, use VPC Endpoint (VPCEP) for LTS
- Transfer to OBS within same region stays on Huawei Cloud internal network

### Encryption

| Data State | Encryption | Notes |
|-----------|------------|-------|
| At rest (LTS storage) | AES-256 | Default, not configurable |
| In transit (ingestion) | TLS 1.2+ | HTTPS endpoints |
| In transit (transfer) | TLS 1.2+ | To OBS/DMS |
| At rest (OBS target) | SSE-OBS or SSE-KMS | Configure in OBS bucket |

## AIOps Integration

### Alarm Storm Delegation

| Scenario | Trigger | Delegate To | Expected Outcome |
|----------|---------|-------------|------------------|
| High frequency LTS alarms | > 10 alarms in 5 min | `huaweicloud-ces-ops` | Configure aggregated alarm rule |
| Storage quota alarm storm | Multiple groups near quota | `huaweicloud-ces-ops` | Set up storage quota monitoring |
| Ingestion spike alarm | Rate > 3x baseline | `huaweicloud-ces-ops` | Set up ingestion rate baseline alert |

### Multi-Metric Correlation

| Metric Pair | Correlation | Delegate To | Interpretation |
|------------|-------------|-------------|----------------|
| Ingestion Rate + Storage | Positive | `huaweicloud-ces-ops` | Normal growth, monitor trend |
| Query Latency + Queue Depth | Positive | `huaweicloud-ces-ops` | Backpressure detected |
| Error Rate + Ingestion | Negative | `huaweicloud-ces-ops` | Data quality issue |

### Anomaly Pattern Routing

| Pattern | Detection | Delegate To | Expected Action |
|---------|-----------|-------------|-----------------|
| `log_ingestion_quota_near` | Ingestion > 80% quota | `huaweicloud-ces-ops` | Set quota usage alarm |
| `storage_quota_near` | Storage > 80% quota | `huaweicloud-ces-ops` | Set storage usage alarm |
| `shard_capacity_near` | Shard > 85% | `huaweicloud-ces-ops` | Set shard usage alarm |
| `error_rate_spike` | Errors > 1% | `huaweicloud-ces-ops` | Set error rate alarm |
