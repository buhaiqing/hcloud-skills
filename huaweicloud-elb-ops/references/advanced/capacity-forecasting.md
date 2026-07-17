# Capacity Forecasting — Huawei Cloud ELB

> Predictive capacity planning for ELB: bandwidth utilization, concurrent connections, and
> backend server utilization forecasts with linear/ARIMA methods and automated alert wiring.
> **Version:** 1.0.0

## Forecast Types

| Forecast Type | Method | Prediction Window | Input Data | Accuracy Target |
|---------------|--------|-------------------|------------|-----------------|
| Bandwidth utilization | Linear regression on traffic_bytes | 24-72h before 85% | 7d in/out_tx_bytes | ±10% |
| Concurrent connections | ARIMA forecast | 48h before limit | 30d current_connections | ±15% |
| Backend server utilization | Holt-Winters exponential smoothing | 7d before saturation | 30d backend_server_rx/tx_bytes | ±20% |
| New connection rate | Linear regression on new_connection_rate | 24h before capacity | 7d new_connection_rate | ±15% |

## Data Acquisition

### CES Metrics Query

```bash
# Bandwidth utilization time-series (7d, 5-min granularity)
hcloud ces list-metric-data \
  --namespace SYS.ELB \
  --metric_name in_tx_bytes \
  --dimension "loadbalancer_id={{user.loadbalancer_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 300 \
  -o json | jq '.datapoints[] | {timestamp, value}'

# Outbound traffic
hcloud ces list-metric-data \
  --namespace SYS.ELB \
  --metric_name out_tx_bytes \
  --dimension "loadbalancer_id={{user.loadbalancer_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 300 \
  -o json | jq '.datapoints[] | {timestamp, value}'

# Backend server RX bytes
hcloud ces list-metric-data \
  --namespace SYS.ELB \
  --metric_name backend_server_rx_bytes \
  --dimension "loadbalancer_id={{user.loadbalancer_id}}" \
  --from "$(date -d '30 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 3600 \
  -o json

# Backend server TX bytes
hcloud ces list-metric-data \
  --namespace SYS.ELB \
  --metric_name backend_server_tx_bytes \
  --dimension "loadbalancer_id={{user.loadbalancer_id}}" \
  --from "$(date -d '30 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 3600 \
  -o json

# New connection rate
hcloud ces list-metric-data \
  --namespace SYS.ELB \
  --metric_name new_connection_rate \
  --dimension "loadbalancer_id={{user.loadbalancer_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 300 \
  -o json
```

## Forecast Algorithms

### Linear Regression (Bandwidth / Connection Rate)

```python
from datetime import datetime, timedelta

def linear_forecast(data_points, days_ahead=30):
    """
    Simple linear regression on bandwidth/connection usage.
    data_points: list of {"timestamp": ms, "value": bytes or count}
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
    days_to_threshold = (threshold - data_points[-1]["value"]) / slope if slope > 0 else float("inf")

    return {
        "current": data_points[-1]["value"],
        "slope_per_day": slope,
        "projected_in_30d": projected,
        "days_to_threshold": days_to_threshold,
        "exhaustion_date": datetime.now() + timedelta(days=days_to_threshold) if days_to_threshold < float("inf") else None,
    }
```

### ARIMA (Concurrent Connections)

```python
import subprocess, json

def arima_forecast(conn_series, forecast_hours=48):
    """
    Wrapper around statsmodels ARIMA.
    conn_series: list of floats (concurrent connections)
    """
    import numpy as np
    try:
        from statsmodels.tsa.arima.model import ARIMA
    except ImportError:
        return {"error": "statsmodels not available, falling back to linear"}

    model = ARIMA(conn_series, order=(2, 1, 2))
    fitted = model.fit()
    forecast = fitted.forecast(steps=forecast_hours)

    return {
        "current": conn_series[-1],
        "forecast_48h": float(forecast[-1]),
        "will_saturate_by_48h": forecast[-1] > 0.85 * max_capacity,
        "trend": "rising" if forecast[-1] > conn_series[-1] else "stable",
    }
```

### Holt-Winters (Backend Utilization)

