# CES Monitoring & Self-Monitoring — Huawei Cloud Cloud Eye Service

## CES Metrics for the Service Itself

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

## Multi-Metric Anomaly Detection

### Detection Criteria

> **Canonical pattern registry**: see [`references/advanced/anomaly-patterns.md`](advanced/anomaly-patterns.md) — single source of truth for pattern names, thresholds, and cross-skill metric mapping.

| Pattern | Metrics | Threshold | Severity |
|---------|---------|-----------|----------|
| cpu_mem_dual_high | cpu_util + memory_util | cpu > 90% AND mem > 85% | Critical |
| disk_io_bottleneck | disk_read/write_bytes_rate + disk_util | I/O spike AND util > 90% | Warning |
| mem_leak_trend | memory_util slope (30min) | slope > 0.5%/min | Critical |
| sudden_cpu_spike | cpu_util delta (5min) | delta > 50% | Warning |
| network_saturation | network_in/out_bytes_rate | > 90% of bandwidth | Critical |
| rds_connection_exhaustion | rds003_conn_usage + rds007_qps | conn > 90% AND qps drops | Critical |

### Core Detection Logic (CLI snippets)

Each pattern follows: **query metric → extract values → compare threshold → output JSON result**.

```bash
REGION="{{env.HW_REGION_ID}}"
INSTANCE_ID="{{user.instance_id}}"
from_ts=$(($(date +%s) - 600))000; to_ts=$(date +%s)000

# Pattern 1: cpu_mem_dual_high — Critical
cpu=$(hcloud ces query-metric-data --region "$REGION" --namespace SYS.ECS --metric-name cpu_util \
  --dimension "instance_id:$INSTANCE_ID" --from "$from_ts" --to "$to_ts" --period 60 --output json | jq -r '.datapoints[0].value')
mem=$(hcloud ces query-metric-data --region "$REGION" --namespace AGT.ECS --metric-name memory_util \
  --dimension "instance_id:$INSTANCE_ID" --from "$from_ts" --to "$to_ts" --period 60 --output json | jq -r '.datapoints[0].value')
detected=$(echo "$cpu $mem" | awk '{print ($1>90 && $2>85) ? "true" : "false"}')

# Pattern 2: disk_io_bottleneck — Warning
read_rate=$(hcloud ces query-metric-data ... --metric-name disk_read_bytes_rate ... | jq -r '.datapoints[0].value')
write_rate=$(hcloud ces query-metric-data ... --metric-name disk_write_bytes_rate ... | jq -r '.datapoints[0].value')
disk_util=$(hcloud ces query-metric-data ... --namespace AGT.ECS --metric-name disk_util ... | jq -r '.datapoints[0].value')
# I/O spike: total bytes/s > {{user.io_threshold_bytes}} AND disk_util > 90

# Pattern 3: mem_leak_trend — Critical
# Query 30min window, period=300 (6 datapoints), calculate linear slope via jq
# slope_per_min = slope / 5; detected if slope_per_min > 0.5

# Pattern 4: sudden_cpu_spike — Warning
# Compare datapoints[0] (current) vs datapoints[5] (5min ago)
# detected if |current - prev| > 50

# Pattern 5: network_saturation — Critical
in_rate=$(hcloud ces query-metric-data ... --metric-name network_in_bytes_rate ... | jq -r '.datapoints[0].value')
out_rate=$(hcloud ces query-metric-data ... --metric-name network_out_bytes_rate ... | jq -r '.datapoints[0].value')
# detected if in_rate > {{user.bandwidth_limit}} * 0.9 OR out_rate > {{user.bandwidth_limit}} * 0.9

# Pattern 6: rds_connection_exhaustion — Critical
conn=$(hcloud ces query-metric-data --namespace SYS.RDS --metric-name rds003_conn_usage ... | jq -r '.datapoints[0].value')
qps=$(hcloud ces query-metric-data --namespace SYS.RDS --metric-name rds007_qps ... | jq -r '.datapoints[0].value')
prev_qps=$(... | jq -r '.datapoints[5].value')
# detected if conn > 90 AND current_qps < prev_qps * 0.8
```

> **TE-1 note**: `{{user.io_threshold_bytes}}` and `{{user.bandwidth_limit}}` must be provided by the user or queried via API. Do not hardcode values like `104857600` or `125000000`.

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

### Storm Detection Logic (CLI)

