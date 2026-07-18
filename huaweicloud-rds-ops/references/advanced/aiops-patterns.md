# AIOps Patterns — RDS

> **Purpose**: RDS 关系型数据库的 AIOps 异常检测模式，基于真实 CES 监控指标与 CTS 事件。
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `connection_exhaust` | `connections_used / max` > 90% | Major | 扩容连接数上限或优化连接池 |
| `storage_full` | `rds004_disk` `disk_usage` > 85% | Critical | 扩容存储，避免只读锁定 |
| `cpu_high` | `rds001_cpu` 持续高位 | Major | 优化慢查询或升配规格 |
| `mem_high` | `rds002_mem` 持续高位 | Major | 优化缓冲/连接，或升配 |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `slow_query_surge` | `rds043_slow_queries` `slow_query_count` 突增 | Major | 分析执行计划，加索引 |
| `replication_lag` | `rds006_replication_lag` 持续升高 | Major | 查主库写入压力与网络 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `ha_switch` | CTS `ha_switch` 事件触发 | Major | 确认切换原因，检查新主库健康 |
| `read_only` | 数据库进入只读状态 | Critical | 解除只读（扩容存储后），恢复写入 |
| `backup_failed` | 备份任务失败 | Major | 重跑备份，核查存储与权限 |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `RDS-P001` | `rds001_cpu` + `rds002_mem` 双高同窗 | Major | 判定资源瓶颈，升配或优化 |
| `RDS-P002` | 连接饱和 (`connections_used/max>90%`) + `rds001_cpu` 高 | Critical | 连接风暴，限流并扩容 |

---

## 2. Alarm Storm Handling

仅交叉引用，避免重复（TE-6）：详见 `references/advanced/alarm-storm-handling.md`。

---

## 3. Root Cause Analysis

1. **主备切换** → 查 CTS `ha_switch` 与 `rds_ha_lag` (>5s) → 确认切换诱因（故障/手动）。
2. **连接耗尽** → 关联 `connections_used/max` 与 `rds001_cpu` → 定位连接泄漏或慢查询拖垮。
3. **存储只读** → 查 `rds004_disk` `disk_usage` → 扩容后解除只读，清理归档。
4. **慢查询** → 查 `rds043_slow_queries` 与 P99 (>5000ms) → 优化 SQL/索引。
5. **复制延迟** → 查 `rds006_replication_lag` → 降低主库写入或排查网络。
