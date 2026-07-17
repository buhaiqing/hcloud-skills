# CDN API & SDK Usage — Huawei Cloud CDN

## Go SDK Import

```go
import (
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    cdn "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cdn/v1"
    cdn_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cdn/v1/model"
)
```

## Endpoints

| Region | Endpoint |
|---|---|
| cn-north-4 | `cdn.cn-north-4.myhuaweicloud.com` |
| cn-east-3 | `cdn.cn-east-3.myhuaweicloud.com` |
| ap-southeast-1 | `cdn.ap-southeast-1.myhuaweicloud.com` |

> Per TE-1, construct endpoint from `{{env.HW_REGION_ID}}`.

## JSON Paths (Top-of-File Declaration)

```text
.output.domain_id   = result[].id
.output.domain_name  = result[].domain_name
.output.cname       = result[].cname
.output.status      = result[].status
.output.job_id      = job_id
.output.hit_rate    = result[].hit_rate
.output.bandwidth   = result[].bps
```

## Common Operations (Go SDK)

### List Domains

```go
//go:build ignore
req := &cdn_model.ListDomainsRequest{}
resp, err := client.ListDomains(req)
for _, d := range resp.Domains {
    fmt.Printf("id=%s name=%s status=%s\n", *d.Id, *d.DomainName, *d.Status)
}
```

### Create Domain

```go
//go:build ignore
req := &cdn_model.CreateDomainRequest{
    Body: &cdn_model.CreateDomainRequestBody{
        Domain: &cdn_model.CreateDomainDetail{
            DomainName:   "example.com",
            BusinessType: "web",
            ServiceArea:  "mainland_china",
            Sources: []cdn_model.SourceDomainConfig{
                {OriginAddr: "1.2.3.4", OriginType: "ipaddr"},
            },
        },
    },
}
resp, err := client.CreateDomain(req)
// resp.Domain.Id — poll for status = "online"
```

### Delete Domain

```go
//go:build ignore
req := &cdn_model.DeleteDomainRequest{DomainId: domainID}
_, err := client.DeleteDomain(req)
```

### Refresh Cache

```go
//go:build ignore
req := &cdn_model.RefreshCacheRequest{
    Body: &cdn_model.RefreshCacheRequestBody{
        RefreshTask: []cdn_model.RefreshCacheTask{
            {Type: "file", Url: "https://example.com/index.html"},
        },
    },
}
resp, err := client.RefreshCache(req)
// resp.JobId — poll with ShowRefreshTaskRequest
```

### Preheat Cache

```go
//go:build ignore
req := &cdn_model.PreheatingRequest{
    Body: &cdn_model.PreheatingRequestBody{
        PreheatingTask: []cdn_model.PreheatingTask{
            {Url: "https://example.com/bundle.js"},
        },
    },
}
resp, err := client.Preheating(req)
// resp.JobId — poll with ShowPreheatingTaskRequest
```

### Query Statistics

```go
//go:build ignore
req := &cdn_model.ShowBandwidthIntervalRequest{
    DomainId: domainID,
    StartTime: 1730000000, // unix seconds
    EndTime:   1730086400,
}
resp, err := client.ShowBandwidthInterval(req)
// resp.Data.BandwidthS — bps
```

## Error Mapping (Top 10 — one per row, ≤3 cols)

| HTTP / `error_code` | Cause | Agent Action |
|---|---|---|
| 400 `InvalidParameter` | Bad domain name / invalid URL | Fix from OpenAPI; retry 0–1 |
| 403 `DomainNotFound` | Domain does not exist | Verify domain_id; HALT |
| 409 `DomainExists` | Domain name already in CDN | List existing; reuse or rename |
| 409 `DomainConfiguring` | Domain in transitional state | Poll until stable |
| 429 `Throttling` | Rate-limited | Backoff w/ Retry-After |
| 500 `InternalError` | Server-side | Retry 3× 2/4/8s |
| 503 `ServiceUnavailable` | Region degraded | Backoff + report RequestId |
| 400 `RefreshQuotaExceeded` | Too many URLs per refresh | Split into batches |
| 409 `DomainOffline` | Operation requires online domain | Start domain first |
| 403 `Unauthorized` | IAM missing | HALT; add `CDN FullAccess` policy |

## Idempotency Notes

| Op | Idempotent? | Safe-retry |
|---|---|---|
| `CreateDomain` | No (domain name unique) | List first; skip if exists |
| `DeleteDomain` | Yes (404 = success) | n/a |
| `RefreshCache` | Yes (idempotent purge) | Safe to retry |
| `PreheatCache` | Yes (best-effort prefill) | Safe to retry |
| `StartDomain` / `StopDomain` | Yes (no-op if already target state) | n/a |
