# CES Monitoring & Self-Monitoring — Huawei Cloud Cloud Eye Service

## CESMetrics for the Service Itself

CES is the monitoring tool, but it also has operational metrics to track:

| What to Monitor | Why | How |
|-----------------|-----|-----|
| Alarm state consistency | Ensure alarms reflect actual resource status | Periodic describe-alarm checks |
| API success rate | Detect API degradation | Track 4xx/5xx response rates |
| Alarm evaluation latency | Ensure timely alarm triggering | Monitor time from metric breach to alarm state change |
| Notification delivery | Verify alert delivery completeness | Track alarm_actions trigger vs actual notification |

## Recommended Dashboards

### Infrastructure Overview Dashboard
- CPU utilization across all ECS instances (SYS.ECS > cpu_util)
- Memory utilization across all ECS instances (AGT.ECS > memory_util)
- RDS CPU and connection usage (SYS.RDS > rds001_cpu_util, rds003_conn_usage)
- VPC bandwidth utilization (SYS.VPC > bandwidth_util)

### Application Performance Dashboard
- ELB active connections (SYS.ELB > l7e_listener_qps)
- ELB 5xx error count
- DCS memory utilization
- DMS consumer lag

### Cost Monitoring Dashboard
- Total cloud resources per service (via respective product skills, aggregated)
- Bandwidth consumption trends
- Storage utilization with projected costs

## Multi-Metric Anomaly Detection Scripts

This section provides executable jq-based detection scripts for 6 common anomaly patterns. Each script queries CES metrics and outputs JSON detection results.

### Detection Criteria Summary

| Pattern | Metrics | Threshold | Severity |
|---------|---------|-----------|----------|
| cpu_mem_dual_high | cpu_util + memory_util | cpu > 90% AND mem > 85% | Critical |
| disk_io_bottleneck | disk_read/write_bytes_rate + disk_util | I/O spike AND util > 90% | Warning |
| mem_leak_trend | memory_util slope (30min) | slope > 0.5%/min | Critical |
| sudden_cpu_spike | cpu_util delta (5min) | delta > 50% | Warning |
| network_saturation | network_in/out_bytes_rate | > 90% of bandwidth | Critical |
| rds_connection_exhaustion | rds003_conn_usage + rds007_qps | conn > 90% AND qps drops | Critical |

### Execution — CLI (Multi-Pattern Anomaly Detection)

