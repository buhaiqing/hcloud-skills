# AIOps Patterns — DCS

> **Purpose**: DCS (分布式缓存Redis) 专属异常检测模式，基于 CES 命名空间 `SYS.DCS` 真实指标。
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `oom_risk` | `memory_usage > 90%` 且 `evicted_keys > 0` | Critical | 扩容/清理大 key，合并为单条 OOM 预警 |
| `network_saturation` | `bytes_out > 80%` 实例带宽上限 | Major | 检查热 key 与大对象，限流或升级规格 |
| `resource_cascade` | `cpu`/`memory`/`connected_clients` 同时 > 80% | Critical | 资源级联耗尽，立即扩容并查热 key |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `hit_rate_decline` | `hit_rate < 70%` 持续 5min 且 `expired_keys` 激增 3x | Major | 排查击穿/雪崩，预热缓存 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `conn_exhaustion` | `connected_clients > 80%` 实例 maxclients 且 `latency` > 2x 基线 | Critical | 排查连接泄漏，调连接池 |
| `cache_penetration` | `hit_rate < 70%` 短时骤降 + `expired_keys` 激增 3x | Major | 热点 key 重建，加互斥锁 |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `multi_metric_same_window` | 5min 内多指标同窗口且偏离基线 2σ | Major | 聚合关联告警，优先查共享根因（热 key/大 key） |

---

## 2. Alarm Storm Handling

仅交叉引用，不重复内容：`详见 references/advanced/alarm-storm-handling.md`

---

## 3. Root Cause Analysis

1. **OOM / evicted_keys** → 查 `memory_usage` 趋势与大 key → 清理或扩容内存 → 合并 `memory_usage`+`evicted_keys` 为单条 OOM 预警。
2. **连接耗尽** → 比 `connected_clients` 与 maxclients → 查连接池配置与泄漏点 → 调小 idle 超时。
3. **命中率骤降** → 关联 `expired_keys` 与 `hit_rate` → 判定击穿/雪崩 → 预热 + 互斥重建。
4. **网络饱和** → 查 `bytes_out` 与热 key 分布 → 拆分大对象或升级带宽规格。
5. **资源级联** → `cpu`/`memory`/`clients` 同时越线 → 热 key 导致 CPU 与连接齐升 → 限流 + 本地缓存兜底。
