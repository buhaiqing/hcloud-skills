# AIOps Patterns — LTS

> **Purpose**: LTS-specific anomaly detection patterns for log ingestion and storage.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `log_ingestion_quota_near` | Ingestion rate > 80% of quota | Warning | 扩容或优化 |
| `storage_quota_near` | Log storage > 80% of quota | Warning | 清理或扩容 |
| `shard_capacity_near` | Shard usage > 85% | Warning | 分裂Shard |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `ingestion_growth_acceleration` | Growth rate increasing | Warning | 分析原因 |
| `log_volume_drop` | Volume < 50% of average | Warning | 检查数据源 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `ingestion_spike` | Rate > 3x baseline | Warning | 检查数据源 |
| `latency_spike` | Query latency > 1s | Warning | 检查存储状态 |
| `error_rate_spike` | Index errors > 1% | Critical | 排查错误 |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `ingestion_storage_correlation` | High ingestion + high storage | Warning | 正常增长 |
| `latency_throughput_correlation` | Low throughput + high latency | Warning | 性能瓶颈 |

---

## 2. Multi-Metric Correlation

| Metric Pair | Correlation | Interpretation |
|------------|-------------|----------------|
| Ingestion Rate + Storage | Positive | Normal growth |
| Query Latency + Queue Depth | Positive | Backpressure |
| Error Rate + Ingention | Negative | Data quality issue |

---

## 3. Alarm Storm Handling

| Condition | Action |
|-----------|--------|
| > 10 alarms in 5 min | Aggregate into single alarm |
| > 50% of log groups affected | Suppress individual alarms |
