# Prompts — Huawei Cloud CSS

> **Purpose:** Structured prompts for CSS (Cloud Search Service) AIOps operations.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 Cluster Health Check
```
Analyze CSS cluster {{resource_id}} health status:
- Cluster status: {{cluster_status}} (yellow/green/red)
- Active shards: {{active_shards}}/{{total_shards}}
- Initializing shards: {{initializing_shards}}
- Unassigned shards: {{unassigned_shards}}
- JVM heap usage: {{jvm_heap_usage}}% ({{jvm_heap_used}}GB / {{jvm_heap_total}}GB)
- CPU usage: {{cpu_usage}}%, Memory: {{mem_usage}}%
Determine cluster health and recommend actions.

Applicable CES metrics: SYS.CSS.jvm_heap_used, SYS.CSS.search_latency, AGT.CSS.jvm_heap
```

### 1.2 JVM Heap Analysis
```
Analyze CSS JVM heap on {{resource_id}}:
- Heap used: {{jvm_heap_used}}GB / {{jvm_heap_total}}GB ({{jvm_heap_usage}}%)
- Old gen usage: {{old_gen_usage}}GB ({{old_gen_percent}}%)
- Young gen usage: {{young_gen_usage}}GB
- GC count (young): {{ygc_count}}/hour
- GC count (old): {{ogc_count}}/hour
- GC time (young): {{ygc_time}}ms, (old): {{ogc_time}}ms
Diagnose memory pressure and recommend GC tuning.

CSS JVM: heap size, GC algorithms (G1GC, CMS), memory pool sizing
```

### 1.3 Search Latency Diagnosis
```
Diagnose CSS search latency on {{resource_id}}:
- Current P50: {{latency_p50}}ms, P99: {{latency_p99}}ms (baseline: {{baseline_p99}}ms)
- Query rate: {{query_rate}}/s
- Slow queries: {{slow_query_count}}/hour (> {{slow_threshold}}ms)
- Index size: {{index_size}}GB
- Shard count: {{shard_count}}
- Search thread pool: {{search_pool}}/{{search_pool_max}}
Identify bottleneck and recommend optimization.

CSS latency factors: query complexity, shard size, JVM GC, thread pool saturation
```

### 1.4 Shard Allocation Analysis
```
Analyze CSS shard allocation on {{resource_id}}:
- Total shards: {{total_shards}}
- Active primary: {{active_primary}}
- Active replica: {{active_replica}}
- Unassigned shards: {{unassigned_shards}}
- Delayed shards: {{delayed_shards}}
- Shard relocation: {{relocating_shards}}
Diagnose allocation issues.

CSS shard issues: disk watermark, node failure, allocation filtering, forced reroute
```

### 1.5 Index Health Check
```
Check CSS index health on {{resource_id}}:
- Index name: {{index_name}}
- Index size: {{index_size}}GB
- Document count: {{doc_count}}
- Shards: {{shard_count}} ({{primary_shards}}p + {{replica_shards}}r)
- Health: {{index_health}} (green/yellow/red)
- Refresh interval: {{refresh_interval}}s
Assess index status.

CSS index: primary/replica shards, mapping, aliases, rollover policy
```

---

## 2. Root Cause Analysis Prompts

### 2.1 Cluster Red Status Analysis
```
Analyze CSS cluster red status on {{resource_id}}:
- Red indices: {{red_indices_count}}
- Affected shards: {{red_shard_count}}
- Unassigned shards: {{unassigned_shards}}
- Cause: {{red_cause}}
- Disk watermark: {{disk_watermark}} (low/high/flood_stage)
Identify primary cause and recommend remediation.

CSS red causes: disk full, shard corruption, mapping conflicts, blocked writes
```

### 2.2 JVM OOM Analysis
```
Analyze CSS JVM OOM risk on {{resource_id}}:
- Heap usage: {{heap_usage}}% ({{heap_used}}GB / {{heap_total}}GB)
- Old gen: {{old_gen_usage}}% ({{old_gen_used}}GB)
- Allocation rate: {{alloc_rate}}MB/s
- GC pressure: {{gc_pressure}}%
- Field data cache: {{field_data_size}}GB
- Request cache: {{request_cache_size}}GB
Assess OOM risk and recommend heap tuning.

CSS OOM triggers: large facet queries, high indexing rate, aggregations on high-cardinality fields
```

