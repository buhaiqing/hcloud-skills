# Prompts — Huawei Cloud RDS

> **Purpose:** Structured prompts for RDS AIOps operations. Derived from `prompt-handbook-template.md`.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 Instance Health Check
```
Analyze RDS instance {{resource_id}} health status:
- Current metric values: CPU {{cpu_usage}}%, Memory {{mem_used}}GB, Disk {{disk_usage}}%
- Active connections: {{active_connections}} / {{max_connections}}
- Replication status: {{replication_status}} (lag: {{replica_lag}}s)
- Recent alert history: {{alert_count}} alerts in past {{time_window}}
Determine if instance is healthy and recommend actions.

Applicable CES metrics: SYS.RDS.cpu_usage, SYS.RDS.mem_used, SYS.RDS.disk_usage, SYS.RDS.connection_usage
```

### 1.2 Root Cause Analysis
```
Given RDS instance {{resource_id}} shows:
- Symptom: {{symptom_description}}
- First observed: {{first_observed_time}}
- Metric anomaly: CPU {{cpu_anomaly}}%, Memory {{mem_anomaly}}%, Connections {{conn_anomaly}}%
- Correlated CTS events: {{cts_events}}
Perform root cause analysis and provide ranked hypothesis list with confidence scores.

Common RDS failure modes: connection pool exhaustion, replication lag, storage full, slow queries, HA failover
```

### 1.3 Performance Degradation Diagnosis
```
RDS instance {{resource_id}} performance degraded:
- Latency increased from {{baseline_latency}}ms to {{current_latency}}ms
- Error rate changed from {{baseline_error}}% to {{current_error}}%
- Throughput {{throughput_change_direction}} by {{throughput_change_percent}}%
- Resource utilization: CPU {{cpu_util}}%, Memory {{mem_util}}%, Disk IO {{io_util}}%
- Active sessions: {{active_sessions}}, Slow queries: {{slow_query_count}}/min
Diagnose root cause and suggest remediation steps.

Possible causes: slow queries, connection exhaustion, replication lag, lock contention, I/O bottleneck
```

### 1.4 HA Failover Detection
```
RDS instance {{resource_id}} HA failover detected:
- HA status changed from {{old_status}} to {{new_status}} at {{failover_time}}
- Connection disruption duration: {{disruption_duration}} seconds
- Replication lag at failover: {{lag_at_failover}}s
- Data sync status: {{sync_status}}
- Affected databases: {{affected_databases}}
Generate post-failover validation checklist and verify data integrity.

RDS HA patterns: planned switchover (minimal disruption), failover due to primary failure (30-120s disruption)
```

### 1.5 Connection Issue Diagnosis
```
RDS instance {{resource_id}} connection issue:
- Connection errors: {{connection_errors}}/min
- Active connections: {{active_connections}} (max: {{max_connections}})
- Connection wait time: {{connection_wait_time}}ms
- Failed connection attempts: {{failed_conn_attempts}}/min
- Application error messages: {{error_messages}}
Diagnose connection issue root cause and recommend actions.

Common causes: max_connections reached, network issues, connection pool leak, authentication failures
```

---

## 2. Root Cause Analysis Prompts

### 2.1 Slow Query Analysis
```
Analyze slow query issue on RDS instance {{resource_id}}:
- Slow query count: {{slow_query_count}}/min (threshold: {{slow_query_threshold}}/min)
- Average query duration: {{avg_query_duration}}ms (P99: {{p99_duration}}ms)
- Top slow queries:
  - Query 1: {{query_1_sql}} (executions: {{q1_exec}}/min, avg: {{q1_avg}}ms)
  - Query 2: {{query_2_sql}} (executions: {{q2_exec}}/min, avg: {{q2_avg}}ms)
- Table sizes: {{table_info}}
Identify root causes and provide optimization recommendations.

RDS slow query patterns: missing indexes, SELECT *, full table scans, improper joins, outdated statistics
```

### 2.2 Replication Lag Analysis
```
Analyze replication lag on RDS instance {{resource_id}}:
- Replica lag: {{replica_lag}}s (threshold: {{lag_threshold}}s)
- Binlog position: {{binlog_file}}:{{binlog_position}}
- Replica IO running: {{io_running}}, Replica SQL running: {{sql_running}}
- Active sessions on replica: {{replica_sessions}}
- Network throughput primary→replica: {{replication_throughput}}KB/s
Determine if lag is caused by network, heavy write load, or replica performance issues.

Replication lag causes: network latency, high write throughput, large transactions, replica resource constraints
```

