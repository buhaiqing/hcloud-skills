# CCE Integration — Huawei Cloud Cloud Container Engine

## JIT Go SDK Setup

### Bootstrap Script

```bash
#!/bin/bash
# JIT Go SDK bootstrap for CCE operations
# Run this script before JIT-compiled CCE SDK operations

set -e

# Check if Go is available
if ! command -v go &> /dev/null; then
    echo "Bootstrapping Go runtime..."
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    [ "$ARCH" = "x86_64" ] && ARCH="amd64"
    [ "$ARCH" = "aarch64" ] && ARCH="arm64"

    GO_RUNTIME_DIR="/tmp/go-runtime-cce"
    mkdir -p "$GO_RUNTIME_DIR"
    curl -fsSL "https://go.dev/dl/go1.25.0.${OS}-${ARCH}.tar.gz" | tar -xz -C "$GO_RUNTIME_DIR"
    export PATH="${GO_RUNTIME_DIR}/go/bin:$PATH"
    export GOPROXY="https://goproxy.cn,direct"
fi

# Verify Go
go version

# Setup JIT project
JIT_DIR="/tmp/cce-jit-$(date +%s)"
mkdir -p "$JIT_DIR"
cd "$JIT_DIR"

# Initialize Go module
go mod init cce-jit

# Download Huawei Cloud SDK
go get github.com/huaweicloud/huaweicloud-sdk-go-v3@latest

# Write the JIT script (passed by agent)
# go run . (executed after agent writes main.go)
echo "JIT environment ready at: $JIT_DIR"
```

### Environment Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `HW_ACCESS_KEY_ID` | Yes | Huawei Cloud Access Key | `AKxxxxxxxxxxxxxxxxxxxx` |
| `HW_SECRET_ACCESS_KEY` | Yes | Huawei Cloud Secret Key | `SKxxxxxxxxxxxxxxxxxxxx` (NEVER expose) |
| `HW_REGION_ID` | Yes | Target region | `cn-north-4` |
| `HW_PROJECT_ID` | Yes | Project ID for API calls | `xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx` |

### Client Initialization (Go)

```go
package main

import (
    "fmt"
    "os"

    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cce/v3"
)

func main() {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    region := os.Getenv("HW_REGION_ID")

    if ak == "" || sk == "" || region == "" {
        fmt.Fprintln(os.Stderr, "HW_ACCESS_KEY_ID, HW_SECRET_ACCESS_KEY, and HW_REGION_ID are required")
        os.Exit(1)
    }

    cfg := config.DefaultHttpConfig()
    client := v3.NewCceClient(
        v3.CceClientBuilder().
            WithEndpoint(fmt.Sprintf("cce.%s.myhuaweicloud.com", region)).
            WithCredential(basic.NewCredentialsBuilder().
                WithAk(ak).WithSk(sk).Build()).
            WithHttpConfig(cfg).Build())

    // Use client for CCE operations
    _ = client
}
```

## Cross-Skill Delegation Matrix

| When User Wants | Primary Skill | Delegate To | Reason |
|----------------|---------------|-------------|--------|
| Create CCE cluster | `huaweicloud-cce-ops` | `huaweicloud-vpc-ops` | Need VPC, subnet, security group first |
| Create CCE cluster | `huaweicloud-cce-ops` | `huaweicloud-ecs-ops` | ECS flavor selection and pricing details |
| CCE persistent storage | `huaweicloud-cce-ops` | `huaweicloud-evs-ops` | EVS volume types, performance tiers |
| CCE LoadBalancer service | `huaweicloud-cce-ops` | `huaweicloud-elb-ops` | ELB configuration, listener setup |
| CCE monitoring/alarm | `huaweicloud-cce-ops` | `huaweicloud-ces-ops` | CES metric queries, alarm rules |
| CCE log collection | `huaweicloud-cce-ops` | `huaweicloud-lts-ops` | LTS log groups, log streams |
| CCE container images | `huaweicloud-cce-ops` | `huaweicloud-swr-ops` | Container image registry (when present) |
| CCE node security | `huaweicloud-cce-ops` | — | Node security groups are CCE-owned; handle internally |
| CCE billing/cost | `huaweicloud-cce-ops` | `huaweicloud-billing-ops` | Cost center, billing queries |
| CCE IAM permissions | `huaweicloud-cce-ops` | `huaweicloud-iam-ops` | IAM role/policy management |

## Dependency Installation Order

```
1. IAM (IAM User + CCE permissions)
2. VPC (VPC → Subnet → Security Group)
3. CCE (Cluster → Addons → Node Pools → Nodes)
4. CES (Monitoring alarms)
5. LTS (Log collection)
```

### Prerequisite Verification

Before creating a CCE cluster:

```bash
# Verify VPC
hcloud vpc describe-vpc --region "{{user.region}}" --vpc-id "{{user.vpc_id}}"

# Verify Subnet
hcloud vpc describe-subnet --region "{{user.region}}" --subnet-id "{{user.subnet_id}}"

# Verify Security Group
hcloud vpc describe-security-group --region "{{user.region}}" --group-id "{{user.security_group_id}}"

# Verify Quota
hcloud ecs describe-flavors --region "{{user.region}}" --name "{{user.flavor}}"
```
