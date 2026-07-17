# Prompts — Huawei Cloud DMS

> **Purpose:** Structured prompts for DMS (Distributed Message Service) AIOps operations.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 Instance Health Check
```
Analyze DMS instance {{resource_id}} health status:
- Current metrics: Messages per second {{msg_rate}}, Consumer count {{consumer_count}}
- Queue depth: {{queue_depth}} messages (max: {{queue_max_depth}})
- Message堆积: {{msg_backlog}} messages
- Consumer lag: {{consumer_lag}} messages
- Dead letter queue: {{dlq_count}} messages
Determine instance health and recommend actions.

Applicable CES metrics: SYS.DMS.queue_depth, SYS.DMS.consumer_count, SYS.DMS.message_backlog
```

### 1.2 Message Backlog Analysis
```
Analyze DMS message backlog on {{resource_id}}:
- Current backlog: {{backlog_count}} messages
- Backlog growth rate: {{backlog_growth}}/min
- Producer rate: {{producer_rate}}/s
- Consumer rate: {{consumer_rate}}/s
- Oldest message age: {{oldest_msg_age}} minutes
- Consumer group: {{consumer_group}}
Assess backlog severity and recommend actions.

DMS backlog: disk storage, retention policy, consumer capacity
```

### 1.3 Consumer Group Health Check
```
Check DMS consumer group health on {{resource_id}}:
- Consumer group: {{consumer_group_id}}
- Active consumers: {{active_consumers}}/{{registered_consumers}}
- Consumer lag: {{consumer_lag}} messages per partition
- Last commit offset: {{last_commit_time}}
- Rebalance status: {{rebalance_status}}
- Dead consumers: {{dead_consumers}}
Identify consumer issues and recommend remediation.

DMS consumer issues: stuck consumers, rebalance storms, offset drift
```

### 1.4 Partition Rebalance Analysis
```
Analyze DMS partition rebalance on {{resource_id}}:
- Topic: {{topic_name}}
- Partitions: {{partition_count}}
- Replicas: {{replica_count}}
- ISR (in-sync): {{isr_count}}/{{partition_count}}
- Leader election count: {{leader_election_count}}/hour
- Rebalance frequency: {{rebalance_freq}}/hour
Diagnose rebalance issues.

DMS rebalance triggers: consumer join/leave, broker failure, partition reassignment
```

### 1.5 Disk Space Alert
```
Analyze DMS disk space on {{resource_id}}:
- Disk usage: {{disk_usage}}% of {{total_disk}}GB
- Message storage: {{msg_storage}}GB
- Index storage: {{index_storage}}GB
- Log retention: {{retention_hours}} hours
- Cleanup rate: {{cleanup_rate}}MB/s
Assess storage risk and recommend cleanup.

DMS storage: message data, index files, transaction logs, retention based cleanup
```

---

## 2. Root Cause Analysis Prompts

### 2.1 Message Consumption Lag Analysis
```
Analyze DMS consumption lag on {{resource_id}}:
- Consumer group: {{consumer_group}}
- Total lag: {{total_lag}} messages
- Partition lag distribution: {{partition_lag}}
- Consumer instances: {{consumer_instances}}
- Fetch rate: {{fetch_rate}}msg/s per consumer
- Processing time: {{proc_time}}ms per message
Identify bottleneck and recommend scaling.

DMS lag causes: slow consumers, network latency, consumer group misconfiguration
```

### 2.2 Producer Failure Analysis
```
Analyze DMS producer failure on {{resource_id}}:
- Failed produce requests: {{failed_produce}}/{{total_produce}}
- Error codes: {{error_codes}}
- Retry rate: {{retry_rate}}%
- Latency P99: {{latency_p99}}ms
- Queue full events: {{queue_full_count}}
- Connection errors: {{conn_errors}}
Diagnose producer issues.

DMS producer errors: broker unavailable, message too large, rate limiting
```

