# Alarm Storm Handling — DNS

> **Purpose**: 处理 DNS 在解析成功率下降、NXDOMAIN 突增、TTL 风暴期间的告警风暴。
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

| 异常信号 | 阈值 | 严重度 |
|----------|------|--------|
| dns_request_success_rate | < 95% | Major |
| nxdomain_count | > 1000/min 或 > 10× 基线 | NXDOMAIN Spike |
| dns_request_latency_p99 | > 100ms | Warning |
| dns_request_count | > 1000 req/s 同名 (TTL storm) | Warning |
| NS 委托链 | 不一致 | Major |

> 风暴判定：同一域名/zone 5 min 内 ≥5 条告警，或 ≥3 类信号同时触发。

## 2. Aggregation Rules

| 原始告警 | 聚合为 |
|----------|--------|
| NXDOMAIN spike + 解析成功率下降 | "解析异常"类（同源合并） |
| TTL storm 单一域名高 QPS | "TTL 风暴"类（抑制批量同名告警） |
| 延迟 + 成功率双降 | "解析服务劣化"类 |

## 3. Suppression Rules

| 抑制条件 | 动作 |
|----------|------|
| TTL storm 单一域名高 QPS | 抑制逐条同名告警，发单条汇总 |
| NXDOMAIN 同源 | 合并为"解析异常"，避免刷屏 |
| 延迟抖动 < 5 min 未伴成功率降 | 仅记录 |
| NS 委托链不一致伴随成功率降 | 合并为单条配置类告警 |

## 4. Response Procedures

```bash
# P1 — 解析成功率下降 + NXDOMAIN spike
# 1. 查看解析量与健康度（需结合具体产品 CLI 子命令，以下为占位说明）
#    hcloud dns show-zone <zone_id>            # 确认 zone 状态
#    hcloud dns list-records <zone_id>         # 查看记录一致性
# 2. 校验 NS 委托链（占位：以实际 CLI 为准）

# P2 — TTL storm (dns_request_count > 1000 req/s 同名)
# 评估客户端缓存/递归放大，必要时限速（人工确认）

# P3 — 延迟 p99 > 100ms
# 联动 ces-ops 下钻地域级延迟
```

| 严重度 | 响应时限 | 动作 |
|--------|----------|------|
| P1 (成功率降/NXDOMAIN) | 立即 | 校验记录与 NS 链 |
| P2 (TTL storm) | < 15 min | 评估限速/缓存 |
| P3 (延迟劣化) | < 1 h | 下钻地域指标 |

## 5. Delegation Matrix

| 场景 | 委派至 | 升级触发 |
|------|---------|----------|
| 回源/CDN 解析 | `huaweicloud-cdn-ops` | 域名解析影响加速 |
| EIP 解析异常 | `huaweicloud-eip-ops` | 绑定 IP 不通 |
| ELB 域名 | `huaweicloud-elb-ops` | 监听器域名解析 |
| 指标下钻 | `huaweicloud-ces-ops` | 需历史曲线 |
| 成本冲击 | `huaweicloud-billing-ops` | 查询量费用异常 |