### 2.3 Slow Query Root Cause
```
Root cause analysis for CSS slow query on {{resource_id}}:
- Query: {{query_type}} (search/aggregations/suggest)
- Execution time: {{exec_time}}ms (baseline: {{baseline_time}}ms)
- Indices queried: {{indices_count}}
- Shards queried: {{shards_queried}}
- Documents matched: {{docs_matched}}
- Script execution: {{script_time}}ms
Identify query bottleneck.

CSS slow query causes: wildcard expansion, sorting, script use, high-cardinality aggregations
```

### 2.4 Disk Watermark Analysis
```
Analyze CSS disk watermark on {{resource_id}}:
- Disk usage: {{disk_usage}}% of {{total_disk}}TB
- Low watermark: {{low_watermark}}% ({{low_watermark_absolute}}GB free)
- High watermark: {{high_watermark}}% ({{high_watermark_absolute}}GB free)
- Flood stage: {{flood_stage}}% ({{flood_stage_absolute}}GB free)
- Indices at watermark: {{indices_at_watermark}}
- Blocked writes: {{blocked_write_count}}
Recommend disk cleanup or cluster expansion.

CSS watermark: allocation filtering, forced reroute, index readonly
```

### 2.5 Yellow Cluster Analysis
```
Analyze CSS yellow cluster on {{resource_id}}:
- Yellow indices: {{yellow_indices_count}}
- Missing replicas: {{missing_replica_count}}
- Delayed shards: {{delayed_shards}}
- Allocation explain: {{allocation_explain}}
- Node capacity: {{node_capacity}}
Determine if auto-heal or manual intervention needed.

CSS yellow: replica creation pending, disk watermark, node leaving
```

---

## 3. Capacity Prompts

### 3.1 Cluster Scaling Assessment
```
Assess CSS cluster scaling for {{resource_id}}:
- Current nodes: {{node_count}} ({{node_type}})
- Shards per node: {{shards_per_node}} (target: < {{target_shards_per_node}})
- Disk usage: {{disk_usage}}% (free: {{free_disk}}GB)
- JVM heap: {{jvm_heap_usage}}%
- Query QPS: {{query_qps}} / {{max_qps}}
- Indexing rate: {{indexing_rate}}/s
Recommend scaling strategy.

CSS scaling: add nodes, scale instance type, index lifecycle management
```

### 3.2 Shard Count Optimization
```
Optimize CSS shard count for {{resource_id}} index {{index_name}}:
- Current shards: {{current_shards}}
- Document count: {{doc_count}}
- Avg shard size: {{avg_shard_size}}GB
- Target shard size: {{target_shard_size}}GB (50GB recommended)
- Growth rate: {{growth_rate}}% monthly
Recommend shard count adjustment.

CSS shard sizing: 30-50GB per shard, avoid too many small shards or too few large shards
```

### 3.3 Memory Right-Sizing
```
Right-size CSS JVM heap for {{resource_id}}:
- Current JVM heap: {{jvm_heap_total}}GB
- Heap usage: {{jvm_heap_usage}}%
- Field data: {{field_data_usage}}GB
- Request cache: {{request_cache_usage}}GB
- Segment count: {{segment_count}}
- Index size: {{total_index_size}}GB
Recommend heap configuration.

CSS heap: 50% of RAM for heap, field data circuit breaker, request cache
```

### 3.4 Storage Capacity Forecast
```
Forecast CSS storage for {{resource_id}}:
- Current storage: {{storage_used}}GB / {{storage_total}}GB
- Index size: {{index_size}}GB
- Translog: {{translog_size}}GB
- Disk growth rate: {{growth_rate}}GB/month
- Projected exhaustion: {{exhaustion_date}} at current rate
Recommend storage expansion or ILM policy.

CSS ILM: hot→warm→cold→delete, shard shrinking, force merge
```

---

## 4. Availability Prompts

### 4.1 Cluster Availability Check
```
Check CSS cluster availability on {{resource_id}}:
- Cluster status: {{cluster_status}} (green/yellow/red)
- Node count: {{node_count}} (data: {{data_nodes}}, master: {{master_nodes}})
- Shard allocation: {{allocation_rate}}% complete
- Index readonly: {{readonly_indices_count}}
- Recent restarts: {{restart_count}} in {{time_window}}
Report availability status.

CSS availability: multi-AZ deployment, dedicated master nodes, snapshot backup
```

