# Integration & Delegation — Huawei Cloud DCS

## JIT SDK Setup

### Go Runtime Bootstrap

```bash
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

go version
```

## Credential Configuration

| Env Variable | Purpose | Required |
|-------------|---------|----------|
| `HW_ACCESS_KEY_ID` | Huawei Cloud AK | Yes |
| `HW_SECRET_ACCESS_KEY` | Huawei Cloud SK | Yes |
| `HW_REGION_ID` | DCS instance region | Yes |
| `HW_PROJECT_ID` | Project ID for API calls | Yes |

```bash
# Verify (existence only, NEVER echo)
test -n "$HW_ACCESS_KEY_ID" && test -n "$HW_SECRET_ACCESS_KEY" \
  && echo "✅ DCS credentials configured" \
  || echo "❌ Missing credentials"
```

## SDK Client Initialization

```go
package main

import (
    "fmt"
    "os"
    dcs "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dcs/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
)

func main() {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")  // NEVER log or print this value
    region := os.Getenv("HW_REGION_ID")

    if ak == "" || sk == "" || region == "" {
        panic("Required env vars HW_ACCESS_KEY_ID, HW_SECRET_ACCESS_KEY, HW_REGION_ID not set")
    }

    cfg := config.DefaultHttpConfig().
        WithTimeout(30).
        WithKeepAlive(true).
        WithRetries(3)

    client := dcs.NewDcsClient(
        dcs.DcsClientBuilder().
            WithEndpoint(fmt.Sprintf("dcs.%s.myhuaweicloud.com", region)).
            WithCredential(basic.NewCredentialsBuilder().
                WithAk(ak).WithSk(sk).Build()).
            WithHttpConfig(cfg).
            Build())

    // Client is ready for DCS API calls
    fmt.Println("DCS client initialized successfully")
}
```

## Cross-Skill Delegation Matrix

| Trigger Condition | Delegate To | Required Input | Expected Output |
|------------------|-------------|----------------|----------------|
| Need to create VPC before DCS | `huaweicloud-vpc-ops` | CIDR, region | `{vpc_id, subnet_id}` |
| Need to verify/create security group | `huaweicloud-vpc-ops` | VPC ID, port rules | `{sg_id}` |
| Set up DCS monitoring alarms | `huaweicloud-ces-ops` | instance_id, metric name, thresholds | `{alarm_rule_id}` |
| Check if DCS is causing cost spikes | Billing skill | instance_id, date range | `{daily_cost, cost_breakdown}` |
| Set up IAM user/permissions for DCS | `huaweicloud-iam-ops` | User name, policy JSON | `{user_id, policy_id}` |
| Investigate ECS running Redis clients | `huaweicloud-ecs-ops` | ECS instance IDs | `{ecs_status, cpu, memory}` |
| Configure app-side query logging | `huaweicloud-lts-ops` | Log group, log stream | `{log_group_id}` |
| Check for security vulnerabilities | `huaweicloud-hss-ops` | ECS host IPs running Redis | `{vulnerability_report}` |

## Dependency Order

When creating DCS from scratch, resources must be created in this order:

```
1. VPC          → huaweicloud-vpc-ops: create-vpc
2. Subnet       → huaweicloud-vpc-ops: create-subnet (in VPC)
3. Security Group → huaweicloud-vpc-ops: create-security-group (allow port 6379)
4. DCS Instance → huaweicloud-dcs-ops: create-instance
5. CES Alarm    → huaweicloud-ces-ops: create-alarm-rule (after instance RUNNING)
```

## Chaining Output Fields

Stable output fields from DCS operations for downstream consumption:

| Field | Source | Type | Example |
|-------|--------|------|---------|
| `instance_id` | Create/Show/List | string | `dcs-0a1b2c3d` |
| `status` | Create/Show/List | string | `RUNNING` |
| `ip` | Show/List | string | `192.168.1.100` |
| `port` | Show/List | number | `6379` |
| `engine` | Show/List | string | `redis` |
| `engine_version` | Show/List | string | `6.0` |
| `capacity` | Show/List | number | `4` (GB) |
| `vpc_id` | Show/List | string | `vpc-abc123` |
| `subnet_id` | Show/List | string | `subnet-def456` |
| `domain_name` | Show/List | string | `dcs-0a1b2c3d.dcs.huaweicloud.com` |
| `max_memory_mb` | Show | number | `4096` |
| `spec_code` | Show/List | string | `dcs.master_standby` |

## Go SDK Type Reference

| Type | Package Path | Purpose |
|------|-------------|---------|
| `CreateInstanceRequest` | `dcs_model.CreateInstanceRequest` | Create instance request |
| `ShowInstanceRequest` | `dcs_model.ShowInstanceRequest` | Get instance details |
| `CreateInstanceRequestBody` | `dcs_model.CreateInstanceRequestBody` | Request body fields |
| `InstanceResponse` | `dcs_model.InstanceResponse` | Response instance details |
| `CreateBackupRequest` | `dcs_model.CreateBackupRequest` | Backup creation |
| `BackupCreateRequest` | `dcs_model.BackupCreateRequestBody` | Backup body |
| `ResizeInstanceRequest` | `dcs_model.ResizeInstanceRequest` | Resize spec |

SDK package: `github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dcs/v2`
