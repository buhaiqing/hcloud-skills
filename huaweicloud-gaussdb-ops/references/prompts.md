# Prompts — Huawei Cloud GaussDB

> **Purpose:** Structured prompts for GaussDB AIOps operations.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 Instance Health Check
```
Analyze GaussDB instance {{resource_id}} health status:
- Current metric values: CPU {{cpu_usage}}%, Memory {{mem_usage}}%, Connection count {{conn_count}}/{{max_conn}}
- Disk usage: {{disk_usage}}%, Transaction log usage: {{tlog_usage}}%
- Recent alert history: {{alert_count}} alerts in past {{time_window}}
- Replication status: {{replication_status}}
Determine if instance is healthy and recommend actions.

Applicable CES metrics: SYS.GaussDB.cpu_usage, SYS.GaussDB.memory_usage, SYS.GaussDB.conn_count, SYS.GaussDB.disk_usage, AGT.GaussDB.transaction_log_size
```

### 1.2 Primary-Standby Switchover Analysis
```
GaussDB instance {{resource_id}} primary-standby switchover detected:
- Switchover time: {{switchover_time}}
- Previous primary: {{old_primary}}, New primary: {{new_primary}}
- Impact duration: {{impact_duration}} seconds
- Connection interruption: {{conn_interruption}}
- Data sync status: {{sync_status}}
Analyze switchover cause and validate data consistency.

GaussDB switchover triggers: manual, HA health check failure, zone outage, resource exhaustion
```

### 1.3 Connection Pool Exhaustion Diagnosis
```
GaussDB connection pool issue on {{resource_id}}:
- Current connections: {{current_conn}}/{{max_conn}} ({{utilization}}% utilized)
- Connection creation rate: {{conn_create_rate}}/s
- Connection error rate: {{conn_error_rate}}/s
- Active transactions: {{active_tx}}
- Waiting queries: {{waiting_queries}}
Diagnose root cause and recommend connection pool tuning.

GaussDB connection limits: max_connections parameter, session memory overhead
```

### 1.4 Storage Space Full Alert
```
GaussDB storage issue on {{resource_id}}:
- Current disk usage: {{disk_usage}}%
- Data disk: {{data_disk_used}}GB / {{data_disk_total}}GB
- Transaction log: {{tlog_used}}GB / {{tlog_total}}GB
- WAL size: {{wal_size}}GB
- Growth rate: {{growth_rate}}GB/day
Assess severity and recommend cleanup or scaling actions.

GaussDB storage: data files, WAL logs, temporary files, system catalogs
```

### 1.5 CPU Overload Diagnosis
```
GaussDB CPU overload on {{resource_id}}:
- CPU usage: {{cpu_usage}}% (baseline: {{baseline_cpu}}%)
- Active sessions: {{active_sessions}}
- Running queries: {{running_queries}}
- Waiting queries: {{waiting_queries}}
- Long-running queries: {{long_running_qry}} (> {{threshold}}s)
Identify query causing CPU spike and suggest optimization.

GaussDB CPU intensive: complex joins, full table scans, statistics updates, vacuum
```

---

## 2. Root Cause Analysis Prompts

### 2.1 Transaction Lock Wait Analysis
```
Analyze GaussDB transaction lock wait on {{resource_id}}:
- Blocked transactions: {{blocked_tx_count}}
- Lock wait time histogram: {{lock_wait_histogram}}
- Table involved: {{table_name}}
- Lock mode: {{lock_mode}} (row/exclusive/shared)
- Waiting transaction: {{waiting_txid}}
- Blocking transaction: {{blocking_txid}}
Provide deadlock risk assessment and resolution.

GaussDB lock types: row lock, table lock, page lock, transaction ID lock
```

### 2.2 Replication Lag Investigation
```
Investigate GaussDB replication lag on {{resource_id}}:
- Replica lag: {{replica_lag}}MB (threshold: {{lag_threshold}}MB)
- WAL receive rate: {{wal_recv_rate}}MB/s
- WAL replay rate: {{wal_replay_rate}}MB/s
- Network latency: {{net_latency}}ms
- Disk I/O on standby: {{standby_disk_io}}ops/s
Identify bottleneck and recommend remediation.

GaussDB replication: synchronous/asynchronous, streaming replication, logical replication
```

### 2.3 Slow Query Root Cause
```
Root cause analysis for slow query on GaussDB {{resource_id}}:
- Query text: {{query_text}}
- Execution time: {{execution_time}}ms (normal: {{normal_time}}ms)
- Rows scanned: {{rows_scanned}}, Rows returned: {{rows_returned}}
- Execution plan: {{execution_plan}}
- Index usage: {{index_usage}}
- Join type: {{join_type}}
Provide specific optimization recommendations.

GaussDB slow query causes: missing index, statistics stale, large sort, sequential scan
```

