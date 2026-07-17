# DNS AIOps Patterns

> Tier-2 advanced content per TE-7: AIOps depth lives here, not in SKILL.md.

## Pattern 1: NXDOMAIN Spike

| Field | Value |
|---|---|
| Metric | `nxdomain_count` per minute |
| Window | 5 min |
| Threshold | >10× normal baseline |
| Action | (a) Check if zone is misconfigured. (b) Verify NS delegation at registrar. (c) Delegate to `huaweicloud-ces-ops` for alarm wiring. |
| Cross-skill | `huaweicloud-ces-ops` (monitoring + alarm) |

## Pattern 2: Resolution Latency

| Field | Value |
|---|---|
| Metric | `dns_request_latency_p99` |
| Window | 5 min |
| Threshold | >100ms sustained |
| Action | (a) Check Huawei Cloud DNS status page. (b) If regional, consider Anycast. |
| Cross-skill | `huaweicloud-ces-ops` |

## Pattern 3: Delegation Chain Failure

| Field | Value |
|---|---|
| Signal | NS record mismatch between registrar and Huawei Cloud DNS |
| Action | Verify NS records at registrar; update if stale |
| Cross-skill | N/A (registrar-dependent) |

## Pattern 4: TTL Storm

| Field | Value |
|---|---|
| Metric | `dns_request_count` per second |
| Window | 1 min |
| Threshold | >1000 req/s for same name |
| Action | Increase TTL for the target record; investigate cause (possible attack) |
| Cross-skill | `huaweicloud-ces-ops` |

## Cross-Skill Delegation Matrix (AIOps)

| Pattern | DNS | CES | EIP | CDN |
|---|:-:|:-:|:-:|:-:|
| 1 NXDOMAIN spike | ✅ | ✅ | | |
| 2 Resolution latency | ✅ | ✅ | | |
| 3 Delegation failure | ✅ | | | |
| 4 TTL storm | ✅ | ✅ | | |
