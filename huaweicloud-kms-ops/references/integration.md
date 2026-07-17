# KMS Integration — Cross-Skill Delegation Matrix

## Delegation Matrix

| Operation | Next Skill | Trigger |
|---|---|---|
| OBS object encryption | `huaweicloud-obs-ops` | After `create-key`, user wants to encrypt OBS bucket |
| RDS TDE enablement | `huaweicloud-rds-ops` | After `create-key`, user wants to enable RDS TDE |
| EVS disk encryption | EVS skill (when present) | After `create-key`, user wants to encrypt EVS volume |
| IAM permission issue | `huaweicloud-iam-ops` | `CMKAccessDenied` error |
| Key cost tracking | `huaweicloud-billing-ops` | Monthly cost analysis, quota budget alerts |
| Cross-region key copy | `huaweicloud-cts-ops` | Audit log for key operations |

## Go Bootstrap (JIT SDK)

```go
import (
    "fmt"
    "os"
    kms "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/kms/v2"
    kms_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/kms/v2/model"
)

func main() {
    ak, sk, region := os.Getenv("HW_ACCESS_KEY_ID"), os.Getenv("HW_SECRET_ACCESS_KEY"), os.Getenv("HW_REGION_ID")
    if ak == "" || sk == "" || region == "" {
        fmt.Fprintln(os.Stderr, "missing required env: HW_ACCESS_KEY_ID / HW_SECRET_ACCESS_KEY / HW_REGION_ID")
        os.Exit(2)
    }
    cfg := config.DefaultHttpConfig()
    client := kms.KmsClientBuilder().
        WithEndpoint(fmt.Sprintf("kms.%s.myhuaweicloud.com", region)).
        WithCredential(basic.NewCredentialsBuilder().WithAk(ak).WithSk(sk).Build()).
        WithHttpConfig(cfg).Build()
}
```

## Environment Variables

| Variable | Required | Default |
|---|---|---|
| `HW_ACCESS_KEY_ID` | Yes | — |
| `HW_SECRET_ACCESS_KEY` | Yes | — |
| `HW_REGION_ID` | No | `cn-north-4` |
| `HW_PROJECT_ID` | No | — |
