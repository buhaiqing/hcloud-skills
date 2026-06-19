# CES AIOps Patterns — Huawei Cloud Cloud Eye

> Advanced AIOps patterns for Cloud Eye (CES) — load only when the agent needs
> anomaly RCA, alarm-storm suppression, or cross-metric correlation work.

## 1. Multi-Metric Correlation Patterns

### 1.1 CPU + Memory joint spike
- Trigger: `cpu_util > 90%` AND `memory_util > 85%` within 5 min
- Suspected root cause: memory leak + JVM full GC; or noisy neighbour
- Action: scale out, then capture `/proc/meminfo` + JVM heap dump

### 1.2 Disk IO + Queue length
- Trigger: `disk_io_util > 80%` AND `disk_read/write_queue > 10`
- Suspected root cause: storage hotspot; replication backlog
- Action: rebalance shards, check kernel `iostat -x`

### 1.3 Network in/out divergence
- Trigger: `network_incoming > 3×` baseline AND `network_outgoing ≈ baseline`
- Suspected root cause: external pull / DDoS / data scrape
- Action: WAF / Anti-DDoS routing + throttling

### 1.4 Alarm storm aggregation
- Trigger: > 10 alarms for same `resource_id` within 5 min
- Action: collapse into a single `summary_alarm`, suppress duplicates,
  emit one CES notification per coalescing window

## 2. Knowledge-Base Patterns (≥5 entries)

| Pattern | Symptoms | First-line diagnostic | Mitigation |
|---------|----------|----------------------|------------|
| DB connection exhaustion | `rds_conn_usage > 90%` | check `performance_schema.threads_connected` | restart service, increase max_connections |
| Disk full | `disk_util > 95%` | `df -h` + `du -sh /*` | expand EVS, archive to OBS |
| API throttling | `apig_throttled_count > 0` | `apig:list_apis` quotas | request quota increase |
| Network partition | `network_outgoing = 0` | `ping` + traceroute | failover EIP, check VPC route |
| Memory leak | memory slope > 0.5%/min sustained | heap dump, GC logs | restart with rolling deploy |

> **Security-Sensitive**: alarm actions that mutate resources (auto-stop,
> auto-reboot) MUST NOT be enabled without explicit operator approval and a
> documented blast-radius. Threshold breaches trigger HALT unless
> `{{user.allow_auto_remediation}}` is set.