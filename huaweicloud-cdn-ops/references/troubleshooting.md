# CDN Troubleshooting Guide — Huawei Cloud CDN

## Top CDN Failure Patterns

### T1: CDN Not Serving Content (403 / 404 from Edge)

| Step | Check | Fix |
|---|---|---|
| 1 | Verify domain is `online` | `hcloud cdn list-domain` → status must be `online` |
| 2 | Verify CNAME resolves at edge | `dig CNAME example.com.cdn.cn-north-4.myhwcdn.com` |
| 3 | Verify origin is reachable from CDN | `curl -I https://example.com -H "Host: example.com"` from CDN region |
| 4 | Check cache key vs origin URL | If origin returns different content per Host header, configure `cache_with_header` |
| 5 | Check Referer hotlink protection | Add user agent / IP to allowlist if blocked |

### T2: Cache Not Refreshing (Stale Content)

| Step | Check | Fix |
|---|---|---|
| 1 | Verify refresh task completed | `hcloud cdn list-tasks --task-type refresh_cache` → status = `finish` |
| 2 | Verify URL matches cache key | Cache key may differ from refresh URL (e.g., `/path` vs `/path/`) |
| 3 | Check TTL is not 0 (no-cache) | If TTL = 0, content is never cached — adjust cache rules |
| 4 | Use directory refresh for multi-file purge | `hcloud cdn refresh-cache --type directory --urls "https://example.com/static/"` |

### T3: High Origin Load (Cache Miss Rate)

| Step | Check | Fix |
|---|---|---|
| 1 | Check hit rate | `hcloud cdn list-stats --stat-type hit_rate` → < 85% is concerning |
| 2 | Identify high-miss paths | Check per-URL stats; common culprits: API calls, auth redirects |
| 3 | Increase TTL for static assets | Modify cache rules: `/*.jpg 86400s` |
| 4 | Add cache key hash for personalized content | Avoid caching user-specific responses |

### T4: CDN Slow / High Latency

| Step | Check | Fix |
|---|---|---|
| 1 | Check CDN region coverage | If `service_area` ≠ global, international users route to distant PoP |
| 2 | Check origin latency | Slow origin = slow CDN even with cache |
| 3 | Check HTTPS handshake overhead | Enable HTTP/2 for connection reuse |
| 4 | Consider HTTP/3 (QUIC) | Available for domains with HTTPS enabled |

### T5: CDN Billing Shock

| Step | Check | Fix |
|---|---|---|
| 1 | Check hit rate trend | Degraded hit rate → more origin egress |
| 2 | Check bandwidth p95 | `hcloud cdn list-stats --stat-type bandwidth` |
| 3 | Check for unexpected traffic (hotlinking) | Referer analysis; add hotlink protection |
| 4 | Review idle domain | Domain `online` with 0 traffic still bills |

## Error Code Quick Reference

| Code | Meaning | Immediate Action |
|---|---|---|
| `DomainNotFound` | Domain ID / name invalid | Verify with `list-domain` |
| `DomainConfiguring` | Domain not yet provisioned | Wait 5–10 min; poll status |
| `DomainOffline` | Domain is stopped | `start-domain` |
| `RefreshQuotaExceeded` | >1000 URLs/day refresh limit | Split across days or use preheat |
| `InvalidOrigin` | Origin IP / hostname unreachable | Verify origin; check security group |
| `CNAMENotConfigured` | CNAME not pointing to CDN | Configure CNAME at DNS registrar |
| `Unauthorized` | IAM `CDN FullAccess` missing | Add IAM policy |
| `QuotaExceeded` | Domain quota hit | Delete unused domains or raise quota |
