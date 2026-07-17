# DNS CLI Usage — Huawei Cloud DNS

## CLI Command Map

| Operation | CLI Command | Notes |
|---|---|---|
| List zones | `hcloud dns list-zones` | Public and private zones |
| Create zone | `hcloud dns create-zone` | Public or private (with --vpc-id) |
| Show zone | `hcloud dns show-zone` | Single zone detail |
| Delete zone | `hcloud dns delete-zone` | **Irreversible** |
| List recordsets | `hcloud dns list-recordsets` | All records in a zone |
| Create recordset | `hcloud dns create-recordset` | A / CNAME / MX / TXT etc. |
| Update recordset | `hcloud dns update-recordset` | Modify value or TTL |
| Delete recordset | `hcloud dns delete-recordset` | Remove single record |
| Show quota | `hcloud dns show-quota` | Zone + record count limits |

> **Verify before use:** `hcloud dns --help`.

## Common Recipes

### Add A record

```bash
ZONE_ID=$(hcloud dns list-zones --region {{env.HW_REGION_ID}} \
  --output json | jq -r '.zones[] | select(.name=="example.com.") | .id')

hcloud dns create-recordset \
  --zone-id "$ZONE_ID" \
  --name "www.example.com." \
  --type "A" \
  --records "1.2.3.4" \
  --ttl 300
```

### Update TTL

```bash
RECORD_ID=$(hcloud dns list-recordsets --zone-id "$ZONE_ID" \
  --output json | jq -r '.recordsets[] | select(.name=="www.example.com.") | .id')

hcloud dns update-recordset \
  --zone-id "$ZONE_ID" \
  --recordset-id "$RECORD_ID" \
  --ttl 3600
```

### CNAME for CDN

```bash
hcloud dns create-recordset \
  --zone-id "$ZONE_ID" \
  --name "cdn.example.com." \
  --type "CNAME" \
  --records "example.com.cdn.cn-north-4.myhwcdn.com." \
  --ttl 60   # Short TTL for CDN fast failover
```

## When to Fall Back to SDK

| CLI missing? | Use SDK call |
|---|---|
| DNSSEC configuration | `SetDnssecStatus` via Go SDK |
| Private zone per VPC | `CreatePrivateZone` with `vpc_id` |
| Line-based routing | `CreateRecordSetWithLine` with `line_id` |
| Batch record operations | SDK batch create/update |