```bash
#!/bin/bash
# Multi-metric anomaly detection scripts
# Detects 6 common patterns: cpu_mem_dual_high, disk_io_bottleneck,
# mem_leak_trend, sudden_cpu_spike, network_saturation, rds_connection_exhaustion

REGION="{{env.HW_REGION_ID}}"
INSTANCE_ID="{{user.instance_id}}"
OUTPUT_DIR="anomaly-detection-results"

mkdir -p "$OUTPUT_DIR"

# ============================================================================
# Pattern 1: cpu_mem_dual_high
# Severity: Critical — cpu > 90% AND mem > 85%
# ============================================================================
detect_cpu_mem_dual_high() {
  local instance_id="${1:-$INSTANCE_ID}"
  local from_ts=$(($(date +%s) - 600))000
  local to_ts=$(date +%s)000

  # Query CPU utilization (SYS.ECS)
  local cpu_data=$(hcloud ces query-metric-data \
    --region "$REGION" \
    --namespace SYS.ECS \
    --metric-name cpu_util \
    --dimension "instance_id:$instance_id" \
    --from "$from_ts" \
    --to "$to_ts" \
    --period 60 \
    --output json 2>/dev/null)

  # Query Memory utilization (AGT.ECS)
  local mem_data=$(hcloud ces query-metric-data \
    --region "$REGION" \
    --namespace AGT.ECS \
    --metric-name memory_util \
    --dimension "instance_id:$instance_id" \
    --from "$from_ts" \
    --to "$to_ts" \
    --period 60 \
    --output json 2>/dev/null)

  # Extract latest values
  local cpu_util=$(echo "$cpu_data" | jq -r '.datapoints[0].value // 0')
  local mem_util=$(echo "$mem_data" | jq -r '.datapoints[0].value // 0')

  # Detection logic: cpu > 90 AND mem > 85
  local detected=false
  if (( $(echo "$cpu_util > 90" | bc -l) )) && \
     (( $(echo "$mem_util > 85" | bc -l) )); then
    detected=true
  fi

  # Output JSON result
  jq -n \
    --arg pattern "cpu_mem_dual_high" \
    --argjson detected "$detected" \
    --arg timestamp "$(date -Iseconds)" \
    --arg resource_id "$instance_id" \
    --json-float 3 \
    --argjson cpu_util "$cpu_util" \
    --argjson memory_util "$mem_util" \
    --arg severity "Critical" \
    --arg recommendation "资源双高压可能导致OOM，建议：(1)检查高负载进程 (2)考虑扩容CPU/内存 (3)检查内存泄漏" \
    '{
      pattern: $pattern,
      detected: $detected,
      timestamp: $timestamp,
      resource_id: $resource_id,
      metric_values: {cpu_util: $cpu_util, memory_util: $memory_util},
      severity: $severity,
      recommendation: $recommendation
    }'
}

# ============================================================================
# Pattern 2: disk_io_bottleneck
# Severity: Warning — I/O rate spike AND disk_util > 90%
# ============================================================================
detect_disk_io_bottleneck() {
  local instance_id="${1:-$INSTANCE_ID}"
  local from_ts=$(($(date +%s) - 600))000
  local to_ts=$(date +%s)000

  # Query disk read bytes rate
  local read_data=$(hcloud ces query-metric-data \
    --region "$REGION" \
    --namespace SYS.ECS \
    --metric-name disk_read_bytes_rate \
    --dimension "instance_id:$instance_id" \
    --from "$from_ts" \
    --to "$to_ts" \
    --period 60 \
    --output json 2>/dev/null)

  # Query disk write bytes rate
  local write_data=$(hcloud ces query-metric-data \
    --region "$REGION" \
    --namespace SYS.ECS \
    --metric-name disk_write_bytes_rate \
    --dimension "instance_id:$instance_id" \
    --from "$from_ts" \
    --to "$to_ts" \
    --period 60 \
    --output json 2>/dev/null)

  # Query disk utilization (AGT.ECS)
  local util_data=$(hcloud ces query-metric-data \
    --region "$REGION" \
    --namespace AGT.ECS \
    --metric-name disk_util \
    --dimension "instance_id:$instance_id" \
    --from "$from_ts" \
    --to "$to_ts" \
    --period 60 \
    --output json 2>/dev/null)

  local read_rate=$(echo "$read_data" | jq -r '.datapoints[0].value // 0')
  local write_rate=$(echo "$write_data" | jq -r '.datapoints[0].value // 0')
  local disk_util=$(echo "$util_data" | jq -r '.datapoints[0].value // 0')

  # Calculate total I/O rate and detect spike (threshold: 100MB/s = 104857600 bytes/s)
  local io_rate=$(echo "$read_rate + $write_rate" | bc -l)
  local io_spike=false
  if (( $(echo "$io_rate > 104857600" | bc -l) )); then
    io_spike=true
  fi

  # Detection: I/O spike AND disk_util > 90%
  local detected=false
  if [ "$io_spike" = true ] && \
     (( $(echo "$disk_util > 90" | bc -l) )); then
    detected=true
  fi

  jq -n \
    --arg pattern "disk_io_bottleneck" \
    --argjson detected "$detected" \
    --arg timestamp "$(date -Iseconds)" \
    --arg resource_id "$instance_id" \
    --json-float 3 \
    --argjson read_rate "$read_rate" \
    --argjson write_rate "$write_rate" \
    --argjson io_rate "$io_rate" \
    --argjson disk_util "$disk_util" \
    --arg severity "Warning" \
    --arg recommendation "磁盘IO瓶颈，建议：(1)检查IO密集型进程 (2)考虑使用高速云盘 (3)优化IO读写模式" \
    '{
      pattern: $pattern,
      detected: $detected,
      timestamp: $timestamp,
      resource_id: $resource_id,
      metric_values: {read_rate_Bps: $read_rate, write_rate_Bps: $write_rate, total_io_Bps: $io_rate, disk_util_pct: $disk_util},
      severity: $severity,
      recommendation: $recommendation
    }'
}

# ============================================================================
# Pattern 3: mem_leak_trend
# Severity: Critical — memory_util slope > 0.5%/min continuously
# ============================================================================
detect_mem_leak_trend() {
  local instance_id="${1:-$INSTANCE_ID}"
  local from_ts=$(($(date +%s) - 1800))000  # 30 min window
  local to_ts=$(date +%s)000

  # Query memory utilization over 30 minutes (6 data points, 5min apart)
  local mem_data=$(hcloud ces query-metric-data \
    --region "$REGION" \
    --namespace AGT.ECS \
    --metric-name memory_util \
    --dimension "instance_id:$instance_id" \
    --from "$from_ts" \
    --to "$to_ts" \
    --period 300 \
    --output json 2>/dev/null)

  # Extract all datapoints for slope calculation
  local datapoints=$(echo "$mem_data" | jq '.datapoints | map(.value)')

  # Calculate linear slope using jq
  # y = mx + b, m = (n*sum(xy) - sum(x)*sum(y)) / (n*sum(x^2) - (sum(x))^2)
  local slope_result=$(echo "$datapoints" | jq --argjson n 6 '
    to_entries | map({
      x: (.key | tonumber),
      y: (.value | tonumber)
    }) | reduce .[] as $p (
      {sx: 0, sy: 0, sxx: 0, sxy: 0};
      {
        sx: (.sx + $p.x),
        sy: (.sy + $p.y),
        sxx: (.sxx + ($p.x * $p.x)),
        sxy: (.sxy + ($p.x * $p.y))
      }
    ) | {
      slope: ((6 * .sxy - .sx * .sy) / (6 * .sxx - .sx * .sx)),
      points: .
    }
  ')

  local slope=$(echo "$slope_result" | jq '.slope')
  # slope is per 5-min interval; convert to %/min: slope / 5
  local slope_per_min=$(echo "$slope" | jq '(. / 5)')

  # Detection: slope > 0.5% per minute continuously
  local detected=false
  if (( $(echo "$slope_per_min > 0.5" | bc -l) )); then
    detected=true
  fi

  jq -n \
    --arg pattern "mem_leak_trend" \
    --argjson detected "$detected" \
    --arg timestamp "$(date -Iseconds)" \
    --arg resource_id "$instance_id" \
    --json-float 6 \
    --argjson slope "$slope" \
    --argjson slope_per_min "$slope_per_min" \
    --argjson threshold 0.5 \
    --arg severity "Critical" \
    --arg recommendation "检测到内存泄漏趋势，建议：(1)分析内存使用进程 (2)检查内存泄漏代码 (3)考虑重启应用或扩容" \
    '{
      pattern: $pattern,
      detected: $detected,
      timestamp: $timestamp,
      resource_id: $resource_id,
      metric_values: {slope_per_interval: $slope, slope_per_min: $slope_per_min, threshold_per_min: $threshold},
      severity: $severity,
      recommendation: $recommendation
    }'
}

# ============================================================================
# Pattern 4: sudden_cpu_spike
# Severity: Warning — delta(5min) > 50%
# ============================================================================
detect_sudden_cpu_spike() {
  local instance_id="${1:-$INSTANCE_ID}"
  local from_ts=$(($(date +%s) - 600))000
  local to_ts=$(date +%s)000

  # Query CPU utilization
  local cpu_data=$(hcloud ces query-metric-data \
    --region "$REGION" \
    --namespace SYS.ECS \
    --metric-name cpu_util \
    --dimension "instance_id:$instance_id" \
    --from "$from_ts" \
    --to "$to_ts" \
    --period 60 \
    --output json 2>/dev/null)

  # Get current and 5-min-ago values
  local current_cpu=$(echo "$cpu_data" | jq -r '.datapoints[0].value // 0')
  local prev_cpu=$(echo "$cpu_data" | jq -r '.datapoints[5].value // 0')

  # Calculate delta
  local delta=$(echo "$current_cpu - $prev_cpu" | bc -l)
  local abs_delta=${delta#-}  # absolute value

  # Detection: |delta| > 50%
  local detected=false
  if (( $(echo "$abs_delta > 50" | bc -l) )); then
    detected=true
  fi

  jq -n \
    --arg pattern "sudden_cpu_spike" \
    --argjson detected "$detected" \
    --arg timestamp "$(date -Iseconds)" \
    --arg resource_id "$instance_id" \
    --json-float 3 \
    --argjson current_cpu "$current_cpu" \
    --argjson prev_cpu "$prev_cpu" \
    --argjson delta "$delta" \
    --argjson threshold 50 \
    --arg severity "Warning" \
    --arg recommendation "检测到CPU突变，建议：(1)检查触发突变的进程 (2)分析突发负载来源 (3)考虑限流或扩容" \
    '{
      pattern: $pattern,
      detected: $detected,
      timestamp: $timestamp,
      resource_id: $resource_id,
      metric_values: {current_cpu_pct: $current_cpu, prev_cpu_pct: $prev_cpu, delta_pct: $delta, threshold_pct: $threshold},
      severity: $severity,
      recommendation: $recommendation
    }'
}

# ============================================================================
# Pattern 5: network_saturation
# Severity: Critical — inbound/outbound > 90% of bandwidth limit
# ============================================================================
detect_network_saturation() {
  local instance_id="${1:-$INSTANCE_ID}"
  local from_ts=$(($(date +%s) - 600))000
  local to_ts=$(date +%s)000

  # Query network in bytes rate
  local net_in=$(hcloud ces query-metric-data \
    --region "$REGION" \
    --namespace SYS.ECS \
    --metric-name network_in_bytes_rate \
    --dimension "instance_id:$instance_id" \
    --from "$from_ts" \
    --to "$to_ts" \
    --period 60 \
    --output json 2>/dev/null)

  # Query network out bytes rate
  local net_out=$(hcloud ces query-metric-data \
    --region "$REGION" \
    --namespace SYS.ECS \
    --metric-name network_out_bytes_rate \
    --dimension "instance_id:$instance_id" \
    --from "$from_ts" \
    --to "$to_ts" \
    --period 60 \
    --output json 2>/dev/null)

  local in_rate=$(echo "$net_in" | jq -r '.datapoints[0].value // 0')
  local out_rate=$(echo "$net_out" | jq -r '.datapoints[0].value // 0')

  # Default bandwidth limit: 1000Mbps = 125000000 bytes/s (if not provided)
  local bandwidth_limit=$(echo "125000000" | jq -r '. // 125000000')
  local threshold_rate=$(echo "$bandwidth_limit * 0.9" | bc -l)

  # Detection: inbound OR outbound > 90% of bandwidth
  local detected=false
  if (( $(echo "$in_rate > $threshold_rate" | bc -l) )) || \
     (( $(echo "$out_rate > $threshold_rate" | bc -l) )); then
    detected=true
  fi

  jq -n \
    --arg pattern "network_saturation" \
    --argjson detected "$detected" \
    --arg timestamp "$(date -Iseconds)" \
    --arg resource_id "$instance_id" \
    --json-float 3 \
    --argjson inbound_rate "$in_rate" \
    --argjson outbound_rate "$out_rate" \
    --argjson bandwidth_limit "$bandwidth_limit" \
    --argjson threshold_pct 90 \
    --arg severity "Critical" \
    --arg recommendation "网络带宽饱和，建议：(1)优化网络流量 (2)启用流量压缩 (3)考虑升级带宽 (4)检查异常流量来源" \
    '{
      pattern: $pattern,
      detected: $detected,
      timestamp: $timestamp,
      resource_id: $resource_id,
      metric_values: {inbound_Bps: $inbound_rate, outbound_Bps: $outbound_rate, bandwidth_limit_Bps: $bandwidth_limit, threshold_pct: $threshold_pct},
      severity: $severity,
      recommendation: $recommendation
    }'
}

# ============================================================================
# Pattern 6: rds_connection_exhaustion
# Severity: Critical — conn > 90% AND qps drops
# ============================================================================
detect_rds_connection_exhaustion() {
  local instance_id="${1:-$INSTANCE_ID}"
  local from_ts=$(($(date +%s) - 600))000
  local to_ts=$(date +%s)000

  # Query RDS connection usage (rds003_conn_usage)
  local conn_data=$(hcloud ces query-metric-data \
    --region "$REGION" \
    --namespace SYS.RDS \
    --metric-name rds003_conn_usage \
    --dimension "instance_id:$instance_id" \
    --from "$from_ts" \
    --to "$to_ts" \
    --period 60 \
    --output json 2>/dev/null)

  # Query RDS QPS (rds007_qps)
  local qps_data=$(hcloud ces query-metric-data \
    --region "$REGION" \
    --namespace SYS.RDS \
    --metric-name rds007_qps \
    --dimension "instance_id:$instance_id" \
    --from "$from_ts" \
    --to "$to_ts" \
    --period 60 \
    --output json 2>/dev/null)

  local conn_usage=$(echo "$conn_data" | jq -r '.datapoints[0].value // 0')
  local current_qps=$(echo "$qps_data" | jq -r '.datapoints[0].value // 0')
  local prev_qps=$(echo "$qps_data" | jq -r '.datapoints[5].value // 0')

  # Detection: conn > 90% AND qps drops
  local detected=false
  if (( $(echo "$conn_usage > 90" | bc -l) )) && \
     (( $(echo "$current_qps < $prev_qps * 0.8" | bc -l) )); then
    detected=true
  fi

  jq -n \
    --arg pattern "rds_connection_exhaustion" \
    --argjson detected "$detected" \
    --arg timestamp "$(date -Iseconds)" \
    --arg resource_id "$instance_id" \
    --json-float 3 \
    --argjson conn_usage_pct "$conn_usage" \
    --argjson current_qps "$current_qps" \
    --argjson prev_qps "$prev_qps" \
    --argjson qps_drop_pct "$(echo "if $prev_qps > 0 then (($current_qps - $prev_qps) / $prev_qps * 100) else 0 end" | bc -l)" \
    --arg severity "Critical" \
    --arg recommendation "数据库连接池接近耗尽，建议：(1)检查连接泄漏 (2)优化连接池配置 (3)考虑扩容数据库实例 (4)优化慢查询" \
    '{
      pattern: $pattern,
      detected: $detected,
      timestamp: $timestamp,
      resource_id: $resource_id,
      metric_values: {conn_usage_pct: $conn_usage_pct, current_qps: $current_qps, prev_qps: $prev_qps},
      severity: $severity,
      recommendation: $recommendation
    }'
}

# ============================================================================
# Main: Run all detections for specified instance
# ============================================================================
run_all_detections() {
  local instance_id="${1:-$INSTANCE_ID}"
  echo "Running anomaly detection for instance: $instance_id"
  echo "=============================================="

  echo "--- cpu_mem_dual_high ---"
  detect_cpu_mem_dual_high "$instance_id" | tee "$OUTPUT_DIR/cpu_mem_dual_high.json"

  echo "--- disk_io_bottleneck ---"
  detect_disk_io_bottleneck "$instance_id" | tee "$OUTPUT_DIR/disk_io_bottleneck.json"

  echo "--- mem_leak_trend ---"
  detect_mem_leak_trend "$instance_id" | tee "$OUTPUT_DIR/mem_leak_trend.json"

  echo "--- sudden_cpu_spike ---"
  detect_sudden_cpu_spike "$instance_id" | tee "$OUTPUT_DIR/sudden_cpu_spike.json"

  echo "--- network_saturation ---"
  detect_network_saturation "$instance_id" | tee "$OUTPUT_DIR/network_saturation.json"

  echo "--- rds_connection_exhaustion ---"
  detect_rds_connection_exhaustion "$instance_id" | tee "$OUTPUT_DIR/rds_connection_exhaustion.json"

  echo "=============================================="
  echo "Results saved to: $OUTPUT_DIR/"
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  run_all_detections "$@"
fi
```

