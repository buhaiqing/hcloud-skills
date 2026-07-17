# Integration — Huawei Cloud SWR

## JIT Go SDK Setup

```bash
mkdir -p /tmp/swr-jit && cd /tmp/swr-jit
go mod init swr-jit
go get github.com/huaweicloud/huaweicloud-sdk-go-v3/services/swr/v2
go get github.com/huaweicloud/huaweicloud-sdk-go-v3/core
```

## Cross-Skill Delegation Matrix

| Scenario | Lead Skill | Delegates To | When |
|----------|-----------|-------------|------|
| CCE pod cannot pull image | `huaweicloud-swr-ops` | `huaweicloud-cce-ops` | K8s image pull secret issue |
| Image vulnerability found | `huaweicloud-swr-ops` | `huaweicloud-hss-ops` | Host/container vulnerability scan |
| IAM permission denied | `huaweicloud-swr-ops` | `huaweicloud-iam-ops` | SWR access policy issue |
| Image pull network timeout | `huaweicloud-swr-ops` | `huaweicloud-vpc-ops` | VPC endpoint or NAT issue |
| Metric anomaly | `huaweicloud-swr-ops` | `huaweicloud-ces-ops` | Alarm or metric analysis |
| Audit trail | `huaweicloud-swr-ops` | `huaweicloud-cts-ops` | Image delete/push tracking |

## AIOps Cross-Skill Delegation Matrix

| Anomaly Pattern | Lead Skill | Delegates To | Trigger Condition |
|----------------|-----------|-------------|-------------------|
| Storage quota near (>80%) | `huaweicloud-swr-ops` | `huaweicloud-ces-ops` | SWR storage metric alert |
| Pull throttling | `huaweicloud-swr-ops` | `huaweicloud-vpc-ops` | Rate limit exceeded |
| Webhook failure spike | `huaweicloud-swr-ops` | `huaweicloud-vpc-ops` | Webhook delivery failure |
| Build failure spike | `huaweicloud-swr-ops` | `huaweicloud-codearts-ops` | Build trigger failed |
| Security vulnerability | `huaweicloud-swr-ops` | `huaweicloud-hss-ops` | Image scan critical finding |
| Multi-resource alarm storm | `huaweicloud-swr-ops` | `huaweicloud-ces-ops` | ≥3 resources affected |

## Environment Variables

```bash
export HW_ACCESS_KEY_ID="your-access-key"
export HW_SECRET_ACCESS_KEY="your-secret-key"
export HW_REGION_ID="cn-north-4"
export HW_PROJECT_ID="your-project-id"
```

## Docker Login Automation

```bash
#!/bin/bash
# Script: swr-login.sh — Login to SWR and test access
set -euo pipefail

REGION="${HW_REGION_ID:-cn-north-4}"
SWR_ENDPOINT="swr.${REGION}.myhuaweicloud.com"

echo "Logging into SWR at ${SWR_ENDPOINT}..."
echo "${HW_SECRET_ACCESS_KEY}" | \
  docker login -u "${HW_ACCESS_KEY_ID}" \
  --password-stdin "${SWR_ENDPOINT}"

echo "Login successful. Testing pull..."

docker pull "${SWR_ENDPOINT}/library/hello-world:latest" || {
  echo "Pull test failed — check network and permissions"
  exit 1
}

echo "SWR integration verified."
```

## Go Module Bootstrap

```go
package main

import (
    "fmt"
    "os"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/swr/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/swr/v2/model"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/swr/v2/region"
)

func main() {
    auth := basic.NewCredentialsBuilder().
        WithAk(os.Getenv("HW_ACCESS_KEY_ID")).
        WithSk(os.Getenv("HW_SECRET_ACCESS_KEY")).
        WithProjectId(os.Getenv("HW_PROJECT_ID")).
        Build()
    client := swr.NewSwrClient(
        swr.SwrClientBuilder().
            WithRegion(region.ValueOf(os.Getenv("HW_REGION_ID"))).
            WithCredential(auth).
            Build(),
    )
    req := &model.ListRepositoriesReq{}
    resp, err := client.ListRepositories(req)
    if err != nil {
        fmt.Println("Error:", err)
        os.Exit(1)
    }
    for _, repo := range resp {
        fmt.Printf("- %s/%s: %d images, %d bytes\n",
            *repo.Namespace, *repo.Name, *repo.NumImages, *repo.Size)
    }
}
```

## Anomaly Correlation Rules

| Pattern A | Pattern B | Correlation | Action |
|-----------|-----------|-------------|--------|
| storage_quota_near | pull_throttling | High | Emergency cleanup优先 |
| webhook_failure_high | build_failure_spike | High | 检查SCM webhook |
| image_count_growth | pull_count_drop | Medium | 清理无用镜像 |
| pull_latency_spike | storage_pull_correlation | High | 检查存储性能 |

## Fault Knowledge Base Reference

See `knowledge-base.md` for 6 common fault patterns with resolution procedures.
