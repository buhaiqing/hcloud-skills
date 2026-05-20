# CES Core Concepts — Huawei Cloud Cloud Eye Service

## Architecture

Cloud Eye Service (CES) is Huawei Cloud's unified monitoring platform that collects, stores, and analyzes metric data from cloud services. It operates in a multi-tenant architecture where:

1. **Data Collection**: Cloud services (ECS, RDS, VPC, ELB, etc.) automatically push metrics to CES. Custom metrics can be pushed via API/Agent.
2. **Data Storage**: Metrics are stored in time-series databases with configurable retention periods.
3. **Alarm Engine**: Evaluates alarm rules against incoming metrics using sliding window evaluations.
4. **Notification**: Triggers alerts through SMN (Simple Message Notification) topics when alarm conditions are met.

## Metric Namespace Conventions

| Namespace Pattern | Description | Examples |
|-------------------|-------------|----------|
| `SYS.*` | Huawei Cloud system metrics | SYS.ECS, SYS.RDS, SYS.VPC, SYS.ELB |
| `SERVICE.*` | Third-party service metrics | SERVICE.custom-app |
| `AGT.*` | Agent-collected metrics | AGT.ECS (requires agent on host) |

## Core Metrics Reference

### SYS.ECS (Elastic Cloud Server)
| Metric Name | Unit | Description |
|-------------|------|-------------|
| cpu_util | % | CPU utilization percentage |
| memory_util | % | Memory utilization percentage (requires agent) |
| disk_util | % | Disk utilization percentage (requires agent) |
| disk_read_bytes_rate | Bytes/s | Disk read throughput |
| disk_write_bytes_rate | Bytes/s | Disk write throughput |
| network_in_bytes_rate | Bytes/s | Network inbound traffic |
| network_out_bytes_rate | Bytes/s | Network outbound traffic |

### SYS.RDS (Relational Database Service)
| Metric Name | Unit | Description |
|-------------|------|-------------|
| rds001_cpu_util | % | CPU utilization |
| rds002_mem_util | % | Memory utilization |
| rds003_conn_usage | % | Connection utilization |
| rds004_iops_util | % | IOPS utilization |
| rds007_qps | Count/s | Queries per second |
| rds008_tps | Count/s | Transactions per second |

### SYS.VPC (Virtual Private Cloud)
| Metric Name | Unit | Description |
|-------------|------|-------------|
| bandwidth_util | % | Bandwidth utilization |
| eip_bandwidth_out | Bytes/s | EIP outbound bandwidth |
| eip_bandwidth_in | Bytes/s | EIP inbound bandwidth |

## Alarm Rule Anatomy

- **Metric Namespace**: Identifies the service (`SYS.ECS`)
- **Metric Name**: Specific metric within the namespace (`cpu_util`)
- **Dimension**: Resource identifier (`instance_id`, `instance_name`)
- **Comparison Operator**: `GT` (greater than), `GTE` (≥), `LT` (<), `LTE` (≤), `EQ` (=)
- **Threshold**: Numeric value to compare against
- **Evaluation Periods**: Consecutive periods that must exceed threshold (default: 3)
- **Period**: Data point interval in seconds (1, 5, 20, 60, 300)
- **Alarm Level**: Severity level (1: critical, 2: major, 3: minor, 4: info)
- **Notification Topic**: SMN Topic URN for alert delivery

## Alarm States

| State | Description |
|-------|-------------|
| `ok` | Metrics within normal thresholds |
| `alarm` | Threshold violation detected |
| `insufficient_data` | Not enough data points for evaluation |

## Data Granularity and Retention

| Granularity | Aggregation Methods | Availability |
|-------------|---------------------|--------------|
| 1 second | MAX, MIN, AVG | Only for services that support raw data |
| 5 minutes | MAX, MIN, AVG, SUM, VAR | Standard |
| 20 minutes | MAX, MIN, AVG, SUM, VAR | Standard |
| 1 hour | MAX, MIN, AVG, SUM, VAR | Standard |

| Retention Period | Default |
|------------------|---------|
| Raw data (≤ 5 min granularity) | 7 days |
| Aggregated data (5 min) | 30 days |
| Aggregated data (1 hour) | 365 days |

## Limits and Quotas

| Resource | Default Limit | Notes |
|----------|--------------|-------|
| Alarm rules per project | 1,000 | Adjustable via ticket |
| Dashboards per project | 50 | |
| Custom dashboards | 100 panels per dashboard | |
| Custom metrics per project | 500 | |
| Metric data query range | 90 days max | |
| API rate limit | 200 req/min | Per project |

## Dependency Graph

```
Cloud Service → Metric Push → CES Storage → Alarm Engine → SMN Topic → Notification
                                                                    ↑
User Request → API / CLI / Console → CES API
```

## SPOF Analysis

- **Alarms are region-scoped**: An alarm rule only monitors resources in its configured region.
- **Cross-region redundancy**: For critical resources, create duplicate alarm rules in each region.
- **Notification depends on SMN**: If SMN topic is deleted or misconfigured, alarms won't deliver notifications.