## Alarm Storm Patterns and Suppression

### Detection Criteria

| Criterion | Threshold | Action |
|-----------|-----------|--------|
| Alarm frequency | > 10 alarms / 5 minutes | Enter storm mode |
| Same resource | > 3 alarms on one instance | Aggregate to single event |
| Same namespace | > 50% from same namespace | Focus diagnosis on product |
| Cascade pattern | Alarm A triggers, B triggers within 2min | Mark B as "likely caused by A" |

### Suppression Workflow

1. **Detect**: Count alarms by resource_id within time window
2. **Correlate**: Group alarms by resource, service, and time
3. **Identify root**: Find the earliest alarm or infrastructure-level alarm
4. **Suppress**: Disable non-critical alarms for affected resources
5. **Escalate**: Create incident with root cause analysis
6. **Restore**: Re-enable suppressed alarms after resolution

## Storm Handling Flow

```
[告警触发]
    │
    ├── Step 1: 验证告警有效性
    │   确认指标值是否确实超阈值 → 误报则检查告警规则配置
    │
    ├── Step 2: 检查资源状态
    │   委托对应产品Skill获取资源当前状态
    │
    ├── Step 3: 多指标关联分析
    │   查询CES相关指标,识别复合异常模式
    │
    ├── Step 4: 深度诊断(如适用)
    │   委托AOM应用监控/LTS日志服务
    │
    └── Step 5: 生成统一诊断报告
        汇总所有Skill发现,给出根因和修复建议
```