### 2.4 Memory Overcommit Diagnosis
```
Diagnose GaussDB memory issue on {{resource_id}}:
- Memory usage: {{mem_usage}}% of {{total_memory}}GB
- Shared buffers: {{shared_buffers}}GB
- Work memory: {{work_mem}}GB
- Maintenance work memory: {{maintenance_work_mem}}GB
- OS memory available: {{os_mem_available}}GB
- OOM events: {{oom_count}} in past {{time_window}}
Identify memory overhead and recommend tuning.

GaussDB memory components: shared buffers, work memory, temp buffers, lock headers
```

### 2.5 Backup Failure Analysis
```
Analyze GaussDB backup failure on {{resource_id}}:
- Backup type: {{backup_type}} (full/incremental)
- Failure time: {{failure_time}}
- Error message: {{error_message}}
- Last successful backup: {{last_backup_time}}
- Backup size: {{backup_size}}GB
- Duration: {{duration}} seconds (expected: {{expected_duration}}s)
Diagnose failure cause and recovery steps.

GaussDB backup methods: CBR vault, pg_dump, logical replication
```

---

## 3. Capacity Prompts

### 3.1 Capacity Planning Review
```
Review GaussDB capacity for {{resource_id}}:
- Current utilization: CPU {{cpu_util}}%, Memory {{mem_util}}%, Disk {{disk_util}}%
- Connection utilization: {{conn_util}}% ({{current_conn}}/{{max_conn}})
- Storage growth trend: {{storage_growth}}GB/month
- Projected capacity exhaustion: {{exhaustion_date}}
- Max connections limit: {{max_connections}}
Provide scaling recommendations with timeline.

Capacity dimensions: compute unit quotas, storage limits, connection limits, replication bandwidth
```

### 3.2 Right-Sizing Recommendation
```
Recommend GaussDB instance right-sizing for {{resource_id}}:
- Current instance type: {{instance_type}} ({{vcpu}} vCPU, {{memory}}GB RAM)
- Average CPU utilization: {{avg_cpu}}% over 30 days
- Peak CPU utilization: {{peak_cpu}}%
- Average memory: {{avg_mem}}% (including buffer cache)
- Connection usage: {{conn_util}}%
Provide optimized instance type recommendation.

Right-sizing triggers: avg CPU < 30%, connections < 20%, memory < 50% sustained
```

### 3.3 Storage Capacity Forecast
```
Forecast GaussDB storage for {{resource_id}}:
- Current data size: {{data_size}}GB
- Transaction log size: {{tlog_size}}GB
- Free space: {{free_space}}GB
- Daily growth rate: {{daily_growth}}GB/day
- Projected full date at current rate: {{projected_full_date}}
- Compression ratio: {{compression_ratio}}:1
Recommend storage expansion or data cleanup.

GaussDB storage: data compression, table partitioning, archival policies
```

### 3.4 Connection Pool Sizing
```
Calculate optimal connection pool for GaussDB {{resource_id}}:
- Max connections (server): {{max_connections}}
- Average active connections: {{avg_active_conn}}
- Average waiting connections: {{avg_waiting_conn}}
- Query execution time: {{avg_query_time}}ms
- Application count: {{app_count}}
- vCPU count: {{vcpu}}
Recommend connection pool size and queue settings.

GaussDB connection formula: pool_size = ((core_count * 2) + effective_spindle_count)
```

---

## 4. Availability Prompts

### 4.1 HA Health Check
```
Perform HA health check for GaussDB {{resource_id}}:
- Primary status: {{primary_status}} (AZ: {{primary_az}})
- Standby status: {{standby_status}} (AZ: {{standby_az}})
- Replication mode: {{replication_mode}} (sync/async)
- Replication lag: {{replica_lag}}MB
- Failover readiness: {{failover_readiness}}
- Last failover: {{last_failover_time}}
Report HA health status and failover readiness.

GaussDB HA: automatic failover, synchronous replication, Multi-AZ deployment
```

### 4.2 Backup Completeness Verification
```
Verify backup completeness for GaussDB {{resource_id}}:
- Last full backup: {{last_full_backup}} (size: {{full_backup_size}}GB)
- Last incremental backup: {{last_incr_backup}}
- Backup retention: {{retention_days}} days
- Backup success rate: {{backup_success_rate}}%
- CBR vault: {{vault_id}}
- Point-in-time recovery window: {{pitr_window}} hours
Validate backup is current and recoverable.

GaussDB backup: automatic daily backup, manual backup, cross-region backup
```

