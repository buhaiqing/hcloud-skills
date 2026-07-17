# Prompts — Huawei Cloud LTS

> **Purpose:** Structured prompts for LTS (Log Tank Service) AIOps operations.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 Instance Health Check
```
Analyze LTS instance {{resource_id}} health status:
- Log groups: {{log_group_count}}
- Log streams: {{log_stream_count}}
- Ingestion rate: {{ingestion_rate}}MB/s
- Log volume: {{log_volume}}GB/day
- Storage used: {{storage_used}}GB / {{storage_quota}}GB
- Query latency P99: {{query_latency_p99}}ms
Determine instance health and recommend actions.

Applicable CES metrics: SYS.LTS.log_amount, SYS.LTS.query_delay, AGT.LTS.log_group_count
```

### 1.2 Log Volume Anomaly Detection
```
Detect LTS log volume anomaly on {{resource_id}}:
- Current volume: {{current_volume}}GB/day (baseline: {{baseline_volume}}GB/day)
- Anomaly type: {{anomaly_type}} (spike/drop/gradual)
- Affected log groups: {{affected_log_groups}}
- Top contributing streams: {{top_streams}}
- Timestamp pattern: {{time_pattern}}
Identify cause of volume change.

LTS volume anomaly: application debug enabled, deployment event, traffic spike, sampling change
```

### 1.3 Query Latency Analysis
```
Analyze LTS query latency on {{resource_id}}:
- Query latency P50: {{latency_p50}}ms, P99: {{latency_p99}}ms (baseline: {{baseline_p99}}ms)
- Query complexity: {{query_type}} (simple/aggregated/regex)
- Time range: {{query_range}} minutes
- Data scanned: {{data_scanned}}GB
- Result size: {{result_size}}KB
Identify query bottlenecks.

LTS latency: time range size, full-text search, aggregation complexity, regex patterns
```

### 1.4 Storage Quota Analysis
```
Analyze LTS storage on {{resource_id}}:
- Storage used: {{storage_used}}GB / {{storage_quota}}GB ({{utilization}}%)
- Retention period: {{retention_days}} days
- Daily ingestion: {{daily_ingestion}}GB
- Compressed size: {{compressed_size}}GB
- Index size: {{index_size}}GB
- Projected exhaustion: {{exhaustion_date}}
Assess storage risk.

LTS storage: ingestion volume, compression ratio, retention period, index overhead
```

### 1.5 Ingestion Rate Analysis
```
Analyze LTS ingestion rate on {{resource_id}}:
- Current rate: {{ingestion_rate}}MB/s ({{ingestion_rate_daily}}GB/day)
- Peak rate: {{peak_rate}}MB/s
- Rate limit: {{rate_limit}}MB/s
- Throttling events: {{throttle_count}}/hour
- Queue depth: {{queue_depth}}MB
Assess ingestion capacity.

LTS ingestion: per-log-group limits, shard-based ingestion, burst handling
```

---

## 2. Root Cause Analysis Prompts

### 2.1 Log Missing Analysis
```
Analyze missing logs on {{resource_id}}:
- Missing time range: {{missing_start}} to {{missing_end}}
- Affected log groups: {{affected_log_groups}}
- Affected log streams: {{affected_streams}}
- Agent status: {{agent_status}} (running/stopped/error)
- Collection configuration: {{collection_config}}
Identify gap cause.

LTS missing logs: agent failure, network issue, quota exceeded, filter rules
```

### 2.2 Query Timeout Analysis
```
Analyze LTS query timeout on {{resource_id}}:
- Query ID: {{query_id}}
- Query: {{query_pattern}}
- Timeout: {{timeout_ms}}ms
- Time range: {{query_range}} minutes
- Data size: {{data_scanned}}GB
- Full-text search: {{fulltext_search}} (yes/no)
Diagnose timeout cause.

LTS timeout: large time range, complex regex, unindexed field search, result set too large
```

