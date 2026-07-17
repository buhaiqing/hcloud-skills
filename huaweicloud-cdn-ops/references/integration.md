# CDN Integration — Cross-Skill Delegation Matrix

## Go JIT Bootstrap

```bash
# Install Go runtime if missing
if ! command -v go &> /dev/null; then
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m); [ "$ARCH" = "x86_64" ] && ARCH="amd64"
    mkdir -p /tmp/go-runtime
    curl -fsSL "https://go.dev/dl/go1.25.0.${OS}-${ARCH}.tar.gz" | tar -xz -C /tmp/go-runtime
    export PATH="/tmp/go-runtime/go/bin:$PATH"
    export GOPROXY="https://goproxy.cn,direct"
fi
```

## Cross-Skill Delegation

| Scenario | Delegate to | Why |
|---|---|---|
| Origin is ECS → manage EIP / bandwidth | `huaweicloud-eip-ops` | Origin IP is EIP |
| HTTPS certificate provisioning | `huaweicloud-waf-ops` | Certificate management |
| Static assets hosted on OBS | `huaweicloud-obs-ops` | OBS bucket as origin |
| CDN billing analysis | `huaweicloud-billing-ops` | Traffic-based billing |
| Bandwidth threshold alarms | `huaweicloud-ces-ops` | CES metric + alarm wiring |
| Origin health monitoring | `huaweicloud-ecs-ops` | If origin is ECS |
| DDoS protection on CDN IP | `huaweicloud-eip-ops` | CDN IP is an EIP |

## Environment Variables

| Variable | Required | Default |
|---|---|---|
| `HW_ACCESS_KEY_ID` | Yes | — |
| `HW_SECRET_ACCESS_KEY` | Yes | — |
| `HW_REGION_ID` | No | `cn-north-4` |
| `HW_PROJECT_ID` | No | — |

## Resource Relationships

```
CDN Domain
  └── Origin Server
        ├── ECS EIP → huaweicloud-eip-ops
        ├── OBS Bucket → huaweicloud-obs-ops
        └── IP Address → (no skill needed)
  └── CDN Billing → huaweicloud-billing-ops
  └── CDN Metrics → huaweicloud-ces-ops
  └── CDN WAF / HTTPS → huaweicloud-waf-ops
```
