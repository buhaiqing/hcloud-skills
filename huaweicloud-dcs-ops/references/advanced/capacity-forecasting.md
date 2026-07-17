# Capacity Forecasting — Huawei Cloud DCS

> Predictive capacity planning for Distributed Cache Service (Redis-compatible):
> memory exhaustion, connection saturation, hit rate degradation, and
> cluster shard rebalancing forecasts.
> **Version:** 1.0.0

## Forecast Types

| Forecast Type | Method | Prediction Window | Input Data | Accuracy Target |
|---------------|--------|-------------------|------------|-----------------|
| Memory exhaustion | Linear regression on memory usage | 24–72h before 90% | 7d memory_used / maxmemory | ±10% |
| Connection saturation | Trend on connected_clients / maxclients | 12h before 90% | 7d connection ratio | ±15% |
| Hit rate degradation | Moving average on hit_rate | 1–4h before < 80% | 1h hit_rate | ±20% |
| Cluster slot rebalancing | Shard key distribution analysis | Before 70/30 skew | Real-time key sampling | — |

## Data Acquisition

### DCS Instance Metrics

```bash
# Memory usage ratio (used_memory / maxmemory)
hcloud ces list-metric-data \
  --namespace SYS.DCS \
  --metric_name memory_usage_ratio \
  --dimension "instance_id={{user.instance_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 3600 \
  -o json

# Connected clients
hcloud ces list-metric-data \
  --namespace SYS.DCS \
  --metric_name connected_clients \
  --dimension "instance_id={{user.instance_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 300 \
  -o json

# Hit rate
hcloud ces list-metric-data \
  --namespace SYS.DCS \
  --metric_name keyspace_hitrate \
  --dimension "instance_id={{user.instance_id}}" \
  --from "$(date -d '1 hour ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 60 \
  -o json
```

### DCS Instance Info

```bash
# Instance specs and current memory
hcloud dcs list-instances -o json | jq '
  .instances[] | {
    id, name, capacity: .spec.capacity,
    max_memory: .spec.capacity * 1024 * 1024,
    max_clients: .spec.max_clients,
    version: .engine_version
  }
'
```

## Forecast Algorithms

### Memory Exhaustion

```python
def forecast_memory_exhaustion(instance_id, days_ahead=3):
    """
    Linear regression on memory usage ratio to predict exhaustion.
    """
    history = query_ces(
        namespace="SYS.DCS",
        metric="memory_usage_ratio",
        dimensions={"instance_id": instance_id},
        window="7d",
        period=3600,
    )

    values = [p["value"] * 100 for p in history]  # convert to percent
    n = len(values)
    x = list(range(n))
    x_mean, y_mean = sum(x) / n, sum(values) / n

    slope = sum((xi - x_mean) * (yi - y_mean) for xi, yi in zip(x, values)) / \
            sum((xi - x_mean) ** 2 for xi in x)
    intercept = y_mean - slope * x_mean

    # Daily growth rate
    hourly_slope = slope
    daily_growth = hourly_slope * 24
    current = values[-1]
    days_to_90 = (90 - current) / daily_growth if daily_growth > 0 else float("inf")

    return {
        "current_usage_pct": current,
        "daily_growth_rate_pct": daily_growth,
        "projected_in_3d": intercept + slope * (n - 1 + 3 * 24),
        "days_to_90": days_to_90,
        "recommendation": "expand_capacity" if days_to_90 <= 3 else "monitor",
    }
```

### Connection Saturation

```python
def forecast_connection_saturation(instance_id, hours_ahead=12):
    """
    Trend analysis on connected_clients / maxclients ratio.
    """
    history = query_ces(
        namespace="SYS.DCS",
        metric="connected_clients",
        dimensions={"instance_id": instance_id},
        window="7d",
        period=300,
    )

    # Get max_clients from instance spec
    instance_info = query_dcs_instance(instance_id)
    max_clients = instance_info["spec"]["max_clients"]

    ratios = [p["value"] / max_clients * 100 for p in history]
    n = len(ratios)

    last_288 = ratios[-288:] if n >= 288 else ratios  # 24h at 5-min
    x = list(range(len(last_288)))
    x_mean = sum(x) / len(x)
    y_mean = sum(last_288) / len(last_288)
    slope = sum((xi - x_mean) * (yi - y_mean) for xi, yi in zip(x, last_288)) / \
            sum((xi - x_mean) ** 2 for xi in x)

    projected = ratios[-1] + slope * hours_ahead * 12
    projected = min(100, max(0, projected))

    return {
        "current_ratio_pct": ratios[-1],
        "slope_per_5min": slope,
        "projected_in_12h": projected,
        "saturates_by_12h": projected > 90,
        "recommendation": "increase_max_clients" if projected > 80 else "monitor",
    }
```

