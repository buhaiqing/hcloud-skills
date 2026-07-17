# DNS Integration — Cross-Skill Delegation Matrix

## Cross-Skill Delegation

| Scenario | Delegate to | Why |
|---|---|---|
| CDN CNAME setup | `huaweicloud-cdn-ops` | CDN domain management |
| EIP binding / PTR | `huaweicloud-eip-ops` | EIP reverse DNS |
| DNS failover with health check | `huaweicloud-elb-ops` | Health-check based routing |
| Billing analysis | `huaweicloud-billing-ops` | Zone-level DNS billing |

## Resource Relationships

```
DNS Zone
  └── Public Zone (Internet) → DNSSEC
  └── Private Zone (VPC) → VPC-scoped IAM
        └── Record Sets
              ├── A / AAAA → EIP (huaweicloud-eip-ops)
              └── CNAME → CDN (huaweicloud-cdn-ops)
```

## Go JIT Bootstrap

```bash
if ! command -v go &> /dev/null; then
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m); [ "$ARCH" = "x86_64" ] && ARCH="amd64"
    mkdir -p /tmp/go-runtime
    curl -fsSL "https://go.dev/dl/go1.25.0.${OS}-${ARCH}.tar.gz" | tar -xz -C /tmp/go-runtime
    export PATH="/tmp/go-runtime/go/bin:$PATH"
    export GOPROXY="https://goproxy.cn,direct"
fi
```