### 2.3 Agent Health Analysis
```
Analyze LTS agent health on {{resource_id}}:
- Host: {{host_name}}
- Agent status: {{agent_status}}
- Last heartbeat: {{last_heartbeat}} minutes ago
- Log groups monitored: {{monitored_groups}}
- Collection errors: {{collection_errors}}/hour
- Dropped logs: {{dropped_logs}}/hour
Diagnose agent issues.

LTS agent: ICAgent, installation issues, configuration drift, resource constraints
```

### 2.4 Index Efficiency Analysis
```
Analyze LTS index efficiency on {{resource_id}}:
- Index fields: {{indexed_fields}}
- Full-text indexed: {{fulltext_fields}}
- Index storage: {{index_size}}GB
- Query patterns: {{query_patterns}}
- Unused indexes: {{unused_indexes}}
Recommend index optimization.

LTS indexing: field indexing vs full-text, index storage overhead, query pattern analysis
```

### 2.5 Cost Spike Analysis
```
Analyze LTS cost spike on {{resource_id}}:
- Current cost: {{current_cost}} CNY/day (baseline: {{baseline_cost}} CNY/day)
- Cost increase: {{cost_increase}}% ({{cost_delta}} CNY)
- Primary driver: {{cost_driver}} (storage/ingestion/query)
- Affected log groups: {{affected_groups}}
- Retention change: {{retention_change}} days
Identify cost driver.

LTS cost: ingestion volume, storage retention, query frequency, index storage
```

---

## 3. Capacity Prompts

### 3.1 Storage Capacity Forecast
```
Forecast LTS storage for {{resource_id}}:
- Current storage: {{storage_used}}GB
- Daily ingestion: {{daily_ingestion}}GB
- Compression ratio: {{compression_ratio}}:1
- Retention: {{retention_days}} days
- Storage quota: {{storage_quota}}GB
- Projected exhaustion: {{exhaustion_date}} at current rate
Recommend storage optimization.

LTS storage: compression, retention tuning, archival to OBS, log sampling
```

### 3.2 Ingestion Capacity Planning
```
Plan LTS ingestion capacity for {{resource_id}}:
- Current rate: {{current_rate}}MB/s
- Rate limit: {{rate_limit}}MB/s
- Peak rate: {{peak_rate}}MB/s
- Headroom: {{headroom}}% ({{headroom_mb}}MB/s)
- Daily volume: {{daily_volume}}GB
- Growth trend: {{growth_rate}}%/month
Recommend capacity scaling.

LTS capacity: ingestion rate limits, log group limits, shard configuration
```

### 3.3 Query Performance Planning
```
Plan LTS query capacity for {{resource_id}}:
- Current QPS: {{query_qps}} queries/s
- Avg latency: {{avg_latency}}ms
- P99 latency: {{p99_latency}}ms
- Concurrent queries: {{concurrent_queries}}
- Query complexity distribution: {{complexity_dist}}
- Storage size: {{storage_size}}GB
Recommend query optimization.

LTS query: parallel execution, index usage, result pagination, query limits
```

### 3.4 Log Group Optimization
```
Optimize LTS log groups for {{resource_id}}:
- Current groups: {{group_count}}
- Streams: {{stream_count}}
- Ingestion distribution: {{ingestion_dist}}
- Top groups by volume: {{top_groups}}
- Recommended consolidation: {{consolidation_plan}}
Reduce management overhead.

LTS log groups: per-application, per-environment, per-region organization
```

---

## 4. Availability Prompts

### 4.1 Log Availability Check
```
Check LTS log availability on {{resource_id}}:
- Active log groups: {{active_groups}}/{{total_groups}}
- Agent coverage: {{agent_coverage}}%
- Data completeness: {{completeness}}%
- Missing data windows: {{missing_windows}}
- Last successful ingestion: {{last_ingestion}}
Report availability status.

LTS availability: agent health, network connectivity, quota status, service health
```

