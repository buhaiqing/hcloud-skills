# AIOps Best Practices — Huawei Cloud ECS

Intelligent operations integration patterns for ECS, enabling ML-driven anomaly detection, predictive scaling, and automated remediation.

## Multi-Metric Correlation Patterns

> **Canonical pattern registry**: see [`huaweicloud-ces-ops/references/advanced/anomaly-patterns.md`](../../huaweicloud-ces-ops/references/advanced/anomaly-patterns.md) — single source of truth for pattern names, thresholds, and severity.

ECS maps canonical patterns to product-specific metrics:

| Pattern | ECS Metrics | ECS Namespace | ML Feature Type |
|---------|------------|---------------|------------------|
| `cpu_mem_dual_high` | `cpu_util` + `mem_usedPercent` | SYS.ECS + AGT.ECS | boolean AND |
| `disk_io_bottleneck` | `read_iops` + `write_iops` + `diskUsage_percent` | SYS.ECS + AGT.ECS | composite |
| `mem_leak_trend` | `mem_usedPercent` (30min window) | AGT.ECS | time_series_slope |
| `sudden_cpu_spike` | `cpu_util` delta (5min) | SYS.ECS | rate_of_change |
| `network_storm` | `net_bits` + `net_pps` | SYS.ECS | anomaly_score |
| `disk_fill_acceleration` | `diskUsage_percent` (1h halves) | AGT.ECS | acceleration |

## ML Integration Requirements

### Feature Engineering

| Pattern | ML Feature | Training Window | Data Source |
|---------|-----------|-----------------|-------------|
| `cpu_mem_dual_high` | `cpu_util`, `mem_usedPercent` | Real-time (5s) | CES DescribeMetricData |
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

### Alarm Storm Detection

> **Canonical implementation**: see [`huaweicloud-ces-ops/references/monitoring.md`](../../huaweicloud-ces-ops/references/monitoring.md#storm-detection-logic-cli) — the common storm detection pattern. ECS skill adds namespace filtering and instance-level grouping below.

#### ECS-Specific Detection Criteria

| Criterion | Threshold | ECS-Specific Logic |
|-----------|-----------|--------------------|
| Alarm frequency | > 10 alarms / 5 minutes | Same as CES common |
| Same instance spam | > 3 alarms on one ECS instance within 5 min | Group by `instance_id` dimension |
| Namespace dominance | > 50% alarms from `SYS.ECS` | Filter `--namespace SYS.ECS` |
| Cascade pattern | ECS alarm → downstream within 2 min | Sequential timing correlation |

#### ECS Additions to Common Pattern

The CES common storm detection script handles frequency counting, time-window filtering, and cascade detection. ECS adds three extensions:

**1. Namespace filter** — scope to ECS alarms only:
```bash
ALARM_EVENTS=$(hcloud ces list-alarm-history --region "$REGION" \
  --namespace "SYS.ECS" \
  --from "$(date -d '-15 minutes' +%s)000" --to "$(date +%s)000" --output json)
```

**2. Instance spam analysis** — group by `instance_id` dimension:
```bash
INSTANCE_SPAM=$(echo "$RECENT_ALARMS" | jq --argjson threshold 3 '
  group_by(.dimensions[] | select(.name == "instance_id") | .value)
  | map(select(length > $threshold))
  | map({instance_id: .[0].dimensions[] | select(.name == "instance_id").value, alarm_count: length})
')
```

**3. ECS root-cause hints** — route by dominant metric:
```bash
case "$DOMINANT_METRIC" in
  cpu_util)           echo "→ CPU saturation / process storm" ;;
  mem_usedPercent)    echo "→ Memory leak / OOM risk" ;;
  diskUsage_percent)  echo "→ Disk filling / log accumulation" ;;
  net_bits|net_pps)   echo "→ Network storm / DDoS — delegate to huaweicloud-waf-ops" ;;
esac
```

#### Integration with ECS Suppression Workflow

1. **Detect** → Storm detection script triggers
2. **Correlate** → Group alarms by instance_id, metric_name, time
3. **Identify root** → Find earliest alarm + affected ECS instance
4. **Suppress** → Disable non-critical alarms for affected instances
5. **Delegate** → Route to appropriate skill based on root metric type
6. **Escalate** → Create incident with root cause analysis
7. **Restore** → Re-enable suppressed alarms after resolution

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