```python
def holt_winters_forecast(backend_series, periods=30):
    """
    Triple exponential smoothing for backend server utilization forecast.
    backend_series: list of utilization percentages (0-100)
    """
    try:
        from statsmodels.tsa.holtwinters import ExponentialSmoothing
    except ImportError:
        return {"error": "statsmodels not available"}

    model = ExponentialSmoothing(
        backend_series,
        trend="add",
        seasonal="add",
        seasonal_periods=288,  # 5-min data, 24h cycle
    )
    fitted = model.fit()
    forecast = fitted.forecast(periods)

    return {
        "current": backend_series[-1],
        "forecast_30d_max": float(max(forecast)),
        "forecast_30d_avg": float(sum(forecast) / len(forecast)),
        "saturates_by_7d": max(forecast[:2016]) > 85,  # 7d at 5-min = 2016 points
    }
```

## Predictive Alert Rules

### CES Alarm Rules

```bash
# Bandwidth utilization forecast (linear extrapolation)
hcloud ces create-alarm-rule \
  --name "ELB-Bandwidth-Forecast" \
  --metric in_tx_bytes \
  --namespace SYS.ELB \
  --condition "forecast_linear(24h) > 85%" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"

# Concurrent connections forecast
hcloud ces create-alarm-rule \
  --name "ELB-Connections-Forecast" \
  --metric current_connections \
  --namespace SYS.ELB \
  --condition "forecast_arima(48h) > 85%" \
  --alarm-level critical \
  --notifications "topic_arn:{{output.topic_arn}}"

# Backend server utilization forecast
hcloud ces create-alarm-rule \
  --name "ELB-Backend-Utilization-Forecast" \
  --metric backend_server_rx_bytes \
  --namespace SYS.ELB \
  --condition "forecast_holt(168h) > 85%" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"
```

## Capacity Thresholds

| Metric | Warning | Critical | Unit |
|--------|---------|----------|------|
| Bandwidth utilization | 70% | 85% | % of limit |
| Concurrent connections | 70% | 85% | % of limit |
| Backend server utilization | 70% | 85% | % of capacity |
| New connection rate | 70% | 85% | % of limit |

## Capacity Planning Tables

### ELB Right-Sizing by Forecast

| Forecast Result | Action | Command |
|-----------------|--------|---------|
| Bandwidth: days_to_85% < 7 | Upgrade bandwidth or enable compression | `hcloud elb update-loadbalancer --bandwidth` |
| Connections: forecast_48h > 85% | Scale out or upgrade规格 | `hcloud as create-scaling-policy` |
| Backend utilization > 85% for 7d | Add backend servers | `hcloud elb add-member` |
| New connection rate spike | Review connection persistence settings | `hcloud elb update-listener` |

### ELB Specification Selection Guide

| Traffic Pattern | Current Spec | Recommended Action |
|-----------------|--------------|-------------------|
| Bandwidth forecast > 70% for 7d | 100Mbps | Upgrade to 500Mbps or shared bandwidth |
| Connections > 80% of limit | Basic edition | Upgrade to performance-optimized |
| Backend RX > 85% consistently | Single AZ | Enable multi-AZ distribution |
| Connection rate > 10k/s | Standard config | Enable connection multiplexing |

---

## Cross-Skill Delegation

| Capacity Issue | Delegate To | Purpose |
|----------------|-------------|---------|
| Bandwidth upgrade needed | ELB skill (update) | Increase bandwidth allocation |
| Backend scale-out needed | ECS skill (scale) | Add backend instances |
| Auto-scaling needed | AS skill | Configure scale-out policy |
| Multi-AZ distribution | VPC skill (subnet) | Configure AZ distribution |
| Connection persistence tuning | ELB skill (listener) | Adjust session persistence |

## Knowledge Base Anchors

- ELB ↔ CES: [`references/monitoring.md`](../../huaweicloud-ces-ops/references/monitoring.md) — metric aggregation and alarm rules
- ELB ↔ VPC: [`references/integration.md`](../../huaweicloud-elb-ops/references/integration.md) — subnet and AZ configuration
- ELB ↔ ECS: [`references/integration.md`](../../huaweicloud-elb-ops/references/integration.md) — backend server management
- Capacity forecast CLI patterns: [`references/cli-usage.md`](../../huaweicloud-elb-ops/references/cli-usage.md)