```bash
REGION="{{env.HW_REGION_ID}}"
WINDOW_MINUTES=5; THRESHOLD=10; SAME_RESOURCE_THRESHOLD=3

ALARM_EVENTS=$(hcloud ces list-alarm-history --region "$REGION" \
  --from "$(date -d "-15 minutes" +%s)000" --to "$(date +%s)000" --output json)

# Count alarms in storm window
WINDOW_START=$(date -d "-${WINDOW_MINUTES} minutes" +%s)
COUNT=$(echo "$ALARM_EVENTS" | jq --arg s "$WINDOW_START" \
  '[.alarm_histories[] | select((.alarm_time | strftime("%s") | tonumber) > ($s | tonumber))] | length')

if [ "$COUNT" -ge "$THRESHOLD" ]; then
  # Resource spam: group by resource_id, filter count > threshold
  # Namespace dominance: group by metric_namespace, find >50%
  # Cascade: sort by alarm_time, find pairs within 120s
  # → trigger suppression workflow
fi
```

### Storm Handling Flow

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

## Idle Alarm Detection

Identifies alarm rules that have not triggered recently, indicating monitoring gaps or over-monitoring.

### Detection Criteria

| Criterion | Threshold | Interpretation |
|-----------|-----------|----------------|
| No triggers in 90 days | `trigger_count == 0` | Alarm may be misconfigured |
| No triggers in 30 days | `trigger_count == 0` | Consider disabling if non-critical |
| Last trigger > 60 days ago | `last_trigger_time` | Review threshold |
| Never triggered since creation | `trigger_count == 0 && age > 7 days` | High priority review |

### Detection Logic (CLI)

```bash
REGION="{{env.HW_REGION_ID}}"
ALL_ALARMS=$(hcloud ces list-alarms --region "$REGION" --output json)

# Find idle alarms: never triggered OR last trigger > 90 days ago
IDLE=$(echo "$ALL_ALARMS" | jq --argjson threshold 90 '
  .alarms | map(select(
    (.trigger_count == 0) or
    ((now - (.last_trigger_time | strftime("%s") | tonumber)) / 86400 > $threshold)
  )) | map({alarm_id, alarm_name, metric_namespace, trigger_count, idle_days, recommendation})
')
```

### Post-Detection Actions

| Category | Action | Priority |
|----------|--------|----------|
| Never triggered (critical alarm) | Verify threshold, resource existence, metric namespace | High |
| Never triggered (non-critical) | Consider disabling; reduce quota usage | Medium |
| Long idle (>90 days) | Review threshold appropriateness | Medium |
| Resource deleted but alarm exists | Delete alarm | Required |

## Cascade Pattern Correlation

Identifies cascading alarm sequences where one alarm triggers downstream alarms, enabling root cause identification.

### Cascade Pattern Types

| Pattern Type | Detection Logic | Root Cause Indicator |
|--------------|-----------------|----------------------|
| Infrastructure → Application | SYS.ECS alarm followed by SYS.RDS alarm within 2 min | Infrastructure likely root cause |
| Network → Service | SYS.VPC alarm followed by SYS.ELB alarm within 2 min | Network issue upstream |
| Database → Application | SYS.RDS alarm followed by app-level metric alarm within 3 min | Database bottleneck |
| Single Resource Cascade | Multiple alarms on same resource in sequence | Single resource failure |
| Cross-Resource Cascade | Alarm on resource A triggers alarm on dependent resource B | Dependency chain failure |

### Correlation Logic (CLI)

```bash
REGION="{{env.HW_REGION_ID}}"
CASCADE_WINDOW=180  # seconds

# Fetch sorted alarm history
ALARMS=$(hcloud ces list-alarm-history --region "$REGION" \
  --from "$(date -d '-30 minutes' +%s)000" --to "$(date +%s)000" --output json \
  | jq '.alarm_histories | sort_by(.alarm_time)')

# Build sequences: for each alarm, find subsequent alarms within cascade window
# that involve related namespaces (priority: ECS/VPC > RDS/DCS > ELB > custom)
# Output: sequence_id, root_alarm, downstream_alarms, cascade_type, root_cause_probability
```

### Cascade Type Reference

| Cascade Type | Pattern | Root Cause Probability | Diagnostic Priority |
|--------------|---------|------------------------|---------------------|
| `infra_to_db` | ECS → RDS | 0.9 | Diagnose ECS first |
| `network_to_lb` | VPC → ELB | 0.9 | Diagnose VPC first |
| `infra_to_app` | ECS → Custom App | 0.85 | Diagnose ECS first |
| `db_to_app` | RDS → Custom App | 0.85 | Diagnose RDS first |
| `single_resource` | Multiple alarms same resource | 0.85 | Single resource diagnosis |
| `cross_resource` | Multiple resources affected | 0.7 | Dependency mapping required |

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
