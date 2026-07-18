# Alarm Storm Handling — VPC

> **Purpose**: 处理 VPC 网络在流量激增或故障期间的告警风暴，聚合噪声、抑制重复指标、跨 skill 联动定位。
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

| 异常信号 | 阈值 | 严重度 |
|----------|------|--------|
| bandwidth_out | > 90% 已购带宽 | Major |
| packet_out | > 100k pps | 疑似 DDoS |
| EIP 绑定/解绑事件 | 实时高频 | Warning |
| 对等连接状态 | PENDING→ACTIVE→REJECTED 频繁翻转 | Major |
| NAT SNAT 连接数 | 接近网关容量 | Major |

> 风暴判定：同一 VPC 在 5 min 内 ≥5 条上述告警，或 ≥3 类信号同时触发。

## 2. Aggregation Rules

| 原始告警 | 聚合为 |
|----------|--------|
| 解绑 EIP + 安全组 0.0.0.0/0 变更 | "暴露面变化"类 |
| bandwidth_out 高 + packet_out 高 | "疑似攻击"类（抑制单一指标，合并展示） |
| 对等连接状态反复翻转 | "对等连接震荡"类 |
| 多子网 NAT SNAT 接近容量 | "NAT 容量紧张"类 |

## 3. Suppression Rules

| 抑制条件 | 动作 |
|----------|------|
| 带宽 + 包量双高同源 | 抑制 packet_out 单指标，仅发"疑似攻击"聚合 |
| 同一安全组 3+ 次变更 | 抑制并计数，每 15 min 汇总一次 |
| 对等连接 PENDING 抖动未达 REJECTED | 仅记录，不发即时告警 |
| 解绑 EIP 与暴露面变更同源 | 合并为单条"暴露面变化"，避免刷屏 |

## 4. Response Procedures

```bash
# P1 — 疑似 DDoS (packet_out > 100k pps)
# 1. 确认 EIP 流量来源（需结合 eip-ops）
hcloud eip list --filter status=ACTIVE
# 2. 检查安全组暴露面
hcloud vpc list-security-groups
# 3. 限速/封禁需人工确认后执行，禁止自动改动生产安全组

# P2 — 带宽打满 (bandwidth_out > 90%)
hcloud vpc show-bandwidth <bandwidth_id>   # 确认已购带宽与用量
# 联动 eip-ops / billing-ops 评估扩容成本

# P3 — 对等连接震荡
hcloud vpc list-peering                  # 查看 peering 状态
hcloud vpc show-peering <peering_id>
```

| 严重度 | 响应时限 | 动作 |
|--------|----------|------|
| P1 (DDoS 疑似) | 立即 | 隔离暴露面、人工确认封禁 |
| P2 (带宽/容量) | < 15 min | 评估扩容，联动 billing |
| P3 (状态震荡) | < 1 h | 记录根因，修复路由/权限 |

## 5. Delegation Matrix

| 场景 | 委派至 | 升级触发 |
|------|---------|----------|
| EIP 流量异常 | `huaweicloud-eip-ops` | packet_out 持续 > 100k pps |
| 后端 ECS 网络瓶颈 | `huaweicloud-ecs-ops` | 实例带宽打满 |
| RDS 跨 VPC 访问失败 | `huaweicloud-rds-ops` | peering 不通 |
| ELB 后端不可达 | `huaweicloud-elb-ops` | 安全组阻断 |
| 成本冲击 | `huaweicloud-billing-ops` | 带宽扩容费用异常 |
| 权限/跨账号 | `huaweicloud-iam-ops` | 路由权限拒绝 |
| 主机入侵迹象 | `huaweicloud-hss-ops` | 检测到恶意流量 |
