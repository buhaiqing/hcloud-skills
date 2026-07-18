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

## 官方错误码参考（API Error Codes）

> **权威说明**：下表为华为云 CDN API **真实返回**的错误码（`CDN.*` 命名空间），故障定位时优先匹配。上文「Error Code Quick Reference」中的 `DomainNotFound` / `RefreshQuotaExceeded` / `InvalidOrigin` 等为本 runbook 的**文档约定码**（语义化助记，非 API 原始返回码），仅用于正文 T1–T5 流程引用；当 `hcloud` 返回 `CDN.*` 码时以下表为准。

来源：[华为云 CDN API 参考 — 附录：错误码](https://support.huaweicloud.com/api-cdn/ErrorCode.html)。以下为真实错误码，可直接用于故障定位。

| Error Code | Meaning | Recovery |
|---|---|---|
| `CDN.0101` | 加速域名已存在 | 提交工单处理 |
| `CDN.0102` | 域名未备案 / 备案过期（审核失败） | 先完成工信部备案再重试 |
| `CDN.0104` | 加速域名数量已达上限（配额超限） | 提交工单提升域名配额 |
| `CDN.0105` | 加速域名不存在 | 核对域名；若正确则提交工单 |
| `CDN.0106` | 当前域名状态不支持该操作（如已停用 CDN） | 检查域名是否被封禁/锁定；仅启用态可停用 |
| `CDN.0109` | 源站域名不能与加速域名相同 | 更换为不同域名作为源站 |
| `CDN.0110` | 刷新/预热 URL 数量超出限制 | 提交工单提升刷新/预热配额 |
| `CDN.0114` | CDN 服务未开通（计费模式未配置） | 开通 CDN 服务并选择计费模式 |
| `CDN.0115` | 加速域名被封禁 | 提交工单申请解封 |
| `CDN.0001` | 参数错误（格式错误或缺参） | 查阅 API 文档并修正参数 |
| `CDN.0002` | 用户未认证（鉴权失败） | 携带正确的 AK/SK 或 token |
| `CDN.0004` | 权限不足 | 获取所需操作权限，否则提交工单 |
| `CDN.0403` | 证书必须为 PEM 格式 | 上传前转换为 PEM 格式 |
| `CDN.0405` | 证书与私钥不匹配 | 确保证书与私钥配对 |
| `CDN.0406` | 证书与域名不匹配 | 使用为加速域名签发的证书 |
| `CDN.0408` | 证书已过期 | 在证书链中续期证书 |
