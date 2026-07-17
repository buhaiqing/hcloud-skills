# Capacity Forecasting — Huawei Cloud CSS

> Predictive capacity planning for Cloud Search Service (OpenSearch-compatible):
> disk exhaustion, JVM heap pressure, shard imbalance, and query latency
> growth forecasts.
> **Version:** 1.0.0

## Forecast Types

| Forecast Type | Method | Prediction Window | Input Data | Accuracy Target |
|---------------|--------|-------------------|------------|-----------------|
| Disk exhaustion | Linear regression on disk_usage | 7d before 90% | 30d disk_usage | ±10% |
| JVM heap pressure | ARIMA on jvm_heap_usage | 24h before 90% | 7d jvm_heap | ±15% |
| Shard imbalance | Stddev on shard distribution | Before 70/30 skew | Real-time allocation | — |
| Query latency growth | Holt-Winters on search_latency | 7d before p99 > 200ms | 30d latency | ±20% |

## Data Acquisition

### CSS Cluster Metrics

```bash
# Disk usage time-series (30d)
hcloud ces list-metric-data \
  --namespace SYS.CSS \
  --metric_name disk_usage \
  --dimension "cluster_id={{user.cluster_id}}" \
  --from "$(date -d '30 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 3600 \
  -o json

# JVM heap usage
hcloud ces list-metric-data \
  --namespace SYS.CSS \
  --metric_name jvm_heap_usage \
  --dimension "cluster_id={{user.cluster_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 300 \
  -o json

# Search latency (p99)
hcloud ces list-metric-data \
  --namespace SYS.CSS \
  --metric_name search_latency \
  --dimension "cluster_id={{user.cluster_id}}" \
  --from "$(date -d '30 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 3600 \
  -o json
```

### Cluster State

```bash
# Cluster health and node count
hcloud css list-clusters -o json | jq '
  .clusters[] | {id: .id, name: .name, status: .status,
                 nodeCount: .nodeCount, version: .version}'
```

## Forecast Algorithms

### Disk Exhaustion

```python
def forecast_disk_exhaustion(cluster_id, days_ahead=7):
    """
    Linear regression on disk_usage percent to predict exhaustion.
    """
    history = query_ces(
        namespace="SYS.CSS",
        metric="disk_usage",
        dimensions={"cluster_id": cluster_id},
        window="30d",
        period=3600,
    )

    values = [p["value"] for p in history]
    n = len(values)
    x = list(range(n))
    x_mean, y_mean = sum(x) / n, sum(values) / n

    slope = sum((xi - x_mean) * (yi - y_mean) for xi, yi in zip(x, values)) / \
            sum((xi - x_mean) ** 2 for xi in x)
    intercept = y_mean - slope * x_mean

    current = values[-1]
    daily_growth = slope * 24
    days_to_90 = (90 - current) / daily_growth if daily_growth > 0 else float("inf")

    return {
        "current_usage_pct": current,
        "daily_growth_rate_pct": daily_growth,
        "projected_in_7d": intercept + slope * (n - 1 + 7 * 24),
        "days_to_90": days_to_90,
        "recommendation": "extend_storage" if days_to_90 <= 7 else "monitor",
    }
```

### JVM Heap Pressure

```python
def forecast_jvm_pressure(cluster_id, hours_ahead=24):
    """
    ARIMA forecast on JVM heap usage to predict GC pressure.
    """
    try:
        from statsmodels.tsa.arima.model import ARIMA
    except ImportError:
        return {"error": "statsmodels not available, use linear fallback"}

    history = query_ces(
        namespace="SYS.CSS",
        metric="jvm_heap_usage",
        dimensions={"cluster_id": cluster_id},
        window="7d",
        period=300,
    )

    values = [p["value"] for p in history]
    if len(values) < 50:
        return {"error": "insufficient data for ARIMA"}

    model = ARIMA(values, order=(2, 1, 2))
    fitted = model.fit()
    forecast_steps = hours_ahead * 12  # 5-min periods
    forecast = fitted.forecast(steps=forecast_steps)

    projected_max = max(forecast)
    projected_avg = sum(forecast) / len(forecast)

    return {
        "current_heap_pct": values[-1],
        "forecast_24h_avg": projected_avg,
        "forecast_24h_max": projected_max,
        "pressure_by_24h": projected_max > 90,
        "recommendation": "increase_heap" if projected_max > 85 else "monitor",
    }
```

