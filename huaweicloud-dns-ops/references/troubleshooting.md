# DNS Troubleshooting Guide — Huawei Cloud DNS

## Top DNS Failure Patterns

### T1: DNS Record Not Resolving

| Step | Check | Fix |
|---|---|---|
| 1 | Verify zone is `ACTIVE` | `hcloud dns show-zone --zone-id {{user.zone_id}}` → status must be `ACTIVE` |
| 2 | Verify NS records at registrar | `dig NS example.com.` — NS must point to Huawei Cloud DNS nameservers |
| 3 | Verify record set exists | `hcloud dns list-recordsets --zone-id {{user.zone_id}}` |
| 4 | Check TTL propagation | TTL expired? DNS changes take up to TTL duration to propagate |
| 5 | Check for duplicate record sets | Same name + type = conflict |

### T2: DNSSEC Validation Failure

| Step | Check | Fix |
|---|---|---|
| 1 | Check zone DNSSEC status | `hcloud dns show-zone --zone-id {{user.zone_id}}` → `dnssec_status` |
| 2 | Verify DS record at registrar | Registrar must host DS record for DNSSEC to work |
| 3 | Disable and re-enable DNSSEC | Force re-sign of all records |

### T3: CNAME + MX Conflict

| Problem | Cause | Fix |
|---|---|---|
| Cannot add MX to a CNAME record | RFC 1034 forbids MX pointing to CNAME | Add MX directly to the canonical name |
| Multiple CNAME targets | One name → one target | Remove duplicate CNAME; use A record |

## Error Code Quick Reference

| Code | Meaning | Immediate Action |
|---|---|---|
| `ZoneNotFound` | Zone ID invalid | Verify with `list-zones` |
| `RecordNotFound` | Recordset ID invalid | Verify with `list-recordsets` |
| `ZoneLocked` | DNSSEC transition in progress | Wait; poll status |
| `ZoneNotEmpty` | Zone has remaining records | Delete all records before deleting zone |
| `RecordExists` | Duplicate (same name+type) | `update-recordset` instead of `create` |
| `InvalidTTL` | TTL out of range | Use value 1–2147483647 |
| `Unauthorized` | IAM missing | Add `DNS FullAccess` policy |
