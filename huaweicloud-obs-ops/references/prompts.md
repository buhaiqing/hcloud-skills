# Prompts — Huawei Cloud OBS

> **Purpose:** Structured prompts for OBS (Object Storage Service) AIOps operations.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 Bucket Health Check
```
Analyze OBS bucket {{resource_id}} health status:
- Bucket name: {{bucket_name}}
- Storage used: {{storage_used}}GB
- Object count: {{object_count}}
- Request count: {{request_count}}/day (GET: {{get_count}}, PUT: {{put_count}})
- Data traffic: {{traffic_in}}GB in, {{traffic_out}}GB out
- Error rate: {{error_rate}}% ({{error_count}}/{{total_request}})
Determine bucket health and recommend actions.

Applicable CES metrics: SYS.OBS.storage_amount, SYS.OBS.request_count, AGT.OBS.traffic
```

### 1.2 Storage Usage Analysis
```
Analyze OBS storage usage on {{resource_id}}:
- Total storage: {{storage_used}}TB / {{storage_quota}}TB ({{utilization}}%)
- Storage by folder: {{folder_distribution}}
- Large objects: {{large_object_count}} (> {{large_threshold}}GB)
- Expired objects: {{expired_count}}
- Versioning overhead: {{versioning_size}}GB
- Multipart uploads pending: {{pending_mpu}}
Identify storage optimization opportunities.

OBS storage: standard, warm, cold tiering, versioning, lifecycle policies
```

### 1.3 Traffic Anomaly Detection
```
Detect OBS traffic anomaly on {{resource_id}}:
- Current traffic: {{current_traffic}}GB/day (baseline: {{baseline_traffic}}GB/day)
- Peak hour: {{peak_hour}} (traffic: {{peak_traffic}}GB/h)
- Traffic by operation: GET {{get_traffic}}%, PUT {{put_traffic}}%
- Traffic by region: {{region_distribution}}
- CDN traffic: {{cdn_traffic}}GB
Identify unexpected traffic patterns.

OBS traffic: data retrieval, uploads, CDN egress, cross-region replication
```

### 1.4 Request Rate Analysis
```
Analyze OBS request rate on {{resource_id}}:
- Total requests: {{request_count}}/day
- GET requests: {{get_count}} ({{get_rate}}%)
- PUT requests: {{put_count}}
- DELETE requests: {{delete_count}}
- 4xx errors: {{error_4xx_count}} ({{error_4xx_rate}}%)
- 5xx errors: {{error_5xx_count}} ({{error_5xx_rate}}%)
Diagnose request patterns and errors.

OBS request errors: 403 forbidden, 404 not found, 500 internal, 503 slow down
```

### 1.5 Data Redundancy Check
```
Check OBS data redundancy on {{resource_id}}:
- Bucket name: {{bucket_name}}
- Storage class: {{storage_class}} (Standard/Warm/Cold)
- Replication: {{replication_enabled}} (cross-region: {{cross_region}})
- Versioning: {{versioning_status}}
- Multipart integrity: {{multipart_integrity}}%
Assess data protection level.

OBS redundancy: multi-AZ storage, cross-region replication, versioning
```

---

## 2. Root Cause Analysis Prompts

### 2.1 403 Access Denied Analysis
```
Analyze OBS 403 access denied on {{resource_id}}:
- Bucket: {{bucket_name}}
- Error count: {{error_count}} in past {{time_window}}
- IAM policy: {{iam_policy}}
- Bucket policy: {{bucket_policy}}
- ACL: {{acl_config}}
- Request authentication: {{auth_type}} (IAM/STS/Presigned)
Identify permission issue.

OBS 403 causes: incorrect IAM policy, bucket policy conflict, ACL restrictions, expired presigned URL
```

### 2.2 Upload Failure Analysis
```
Analyze OBS upload failure on {{resource_id}}:
- Failed uploads: {{failed_uploads}}/{{total_uploads}}
- Error codes: {{error_codes}}
- Failure by size: {{size_distribution}}
- Multipart failures: {{mpu_failures}}
- Upload latency: {{upload_latency}}ms (P99)
Diagnose upload issues.

OBS upload failures: network timeout, part size mismatch, incorrect ETag, storage quota
```