### 2.3 Consumer Rebalance Storm
```
Diagnose DMS rebalance storm on {{resource_id}}:
- Rebalance count: {{rebalance_count}} in past hour
- Join group requests: {{join_requests}}/hour
- Sync group requests: {{sync_requests}}/hour
- Average rebalance time: {{rebalance_time}}ms
- Session timeout: {{session_timeout}}ms
- Heartbeat interval: {{heartbeat_interval}}ms
Identify rebalance trigger and recommend fixes.

DMS rebalance storm: network glitches, GC pauses, session timeout misconfiguration
```

### 2.4 API Rate Limiting Analysis
```
Analyze DMS API rate limiting on {{resource_id}}:
- API calls: {{api_calls}}/second (quota: {{api_quota}}/second)
- Throttled requests: {{throttled_requests}}
- Quota type: {{quota_type}} (produce/consume/admin)
- Retry after: {{retry_after}}ms
- Rate limit tier: {{rate_tier}}
Recommend API usage optimization.

DMS rate limiting: per-topic quotas, per-user quotas, partition-level limiting
```

### 2.5 Topic Creation Failure Analysis
```
Analyze DMS topic creation failure on {{resource_id}}:
- Topic name: {{topic_name}}
- Error: {{error_message}}
- Partition count requested: {{partitions_requested}}
- Replication factor requested: {{replicas_requested}}
- Available broker capacity: {{broker_capacity}}
- Topic limit: {{topic_limit}} (current: {{current_topics}})
Diagnose failure cause.

DMS topic limits: partition per broker, replication factor, storage quota
```

---

## 3. Capacity Prompts

### 3.1 Capacity Planning Review
```
Review DMS capacity for {{resource_id}}:
- Current QPS: Produce {{produce_qps}}, Consume {{consume_qps}}
- Storage utilization: {{storage_util}}% ({{storage_used}}GB / {{storage_total}}GB)
- Partition count: {{partition_count}} / {{partition_limit}}
- Consumer lag: {{consumer_lag}} messages
- Message retention: {{retention_hours}} hours
- Growth trend: {{growth_rate}}% weekly
Provide scaling recommendations.

Capacity dimensions: QPS limits, storage capacity, partition limits, consumer capacity
```

### 3.2 Partition Right-Sizing
```
Recommend partition right-sizing for {{resource_id}} topic {{topic_name}}:
- Current partitions: {{partition_count}}
- Current consumers: {{consumer_count}}
- Produce rate: {{produce_rate}}/s
- Consume rate: {{consume_rate}}/s
- Target lag: {{target_lag}} messages
- Avg message size: {{avg_msg_size}} bytes
Recommend optimal partition count.

DMS partition formula: partitions = max(produce_rate/throughput, consume_rate/consumer_throughput)
```

### 3.3 Storage Capacity Forecast
```
Forecast DMS storage for {{resource_id}}:
- Current storage: {{storage_used}}GB
- Daily growth: {{daily_growth}}GB
- Retention period: {{retention_hours}} hours
- Message size: {{avg_msg_size}} bytes
- Projected exhaustion: {{exhaustion_date}} at current growth
Recommend storage expansion or retention tuning.

DMS storage: message payload, index, logs, compaction
```

### 3.4 Consumer Scaling Assessment
```
Assess DMS consumer scaling for {{resource_id}} group {{consumer_group}}:
- Current consumers: {{consumer_count}}
- Target lag: {{target_lag}} messages
- Current lag: {{current_lag}} messages
- Process capacity: {{proc_capacity}}msg/s per consumer
- Network bandwidth: {{net_bandwidth}}Mbps
Recommend consumer scaling.

DMS scaling: partition count limits consumer parallelism
```

---

## 4. Availability Prompts

### 4.1 Broker Health Check
```
Perform DMS broker health check on {{resource_id}}:
- Broker ID: {{broker_id}}
- Status: {{broker_status}} (online/offline)
- Leader partitions: {{leader_partitions}}
- ISR count: {{isr_count}}/{{replicas}}
- Disk usage: {{disk_usage}}%
- CPU: {{cpu_usage}}%, Memory: {{mem_usage}}%
Report broker health.

DMS broker: leader election, ISR replication, disk health
```