### Shard Imbalance Detection

```python
def detect_shard_imbalance(cluster_id):
    """
    Analyze shard distribution across data nodes.
    Returns nodes with > 30% deviation from average.
    """
    # Query cluster allocation
    allocation = query_css_api(
        "GET",
        f"/_cluster/stats?level=shards",
        cluster_id=cluster_id,
    )

    nodes = allocation["nodes"]
    shard_counts = [n["shard_count"] for n in nodes.values()]
    avg = sum(shard_counts) / len(shard_counts)
    threshold = avg * 0.3

    imbalanced = {
        node_id: count
        for node_id, count in nodes.items()
        if abs(count - avg) > threshold
    }

    return {
        "total_shards": sum(shard_counts),
        "average_per_node": avg,
        "min_shards": min(shard_counts),
        "max_shards": max(shard_counts),
        "imbalanced_nodes": imbalanced,
        "skew_ratio": max(shard_counts) / min(shard_counts) if min(shard_counts) > 0 else float("inf"),
        "needs_rebalance": max(shard_counts) / min(shard_counts) > 1.5 if min(shard_counts) > 0 else False,
    }
```

## Capacity Planning Tables

### CSS Cluster Scaling

| Resource | Warning | Critical | Action |
|----------|---------|----------|--------|
| Disk usage | > 75% | > 85% | Extend storage or add cold nodes |
| JVM heap | > 80% | > 90% | Increase heap size (scale up) |
| Shard skew | > 1.5× | > 2× | Trigger rebalance |
| Search latency p99 | > 150ms | > 200ms | Add client nodes or optimize queries |

### Storage Tier Recommendations

| Data Type | Tier | Retention | ILM Policy |
|-----------|------|-----------|------------|
| Hot data | hot nodes (SSD) | 0–7 days | No ILM |
| Warm data | warm nodes | 7–30 days | `shrink` + `forcemerge` |
| Cold data | cold nodes | 30–90 days | `freeze` |
| Archive | OBS (external) | 90+ days | Snapshot to OBS |

## Predictive Alert Rules

```bash
# Disk exhaustion forecast
hcloud ces create-alarm-rule \
  --name "CSS-Disk-Exhaust-Forecast" \
  --metric disk_usage \
  --namespace SYS.CSS \
  --dimension "cluster_id={{user.cluster_id}}" \
  --condition "forecast_linear(168h) > 90%" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"

# JVM heap pressure forecast
hcloud ces create-alarm-rule \
  --name "CSS-JVM-Heap-Pressure" \
  --metric jvm_heap_usage \
  --namespace SYS.CSS \
  --dimension "cluster_id={{user.cluster_id}}" \
  --condition "forecast_arima(24h) > 85%" \
  --alarm-level critical \
  --notifications "topic_arn:{{output.topic_arn}}"

# Query latency growth
hcloud ces create-alarm-rule \
  --name "CSS-Search-Latency-Growth" \
  --metric search_latency \
  --namespace SYS.CSS \
  --dimension "cluster_id={{user.cluster_id}}" \
  --condition "forecast_holt(168h) > 200" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"
```

## Cross-Skill Delegation

| Capacity Issue | Delegate To | Purpose |
|----------------|-------------|---------|
| Storage expansion | CSS skill (resize) | Extend cluster storage |
| JVM heap increase | CSS skill (scale) | Scale up node flavor |
| Shard rebalance | CSS skill (rebalance) | Redistribute shards |
| OBS archival | OBS skill | Cold data tiering |
| Cost from over-provisioning | Billing skill | Right-sizing |

## Knowledge Base Anchors

- CSS ↔ CES: [`references/advanced/aiops-best-practices.md`](./aiops-best-practices.md) — anomaly patterns
- CSS ↔ OBS: [`references/observability.md`](./observability.md) — snapshot and archival
- Cluster management: [`references/troubleshooting.md`](../../huaweicloud-css-ops/references/troubleshooting.md)
