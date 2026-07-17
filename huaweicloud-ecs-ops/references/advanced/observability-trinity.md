# Observability Trinity — Huawei Cloud ECS

> **Purpose**: Metrics → Logs → Traces linkage rules for ECS.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

## 1. Observability Trinity Overview

| Component | Data Source | Purpose |
|-----------|-------------|---------|
| Metrics | CES (SYS.ECS, AGT.ECS) | CPU%, memory%, disk I/O, network |
| Logs | LTS (instance system logs, application logs) | Process events, errors, OOM |
| Traces | APM (Application Performance Management) | Request flow across ECS services |

## 2. Linkage Rules

### 2.1 Metric → Log Linkage

| When CES metric alerts | Check LTS logs |
|------------------------|----------------|
| `cpu_util` > 90% | System process logs, `top` output |
| `mem_usedPercent` leak | Application memory allocation logs |
| `diskUsage_percent` > 90% | Large file search in `/var/log/` |
| `load1` > vCPU count | System audit log for process explosion |
| `net_bits` > bandwidth * 0.9 | Network interface error logs |
| `read_iops`/`write_iops` spike | Disk I/O subsystem logs |

### 2.2 Log → Metric Linkage

| When LTS log pattern detected | Check CES metrics |
|------------------------------|-------------------|
| OOM errors (`killed`, `oom`) | `mem_usedPercent`, `memory_util` |
| Connection timeouts | `tcp_curr_estab`, connection metrics |
| Disk full (`no space left`) | `diskUsage_percent`, `disk_util` |
| Process crash (`segfault`) | `cpu_util`, process count |
| SSH brute force | Network `net_pps` metrics |
| ` Too many open files` | Process file descriptor count |

### 2.3 Trace → Metric/Log Linkage

| When APM trace shows | Check metrics + logs |
|---------------------|---------------------|
| Span duration > 500ms | CPU/memory of specific ECS instance |
| Error in trace | Error logs of application on that ECS |
| Timeout in trace | Downstream service health metrics |
| Database call latency | RDS metrics (if ECS hosts DB) |

## 3. Data Source Mapping

| Observable | CES Namespace | LTS Log Group | APM Trace |
|-----------|--------------|---------------|-----------|
| ECS instance system | SYS.ECS | `{{user.instance_name}}-syslog` | Yes |
| ECS application | AGT.ECS | `{{user.instance_name}}-applog` | Yes |
| ECS network | SYS.VPC | `{{user.instance_name}}-netlog` | Via ELB |
| ECS disk I/O | SYS.ECS, AGT.ECS | `{{user.instance_name}}-syslog` | No |

## 4. Correlation Query Examples

### 4.1 Metric Alert → Find Related Logs

```bash
# CPU spike on ECS instance
INSTANCE_ID="{{user.instance_id}}"
REGION="{{env.HW_REGION_ID}}"
LOG_GROUP="{{user.ecs_log_group_id}}"
LOG_STREAM="{{user.ecs_log_stream_id}}"

# 1. Query CPU metric to confirm alert
hcloud ces query-metric-data \
  --namespace "SYS.ECS" \
  --metric-name "cpu_util" \
  --dimension "instance_id=$INSTANCE_ID" \
  --period 60 \
  --from "$(( $(date +%s) - 300 ))" \
  --to "$(date +%s)" \
  --output json

# 2. Query LTS for process logs around alert time
hcloud lts query-log \
  --log-group-id "$LOG_GROUP" \
  --log-stream-id "$LOG_STREAM" \
  --start-time "$(( $(date +%s) * 1000 - 30 * 60 * 1000 ))" \
  --end-time "$(date +%s)" \
  --keywords "cpu|process|top" \
  --output json
```

### 4.2 Log Pattern → Find Related Metrics

```bash
# OOM detected in logs
INSTANCE_ID="{{user.instance_id}}"
REGION="{{env.HW_REGION_ID}}"

# Query memory metrics
hcloud ces query-metric-data \
  --namespace "AGT.ECS" \
  --metric-name "memory_util" \
  --dimension "instance_id=$INSTANCE_ID" \
  --period 60 \
  --from "$(( $(date +%s) - 600 ))" \
  --to "$(date +%s)" \
  --output json
```

### 4.3 Trace → Metric Correlation

```bash
# High latency trace → check ECS metrics
TRACE_ID="{{user.trace_id}}"
INSTANCE_ID="{{user.instance_id}}"

# Get span details from APM
hcloud apm query-trace \
  --trace-id "$TRACE_ID" \
  --output json

# Query ECS metrics for that instance
hcloud ces query-metric-data \
  --namespace "SYS.ECS" \
  --metric-name "cpu_util" \
  --dimension "instance_id=$INSTANCE_ID" \
  --period 60 \
  --from "$(( $(date +%s) - 300 ))" \
  --to "$(date +%s)" \
  --output json
```

## 5. Trinity-Driven Diagnosis Workflow

```
[CES Metric Alert: cpu_util > 90%]
    │
    ├── 1. Query LTS: process logs during alert window
    │   └── hcloud lts query-log --keywords "cpu|top|process"
    │
    ├── 2. Query APM: traces with high span duration on this instance
    │   └── hcloud apm query-trace --instance-id "$INSTANCE_ID"
    │
    └── 3. Correlate:
        ├── If unknown process in logs → HSS scan + kill
        ├── If Java/Node process → heap/profile analysis
        └── If OOM → memory metric confirm + restart
```

## 6. Cross-Service Linkage

| ECS symptom | Downstream check |
|-------------|------------------|
| ECS CPU spike | RDS: `rds001_cpu_util` (if DB on ECS) |
| ECS disk full | OBS: bucket usage (if OBS mounted) |
| ECS network saturation | ELB: `l7e_listener_qps`, backend health |
| ECS connection timeout | RDS: `rds003_conn_usage`, DCS: `redis_connections` |

## 7. Compliance Checklist

- [x] Metrics → Logs linkage defined (SYS.ECS + AGT.ECS)
- [x] Logs → Metrics linkage defined (OOM, timeouts, disk full)
- [x] Trace → Metric/Log linkage defined (span duration → instance metrics)
- [x] Data source mapping documented (CES namespace → LTS group → APM)
- [x] Correlation query examples provided (3 CLI examples)
- [x] Cross-service linkage defined (ECS → RDS/OBS/ELB)
