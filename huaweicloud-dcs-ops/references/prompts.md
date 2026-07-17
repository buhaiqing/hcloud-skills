# Prompts — Huawei Cloud DCS

> **Purpose:** Structured prompts for DCS (Distributed Cache Service) AIOps operations.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 Instance Health Check
```
Analyze DCS instance {{resource_id}} health status:
- Current metric values: CPU {{cpu_usage}}%, Memory {{mem_usage}}%, Connections {{conn_count}}/{{max_connections}}
- Hit rate: {{hit_rate}}% (target: > {{target_hit_rate}}%)
- Memory fragmentation: {{mem_fragmentation}}%
- Eviction count: {{eviction_count}}/s
- Recent alert history: {{alert_count}} alerts
Determine if instance is healthy and recommend actions.

Applicable CES metrics: SYS.DCS.cpu_usage, SYS.DCS.memory_usage, SYS.DCS.connection_count, AGT.DCS.hit_rate
```

### 1.2 Memory Overload Diagnosis
```
DCS memory issue on {{resource_id}}:
- Memory used: {{mem_used}}MB / {{mem_total}}MB ({{mem_usage}}%)
- Memory fragmentation: {{fragmentation}}%
- Fragmentation rate: {{frag_rate}}%/hour
- Evictions: {{eviction_count}} keys/sec
- Expired keys: {{expired_keys}}/sec
- Max memory policy: {{maxmemory_policy}}
Assess risk and recommend memory optimization.

DCS memory: used_memory, maxmemory, eviction policies (volatile-lru, allkeys-lru)
```

### 1.3 Connection Exhaustion Analysis
```
DCS connection exhaustion on {{resource_id}}:
- Current connections: {{current_conn}}/{{max_conn}} ({{utilization}}%)
- Connection creation rate: {{conn_create_rate}}/s
- Command rate: {{cmd_rate}}/s
- Blocked clients: {{blocked_clients}}
- Client output buffer: {{output_buffer}}KB
Identify root cause and recommend connection tuning.

DCS connection limits: maxclients parameter, client query buffer limit
```

### 1.4 Hit Rate Degradation Analysis
```
DCS hit rate degradation on {{resource_id}}:
- Current hit rate: {{hit_rate}}% (baseline: {{baseline_hit_rate}}%)
- Miss rate: {{miss_rate}}/s
- Memory usage: {{mem_usage}}%
- Key count: {{key_count}} (avg: {{avg_key_size}} bytes)
- Expired keys: {{expired_keys}}/sec
- Evicted keys: {{evicted_keys}}/sec
Diagnose cause and recommend cache optimization.

DCS hit rate factors: memory size, key TTL distribution, eviction policy, access patterns
```

### 1.5 Persistence Health Check
```
Check DCS persistence health on {{resource_id}}:
- RDB status: {{rdb_status}} (last save: {{last_rdb_save}})
- AOF status: {{aof_status}} (fsync: {{aof_fsync_mode}})
- Persistence latency: {{persistence_latency}}ms
- Last persistence: {{last_persistence_time}}
- Child process status: {{child_status}}
Assess data durability risk.

DCS persistence: RDB snapshots, AOF log, mixed persistence
```

---

## 2. Root Cause Analysis Prompts

### 2.1 Slow Command Analysis
```
Analyze slow commands on DCS {{resource_id}}:
- Slow log entries: {{slow_log_count}} in past hour
- Slowest commands: {{slowest_commands}}
- Average latency: {{avg_latency}}ms
- P99 latency: {{p99_latency}}ms
- Memory operations: {{mem_ops}} (SMEMBERS, SUNION)
Identify problematic commands and access patterns.

DCS slow commands: KEYS, SMEMBERS, SUNION, large HGETALL, FLUSHDB
```

### 2.2 Cluster Node Failure Analysis
```
Analyze DCS cluster node failure on {{resource_id}}:
- Failed node: {{failed_node}} ({{failed_node_az}})
- Cluster status: {{cluster_status}}
- Slots affected: {{affected_slots}}
- Client redirects: {{client_redirects}}
- Replica sync status: {{replica_sync}}
Determine failure cause and recovery path.

DCS cluster: slot migration, node failover, replica promotion
```