### 4.2 Backup Status Verification
```
Verify CSS backup status on {{resource_id}}:
- Last snapshot: {{last_snapshot}} (size: {{snapshot_size}}GB)
- Snapshot status: {{snapshot_status}}
- Retention: {{retention_days}} days
- Auto snapshot: {{auto_snapshot_enabled}}
- Repository: {{repository_type}}
- Restore test: {{restore_test_status}}
Validate backup completeness.

CSS backup: snapshot API, cross-region replication, CBR integration
```

### 4.3 Node Failure Impact
```
Assess CSS node failure impact on {{resource_id}}:
- Failed node: {{failed_node}} ({{failed_node_az}})
- Shards on node: {{shards_on_node}} ({{primary}}/{{replica}})
- Shard relocation: {{relocation_time}} estimated
- Cluster status transition: {{status_transition}}
- Query impact: {{query_impact}}%
Evaluate recovery path.

CSS node failure: automatic replica promotion, shard reallocation, master election
```

### 4.4 SLA Compliance Report
```
Report CSS SLA compliance for {{resource_id}}:
- Availability: {{availability}}% (target: {{target_availability}}%)
- Search latency P99: {{latency_p99}}ms (target: < {{target_latency}}ms)
- Indexing latency P99: {{index_latency_p99}}ms
- Query success: {{query_success_rate}}%
- Cluster health: {{cluster_health}}% green time
Report SLA violations.

CSS SLA: cluster health, search latency, indexing throughput
```

---

## 5. Inspection Prompts

### 5.1 Routine Health Inspection
```
Perform routine CSS cluster inspection:
- List all CSS clusters in {{scope}}
- Check cluster status = green (not yellow/red)
- Identify JVM heap > {{jvm_threshold}}%
- Flag disk usage > {{disk_threshold}}%
- Check unassigned shards > 0
- Verify replicas = expected count
- Check slow query log for > {{slow_threshold}}ms
Report findings.

Scope: region, project, tag-based resource group
```

### 5.2 Security Compliance Check
```
Audit CSS security compliance:
- Verify VPC private access: {{vpc_private}}
- Check SSL/TLS enabled: {{ssl_enabled}}
- Validate authentication: {{auth_type}} (native/ldap)
- Confirm authorization: {{authz_enabled}}
- Check audit logging: {{audit_enabled}}
- Verify security groups: {{sg_rules}}
Report compliance status.

Severity: Critical = public access, High = no SSL, Medium = weak auth
```

### 5.3 Cost Optimization Scan
```
Scan CSS for cost optimization:
- Identify small shards (< {{small_shard_threshold}}GB)
- Find unused indices (no writes > 30 days)
- Check disk usage vs index size ratio
- Verify ILM policies applied
- Check node instance type vs utilization
Provide action list with estimated savings.

CSS cost: node instance type, storage, snapshot storage, multi-AZ
```

### 5.4 Index Mapping Inspection
```
Inspect CSS index mappings on {{resource_id}}:
- Index: {{index_name}}
- Field count: {{field_count}}
- Nested fields: {{nested_count}}
- Dynamic mapping: {{dynamic_mapping}}
- Keyword fields: {{keyword_count}}
- Text fields: {{text_count}} (with analyzers)
Flag mapping bloat or anti-patterns.

CSS mapping: avoid nested too deep, disable dynamic, optimize keyword vs text
```

### 5.5 Configuration Drift Detection
```
Detect configuration drift for CSS {{resource_id}}:
- Expected JVM heap: {{expected_jvm_heap}}GB
- Actual JVM heap: {{actual_jvm_heap}}GB
- Expected shard count: {{expected_shards}}
- Actual shard count: {{actual_shards}}
- Expected replica count: {{expected_replicas}}
- Actual replica count: {{actual_replicas}}
Recommend reconciliation.

CSS drift: JVM parameters, ILM policies, allocation awareness
```

---

## Appendix: CSS-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{resource_id}}` | CSS cluster ID | `csse9-aabb1234` |
| `{{cluster_status}}` | Cluster health | `green` |
| `{{jvm_heap_usage}}` | JVM heap percent | `75.5` |
| `{{index_name}}` | Index name | `logs-2024-01` |
| `{{shard_count}}` | Shard count | `15` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance*