## Proactive Inspection (巡检) Workflow

### Five-Step Loop

```
[资源发现] → [指标采集] → [异常检测] → [跨Skill诊断] → [报告生成]
```

### Daily Checks
- Verify all critical alarm rules are enabled
- Check for alarms in `insufficient_data` state (indicates monitoring gap)
- Review API error rates in last 24 hours

### Weekly Checks
- Audit unused alarm rules (no triggers in 30 days)
- Review dashboard accuracy (resource changes may make widgets stale)
- Check metric retention and storage costs

### Monthly Checks
- Review alert response SLA compliance
- Analyze alarm accuracy (false positive rate)
- Optimize alarm thresholds based on actual usage patterns

## Cross-Skill Delegation Matrix (Alarm-Driven)

| Alarm Type | CES Namespace | Primary Skill | Secondary Skill | Delegation Level |
|-----------|---------------|---------------|-----------------|------------------|
| CPU高 | SYS.ECS / AGT.ECS | huaweicloud-ecs-ops | huaweicloud-cce-ops (if container) | Recommended |
| 内存泄漏 | AGT.ECS | huaweicloud-ecs-ops | — | Required |
| 磁盘满 | AGT.ECS | huaweicloud-ecs-ops | — | Required |
| 数据库慢 | SYS.RDS | huaweicloud-rds-ops | huaweicloud-ces-ops (metrics) | Required |
| 带宽饱和 | SYS.VPC | huaweicloud-vpc-ops | huaweicloud-ces-ops (metrics) | Recommended |
| ELB错误率 | SYS.ELB | huaweicloud-elb-ops | huaweicloud-ecs-ops (backend) | Recommended |
| 安全告警 | SYS.HSS | huaweicloud-hss-ops | huaweicloud-ecs-ops (isolation) | Required |

## Idle Alarm Detection Script (HIGH-1)

**Purpose**: Identify alarm rules that have not triggered in extended periods, indicating potential monitoring gaps or over-monitoring.

### Detection Criteria

