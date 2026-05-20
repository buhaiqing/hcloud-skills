# Integration — Huawei Cloud DMS

## JIT Go SDK Setup

```bash
# Create temporary Go module for JIT execution
mkdir -p /tmp/dms-jit && cd /tmp/dms-jit
go mod init dms-jit

# Add DMS SDK dependency
go get github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dms/v2
go get github.com/huaweicloud/huaweicloud-sdk-go-v3/core
```

## Cross-Skill Delegation Matrix

| Scenario | Lead Skill | Delegates To | When |
|----------|-----------|-------------|------|
| Producer network error | `huaweicloud-dms-ops` | `huaweicloud-vpc-ops` | When cause is VPC/subnet/SG issue |
| Instance authentication failure | `huaweicloud-dms-ops` | `huaweicloud-iam-ops` | When AK/SK or credential issue suspected |
| Instance metric anomaly | `huaweicloud-dms-ops` | `huaweicloud-ces-ops` | When alarm or metric analysis needed |
| Audit event investigation | `huaweicloud-dms-ops` | `huaweicloud-cts-ops` | When tracking resource operation history |
| Image deployment uses SWR | `huaweicloud-cce-ops` | `huaweicloud-swr-ops` | When image pull issues in K8s |

## Environment Variables

```bash
# Required
export HW_ACCESS_KEY_ID="your-access-key"
export HW_SECRET_ACCESS_KEY="your-secret-key"
export HW_REGION_ID="cn-north-4"
export HW_PROJECT_ID="your-project-id"

# Optional
export DMS_CLI_TIMEOUT="300"        # CLI timeout in seconds
export DMS_POLL_INTERVAL="30"       # Status polling interval
export DMS_MAX_POLL_TIME="600"      # Max wait for async operations
```

## Go Module Bootstrap

```bash
#!/bin/bash
# JIT execution script for DMS operations
PROJECT_ID="{{env.HW_PROJECT_ID}}"
ACCESS_KEY="{{env.HW_ACCESS_KEY_ID}}"
SECRET_KEY="{{env.HW_SECRET_ACCESS_KEY}}"
REGION="{{env.HW_REGION_ID}}"

go run -exec '' -mod=mod << 'EOF'
package main

import (
    "fmt"
    "os"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dms/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dms/v2/model"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dms/v2/region"
)

func main() {
    auth := basic.NewCredentialsBuilder().
        WithAk(os.Getenv("HW_ACCESS_KEY_ID")).
        WithSk(os.Getenv("HW_SECRET_ACCESS_KEY")).
        WithProjectId(os.Getenv("HW_PROJECT_ID")).
        Build()
    client := dms.NewDmsClient(
        dms.DmsClientBuilder().
            WithRegion(region.ValueOf(os.Getenv("HW_REGION_ID"))).
            WithCredential(auth).
            Build(),
    )
    req := &model.ListInstancesReq{}
    resp, err := client.ListInstances(req)
    if err != nil {
        fmt.Println("Error:", err)
        os.Exit(1)
    }
    for _, inst := range resp.Instances {
        fmt.Printf("- %s (%s): %s [%s]\n", *inst.Name, *inst.InstanceId, *inst.Status, *inst.Engine)
    }
}
EOF
```
