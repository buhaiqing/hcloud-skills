# AIOps Best Practices — Huawei Cloud ECS

Intelligent operations integration patterns for ECS, enabling ML-driven anomaly detection, predictive scaling, and automated remediation.

## Multi-Metric Correlation Patterns

ECS AIOps leverages multiple CES metrics for intelligent anomaly detection:

### Pattern Detection Matrix

| Pattern Name | Metrics Correlated | Logic | ML Feature Type | Severity |
|--------------|-------------------|-------|-----------------|----------|
| `cpu_mem_dual_high` | `cpu_util`, `mem_usedPercent` | cpu>80% AND mem>85% | boolean AND | Critical |
| `disk_io_bottleneck` | `read_iops`, `write_iops`, `diskUsage_percent` | IOPS>limit AND diskUtil>90% | composite | Critical |
| `mem_leak_trend` | `mem_usedPercent` (30min window) | slope > 0.5%/min | time_series_slope | Warning |
| `sudden_cpu_spike` | `cpu_util` delta | delta(5min) > 50% | rate_of_change | Warning |
| `network_storm` | `net_bits`, `net_pps` | pps > 10× baseline | anomaly_score | Critical |
| `disk_fill_acceleration` | `diskUsage_percent` (1h halves) | rate_half2 > rate_half1 | acceleration | Critical |

## ML Integration Requirements

### Feature Engineering

| Pattern | ML Feature | Training Window | Data Source |
|---------|-----------|-----------------|-------------|
| `cpu_mem_dual_high` | `cpu_util`, `mem_usedPercent` values | Real-time (5s) | CES DescribeMetricData |
| `mem_leak_trend` | Slope coefficient | 30 min sliding | CES time-series |
| `disk_fill_acceleration` | Fill rate half1, half2 | 1h window (2 × 30min) | CES aggregated |
| `network_storm` | `net_pps` baseline, current | 7d baseline + current | CES historical |
| `sudden_cpu_spike` | `cpu_util` previous, current | 5min × 2 | CES delta query |

### Model Input Schema

```json
{
  "instance_id": "string",
  "timestamp": "ISO8601",
  "features": {
    "cpu_util": "float (0-100)",
    "mem_usedPercent": "float (0-100)",
    "diskUsage_percent": "float (0-100)",
    "read_iops": "int",
    "write_iops": "int",
    "net_bits": "int",
    "net_pps": "int",
    "load1": "float",
    "load5": "float",
    "load15": "float"
  },
  "derived_features": {
    "cpu_mem_ratio": "float",
    "cpu_delta_5min": "float",
    "mem_slope_30min": "float",
    "disk_fill_rate": "float",
    "net_pps_baseline_ratio": "float"
  }
}
```

## Predictive AIOps Patterns

### Capacity Forecasting

| Forecast Type | Method | Prediction Window | Input Data | Accuracy Target |
|---------------|--------|-------------------|------------|-----------------|
| Disk capacity exhaustion | Linear regression on diskUsage | 24-72h before 90% | 7d diskUsage time-series | ±10% |
| Memory exhaustion | ARIMA forecast | 48h before OOM | 30d mem_usedPercent | ±15% |
| CPU saturation | Holt-Winters exponential smoothing | 7d before 100% | 30d cpu_util | ±20% |
| Cost spike | Billing trend analysis | Next billing cycle | BSS daily_cost | ±25% |

### Predictive Alert Rules

```bash
# Disk fill forecast (CES + BSS)
hcloud ces create-alarm-rule \
  --name "ECS-Disk-Fill-Forecast" \
  --metric diskUsage_percent \
  --namespace SYS.ECS \
  --condition "forecast_linear(24h) > 90%" \
  --alarm-level warning \
  --notifications "topic_arn:{{output.topic_arn}}"
```

## Automated Remediation Triggers

### Auto-Action Decision Matrix

| Trigger | Condition | Auto-Action | Safety Gate | Recovery |
|---------|-----------|-------------|-------------|----------|
| Disk > 95% | diskUsage > 95% AND forecast < 2h | CloudShell: clean /tmp, rotate logs | Verify no app files deleted | Log to LTS |
| Memory leak | mem_slope > 1%/min AND duration > 10min | CloudShell: restart app service | Check systemd service exists | Verify restart success |
| CPU storm | cpu_util > 95% AND unknown process | CloudShell: kill -STOP suspicious PID | HSS scan first | Log PID details to LTS |
| Spot pre-reclaim | Spot price > bid price + 10% | AS: launch replacement instance | Verify AS configured | Notify + new instance ID |

### Remediation Workflow

```
[Anomaly Detected]
    │
    ├── 1. Verify anomaly (cross-check 2+ metrics)
    ├── 2. Safety gate check (no critical data loss risk)
    ├── 3. Execute remediation via CloudShell
    ├── 4. Validate remediation success
    ├── 5. Log action to LTS for audit
    └── 6. Notify user with action summary
```

## Alert Storm Prevention

### Suppression Rules

| Condition | Suppression | Duration | Reason |
|-----------|-------------|----------|--------|
| Same alarm within 5min | Suppress duplicate | 5min | Avoid noise |
| Child alarm when parent active | Suppress child | Parent duration | Root cause linked |
| Remediation in progress | Suppress new alerts | 10min | Let action complete |
| Maintenance window | Suppress all | Window duration | Planned activity |

### Correlation Groups

```yaml
correlation_groups:
  resource_pressure:
    - cpu_util_high
    - mem_usedPercent_high
    - load_high
    suppress_after_first: true
    
  storage_saturation:
    - diskUsage_percent_critical
    - write_iops_saturation
    - read_iops_saturation
    suppress_after_first: true
```

## Cross-Skill AIOps Integration

### Delegation Matrix

| ECS Anomaly | Primary | Delegate To | Purpose |
|-------------|---------|-------------|---------|
| Java memory leak | ECS skill | AOM skill | Heap dump, GC analysis |
| Unknown process storm | ECS skill | HSS skill | Malware scan, process whitelist |
| Database on ECS slow | ECS skill | RDS skill (if self-managed DB) | DB query analysis |
| Network storm | ECS skill | WAF skill | DDoS detection |

### Data Flow

```
ECS CES Metrics ──▶ AIOps Engine ──▶ Anomaly Detection
                        │
                        ├──▶ CloudShell Remediation
                        ├──▶ Cross-Skill Delegation
                        └──▶ LTS Audit Logging
```

## AIOps Skill Integration Checklist

When integrating this ECS skill with an AIOps platform:

- [ ] CES metrics exported to ML pipeline (Prometheus/Cloud Eye API)
- [ ] Feature engineering pipeline configured
- [ ] Anomaly detection model trained on historical data
- [ ] Remediation actions mapped to CloudShell commands
- [ ] Alert suppression rules configured
- [ ] Cross-skill delegation paths tested
- [ ] Audit logging to LTS verified
- [ ] Safety gates validated (no data loss)