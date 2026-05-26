# Monitoring — Huawei Cloud ECS

## CES Metrics (Cloud Eye Service)

Namespace: `SYS.ECS`

### Key Metrics

| Metric | Name | Unit | Recommended Threshold |
|--------|------|------|---------------------|
| `cpu_util` | CPU utilization | % | Warning: 75%, Critical: 90% |
| `mem_usedPercent` | Memory utilization | % | Warning: 80%, Critical: 95% |
| `diskUsage_percent` | Disk usage | % | Warning: 80%, Critical: 90% |
| `read_iops` | Disk read IOPS | count/s | Baseline-dependent |
| `write_iops` | Disk write IOPS | count/s | Baseline-dependent |
| `net_bits` | Network bandwidth | bit/s | Baseline-dependent |
| `net_pps` | Network packets/s | count/s | Baseline-dependent |
| `load1`, `load5`, `load15` | System load | — | Warning: vCPU count |

## Alert Patterns

### Resource Pressure Alerts

| Alert | Metric | Condition | Severity |
|-------|--------|-----------|----------|
| CPU overload | `cpu_util` | avg(5min) > 90% | Critical |
| Memory exhaustion | `mem_usedPercent` | avg(5min) > 95% | Critical |
| Disk full | `diskUsage_percent` | value > 90% | Critical |
| IOPS saturation | `read_iops + write_iops` | > 80% of flavor limit | Warning |
| Bandwidth saturation | `net_bits` | > 80% of limit | Warning |

### Anomaly Patterns

| Pattern | Metrics | Detection Logic | Severity |
|---------|---------|----------------|----------|
| cpu_mem_dual_high | `cpu_util`, `mem_usedPercent` | cpu>80% AND mem>85% | Critical |
| disk_io_bottleneck | `read_iops`, `write_iops`, `diskUsage` | IOPS peak > limit AND diskUtil>90% | Critical |
| mem_leak_trend | `mem_usedPercent` (30min) | slope > 0.5%/min continuously | Warning |
| sudden_cpu_spike | `cpu_util` | delta(5min) > 50% | Warning |
| network_storm | `net_bits`, `net_pps` | pps > 10× baseline | Critical |
| disk_fill_acceleration | `diskUsage_percent` (1h) | fill rate increasing (half1 < half2 rate) | Critical |

## Predictive AIOps Patterns

### Capacity Forecasting Patterns

| Pattern | Detection Method | Prediction Window | Accuracy | Action |
|---------|-----------------|-------------------|----------|--------|
| disk_fill_forecast | Linear regression on diskUsage | 24-72h before 90% | ±10% | Preemptive cleanup |
| memory_exhaustion_forecast | ARIMA model on mem_usedPercent | 48h before OOM | ±15% | Scale recommendation |
| cpu_saturation_forecast | Holt-Winters smoothing on cpu_util | 7d before 100% | ±20% | Right-size alert |
| cost_spike_forecast | Billing trend analysis | Next billing cycle | ±25% | Budget alert |

### Forecast-Based Alert Rules

```bash
# Disk capacity exhaustion prediction
hcloud ces create-alarm-rule \
  --name "ECS-Disk-Forecast-90" \
  --metric diskUsage_percent \
  --namespace SYS.ECS \
  --condition "forecast_linear(72h) >= 90" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"

# Memory exhaustion prediction
hcloud ces create-alarm-rule \
  --name "ECS-Memory-OOM-Forecast" \
  --metric mem_usedPercent \
  --namespace SYS.ECS \
  --condition "forecast_arima(48h) >= 95" \
  --alarm-level critical
```

## ML Feature Metadata

### Feature Engineering for Anomaly Detection

| Pattern | ML Feature | Type | Training Window | Normalization |
|---------|-----------|------|-----------------|---------------|
| cpu_mem_dual_high | `cpu_util`, `mem_usedPercent` | float (0-100) | Real-time (5s) | Percentage |
| mem_leak_trend | Slope coefficient | float | 30 min sliding | Rate per minute |
| disk_fill_acceleration | Fill rate half1, half2 | float | 1h window (2 × 30min) | Rate per hour |
| network_storm | `net_pps` baseline_ratio | float | 7d baseline | Ratio (current/baseline) |
| sudden_cpu_spike | `cpu_util` delta_5min | float | 5min × 2 | Absolute delta |

### Model Input Schema (JSON)

```json
{
  "instance_id": "{{user.instance_id}}",
  "timestamp": "2026-05-26T10:00:00Z",
  "features": {
    "cpu_util": 75.2,
    "mem_usedPercent": 82.1,
    "diskUsage_percent": 67.5,
    "read_iops": 1200,
    "write_iops": 800,
    "net_bits": 50000000,
    "net_pps": 5000,
    "load1": 2.5,
    "load5": 2.2,
    "load15": 2.0
  },
  "derived_features": {
    "cpu_mem_ratio": 0.91,
    "cpu_delta_5min": 12.5,
    "mem_slope_30min": 0.8,
    "disk_fill_rate": 0.15,
    "net_pps_baseline_ratio": 1.2
  }
}
```

## Dashboards

- CES Console: `https://console.huaweicloud.com/ces/#/metricView/instances`
- Recommended dashboard: group by environment (prod/staging/dev), filter by tag
- Custom dashboards via CES CreateDashboard API

## Alarm Rules (CES)

```bash
# Create alarm for CPU > 85%
hcloud ces create-alarm-rule \
  --region {{env.HW_REGION_ID}} \
  --name "ECS-CPU-High" \
  --metric cpu_util \
  --namespace SYS.ECS \
  --condition "average > 85, 3 times" \
  --alarm-level critical \
  --notifications "topic_arn:{{output.topic_arn}}"
```

## Cost & Performance Metrics

| Metric | Purpose | Optimization Action |
|--------|---------|-------------------|
| `ecs_monthly_cost` (BSS) | Monthly cost per instance | Right-size or decommission |
| `cpu_util` avg(7d) < 10% | Idle instance detection | Downgrade, stop, or delete |
| `cpu_util` avg(7d) > 80% | Overloaded instance | Upgrade flavor or scale out |