### 2.3 Storage Issue Analysis
```
Analyze storage issue on RDS instance {{resource_id}}:
- Disk usage: {{disk_usage}}% (threshold: {{disk_threshold}}%)
- Data disk: {{data_disk_used}}GB / {{data_disk_total}}GB
- Log disk: {{log_disk_used}}GB / {{log_disk_total}}GB
- Binlog size: {{binlog_size}}GB
- Temp file usage: {{temp_file_usage}}GB
- Table fragmentation: {{fragmentation_rate}}%
Identify space consumption sources and recommend cleanup actions.

Storage consumption patterns: data growth, binlog accumulation, temporary tables, undo log buildup, index bloat
```

### 2.4 Lock Contention Analysis
```
Analyze lock contention on RDS instance {{resource_id}}:
- Lock wait count: {{lock_wait_count}}/min
- Average lock wait time: {{lock_wait_time}}ms
- Deadlock count: {{deadlock_count}} in past {{time_window}}
- Top waiting transactions:
  - Transaction {{tx_id_1}}: {{tx1_query}} (waiting for {{lock_type_1}}, held by {{holder_1}})
  - Transaction {{tx_id_2}}: {{tx2_query}} (waiting for {{lock_type_2}}, held by {{holder_2}})
Identify blocking transactions and recommend resolution steps.

Lock contention patterns: long-running transactions, uncommitted changes, incompatible lock modes, missing index scans
```

### 2.5 Memory Issue Analysis
```
Analyze memory issue on RDS instance {{resource_id}}:
- Memory usage: {{mem_used}}GB / {{mem_total}}GB ({{mem_percent}}%)
- Buffer pool hit ratio: {{buffer_hit_ratio}}%
- Buffer pool size: {{buffer_pool_size}}GB
- Query cache size: {{query_cache_size}}MB (if applicable)
- Temporary table creations: {{temp_table_count}}/min
- Memory sorts: {{memory_sorts}}/min vs disk sorts: {{disk_sorts}}/min
Identify if memory pressure is from buffer pool, query cache, or temporary objects.

Memory pressure indicators: low buffer hit ratio, high disk sorts, swap usage, OOM killer activity
```

---

## 3. Capacity Prompts

### 3.1 Capacity Planning Review
```
Review RDS capacity for {{scope}}:
- Current utilization: CPU {{cpu_util}}%, Memory {{mem_util}}%, Disk {{disk_util}}%
- Instance type: {{instance_type}} ({{vcpu}}vCPU, {{memory}}GB RAM)
- Connection utilization: {{conn_util}}% ({{active_connections}}/{{max_connections}})
- Storage utilization: {{storage_util}}% ({{used_storage}}GB/{{total_storage}}GB)
- Growth trend: {{growth_rate}}% weekly average
- Projected capacity exhaustion: {{exhaustion_date}}
Provide scaling recommendations with timeline.

Capacity dimensions: vCPU, memory, storage, max_connections, read replica capacity
```

### 3.2 Right-Sizing Recommendation
```
Right-size RDS instance {{resource_id}}:
- Current instance type: {{current_type}} ({{vcpu}}vCPU, {{memory}}GB)
- Current average utilization:
  - CPU: {{avg_cpu}}% (peak: {{peak_cpu}}%)
  - Memory: {{avg_mem}}% (peak: {{peak_mem}}%)
  - Connections: {{avg_conn}}% (peak: {{peak_conn}}%)
- Storage growth: {{storage_growth_per_month}}GB/month
- Cost sensitivity: {{cost_sensitivity}}
Recommend optimal instance type with cost-benefit analysis.

Right-sizing criteria: avg CPU < 40% → downsize; avg CPU > 80% → upsize; memory > 90% → upsize
```

### 3.3 Storage Capacity Forecast
```
Forecast storage capacity for RDS instance {{resource_id}}:
- Current disk usage: {{disk_used}}GB / {{disk_total}}GB ({{disk_percent}}%)
- Daily growth rate: {{daily_growth_gb}}GB/day
- Historical growth pattern: {{growth_pattern}} (linear/exponential/seasonal)
- Scheduled data cleanup: {{cleanup_schedule}}
- Auto-scaling status: {{auto_scaling_enabled}} ({{scaling_threshold}}% trigger)
Project exhaustion date and recommend capacity actions.

Storage forecast factors: data growth rate, retention policy, archiving strategy, binlog cleanup
```