### 2.3 Download Latency Analysis
```
Analyze OBS download latency on {{resource_id}}:
- Download latency P50: {{latency_p50}}ms, P99: {{latency_p99}}ms
- First byte latency: {{first_byte_latency}}ms
- CDN hit rate: {{cdn_hit_rate}}%
- Object size: {{avg_object_size}}MB
- Region: {{region}} → {{client_region}}
Identify latency bottlenecks.

OBS latency: bucket region, object size, CDN caching, network path
```

### 2.4 Consistency Issue Analysis
```
Analyze OBS consistency issue on {{resource_id}}:
- Stale read count: {{stale_read_count}}
- Missing object reports: {{missing_reports}}
- Last update time: {{last_update}}
- Replication status: {{replication_status}}
- Event notifications: {{event_notification_status}}
Diagnose consistency problem.

OBS consistency: eventual consistency window, replication lag, event notification delays
```

### 2.5 Bucket Quota Analysis
```
Analyze OBS bucket quota on {{resource_id}}:
- Storage quota: {{storage_quota}}TB
- Current usage: {{storage_used}}TB ({{utilization}}%)
- Quota by folder: {{folder_quota}}
- Daily growth: {{daily_growth}}GB
- Projected exhaustion: {{exhaustion_date}}
Identify quota risks.

OBS quota: storage quota per bucket, object count limit, request rate limit
```

---

## 3. Capacity Prompts

### 3.1 Storage Capacity Forecast
```
Forecast OBS storage for {{resource_id}}:
- Current storage: {{storage_used}}TB
- Daily growth: {{daily_growth}}GB
- Monthly growth: {{monthly_growth}}TB
- Storage class mix: Standard {{std_pct}}%, Warm {{warm_pct}}%, Cold {{cold_pct}}%
- Projected 6-month usage: {{projected_6m}}TB
Recommend tiering optimization.

OBS tiering: lifecycle policies, storage class optimization, archival
```

### 3.2 Cost Optimization Analysis
```
Analyze OBS cost optimization for {{resource_id}}:
- Current cost: {{monthly_cost}} CNY
- Storage cost: {{storage_cost}} ({{storage_used}}TB × {{storage_rate}}/GB)
- Traffic cost: {{traffic_cost}} ({{traffic_out}}GB × {{traffic_rate}}/GB)
- Request cost: {{request_cost}} ({{request_count}} × {{request_rate}}/10k)
- Potential savings: {{potential_savings}} CNY/month
Provide tiering and policy recommendations.

OBS cost: storage class selection, lifecycle policies, request optimization
```

### 3.3 Request Rate Planning
```
Plan OBS request capacity for {{resource_id}}:
- Current QPS: {{current_qps}} (peak: {{peak_qps}})
- Request limit: {{request_limit}}/s
- QPS by operation: GET {{get_qps}}%, PUT {{put_qps}}%
- Burst capacity: {{burst_qps}}
Recommend request optimization.

OBS rate limits: per-bucket limits, prefix-based limits, concurrent multipart
```

### 3.4 CDN Usage Optimization
```
Optimize OBS CDN usage for {{resource_id}}:
- CDN traffic: {{cdn_traffic}}GB/month ({{cdn_percent}}% of total)
- CDN hit rate: {{cdn_hit_rate}}%
- Cache TTL: {{cache_ttl}} seconds
- Origin request count: {{origin_requests}}
- Miss traffic: {{miss_traffic}}GB
Recommend CDN configuration.

OBS CDN: cache rules, TTL tuning, cache key optimization, prefetch
```

---

## 4. Availability Prompts

### 4.1 Data Protection Check
```
Check OBS data protection on {{resource_id}}:
- Replication status: {{replication_status}}
- Cross-region replication: {{cross_region_replication}} ({{target_region}})
- Versioning: {{versioning_status}} ({{version_count}} versions)
- Lifecycle policy: {{lifecycle_policy}}
- Last backup: {{last_backup}}
Assess protection level.

OBS protection: versioning, cross-region replication, lifecycle management
```

### 4.2 Access Pattern Analysis
```
Analyze OBS access patterns on {{resource_id}}:
- Read/Write ratio: {{read_write_ratio}}
- Access frequency: {{hot_count}} hot, {{warm_count}} warm, {{cold_count}} cold
- Access by hour: {{hourly_distribution}}
- Access by client: {{client_distribution}}
- Prefix access pattern: {{prefix_pattern}}
Recommend lifecycle optimization.

OBS access: hot→warm→cold migration, expiry policies, storage class selection
```

