# Capacity Forecasting — Huawei Cloud DCS

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

**Use case**: Stable, linear growth patterns (memory usage, key count)
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

**Use case**: Load patterns with clear seasonality (flash sales, campaign events)
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

## 2. DCS-Specific Metrics

| Metric | Namespace | Unit | Quota Reference |
|--------|-----------|------|-----------------|
| `memory_used_ratio` | SYS.DCS | % | DCS instance memory quota |
| `used_memory` | SYS.DCS | MB | DCS instance memory used |
| `keys_count` | SYS.DCS | count | Instance key count |
| `connected_clients` | SYS.DCS | count | Current connections |
| `cmd_qps` | SYS.DCS | count/s | Commands per second |
| `hit_ratio` | SYS.DCS | % | Cache hit ratio |
| `expired_keys` | SYS.DCS | count | Expired keys per second |
| `evicted_keys` | SYS.DCS | count | Evicted keys per second |
| `network_traffic` | SYS.DCS | B/s | Network I/O bytes |

---

## 3. Capacity Planning Workflow

### Step 1: Collect Historical Data

```bash
# Query historical metrics from CES
REGION="{{env.HW_REGION_ID}}"
INSTANCE_ID="[instance_id]"

# Memory usage ratio
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.DCS" \
  --metric-name "memory_used_ratio" \
  --dim.0 "instance_id=$INSTANCE_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json

# Used memory in MB
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.DCS" \
  --metric-name "used_memory" \
  --dim.0 "instance_id=$INSTANCE_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json

# Key count
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.DCS" \
  --metric-name "keys_count" \
  --dim.0 "instance_id=$INSTANCE_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json

# Connected clients
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.DCS" \
  --metric-name "connected_clients" \
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

def predict_memory_exhaustion(current_mb, max_memory_mb, daily_growth_rate_mb):
    """Predict days until memory exhaustion."""
    available_mb = max_memory_mb - current_mb
    if daily_growth_rate_mb <= 0:
        return None

    days_to_exhaustion = available_mb / daily_growth_rate_mb
    return days_to_exhaustion

def predict_key_exhaustion(current_keys, max_keys, daily_growth_rate):
    """Predict days until key count limit."""
    available = max_keys - current_keys
    if daily_growth_rate <= 0:
        return None

    days_to_exhaustion = available / daily_growth_rate
    return days_to_exhaustion
```

### Step 3: Cache Efficiency Analysis

```python
def analyze_cache_efficiency(hit_ratio_series, evicted_series):
    """
    Analyze cache efficiency and predict eviction issues.
    Returns efficiency score and recommendations.
    """
    avg_hit_ratio = np.mean(hit_ratio_series)
    total_evicted = np.sum(evicted_series)
    eviction_rate = np.mean(evicted_series)

    # Low hit ratio + high eviction = memory pressure
    if avg_hit_ratio < 0.5 and eviction_rate > 100:
        return {
            'efficiency': 'low',
            'recommendation': 'scale_up_memory',
            'risk': 'high'
        }
    elif avg_hit_ratio > 0.9 and eviction_rate < 10:
        return {
            'efficiency': 'high',
            'recommendation': 'current_sizing_adequate',
            'risk': 'low'
        }
    else:
        return {
            'efficiency': 'medium',
            'recommendation': 'monitor_trends',
            'risk': 'medium'
        }
```

### Step 4: Generate Capacity Report

