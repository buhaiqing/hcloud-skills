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

## Multi-Metric Anomaly Patterns

| Pattern | Metrics Involved | Detection Logic | Severity | Interpretation |
|---------|-----------------|-----------------|----------|----------------|
| cpu_mem_dual_high | SYS.ECS > cpu_util + AGT.ECS > memory_util | cpu > 90% AND mem > 85% | Critical | 资源双高压，可能OOM |
| disk_io_bottleneck | SYS.ECS > disk_read_bytes_rate + disk_write_bytes_rate + AGT.ECS > disk_util | I/O rate spike AND diskUtil > 90% | Warning | 磁盘IO瓶颈 |
| mem_leak_trend | AGT.ECS > memory_util (30min slope) | slope > 0.5%/min continuously | Critical | 内存泄漏趋势 |
| sudden_cpu_spike | SYS.ECS > cpu_util | delta(5min) > 50% | Warning | 突发性CPU飙升 |
| network_saturation | SYS.ECS > network_in_bytes_rate + network_out_bytes_rate | inbound/outbound > 90% of bandwidth limit | Critical | 网络带宽饱和 |
| rds_connection_exhaustion | SYS.RDS > rds003_conn_usage + SYS.RDS > rds007_qps | conn > 90% AND qps drops | Critical | 数据库连接池耗尽 |

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
