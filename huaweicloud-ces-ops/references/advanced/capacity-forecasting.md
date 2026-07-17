# Capacity Forecasting — Huawei Cloud CES

> Predictive capacity planning for Cloud Eye Service: monitoring quota, alarm rules, and API rate limits.
> **Version:** 1.0.0

## Forecast Types

| Forecast Type | Method | Prediction Window | Input Data | Accuracy Target |
|---------------|--------|-------------------|------------|-----------------|
| Monitoring quota usage | Linear regression on quota utilization | 7-30d before 80% | CES quota API response | ±10% |
| Alarm rule count forecast | Trend analysis on rule creation rate | 30d before quota exhausted | Historical rule creation rate | ±15% |
| API call quota prediction | ARIMA forecast | 24-72h before throttling | CES API call time-series | ±20% |
| Notification endpoint scaling | Capacity modeling | 7d before limit | SMN topic subscription count | ±25% |

## Data Acquisition

### CES Quota Query

```bash
# Query CES monitoring quota usage
hcloud ces list-quota-resources \
  --region "{{env.HW_REGION_ID}}" \
  -o json | jq '.quotas[] | {type: .type, used: .used, quota: .quota, unit: .unit}'

# Alarm rules quota status
hcloud ces list-alarm-rules \
  --region "{{env.HW_REGION_ID}}" \
  --output json | jq 'length'
```

### CES Metrics Query for Capacity Planning

```bash
# Query CES API call count (SYS.CES namespace)
hcloud ces list-metric-data \
  --namespace SYS.CES \
  --metric_name api_request_count \
  --dimension "project_id={{env.HW_PROJECT_ID}}" \
  --from "$(date -d '30 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 3600 \
  -o json | jq '.datapoints[] | {timestamp, value}'

# Query alarm rule count time-series (custom metric)
hcloud ces list-metric-data \
  --namespace SYS.CES \
  --metric_name alarm_rule_count \
  --dimension "project_id={{env.HW_PROJECT_ID}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 86400 \
  -o json

# Query notification API calls
hcloud ces list-metric-data \
  --namespace SYS.CES \
  --metric_name notification_invoke_count \
  --dimension "project_id={{env.HW_PROJECT_ID}}" \
  --from "$(date -d '30 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 3600 \
  -o json
```

### Quota Limit Query

```bash
# List all CES resource quotas
hcloud ces list-quota-resources \
  --region "{{env.HW_REGION_ID}}" \
  --output json

# List specific quota for alarm rules
hcloud ces list-quota-resources \
  --region "{{env.HW_REGION_ID}}" \
  --quota_type alarm_rule \
  -o json
```

## Forecast Algorithms

### Linear Regression (Quota Utilization)

```python
from datetime import datetime, timedelta

def linear_forecast(data_points, days_ahead=30):
    """
    Simple linear regression on quota utilization.
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
    days_to_80 = (80 - data_points[-1]["value"]) / slope if slope > 0 else float("inf")

    return {
        "current": data_points[-1]["value"],
        "slope_per_day": slope,
        "projected_in_30d": projected,
        "days_to_80pct": days_to_80,
        "exhaustion_date": datetime.now() + timedelta(days=days_to_80) if days_to_80 < float("inf") else None,
    }
```

### ARIMA (API Call Rate)

```python
import subprocess, json

def arima_forecast(api_call_series, forecast_hours=48):
    """
    Wrapper around statsmodels ARIMA for API call rate prediction.
    api_call_series: list of floats (API calls per hour)
    """
    import numpy as np
    try:
        from statsmodels.tsa.arima.model import ARIMA
    except ImportError:
        return {"error": "statsmodels not available, falling back to linear"}

    model = ARIMA(api_call_series, order=(2, 1, 2))
    fitted = model.fit()
    forecast = fitted.forecast(steps=forecast_hours)

    return {
        "current": api_call_series[-1],
        "forecast_48h": float(forecast[-1]),
        "will_throttle_by_48h": forecast[-1] > 0.9 * 1000000,  # Example throttle limit
        "trend": "rising" if forecast[-1] > api_call_series[-1] else "stable",
    }
```

### Trend Analysis (Alarm Rule Count)