### 4.3 Disaster Recovery Readiness
```
Assess OBS DR readiness for {{resource_id}}:
- Replication: {{replication_type}} (none/cross-region/same-region)
- RPO: {{rpo_achieved}}min (target: {{rpo_target}}min)
- RTO: {{rto_achieved}}min (target: {{rto_target}}min)
- Last DR test: {{last_dr_test}}
- Cross-region copy: {{cross_region_copy}}% complete
Evaluate DR capabilities.

OBS DR: cross-region replication, versioning, bucket copy, CBR backup
```

### 4.4 SLA Compliance Report
```
Report OBS SLA compliance for {{resource_id}}:
- Availability: {{availability}}% (target: {{target_availability}}%)
- Request success: {{request_success_rate}}%
- Latency P99: {{latency_p99}}ms (target: < {{target_latency}}ms)
- Data durability: {{durability}}% (target: {{target_durability}}%)
Report SLA violations.

OBS SLA: 99.9% availability, 99.999999999% durability
```

---

## 5. Inspection Prompts

### 5.1 Routine Health Inspection
```
Perform routine OBS inspection:
- List all buckets in {{scope}}
- Check storage utilization > {{storage_threshold}}%
- Identify hot buckets (QPS > {{qps_threshold}})
- Flag buckets without versioning
- Check buckets without lifecycle policies
- Verify replication configured
- Check public access settings
Report findings.

Scope: region, project, tag-based resource group
```

### 5.2 Security Compliance Check
```
Audit OBS security compliance:
- Public access: {{public_access}} (blocked/restricted/permissive)
- Encryption at rest: {{encryption_at_rest}} (AES256/KMS)
- Encryption in transit: {{encryption_in_transit}} (TLS version)
- Bucket policy: {{bucket_policy_status}}
- ACL: {{acl_status}} (owner only/private/public-read)
- CORS: {{cors_configured}}
- KMS key rotation: {{kms_rotation}}
Report compliance status.

Severity: Critical = public write, High = no encryption, Medium = open read
```

### 5.3 Cost Anomaly Detection
```
Detect OBS cost anomalies:
- Current cost: {{current_cost}} CNY (baseline: {{baseline_cost}} CNY)
- Cost by bucket: {{bucket_costs}}
- Cost by component: storage {{storage_pct}}%, traffic {{traffic_pct}}%, requests {{request_pct}}%
- Anomaly bucket: {{anomaly_bucket}} ({{anomaly_delta}}% vs baseline)
- Traffic spike: {{traffic_spike}}GB ({{spike_percent}}% increase)
Identify cost anomalies.

OBS cost anomaly: unexpected traffic, large uploads, access pattern change
```

### 5.4 Lifecycle Policy Audit
```
Audit OBS lifecycle policies:
- Total buckets: {{bucket_count}}
- Buckets with lifecycle: {{lifecycle_buckets}}
- Policy rules: {{policy_rules}}
- Transition actions: {{transition_actions}} (Standard→Warm: {{std_warm}}, Warm→Cold: {{warm_cold}})
- Expiration actions: {{expiration_actions}}
- Estimated savings: {{estimated_savings}} CNY/month
Recommend policy updates.

OBS lifecycle: transition to warm/cold, object expiration, abort incomplete multipart
```

### 5.5 Configuration Drift Detection
```
Detect OBS configuration drift:
- Bucket: {{bucket_name}}
- Expected versioning: {{expected_versioning}}
- Actual versioning: {{actual_versioning}}
- Expected encryption: {{expected_encryption}}
- Actual encryption: {{actual_encryption}}
- Expected ACL: {{expected_acl}}
- Actual ACL: {{actual_acl}}
Recommend reconciliation.

OBS drift: versioning toggle, encryption changes, ACL modifications
```

---

## Appendix: OBS-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{resource_id}}` | OBS bucket name | `my-bucket` |
| `{{bucket_name}}` | Bucket name | `data-lake-2024` |
| `{{storage_used}}` | Storage used | `12.5` |
| `{{storage_class}}` | Storage class | `STANDARD` |
| `{{replication_status}}` | Replication status | `complete` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance*
