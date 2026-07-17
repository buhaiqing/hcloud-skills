# CDN Well-Architected + Three-Pillar Assessment

## 1. Security Pillar (安全支柱)

### IAM Minimum Permissions

| Role | Required Permissions |
|---|---|
| CDN Viewer | `CDN FullControl` or `CDN ReadOnly` |
| CDN Operator | `CDN FullControl` — create / configure / refresh / delete |
| CDN Admin | `CDN FullControl` + `CDN FullAccess` (billing + quota) |

### HTTPS Enforcement

| Setting | Recommendation |
|---|---|
| HTTPS | Enable force HTTPS redirect; avoid mixed content |
| TLS version | Minimum TLS 1.2; prefer TLS 1.3 |
| Certificate | Auto-renewal from Huawei SSL Certificates; delegate to `huaweicloud-waf-ops` |

### Access Control

| Feature | Use Case | Risk |
|---|---|---|
| Referer whitelist | Restrict access to specific domains | Overly restrictive blocks bots |
| IP blacklist/whitelist | Block specific IP ranges | Bypass via spoofing |
| Origin authentication | Prevent direct origin access | Complex; validate URL signature format |

## 2. Stability Pillar (稳定支柱)

### CDN HA Design

| Practice | Recommendation |
|---|---|
| Multi-origin | Configure ≥2 origin IPs / servers for redundancy |
| Origin health check | Enable origin monitoring; auto-remove failed origin |
| CNAME TTL | Set TTL ≤ 300s before domain switch; increase after |

### DR Runbook

1. **Origin failover**: If origin is down → stop CDN domain → point CNAME to backup origin → start domain.
2. **DDoS on CDN**: Delegate to `huaweicloud-eip-ops` (CDN IP is an EIP); enable rate limiting rules.

## 3. FinOps (财务运营)

### 3.1 Billing Model

CDN bills on **egress traffic** (outbound from CDN edge to user) + **bandwidth peak**.

| Component | Unit | Approximate CN Rate |
|---|---|---|
| Mainland China traffic | CNY/GB | ~0.27 |
| International traffic | CNY/GB | ~0.58 |
| Bandwidth peak (95th percentile) | CNY/Mbps/month | varies by tier |

### 3.2 Cost Optimization

| Pattern | Before | After | Saving |
|---|---|---|---|
| Low hit rate (<70%) | Cache TTL = 300s | Increase static asset TTL to 86400s | ~40% origin egress |
| Idle domain | Domain `online` 24×7 | Stop domain when not needed | 100% of idle CDN cost |
| Batch refresh abuse | Refresh every 5 min | Scheduled refresh at 6h interval | Avoids quota limit |
| Global CDN for China-only content | `service_area=global` | `mainland_china` | ~30% lower cost |

### 3.3 Idle Domain Detection

```bash
hcloud cdn list-domain --region {{env.HW_REGION_ID}} --output json \
  | jq -r '.result[] | select(.status=="online") | .domain_name' \
  | while read d; do
    # Query 7-day bandwidth
    echo "$d: $(hcloud cdn list-stats ... --stat-type bandwidth | jq '.total[0].bandwidth // 0') bps"
  done
```

## 4. SecOps (安全运营)

### 4.1 Hotlink Protection

| Trigger | Action |
|---|---|
| Referer block detected in logs | Add allowed referers; enable anti-leech |
| Unexpected traffic spike | Check for hotlinking; add rate limit |

### 4.2 Cache Poisoning Prevention

| Trigger | Action |
|---|---|
| Origin returns Vary: Cookie/Accept-Encoding | Configure `cache_with_header` properly; avoid caching personalized content |

## 5. Operational Efficiency

- **Batch refresh**: Use SDK batch API for >100 URLs; avoid CLI per-URL loop.
- **Preheat before big events**: Preheat 30–60 min before planned traffic spikes.
- **Cache analysis**: Weekly review of top miss URLs; add cache rules.

## Worker Output Contract

> Read-only assessment mode: `{{user.mode}}=well-architected-readonly` → return `{{output.product_assessment}}`.

**Schema:** [worker-output-schema.md](../../huaweicloud-skill-generator/references/worker-output-schema.md)

| Constant | Value |
|----------|-------|
| `skill_id` | `huaweicloud-cdn-ops` |
| `product` | `cdn` |
| Finding `id` | `cdn-{rel\|sec\|cost\|eff}-NNN` |

| `pillars` key | Source sections |
|----------------|-----------------|
| `reliability` | Stability / DR / multi-origin / CNAME TTL |
| `security` | HTTPS / DNSSEC / hotlink protection / IAM |
| `cost` | FinOps / hit rate / idle domain / TTL optimization |
| `efficiency` | Batch refresh / preheat scheduling |

### Example `{{output.product_assessment}}`

```json
{
  "skill_id": "huaweicloud-cdn-ops",
  "product": "cdn",
  "region": "cn-north-4",
  "scope": "account-wide",
  "assessment_date": "2026-06-24T10:00:00+08:00",
  "status": "PARTIAL",
  "partial": false,
  "resource_count": 5,
  "pillars": {
    "reliability": {"score": 80, "status": "assessed", "findings": []},
    "security": {"score": 75, "status": "assessed", "findings": []},
    "cost": {
      "score": 60,
      "status": "assessed",
      "findings": [
        {
          "id": "cdn-cost-001",
          "severity": "Medium",
          "confidence": "HIGH",
          "title": "3 CDN domains online with hit rate < 70%",
          "evidence": "list-stats returned flux_hit_rate < 0.70 for 3 domains over 7 days",
          "recommendation": "Increase TTL for static assets; review cache rules",
          "effort": "quick"
        }
      ]
    },
    "efficiency": {"score": 80, "status": "assessed", "findings": []}
  },
  "recommendations": [
    {"pillar": "cost", "text": "Increase TTL for static assets to improve hit rate"}
  ],
  "trace": {"commands": ["hcloud cdn list-domain ..."], "request_ids": []},
  "errors": []
}
```
