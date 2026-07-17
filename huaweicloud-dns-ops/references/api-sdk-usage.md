# DNS API & SDK Usage — Huawei Cloud DNS

## Go SDK Import

```go
import (
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    dns "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2"
    dns_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dns/v2/model"
)
```

## Endpoints

| Region | Endpoint |
|---|---|
| Global API | `dns.myhuaweicloud.com` |

> DNS API is global; `{{env.HW_REGION_ID}}` is ignored for DNS but retained for convention.

## JSON Paths (Top-of-File Declaration)

```text
.output.zone_id       = zone.id
.output.zone_name    = zone.name
.output.recordset_id = recordset.id
.output.record_name  = recordset.name
.output.record_type  = recordset.type
.output.record_value = recordset.records
```

## Common Operations (Go SDK)

### List Zones

```go
//go:build ignore
req := &dns_model.ListPublicZonesRequest{}
resp, err := client.ListPublicZones(req)
for _, z := range resp.Zones {
    fmt.Printf("id=%s name=%s status=%s\n", *z.Id, *z.Name, *z.Status)
}
```

### Create Zone

```go
//go:build ignore
req := &dns_model.CreatePublicZoneRequest{
    Body: &dns_model.CreatePublicZoneRequestBody{
        Name:        "example.com.",
        ZoneType:    "public",
        Email:       "admin@example.com",
        Description: "Production zone",
    },
}
resp, err := client.CreatePublicZone(req)
fmt.Printf("zone_id=%s\n", *resp.Zone.Id)
```

### Create Record Set

```go
//go:build ignore
req := &dns_model.CreateRecordSetWithLineRequest{
    ZoneId: zoneID,
    Body: &dns_model.CreateRecordSet{
        Name:  "www.example.com.",
        Type:  "A",
        Records: []string{"1.2.3.4"},
        TTL:    300,
    },
}
resp, err := client.CreateRecordSetWithLine(req)
fmt.Printf("recordset_id=%s\n", *resp.RecordSet.Id)
```

### Delete Zone

```go
//go:build ignore
req := &dns_model.DeletePublicZoneRequest{ZoneId: zoneID}
_, err := client.DeletePublicZone(req)
```

## Error Mapping (Top 10 — one per row, ≤3 cols)

| HTTP / `error_code` | Cause | Agent Action |
|---|---|---|
| 400 `InvalidParameter` | Bad zone name / invalid record value | Fix from OpenAPI |
| 404 `ZoneNotFound` | Zone does not exist | Verify zone_id |
| 409 `ZoneExists` | Zone already created | List existing; reuse |
| 429 `Throttling` | Rate-limited | Backoff w/ Retry-After |
| 500 `InternalError` | Server-side | Retry 3× 2/4/8s |
| 400 `RecordExists` | Duplicate record set | List; update existing |
| 400 `InvalidTTL` | TTL out of range (1–2147483647) | Fix TTL value |
| 403 `ZoneLocked` | Zone locked (DNSSEC pending) | Wait; check DNSSEC status |
| 409 `ZoneNotEmpty` | Zone has remaining records | Delete all records first |
| 400 `InvalidRecordValue` | Wrong record type for value | Validate against record type |

## Idempotency Notes

| Op | Idempotent? | Pre-retry step |
|---|---|---|
| `CreateZone` | No (name unique) | `list-zones`; skip if exists |
| `CreateRecordSet` | No (type+name unique per zone) | `list-recordsets`; update if exists |
| `DeleteZone` | Yes (404 = success) | n/a |
| `DeleteRecordSet` | Yes | n/a |
| `UpdateRecordSet` | Yes (update replaces) | n/a |
