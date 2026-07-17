# Capacity Forecasting — Huawei Cloud CCE

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

**Use case**: Load patterns with clear seasonality (batch workloads, cron jobs)
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

## 2. CCE-Specific Metrics

| Metric | Namespace | Unit | Quota Reference |
|--------|-----------|------|-----------------|
| `cpu_usage` | SYS.CCE | % | CCE node CPU quota |
| `mem_usage` | SYS.CCE | % | CCE node memory quota |
| `disk_usage` | SYS.CCE | % | Node disk usage |
| `cpu_usage_by_node` | SYS.CCE | % | Per-node CPU |
| `mem_usage_by_node` | SYS.CCE | % | Per-node memory |
| `pod_count` | SYS.CCE | count | Pods per node (max 256) |
| `service_count` | SYS.CCE | count | Cluster service count |
| `deployment_replicas` | SYS.CCE | count | Desired vs actual replicas |

---

## 3. Capacity Planning Workflow

### Step 1: Collect Historical Data

```bash
# Query historical metrics from CES
REGION="{{env.HW_REGION_ID}}"
CLUSTER_ID="[cluster_id]"

# Node CPU usage
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.CCE" \
  --metric-name "cpu_usage" \
  --dim.0 "cluster_id=$CLUSTER_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json

# Node memory usage
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.CCE" \
  --metric-name "mem_usage" \
  --dim.0 "cluster_id=$CLUSTER_ID" \
  --period 3600 \
  --from "$(( $(date +%s) - 86400 * 30 ))" \
  --to "$(date +%s)" \
  --output json

# Pod count trend
hcloud ces query-metric-data \
  --region "$REGION" \
  --namespace "SYS.CCE" \
  --metric-name "pod_count" \
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

def predict_exhaustion(current_value, quota_limit, daily_growth_rate):
    """Predict days until resource exhaustion."""
    if daily_growth_rate <= 0:
        return None

    days_to_exhaustion = (quota_limit - current_value) / daily_growth_rate
    return days_to_exhaustion
```

### Step 3: Node-Level Analysis

```python
def analyze_node_capacity(cluster_id, node_metrics):
    """
    Analyze per-node capacity and identify bottlenecks.
    Returns list of nodes sorted by exhaustion risk.
    """
    node_analysis = []

    for node in node_metrics:
        cpu_slope, cpu_r2 = calculate_growth_rate(node['cpu_history'])
        mem_slope, mem_r2 = calculate_growth_rate(node['mem_history'])

        node_analysis.append({
            'node_id': node['node_id'],
            'cpu_days_to_exhaust': predict_exhaustion(
                node['current_cpu'], 100, cpu_slope
            ),
            'mem_days_to_exhaust': predict_exhaustion(
                node['current_mem'], 100, mem_slope
            ),
            'cpu_confidence': cpu_r2,
            'mem_confidence': mem_r2,
            'risk_level': min(
                node['cpu_days_to_exhaust'] or 999,
                node['mem_days_to_exhaust'] or 999
            )
        })

    return sorted(node_analysis, key=lambda x: x['risk_level'])
```

### Step 4: Generate Capacity Report

```yaml
capacity_report:
  resource_id: "[CCE cluster_id]"
  product: "CCE"
  generated_at: "[timestamp]"

  cluster_overview:
    total_nodes: [count]
    total_pods: [count]
    namespace_count: [count]

  current_usage:
    cluster_cpu_util: [value]
    cluster_mem_util: [value]
    cluster_disk_util: [value]
    collected_at: "[timestamp]"

  growth_analysis:
    model: "[linear|seasonal|exponential]"
    daily_growth_rate:
      cpu: [rate]
      memory: [rate]
      pods: [rate]
    r_squared: [confidence]
    trend_direction: "[increasing|decreasing|stable]"

  node_analysis:
    - node_id: "[node_id]"
      days_to_cpu_exhaust: [days]
      days_to_mem_exhaust: [days]
      risk_level: "[critical|high|medium|low]"

  predictions:
    - metric: "cpu_usage"
      predicted_value_at: "[future_date]"
      predicted_value: [value]
      confidence: [0-1]
    - metric: "pod_count"
      predicted_value_at: "[future_date]"
      predicted_value: [value]
      confidence: [0-1]

  exhaustion_analysis:
    cluster_cpu:
      quota_limit: 100
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"
    cluster_memory:
      quota_limit: 100
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"
    pod_count:
      quota_limit: [max_pods_per_node * node_count]
      days_to_exhaustion: [days]
      exhaustion_date: "[date]"
      risk_level: "[critical|high|medium|low]"

  recommendations:
    - action: "[add_node|scale_up|optimize_pods|cluster_upgrade]"
      target: "[cpu|memory|pods]"
      estimated_cost: "[cost_impact]"
      priority: "[P0|P1|P2]"
```

---

## 4. Capacity Alert Rules

| Metric | Warning Threshold | Critical Threshold | Recommended Action |
|--------|-----------------|-------------------|-------------------|
| Node CPU utilization trend | 30 days to >80% | 14 days to >90% | Add node to pool / optimize workloads |
| Node memory utilization trend | 30 days to >85% | 14 days to >95% | Optimize pod memory / add node |
| Pod count per node trend | 30 days to >180 pods | 14 days to >230 pods | Scale cluster / optimize pod density |
| Cluster disk usage trend | 60 days to >85% | 30 days to >95% | Add node disk / cleanup |
| Deployment replicas mismatch | 30 days to >10% drift | 14 days to >20% drift | Investigate scheduling issues |

---

## 5. Model Selection Guide

| Scenario | Recommended Model | Why |
|----------|-----------------|-----|
| Stable, linear growth (storage) | Linear Regression | Simple, reliable for steady trends |
| Periodic workload (batch/CI) | Seasonal Decomposition | Captures weekly/monthly cycles |
| Rapid scaling events | Exponential Smoothing | Reacts quickly to changes |
| Multi-node cluster | Ensemble (per-node + cluster) | Combines node and cluster patterns |
| New cluster (< 7 days data) | Rule-based | Insufficient data for ML |

---

## 6. CCE-Specific Considerations

### 6.1 Cluster Limits

| Cluster Version | Max Nodes | Max Pods/Node | Max Services |
|-----------------|-----------|---------------|--------------|
| v1.21+ | 500 | 256 | 10000 |
| v1.25+ | 1000 | 256 | 20000 |

> Query current quotas: `hcloud cce list-clusters --region {{env.HW_REGION_ID}}`

### 6.2 Scaling Operations

| Operation | Command | Cooldown |
|-----------|---------|----------|
| Add node to pool | `hcloud cce resize-node-pool --cluster-id X --node-pool-id Y --count Z` | 5 min |
| Scale cluster | `hcloud cce resize-cluster --cluster-id X --node-count Y` | 10 min |
| Update node template | `hcloud cce update-node-pool --node-pool-id X --template Y` | 3 min |

---

## 7. Compliance Checklist

- [ ] All 3 prediction models documented
- [ ] Capacity planning workflow implemented
- [ ] Alert rules defined for all key CCE metrics
- [ ] Model selection guide provided
- [ ] Minimum data requirements specified
- [ ] Confidence thresholds defined
- [ ] CCE-specific cluster limits documented
- [ ] Node-level capacity analysis included
