# Capacity Forecasting — Huawei Cloud ECS

> Predictive capacity planning for ECS instances: disk, memory, CPU, and cost
> exhaustion forecasts with linear/ARIMA methods and automated alert wiring.
> **Version:** 1.0.0

## Forecast Types

| Forecast Type | Method | Prediction Window | Input Data | Accuracy Target |
|---------------|--------|-------------------|------------|-----------------|
| Disk capacity exhaustion | Linear regression on diskUsage | 24–72h before 90% | 7d diskUsage time-series | ±10% |
| Memory exhaustion | ARIMA forecast | 48h before OOM | 30d mem_usedPercent | ±15% |
| CPU saturation | Holt-Winters exponential smoothing | 7d before 100% | 30d cpu_util | ±20% |
| Cost spike | Billing trend analysis | Next billing cycle | BSS daily_cost | ±25% |

## Data Acquisition

### CES Metrics Query

```bash
# Disk usage time-series (7d, 5-min granularity)
hcloud ces list-metric-data \
  --namespace SYS.ECS \
  --metric_name diskUsage_percent \
  --dimension "instance_id={{user.instance_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 300 \
  -o json | jq '.datapoints[] | {timestamp, value}'

# Memory usage time-series (30d)
hcloud ces list-metric-data \
  --namespace AGT.ECS \
  --metric_name mem_usedPercent \
  --dimension "instance_id={{user.instance_id}}" \
  --from "$(date -d '30 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 3600 \
  -o json
```

### BSS Cost Query

```bash
# Daily cost trend
hcloud bss list-daily-costs \
  --start_date "$(date -d '30 days ago' +%Y-%m-%d)" \
  --end_date "$(date +%Y-%m-%d)" \
  --product_code "hws.product.ecs" \
  -o json | jq '.costs[] | {date, amount, currency}'
```

## Forecast Algorithms

### Linear Regression (Disk / Storage)

```python
from datetime import datetime, timedelta

def linear_forecast(data_points, days_ahead=30):
    """
    Simple linear regression on disk usage.
    data_points: list of {"timestamp": ms, "value": percent}
    """
    n = len(data_points)
    x = [(p["timestamp"] - data_points[0]["timestamp"]) / 86400000 for p in data_points]  # days
    y = [p["value"] for p in data_points]

    x_mean = sum(x) / n
    y_mean = sum(y) / n

    slope = sum((xi - x_mean) * (yi - y_mean) for xi, yi in zip(x, y)) / \
            sum((xi - x_mean) ** 2 for xi in x)
    intercept = y_mean - slope * x_mean

    projected = slope * (x[-1] + days_ahead) + intercept
    days_to_90 = (90 - data_points[-1]["value"]) / slope if slope > 0 else float("inf")

    return {
        "current": data_points[-1]["value"],
        "slope_per_day": slope,
        "projected_in_30d": projected,
        "days_to_90pct": days_to_90,
        "exhaustion_date": datetime.now() + timedelta(days=days_to_90) if days_to_90 < float("inf") else None,
    }
```

### ARIMA (Memory)

```python
import subprocess, json

def arima_forecast(mem_series, forecast_hours=48):
    """
    Wrapper around statsmodels ARIMA.
    mem_series: list of floats (0–100 percent)
    """
    import numpy as np
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
    """
    Triple exponential smoothing for CPU saturation forecast.
    cpu_series: list of CPU util percentages (0–100)
    """
    try:
        from statsmodels.tsa.holtwinters import ExponentialSmoothing
    except ImportError:
        return {"error": "statsmodels not available"}

    model = ExponentialSmoothing(
        cpu_series,
        trend="add",
        seasonal="add",
        seasonal_periods=288,  # 5-min data, 24h cycle
    )
    fitted = model.fit()
    forecast = fitted.forecast(periods)

    return {
        "current": cpu_series[-1],
        "forecast_30d_max": float(max(forecast)),
        "forecast_30d_avg": float(sum(forecast) / len(forecast)),
        "saturates_by_7d": max(forecast[:2016]) > 95,  # 7d at 5-min = 2016 points
    }
```

## Predictive Alert Rules

### CES Alarm Rules

```bash
# Disk fill forecast (linear extrapolation)
hcloud ces create-alarm-rule \
  --name "ECS-Disk-Fill-Forecast" \
  --metric diskUsage_percent \
  --namespace SYS.ECS \
  --condition "forecast_linear(24h) > 90%" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"

# Memory exhaustion forecast
hcloud ces create-alarm-rule \
  --name "ECS-Memory-Exhaust-Forecast" \
  --metric mem_usedPercent \
  --namespace AGT.ECS \
  --condition "forecast_arima(48h) > 95%" \
  --alarm-level critical \
  --notifications "topic_arn:{{output.topic_arn}}"

# CPU saturation forecast
hcloud ces create-alarm-rule \
  --name "ECS-CPU-Saturation-Forecast" \
  --metric cpu_util \
  --namespace SYS.ECS \
  --condition "forecast_holt(168h) > 95%" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"
```

## Capacity Planning Tables

### ECS Instance Right-Sizing by Forecast

| Forecast Result | Action | Command |
|-----------------|--------|---------|
| Disk: days_to_90 < 7 | Expand disk immediately | `hcloud ecs resize-disk --size +100` |
| Memory: forecast_48h > 95% | Upgrade instance type | `hcloud ecs change-instance-type` |
| CPU: saturates_by_7d | Enable auto-scaling or upgrade | `hcloud as create-scaling-policy` |
| Cost: next_cycle > avg * 1.5 | Create cost alert, audit resource | BSS cost alert |

### Instance Type Selection Guide

| Resource Pressure | Current Flavor | Recommended Action |
|-------------------|----------------|-------------------|
| CPU forecast > 80% for 7d |通用型 | 切换至计算优化型 (c3) |
| Memory forecast > 85% for 7d |通用型 | 切换至内存优化型 (m3) |
| Both CPU+Memory > 80% |— | 切换至大型实例或弹性伸缩 |
| GPU-bound workload |— | 使用GPU加速型 (p2) |

## Cross-Skill Delegation

| Capacity Issue | Delegate To | Purpose |
|----------------|-------------|---------|
| Disk expansion requires new DataDisk | ECS skill (resize) | Expand disk |
| Storage > 90% on Windows | ECS skill (cleanup) | Disk cleanup |
| Instance type change | ECS skill (resize) | Flavor upgrade |
| Auto-scaling needed | AS skill | Scale-out policy |
| Cost spike from reserved instance | Billing skill | Cost anomaly analysis |

## Knowledge Base Anchors

- ECS ↔ CES: [`references/monitoring.md`](../../huaweicloud-ces-ops/references/monitoring.md) — metric aggregation and alarm rules
- ECS ↔ AS: [`references/integration.md`](../../huaweicloud-ecs-ops/references/integration.md) — auto-scaling configuration
- Capacity forecast CLI patterns: [`references/cli-usage.md`](../../huaweicloud-ecs-ops/references/cli-usage.md)