| Criterion | Threshold | Interpretation |
|-----------|-----------|----------------|
| No triggers in 90 days | `trigger_count == 0` | Alarm may be misconfigured or resource changed |
| No triggers in 30 days | `trigger_count == 0` | Consider disabling if non-critical |
| Last trigger > 60 days ago | `last_trigger_time` | Review threshold appropriateness |
| Never triggered since creation | `trigger_count == 0 && age > 7 days` | High priority review |

### Execution — CLI (Idle Alarm Detection)

```bash
#!/bin/bash
# Idle alarm detection script
# Identifies alarms that have not triggered recently

REGION="{{env.HW_REGION_ID}}"
IDLE_DAYS_THRESHOLD=90
OUTPUT_FILE="idle-alarms-report.json"

# Step 1: List all alarms with trigger statistics
ALL_ALARMS=$(hcloud ces list-alarms \
  --region "$REGION" \
  --output json)

# Step 2: Process each alarm
IDLE_ALARMS=$(echo "$ALL_ALARMS" | jq --argjson threshold "$IDLE_DAYS_THRESHOLD" '
  .alarms | map(select(
    (.trigger_count == 0) or
    ((now - (.last_trigger_time | strftime("%s") | tonumber)) / 86400 > $threshold)
  )) | map({
    alarm_id,
    alarm_name,
    alarm_enabled,
    metric_namespace,
    metric_name,
    trigger_count,
    last_trigger_time,
    idle_days: (if .trigger_count == 0 then "never" else ((now - (.last_trigger_time | strftime("%s") | tonumber)) / 86400 | floor | tostring) end),
    recommendation: (if .trigger_count == 0 then "review_or_disable" else "threshold_review" end)
  })
')

# Step 3: Categorize by severity
NEVER_TRIGGERED=$(echo "$IDLE_ALARMS" | jq 'map(select(.trigger_count == 0))')
LONG_IDLE=$(echo "$IDLE_ALARMS" | jq 'map(select(.trigger_count > 0 and (.idle_days | tonumber) > 60))')

# Step 4: Generate report
REPORT=$(jq -n \
  --argjson never "$NEVER_TRIGGERED" \
  --argjson long_idle "$LONG_IDLE" \
  --arg timestamp "$(date -Iseconds)" \
  --arg region "$REGION" \
  '{
    generated_at: $timestamp,
    region: $region,
    summary: {
      total_idle: ($never | length) + ($long_idle | length),
      never_triggered: ($never | length),
      long_idle: ($long_idle | length)
    },
    never_triggered_alarms: $never,
    long_idle_alarms: $long_idle
  }')

echo "$REPORT" > "$OUTPUT_FILE"

# Step 5: Summary output
echo "📊 Idle Alarm Detection Report"
echo "   Region: $REGION"
echo "   Never triggered: $(echo "$NEVER_TRIGGERED" | jq 'length') alarms"
echo "   Long idle (>60 days): $(echo "$LONG_IDLE" | jq 'length') alarms"
echo "   Report saved to: $OUTPUT_FILE"

# Step 6: Recommend actions
if [ $(echo "$NEVER_TRIGGERED" | jq 'length') -gt 0 ]; then
  echo "⚠️ Recommended actions:"
  echo "   1. Review never-triggered alarms for misconfiguration"
  echo "   2. Verify monitored resource still exists"
  echo "   3. Consider disabling unused alarms to reduce quota usage"
fi
```

### Execution — SDK (Go Implementation)

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ces/v1/model"
)

type IdleAlarmReport struct {
    GeneratedAt         string           `json:"generated_at"`
    Region              string           `json:"region"`
    Summary             Summary          `json:"summary"`
    NeverTriggeredAlarms []AlarmInfo     `json:"never_triggered_alarms"`
    LongIdleAlarms      []AlarmInfo      `json:"long_idle_alarms"`
}

type Summary struct {
    TotalIdle       int `json:"total_idle"`
    NeverTriggered  int `json:"never_triggered"`
    LongIdle        int `json:"long_idle"`
}

type AlarmInfo struct {
    AlarmID        string `json:"alarm_id"`
    AlarmName      string `json:"alarm_name"`
    AlarmEnabled   bool   `json:"alarm_enabled"`
    MetricNamespace string `json:"metric_namespace"`
    MetricName     string `json:"metric_name"`
    TriggerCount   int    `json:"trigger_count"`
    LastTriggerTime string `json:"last_trigger_time"`
    IdleDays       string `json:"idle_days"`
    Recommendation string `json:"recommendation"`
}

func DetectIdleAlarms(region string, idleThresholdDays int) (*IdleAlarmReport, error) {
    // List all alarms (SDK call - implementation depends on client setup)
    // listResp, err := client.ListAlarms(&model.ListAlarmsRequest{Region: region})
    
    var idleAlarms []AlarmInfo
    var neverTriggered []AlarmInfo
    var longIdle []AlarmInfo
    
    // Process alarms (pseudo-code - implement with actual SDK response)
    // for _, alarm := range listResp.Alarms {
    //     if alarm.TriggerCount == 0 {
    //         neverTriggered = append(neverTriggered, buildAlarmInfo(alarm, "never", "review_or_disable"))
    //     } else {
    //         idleDays := calculateIdleDays(alarm.LastTriggerTime)
    //         if idleDays > idleThresholdDays {
    //             longIdle = append(longIdle, buildAlarmInfo(alarm, idleDays, "threshold_review"))
    //         }
    //     }
    // }
    
    report := &IdleAlarmReport{
        GeneratedAt:         time.Now().Format(time.RFC3339),
        Region:              region,
        Summary:             Summary{TotalIdle: len(neverTriggered) + len(longIdle), NeverTriggered: len(neverTriggered), LongIdle: len(longIdle)},
        NeverTriggeredAlarms: neverTriggered,
        LongIdleAlarms:      longIdle,
    }
    
    return report, nil
}