```python
def alarm_rule_trend_forecast(rule_counts, forecast_days=30):
    """
    Forecast alarm rule count based on creation rate.
    rule_counts: list of daily rule counts
    """
    if len(rule_counts) < 7:
        return {"error": "Insufficient data for trend analysis"}

    # Calculate 7-day moving average
    recent = rule_counts[-7:]
    avg_daily_creation = (recent[-1] - recent[0]) / len(recent)

    projected_count = recent[-1] + avg_daily_creation * forecast_days

    return {
        "current_count": recent[-1],
        "avg_daily_creation": avg_daily_creation,
        "projected_count_30d": projected_count,
        "quota_exhaustion_risk": "high" if projected_count > 1000 else "low",
    }
```

## Predictive Alert Rules

### CES Capacity Alarm Rules

```bash
# Monitoring quota usage forecast
hcloud ces create-alarm-rule \
  --name "CES-Quota-Usage-Forecast" \
  --metric quota_usage_percent \
  --namespace SYS.CES \
  --dimension "project_id={{env.HW_PROJECT_ID}}" \
  --condition "forecast_linear(168h) > 70%" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"

# Alarm rule count forecast
hcloud ces create-alarm-rule \
  --name "CES-AlarmRule-Count-Forecast" \
  --metric alarm_rule_count \
  --namespace SYS.CES \
  --dimension "project_id={{env.HW_PROJECT_ID}}" \
  --condition "forecast_linear(720h) > 850" \
  --alarm-level critical \
  --notifications "topic_arn:{{output.topic_arn}}"

# API call rate forecast
hcloud ces create-alarm-rule \
  --name "CES-API-Rate-Forecast" \
  --metric api_request_count \
  --namespace SYS.CES \
  --dimension "project_id={{env.HW_PROJECT_ID}}" \
  --condition "forecast_arima(48h) > 900000" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"
```

### Capacity Threshold Reference

| Resource | Warning Threshold | Critical Threshold | Quota Limit |
|----------|-------------------|--------------------|-------------|
| Alarm Rules | 70% (700) | 85% (850) | 1000 |
| API Calls | 70% of throttle | 85% of throttle | Rate limited |
| Custom Metrics | 70% (7000) | 85% (8500) | 10000 |
| Notification Endpoints | 70% (70) | 85% (85) | 100 |

## Capacity Planning Tables

### CES Right-Sizing by Forecast

| Forecast Result | Action | Command |
|-----------------|--------|---------|
| Quota usage: days_to_80 < 7 | Request quota increase or clean up | `hcloud ces list-alarm-rules --output json` |
| Alarm rules: projected > 850 | Delete unused rules or request increase | `hcloud ces delete-alarm-rule --alarm_id` |
| API rate: forecast_48h > 85% | Implement rate limiting or cache | Reduce polling frequency |
| Notification: endpoint_count > 85 | Clean up SMN subscriptions | `hcloud smn list-subscriptions` |

### Quota Increase Request Guide

| Quota Type | Default Limit | Max Limit | Request Via |
|------------|---------------|-----------|-------------|
| Alarm Rules per project | 1000 | 5000 | Support ticket |
| Custom Metrics per project | 10000 | 50000 | Support ticket |
| API requests per second | 1000 | 5000 | Support ticket |
| Notification endpoints per topic | 100 | 500 | Support ticket |

## Cross-Skill Delegation

| Capacity Issue | Delegate To | Purpose |
|----------------|-------------|---------|
| Alarm rule cleanup needed | CES skill (delete) | Remove unused rules |
| Quota increase required | BSS skill (quota request) | Submit quota request |
| SMN endpoint cleanup | SMN skill (subscription management) | Clean up notification endpoints |
| API rate limiting needed | API Gateway skill | Implement rate limiting |
| Data retention optimization | LTS skill (log management) | Reduce log storage volume |

## Knowledge Base Anchors

- CES ↔ BSS: [`references/integration.md`](../integration.md) — quota management and billing
- CES ↔ SMN: [`references/integration.md`](../integration.md) — notification channel integration
- CES ↔ LTS: [`references/integration.md`](../integration.md) — log-based metrics
- Capacity forecast CLI patterns: [`references/cli-usage.md`](../cli-usage.md)
