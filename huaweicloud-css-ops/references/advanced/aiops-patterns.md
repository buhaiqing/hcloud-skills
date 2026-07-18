# AIOps Patterns — CSS

> **Purpose**: CSS (Elasticsearch) specific anomaly detection patterns for search clusters.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `cluster_unhealthy` | `cluster_health` = Red/Yellow | Critical | 检查 `unassigned_shards` 与节点状态 |
| `disk_write_reject` | `disk_usage` > 90%（写入被拒） | Critical | 清理索引或扩容磁盘 |
| `jvm_heap_pressure` | `jvm_heap` > 85% | Major | 降低 bulk 并发，触发 GC/扩容 |
| `node_loss` | `node_count` < 期望值 | Critical | 排查单节点故障与网络分区 |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `storage_growth_anomaly` | 存储增速 > 2× 基线 | Major | 检查 rollover/ILM 策略是否生效 |
| `shard_imbalance` | 分片分布变异系数 `cv` > 0.3 | Minor | 触发 reroute/weights 均衡 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `query_latency_spike` | 查询 P99 > 基线 + 3σ | Major | 排查慢查询与热点分片 |
| `search_latency_spike` | `search_latency` > 500ms | Major | 检查 cache 命中与 CPU 竞争 |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `single_root_query` | `search_latency` 与 `cpu` 同时异常 | Major | 单根因：CPU 竞争，限流查询 |
| `single_node_failure_cascade` | 单节点故障 → `unassigned_shards`↑ → yellow → red（同 `cluster_id`） | Critical | 恢复故障节点，等待分片重分配 |

---

## 2. Alarm Storm Handling

告警风暴处理策略、抑制与聚合规则详见 `references/advanced/alarm-storm-handling.md`，本文件不重复。

---

## 3. Root Cause Analysis

1. **Yellow/Red 健康态** → 关联 `unassigned_shards` 与 `node_count` → 节点丢失优先恢复节点而非强制分配。
2. **写入被拒** → 检查 `disk_usage` 是否触 watermark → 清理旧索引或临时调高阈值后扩容。
3. **查询延迟尖刺** → 比对 `search_latency` 与 `cpu` → 确认 CPU 竞争类单根因，而非独立慢查询。
4. **JVM 堆压 + 磁盘高** → 双压力叠加时先降查询负载再处理存储，避免 GC 停顿放大写入拒绝。
5. **分片不均 cv > 0.3** → 检查 shard allocation 权重 → 手动 reroute 高负载节点分片。
