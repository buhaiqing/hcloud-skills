# Capacity Forecasting — Huawei Cloud RDS

> **Purpose**: Predict resource exhaustion and growth trends for proactive capacity management.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §14
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Prediction Models

### 1.1 Linear Regression (Stable Growth)

```
y = mx + b

Where:
- y = predicted metric value
- m = slope (growth rate per day)
- b = current value

Exhaustion_Date = (Quota_Limit - Current_Value) / m
```

**Use case**: Stable, linear growth patterns (storage usage, connection count)
**Accuracy**: Medium
**Complexity**: Low

### 1.2 Seasonal Decomposition (Periodic)

```
y(t) = Trend(t) + Seasonal(t) + Residual(t)

Where:
- Trend = long-term growth direction
- Seasonal = periodic pattern (weekly/monthly)
- Residual = noise/anomalies
```

**Use case**: Load patterns with clear seasonality (business hours, month-end)
**Accuracy**: High
**Complexity**: Medium

### 1.3 Exponential Smoothing

```
Forecast = α × Last_Value + (1-α) × Previous_Forecast

Where α = smoothing factor (0.1 ~ 0.3)
```

**Use case**: Short-term prediction, trending data
**Accuracy**: High
**Complexity**: Low

---

## 2. RDS-Specific Metrics

| Metric | Namespace | Unit | Quota Reference |
|--------|-----------|------|-----------------|
| `cpu_util` | SYS.RDS | % | RDS instance CPU quota |
| `memory_used_percent` | SYS.RDS | % | RDS instance memory quota |
| `disk_util` | SYS.RDS | % | RDS storage quota |
| `connection_count` | SYS.RDS | count | Max connections quota |
| `qps` | SYS.RDS | count/s | RDS QPS quota |
| `slow_queries` | SYS.RDS | count | Slow query threshold |
| `transaction_logs_storage` | SYS.RDS | MB | Transaction log storage |
| `binlog_storage` | SYS.RDS | MB | Binary log storage |
| `tps` | SYS.RDS | count/s | Transaction per second |

---

## 3. Capacity Planning Workflow

### Step 1: Collect Historical Data

```bash
# Query historical metrics from CES
REGION="{{env.HW_REGION_ID}}"
INSTANCE_ID="[instance_id]"

# CPU utilization
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.RDS" \
  --metric-name "cpu_util" \
  --dim.0 "instance_id=$INSTANCE_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json

# Storage usage
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.RDS" \
  --metric-name "disk_util" \
  --dim.0 "instance_id=$INSTANCE_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json

# Connection count
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.RDS" \
  --metric-name "connection_count" \
  --dim.0 "instance_id=$INSTANCE_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json

# QPS trend
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.RDS" \
  --metric-name "qps" \
  --dim.0 "instance_id=$INSTANCE_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json
```

### Step 2: Calculate Growth Rate

```python
import numpy as np

def calculate_growth_rate(values):
    """Calculate daily growth rate using linear regression."""
    x = np.arange(len(values))
    coefficients = np.polyfit(x, values, 1)
    slope = coefficients[0]

    y_pred = np.polyval(coefficients, x)
    ss_res = np.sum((values - y_pred) ** 2)
    ss_tot = np.sum((values - np.mean(values)) ** 2)
    r_squared = 1 - (ss_res / ss_tot) if ss_tot != 0 else 0

    return slope, r_squared

def predict_exhaustion(current_value, quota_limit, daily_growth_rate):
    """Predict days until resource exhaustion."""
    if daily_growth_rate <= 0:
        return None

    days_to_exhaustion = (quota_limit - current_value) / daily_growth_rate
    return days_to_exhaustion

def analyze_storage_growth(values, binlog_values):
    """
    Analyze combined storage growth including data and logs.
    Returns total days to exhaustion.
    """
    data_slope, _ = calculate_growth_rate(values)
    binlog_slope, _ = calculate_growth_rate(binlog_values)

    total_slope = data_slope + binlog_slope
    return total_slope
```

### Step 3: Database-Specific Analysis

```python
def predict_connection_exhaustion(current_connections, max_connections, daily_growth_rate):
    """
    Predict connection pool exhaustion.
    RDS MySQL: max_connections = max_connections parameter
    RDS PostgreSQL: max_connections = max_connections setting
    """
    if daily_growth_rate <= 0:
        return None

    available = max_connections - current_connections
    days_to_exhaustion = available / daily_growth_rate
    return days_to_exhaustion

def detect_seasonality(values, period=24):
    """
    Detect hourly seasonality (daily pattern).
    period=24 for hourly data showing daily cycle.
    """
    if len(values) < period * 2:
        return None

    fft = np.fft.fft(values)
    frequencies = np.fft.fftfreq(len(values))

    half = len(fft) // 2
    dominant_idx = np.argmax(np.abs(fft[1:half])) + 1
    dominant_freq = frequencies[dominant_idx]

    period_detected = int(round(1 / dominant_freq)) if dominant_freq != 0 else None
    return period_detected
```

