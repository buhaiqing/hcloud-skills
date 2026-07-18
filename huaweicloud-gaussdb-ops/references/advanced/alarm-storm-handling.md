# Alarm Storm Handling — GaussDB

> **Purpose**: Handle alarm storms for GaussDB caused by failover, connection exhaustion, storage/cpu pressure, and replication lag.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

| Threshold | Time Window | Severity |
|-----------|-------------|----------|
| replica_lag > 10MB | 实时 | Warning（主备切换信号） |
| conn / max > 95% | 实时 | Critical（连接池耗尽） |
| disk > 90% | 实时 | Critical（存储满） |
| CPU > 90% | 10 min | Critical |
| lock_wait > 50 | 5 min | Warning |
| 复制延迟 > 1GB | 实时 | Critical |
| slow_query > 100/h | 1 h | Warning |
| 备份失败 | 实时 | Critical |

## 2. Aggregation Rules

| Original Alarm | Aggregated To |
|---------------|---------------|
| G-002 连接耗尽 + 慢查询 | 「慢查询引发连接耗尽」单根因 |
| 主备切换 + 连接中断 + replica_lag | 「主备切换」单事件 |
| CPU > 90% + 慢查询 + lock_wait | 「负载过载」同源事件 |
| 多实例 disk > 90% | 按实例组聚合「存储满风暴」 |

## 3. Suppression Rules

| Condition | Action |
|-----------|--------|
| 主备切换引发连接中断+lag | 抑制下游子告警，仅报切换事件 |
| G-002 连接耗尽已关联慢查询 | 抑制重复连接告警，标注根因为慢查询 |
| disk 已 Critical | 抑制 replication/backup 重复，标注根因 |
| slow_query 抖动 | 1 h 内聚合计数，不逐条轰炸 |

## 4. Response Procedures

### P1 — Critical (连接耗尽 / 存储满 / 主备切换 / 复制延迟 >1GB)
```
1. 定位根因：慢查询 / 存储满 / 实例故障
2. 连接耗尽：杀慢查询释放连接（G-002）
3. 存储满：清理或扩容，解除只读
4. 主备切换：确认新主可用，监控 replica_lag 收敛
```

### P2 — Warning (replica_lag / lock_wait / 慢查询)
```
1. 优化慢 SQL，kill 长事务
2. 降低写入压力，观察 lag 收敛
```

### P3 — Minor
```
1. 记录趋势，规划规格/索引优化
```

```bash
# 查看实例与指标（子命令以 hcloud gaussdb --help 为准）
hcloud gaussdb list
hcloud ces show-metric --metric-name conn_usage
```

## 5. Delegation Matrix

| Scenario | Delegate To | Escalation Trigger |
|----------|-------------|-------------------|
| 指标/告警异常 | `huaweicloud-ces-ops` | 监控缺失 |
| 网络不通/VPC 问题 | `huaweicloud-vpc-ops` | 实例不可达 |
| 备份存储异常 | `huaweicloud-obs-ops` | 备份写失败 |
| 权限/账号失效 | `huaweicloud-iam-ops` | 403 操作 |