func main() {
    region := os.Getenv("HW_REGION_ID")
    report, err := DetectIdleAlarms(region, 90)
    if err != nil {
        log.Fatalf("Detection failed: %v", err)
    }
    
    output, _ := json.MarshalIndent(report, "", "  ")
    fmt.Println(string(output))
    
    // Save to file
    os.WriteFile("idle-alarms-report.json", output, 0644)
}
```

### Post-Detection Actions

| Category | Action | Priority |
|----------|--------|----------|
| Never triggered (critical alarm) | Verify threshold, resource existence, metric namespace | High |
| Never triggered (non-critical) | Consider disabling; reduce quota usage | Medium |
| Long idle (>90 days) | Review threshold appropriateness | Medium |
| Resource deleted but alarm exists | Delete alarm | Required |

## Alarm Storm Detection Script (HIGH-2)

**Purpose**: Detect alarm storms (high-frequency alarm events) to trigger suppression workflow and prevent alert fatigue.

### Detection Criteria

| Criterion | Threshold | Detection Logic |
|-----------|-----------|-----------------|
| Alarm frequency | > 10 alarms / 5 minutes | Time-window counting |
| Same resource spam | > 3 alarms on one instance within 5 min | Group by resource_id |
| Namespace dominance | > 50% alarms from same namespace | Namespace distribution analysis |
| Cascade pattern | Alarm A followed by B within 2 min | Sequential timing correlation |

### Execution — CLI (Alarm Storm Detection)

```bash
#!/bin/bash
# Alarm storm detection script
# Detects high-frequency alarm events for suppression trigger

REGION="{{env.HW_REGION_ID}}"
STORM_WINDOW_MINUTES=5
STORM_THRESHOLD=10
SAME_RESOURCE_THRESHOLD=3