```yaml
capacity_report:
  resource_id: "[DCS instance_id]"
  product: "DCS"
  generated_at: "[timestamp]"

  cache_info:
    engine: "[Redis|Memcached]"
    version: "[version]"
    instance_type: "[type]"
    max_memory_mb: [value]
    max_connections: [value]

  current_usage:
    memory_used_ratio: [value]
    used_memory_mb: [value]
    keys_count: [value]
    connected_clients: [value]
    cmd_qps: [value]
    hit_ratio: [value]
    collected_at: "[timestamp]"

  growth_analysis:
    model: "[linear|seasonal|exponential]"
    daily_growth_rate:
      memory_mb: [rate]
      keys: [rate]
      connections: [rate]
    r_squared: [confidence]
    trend_direction: "[increasing|decreasing|stable]"
    seasonal_pattern: "[daily|weekly|campaign|none]"

  cache_efficiency:
    hit_ratio_avg: [value]
    eviction_rate_avg: [value]
    efficiency_score: "[high|medium|low]"
    risk_level: "[critical|high|medium|low]"

  predictions:
    - metric: "memory_used_ratio"
      predicted_value_at: "[future_date]"
      predicted_value: [value]
      confidence: [0-1]
    - metric: "keys_count"
      predicted_value_at: "[future_date]"
      predicted_value: [value]
      confidence: [0-1]

  exhaustion_analysis:
    memory:
      quota_limit_mb: [max_memory]
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"
    keys:
      quota_limit: [max_keys]
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"
    connections:
      quota_limit: [max_connections]
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"

  recommendations:
    - action: "[scale_up|optimize_ttl|cleanup_keys|cluster_scale]"
      target: "[memory|keys|connections]"
      estimated_cost: "[cost_impact]"
      priority: "[P0|P1|P2]"
```

---

## 4. Capacity Alert Rules

| Metric | Warning Threshold | Critical Threshold | Recommended Action |
|--------|-----------------|-------------------|-------------------|
| Memory usage trend | 30 days to >80% | 14 days to >95% | Scale up instance / optimize key sizes |
| Key count trend | 30 days to >80% limit | 14 days to >95% limit | Cleanup expired keys / scale |
| Connection trend | 30 days to >80% quota | 14 days to >95% quota | Scale connections / check connection leaks |
| Eviction rate trend | 30 days to >100/s | 14 days to >1000/s | Memory pressure / scale up |
| Hit ratio decline | 30 days to <70% | 14 days to <50% | Data access pattern issue |
| QPS trend | 30 days to >80% quota | 14 days to >95% quota | Scale bandwidth / optimize commands |

---

## 5. Model Selection Guide

| Scenario | Recommended Model | Why |
|----------|-----------------|-----|
| Stable memory growth | Linear Regression | Simple, reliable for steady trends |
| Campaign/flash sales | Seasonal Decomposition | Captures campaign cycles |
| Rapid memory growth | Exponential Smoothing | Reacts quickly to changes |
| Mixed workload | Ensemble (weighted average) | Combines multiple patterns |
| New instance (< 7 days) | Rule-based | Insufficient data for ML |

---

## 6. DCS-Specific Considerations

### 6.1 Instance Limits

| Instance Type | Max Memory (MB) | Max Connections | Max QPS |
|---------------|-----------------|-----------------|---------|
| 128MB | 128 | 10000 | 5000 |
| 512MB | 512 | 20000 | 20000 |
| 1GB | 1024 | 40000 | 50000 |
| 4GB | 4096 | 100000 | 100000 |
| 16GB | 16384 | 200000 | 200000 |
| 32GB | 32768 | 400000 | 400000 |

> Query current quotas: `hcloud dcs list-quotas --region {{env.HW_REGION_ID}}`

### 6.2 Scaling Operations

| Operation | Command | Cooldown |
|-----------|---------|----------|
| Scale up instance | `hcloud dcs resize-instance --instance-id X --capacity Y` | 5 min |
| Enable auto-scaling | `hcloud dcs enable-auto-scaling --instance-id X --threshold 80` | 1 min |
| Modify max connections | `hcloud dcs update-instance --instance-id X --max-clients Y` | 1 min |

### 6.3 DCS Redis-Specific Notes

- **Eviction policies**: `volatile-lru`, `allkeys-lru`, `volatile-ttl`, `noeviction`
- **Memory fragmentation**: Monitor `mem_fragmentation_ratio` > 1.5
- **AOF vs RDB**: AOF uses additional memory during rewrite

---

## 7. Compliance Checklist

- [ ] All 3 prediction models documented
- [ ] Capacity planning workflow implemented
- [ ] Alert rules defined for all key DCS metrics
- [ ] Model selection guide provided
- [ ] Minimum data requirements specified
- [ ] Confidence thresholds defined
- [ ] DCS-specific instance limits documented
- [ ] Cache efficiency analysis included
