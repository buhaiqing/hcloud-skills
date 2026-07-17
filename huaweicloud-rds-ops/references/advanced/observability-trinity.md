# Observability Trinity — Huawei Cloud RDS

> **Purpose**: Metrics → Logs → Traces linkage rules for RDS for MySQL/PostgreSQL/SQL Server.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

## 1. Observability Trinity Overview

| Component | Data Source | Purpose |
|-----------|-------------|---------|
| Metrics | CES (SYS.RDS) | CPU%, memory%, connections, slow queries, disk I/O |
| Logs | LTS (slow query log, error log, audit log) | Query performance, errors, lock waits |
| Traces | APM (RDS connection traces) | SQL execution flow (when DAS enabled) |

## 2. Linkage Rules

### 2.1 Metric → Log Linkage

| When RDS metric alerts | Check LTS logs |
|------------------------|----------------|
| `rds001_cpu_util` > 90% | Error log, slow query log (CPU-intensive queries) |
| `rds003_conn_usage` > 90% | Error log for connection pool exhaustion |
| `rds039_disk_usage` > 85% | Error log for disk full warnings |
| `rds044_innodb_row_lock_waits` spike | Error log for lock wait timeouts |
| `rds048_replica_lag` > 10s | Error log for replication errors |
| `rds049_slow_queries` > 50/min | Slow query log analysis |

### 2.2 Log → Metric Linkage

| When LTS log pattern detected | Check CES metrics |
|------------------------------|-------------------|
| `Deadlock found` | `rds001_cpu_util`, transaction throughput |
| `Lock wait timeout` | `rds044_innodb_row_lock_waits` |
| `Connection refused` | `rds003_conn_usage`, connection pool |
| `Table is full` | `rds039_disk_usage` |
| `Slow query` (> 1s) | `rds049_slow_queries`, query latency |
| `Replication error` | `rds048_replica_lag` |
| `InnoDB: cannot allocate memory` | `rds039_disk_usage`, memory metrics |

### 2.3 Trace → Metric/Log Linkage

| When APM/DAS trace shows | Check metrics + logs |
|-------------------------|---------------------|
| SQL execution > 5s | `rds001_cpu_util`, slow query log |
| Lock wait in trace | `rds044_innodb_row_lock_waits`, error log |
| Connection timeout | `rds003_conn_usage`, connection metrics |
| Full table scan | `rds007_qps`, slow query log |

## 3. Data Source Mapping

| Observable | CES Namespace | LTS Log Type | APM Trace |
|-----------|--------------|--------------|-----------|
| RDS CPU | SYS.RDS | Error log, slow query log | Via DAS |
| RDS connections | SYS.RDS | Error log | Via DAS |
| RDS slow queries | SYS.RDS | Slow query log | Via DAS |
| RDS replication | SYS.RDS | Error log (replica) | No |
| RDS disk I/O | SYS.RDS | Error log, audit log | No |
| RDS audit | SYS.RDS | Audit log | No |

## 4. Correlation Query Examples

### 4.1 Metric Alert → Find Related Logs

```bash
# Slow query storm on RDS instance
INSTANCE_ID="{{user.instance_id}}"
REGION="{{env.HW_REGION_ID}}"
LOG_GROUP="{{user.rds_log_group_id}}"

# 1. Confirm metric alert
hcloud ces query-metric-data \
  --namespace "SYS.RDS" \
  --metric-name "rds049_slow_queries" \
  --dimension "instance_id=$INSTANCE_ID" \
  --period 60 \
  --from "$(( $(date +%s) - 300 ))" \
  --to "$(date +%s)" \
  --output json

# 2. Query LTS slow query log
hcloud lts query-log \
  --log-group-id "$LOG_GROUP" \
  --log-stream-id "slow_query_log" \
  --start-time "$(( $(date +%s) * 1000 - 30 * 60 * 1000 ))" \
  --end-time "$(date +%s)" \
  --keywords "Query_time|SELECT|UPDATE" \
  --output json
```

### 4.2 Log Pattern → Find Related Metrics

```bash
# Deadlock detected in error log
INSTANCE_ID="{{user.instance_id}}"
REGION="{{env.HW_REGION_ID}}"

# Query lock wait and CPU metrics
hcloud ces query-metric-data \
  --namespace "SYS.RDS" \
  --metric-name "rds044_innodb_row_lock_waits" \
  --dimension "instance_id=$INSTANCE_ID" \
  --period 60 \
  --from "$(( $(date +%s) - 600 ))" \
  --to "$(date +%s)" \
  --output json

hcloud ces query-metric-data \
  --namespace "SYS.RDS" \
  --metric-name "rds001_cpu_util" \
  --dimension "instance_id=$INSTANCE_ID" \
  --period 60 \
  --from "$(( $(date +%s) - 600 ))" \
  --to "$(date +%s)" \
  --output json
```

### 4.3 Trace → Metric Correlation

```bash
# Slow SQL trace → check RDS metrics
TRACE_ID="{{user.trace_id}}"
INSTANCE_ID="{{user.instance_id}}"

# Get trace details from DAS
hcloud das query-sql-trace \
  --trace-id "$TRACE_ID" \
  --output json

# Query RDS metrics for that instance
hcloud ces query-metric-data \
  --namespace "SYS.RDS" \
  --metric-name "rds001_cpu_util" \
  --dimension "instance_id=$INSTANCE_ID" \
  --period 60 \
  --from "$(( $(date +%s) - 300 ))" \
  --to "$(date +%s)" \
  --output json
```

## 5. Trinity-Driven Diagnosis Workflow

```
[RDS Metric Alert: rds049_slow_queries > 50/min]
    │
    ├── 1. Query LTS: slow query log for top slow queries
    │   └── hcloud lts query-log --log-stream-id "slow_query_log"
    │
    ├── 2. Query APM/DAS: SQL execution traces for the slow queries
    │   └── hcloud das query-sql-trace --instance-id "$INSTANCE_ID"
    │
    └── 3. Correlate:
        ├── If full table scan →建议添加索引
        ├── If large JOIN →优化SQL
        ├── If lock wait →检查rds044_innodb_row_lock_waits
        └── If CPU high →检查rds001_cpu_util
```

## 6. Cross-Service Linkage

| RDS symptom | Downstream check |
|-------------|------------------|
| RDS CPU spike | ECS: `cpu_util` (if app on ECS) |
| RDS connection exhaustion | ECS: connection pool metrics |
| RDS disk full | OBS: backup bucket (if backup to OBS) |
| RDS replica lag | ECS: network metrics (if replica on ECS) |
| RDS slow queries | ELB: request latency (if app on ECS behind ELB) |

## 7. Compliance Checklist

- [x] Metrics → Logs linkage defined (SYS.RDS → LTS error/slow query log)
- [x] Logs → Metrics linkage defined (deadlock, lock timeout, slow queries)
- [x] Trace → Metric/Log linkage defined (SQL trace → slow query log + metrics)
- [x] Data source mapping documented (CES namespace → LTS log type → APM/DAS)
- [x] Correlation query examples provided (3 CLI examples)
- [x] Cross-service linkage defined (RDS → ECS/OBS/ELB)
