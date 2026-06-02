# Integration — BSS (费用中心)

## JIT Go SDK Bootstrap

When CLI is unavailable or insufficient, use Go JIT (Just-In-Time) SDK fallback:

```go
package main

import (
    "fmt"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    bss "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/bss/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/bss/v2/model"
)

func main() {
    auth := basic.NewCredentialsBuilder().
        WithAk(os.Getenv("HW_ACCESS_KEY_ID")).
        WithSk(os.Getenv("HW_SECRET_ACCESS_KEY")).
        Build()

    client := bss.NewBssClient(
        bss.BssClientBuilder().
            WithEndpoint(os.Getenv("HW_BSS_ENDPOINT")).
            WithCredential(auth).
            WithHttpConfig(config.DefaultHttpConfig()).
            Build())

    req := &model.ShowCustomerAccountInfoRequest{}
    resp, err := client.ShowCustomerAccountInfo(req)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Balance: %s %s\n", resp.AccountBalances[0].Amount, resp.AccountBalances[0].Currency)
}
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `HW_ACCESS_KEY_ID` | Yes | — | Huawei Cloud AK |
| `HW_SECRET_ACCESS_KEY` | Yes | — | Huawei Cloud SK |
| `HW_REGION_ID` | Yes | — | Region (e.g., cn-north-4) |
| `HW_PROJECT_ID` | No | — | Project ID for bill scoping |
| `HW_BSS_ENDPOINT` | No | `bss.myhuaweicloud.com` | BSS API endpoint |

## Dependencies

```go
import (
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/bss/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
)
```

Install: `go get github.com/huaweicloud/huaweicloud-sdk-go-v3`

## Cross-Skill Delegation Matrix

This skill is the FinOps anchor. It delegates physical operations to product skills.

| Delegation Target | When | Operations |
|------------------|------|------------|
| `huaweicloud-ecs-ops` | P1 idle ECS, P2 resize ECS, P8 zombie ECS | stop, resize, delete |
| `huaweicloud-rds-ops` | P1 idle RDS, P2 resize RDS | stop, resize |
| `huaweicloud-obs-ops` | P5 storage tiering | lifecycle policy, archive |
| `huaweicloud-lts-ops` | P6 log retention | delete log stream, archive |
| `huaweicloud-ces-ops` | Monitoring integration for budget alerts | alarm query |
| `huaweicloud-cts-ops` | Audit trail for cost anomalies | trace query |

## Optimization Backlog Integration

The optimization backlog and closed-loop tracker use file-based storage at `~/.hcloud/`. The directory structure:

```
~/.hcloud/
├── optimization_backlog/
│   └── {cycle_id}/
│       └── patterns/
│           ├── P1_idle_ecs.json
│           ├── P2_rightsize.json
│           └── ...
├── optimization_tracker.jsonl        # Op 14 log
├── maturity_scorecard.json           # Op 15 score
└── config.yaml                       # User overrides
```