# Step 1: Query recent alarm events (last 15 minutes for analysis)
ALARM_EVENTS=$(hcloud ces list-alarm-history \
  --region "$REGION" \
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
  echo "🚨 ALARM STORM DETECTED: $ALARM_COUNT alarms in last $STORM_WINDOW_MINUTES minutes"
  
  # Step 4: Resource spam analysis
  RESOURCE_SPAM=$(echo "$RECENT_ALARMS" | jq --argjson threshold "$SAME_RESOURCE_THRESHOLD" '
    group_by(.resource_id) | map(select(length > $threshold)) | map({
      resource_id: .[0].resource_id,
      alarm_count: length,
      alarm_names: [.[].alarm_name]
    })
  ')
  
  SPAM_COUNT=$(echo "$RESOURCE_SPAM" | jq 'length')
  if [ "$SPAM_COUNT" -gt 0 ]; then
    echo "⚠️ Resource spam detected: $SPAM_COUNT resources with > $SAME_RESOURCE_THRESHOLD alarms"
    echo "$RESOURCE_SPAM" | jq -r '.[] | "   Resource: \(.resource_id), Alarms: \(.alarm_count)"'
  fi
  
  # Step 5: Namespace distribution analysis
  NAMESPACE_DOMINANCE=$(echo "$RECENT_ALARMS" | jq '
    group_by(.metric_namespace) | map({namespace: .[0].metric_namespace, count: length})
    | sort_by(-.count) | .[0]
  ')
  
  DOMINANT_NAMESPACE=$(echo "$NAMESPACE_DOMINANCE" | jq -r '.namespace')
  DOMINANT_PERCENT=$(echo "$NAMESPACE_DOMINANCE" | jq --argjson total "$ALARM_COUNT" '.count * 100 / $total')
  
  if [ "$DOMINANT_PERCENT" -gt 50 ]; then
    echo "📊 Namespace dominance: $DOMINANT_NAMESPACE accounts for ${DOMINANT_PERCENT}% of alarms"
  fi
  
  # Step 6: Cascade pattern detection
  CASCADE_PATTERN=$(echo "$RECENT_ALARMS" | jq '
    sort_by(.alarm_time) | [.[]] | 
    reduce .[] as $alarm (
      {patterns: [], prev: null};
      if .prev != null and (($alarm.alarm_time | strftime("%s") | tonumber) - (.prev.alarm_time | strftime("%s") | tonumber)) < 120
      then .patterns += [{first: .prev.alarm_name, second: $alarm.alarm_name, time_diff_seconds: (($alarm.alarm_time | strftime("%s") | tonumber) - (.prev.alarm_time | strftime("%s") | tonumber))}]
      else .
      end |
      .prev = $alarm
    ) | .patterns
  ')
  
  CASCADE_COUNT=$(echo "$CASCADE_PATTERN" | jq 'length')
  if [ "$CASCADE_COUNT" -gt 0 ]; then
    echo "🔗 Cascade patterns detected: $CASCADE_COUNT potential cascade sequences"
    echo "$CASCADE_PATTERN" | jq -r '.[] | "   \(.first) → \(.second) (\(.time_diff_seconds)s)"'
  fi
  
  # Step 7: Trigger suppression workflow
  echo "📋 Triggering alarm suppression workflow..."
  
  # Identify root alarm (earliest in storm)
  ROOT_ALARM=$(echo "$RECENT_ALARMS" | jq 'sort_by(.alarm_time) | .[0]')
  ROOT_ALARM_ID=$(echo "$ROOT_ALARM" | jq -r '.alarm_id')
  ROOT_ALARM_NAME=$(echo "$ROOT_ALARM" | jq -r '.alarm_name')
  
  echo "   Root alarm identified: $ROOT_ALARM_NAME ($ROOT_ALARM_ID)"
  
  # Output storm report for automation
  jq -n \
    --argjson storm_detected true \
    --argjson alarm_count "$ALARM_COUNT" \
    --argjson window_minutes "$STORM_WINDOW_MINUTES" \
    --argjson resource_spam "$RESOURCE_SPAM" \
    --arg dominant_namespace "$DOMINANT_NAMESPACE" \
    --argjson cascade_patterns "$CASCADE_PATTERN" \
    --arg root_alarm_id "$ROOT_ALARM_ID" \
    --arg timestamp "$(date -Iseconds)" \
    '{
      storm_detected: $storm_detected,
      alarm_count: $alarm_count,
      window_minutes: $window_minutes,
      resource_spam: $resource_spam,
      dominant_namespace: $dominant_namespace,
      cascade_patterns: $cascade_patterns,
      root_alarm_id: $root_alarm_id,
      timestamp: $timestamp,
      action: "trigger_suppression_workflow"
    }' | tee storm-detection-report.json
  
else
  echo "✅ No alarm storm: $ALARM_COUNT alarms in last $STORM_WINDOW_MINUTES minutes (threshold: $STORM_THRESHOLD)"
fi
```

### Storm Detection Output Format

```json
{
  "storm_detected": true,
  "alarm_count": 15,
  "window_minutes": 5,
  "resource_spam": [
    {"resource_id": "i-abc123", "alarm_count": 5, "alarm_names": ["cpu_high", "mem_high", "disk_high"]}
  ],
  "dominant_namespace": "SYS.ECS",
  "cascade_patterns": [
    {"first": "cpu_high", "second": "mem_high", "time_diff_seconds": 45}
  ],
  "root_alarm_id": "alarm-001",
  "timestamp": "2026-05-26T10:30:00Z",
  "action": "trigger_suppression_workflow"
}
```

### Integration with Suppression Workflow

1. **Detect** → Storm detection script triggers
2. **Correlate** → Group alarms by resource, service, time
3. **Identify root** → Find earliest alarm or infrastructure-level alarm
4. **Suppress** → Disable non-critical alarms for affected resources
5. **Escalate** → Create incident with root cause analysis
6. **Restore** → Re-enable suppressed alarms after resolution (see Self-Healing flow)

## Cascade Pattern Correlation Algorithm (HIGH-3)

**Purpose**: Identify cascading alarm patterns where one alarm triggers downstream alarms, enabling root cause identification and targeted suppression.

### Cascade Pattern Definition

| Pattern Type | Detection Logic | Root Cause Indicator |
|--------------|-----------------|----------------------|
| **Infrastructure → Application** | SYS.ECS alarm followed by SYS.RDS alarm within 2 min | Infrastructure likely root cause |
| **Network → Service** | SYS.VPC alarm followed by SYS.ELB alarm within 2 min | Network issue upstream |
| **Database → Application** | SYS.RDS alarm followed by app-level metric alarm within 3 min | Database bottleneck |
| **Single Resource Cascade** | Multiple alarms on same resource in sequence | Single resource failure spreading |
| **Cross-Resource Cascade** | Alarm on resource A triggers alarm on dependent resource B | Dependency chain failure |

### Execution — CLI (Cascade Pattern Correlation)

```bash
#!/bin/bash
# Cascade pattern correlation algorithm
# Identifies cascading alarm sequences for root cause analysis

REGION="{{env.HW_REGION_ID}}"
CASCADE_WINDOW_SECONDS=180  # 3 minutes
MIN_SEQUENCE_LENGTH=2

# Step 1: Fetch alarm history for analysis period
ALARM_HISTORY=$(hcloud ces list-alarm-history \
  --region "$REGION" \
  --from "$(date -d '-30 minutes' +%s)000" \
  --to "$(date +%s)000" \
  --output json)

# Step 2: Sort alarms chronologically
SORTED_ALARMS=$(echo "$ALARM_HISTORY" | jq '.alarm_histories | sort_by(.alarm_time)')

# Step 3: Build cascade sequences
# Algorithm: For each alarm, find subsequent alarms within cascade window
# that involve related resources or namespaces

CASCADE_SEQUENCES=$(echo "$SORTED_ALARMS" | jq --argjson window "$CASCADE_WINDOW_SECONDS" --argjson min_len "$MIN_SEQUENCE_LENGTH" '
  # Define namespace hierarchy (infrastructure -> application)
  def namespace_priority(ns):
    if ns | startswith("SYS.ECS") or ns | startswith("SYS.VPC") then 1
    elif ns | startswith("SYS.RDS") or ns | startswith("SYS.DCS") then 2
    elif ns | startswith("SYS.ELB") then 3
    else 4
    end;
  
  # Build sequences
  reduce .[] as $alarm (
    {sequences: [], current_seq: []};
    
    # Check if this alarm continues current sequence
    if (.current_seq | length) > 0 and
       (($alarm.alarm_time | strftime("%s") | tonumber) - 
        ((.current_seq | last).alarm_time | strftime("%s") | tonumber)) < $window and
       (namespace_priority($alarm.metric_namespace) >= namespace_priority((.current_seq | last).metric_namespace))
    then
      # Continue sequence
      .current_seq += [$alarm]
    else
      # Start new sequence
      if (.current_seq | length) >= $min_len then
        .sequences += [.current_seq]
      end |
      .current_seq = [$alarm]
    end
  ) |
  
  # Finalize
  if (.current_seq | length) >= $min_len then
    .sequences += [.current_seq]
  end |
  
  # Format output
  .sequences | map({
    sequence_id: (.[0].alarm_id + "-" + (length | tostring)),
    length: length,
    start_time: .[0].alarm_time,
    end_time: (last.alarm_time),
    duration_seconds: ((last.alarm_time | strftime("%s") | tonumber) - (.[0].alarm_time | strftime("%s") | tonumber)),
    root_alarm: {
      alarm_id: .[0].alarm_id,
      alarm_name: .[0].alarm_name,
      namespace: .[0].metric_namespace,
      resource_id: .[0].resource_id
    },
    downstream_alarms: .[1:] | map({
      alarm_id,
      alarm_name,
      namespace: metric_namespace,
      resource_id,
      time_offset_seconds: ((alarm_time | strftime("%s") | tonumber) - (.[0].alarm_time | strftime("%s") | tonumber))
    }),
    cascade_type: (
      if (.[0].metric_namespace | startswith("SYS.ECS")) and (.[1:].[].metric_namespace | any(startswith("SYS.RDS"))) then "infra_to_db"
      elif (.[0].metric_namespace | startswith("SYS.VPC")) and (.[1:].[].metric_namespace | any(startswith("SYS.ELB"))) then "network_to_lb"
      elif (map(.resource_id) | unique | length) == 1 then "single_resource"
      else "cross_resource"
      end
    ),
    root_cause_probability: (
      if .cascade_type == "infra_to_db" or .cascade_type == "network_to_lb" then 0.9
      elif .cascade_type == "single_resource" then 0.85
      else 0.7
      end
    )
  })
')

# Step 4: Output cascade analysis
SEQUENCE_COUNT=$(echo "$CASCADE_SEQUENCES" | jq 'length')
echo "🔗 Cascade Pattern Analysis"
echo "   Detected: $SEQUENCE_COUNT cascade sequences"

if [ "$SEQUENCE_COUNT" -gt 0 ]; then
  echo "$CASCADE_SEQUENCES" | jq -r '.[] |
    "   Sequence: \(.sequence_id) (\(.length) alarms, \(.duration_seconds)s)\n" +
    "   Root alarm: \(.root_alarm.alarm_name) [\(.root_alarm.namespace)]\n" +
    "   Cascade type: \(.cascade_type) (probability: \(.root_cause_probability))\n" +
    "   Downstream: \(.downstream_alarms | map("\(.alarm_name")") | join(", "))"
  '
  
  # Step 5: Identify highest probability root cause
  TOP_ROOT_CAUSE=$(echo "$CASCADE_SEQUENCES" | jq 'sort_by(-.root_cause_probability) | .[0]')
  ROOT_CAUSE_ALARM_ID=$(echo "$TOP_ROOT_CAUSE" | jq -r '.root_alarm.alarm_id')
  ROOT_CAUSE_ALARM_NAME=$(echo "$TOP_ROOT_CAUSE" | jq -r '.root_alarm.alarm_name')
  PROBABILITY=$(echo "$TOP_ROOT_CAUSE" | jq -r '.root_cause_probability')
  
  echo ""
  echo "🎯 Most Likely Root Cause:"
  echo "   Alarm: $ROOT_CAUSE_ALARM_NAME ($ROOT_CAUSE_ALARM_ID)"
  echo "   Probability: $PROBABILITY"
  echo "   Recommended: Focus diagnosis on this alarm first"
  
  # Step 6: Generate incident report
  jq -n \
    --argjson sequences "$CASCADE_SEQUENCES" \
    --arg top_root_cause_id "$ROOT_CAUSE_ALARM_ID" \
    --arg top_root_cause_name "$ROOT_CAUSE_ALARM_NAME" \
    --argjson top_probability "$PROBABILITY" \
    --arg timestamp "$(date -Iseconds)" \
    '{
      cascade_sequences: $sequences,
      analysis_summary: {
        total_sequences: ($sequences | length),
        top_root_cause: {
          alarm_id: $top_root_cause_id,
          alarm_name: $top_root_cause_name,
          probability: $top_probability
        }
      },
      timestamp: $timestamp,
      recommended_action: "diagnose_root_cause_first"
    }' | tee cascade-analysis-report.json
fi
```

### Cascade Pattern Output Format

```json
{
  "cascade_sequences": [
    {
      "sequence_id": "alarm-001-3",
      "length": 3,
      "start_time": "2026-05-26T10:00:00Z",
      "end_time": "2026-05-26T10:02:30Z",
      "duration_seconds": 150,
      "root_alarm": {
        "alarm_id": "alarm-001",
        "alarm_name": "ecs_cpu_high",
        "namespace": "SYS.ECS",
        "resource_id": "i-abc123"
      },
      "downstream_alarms": [
        {"alarm_id": "alarm-002", "alarm_name": "rds_conn_high", "namespace": "SYS.RDS", "resource_id": "rds-def456", "time_offset_seconds": 45},
        {"alarm_id": "alarm-003", "alarm_name": "app_latency_high", "namespace": "CUSTOM.APP", "resource_id": "app-xyz789", "time_offset_seconds": 120}
      ],
      "cascade_type": "infra_to_db",
      "root_cause_probability": 0.9
    }
  ],
  "analysis_summary": {
    "total_sequences": 1,
    "top_root_cause": {
      "alarm_id": "alarm-001",
      "alarm_name": "ecs_cpu_high",
      "probability": 0.9
    }
  },
  "timestamp": "2026-05-26T10:30:00Z",
  "recommended_action": "diagnose_root_cause_first"
}
```

### Cascade Pattern Types Reference

| Cascade Type | Pattern | Root Cause Probability | Diagnostic Priority |
|--------------|---------|------------------------|---------------------|
| `infra_to_db` | ECS → RDS | 0.9 | Diagnose ECS first |
| `network_to_lb` | VPC → ELB | 0.9 | Diagnose VPC first |
| `infra_to_app` | ECS → Custom App | 0.85 | Diagnose ECS first |
| `db_to_app` | RDS → Custom App | 0.85 | Diagnose RDS first |
| `single_resource` | Multiple alarms same resource | 0.85 | Single resource diagnosis |
| `cross_resource` | Multiple resources affected | 0.7 | Dependency mapping required |

### Integration with Cross-Skill Delegation

When cascade pattern detected, delegate diagnosis to appropriate skills:

| Cascade Type | Primary Diagnosis Skill | Secondary Skill |
|--------------|------------------------|-----------------|
| `infra_to_db` | huaweicloud-ecs-ops | huaweicloud-rds-ops |
| `network_to_lb` | huaweicloud-vpc-ops | huaweicloud-elb-ops |
| `infra_to_app` | huaweicloud-ecs-ops | huaweicloud-aom-ops |
| `db_to_app` | huaweicloud-rds-ops | huaweicloud-aom-ops |
| `single_resource` | Resource-specific skill | huaweicloud-ces-ops (metrics) |
| `cross_resource` | Multi-skill parallel diagnosis | Orchestrated via CES skill |
