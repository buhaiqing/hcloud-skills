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
