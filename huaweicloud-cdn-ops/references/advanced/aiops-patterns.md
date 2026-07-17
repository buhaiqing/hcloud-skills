# CDN AIOps Patterns

> Tier-2 advanced content per TE-7: AIOps depth lives here, not in SKILL.md.
> All patterns emit CES metric + cross-skill delegation.

## Pattern 1: Cache Purge Storm

| Field | Value |
|---|---|
| Metric | `refresh_cache_request_count` per hour |
| Window | 1h |
| Threshold | >100 refresh requests in 1h |
| Action | (a) Check if origin can absorb resulting concurrent pulls. (b) Stagger refreshes into batches. (c) Delegate to `huaweicloud-billing-ops` for origin egress cost estimate. |
| Cross-skill | `huaweicloud-billing-ops` (origin cost), `huaweicloud-ces-ops` (alarm wiring) |

## Pattern 2: Hit Rate Degradation

| Field | Value |
|---|---|
| Metric | `flux_hit_rate` |
| Window | 1h rolling average |
| Threshold | hit_rate < 70% for 1h |
| Action | (a) List top miss URLs via CDN statistics API. (b) Increase TTL for static assets. (c) Check for Vary header on personalized content. |
| Cross-skill | `huaweicloud-ces-ops` (monitoring), `huaweicloud-ecs-ops` (if origin is ECS) |

## Pattern 3: Origin 5xx Spike

| Field | Value |
|---|---|
| Metric | `origin_http_code_5xx_rate` |
| Window | 10 min |
| Threshold | >10% of requests return 5xx |
| Action | (a) Check origin server health. (b) Enable CDN error page caching for 5xx. (c) Delegate to origin skill (ECS / OBS / custom). |
| Cross-skill | `huaweicloud-ecs-ops`, `huaweicloud-obs-ops` |

## Pattern 4: Bandwidth DDoS (Spike)

| Field | Value |
|---|---|
| Metric | `outgoing_bandwidth` p99 vs p50 |
| Window | 10 min |
| Threshold | p99 > 10× p50 |
| Action | (a) Enable rate limiting on CDN domain. (b) Delegate to `huaweicloud-eip-ops` for EIP-level rate limit. (c) Investigate traffic source via CDN access logs. |
| Cross-skill | `huaweicloud-eip-ops`, `huaweicloud-ces-ops` |

## Cross-Skill Delegation Matrix (AIOps)

| Pattern | CDN | CES | Billing | EIP | ECS | OBS |
|---|:-:|:-:|:-:|:-:|:-:|:-:|
| 1 Purge storm | ✅ | ✅ | ✅ | | | |
| 2 Hit rate deg | ✅ | ✅ | | | ✅ | |
| 3 Origin 5xx | ✅ | ✅ | | | ✅ | ✅ |
| 4 Bandwidth DDoS | ✅ | ✅ | | ✅ | | |
