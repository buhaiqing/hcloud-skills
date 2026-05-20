# Integration — Huawei Cloud ELB

## Environment Setup

**Primary path**: `hcloud elb` CLI
**Fallback path**: JIT Go SDK via `huaweicloud-sdk-go-v3/services/elb/v3`

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

1. Initialize: `mkdir -p /tmp/elb-sdk-workspace && cd /tmp/elb-sdk-workspace && go mod init elb-script`
2. Get dependencies: `go get github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v3`
3. Generate operation-specific `.go` file
4. Execute: `go run ./main.go`

## Cross-Skill Delegation Matrix

| Alarm/Event Type | Metric | Primary Skill | Secondary Skill | Description |
|-----------------|--------|--------------|----------------|-------------|
| Backend unhealthy | `m9_unhealthy_host` > 0 | huaweicloud-elb-ops | huaweicloud-ecs-ops | Check ECS status, restart service |
| High error rate | `m7_req_5xx` > 5% | huaweicloud-elb-ops | huaweicloud-ecs-ops | Check backend app health |
| Traffic spike | `m1_cps` > 3× baseline | huaweicloud-elb-ops | huaweicloud-ces-ops | Analyze traffic pattern, scale out |
| Certificate expiring | — | huaweicloud-elb-ops | — | Renew/upload SSL cert |
| DDoS suspicion | `m5_drop_rate` > 0 + `m1_cps` spike | huaweicloud-elb-ops | huaweicloud-waf-ops | Enable WAF, rate limiting |
| Backend ECS down | ECS `cpu_util` = 0 | huaweicloud-ecs-ops | huaweicloud-elb-ops | Remove member from pool, recover ECS |
| VPC/subnet issue | LB creation fails | huaweicloud-elb-ops | huaweicloud-vpc-ops | Verify VPC/subnet config |
| Cost spike | Bill increases > 20% | huaweicloud-billing-ops | huaweicloud-elb-ops | Check LB usage, optimize |

## Delegation Protocol

```
[ELB Alarm Triggered]
    │
    ├── 1. Identify LB ID from alarm context
    ├── 2. Check LB provisioning_status & operating_status
    ├── 3. Get pool members and their health status
    ├── 4. If backend unhealthy:
    │       ├── Check health monitor config
    │       ├── Delegate to ECS skill: check member server status
    │       ├── Check member security group allows LB subnet traffic
    │       └── If all fine, suspect health check config
    ├── 5. If connection errors:
    │       ├── Check CES metrics (m1_cps, m2_act_conn, m5_drop_rate)
    │       ├── Check listener config (port, protocol, certificate)
    │       └── If DDoS suspicion, delegate to WAF
    ├── 6. If performance issues:
    │       ├── Check latency metrics (m8/m10)
    │       ├── Check member capacity via ECS skill
    │       └── Suggest scaling or LB upgrade
    └── 7. Summarize findings into unified report
```

## IAM Minimum Permissions for ELB Operations

```json
{
  "Version": "1.1",
  "Statement": [
    { "Effect": "Allow", "Action": ["elb:*List*", "elb:*Show*", "elb:*Get*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["elb:*Create*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["elb:*Update*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["elb:*Delete*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["ecs:*List*", "ecs:*Show*", "ecs:*Get*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["vpc:*List*", "vpc:*Show*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["ces:metricData:list"], "Resource": ["*"] }
  ]
}
```
