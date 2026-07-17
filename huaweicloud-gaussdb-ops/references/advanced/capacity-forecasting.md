# Capacity Forecasting — Huawei Cloud GaussDB

> Predictive capacity planning for GaussDB instances: connections, storage, transaction logs.
> **Version:** 1.0.0

## Forecast Types

| Forecast Type | Method | Prediction Window | Input Data | Accuracy Target |
|---------------|--------|-------------------|------------|-----------------|
| Connection capacity exhaustion | Linear regression on activeConnections | 24–72h before 80% | 7d active_connections time-series | ±10% |
| Storage capacity exhaustion | Linear regression on diskUsage | 24–72h before 90% | 7d storage usage time-series | ±10% |
| Transaction log capacity | Linear regression on transLogUsage | 48h before 85% | 7d transaction log time-series | ±15% |
| CPU saturation | Holt-Winters exponential smoothing | 7d before 100% | 30d cpu_util | ±20% |
| Memory exhaustion | ARIMA forecast | 48h before OOM | 30d mem_usedPercent | ±15% |

## Data Acquisition

### CES Metrics Query

```bash
# Connection count time-series (7d, 5-min granularity)
hcloud ces list-metric-data \
  --namespace SYS.GAUSSDB \
  --metric_name activeConnections \
  --dimension "instance_id={{user.instance_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 300 \
  -o json | jq '.datapoints[] | {timestamp, value}'

# Storage usage time-series (7d)
hcloud ces list-metric-data \
  --namespace SYS.GAUSSDB \
  --metric_name diskUsage_percent \
  --dimension "instance_id={{user.instance_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 300 \
  -o json

# Transaction log usage (7d)
hcloud ces list-metric-data \
  --namespace SYS.GAUSSDB \
  --metric_name transLogUsage_percent \
  --dimension "instance_id={{user.instance_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 300 \
  -o json

# Memory usage time-series (30d)
hcloud ces list-metric-data \
  --namespace SYS.GAUSSDB \
  --metric_name mem_usedPercent \
  --dimension "instance_id={{user.instance_id}}" \
  --from "$(date -d '30 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 3600 \
  -o json
```

## Forecast Algorithms

### Linear Regression (Connections / Storage / TransLog)

```python
from datetime import datetime, timedelta

def linear_forecast(data_points, days_ahead=30, threshold_pct=80):
    """
    Simple linear regression on usage metrics.
    data_points: list of {"timestamp": ms, "value": percent}
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

### ARIMA (Memory)

```python
def arima_forecast(mem_series, forecast_hours=48):
    try:
        from statsmodels.tsa.arima.model import ARIMA
    except ImportError:
        return {"error": "statsmodels not available, falling back to linear"}

    model = ARIMA(mem_series, order=(2, 1, 2))
    fitted = model.fit()
    forecast = fitted.forecast(steps=forecast_hours)

    return {
        "current": mem_series[-1],
        "forecast_48h": float(forecast[-1]),
        "will_oom_by_48h": forecast[-1] > 95,
        "trend": "rising" if forecast[-1] > mem_series[-1] else "stable",
    }
```

### Holt-Winters (CPU)

```python
def holt_winters_forecast(cpu_series, periods=30):
    try:
        from statsmodels.tsa.holtwinters import ExponentialSmoothing
    except ImportError:
        return {"error": "statsmodels not available"}

    model = ExponentialSmoothing(
        cpu_series,
        trend="add",
        seasonal="add",
        seasonal_periods=288,
    )
    fitted = model.fit()
    forecast = fitted.forecast(periods)

    return {
        "current": cpu_series[-1],
        "forecast_30d_max": float(max(forecast)),
        "forecast_30d_avg": float(sum(forecast) / len(forecast)),
        "saturates_by_7d": max(forecast[:2016]) > 95,
    }
```

## Predictive Alert Rules

### CES Alarm Rules

```bash
# Connection capacity forecast (linear)
hcloud ces create-alarm-rule \
  --name "GaussDB-Connection-Forecast" \
  --metric activeConnections \
  --namespace SYS.GAUSSDB \
  --condition "forecast_linear(24h) > 80%" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"

# Storage fill forecast (linear)
hcloud ces create-alarm-rule \
  --name "GaussDB-Storage-Forecast" \
  --metric diskUsage_percent \
  --namespace SYS.GAUSSDB \
  --condition "forecast_linear(24h) > 90%" \
  --alarm-level critical \
  --notifications "topic_arn:{{output.topic_arn}}"

# Transaction log forecast (linear)
hcloud ces create-alarm-rule \
  --name "GaussDB-TransLog-Forecast" \
  --metric transLogUsage_percent \
  --namespace SYS.GAUSSDB \
  --condition "forecast_linear(48h) > 85%" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"

# Memory exhaustion forecast
hcloud ces create-alarm-rule \
  --name "GaussDB-Memory-Exhaust-Forecast" \
  --metric mem_usedPercent \
  --namespace SYS.GAUSSDB \
  --condition "forecast_arima(48h) > 95%" \
  --alarm-level critical \
  --notifications "topic_arn:{{output.topic_arn}}"
```

## Capacity Planning Tables

### GaussDB Instance Right-Sizing by Forecast

| Forecast Result | Action | Command |
|-----------------|--------|---------|
| Connections: days_to_80 < 7 | Increase max_connections | GaussDB parameter group update |
| Storage: days_to_90 < 7 | Expand storage immediately | `hcloud gaussdb resize-storage` |
| TransLog: days_to_85 < 7 | Expand log storage or archive | `hcloud gaussdb modify-trans-log` |
| Memory: forecast_48h > 95% | Upgrade instance规格 | `hcloud gaussdb resize-instance` |
| CPU: saturates_by_7d | Enable auto-scaling or upgrade | AS skill |

### Instance Type Selection Guide

| Resource Pressure | Current Flavor | Recommended Action |
|-------------------|----------------|-------------------|
| CPU forecast > 80% for 7d |通用型 | 切换至计算优化型 |
| Memory forecast > 85% for 7d |通用型 | 切换至内存优化型 |
| Connections forecast > 80% |基础版 | 切换至集群版 |
| Storage forecast > 90% |— | 扩容存储或清理冷数据 |

## Cross-Skill Delegation

| Capacity Issue | Delegate To | Purpose |
|----------------|-------------|---------|
| Storage expansion | GaussDB skill (resize) | Expand disk |
| Instance type change | GaussDB skill (resize) | Flavor upgrade |
| Auto-scaling needed | AS skill | Scale-out policy |
| Transaction log full | GaussDB skill (archive) | Log archival configuration |
| Connection pool tuning | GaussDB skill (parameter) | max_connections adjustment |

## Knowledge Base Anchors

- GaussDB ↔ CES: [`references/monitoring.md`](../../huaweicloud-ces-ops/references/monitoring.md) — metric aggregation and alarm rules
- GaussDB ↔ Backup: [`references/backup.md`](../../huaweicloud-cbr-ops/references/backup.md) — backup and restore integration
- Capacity forecast CLI patterns: [`references/cli-usage.md`](../../huaweicloud-gaussdb-ops/references/cli-usage.md)