### Hit Rate Degradation

```python
def forecast_hit_rate_degradation(instance_id, hours_ahead=4):
    """
    Moving average + trend on keyspace hit rate.
    """
    history = query_ces(
        namespace="SYS.DCS",
        metric="keyspace_hitrate",
        dimensions={"instance_id": instance_id},
        window="1h",
        period=60,
    )

    values = [p["value"] * 100 for p in history]  # percent
    if not values:
        return {"error": "no data"}

    # 5-min moving average
    ma = sum(values[-12:]) / 12 if len(values) >= 12 else sum(values) / len(values)
    trend = values[-1] - values[0] if len(values) >= 2 else 0

    projected = ma + trend * hours_ahead * 12
    projected = min(100, max(0, projected))

    return {
        "current_hit_rate_pct": values[-1],
        "moving_avg_5min_pct": ma,
        "trend_per_min": trend / len(values) if values else 0,
        "projected_in_4h": projected,
        "degrades_below_80pct": projected < 80,
        "recommendation": "investigate_keys" if projected < 80 else "monitor",
    }
```

## Capacity Planning Tables

### DCS Instance Right-Sizing

| Avg Memory (7d) | Avg Connections (7d) | Recommendation | Expected Savings |
|-----------------|----------------------|----------------|------------------|
| < 30% | < 30% | Downgrade flavor | 30–60% |
| < 30% | > 70% | Switch to connection-optimized | 10–20% |
| > 80% | Any | Upgrade or shard | — |
| Spiky (> 3× avg) | — | Burst plan + replication | 20–50% |

### Memory Expansion Tiers

| Current Capacity | Expansion Options | Notes |
|------------------|-------------------|-------|
| 128 MB – 2 GB | +1 GB increments | Entry tier |
| 2 – 16 GB | +4 GB increments | Standard tier |
| 16 – 128 GB | +16 GB increments | Large tier |
| Any (single-slot) | Switch to cluster mode | Horizontal scaling |

## Predictive Alert Rules

```bash
# Memory exhaustion forecast
hcloud ces create-alarm-rule \
  --name "DCS-Memory-Exhaust-Forecast" \
  --metric memory_usage_ratio \
  --namespace SYS.DCS \
  --dimension "instance_id={{user.instance_id}}" \
  --condition "forecast_linear(72h) > 90%" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"

# Connection saturation forecast
hcloud ces create-alarm-rule \
  --name "DCS-Connection-Saturation-Forecast" \
  --metric connected_clients \
  --namespace SYS.DCS \
  --dimension "instance_id={{user.instance_id}}" \
  --condition "forecast_linear(12h) > max_clients * 0.9" \
  --alarm-level critical \
  --notifications "topic_arn:{{output.topic_arn}}"

# Hit rate degradation
hcloud ces create-alarm-rule \
  --name "DCS-HitRate-Degradation" \
  --metric keyspace_hitrate \
  --namespace SYS.DCS \
  --dimension "instance_id={{user.instance_id}}" \
  --condition "ma(5min) < 0.8" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"
```

## Cross-Skill Delegation

| Capacity Issue | Delegate To | Purpose |
|----------------|-------------|---------|
| Memory expansion | DCS skill (resize) | Expand capacity |
| Cluster rebalancing | DCS skill (shard) | Redistribute slots |
| Connection limit increase | DCS skill (parameter) | Update maxclients |
| Cost from over-provisioning | Billing skill | Right-sizing |
| Application-level cache miss | ECS skill (app server) | Investigate cache usage |

## Knowledge Base Anchors

- DCS ↔ ECS: [`references/integration.md`](../../huaweicloud-dcs-ops/references/integration.md) — application caching patterns
- DCS FinOps: [`references/advanced/cost-optimization.md`](./cost-optimization.md) — idle detection, right-sizing
- Cache troubleshooting: [`references/troubleshooting.md`](../../huaweicloud-dcs-ops/references/troubleshooting.md)