### 3.4 Connection Pool Sizing
```
Evaluate connection pool sizing for RDS instance {{resource_id}}:
- Current max_connections: {{max_connections}}
- Average active connections: {{avg_active_connections}}
- Peak connections: {{peak_connections}} (at {{peak_time}})
- Connection wait time: {{conn_wait_time}}ms (threshold: {{wait_threshold}}ms)
- Application connection pool config: {{app_pool_config}}
- vCPU count: {{vcpu}} (recommended max_connections = vcpu × 100)
Recommend optimal max_connections and connection pool settings.

Connection pool formula: max_connections should balance application demand vs. instance capacity (vcpu × 100 for MySQL)
```

### 3.5 Read Replica Scaling
```
Evaluate read replica scaling for RDS instance {{resource_id}}:
- Primary instance: {{primary_type}} ({{primary_vcpu}}vCPU)
- Current replicas: {{replica_count}} ({{replica_types}})
- Replica lag: {{avg_replica_lag}}s (max: {{max_replica_lag}}s)
- Read traffic ratio: {{read_traffic_percent}}% of total traffic
- Write QPS: {{write_qps}}, Read QPS: {{read_qps}}
- Replication throughput: {{replication_throughput}}MB/s
Recommend number of replicas and instance types.

Replica scaling triggers: read traffic > 70%, replica lag < 30s, read latency > 100ms
```

---

## 4. Availability Prompts

### 4.1 SLA Analysis
```
Analyze SLA compliance for RDS instance {{resource_id}}:
- SLO target: {{slo_target}}% availability ({{downtime_minutes}}min/month allowed)
- Actual availability: {{actual_availability}}% in past 30 days
- Downtime incidents: {{incident_count}}
  - Incident 1: {{incident_1_type}} at {{incident_1_time}}, duration {{incident_1_duration}}min
  - Incident 2: {{incident_2_type}} at {{incident_2_time}}, duration {{incident_2_duration}}min
- Error budget remaining: {{error_budget_remaining}}min
Assess compliance status and recommend improvements.

SLO calculation: (total_minutes - downtime_minutes) / total_minutes × 100%
```

### 4.2 Backup Verification
```
Verify backup status for RDS instance {{resource_id}}:
- Automated backup: {{auto_backup_enabled}} (retention: {{backup_retention}} days)
- Latest backup: {{last_backup_time}} ({{last_backup_size}}GB)
- Backup success rate: {{backup_success_rate}}% (last 30 days)
- Manual backups: {{manual_backup_count}}
- Point-in-time recovery: {{pitr_enabled}} (earliest: {{pitr_start_time}})
- Cross-region backup: {{cross_region_backup_enabled}}
Validate backup completeness and recommend verification steps.

Backup validation: verify backup file integrity, test restoration on dev instance, confirm PITR capability
```

### 4.3 Disaster Recovery Assessment
```
Assess disaster recovery readiness for RDS instance {{resource_id}}:
- Primary region: {{primary_region}}, AZ: {{primary_az}}
- Backup region: {{backup_region}}, AZ: {{backup_az}}
- RTO target: {{rto_target}} minutes, RPO target: {{rpo_target}} minutes
- Current RTO estimate: {{current_rto}} minutes, Current RPO: {{current_rpo}} minutes
- HA configuration: {{ha_type}} (single/primary-standby/polynomial)
- Backup frequency: {{backup_frequency}}
- Cross-region replication: {{cross_region_replication_status}}
Evaluate DR gaps and provide recommendations.

DR mechanisms: automated backup to OBS, cross-region backup, read replica promotion, HA failover
```

### 4.4 HA Configuration Review
```
Review HA configuration for RDS instance {{resource_id}}:
- HA mode: {{ha_mode}} (single/primary-standby/ha-cluster)
- PrimaryAZ: {{primary_az}}, StandbyAZ: {{standby_az}}
- Replication mode: {{replication_mode}} (sync/async)
- Failover threshold: {{failover_threshold}}s
- Switchover status: {{switchover_count}} in past 30 days
- Last failover: {{last_failover_time}} (reason: {{failover_reason}})
Validate HA configuration meets availability requirements.

HA best practices: sync replication for zero RPO, multi-AZ deployment, regular failover testing
```

### 4.5 Incident Impact Assessment
```
Assess impact of RDS incident on {{resource_id}}:
- Incident type: {{incident_type}} (performance/degradation/outage)
- Start time: {{incident_start}}, Duration: {{incident_duration}}min
- Affected operations: {{affected_operations}} (reads/writes/all)
- Error rate during incident: {{error_rate}}%
- Transaction rollback rate: {{rollback_rate}}%
- Affected users/applications: {{affected_services}}
- Business impact: {{business_impact_description}}
Quantify incident severity and document lessons learned.

Incident severity: P1 (complete outage), P2 (degraded performance), P3 (minor impact), P4 (cosmetic)
```

