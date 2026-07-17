# Capacity Forecasting — Huawei Cloud RDS

> Predictive capacity planning for RDS for MySQL/PostgreSQL/SQL Server:
> storage exhaustion, connection saturation, replica lag growth, and
> backup capacity forecasting.
> **Version:** 1.0.0

## Forecast Types

| Forecast Type | Method | Prediction Window | Input Data | Accuracy Target |
|---------------|--------|-------------------|------------|-----------------|
| Storage exhaustion | Linear regression on disk_usage | 7d before 90% | 30d disk_usage | ±10% |
| Connection saturation | Trend on used_connections / max_connections | 24h before 90% | 7d connection ratio | ±15% |
| Replica lag growth | Linear extrapolation on replica_lag | 1–4h before lag > 60s | 2h replica_lag | ±20% |
| Backup volume growth | Linear regression on backup size | 30d before storage 90% | 90d backup size | ±20% |

## Data Acquisition

### RDS Instance Metrics

```bash
# Disk usage time-series (30d)
hcloud ces list-metric-data \
  --namespace SYS.RDS \
  --metric_name rds039_disk_usage \
  --dimension "instance_id={{user.instance_id}}" \
  --from "$(date -d '30 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 3600 \
  -o json

# Connection usage
hcloud ces list-metric-data \
  --namespace SYS.RDS \
  --metric_name rds053_connections_usage \
  --dimension "instance_id={{user.instance_id}}" \
  --from "$(date -d '7 days ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 300 \
  -o json

# Replica lag
hcloud ces list-metric-data \
  --namespace SYS.RDS \
  --metric_name rds048_replica_lag \
  --dimension "instance_id={{user.instance_id}}" \
  --from "$(date -d '2 hours ago' +%s)000" \
  --to "$(date +%s)000" \
  --period 60 \
  -o json
```

### Instance Information

```bash
# Instance specs and storage
hcloud rds list-instances -o json | jq '
  .instances[] | {
    id, name,
    status: .status,
    vcpus: .vcpus,
    ram: .ram,
    disk: .volume.size,
    max_connections: .volume.max_connections
  }
'
```

## Forecast Algorithms

### Storage Exhaustion

```python
def forecast_storage_exhaustion(instance_id, days_ahead=7):
    """
    Linear regression on disk_usage to predict exhaustion date.
    """
    history = query_ces(
        namespace="SYS.RDS",
        metric="rds039_disk_usage",
        dimensions={"instance_id": instance_id},
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

    # days_to_90 = (90 - current) / (slope * 24)
    current = values[-1]
    daily_growth = slope * 24
    days_to_90 = (90 - current) / daily_growth if daily_growth > 0 else float("inf")

    return {
        "current_usage_pct": current,
        "daily_growth_rate_pct": daily_growth,
        "projected_in_7d": slope * (n - 1 + 7 * 24) + intercept,
        "days_to_90": days_to_90,
        "exhaustion_date": days_to_90 if days_to_90 < float("inf") else None,
        "recommendation": "expand_storage" if days_to_90 <= 7 else "monitor",
    }
```

### Connection Saturation

```python
def forecast_connection_saturation(instance_id, hours_ahead=24):
    """
    Trend analysis on connection usage ratio.
    """
    history = query_ces(
        namespace="SYS.RDS",
        metric="rds053_connections_usage",
        dimensions={"instance_id": instance_id},
        window="7d",
        period=300,
    )

    values = [p["value"] for p in history]
    n = len(values)

    # Compute slope over last 24h
    last_288 = values[-288:] if len(values) >= 288 else values  # 288 * 5min = 24h
    x = list(range(len(last_288)))
    x_mean = sum(x) / len(x)
    y_mean = sum(last_288) / len(last_288)
    slope = sum((xi - x_mean) * (yi - y_mean) for xi, yi in zip(x, last_288)) / \
            sum((xi - x_mean) ** 2 for xi in x)

    projected = values[-1] + slope * hours_ahead * 12  # 12 periods per hour
    projected = min(100, max(0, projected))

    return {
        "current_ratio": values[-1],
        "trend_per_5min": slope,
        "projected_in_24h": projected,
        "saturates_by_24h": projected > 90,
        "recommendation": "scale_connections" if projected > 80 else "monitor",
    }
```

## Capacity Planning Tables

### RDS Instance Right-Sizing

| Resource | Warning Threshold | Critical Threshold | Action |
|----------|-------------------|--------------------|--------|
| Disk usage | > 75% | > 85% | Expand storage |
| Connection ratio | > 70% | > 85% | Increase max_connections or scale up |
| CPU | > 80% | > 90% | Scale up vCPU |
| Memory | > 85% | > 95% | Scale up RAM |
| Replica lag | > 30s | > 60s | Investigate primary load |

### Storage Expansion Tiers

| Current Storage | Expansion Options | Use Case |
|-----------------|-------------------|----------|
| ≤ 40 GB | +40 GB increments | Small dev/staging |
| 40–500 GB | +100 GB increments | Standard production |
| > 500 GB | +200 GB increments | Large production |
| Any (SSD) | Switch to ESSD | High IOPS requirement |

## Predictive Alert Rules

```bash
# Storage exhaustion forecast
hcloud ces create-alarm-rule \
  --name "RDS-Storage-Exhaust-Forecast" \
  --metric rds039_disk_usage \
  --namespace SYS.RDS \
  --dimension "instance_id={{user.instance_id}}" \
  --condition "forecast_linear(168h) > 90%" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"

# Connection saturation forecast
hcloud ces create-alarm-rule \
  --name "RDS-Connection-Saturation-Forecast" \
  --metric rds053_connections_usage \
  --namespace SYS.RDS \
  --dimension "instance_id={{user.instance_id}}" \
  --condition "forecast_linear(24h) > 85%" \
  --alarm-level critical \
  --notifications "topic_arn:{{output.topic_arn}}"

# Replica lag warning
hcloud ces create-alarm-rule \
  --name "RDS-Replica-Lag-Growth" \
  --metric rds048_replica_lag \
  --namespace SYS.RDS \
  --dimension "instance_id={{user.instance_id}}" \
  --condition "slope(1h) > 10" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"
```

## Cross-Skill Delegation

| Capacity Issue | Delegate To | Purpose |
|----------------|-------------|---------|
| Storage expansion | RDS skill (resize) | Expand disk |
| Connection limit increase | RDS skill (parameter group) | Update max_connections |
| Read replica promotion | RDS skill (failover) | Handle replica lag |
| Backup failure | CBR skill | Vault and backup policy |
| Cost from over-sized instance | Billing skill | Right-sizing recommendation |

## Knowledge Base Anchors

- RDS ↔ ECS: [`references/integration.md`](../../huaweicloud-rds-ops/references/integration.md) — application connection patterns
- Slow query analysis: [`references/troubleshooting.md`](../../huaweicloud-rds-ops/references/troubleshooting.md)
- Cost anomaly: [`references/well-architected-assessment.md`](../../huaweicloud-rds-ops/references/well-architected-assessment.md) — FinOps
