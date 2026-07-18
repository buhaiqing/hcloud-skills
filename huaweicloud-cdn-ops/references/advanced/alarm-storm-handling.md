# Alarm Storm Handling — CDN

> **Purpose**: 处理 CDN 在带宽峰值、回源异常、缓存命中率下降、刷新风暴期间的告警风暴。
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

| 异常信号 | 阈值 | 严重度 |
|----------|------|--------|
| bandwidth_flux | > 100 Gbps | Major |
| flux_hit_rate | < 70% (1h) / < 50% (7d) | Warning / Major |
| origin_flux | > 2× 用户流量 | 源站压力 |
| refresh_cache_request_count | > 100/h (purge storm) | Warning |
| origin_http_code_5xx_rate | > 10% | Critical |
| outgoing_bandwidth p99 | > 10× p50 (DDoS) | Critical |

> 风暴判定：同一加速域名 5 min 内 ≥5 条告警，或 ≥3 类信号同时触发。

## 2. Aggregation Rules

| 原始告警 | 聚合为 |
|----------|--------|
| origin 5xx + hit rate 下降 + origin flux 飙升 | "回源异常"类（源站故障，同源合并） |
| purge storm 引发回源 | 抑制后续命中率告警，归因为刷新 |
| bandwidth_flux 峰值 + p99/p50 悬殊 | "流量/DDoS"类 |

## 3. Suppression Rules

| 抑制条件 | 动作 |
|----------|------|
| purge storm 引发命中率下降 | 抑制命中率告警，归因刷新风暴 |
| 回源异常同源 (5xx+hit+flux) | 合并为单条"回源异常" |
| 带宽峰值 < 100Gbps 且无 5xx | 仅记录，不即时告警 |
| 同一域名 3+ 同类告警 | 抑制并计数，每 15 min 汇总 |

## 4. Response Procedures

```bash
# P1 — 回源异常 (origin_http_code_5xx_rate > 10%)
# 1. 确认源站状态（联动 ecs-ops / obs-ops）
# 2. 查看刷新与回源量（占位：以实际 CLI 子命令为准）
#    hcloud cdn show-domain <domain_id>         # 域名配置与状态
#    hcloud cdn show-stats <domain_id>          # 带宽/回源统计

# P2 — 命中率下降 (flux_hit_rate < 70%)
# 排查缓存规则与 purge 风暴，必要时调整 TTL（人工确认）

# P3 — 刷新风暴 (refresh_cache_request_count > 100/h)
# 评估客户端刷新逻辑，限制频率（人工确认）
```

| 严重度 | 响应时限 | 动作 |
|--------|----------|------|
| P1 (回源 5xx) | 立即 | 定位源站，联动 ECS/OBS |
| P2 (命中率降) | < 15 min | 调缓存规则/TTL |
| P3 (purge 风暴) | < 1 h | 限频/排查客户端 |

## 5. Delegation Matrix

| 场景 | 委派至 | 升级触发 |
|------|---------|----------|
| 源站 EIP | `huaweicloud-eip-ops` | 源站 IP 不通 |
| Web 攻击/CC | `huaweicloud-waf-ops` | 回源伴异常请求 |
| 源站 OBS | `huaweicloud-obs-ops` | 对象存储回源失败 |
| 成本冲击 | `huaweicloud-billing-ops` | 回源流量费用异常 |
| 指标下钻 | `huaweicloud-ces-ops` | 需历史曲线 |
| 源站实例 | `huaweicloud-ecs-ops` | 源站实例故障 |