### 4.2 Backup Status Verification
```
Verify LTS backup status for {{resource_id}}:
- Backup enabled: {{backup_enabled}}
- Last backup: {{last_backup}}
- Backup destination: {{backup_dest}} (OBS {{obs_bucket}})
- Backup retention: {{backup_retention}} days
- Restore test: {{restore_test_status}}
Validate backup completeness.

LTS backup: OBS archival, cross-region backup, long-term retention
```

### 4.3 Cross-Region Analysis
```
Analyze LTS cross-region access on {{resource_id}}:
- Primary region: {{primary_region}}
- Access from regions: {{access_regions}}
- Cross-region latency: {{cross_region_latency}}ms
- Data transfer: {{cross_region_transfer}}GB/month
- Cost impact: {{transfer_cost}} CNY/month
Optimize cross-region access.

LTS cross-region: dedicated line, CDN, regional aggregation
```

### 4.4 SLA Compliance Report
```
Report LTS SLA compliance for {{resource_id}}:
- Log availability: {{log_availability}}% (target: {{target_availability}}%)
- Query success rate: {{query_success_rate}}%
- Query latency P99: {{latency_p99}}ms (target: < {{target_latency}}ms)
- Data freshness: {{data_freshness}}s
- Ingestion success: {{ingestion_success_rate}}%
Report SLA violations.

LTS SLA: log availability, query latency, ingestion success, data retention
```

---

## 5. Inspection Prompts

### 5.1 Routine Health Inspection
```
Perform routine LTS inspection:
- List all LTS instances in {{scope}}
- Check storage utilization > {{storage_threshold}}%
- Identify log groups with no data > 24h
- Flag ingestion errors > {{error_threshold}}/hour
- Check agent heartbeat failures
- Verify backup status
- Check quota usage vs limits
Report findings.

Scope: region, project, tag-based resource group
```

### 5.2 Security Compliance Check
```
Audit LTS security compliance:
- IAM access: {{iam_access}} (logging only/no full access)
- Log access policy: {{access_policy}}
- Sensitive data masking: {{masking_enabled}}
- Access audit: {{access_audit_enabled}}
- Data encryption: {{encryption_at_rest}}
- Network isolation: {{vpc_private}}
Report compliance status.

Severity: High = no masking for sensitive fields, Medium = open access policy
```

### 5.3 Cost Optimization Scan
```
Scan LTS for cost optimization:
- Identify unused log groups (> 30 days no data)
- Find high-volume low-value logs
- Check retention period vs actual need
- Verify compression enabled
- Check index selectivity
- Review query frequency vs value
Provide action list with estimated savings.

LTS cost: ingestion volume, storage retention, index storage, query costs
```

### 5.4 Log Quality Inspection
```
Inspect LTS log quality on {{resource_id}}:
- Sampling rate: {{sampling_rate}}%
- Format consistency: {{format_consistency}}%
- Error log ratio: {{error_ratio}}%
- Sensitive fields: {{sensitive_fields}}
- Required fields: {{required_fields}}
- Timestamp coverage: {{timestamp_coverage}}%
Recommend quality improvements.

LTS log quality: structured logging, field completeness, timestamp accuracy
```

### 5.5 Configuration Drift Detection
```
Detect configuration drift for LTS {{resource_id}}:
- Expected retention: {{expected_retention}} days
- Actual retention: {{actual_retention}} days
- Expected ingestion: {{expected_ingestion}}MB/s
- Actual ingestion: {{actual_ingestion}}MB/s
- Expected sampling: {{expected_sampling}}%
- Actual sampling: {{actual_sampling}}%
Recommend reconciliation.

LTS drift: retention changes, sampling rate changes, filter rule modifications
```

---

## Appendix: LTS-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{resource_id}}` | LTS instance/group ID | `100degsa` |
| `{{log_group}}` | Log group name | `/ECS/application` |
| `{{log_stream}}` | Log stream name | `instance-01` |
| `{{ingestion_rate}}` | Ingestion rate MB/s | `15.5` |
| `{{query_latency_p99}}` | Query P99 latency ms | `2500` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance*
