# AIOps Patterns — SWR

> **Purpose**: SWR-specific anomaly detection patterns for container image registry.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `storage_quota_near` | Image storage > 80% | Warning | 清理镜像 |
| `pull_throttling` | Pull rate limit near | Warning | 优化镜像大小 |
| `webhook_failure_high` | Webhook failure > 5% | Warning | 检查 webhook 配置 |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `image_count_growth` | Growth rate acceleration | Warning | 正常增长 |
| `pull_count_drop` | Pulls < 50% of average | Warning | 检查业务 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `pull_latency_spike` | P99 > 1s | Warning | 检查存储 |
| `build_failure_spike` | Build failures > 10% | Critical | 检查 Dockerfile |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `storage_pull_correlation` | High storage + low pulls | Warning | 清理无用镜像 |

---

## 2. Alarm Storm Handling

### 2.1 Threshold Configuration

```yaml
# CES alarm rules for SWR
alarm_rules:
  - metric: swr_storage_usage_percent
    threshold: 80
    period: 300
    evaluation_periods: 2
    severity: Warning

  - metric: swr_pull_latency_p99
    threshold: 1000  # ms
    period: 60
    evaluation_periods: 3
    severity: Warning

  - metric: swr_webhook_failure_rate
    threshold: 0.05  # 5%
    period: 300
    evaluation_periods: 2
    severity: Warning
```

### 2.2 Suppression Strategy

- **Time-window based**: Suppress duplicate alarms within 15 minutes for same pattern
- **Severity escalation**: Critical alarms bypass suppression
- **Aggregation**: Multiple instances of same pattern aggregated into single notification

### 2.3 Root Cause Analysis

1. **Storage pressure** → Check image cleanup policy → Analyze image age distribution
2. **Pull throttling** → Review rate limits → Optimize image layer structure
3. **Webhook failures** → Verify endpoint health → Check network connectivity
