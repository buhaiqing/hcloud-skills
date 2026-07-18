# Alarm Storm Handling — Billing

> **Purpose**: Handle alarm storms caused by budget thresholds, balance/credit exhaustion, and cost anomaly bursts.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

| Threshold | Time Window | Severity |
|-----------|-------------|----------|
| 余额/信用 < ¥500 | 实时 | Warning |
| 余额/信用 < ¥100 | 实时 | Critical |
| 预算消耗率 > 80% | 实时 | Warning |
| 预算预测消耗 > 100% | 实时 | Critical |
| create-budget 阈值设 0% | 触发即发 | Critical（一次性 storm 根因） |
| ≥5 条 cost center 告警/10min | 10 min | Warning |

## 2. Aggregation Rules

| Original Alarm | Aggregated To |
|---------------|---------------|
| 多条单资源余额告警 | 按 cost center 聚合为「成本中心余额风暴」 |
| 多预算消耗率越界 | 「预算消耗风暴」(按 cost center) |
| create-budget 0% 触发的一次性告警洪峰 | 「预算阈值配置缺陷」单根因事件 |
| 跨产品（ECS/RDS/OBS）费用突增 | 「跨产品成本风暴」 |

## 3. Suppression Rules

| Condition | Action |
|-----------|--------|
| create-budget 阈值 = 0% 触发洪峰 | 抑制重复告警，仅报一次「阈值下限缺陷」，必须设阈值下限 |
| 同一 cost center 3+ 条告警 | 聚合为单条摘要，计数不轰炸 |
| 余额告警 < ¥100 且已通知 | 5 min 内抑制重复，仅持续计数 |
| 预算消耗率已达 Critical | 升级后抑制同阈值级联告警 |

## 4. Response Procedures

### P1 — Critical (余额 < ¥100 / 预测超 100% / 0% 阈值 storm)
```
1. 确认是否 0% 阈值误配：hz 排查 create-budget 参数
2. 充值或调整信用额度，解除停机风险
3. 修正预算阈值下限（禁止 0%）
4. 通知财务与业务方
```

### P2 — Warning (余额 < ¥500 / 消耗率 > 80%)
```
1. 按 cost center 聚合告警，定位消耗来源
2. 联动 ecs-ops/rds-ops/obs-ops 核查资源使用
3. 推动降配或释放闲置资源
```

### P3 — Minor
```
1. 记录并分析消耗趋势
2. 调整预算阈值至合理下限
```

```bash
# 列出成本中心用量（示例，子命令以 hcloud billing --help 为准）
hcloud billing list --cost-center <cc_id>
hcloud ecs list --project-id <project_id>
```

## 5. Delegation Matrix

| Scenario | Delegate To | Escalation Trigger |
|----------|-------------|-------------------|
| ECS 闲置资源导致费用突增 | `huaweicloud-ecs-ops` | 降配后仍超预算 |
| RDS 实例费用异常 | `huaweicloud-rds-ops` | 存储扩容未停 |
| OBS 存储费用风暴 | `huaweicloud-obs-ops` | 生命周期未配置 |
| 日志存储费用异常 | `huaweicloud-lts-ops` | 索引 retention 过长 |
| 监控指标费用异常 | `huaweicloud-ces-ops` | 自定义指标过多 |
| 审计费用异常 | `huaweicloud-cts-ops` | 追踪器范围过大 |
