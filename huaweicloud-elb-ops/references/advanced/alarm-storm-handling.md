# Alarm Storm Handling — ELB

> **Purpose**: 处理 ELB 在后端故障、连接风暴、流量激增期间的告警风暴，聚合同源指标避免刷屏。
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

| 异常信号 | 阈值 | 严重度 |
|----------|------|--------|
| m7_req_5xx | > 1% (Critical) / > 5% / 5min | Critical |
| m9_unhealthy_host | > 0 | Critical |
| m5_drop_rate | > 0% | Critical |
| m1_cps | > 3× 基线 (traffic_surge) | Warning |
| m2_act_conn + m5_drop_rate | 连接风暴 | Critical |
| m10 P99 | > 3s | Warning |

> 风暴判定：同一 LB 5 min 内 ≥5 条告警，或 m7/m9/m5 多信号叠加。

## 2. Aggregation Rules

| 原始告警 | 聚合为 |
|----------|--------|
| m9_unhealthy_host + m7_req_5xx + m8 时延 | "后端降级"类（抑制重复 5xx） |
| m5_drop_rate + m1_cps | "连接风暴"类 |
| m1_cps + m2_act_conn 超基线 | "流量激增"类 |

## 3. Suppression Rules

| 抑制条件 | 动作 |
|----------|------|
| 后端降级同源 (m9+m7) | 抑制逐条 5xx，发"后端降级"汇总 |
| 连接风暴 (m5+m1) | 合并为单条，避免每秒刷屏 |
| 同一 LB 3+ 同类告警 | 抑制并计数，每 15 min 汇总 |
| m10 P99 抖动 < 5 min | 仅记录，不发即时告警 |

## 4. Response Procedures

```bash
# P1 — 后端降级 / 连接风暴
hcloud elb list-members <pool_id>              # 查看健康成员
hcloud elb show-member-health <pool_id>        # 确认 unhealthy_host
# 隔离异常后端需人工确认后执行

# P2 — 5xx 激增 (m7_req_5xx > 5%/5min)
hcloud elb show <loadbalancer_id>              # 查看监听器与转发
# 联动 ecs-ops 排查后端实例

# P3 — 流量激增 (m1_cps > 3× 基线)
hcloud elb show-metrics <loadbalancer_id>      # 下钻 CPS/连接数
```

| 严重度 | 响应时限 | 动作 |
|--------|----------|------|
| P1 (后端降级/风暴) | 立即 | 隔离异常后端，人工确认 |
| P2 (5xx 激增) | < 15 min | 排查后端实例/应用 |
| P3 (流量激增) | < 1 h | 评估扩容/限流 |

## 5. Delegation Matrix

| 场景 | 委派至 | 升级触发 |
|------|---------|----------|
| 后端 ECS 故障 | `huaweicloud-ecs-ops` | 实例不健康 |
| 指标下钻 | `huaweicloud-ces-ops` | 需历史曲线 |
| Web 攻击/CC | `huaweicloud-waf-ops` | 5xx 伴异常 UA |
| 网络层阻断 | `huaweicloud-vpc-ops` | 安全组问题 |
| 成本冲击 | `huaweicloud-billing-ops` | 流量激增费用异常 |
