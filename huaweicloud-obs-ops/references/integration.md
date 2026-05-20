# Integration & Delegation — Huawei Cloud OBS

## obsutil Install & Config

### Install obsutil

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
[ "$ARCH" = "x86_64" ] && ARCH="64" || ARCH="arm64"
curl -fsSL "https://obs-community.obs.cn-north-4.myhuaweicloud.com/obsutil/current/obsutil_${OS}_${ARCH}.tar.gz" | tar -xz
chmod +x obsutil
./obsutil version
```

### Configure Credentials

```bash
./obsutil config -i={{env.HW_ACCESS_KEY_ID}} -k={{env.HW_SECRET_ACCESS_KEY}} -e={{env.HW_ENDPOINT}}
```

## Go SDK Initialization

```go
package main

import (
    "fmt"
    "os"
    "github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)

func main() {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")  // NEVER log/print
    endpoint := os.Getenv("HW_ENDPOINT")

    if ak == "" || sk == "" || endpoint == "" {
        panic("Required env vars HW_ACCESS_KEY_ID, HW_SECRET_ACCESS_KEY, HW_ENDPOINT not set")
    }

    client, err := obs.New(ak, sk, endpoint)
    if err != nil {
        panic(err)
    }
    fmt.Println("OBS client initialized successfully")
}
```

## Cross-Skill Delegation Matrix

| Trigger Condition | Delegate To | Required Input | Expected Output |
|------------------|-------------|----------------|----------------|
| Configure CDN with OBS origin | CDN skill | bucket_name, CDN domain | `{cdn_domain, CNAME_record}` |
| Set IAM policies for OBS access | `huaweicloud-iam-ops` | User, policy JSON | `{user_id, policy_id}` |
| Set up OBS monitoring alarms | `huaweicloud-ces-ops` | bucket, metric, thresholds | `{alarm_rule_id}` |
| Create VPC Endpoint for OBS | `huaweicloud-vpc-ops` | VPC, subnet, service | `{endpoint_id}` |
| Configure OBS access logging→LTS | `huaweicloud-lts-ops` | Log group/stream, bucket | `{log_group_id}` |
| Set up OBS event notifications | `huaweicloud-smn-ops` (when present) | Topic, event type | `{topic_arn}` |

## Dependency Order

For OBS + CDN deployment:

```
1. OBS Bucket        → huaweicloud-obs-ops: create-bucket
2. Upload Objects    → huaweicloud-obs-ops: upload objects
3. Set ACL           → huaweicloud-obs-ops: configure ACL (private or public-read)
4. CDN Origin Config → CDN skill: set OBS as origin
5. CDN Domain        → CDN skill: configure CDN domain + SSL
6. DNS CNAME         → DNS: point domain to CDN CNAME
```

## Chaining Output Fields

| Field | Source | Type | Example |
|-------|--------|------|---------|
| `bucket_name` | Create/List | string | `my-app-data` |
| `endpoint` | Config | string | `obs.cn-north-4.myhuaweicloud.com` |
| `object_key` | Upload/List | string | `uploads/image-001.jpg` |
| `etag` | Upload/Copy | string | `"abc123def456"` (includes quotes) |
| `versionId` | Upload (if versioning) | string | `CAESBxgxNj...` |
| `storage_class` | Upload/Config | string | `Standard`, `WARM`, `COLD` |
| `location` | GetBucketLocation | string | `cn-north-4` |
| `website_url` | SetBucketWebsite | string | `http://bucket.website.obs.region.myhuaweicloud.com` |
