# CDN Core Concepts — Huawei Cloud Content Delivery Network

## 1. What Is CDN?

Huawei Cloud CDN accelerates content delivery by caching origin content at globally distributed edge nodes. End users receive cached content from the nearest edge node instead of the origin server, reducing latency and origin load.

### CDN Domain Model

```
User → Edge Node (cache hit) → Origin Server (cache miss + fetch)
```

| Component | Description |
|---|---|
| **Accelerated Domain** (加速域名) | The public domain added to CDN (e.g., `example.com`) |
| **CNAME** | DNS alias from domain registrar → CDN edge (`example.com.cdn.cn-north-4.myhwcdn.com`) |
| **Origin Server** (源站) | The actual content source (ECS EIP / OBS bucket / IP address) |
| **Edge Node** (边缘节点) | CDN PoP; caches content based on TTL rules |
| **Cache Key** | The URL used for cache lookup; may differ from origin URL |

## 2. Origin Types

| Type | Code | Use Case | Notes |
|---|---|---|---|
| IP address | `ipaddr` | Self-hosted origin (ECS, bare metal) | Supports port (default 80/443) |
| OBS bucket | `obs` | Static object hosting | Huawei Cloud internal; no egress cost |
| Mirror origin | `mirror` | Dynamic content | Pass-through to origin |
| FunctionGraph | `function` | Serverless CDN | Edge computing |

## 3. Cache Behavior

| Concept | Description |
|---|---|
| **Cache TTL** | Time content stays cached at edge (default: 300s / 5min) |
| **Cache Key** | URL used for cache lookup |
| **Refresh** | Purge specific URLs or directories from all edge caches |
| **Preheat** | Pre-populate cache before a big event (e.g., product launch) |
| **Miss ratio** | `1 - hit_rate`; high miss = more origin load |
| **304 Not Modified** | Edge revalidates with origin; no data transfer |

### Cache Rules (优先级升序)

| Rule type | Match | TTL | Priority |
|---|---|---|---|
| Default | All content | 300s | 0 |
| Homepage | `/` | 60s | 1 |
| Static assets | `/*.css`, `/*.js`, `/*.png` | 86400s | 2 |
| API | `/api/*` | 0 (no-cache) | 3 |

## 4. Service Areas

| Area | Code | Description |
|---|---|---|
| Mainland China | `mainland_china` | CDN nodes within mainland China |
| Outside mainland | `outside_mainland` | CDN nodes outside mainland China |
| Global | `global` | Both; higher cost |

## 5. Domain Status Lifecycle

```
[NOT_EXIST] --create-domain--> [CONFIGURING] --(DNS/CNAME ready)--> [ONLINE] --stop--> [OFFLINE]
                                                                                             |
                                                          --delete--> [DELETING] --> [NOT_EXIST]
```

| Status | Meaning | Agent Action |
|---|---|---|
| `configuring` | Domain is being provisioned | Poll; do not retry |
| `online` | CDN active; serving traffic | Safe to modify or stop |
| `offline` | CDN suspended; no caching | Safe to delete |
| `checking` | CNAME / DNS verification in progress | Wait; do not modify |

## 6. Key Resource Identifiers

| Field | Source | Used as |
|---|---|---|
| `{{output.domain_id}}` | `result[].id` | All domain operations |
| `{{user.domain_name}}` | User input | Domain name (e.g., `example.com`) |
| `{{output.cname}}` | `result[].cname` | DNS CNAME configuration |
| `{{output.job_id}}` | `job_id` from async ops | Cache refresh / preheat polling |
| `{{user.cache_urls}}` | User input | Refresh / preheat targets |

## 7. Quotas (Dynamic — Always Query)

```go
// via Go SDK — GET /v1.0/cdn/domain-detail-quota
//go:build ignore
req := &model.ShowDomainDetailQuotaRequest{}
resp, err := client.ShowDomainDetailQuota(req)
// resp.QuotaInfos[].Used / resp.QuotaInfos[].Quota
```
