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

### Alarm Storm Detection Script

**Purpose**: Detect alarm storms (high-frequency ECS alarm events) to trigger suppression workflow and prevent alert fatigue.

#### Detection Criteria

| Criterion | Threshold | Detection Logic |
|-----------|-----------|-----------------|
| Alarm frequency | > 10 alarms / 5 minutes | Time-window counting |
| Same instance spam | > 3 alarms on one ECS instance within 5 min | Group by instance_id |
| Namespace dominance | > 50% alarms from SYS.ECS namespace | Namespace distribution analysis |
| Cascade pattern | ECS alarm followed by downstream alarm within 2 min | Sequential timing correlation |

#### Execution — CLI (ECS Alarm Storm Detection)

```bash
#!/bin/bash
# ECS alarm storm detection script
# Detects high-frequency alarm events for suppression trigger
# Adapted from CES storm detection for SYS.ECS namespace

REGION="{{env.HW_REGION_ID}}"
ECS_NAMESPACE="SYS.ECS"
STORM_WINDOW_MINUTES=5
STORM_THRESHOLD=10
SAME_INSTANCE_THRESHOLD=3

# Step 1: Query recent ECS alarm events (last 15 minutes for analysis)
ALARM_EVENTS=$(hcloud ces list-alarm-history \
  --region "$REGION" \
  --namespace "$ECS_NAMESPACE" \
  --from "$(date -d '-15 minutes' +%s)000" \
  --to "$(date +%s)000" \
  --output json)

# Step 2: Count alarms in storm window (last 5 minutes)
WINDOW_START=$(date -d "-${STORM_WINDOW_MINUTES} minutes" +%s)
RECENT_ALARMS=$(echo "$ALARM_EVENTS" | jq --arg start "$WINDOW_START" '
  [.alarm_histories[] | select((.alarm_time | strftime("%s") | tonumber) > ($start | tonumber))]
')

ALARM_COUNT=$(echo "$RECENT_ALARMS" | jq 'length')

# Step 3: Storm detection - frequency check
if [ "$ALARM_COUNT" -ge "$STORM_THRESHOLD" ]; then
  echo "🚨 ECS ALARM STORM DETECTED: $ALARM_COUNT alarms in last $STORM_WINDOW_MINUTES minutes"
  
  # Step 4: Instance spam analysis (ECS-specific: group by instance_id)
  INSTANCE_SPAM=$(echo "$RECENT_ALARMS" | jq --argjson threshold "$SAME_INSTANCE_THRESHOLD" '
    group_by(.dimensions[] | select(.name == "instance_id") | .value) | map(select(length > $threshold)) | map({
      instance_id: .[0].dimensions[] | select(.name == "instance_id") | .value,
      alarm_count: length,
      alarm_names: [.[].alarm_name],
      alarm_types: [.[].metric_name]
    })
  ')
  
  SPAM_COUNT=$(echo "$INSTANCE_SPAM" | jq 'length')
  if [ "$SPAM_COUNT" -gt 0 ]; then
    echo "⚠️ Instance spam detected: $SPAM_COUNT ECS instances with > $SAME_INSTANCE_THRESHOLD alarms"
    echo "$INSTANCE_SPAM" | jq -r '.[] | "   Instance: \(.instance_id), Alarms: \(.alarm_count), Types: \(.alarm_types | join(", "))"'
  fi
  
  # Step 5: Metric distribution analysis (ECS-specific: cpu_util, mem_usedPercent, diskUsage_percent)
  METRIC_DOMINANCE=$(echo "$RECENT_ALARMS" | jq '
    group_by(.metric_name) | map({metric: .[0].metric_name, count: length})
    | sort_by(-.count) | .[0]
  ')
  
  DOMINANT_METRIC=$(echo "$METRIC_DOMINANCE" | jq -r '.metric')
  DOMINANT_PERCENT=$(echo "$METRIC_DOMINANCE" | jq --argjson total "$ALARM_COUNT" '.count * 100 / $total')
  
  if [ "$DOMINANT_PERCENT" -gt 50 ]; then
    echo "📊 Metric dominance: $DOMINANT_METRIC accounts for ${DOMINANT_PERCENT}% of alarms"
    
    # ECS-specific: suggest root cause based on dominant metric
    case "$DOMINANT_METRIC" in
      cpu_util) echo "   💡 Likely cause: CPU saturation, process storm, or workload spike" ;;
      mem_usedPercent) echo "   💡 Likely cause: Memory leak, OOM risk, or app memory bloat" ;;
      diskUsage_percent) echo "   💡 Likely cause: Disk filling, log accumulation, or storage exhaustion" ;;
      net_bits|net_pps) echo "   💡 Likely cause: Network storm, DDoS, or bandwidth saturation" ;;
      *) echo "   💡 Check ECS instance health for root cause" ;;
    esac
  fi
  
  # Step 6: Cascade pattern detection (ECS → downstream services)
  CASCADE_PATTERN=$(echo "$RECENT_ALARMS" | jq '
    sort_by(.alarm_time) | [.[]] | 
    reduce .[] as $alarm (
      {patterns: [], prev: null};
      if .prev != null and (($alarm.alarm_time | strftime("%s") | tonumber) - (.prev.alarm_time | strftime("%s") | tonumber)) < 120
      then .patterns += [{
        first: .prev.alarm_name,
        first_metric: .prev.metric_name,
        second: $alarm.alarm_name,
        second_metric: $alarm.metric_name,
        time_diff_seconds: (($alarm.alarm_time | strftime("%s") | tonumber) - (.prev.alarm_time | strftime("%s") | tonumber))
      }]
      else .
      end |
      .prev = $alarm
    ) | .patterns
  ')
  
  CASCADE_COUNT=$(echo "$CASCADE_PATTERN" | jq 'length')
  if [ "$CASCADE_COUNT" -gt 0 ]; then
    echo "🔗 Cascade patterns detected: $CASCADE_COUNT potential cascade sequences"
    echo "$CASCADE_PATTERN" | jq -r '.[] | "   \(.first) [\(.first_metric)] → \(.second) [\(.second_metric)] (\(.time_diff_seconds)s)"'
  fi
  
  # Step 7: Trigger suppression workflow
  echo "📋 Triggering ECS alarm suppression workflow..."
  
  # Identify root alarm (earliest in storm)
  ROOT_ALARM=$(echo "$RECENT_ALARMS" | jq 'sort_by(.alarm_time) | .[0]')
  ROOT_ALARM_ID=$(echo "$ROOT_ALARM" | jq -r '.alarm_id')
  ROOT_ALARM_NAME=$(echo "$ROOT_ALARM" | jq -r '.alarm_name')
  ROOT_INSTANCE_ID=$(echo "$ROOT_ALARM" | jq -r '.dimensions[] | select(.name == "instance_id") | .value')
  
  echo "   Root alarm identified: $ROOT_ALARM_NAME ($ROOT_ALARM_ID)"
  echo "   Affected instance: $ROOT_INSTANCE_ID"
  
  # ECS-specific: delegate to appropriate skill based on root alarm type
  ROOT_METRIC=$(echo "$ROOT_ALARM" | jq -r '.metric_name')
  case "$ROOT_METRIC" in
    cpu_util|mem_usedPercent) echo "   → Delegate to: huaweicloud-ecs-ops (instance-level diagnosis)" ;;
    diskUsage_percent) echo "   → Delegate to: huaweicloud-ecs-ops + storage cleanup" ;;
    net_bits|net_pps) echo "   → Delegate to: huaweicloud-waf-ops (potential DDoS)" ;;
  esac
  
  # Output storm report for automation
  jq -n \
    --argjson storm_detected true \
    --argjson alarm_count "$ALARM_COUNT" \
    --argjson window_minutes "$STORM_WINDOW_MINUTES" \
    --argjson instance_spam "$INSTANCE_SPAM" \
    --arg dominant_metric "$DOMINANT_METRIC" \
    --argjson dominant_percent "$DOMINANT_PERCENT" \
    --argjson cascade_patterns "$CASCADE_PATTERN" \
    --arg root_alarm_id "$ROOT_ALARM_ID" \
    --arg root_alarm_name "$ROOT_ALARM_NAME" \
    --arg root_instance_id "$ROOT_INSTANCE_ID" \
    --arg root_metric "$ROOT_METRIC" \
    --arg timestamp "$(date -Iseconds)" \
    '{
      storm_detected: $storm_detected,
      alarm_count: $alarm_count,
      window_minutes: $window_minutes,
      instance_spam: $instance_spam,
      dominant_metric: $dominant_metric,
      dominant_percent: $dominant_percent,
      cascade_patterns: $cascade_patterns,
      root_alarm: {
        alarm_id: $root_alarm_id,
        alarm_name: $root_alarm_name,
        instance_id: $root_instance_id,
        metric: $root_metric
      },
      timestamp: $timestamp,
      action: "trigger_suppression_workflow",
      delegation_target: (if $root_metric | test("cpu|mem") then "huaweicloud-ecs-ops" elif $root_metric | test("disk") then "huaweicloud-ecs-ops+storage" elif $root_metric | test("net") then "huaweicloud-waf-ops" else "huaweicloud-ecs-ops" end)
    }' | tee ecs-storm-detection-report.json
  
else
  echo "✅ No ECS alarm storm: $ALARM_COUNT alarms in last $STORM_WINDOW_MINUTES minutes (threshold: $STORM_THRESHOLD)"
fi
```

#### Storm Detection Output Format

```json
{
  "storm_detected": true,
  "alarm_count": 15,
  "window_minutes": 5,
  "instance_spam": [
    {
      "instance_id": "i-abc123",
      "alarm_count": 5,
      "alarm_names": ["cpu_high", "mem_high", "disk_high"],
      "alarm_types": ["cpu_util", "mem_usedPercent", "diskUsage_percent"]
    }
  ],
  "dominant_metric": "cpu_util",
  "dominant_percent": 60,
  "cascade_patterns": [
    {
      "first": "cpu_high",
      "first_metric": "cpu_util",
      "second": "mem_high",
      "second_metric": "mem_usedPercent",
      "time_diff_seconds": 45
    }
  ],
  "root_alarm": {
    "alarm_id": "alarm-001",
    "alarm_name": "cpu_high",
    "instance_id": "i-abc123",
    "metric": "cpu_util"
  },
  "timestamp": "2026-05-26T10:30:00Z",
  "action": "trigger_suppression_workflow",
  "delegation_target": "huaweicloud-ecs-ops"
}
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