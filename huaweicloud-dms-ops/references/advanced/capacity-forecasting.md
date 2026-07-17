# Capacity Forecasting — Huawei Cloud DMS

> Predictive capacity planning for DMS queues: queue depth, message backlog, and consumer capacity.
> **Version:** 1.0.0

## Forecast Types

| Forecast Type | Method | Prediction Window | Input Data | Accuracy Target |
|---------------|--------|-------------------|------------|-----------------|
| Queue depth exhaustion | Linear regression on queueDepth | 24–72h before 80% | 7d queue_depth time-series | ±10% |
| Message backlog accumulation | Linear regression on messageAccumulates | 48h before threshold | 7d backlog time-series | ±15% |
| Consumer capacity saturation | Linear regression on consumerCount | 24h before 90% | 7d consumer time-series | ±10% |
| Storage capacity | Linear regression on storageUsed | 48h before 85% | 7d storage time-series | ±15% |
| Produce rate spike | Holt-Winters exponential smoothing | 7d before capacity | 30d produce_rate | ±20% |

## Data Acquisition

### CES Metrics Query

```bash
# Queue depth time-series (7d, 5-min granularity)
hcloud ces list-metric-data \
  --namespace SYS.DMS \
  --metric_name queueDepth \
  --dimension "queue_id={{user.queue_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 300 \
  -o json | jq '.datapoints[] | {timestamp, value}'

# Message backlog time-series (7d)
hcloud ces list-metric-data \
  --namespace SYS.DMS \
  --metric_name messageAccumulates \
  --dimension "queue_id={{user.queue_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 300 \
  -o json

# Consumer count time-series (7d)
hcloud ces list-metric-data \
  --namespace SYS.DMS \
  --metric_name consumerCount \
  --dimension "queue_id={{user.queue_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 300 \
  -o json

# Storage usage time-series (7d)
hcloud ces list-metric-data \
  --namespace SYS.DMS \
  --metric_name storageUsed \
  --dimension "queue_id={{user.queue_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 300 \
  -o json

# Produce rate time-series (30d)
hcloud ces list-metric-data \
  --namespace SYS.DMS \
  --metric_name produce_rate \
  --dimension "queue_id={{user.queue_id}}" \
  --from "$(date -d '30 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 3600 \
  -o json
```

## Forecast Algorithms

### Linear Regression (Queue Depth / Backlog / Consumer / Storage)

```python
from datetime import datetime, timedelta

def linear_forecast(data_points, days_ahead=30, threshold_pct=80):
    """
    Simple linear regression on DMS queue metrics.
    data_points: list of {"timestamp": ms, "value": percent or count}
    """
    n = len(data_points)
    x = [(p["timestamp"] - data_points[0]["timestamp"]) / 86400000 for p in data_points]
    y = [p["value"] for p in data_points]

    x_mean = sum(x) / n
    y_mean = sum(y) / n

    slope = sum((xi - x_mean) * (yi - y_mean) for xi, yi in zip(x, y)) / \
            sum((xi - x_mean) ** 2 for xi in x)
    intercept = y_mean - slope * x_mean

    projected = slope * (x[-1] + days_ahead) + intercept
    days_to_threshold = (threshold_pct - data_points[-1]["value"]) / slope if slope > 0 else float("inf")

    return {
        "current": data_points[-1]["value"],
        "slope_per_day": slope,
        "projected_in_30d": projected,
        "days_to_threshold": days_to_threshold,
        "exhaustion_date": datetime.now() + timedelta(days=days_to_threshold) if days_to_threshold < float("inf") else None,
    }
```

### Holt-Winters (Produce Rate)

```python
def holt_winters_forecast(produce_series, periods=30):
    """
    Triple exponential smoothing for produce rate forecast.
    produce_series: list of produce rate values
    """
    try:
        from statsmodels.tsa.holtwinters import ExponentialSmoothing
    except ImportError:
        return {"error": "statsmodels not available"}

    model = ExponentialSmoothing(
        produce_series,
        trend="add",
        seasonal="add",
        seasonal_periods=288,
    )
    fitted = model.fit()
    forecast = fitted.forecast(periods)

    return {
        "current": produce_series[-1],
        "forecast_30d_max": float(max(forecast)),
        "forecast_30d_avg": float(sum(forecast) / len(forecast)),
        "saturates_by_7d": max(forecast[:2016]) > 0.9 * max(produce_series),
    }
```

