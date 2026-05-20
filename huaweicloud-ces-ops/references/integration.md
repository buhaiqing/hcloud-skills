# CES Integration — Huawei Cloud Cloud Eye Service

## JIT Go SDK Setup

### Dependencies

```go
go mod init ces-ops-script

require github.com/huaweicloud/huaweicloud-sdk-go-v3 v0.1.123
```

### Bootstrap Script

```bash
#!/bin/bash
# JIT Go SDK setup for CES operations

if ! command -v go &> /dev/null; then
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    [ "$ARCH" = "x86_64" ] && ARCH="amd64"
    [ "$ARCH" = "aarch64" ] && ARCH="arm64"

    mkdir -p /tmp/go-runtime
    curl -fsSL "https://go.dev/dl/go1.25.0.${OS}-${ARCH}.tar.gz" | tar -xz -C /tmp/go-runtime
    export PATH="/tmp/go-runtime/go/bin:$PATH"
    export GOPROXY="https://goproxy.cn,direct"
    export GOPATH="/tmp/go-workspace"
    export GOCACHE="/tmp/go-cache"
fi

echo "Go version: $(go version)"
```

### Minimum Script Example

```go
package main

import (
    "fmt"
    "os"

    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ces/v1"
)

func main() {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    region := os.Getenv("HW_REGION_ID")

    client, err := v1.NewCesClient(
        v1.CesClientBuilder().
            WithEndpoint(fmt.Sprintf("ces.%s.myhuaweicloud.com", region)).
            WithCredential(basic.NewCredentialsBuilder().
                WithAk(ak).WithSk(sk).Build()).
            Build(),
    )
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to create CES client: %v\n", err)
        os.Exit(1)
    }

    // Use client operations here
    fmt.Println("CES client initialized successfully")
}
```

## Credential Configuration

- **Environment variables**: `HW_ACCESS_KEY_ID`, `HW_SECRET_ACCESS_KEY`, `HW_REGION_ID`, `HW_PROJECT_ID`
- **MUST NOT**: Hardcode credentials, print secret keys in any output
- **Verification**: `test -n "$HW_SECRET_ACCESS_KEY" && echo "✅ Credentials OK"` (existence check only)

## Cross-Skill Delegation Matrix

| Alarm Type | CES Metric | Primary Skill | Secondary Skill | AIOps Delegation |
|-----------|-----------|---------------|-----------------|------------------|
| CPU高 | SYS.ECS > cpu_util | huaweicloud-ecs-ops | huaweicloud-cce-ops (if container) | Recommended |
| 内存泄漏 | AGT.ECS > memory_util | huaweicloud-ecs-ops | — | Required |
| 磁盘满 | AGT.ECS > disk_util | huaweicloud-ecs-ops | huaweicloud-evs-ops (I/O) | Recommended |
| 数据库连接耗尽 | SYS.RDS > rds003_conn_usage | huaweicloud-rds-ops | — | Required |
| 带宽饱和 | SYS.VPC > bandwidth_util | huaweicloud-vpc-ops | huaweicloud-ces-ops | Recommended |
| ELB错误率 | SYS.ELB > 5xx_error_count | huaweicloud-elb-ops | huaweicloud-ecs-ops (backend health) | Recommended |
| 安全告警 | SYS.HSS events | huaweicloud-hss-ops | huaweicloud-ecs-ops (isolation) | Required |
| SMN投递失败 | CES alarm_actions | huaweicloud-smn-ops | — | Optional |

| Scenario | Delegating Skill | Target Skill | Delegation Type |
|----------|-----------------|--------------|-----------------|
| Alarm for ECS CPU > 90% | huaweicloud-ces-ops | huaweicloud-ecs-ops | Resource verification |
| Alarm for RDS connection pool | huaweicloud-ces-ops | huaweicloud-rds-ops | Resource verification |
| SMN topic creation for alarms | huaweicloud-ces-ops | huaweicloud-smn-ops (when present) | Notification setup |
| Cost analysis of monitoring | huaweicloud-ces-ops | huaweicloud-billing-ops (when present) | FinOps query |
| Permission setup for CES access | huaweicloud-ces-ops | huaweicloud-iam-ops | IAM configuration |

## Environment Variables Reference

| Variable | Required | Description |
|----------|----------|-------------|
| `HW_ACCESS_KEY_ID` | Yes | IAM access key |
| `HW_SECRET_ACCESS_KEY` | Yes | IAM secret key (NEVER log/echo) |
| `HW_REGION_ID` | Yes | Target region (e.g., cn-north-4) |
| `HW_PROJECT_ID` | Yes | Project ID for API routing |
| `GOPROXY` | JIT only | Go module proxy (recommended: https://goproxy.cn,direct) |