### 2.3 Network Partition Diagnosis
```
Diagnose DCS network partition on {{resource_id}}:
- Cluster state: {{cluster_state}} (fail/ok/failover)
- Node reachability: {{reachable_nodes}}/{{total_nodes}}
- PONG responses: {{pong_responses}}/{{total_nodes}}
- Known nodes: {{known_nodes}}
- Last gossip: {{last_gossip_time}}
Assess partition severity and recommend actions.

DCS network: gossip protocol, ping intervals, node timeout
```

### 2.4 Eviction Surge Root Cause
```
Investigate DCS eviction surge on {{resource_id}}:
- Eviction rate: {{eviction_rate}} keys/sec (baseline: {{baseline_eviction}})
- Memory usage trend: {{mem_trend}}% over {{time_window}}
- Key TTL distribution: {{ttl_distribution}}
- Big key count: {{big_key_count}}
- Command analysis: {{cmd_patterns}}
Identify eviction trigger and recommend solutions.

DCS eviction triggers: memory pressure, TTL expiry, LRU/LFU policy
```

### 2.5 Replication Lag Analysis
```
Analyze DCS replication lag on {{resource_id}}:
- Master: {{master_id}}, Replica: {{replica_id}}
- Replica lag: {{replica_lag}} bytes (threshold: {{lag_threshold}}MB)
- Master sync rate: {{sync_rate}}KB/s
- Replica apply rate: {{apply_rate}}KB/s
- Command backlog: {{backlog_size}}MB
Identify bottleneck and recommend remediation.

DCS replication: async replication, psync, partial sync, full sync triggers
```

---

## 3. Capacity Prompts

### 3.1 Capacity Planning Review
```
Review DCS capacity for {{resource_id}}:
- Memory utilization: {{mem_util}}% ({{mem_used}}MB / {{mem_total}}MB)
- Connection utilization: {{conn_util}}% ({{conn_used}}/{{conn_max}})
- Key count: {{key_count}} (max: {{max_keys}})
- QPS capacity headroom: {{qps_headroom}}%
- Growth trend: {{growth_rate}}% weekly
- Projected exhaustion: {{exhaustion_date}}
Provide scaling recommendations.

Capacity dimensions: memory limits, connection limits, cluster slot capacity
```

### 3.2 Memory Right-Sizing
```
Right-size DCS memory for {{resource_id}}:
- Current memory: {{mem_total}}MB, Used: {{mem_used}}MB ({{mem_usage}}%)
- Eviction rate: {{eviction_rate}}/s
- Fragmentation: {{fragmentation}}%
- Avg key size: {{avg_key_size}} bytes
- Peak memory: {{peak_mem}}MB
Recommend optimal memory configuration.

DCS memory: maxmemory, memory overhead, fragmentation threshold
```

### 3.3 QPS Capacity Analysis
```
Analyze DCS QPS capacity for {{resource_id}}:
- Current QPS: {{current_qps}} (peak: {{peak_qps}})
- Target QPS: {{target_qps}}
- CPU utilization: {{cpu_util}}%
- Network bandwidth: {{net_bandwidth}}Mbps (max: {{max_bandwidth}}Mbps)
- Latency P99: {{latency_p99}}ms
Provide capacity headroom analysis.

DCS QPS limits: instance type, network bandwidth, command complexity
```

### 3.4 Cluster Scaling Assessment
```
Assess DCS cluster scaling needs for {{resource_id}}:
- Current nodes: {{node_count}} ({{node_type}})
- Slots per node: {{slots_per_node}}
- Memory per node: {{mem_per_node}}MB
- Keys per node: {{keys_per_node}}
- Rebalance cost: {{rebalance_time}} minutes
Recommend cluster expansion strategy.

DCS scaling: horizontal (add nodes), vertical (larger instance), slot migration
```

---

## 4. Availability Prompts

### 4.1 HA Health Check
```
Perform HA health check for DCS {{resource_id}}:
- Instance type: {{instance_type}} (single/proxy/cluster)
- Proxy nodes: {{proxy_count}} (healthy: {{proxy_healthy}})
- Data nodes: {{data_node_count}} (healthy: {{data_node_healthy}})
- Replication status: {{replication_status}}
- Failover readiness: {{failover_readiness}}
- Last failover: {{last_failover}}
Report HA health status.

DCS HA: proxy HA, data node HA, automatic failover
```