### Step 4: Generate Capacity Report

```yaml
capacity_report:
  resource_id: "[RDS instance_id]"
  product: "RDS"
  generated_at: "[timestamp]"

  database_info:
    engine: "[MySQL|PostgreSQL|SQL Server]"
    version: "[version]"
    instance_type: "[type]"

  current_usage:
    cpu_util: [value]
    memory_used_percent: [value]
    disk_util: [value]
    connection_count: [value]
    qps: [value]
    collected_at: "[timestamp]"

  growth_analysis:
    model: "[linear|seasonal|exponential]"
    daily_growth_rate:
      cpu: [rate]
      memory: [rate]
      disk: [rate]
      connections: [rate]
    r_squared: [confidence]
    trend_direction: "[increasing|decreasing|stable]"
    seasonal_pattern: "[daily|weekly|monthly|none]"

  predictions:
    - metric: "disk_util"
      predicted_value_at: "[future_date]"
      predicted_value: [value]
      confidence: [0-1]
    - metric: "connection_count"
      predicted_value_at: "[future_date]"
      predicted_value: [value]
      confidence: [0-1]
    - metric: "qps"
      predicted_value_at: "[future_date]"
      predicted_value: [value]
      confidence: [0-1]

  exhaustion_analysis:
    disk:
      quota_limit: [storage_gb]
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"
    connections:
      quota_limit: [max_connections]
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"
    cpu:
      quota_limit: 100
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"

  recommendations:
    - action: "[expand_storage|scale_instance|optimize_queries|tune_connections]"
      target: "[disk|cpu|memory|connections]"
      estimated_cost: "[cost_impact]"
      priority: "[P0|P1|P2]"
```

---

## 4. Capacity Alert Rules

| Metric | Warning Threshold | Critical Threshold | Recommended Action |
|--------|-----------------|-------------------|-------------------|
| CPU utilization trend | 30 days to >75% | 14 days to >90% | Scale up instance type / optimize queries |
| Memory utilization trend | 30 days to >80% | 14 days to >95% | Optimize buffer pool / scale up |
| Disk usage growth | 60 days to >80% | 30 days to >95% | Expand storage / cleanup logs |
| Connection count trend | 30 days to >80% quota | 14 days to >95% quota | Tune max_connections / scale up |
| QPS trend | 30 days to >80% quota | 14 days to >95% quota | Scale up / optimize slow queries |
| Slow query trend | 30 days to >100/hour | 14 days to >500/hour | Optimize indexes / rewrite queries |

---

## 5. Model Selection Guide

| Scenario | Recommended Model | Why |
|----------|-----------------|-----|
| Stable storage growth | Linear Regression | Simple, reliable for steady trends |
| Business hour patterns | Seasonal Decomposition | Captures daily/weekly cycles |
| Rapid connection growth | Exponential Smoothing | Reacts quickly to changes |
| Mixed workload (OLTP/OLAP) | Ensemble (weighted average) | Combines multiple patterns |
| New instance (< 7 days) | Rule-based | Insufficient data for ML |

---

## 6. RDS-Specific Considerations

### 6.1 Instance Limits

| Instance Type | Max Storage (GB) | Max Connections | Max QPS |
|---------------|------------------|-----------------|---------|
|通用型 | 1000 | 4000 | 15000 |
|独享型 | 4000 | 16000 | 100000 |
|哈勃型 | 6000 | 32000 | 200000 |

> Query current quotas: `hcloud rds list-quotas --region {{env.HW_REGION_ID}}`

### 6.2 Scaling Operations

| Operation | Command | Cooldown |
|-----------|---------|----------|
| Expand storage | `hcloud rds resize-instance --instance-id X --volume-size Y` | 5 min |
| Change instance type | `hcloud rds resize-instance --instance-id X --flavor-id Y` | 15 min |
| Modify max_connections | `hcloud rds set-parameter --instance-id X --param max_connections=Z` | 1 min |

### 6.3 Storage Auto-Scaling

RDS supports automatic storage expansion:
```bash
# Enable auto-scaling
hcloud rds enable-storage-auto-expand --instance-id X --threshold 80

# Check auto-scaling policy
hcloud rds list-storage-auto-expand --instance-id X
```

---

## 7. Compliance Checklist

- [ ] All 3 prediction models documented
- [ ] Capacity planning workflow implemented
- [ ] Alert rules defined for all key RDS metrics
- [ ] Model selection guide provided
- [ ] Minimum data requirements specified
- [ ] Confidence thresholds defined
- [ ] RDS-specific instance limits documented
- [ ] Storage auto-scaling documented
