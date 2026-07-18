# Alarm Storm Handling — ECS

> **Purpose**: Handle alarm storms for ECS caused by resource exhaustion, network bursts, and spot instance recycling.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

| Threshold | Time Window | Severity |
|-----------|-------------|----------|
| cpu_util > 95% | 10 min | Critical |
| diskUsage > 90% | 实时 | Critical |
| mem_usedPercent 斜率 > 0.5%/min | 10 min | Warning（内存泄漏） |
| pps > 10× 基线 | 5 min | Critical（网络风暴） |
| 竞价实例回收告警 ≥3 | 实时 | Warning |
| 单 AZ ≥10 台 ECS 同异常 | 10 min | Critical |

## 2. Aggregation Rules

| Original Alarm | Aggregated To |
|---------------|---------------|
| 多台 ECS cpu_util > 95% | 「ECS CPU 满载风暴」(按 AZ/可用区) |
| 多台 diskUsage > 90% | 「ECS 磁盘耗尽风暴」 |
| 内存斜率异常多台 | 「内存泄漏风暴」 |
| ELB active_connection↓ + ECS cpu_util↓ + ELB 504↑ + RDS connections↓ | 「上游级联故障」单根因事件 |

## 3. Suppression Rules

| Condition | Action |
|-----------|--------|
| 同 AZ ≥5 台相同指标告警 | 聚合为单条摘要，计数不逐台轰炸 |
| 竞价实例回收告警 | 抑制重复，聚合为「竞价池回收」 |
| 已确认级联故障 | 抑制下游（ELB 504/RDS 连接↓）子告警 |
| cpu_util 已 Critical | 5 min 内抑制同指标重复告警 |

## 4. Response Procedures

### P1 — Critical (CPU 满载 / 磁盘耗尽 / 网络风暴)
```
1. 确认是否级联：查 ELB 与 RDS 同源指标
2. 扩容或疏散热点实例：hcloud ecs resize / 迁移
3. 网络风暴：联动 WAF/ELB 限流
4. 竞价回收：补充按需实例保底
```

### P2 — Warning (内存泄漏斜率)
```
1. 定位泄漏进程，重启或回滚版本
2. 调整内存规格或迁移至大内存型
```

### P3 — Minor
```
1. 记录趋势，规划规格优化
```

```bash
# 查看实例与监控（子命令以 hcloud ecs --help 为准）
hcloud ecs list
hcloud ces show-metric --metric-name cpu_util
```

## 5. Delegation Matrix

| Scenario | Delegate To | Escalation Trigger |
|----------|-------------|-------------------|
| 指标采集/告警异常 | `huaweicloud-ces-ops` | 指标缺失 |
| 应用性能/日志 | `huaweicloud-aom-ops` | AOM 看板异常 |
| 主机安全/入侵 | `huaweicloud-hss-ops` | 暴力破解告警 |
| 上游流量/504 | `huaweicloud-elb-ops` | 连接数持续下降 |
| Web 攻击导致风暴 | `huaweicloud-waf-ops` | CC 攻击 |
| 数据库连接耗尽 | `huaweicloud-rds-ops` | connections 触底 |
| 容器化 ECS 节点 | `huaweicloud-cce-ops` | Node NotReady |