### 4.2 Backup Status Verification
```
Verify DCS backup status for {{resource_id}}:
- Last backup: {{last_backup_time}} (type: {{backup_type}})
- Backup success: {{backup_success}} (failures: {{backup_failures}})
- Backup size: {{backup_size}}MB
- Retention: {{retention_days}} days
- Next scheduled: {{next_backup}}
Validate backup completeness.

DCS backup: manual backup, scheduled backup, cross-region backup
```

### 4.3 SLA Compliance Report
```
Report DCS SLA compliance for {{resource_id}}:
- Uptime: {{uptime}}% (target: {{target_uptime}}%)
- Availability incidents: {{incident_count}} (total downtime: {{downtime}}min)
- Hit rate: {{hit_rate}}% (target: > {{target_hit_rate}}%)
- Latency P99: {{latency_p99}}ms (target: < {{target_latency}}ms)
- Error rate: {{error_rate}}% (target: < {{target_error_rate}}%)
Report SLA violations and causes.

DCS SLA: instance availability, hit rate, latency
```

### 4.4 Disaster Recovery Readiness
```
Assess DCS DR readiness for {{resource_id}}:
- Primary: {{primary_region}}/{{primary_az}}
- DR: {{dr_region}}/{{dr_az}}
- Replication: {{replication_type}} (sync/async)
- RPO: {{rpo_achieved}}min (target: {{rpo_target}}min)
- RTO: {{rto_achieived}}min (target: {{rto_target}}min)
- Last DR test: {{last_dr_test}}
Evaluate DR capabilities.

DCS DR: cross-region replication, backup/restore, instance migration
```

---

## 5. Inspection Prompts

### 5.1 Routine Health Inspection
```
Perform routine inspection on DCS instances:
- List all DCS instances in {{scope}}
- Check memory > 85% OR CPU > 80% sustained for 30 min
- Identify hit rate < {{threshold_hit_rate}}%
- Flag instances with eviction > {{threshold_eviction}}/s
- Check backup age > 7 days
- Verify HA proxy health
Report findings.

Scope: region, AZ, project, tag-based resource group
```

### 5.2 Security Compliance Check
```
Audit DCS security compliance:
- Verify VPC private access: {{vpc_private}}
- Check SSL/TLS enabled: {{ssl_enabled}}
- Validate password authentication: {{auth_enabled}}
- Confirm audit logging: {{audit_enabled}}
- Check public IP exposure: {{public_ip_count}}
- Verify security group rules: {{sg_rules}}
Report compliance status.

Severity: Critical = public IP, High = no SSL, Medium = weak auth
```

### 5.3 Cost Optimization Scan
```
Scan DCS for cost optimization:
- Identify idle instances (QPS < {{qps_threshold}}, memory < 30% for 14 days)
- Find oversized instances (avg CPU < 20%, hit rate < 80%)
- Check unused backup storage
- Verify reserved capacity vs on-demand
- Check auto-scaling policy
Provide action list with estimated savings.

DCS cost: instance type, memory size, backup storage, traffic
```

### 5.4 Performance Baseline Inspection
```
Inspect DCS performance baseline:
- Average QPS: {{avg_qps}} (peak: {{peak_qps}})
- Hit rate: {{hit_rate}}%
- Latency avg: {{latency_avg}}ms, P99: {{latency_p99}}ms
- Memory fragmentation: {{fragmentation}}%
- Connection utilization: {{conn_util}}%
- Eviction rate: {{eviction_rate}}/s
Compare against baseline and flag anomalies.

DCS baseline: rolling 30-day average, peak hour comparison
```

### 5.5 Configuration Drift Detection
```
Detect configuration drift for DCS {{resource_id}}:
- Expected maxmemory: {{expected_maxmemory}}MB
- Actual maxmemory: {{actual_maxmemory}}MB
- Expected maxclients: {{expected_maxclients}}
- Actual maxclients: {{actual_maxclients}}
- Expected eviction policy: {{expected_policy}}
- Actual eviction policy: {{actual_policy}}
Recommend reconciliation.

DCS drift: maxmemory changes, policy changes, timeout settings
```

---

## Appendix: DCS-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{resource_id}}` | DCS instance ID | `dcsabcd1234` |
| `{{instance_type}}` | DCS flavor | `dcs.redis.memcached.ha` |
| `{{hit_rate}}` | Cache hit rate | `95.5` |
| `{{maxmemory_policy}}` | Eviction policy | `volatile-lru` |
| `{{cluster_status}}` | Cluster state | `ok` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance*
