# Integration — Huawei Cloud ECS

## Environment Setup

**Primary path**: `hcloud ecs` CLI
**Fallback path**: JIT Go SDK via `huaweicloud-sdk-go-v3/services/ecs/v2`

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
    export GOPATH="/tmp/go-workspace"
    export GOPROXY="https://goproxy.cn,direct"
fi
```

### JIT Go SDK Workflow

1. Initialize: `mkdir -p /tmp/ecs-sdk-workspace && cd /tmp/ecs-sdk-workspace && go mod init ecs-script`
2. Get dependencies: `go get github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2`
3. Generate operation-specific `.go` file
4. Execute: `go run ./main.go`

## Cross-Skill Delegation Matrix

| Alarm/Event Type | Metric | Primary Skill | Secondary Skill | HSS/AOM Deleg. |
|-----------------|--------|--------------|----------------|---------------|
| ECS CPU High | `cpu_util` > 90% | huaweicloud-ecs-ops | huaweicloud-ces-ops | Optional |
| ECS Memory Leak | `mem_usedPercent` trend ↑ | huaweicloud-ecs-ops | huaweicloud-aom-ops | Recommended |
| Disk Full | `diskUsage_percent` > 90% | huaweicloud-ecs-ops | huaweicloud-ces-ops | — |
| SSH Brute Force | HSS event | huaweicloud-hss-ops | huaweicloud-ecs-ops | Required |
| Network DDoS | ELB dropped packets | huaweicloud-elb-ops | huaweicloud-waf-ops | Recommended |
| ECS Instance Down | `status == ERROR` | huaweicloud-ecs-ops | huaweicloud-ces-ops | Recommended |
| Database Performance | RDS CPU/Memory | huaweicloud-rds-ops | huaweicloud-ecs-ops (if on self-managed) | Optional |
| CCE Node Failure | Node CPU/Memory | huaweicloud-cce-ops | huaweicloud-ecs-ops (node investigation) | Recommended |

## Delegation Protocol

```
[ECS Alarm Triggered]
    │
    ├── 1. Identify metric (sys.ecs namespace)
    ├── 2. Check matrix: ECS is primary diagnostic skill
    ├── 3. Call ECS skill to check instance status
    ├── 4. If CPU high + running Java → delegate to AOM for thread dump
    ├── 5. If security event → delegate to HSS
    ├── 6. Summarize all outputs into unified report
    └── 7. Provide actionable fix to user
```

## IAM Minimum Permissions for ECS Operations

```json
{
  "Version": "1.1",
  "Statement": [
    { "Effect": "Allow", "Action": ["ecs:*List*", "ecs:*Describe*", "ecs:*Show*", "ecs:*Get*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["ecs:*Create*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["ecs:*Start*", "ecs:*Stop*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["ecs:*Delete*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["ecs:*Resize*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["ecs:*CloudCell*"], "Resource": ["*"] }
  ]
}
```
