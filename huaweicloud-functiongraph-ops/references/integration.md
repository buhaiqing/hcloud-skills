# Integration — Huawei Cloud FunctionGraph

## Environment Setup

**Primary path**: JIT Go SDK via `huaweicloud-sdk-go-v3/services/functiongraph/v2`
**CLI**: Not available — SDK-only.

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

1. Initialize: `mkdir -p /tmp/fg-sdk-workspace && cd /tmp/fg-sdk-workspace && go mod init fg-script`
2. Get dependencies: `go get github.com/huaweicloud/huaweicloud-sdk-go-v3/services/functiongraph/v2`
3. Generate operation-specific `.go` file
4. Execute: `go run ./main.go`

## Cross-Skill Delegation Matrix

| Alarm/Event Type | Primary Skill | Secondary Skill | Description |
|-----------------|--------------|----------------|-------------|
| Function error rate high | huaweicloud-functiongraph-ops | huaweicloud-ces-ops | Check CES metrics, correlate with code change |
| Function timeout | huaweicloud-functiongraph-ops | huaweicloud-lts-ops | Check LTS logs for slow code, optimize |
| APIG trigger down | huaweicloud-functiongraph-ops | huaweicloud-apig-ops | Check APIG configuration, function availability |
| OBS trigger not firing | huaweicloud-functiongraph-ops | huaweicloud-obs-ops | Verify OBS bucket, event notification config |
| SMN message loss | huaweicloud-functiongraph-ops | huaweicloud-smn-ops | Check SMN topic, subscription status |
| CTS audit trail | huaweicloud-functiongraph-ops | huaweicloud-cts-ops | Trace function creation/deletion events |
| Function cost spike | huaweicloud-functiongraph-ops | huaweicloud-billing-ops | Check invocation volume, optimize memory/timeout |
| DMS/Kafka consumer lag | huaweicloud-functiongraph-ops | huaweicloud-dms-ops | Check DMS queue, function processing rate |

## Delegation Protocol

```
[FunctionGraph Alarm Triggered]
    │
    ├── 1. Identify function URN from alarm context
    ├── 2. Check function state (Active/Failed)
    ├── 3. Get recent invocation metrics (CES)
    ├── 4. If error rate high:
    │       ├── Check LTS logs for error details
    │       ├── Check recent code/config changes (version diff)
    │       └── Test sync invoke to reproduce
    ├── 5. If trigger issue:
    │       ├── Check trigger status and config
    │       └── Delegate to trigger source skill (APIG/OBS/CTS)
    ├── 6. If performance issue:
    │       ├── Check duration metrics vs baseline
    │       ├── Check concurrent executions
    │       └── Suggest memory/timeout optimization
    └── 7. Summarize findings into unified report
```

## IAM Minimum Permissions for FunctionGraph Operations

```json
{
  "Version": "1.1",
  "Statement": [
    { "Effect": "Allow", "Action": ["functiongraph:*List*", "functiongraph:*Show*", "functiongraph:*Get*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["functiongraph:*Create*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["functiongraph:*Update*", "functiongraph:*Deploy*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["functiongraph:*Delete*", "functiongraph:*Remove*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["functiongraph:*Invoke*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["functiongraph:*Trigger*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["functiongraph:*Version*", "functiongraph:*Alias*"], "Resource": ["*"] },
    { "Effect": "Allow", "Action": ["obs:object:GetObject"], "Resource": ["obs:*:*:*:bucket-name/*"] },
    { "Effect": "Allow", "Action": ["ces:metricData:list"], "Resource": ["*"] }
  ]
}
```
