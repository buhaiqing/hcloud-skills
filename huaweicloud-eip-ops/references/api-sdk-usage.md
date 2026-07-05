# EIP API & SDK Usage — Huawei Cloud Elastic IP

## Go SDK Import

```go
import (
    eip "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2"
    eip_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2/model"
)
```

## Endpoints

| Region | Endpoint |
|---|---|
| cn-north-4 | `eip.cn-north-4.myhuaweicloud.com` |
| cn-east-3 | `eip.cn-east-3.myhuaweicloud.com` |
| ap-southeast-1 | `eip.ap-southeast-1.myhuaweicloud.com` |
| (others) | `eip.{region}.myhuaweicloud.com` |

> **TE-1:** Do not hardcode the endpoint; construct from `{{env.HW_REGION_ID}}`.

## JSON Paths (Top-of-File Declaration)

```text
# EIP allocate / describe
.output.eip_id       = publicip.id
.output.public_ip    = publicip.public_ip_address
.output.status       = publicip.status
.output.port_id      = publicip.port_id
.output.type         = publicip.type

# Bandwidth (per-EIP and shared)
.output.bandwidth_id = bandwidth.id
.output.bandwidth_size = bandwidth.size
.output.charge_mode  = bandwidth.charge_mode
.output.share_type   = bandwidth.share_type
```

> Per **TE-4**, all JSON paths are declared once at the top of the file and re-used
> without duplication throughout the file.

## Common Operations (Go SDK)

### Allocate EIP

```go
//go:build ignore
req := &eip_model.CreatePublicipRequest{
    Body: &eip_model.CreatePublicipRequestBody{
        Publicip: &eip_model.CreatePublicipOption{Type: "5_bgp"},
        Bandwidth: &eip_model.CreatePublicipBandwidthOption{
            Name:       "eip-bw",
            Size:       5,
            ShareType:  "PER",
            ChargeMode: "bandwidth",
        },
    },
}
resp, err := client.CreatePublicip(req)
```

### Describe EIP

```go
//go:build ignore
req := &eip_model.ShowPublicipRequest{PublicipId: eipID}
resp, err := client.ShowPublicip(req)
// resp.Publicip.Id / PublicIpAddress / Status / PortId
```

### Bind / Unbind

```go
//go:build ignore
req := &eip_model.UpdatePublicipRequest{
    PublicipId: eipID,
    Body: &eip_model.UpdatePublicipRequestBody{
        PortId: &portID, // null = unbind
    },
}
resp, err := client.UpdatePublicip(req)
```

### Move EIP into Shared Bandwidth

```go
//go:build ignore
// Allocate / describe a WHOLE bandwidth, then update the EIP with its bandwidth.id
bandwidthID := "bw-xxxxxxxx"
req := &eip_model.UpdatePublicipRequest{
    PublicipId: eipID,
    Body: &eip_model.UpdatePublicipRequestBody{
        BandwidthId: &bandwidthID, // move into pool
    },
}
_, err := client.UpdatePublicip(req)
```

### Release EIP

```go
//go:build ignore
req := &eip_model.DeletePublicipRequest{PublicipId: eipID}
_, err := client.DeletePublicip(req)
```

## Error Mapping (Top 10 — one per row, ≤3 cols)

| HTTP / `error_code` | Cause | Agent Action |
|---|---|---|
| 400 `InvalidParameter` | Bad type / size / share_type | Fix from OpenAPI; retry 0–1 |
| 403 `Eip.0001` | IAM missing | HALT; ask user to add `vpc:eip:*` |
| 404 `ResourceNotFound` | EIP / bandwidth already gone | Treat as success on release path |
| 409 `EipAllocateFailed` | Region sold out / IP pool exhausted | HALT; suggest adjacent region |
| 409 `EipHasBandwidth` | EIP still has bandwidth attached | Detach bandwidth first |
| 409 `EipInUse` | Trying to release bound EIP | Unbind first |
| 409 `QuotaExceeded` | Region EIP quota hit | HALT; quota raise |
| 429 `Throttling` | Rate-limited | Backoff w/ Retry-After |
| 500 `InternalError` | Server-side | Retry 3× 2/4/8s |
| 503 `ServiceUnavailable` | Region degraded | Backoff + report RequestId |

> **TE-3:** Error table is 4 columns max, one error per row.

## Idempotency Notes

| Op | Idempotent? | Safe-retry token |
|---|---|---|
| `CreatePublicip` | No (may bill 2 EIPs) | n/a — must `list` + dedupe by `public_ip_address` |
| `UpdatePublicip` (bandwidth / port_id) | Yes | n/a |
| `DeletePublicip` | Yes (404 = success) | n/a |
| `CreateBandwidth` (WHOLE) | No (may bill 2 pools) | n/a — must `list` + dedupe by `name` |
