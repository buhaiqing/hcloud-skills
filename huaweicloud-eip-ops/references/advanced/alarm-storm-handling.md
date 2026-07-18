# Alarm Storm Handling — EIP

> **Purpose**: 处理 EIP 在带宽打满、状态异常、绑定翻转及成本冲击期间的告警风暴。
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

| 异常信号 | 阈值 | 严重度 |
|----------|------|--------|
| outgoing_bandwidth / bandwidth_size | 0.8 (Warning) / 0.95 (Critical) | Warning→Critical |
| eip_status ≠ ACTIVE | 持续 5 min / 15 min | Warning / Critical |
| eip_association_status | 频繁翻转 | Major |
| 日 outgoing_bytes | > 3× 7 天中位数 | 成本冲击 (Major) |

> 风暴判定：同区域 ≥3 个 EIP 同时触发带宽告警，或单 EIP 5 min 内多信号叠加。

## 2. Aggregation Rules

| 原始告警 | 聚合为 |
|----------|--------|
| 同 EIP status 异常 + association 翻转 | 单一"EIP 状态不稳定"类 |
| 多 EIP 同时带宽打满 | "区域流量高峰"类（抑制逐条 EIP 告警） |
| 带宽打满 + 日流量超中位数 | "成本冲击"类 |

## 3. Suppression Rules

| 抑制条件 | 动作 |
|----------|------|
| 多 EIP 同源带宽高峰 | 抑制逐条，发"区域流量高峰"汇总 |
| 同一 EIP status 抖动 < 5 min | 仅计数，不发即时告警 |
| association 翻转伴随 status 异常 | 合并为单条状态告警 |
| 成本冲击类与带宽类同源 | 抑制重复，仅发成本侧汇总 |

## 4. Response Procedures

```bash
# P1 — 带宽打满 (outgoing_bandwidth/bandwidth_size > 0.95)
hcloud eip show <eip_id>                       # 确认带宽与用量
# 评估扩容（联动 billing-ops 确认成本）：
hcloud eip update-bandwidth <eip_id> --size <new_size>   # 需人工确认

# P2 — EIP 状态异常 (eip_status ≠ ACTIVE > 15min)
hcloud eip list --filter status=ERROR          # 列出异常 EIP
hcloud eip show <eip_id>

# P3 — 绑定翻转 (eip_association_status 抖动)
hcloud eip list --filter association_status=BOUND
```

| 严重度 | 响应时限 | 动作 |
|--------|----------|------|
| P1 (带宽 95%) | 立即 | 扩容评估 + 人工确认 |
| P2 (状态异常) | < 15 min | 排查解绑/后端实例 |
| P3 (绑定翻转) | < 1 h | 检查实例生命周期/脚本 |

## 5. Delegation Matrix

| 场景 | 委派至 | 升级触发 |
|------|---------|----------|
| 绑定实例故障 | `huaweicloud-ecs-ops` | 实例关机导致解绑 |
| NAT 共享带宽 | `huaweicloud-vpc-ops` | SNAT 容量紧张 |
| ELB 后端 EIP | `huaweicloud-elb-ops` | 后端不可达 |
| DDoS 攻击 | `huaweicloud-hss-ops` / Anti-DDoS | packet_out 异常 |
| 成本冲击 | `huaweicloud-billing-ops` | 日流量 > 3× 中位数 |
| 流量监控 | `huaweicloud-ces-ops` | 需指标下钻 |
| 暴露面/网络 | `huaweicloud-vpc-ops` | 安全组问题 |