## Predictive Alert Rules

### CES Alarm Rules

```bash
# Queue depth forecast (linear)
hcloud ces create-alarm-rule \
  --name "DMS-QueueDepth-Forecast" \
  --metric queueDepth \
  --namespace SYS.DMS \
  --condition "forecast_linear(24h) > 80%" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"

# Message backlog forecast (linear)
hcloud ces create-alarm-rule \
  --name "DMS-Backlog-Forecast" \
  --metric messageAccumulates \
  --namespace SYS.DMS \
  --condition "forecast_linear(48h) > 100000" \
  --alarm-level critical \
  --notifications "topic_arn:{{output.topic_arn}}"

# Consumer capacity forecast (linear)
hcloud ces create-alarm-rule \
  --name "DMS-Consumer-Forecast" \
  --metric consumerCount \
  --namespace SYS.DMS \
  --condition "forecast_linear(24h) > 90%" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"

# Storage capacity forecast (linear)
hcloud ces create-alarm-rule \
  --name "DMS-Storage-Forecast" \
  --metric storageUsed \
  --namespace SYS.DMS \
  --condition "forecast_linear(48h) > 85%" \
  --alarm-level critical \
  --notifications "topic_arn:{{output.topic_arn}}"

# Produce rate spike forecast
hcloud ces create-alarm-rule \
  --name "DMS-ProduceRate-Forecast" \
  --metric produce_rate \
  --namespace SYS.DMS \
  --condition "forecast_holt(168h) > 0.9 * max_produce_rate" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"
```

## Capacity Planning Tables

### DMS Queue Right-Sizing by Forecast

| Forecast Result | Action | Command |
|-----------------|--------|---------|
| Queue depth: days_to_80 < 7 | Increase queue partitions | `hcloud dms update-queue --partitions +N` |
| Backlog: days_to_threshold < 2 | Scale consumers or increase partitions | `hcloud dms scale-consumer` |
| Consumer: days_to_90 < 1 | Add consumer instances | `hcloud dms create-consumer-group` |
| Storage: days_to_85 < 7 | Expand storage or enable auto-cleanup | `hcloud dms update-queue --retention N` |
| Produce rate spike | Increase throughput quota or partitions | `hcloud dms update-queue --partitions +N` |

### Queue Configuration Guide

| Queue Pressure | Current Config | Recommended Action |
|----------------|----------------|-------------------|
| Depth > 80% for 7d | Standard queue | 增加分区数 (partitions) |
| Consumer lag > threshold | 单 Consumer Group | 增加消费者实例 |
| Storage > 85% | 默认保留 72h | 降低保留时间或启用自动清理 |
| Produce rate spike | 单分区 | 增加分区数并重新分区 |
| Cross-region DR | 单区域 | 启用跨区域复制 |

## Cross-Skill Delegation

| Capacity Issue | Delegate To | Purpose |
|----------------|-------------|---------|
| Queue partition increase | DMS skill (update) | Expand queue capacity |
| Consumer group scaling | DMS skill (scale) | Add consumers |
| Storage retention policy | DMS skill (update) | Configure retention |
| Cross-region replication | VPC/ELB skill | DR configuration |
| Throughput quota increase | Billing skill | Quota request |

## Knowledge Base Anchors

- DMS ↔ CES: [`references/monitoring.md`](../../huaweicloud-ces-ops/references/monitoring.md) — metric aggregation and alarm rules
- DMS ↔ ECS: [`references/integration.md`](../../huaweicloud-dms-ops/references/integration.md) — consumer application integration
- Capacity forecast CLI patterns: [`references/cli-usage.md`](../../huaweicloud-dms-ops/references/cli-usage.md)
