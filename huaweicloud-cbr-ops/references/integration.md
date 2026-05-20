# Integration — Huawei Cloud CBR

## JIT Go SDK Setup

```bash
mkdir -p /tmp/cbr-jit && cd /tmp/cbr-jit
go mod init cbr-jit
go get github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cbr/v3
go get github.com/huaweicloud/huaweicloud-sdk-go-v3/core
```

## Cross-Skill Delegation Matrix

| Scenario | Lead Skill | Delegates To | When |
|----------|-----------|-------------|------|
| ECS backup failure | `huaweicloud-cbr-ops` | `huaweicloud-ecs-ops` | ECS instance not found or in invalid state |
| RDS backup/restore | `huaweicloud-cbr-ops` | `huaweicloud-rds-ops` | RDS native backup management |
| DCS backup/restore | `huaweicloud-cbr-ops` | `huaweicloud-dcs-ops` | DCS backup management |
| Permission denied | `huaweicloud-cbr-ops` | `huaweicloud-iam-ops` | IAM policy issue |
| Backup metric anomaly | `huaweicloud-cbr-ops` | `huaweicloud-ces-ops` | Alarm or metric analysis |
| Audit trail | `huaweicloud-cbr-ops` | `huaweicloud-cts-ops` | Backup delete/restore tracking |

## Environment Variables

```bash
export HW_ACCESS_KEY_ID="your-access-key"
export HW_SECRET_ACCESS_KEY="your-secret-key"
export HW_REGION_ID="cn-north-4"
export HW_PROJECT_ID="your-project-id"
export CBR_POLL_INTERVAL="10"       # Backup status polling (seconds)
export CBR_BACKUP_TIMEOUT="3600"    # Max wait for backup (seconds)
```

## Go Module Bootstrap

```go
package main

import (
    "fmt"
    "os"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cbr/v3"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cbr/v3/model"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cbr/v3/region"
)

func main() {
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
    req := &model.ListVaultsReq{}
    resp, err := client.ListVaults(req)
    if err != nil {
        fmt.Println("Error:", err)
        os.Exit(1)
    }
    for _, v := range resp.Vaults {
        fmt.Printf("%s: %dGB/%dGB (%s)\n", *v.Name, *v.Billing.Used, *v.Billing.Size, *v.Billing.Status)
    }
}
```
