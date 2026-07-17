# Capacity Forecasting — Huawei Cloud CSS

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

**Use case**: Stable, linear growth patterns (storage usage, index size)
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

**Use case**: Load patterns with clear seasonality (log ingestion, time-series data)
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

## 2. CSS-Specific Metrics

| Metric | Namespace | Unit | Quota Reference |
|--------|-----------|------|-----------------|
| `cpu_usage` | SYS.CSS | % | CSS cluster CPU quota |
| `memory_used_percent` | SYS.CSS | % | CSS cluster memory quota |
| `disk_usage` | SYS.CSS | % | CSS cluster disk quota |
| `index_storage_size` | SYS.CSS | GB | Index storage quota |
| `storage_used` | SYS.CSS | GB | Total storage used |
| `cluster_health` | SYS.CSS | status | Cluster health state |
| `search_qps` | SYS.CSS | count/s | Search queries per second |
| `indexing_qps` | SYS.CSS | count/s | Indexing operations per second |
| `active_shards` | SYS.CSS | count | Active shard count |
| `unassigned_shards` | SYS.CSS | count | Unassigned shard count |
| `nodes` | SYS.CSS | count | Number of nodes in cluster |

---

## 3. Capacity Planning Workflow

### Step 1: Collect Historical Data

```bash
# Query historical metrics from CES
REGION="{{env.HW_REGION_ID}}"
CLUSTER_ID="[cluster_id]"

# CPU usage
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.CSS" \
  --metric-name "cpu_usage" \
  --dim.0 "cluster_id=$CLUSTER_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json

# Memory usage
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.CSS" \
  --metric-name "memory_used_percent" \
  --dim.0 "cluster_id=$CLUSTER_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json

# Disk usage
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.CSS" \
  --metric-name "disk_usage" \
  --dim.0 "cluster_id=$CLUSTER_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json

# Index storage size
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.CSS" \
  --metric-name "index_storage_size" \
  --dim.0 "cluster_id=$CLUSTER_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json

# Search QPS
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.CSS" \
  --metric-name "search_qps" \
  --dim.0 "cluster_id=$CLUSTER_ID" \
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

def predict_storage_exhaustion(current_gb, max_gb, daily_growth_rate_gb):
    """Predict days until storage exhaustion."""
    available_gb = max_gb - current_gb
    if daily_growth_rate_gb <= 0:
        return None

    days_to_exhaustion = available_gb / daily_growth_rate_gb
    return days_to_exhaustion

def analyze_shard_health(unassigned_series, active_series):
    """Analyze shard distribution health."""
    avg_unassigned = np.mean(unassigned_series)
    total_shards = np.mean(active_series) + avg_unassigned

    if avg_unassigned > 0:
        unassigned_ratio = avg_unassigned / total_shards
        return {
            'health_status': 'degraded' if unassigned_ratio > 0.1 else 'healthy',
            'unassigned_ratio': unassigned_ratio,
            'recommendation': 'rebalance_or_scale' if unassigned_ratio > 0.1 else 'none'
        }
    return {'health_status': 'healthy', 'unassigned_ratio': 0}
```

### Step 3: Index Growth Analysis

```python
def predict_index_growth(index_sizes_series):
    """
    Predict index storage growth and recommend scaling.
    CSS/ES indices grow with document ingestion.
    """
    growth_rate, confidence = calculate_growth_rate(index_sizes_series)

    return {
        'daily_growth_gb': growth_rate,
        'confidence': confidence,
        'monthly_projection_gb': growth_rate * 30,
        'quarterly_projection_gb': growth_rate * 90
    }

def detect_seasonality(values, period=7):
    """Detect weekly seasonality in CSS metrics."""
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
  resource_id: "[CSS cluster_id]"
  product: "CSS"
  generated_at: "[timestamp]"

  cluster_info:
    version: "[version]"
    instance_type: "[type]"
    node_count: [count]
    total_storage_gb: [value]
    shard_configuration: "[primary_replicas]"

  current_usage:
    cpu_usage: [value]
    memory_used_percent: [value]
    disk_usage: [value]
    index_storage_size_gb: [value]
    search_qps: [value]
    indexing_qps: [value]
    cluster_health: "[green|yellow|red]"
    collected_at: "[timestamp]"

  growth_analysis:
    model: "[linear|seasonal|exponential]"
    daily_growth_rate:
      storage_gb: [rate]
      index_size_gb: [rate]
      search_qps: [rate]
    r_squared: [confidence]
    trend_direction: "[increasing|decreasing|stable]"
    seasonal_pattern: "[daily|weekly|log_ingestion|none]"

  shard_analysis:
    active_shards: [count]
    unassigned_shards: [count]
    health_status: "[healthy|degraded|critical]"

  index_projections:
    monthly_growth_gb: [value]
    quarterly_growth_gb: [value]
    days_to_storage_full: [days]
    exhaustion_date: "[date]"

  predictions:
    - metric: "disk_usage"
      predicted_value_at: "[future_date]"
      predicted_value: [value]
      confidence: [0-1]
    - metric: "index_storage_size"
      predicted_value_at: "[future_date]"
      predicted_value: [value]
      confidence: [0-1]

  exhaustion_analysis:
    storage:
      quota_limit_gb: [max_storage]
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"
    index_storage:
      quota_limit_gb: [max_index_storage]
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"
    qps:
      quota_limit: [max_qps]
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"

  recommendations:
    - action: "[expand_storage|add_nodes|scale_cluster|rebalance_shards|optimize_indices]"
      target: "[storage|compute|shards|indices]"
      estimated_cost: "[cost_impact]"
      priority: "[P0|P1|P2]"
```