### 4.2 Replication Health Analysis
```
Analyze DMS replication health on {{resource_id}}:
- Topic: {{topic_name}}
- Partition: {{partition_id}}
- Leader: {{leader_broker}}
- ISR brokers: {{isr_brokers}}
- End offset: {{end_offset}}
- LEO (log end offset): {{leo}}
- HW (high watermark): {{hw}}
Assess replication consistency.

DMS replication: synchronous (min.insync.replicas), asynchronous, leader election
```

### 4.3 Failover Readiness
```
Assess DMS failover readiness for {{resource_id}}:
- Broker count: {{broker_count}}
- Partition count: {{partition_count}}
- Replication factor: {{replication_factor}}
- Under-replicated partitions: {{urp_count}}
- Offline partitions: {{offline_partitions}}
- Preferred leader election: {{preferred_leader_enabled}}
Evaluate DR capabilities.

DMS failover: broker failure, AZ outage, controller switch
```

### 4.4 SLA Monitoring
```
Monitor DMS SLA for {{resource_id}}:
- Message delivery: {{delivery_rate}}% (target: {{target_delivery}}%)
- Latency P99: {{latency_p99}}ms (target: < {{target_latency}}ms)
- Produce success: {{produce_success_rate}}%
- Consume success: {{consume_success_rate}}%
- Queue depth alarm: {{queue_alarm_count}}
Report SLA compliance.

DMS SLA: message delivery rate, latency, availability
```

---

## 5. Inspection Prompts

### 5.1 Routine Health Inspection
```
Perform routine inspection on DMS instances:
- List all DMS instances in {{scope}}
- Check queue depth > {{threshold_depth}} messages
- Identify consumer lag > {{threshold_lag}} messages
- Flag partitions with ISR < {{replication_factor}}
- Check disk usage > {{threshold_disk}}%
- Verify no offline partitions
Report findings.

Scope: region, project, tag-based resource group
```

### 5.2 Security Compliance Check
```
Audit DMS security compliance:
- Verify VPC private access: {{vpc_private}}
- Check SASL/SSL enabled: {{sasl_ssl_enabled}}
- Validate access policy: {{access_policy}}
- Confirm topic permissions: {{topic_permissions}}
- Check public endpoint: {{public_endpoint}}
- Verify security groups: {{security_groups}}
Report compliance status.

Severity: Critical = public access, High = no SASL, Medium = open permissions
```

### 5.3 Cost Optimization Scan
```
Scan DMS for cost optimization:
- Identify low-usage topics (messages < {{threshold_msgs}}/day)
- Find oversized retention periods
- Check for duplicate consumers
- Verify partition count vs utilization
- Check reserved vs on-demand pricing
Provide action list with estimated savings.

DMS cost: partition count, storage, retention, traffic
```

### 5.4 Consumer Group Audit
```
Audit DMS consumer groups on {{resource_id}}:
- Total groups: {{group_count}}
- Active groups: {{active_groups}}
- Stale groups (no lag update > 24h): {{stale_groups}}
- Max lag groups: {{max_lag_groups}}
- Orphaned offsets: {{orphaned_offsets}}
Recommend cleanup actions.

DMS consumer cleanup: inactive groups, stale offsets, lag clearance
```

### 5.5 Configuration Drift Detection
```
Detect configuration drift for DMS {{resource_id}}:
- Expected retention: {{expected_retention}} hours
- Actual retention: {{actual_retention}} hours
- Expected partition count: {{expected_partitions}}
- Actual partition count: {{actual_partitions}}
- Expected replication factor: {{expected_replication}}
- Actual replication factor: {{actual_replication}}
Recommend reconciliation.

DMS drift: retention changes, partition changes, config updates
```

---

## Appendix: DMS-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{resource_id}}` | DMS instance ID | `dms-12345678` |
| `{{topic_name}}` | Topic name | `order-events` |
| `{{consumer_group}}` | Consumer group ID | `payment-processor` |
| `{{partition_id}}` | Partition ID | `2` |
| `{{queue_depth}}` | Queue message count | `15420` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance*
