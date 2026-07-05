# Anomaly Pattern Registry — Canonical Source

> **Single source of truth** for cross-skill anomaly detection patterns. All skills
> reference this file for pattern names, thresholds, and severity. Product-specific
> metric names are listed per skill below.

## Pattern Definitions

| # | Pattern Name | Description | Canonical Threshold | Severity | Affected Skills |
|---|-------------|-------------|---------------------|----------|-----------------|
| P1 | `cpu_mem_dual_high` | CPU and memory both high simultaneously | cpu > 90% AND mem > 85% within 5 min | Critical | CES, ECS, CCE |
| P2 | `disk_io_bottleneck` | Disk I/O saturation with high utilization | I/O rate spike AND disk_util > 90% | Warning | CES, ECS |
| P3 | `mem_leak_trend` | Memory utilization monotonically increasing | slope > 0.5%/min over 30 min | Critical | CES, ECS |
| P4 | `sudden_cpu_spike` | Rapid CPU utilization change | delta(5min) > 50% | Warning | CES, ECS |
| P5 | `network_saturation` | Network bandwidth near capacity | inbound OR outbound > 90% of bandwidth | Critical | CES, ECS |
| P6 | `rds_connection_exhaustion` | Database connection pool depleted | conn_usage > 90% AND qps drops > 20% | Critical | CES, RDS |
| P7 | `disk_fill_acceleration` | Disk fill rate accelerating | fill_rate(half2) > fill_rate(half1) in 1h window | Critical | ECS |
| P8 | `alarm_storm` | High-frequency alarm events | > 10 alarms / 5 min on same resource | Critical | CES, ECS |

## Product-Specific Metric Mapping

### ECS (SYS.ECS / AGT.ECS)

| Pattern | ECS Metric Name | ECS Namespace |
|---------|----------------|---------------|
| P1 cpu_mem_dual_high | `cpu_util` + `mem_usedPercent` | SYS.ECS + AGT.ECS |
| P2 disk_io_bottleneck | `read_iops` + `write_iops` + `diskUsage_percent` | SYS.ECS + AGT.ECS |
| P3 mem_leak_trend | `mem_usedPercent` | AGT.ECS |
| P4 sudden_cpu_spike | `cpu_util` | SYS.ECS |
| P5 network_saturation | `net_bits` + `net_pps` | SYS.ECS |
| P7 disk_fill_acceleration | `diskUsage_percent` | AGT.ECS |

### CES Generic (cross-product)

| Pattern | CES Metric Name | CES Namespace |
|---------|----------------|---------------|
| P1 cpu_mem_dual_high | `cpu_util` + `memory_util` | SYS.ECS (varies by product) |
| P2 disk_io_bottleneck | `disk_read_bytes_rate` + `disk_util` | SYS.ECS + AGT.ECS |
| P3 mem_leak_trend | `memory_util` | AGT.ECS |
| P5 network_saturation | `network_in_bytes_rate` + `network_out_bytes_rate` | SYS.ECS |
| P6 rds_connection_exhaustion | `rds003_conn_usage` + `rds007_qps` | SYS.RDS |
| P8 alarm_storm | alarm history events | CES |

## Threshold Notes

- **P1 cpu_mem_dual_high**: CPU threshold is 90% (canonical). ECS skill may use 80% for more sensitive detection — document deviation if so.
- **P5 network_saturation**: Threshold depends on `{{user.bandwidth_limit}}`. Do NOT hardcode bandwidth values.
- **P8 alarm_storm**: Window is 5 minutes. Same-resource threshold is 3 alarms.
