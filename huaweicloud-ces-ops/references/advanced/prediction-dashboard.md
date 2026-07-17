# Prediction Dashboard — L5 Predictive Maintenance

> **Purpose**: Visualization of predictions, accuracy metrics, and capacity forecasts.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Dashboard Components

### 1.1 Resource Health Overview

```
┌─────────────────────────────────────────────────────────────┐
│  Resource Health Predictions                    [Refresh ▼] │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐           │
│  │  ECS    │ │  RDS    │ │  CCE    │ │  ELB    │           │
│  │  12 OK  │ │  3 OK   │ │  5 OK   │ │  2 OK   │           │
│  │  2 ⚠️   │ │  1 ⚠️   │ │  1 ⚠️   │ │  0 ⚠️   │           │
│  │  0 🔴   │ │  0 🔴   │ │  0 🔴   │ │  1 🔴   │           │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘           │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Forecast Chart

```
┌─────────────────────────────────────────────────────────────┐
│  CPU Utilization Forecast — ecs-production-01               │
├─────────────────────────────────────────────────────────────┤
│  100% ─ ┤                              ╭──╮                  │
│   90% ─ ┤                    ╭────────╯  ╰──╮               │
│   80% ─ ┤          ╭────────╯              ╰──               │
│   70% ─ ┤ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─   │
│   60% ─ ┤                                                 │
│        7/11   7/12   7/13   7/14   7/15   7/16   7/17       │
│        Historical          Forecast (85% CI)               │
├─────────────────────────────────────────────────────────────┤
│  Predicted threshold exceeded: 2026-07-14 14:00 (85.3%)    │
│  Recommended action: Scale up (ECS-A02)                    │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 Model Accuracy Metrics

| Model | Accuracy (R²) | MAE | MAPE | Predictions Made |
|-------|--------------|-----|------|------------------|
| Linear Regression | 0.87 | 3.2% | 4.1% | 1,245 |
| Seasonal | 0.92 | 2.8% | 3.5% | 892 |
| Exponential Smoothing | 0.85 | 3.5% | 4.8% | 2,103 |
| Anomaly Detection | 0.89 | — | — | 456 |

### 1.4 Capacity Planning View

```
┌─────────────────────────────────────────────────────────────┐
│  Capacity Forecast — 30 Day Projection                      │
├─────────────────────────────────────────────────────────────┤
│  Resource     Current    30-Day    Trend      Alert        │
│  ─────────────────────────────────────────────────────────  │
│  ECS (CPU)    65%        82%       +17%       ⚠️ High      │
│  RDS (Disk)   70%        88%       +18%       🔴 Critical  │
│  ELB (Conn)   45%        52%       +7%        ✅ OK        │
│  CCE (Pods)   60%        75%       +15%       ⚠️ Monitor   │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Dashboard Data Sources

| Panel | Data Source | Refresh |
|-------|-------------|---------|
| Resource Health | CES metrics + prediction results | 5 min |
| Forecast Chart | prediction_results table | 15 min |
| Model Accuracy | prediction_results + accuracy tracking | 1 hour |
| Capacity Planning | Aggregated forecasts | 1 hour |

---

## 3. Export Capabilities

### 3.1 PDF Report

```python
def generate_capacity_report(date_range, resource_filter=None):
    """
    Generate PDF capacity forecast report.
    """
    forecasts = get_all_forecasts(date_range, resource_filter)

    report = {
        "title": "Capacity Forecast Report",
        "date_range": date_range,
        "generated_at": current_timestamp(),
        "resources": []
    }

    for resource_id, forecast in forecasts.items():
        report["resources"].append({
            "resource_id": resource_id,
            "current_usage": get_current_usage(resource_id),
            "forecast_30d": forecast.values[-1],
            "growth_rate": calculate_growth_rate(forecast),
            "threshold_date": find_threshold_exceed_date(forecast),
            "recommendations": generate_recommendations(forecast)
        })

    return render_pdf("capacity_report.html", report)
```

---

## 4. Compliance Checklist

- [ ] Resource health overview panel
- [ ] Forecast chart with confidence intervals
- [ ] Model accuracy metrics table
- [ ] Capacity planning view
- [ ] Export to PDF capability
