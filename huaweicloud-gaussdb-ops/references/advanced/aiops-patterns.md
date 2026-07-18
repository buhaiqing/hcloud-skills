# AIOps Patterns — GaussDB

> **Purpose**: GaussDB-specific anomaly detection patterns for distributed databases.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `conn_pool_exhaust` | `conn / max` > 95% (G-002) | Critical | 排查慢查询，扩连接池或限流 |
| `storage_full` | `disk` > 90% | Critical | 清理归档/扩容存储 |
| `cpu_overload` | `cpu` > 90% | Major | 排查高耗 SQL，扩容规格 |
| `lock_wait_high` | `lock_wait` > 50 | Major | 定位持有锁的长事务 |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `slow_query_drift` | `slow_query` > 100/h 持续上升 | Major | 分析执行计划，建索引/重写 |
| `replication_lag_grow` | `replica_lag` 增速 > 1GB 窗口 | Major | 检查备节点 IO 与网络 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `ha_failover` | 主备切换（`replica_lag` > 10MB 或连接中断） | Critical | 确认新主健康，校验数据一致性 |
| `backup_failure` | 备份任务失败 | Critical | 检查存储/网络，重试备份 |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `slow_query_conn_cascade` | 慢查询激增 → 连接池耗尽 (G-002) | Critical | 单根因：慢 SQL，先 kill 堵源 |
| `failover_connection_break` | 主备切换 → 连接中断 + `replica_lag` 同时异常 | Critical | 单事件，等待切换完成自动恢复 |

---

## 2. Alarm Storm Handling

告警风暴处理策略、抑制与聚合规则详见 `references/advanced/alarm-storm-handling.md`，本文件不重复。

---

## 3. Root Cause Analysis

1. **连接池耗尽 G-002** → 关联 `slow_query` 速率 → 多为慢查询占满连接，先定位并终止堵源 SQL。
2. **主备切换** → 同时观察连接中断与 `replica_lag` → 判定为单切换事件，验证新主接管后再查诱因。
3. **存储满写入失败** → 检查 `disk` 与归档策略 → 清理或扩容，避免触发只读保护。
4. **锁等待堆积** → 关联 `lock_wait` 与长事务 → 定位持有者，必要时回滚阻塞事务。
5. **复制延迟超 1GB** → 检查备节点 IO/网络带宽 → 排除批量写入或备份导致的追赶滞后。