### 4.3 Disaster Recovery Assessment
```
Assess DR readiness for GaussDB {{resource_id}}:
- Primary region: {{primary_region}}, AZ: {{primary_az}}
- DR region: {{dr_region}}, AZ: {{dr_az}}
- RPO achieved: {{rpo_achieved}} minutes (target: {{rpo_target}} minutes)
- RTO achieved: {{rto_achieved}} minutes (target: {{rto_target}} minutes)
- Replication status: {{replication_status}}
- Last DR drill: {{last_dr_drill}}
Evaluate DR capabilities and gaps.

GaussDB DR: cross-region replication, CBR cross-region backup, read replica promotion
```

### 4.4 Performance SLA Monitoring
```
Monitor GaussDB performance SLA for {{resource_id}}:
- SLO targets: QPS {{target_qps}}, Latency P99 {{target_latency}}ms
- Current QPS: {{current_qps}} (avg: {{avg_qps}})
- Latency P50: {{latency_p50}}ms, P99: {{latency_p99}}ms
- Slow queries: {{slow_query_count}}/hour (> {{slow_query_threshold}}s)
- Active sessions: {{active_sessions}}
Report SLA compliance and violations.

GaussDB SLA metrics: query throughput, transaction latency, connection availability
```

---

## 5. Inspection Prompts

### 5.1 Routine Health Inspection
```
Perform routine inspection on GaussDB instances:
- List all GaussDB instances in {{scope}}
- Check CPU > 80% OR Memory > 85% sustained for 30 minutes
- Identify instances with replication lag > 1GB
- Flag any instances with active CES alerts
- Check backup status and age
- Verify HA pair health and failover readiness
Report findings in structured format.

Scope options: region, AZ, project, tag-based resource group
```

### 5.2 Security Compliance Check
```
Audit GaussDB security compliance:
- Verify SSL/TLS encryption enabled: {{ssl_enabled}}
- Check password policy compliance: {{password_policy}}
- Validate SSL certificate expiration: {{cert_expiry}}
- Confirm audit logging (CTS) enabled: {{cts_enabled}}
- Check public IP exposure: {{public_ip_count}}
- Verify VPC private access only: {{vpc_only}}
Report compliance status and remediation priorities.

Severity: Critical = public IP exposed, High = SSL disabled, Medium = weak password policy
```

### 5.3 Cost Optimization Scan
```
Scan GaussDB for cost optimization:
- Identify idle instances (connections < 5, CPU < 5% for 14 days)
- Find oversized instances (avg CPU < 30%, memory < 50%)
- Check for unused read replicas
- Verify backup retention policy (30/7/1 policy)
- Check auto-scaling configuration
Provide prioritized action list with estimated monthly savings.

GaussDB cost: instance type, storage volume, backup storage, traffic
```

### 5.4 Performance Tuning Inspection
```
Inspect GaussDB performance tuning status:
- Shared buffer hit ratio: {{buffer_hit_ratio}}% (target: > 95%)
- Index hit ratio: {{index_hit_ratio}}%
- Vacuum progress: {{vacuum_progress}}%
- Statistics age: {{stats_age}} days
- Lock wait time: {{lock_wait_time}}ms
- Connection wait time: {{conn_wait_time}}ms
Recommend tuning actions.

GaussDB tuning: shared_buffers, work_mem, maintenance_work_mem, effective_cache_size
```

### 5.5 Configuration Drift Detection
```
Detect configuration drift for GaussDB {{resource_id}}:
- Expected parameters: {{expected_params}}
- Actual parameters: {{actual_params}}
- Expected security settings: {{expected_security}}
- Actual security settings: {{actual_security}}
- Expected backup schedule: {{expected_backup}}
- Actual backup schedule: {{actual_backup}}
Recommend reconciliation actions.

GaussDB drift: parameter group changes, security rule modifications, backup schedule changes
```

---

## Appendix: GaussDB-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{resource_id}}` | GaussDB instance ID | `05aa28c3-6d24-4c9d-9f2e-3a7d8e2f1c0b` |
| `{{az}}` | Availability zone | `cn-north-4a` |
| `{{instance_type}}` | GaussDB flavor | `gaussdb.mysql.xlarge.ha` |
| `{{max_connections}}` | Max connection limit | `2000` |
| `{{replication_mode}}` | Replication mode | `sync` |
| `{{vault_id}}` | CBR vault ID | `vault-12345` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance*