---

## 5. Inspection Prompts

### 5.1 Routine Health Inspection
```
Perform routine inspection on RDS instances:
- List all RDS instances in {{scope}}
- Check for instances with CPU > 80% OR Memory > 85% sustained for 30 minutes
- Identify instances with replication lag > 30s
- Flag any instances with active CES alerts
- Check for instances in error state or backing up
- Verify HA status on all instances
Report findings in structured format.

Scope options: region, AZ, project, tag-based resource group
```

### 5.2 Security Compliance Check
```
Audit RDS security compliance:
- Verify security groups follow least-privilege (port 3306 from trusted sources only)
- Check for instances with public accessibility enabled
- Validate SSL connections are enforced: {{ssl_required}}
- Confirm audit logging is enabled: {{audit_log_status}}
- Check for weak passwords or default accounts
- Verify encryption at rest is enabled: {{encryption_status}}
- Confirm backup encryption: {{backup_encryption}}
Report compliance status and remediation priorities.

Security standards: non-zero CIDR blocks for MySQL port, SSL/TLS required, audit logging enabled, encryption at rest
```

### 5.3 Cost Optimization Scan
```
Scan RDS for cost optimization opportunities:
- Identify idle instances (CPU < 5%, connections < 10 for 14 days)
- Find oversized instances (avg CPU < 30%, avg memory < 40% over 30 days)
- Check for unused read replicas (replication lag > 1h consistently)
- Verify reserved capacity coverage vs on-demand usage
- Check for overdue storage (deleted instances still incurring backup costs)
- Evaluate instance type changes for better cost efficiency
Provide prioritized action list with estimated monthly savings.

Cost optimization triggers: idle > 14 days, utilization < 30% for 30 days, single-AZ with HA waste
```

### 5.4 Parameter Configuration Audit
```
Audit RDS parameter configuration for instance {{resource_id}}:
- Critical parameters:
  - max_connections: {{max_connections}} (recommended: vcpu × 100 = {{recommended_connections}})
  - innodb_buffer_pool_size: {{buffer_pool_size}}GB (recommended: {{recommended_buffer_size}}GB)
  - wait_timeout: {{wait_timeout}}s (recommended: 60-300s)
  - long_query_time: {{long_query_time}}s (recommended: 1-5s)
  - binlog_expire_logs_seconds: {{binlog_expire}} (recommended: {{recommended_binlog_expire}})
- Recent parameter changes: {{parameter_changes}}
- Non-default values: {{custom_parameters}}
Identify misconfigurations and recommend corrections.

Parameter tuning: buffer pool 70-80% of RAM, connection limit based on vCPU, binlog retention based on backup schedule
```

### 5.5 Monitoring Coverage Check
```
Verify monitoring coverage for RDS instance {{resource_id}}:
- CES metrics enabled: {{ces_enabled}} (namespace: SYS.RDS)
- Monitored metrics:
  - CPU: {{cpu_monitored}}%, collected every {{cpu_interval}}s
  - Memory: {{mem_monitored}}%, collected every {{mem_interval}}s
  - Disk: {{disk_monitored}}%, collected every {{disk_interval}}s
  - Connections: {{conn_monitored}}, collected every {{conn_interval}}s
  - Replication: {{replication_monitored}}, collected every {{replication_interval}}s
- Alarm rules configured: {{alarm_rules_count}}
- Alarm notification: {{notification_enabled}} ({{notification_method}})
Identify monitoring gaps and recommend additions.

Essential RDS metrics: cpu_usage, mem_used, disk_usage, connection_usage, rds_ha_lag, slow_query_count
```

---

## Appendix: RDS-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{resource_id}}` | RDS instance ID | `01b52a47-e23f-403a-9be8-4a5d2e1c3f67` |
| `{{az}}` | Availability zone | `cn-north-4a` |
| `{{instance_type}}` | RDS flavor | `rds.mysql.s3.large.2` |
| `{{vcpu}}` | vCPU count | `2` |
| `{{memory}}` | Memory in GB | `4` |
| `{{replica_lag}}` | Replication lag in seconds | `0.5` |
| `{{max_connections}}` | Maximum connections allowed | `4000` |
| `{{slow_query_count}}` | Slow query count per minute | `15` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance (Prompt Handbook P1-3)*
