# DNS Core Concepts — Huawei Cloud DNS

## 1. DNS Zone Model

```
Registrar (注册商) → NS records → Huawei Cloud DNS ( authoritative nameserver )
                                    └── Zone (example.com.)
                                          └── Record Sets (A, CNAME, MX, TXT...)
```

| Component | Description |
|---|---|
| **Zone** (zone_name) | The DNS namespace managed by Huawei Cloud DNS (e.g., `example.com.`) |
| **Record Set** (recordset) | A collection of DNS records with the same name + type |
| **TTL** | Time-to-live; how long resolvers cache the record |
| **Zone type** | `public` (Internet-resolvable) or `private` (VPC-scoped) |
| **NS record** | Delegation from registrar to Huawei Cloud DNS nameservers |

## 2. Record Types

| Type | Use Case | Value format | Notes |
|---|---|---|---|
| A | IPv4 address | `1.2.3.4` | One IP per record |
| AAAA | IPv6 address | `2001:db8::1` | One IPv6 per record |
| CNAME | Canonical name | `cdn.example.com.` | Alias; must be FQDN with trailing dot |
| MX | Mail exchange | `10 mail.example.com.` | Priority + hostname |
| TXT | Text record | `"v=spf1 include:_spf.example.com ~all"` | SPF, DKIM, verification |
| NS | Nameserver delegation | `ns1.hwclouds-dns.com.` | Only at zone apex |
| PTR | Reverse DNS | `example.com.` | For EIP reverse lookup |
| SRV | Service locator | `_http._tcp.example.com.` | Priority + weight + port + target |

## 3. Zone Status Lifecycle

```
[NOT_EXIST] --create-zone--> [PENDING_CREATE] --(NS delegated)--> [ACTIVE]
                                                               --delete-zone--> [PENDING_DELETE]
```

## 4. TTL Strategy

| Scenario | Recommended TTL | Rationale |
|---|---|---|
| Default | 300s (5 min) | Balance between cache and update speed |
| CDN CNAME | 60s | Allow fast failover |
| Stable A record | 3600s (1h) | Rarely changes |
| Migration | Start: 60s; migrate; increase to 3600s | Fast rollback |
| Emergency | 0s | No cache; immediate propagation |

## 5. Key Resource Identifiers

| Field | Source | Used as |
|---|---|---|
| `{{output.zone_id}}` | `zone.id` | All zone operations |
| `{{output.recordset_id}}` | `recordset.id` | Record CRUD |
| `{{user.zone_name}}` | User input | Must end with `.` (e.g., `example.com.`) |
| `{{user.record_type}}` | User input | A / AAAA / CNAME / MX / TXT |
