# VPC Integration — Huawei Cloud Virtual Private Cloud

## JIT Go SDK Setup

### Dependencies

VPC uses multiple sub-packages:

```go
require (
    github.com/huaweicloud/huaweicloud-sdk-go-v3 v0.1.123
)

import (
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2"
    vpc_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2/model"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/nat/v2"
    nat_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/nat/v2/model"
)
```

### Bootstrap Script

```bash
#!/bin/bash
if ! command -v go &> /dev/null; then
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    [ "$ARCH" = "x86_64" ] && ARCH="amd64"
    [ "$ARCH" = "aarch64" ] && ARCH="arm64"
    mkdir -p /tmp/go-runtime
    curl -fsSL "https://go.dev/dl/go1.25.0.${OS}-${ARCH}.tar.gz" | tar -xz -C /tmp/go-runtime
    export PATH="/tmp/go-runtime/go/bin:$PATH"
    export GOPROXY="https://goproxy.cn,direct"
fi
```

## Credential Configuration

- **Environment variables**: `HW_ACCESS_KEY_ID`, `HW_SECRET_ACCESS_KEY`, `HW_REGION_ID`, `HW_PROJECT_ID`
- **MUST NOT**: Hardcode credentials, print secret keys
- **Verification**: `test -n "$HW_SECRET_ACCESS_KEY" && echo "✅ Credentials OK"`

## Cross-Skill Delegation Matrix

| Scenario | Delegating Skill | Target Skill | Delegation Type |
|----------|-----------------|--------------|-----------------|
| Create ECS in VPC | huaweicloud-vpc-ops | huaweicloud-ecs-ops | Provisioning |
| Bind EIP to ECS | huaweicloud-vpc-ops | huaweicloud-ecs-ops | Resource verification |
| RDS in VPC subnet | huaweicloud-vpc-ops | huaweicloud-rds-ops | Provisioning |
| ELB in VPC | huaweicloud-vpc-ops | huaweicloud-elb-ops | Provisioning |
| VPC bandwidth cost | huaweicloud-vpc-ops | huaweicloud-billing-ops | FinOps query |
| VPC IAM permissions | huaweicloud-vpc-ops | huaweicloud-iam-ops | IAM configuration |

## Environment Variables Reference

| Variable | Required | Description |
|----------|----------|-------------|
| `HW_ACCESS_KEY_ID` | Yes | IAM access key |
| `HW_SECRET_ACCESS_KEY` | Yes | IAM secret key (NEVER log/echo) |
| `HW_REGION_ID` | Yes | Target region (e.g., cn-north-4) |
| `HW_PROJECT_ID` | Yes | Project ID for API routing |
| `GOPROXY` | JIT only | Go module proxy |
