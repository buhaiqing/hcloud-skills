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

### Core Delegation (P0)

| Trigger Condition | Delegate To | Required Input | Expected Output |
|------------------|-------------|----------------|----------------|
| Configure CDN with OBS origin | CDN skill | bucket_name, CDN domain | `{cdn_domain, CNAME_record}` |
| Set IAM policies for OBS access | `huaweicloud-iam-ops` | User, policy JSON | `{user_id, policy_id}` |
| Set up OBS monitoring alarms | `huaweicloud-ces-ops` | bucket, metric, thresholds | `{alarm_rule_id}` |
| Create VPC Endpoint for OBS | `huaweicloud-vpc-ops` | VPC, subnet, service | `{endpoint_id}` |
| Configure OBS access logging→LTS | `huaweicloud-lts-ops` | Log group/stream, bucket | `{log_group_id}` |
| Set up OBS event notifications | `huaweicloud-smn-ops` (when present) | Topic, event type | `{topic_arn}` |

### Extended Delegation (P1)

| Trigger Condition | Delegate To | Required Input | Expected Output |
|------------------|-------------|----------------|----------------|
| Storage quota alarm triggers | `huaweicloud-ces-ops` | bucket, threshold, alarm_action | `{alarm_id}` |
| Cross-region replication lag detected | `huaweicloud-ces-ops` | bucket, lag_threshold | `{alarm_id}` |
| Request throttling detected | `huaweicloud-ces-ops` | bucket, throttle_rate | `{alarm_id}` |
| IAM permission issue affecting OBS | `huaweicloud-iam-ops` | user/ak, permission_check | `{permission_status}` |
| Network route issue affecting OBS | `huaweicloud-vpc-ops` | vpc, endpoint_status | `{endpoint_id, route_status}` |
| Cost anomaly for OBS storage | `huaweicloud-billing-ops` | bucket, cost_threshold | `{cost_report}` |
| OBS bucket delete protection | `huaweicloud-ces-ops` | bucket, protection_enabled | `{protection_status}` |

### AIOps Delegation

| Trigger Condition | Delegate To | Required Input | Expected Output |
|------------------|-------------|----------------|----------------|
| Alarm storm detected for OBS | `huaweicloud-ces-ops` | aggregation_rule | `{suppressed_count, summary}` |
| Multi-metric correlation anomaly | `huaweicloud-ces-ops` | metric_pairs, correlation_id | `{correlation_result}` |
| Latency spike + bandwidth high | `huaweicloud-vpc-ops` | vpc, bandwidth_stats | `{bottleneck_analysis}` |
| Storage growth acceleration | `huaweicloud-ces-ops` | bucket, growth_rate | `{trend_analysis}` |

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