---

## 4. Capacity Alert Rules

| Metric | Warning Threshold | Critical Threshold | Recommended Action |
|--------|-----------------|-------------------|-------------------|
| Disk usage trend | 60 days to >80% | 30 days to >95% | Expand storage / delete old indices |
| Index storage trend | 60 days to >80% | 30 days to >95% | Index lifecycle management |
| CPU utilization trend | 30 days to >75% | 14 days to >90% | Add nodes / optimize queries |
| Memory utilization trend | 30 days to >80% | 14 days to >95% | Add nodes / reduce replicas |
| Unassigned shards | >5 for 30 min | >20 for 10 min | Rebalance / investigate node failure |
| Search QPS trend | 30 days to >80% quota | 14 days to >95% quota | Scale cluster / optimize queries |
| Yellow cluster health | >1 hour | >15 minutes | Investigate shard allocation |

---

## 5. Model Selection Guide

| Scenario | Recommended Model | Why |
|----------|-----------------|-----|
| Log ingestion (steady growth) | Linear Regression | Simple, reliable for steady trends |
| Time-series data (periodic) | Seasonal Decomposition | Captures daily/weekly patterns |
| Variable ingestion rates | Exponential Smoothing | Reacts to rate changes |
| Multi-index complex cluster | Ensemble (weighted average) | Combines multiple patterns |
| New cluster (< 7 days) | Rule-based | Insufficient data for ML |

---

## 6. CSS-Specific Considerations

### 6.1 Cluster Limits

| Cluster Type | Max Nodes | Max Storage (GB) | Max Shards |
|--------------|-----------|------------------|------------|
| 单节点 | 1 | 1000 | 500 |
| 3节点集群 | 3 | 5000 | 1500 |
| 5节点集群 | 5 | 10000 | 3000 |
| 多可用区 | 6+ | 50000 | 15000 |

> Query current quotas: `hcloud css list-clusters --region {{env.HW_REGION_ID}}`

### 6.2 Scaling Operations

| Operation | Command | Cooldown |
|-----------|---------|----------|
| Expand storage | `hcloud css expand-cluster --cluster-id X --size Y` | 10 min |
| Add nodes | `hcloud css extend-node --cluster-id X --count Z` | 15 min |
| Modify shard count | `hcloud css modify-shard --cluster-id X --shard-count Y` | 5 min |

### 6.3 Index Lifecycle Management (ILM)

CSS supports ILM for automatic index management:
```bash
# Create ILM policy
hcloud css create-ilm-policy --cluster-id X --policy '{
  "policy": {
    "phases": {
      "hot": {"min_age": "0ms", "actions": {"rollover": {"max_size": "50GB"}}},
      "warm": {"min_age": "7d", "actions": {"shrink": {"number_of_shards": 1}}},
      "cold": {"min_age": "30d", "actions": {"freeze": {}}},
      "delete": {"min_age": "90d", "actions": {"delete": {}}}
    }
  }
}'
```

---

## 7. Compliance Checklist

- [ ] All 3 prediction models documented
- [ ] Capacity planning workflow implemented
- [ ] Alert rules defined for all key CSS metrics
- [ ] Model selection guide provided
- [ ] Minimum data requirements specified
- [ ] Confidence thresholds defined
- [ ] CSS-specific cluster limits documented
- [ ] Index lifecycle management documented
- [ ] Shard health analysis included
