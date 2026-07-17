# AIOps Patterns — OBS

> **Purpose**: OBS-specific anomaly detection patterns for object storage operations.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `storage_quota_near` | Bucket size > 80% of quota | Warning | 清理或扩容 |
| `request_throttling` | HTTP 503 responses > 5% | Warning | 限流或扩容 |
| `bandwidth_saturation` | Bandwidth > 90% of limit | Critical | 扩容带宽 |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `storage_growth_acceleration` | Growth rate increasing > 20% week-over-week | Warning | 分析增长原因 |
| `request_count_drop` | Request count < 50% of weekly average | Warning | 检查业务异常 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `traffic_spike` | Request rate > 3x baseline in 5min | Warning | 检查是否攻击 |
| `latency_spike` | P99 latency > 500ms | Warning | 检查后端状态 |
| `error_rate_spike` | 5xx errors > 1% | Critical | 排查错误原因 |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `bandwidth_latency_correlation` | Bandwidth high AND Latency high | Warning | 检查网络问题 |
| `quota_request_correlation` | Quota near AND Request rate high | Critical | 立即扩容 |

---

## 2. Multi-Metric Correlation

| Metric Pair | Correlation | Interpretation |
|------------|-------------|----------------|
| Bandwidth + Latency | Positive | Network bottleneck |
| Request Count + Error Rate | Negative | Service degradation |
| Storage + Request Count | Positive | Usage pattern normal |
| Bandwidth + 503 Rate | Positive | Throttling active |

---

## 3. Alarm Storm Handling

| Condition | Action |
|-----------|--------|
| > 10 alarms in 5 min for same bucket | Aggregate into single alarm |
| > 50% of buckets affected | Suppress individual alarms, send summary |
| Alarm rate increasing | Enable exponential backoff |

---

## 4. Knowledge Base Reference

See `references/knowledge-base.md` for fault patterns and remediation.
