# Prompts — Huawei Cloud CTS

> **Purpose:** Structured prompts for CTS (Cloud Trace Service) AIOps operations.
> **Version:** 1.0.0
> **Status:** Reference document

---

## 1. Diagnostic Prompts

### 1.1 Tracker Health Check
```
Analyze CTS tracker {{resource_id}} health status:
- Tracker type: {{tracker_type}} (system/custom)
- Tracker status: {{tracker_status}} (enabled/disabled)
- Events tracked: {{events_tracked}}/day
- Storage used: {{storage_used}}GB / {{storage_quota}}GB
- Retention: {{retention_days}} days
- Last event: {{last_event_time}}
Determine tracker health and recommend actions.

Applicable CES metrics: SYS.CTS.trace_event_count, SYS.CTS.api_call_count, AGT.CTS.storage_usage
```

### 1.2 Event Volume Analysis
```
Analyze CTS event volume on {{resource_id}}:
- Events per day: {{daily_events}} (baseline: {{baseline_events}})
- Events by service: {{events_by_service}}
- Events by type: {{events_by_type}} (write: {{write_count}}, read: {{read_count}})
- Volume trend: {{volume_trend}}% (up/down/stable)
- Peak hour: {{peak_hour}} ({{peak_events}} events)
Identify unusual activity.

CTS events: API calls, management operations, data changes, policy changes
```

### 1.3 API Call Pattern Analysis
```
Analyze CTS API call patterns on {{resource_id}}:
- Total API calls: {{api_calls}}/day
- Failed calls: {{failed_calls}} ({{error_rate}}%)
- Top APIs: {{top_apis}}
- Top users: {{top_users}}
- Top resources: {{top_resources}}
- Geographic distribution: {{geo_dist}}
Identify access patterns.

CTS API: management console, SDK, API Explorer, CLI (hcloud)
```

### 1.4 Trace Query Analysis
```
Analyze CTS trace query on {{resource_id}}:
- Query latency P50: {{latency_p50}}ms, P99: {{latency_p99}}ms
- Query range: {{query_range}} hours
- Results returned: {{result_count}} events
- Query type: {{query_type}} (simple/aggregated/export)
- Filters: {{filter_count}} applied
Diagnose query performance.

CTS query: time range, filters, resource type, user, event type
```

### 1.5 Storage Utilization Analysis
```
Analyze CTS storage on {{resource_id}}:
- Storage used: {{storage_used}}GB / {{storage_quota}}GB ({{utilization}}%)
- Daily ingestion: {{daily_ingestion}}GB
- Compression ratio: {{compression_ratio}}:1
- Retention: {{retention_days}} days
- Projected exhaustion: {{exhaustion_date}}
Assess storage risk.

CTS storage: event logs, indexes, aggregations, OBS archival
```

---

## 2. Root Cause Analysis Prompts

### 2.1 Security Event Analysis
```
Analyze CTS security event on {{resource_id}}:
- Event type: {{event_type}} (login/delete/permission_change)
- User: {{user_id}} ({{user_type}})
- IP: {{source_ip}} ({{geo_location}})
- Time: {{event_time}}
- Resource affected: {{resource_type}} {{resource_id}}
- Risk level: {{risk_level}} (low/medium/high/critical)
Assess security impact.

CTS security events: failed login, privilege escalation, data deletion, policy modification
```

### 2.2 Operation Audit Analysis
```
Analyze CTS operation audit on {{resource_id}}:
- Operation: {{operation_type}}
- Service: {{service_name}}
- User: {{user_id}}
- Time: {{operation_time}}
- Resource: {{resource_type}} {{resource_id}}
- Result: {{operation_result}} (success/failure)
- Changes: {{changes_summary}}
Document operational impact.

CTS audit: configuration changes, resource lifecycle, access control changes
```

### 2.3 Data Deletion Analysis
```
Analyze CTS data deletion on {{resource_id}}:
- Deleted resource: {{resource_type}} {{resource_id}}
- Deletion time: {{deletion_time}}
- Deleted by: {{user_id}}
- Deletion method: {{deletion_type}} (manual/automated)
- Backup available: {{backup_available}}
- Related events: {{related_events}}
Assess data loss risk.

CTS deletion: cascade deletion, force delete, deletion after expiration
```

### 2.4 Compliance Violation Analysis
```
Analyze CTS compliance violation on {{resource_id}}:
- Violation type: {{violation_type}}
- Event timestamp: {{event_time}}
- User: {{user_id}}
- Resource: {{resource_type}} {{resource_id}}
- Policy violated: {{policy_name}}
- Compliance framework: {{framework}} (ISO/SOC/PCI-DSS)
Identify remediation needs.

CTS compliance: data residency, access control, audit trail, retention requirements
```

### 2.5 Cost Spike Analysis
```
Analyze CTS cost spike on {{resource_id}}:
- Current cost: {{current_cost}} CNY/day (baseline: {{baseline_cost}} CNY/day)
- Cost increase: {{cost_increase}}% ({{cost_delta}} CNY)
- Primary driver: {{cost_driver}}
- Affected period: {{affected_period}}
- Event volume change: {{volume_change}}%
Identify cost anomaly.

CTS cost: storage, API calls, OBS archival, cross-region transfer
```

---

## 3. Capacity Prompts

### 3.1 Storage Capacity Forecast
```
Forecast CTS storage for {{resource_id}}:
- Current storage: {{storage_used}}GB
- Daily ingestion: {{daily_ingestion}}GB
- Compression ratio: {{compression_ratio}}:1
- Retention period: {{retention_days}} days
- Storage quota: {{storage_quota}}GB
- Projected exhaustion: {{exhaustion_date}}
Recommend storage optimization.

CTS storage: event logs, index storage, OBS archival, retention tuning
```

### 3.2 Retention Policy Optimization
```
Optimize CTS retention for {{resource_id}}:
- Current retention: {{retention_days}} days
- Compliance requirement: {{compliance_retention}} days
- Storage tier: {{storage_tier}} (LTS/OBS)
- Archive policy: {{archive_policy}}
- Cost per GB: {{cost_per_gb}} CNY
- Recommended retention: {{recommended_retention}} days
Balance compliance and cost.

CTS retention: regulatory requirements, internal policy, storage tiering
```

### 3.3 Event Collection Planning
```
Plan CTS event collection for {{resource_id}}:
- Current events/day: {{daily_events}}
- Services tracked: {{services_tracked}} / {{total_services}}
- Coverage gap: {{coverage_gap}}
- Growth trend: {{growth_rate}}%/month
- API rate limit: {{api_limit}}/day
Recommend collection optimization.

CTS collection: service coverage, event filtering, sampling, aggregation
```

### 3.4 OBS Archival Planning
```
Plan CTS OBS archival for {{resource_id}}:
- LTS storage: {{lts_storage}}GB
- OBS archival: {{obs_storage}}GB
- OBS bucket: {{obs_bucket}}
- Archival format: {{archival_format}}
- Retrieval time: {{retrieval_time}} (standard: {{standard_hours}}h, expedited: {{expedited_min}}m)
- Archival cost: {{archival_cost}} CNY/month
Optimize archival strategy.

CTS archival: OBS vs LTS, retrieval tiers, format selection
```

---

## 4. Availability Prompts

### 4.1 Tracker Availability Check
```
Check CTS tracker availability on {{resource_id}}:
- Tracker status: {{tracker_status}}
- Events received: {{events_received}}/hour
- Events stored: {{events_stored}}/hour
- Storage health: {{storage_health}}
- OBS connection: {{obs_connection_status}}
- Last event: {{last_event_time}}
Report availability status.

CTS availability: tracker enabled, event ingestion, storage capacity, OBS connectivity
```

### 4.2 Audit Trail Integrity
```
Verify CTS audit trail integrity on {{resource_id}}:
- Event chain: {{event_chain_status}} (intact/broken)
- Sequence gaps: {{sequence_gaps}}
- Event integrity: {{event_integrity}}%
- Tampering detection: {{tampering_detected}}
- Log cryptographic sign: {{log_signed}}
Validate audit reliability.

CTS integrity: sequence numbering, hash chain, digital signature, tamper detection
```

### 4.3 Cross-Region Analysis
```
Analyze CTS cross-region tracking on {{resource_id}}:
- Source regions: {{source_regions}}
- Aggregated region: {{aggregate_region}}
- Replication lag: {{replication_lag}} minutes
- Event count: {{event_count}} / day
- Transfer cost: {{transfer_cost}} CNY/month
Optimize cross-region setup.

CTS cross-region: centralized tracking, regional aggregation, data residency
```

### 4.4 SLA Compliance Report
```
Report CTS SLA compliance for {{resource_id}}:
- Event availability: {{availability}}% (target: {{target}}%)
- Query success rate: {{query_success_rate}}%
- Query latency P99: {{latency_p99}}ms (target: {{target_latency}}ms)
- Trace completeness: {{completeness}}%
- Data retention: {{retention_days}} days (target: {{target_retention}} days)
Report SLA violations.

CTS SLA: event capture, query performance, storage retention, data integrity
```

---

## 5. Inspection Prompts

### 5.1 Routine Health Inspection
```
Perform routine CTS inspection:
- List all CTS trackers in {{scope}}
- Check tracker status = enabled
- Verify storage utilization < {{threshold_util}}%
- Identify event gaps > {{gap_threshold}} minutes
- Check OBS archival status
- Verify all critical services tracked
- Check for event drops
Report findings.

Scope: region, project, organization
```

### 5.2 Security Compliance Check
```
Audit CTS security compliance:
- Tracker enabled: {{tracker_enabled}}
- Key operations tracked: {{key_operations}} (all/partial/none)
- Sensitive data masking: {{masking_enabled}}
- Access control: {{access_control}} (org admin only)
- Retention period: {{retention_days}} days (compliance: {{compliance_days}})
- OBS encryption: {{obs_encryption}} (AES256/KMS)
Report compliance status.

Severity: Critical = tracker disabled, High = incomplete tracking, Medium = short retention
```

### 5.3 Coverage Audit
```
Audit CTS service coverage on {{resource_id}}:
- Total services: {{total_services}}
- Services tracked: {{tracked_services}} ({{coverage}}%)
- Services missing: {{missing_services}}
- Critical services: {{critical_services}} (all tracked: {{critical_tracked}})
- Tracked operations: {{tracked_ops}} / {{total_ops}}
Identify coverage gaps.

CTS coverage: Huawei Cloud services, management operations, data operations
```

### 5.4 Cost Allocation Audit
```
Audit CTS cost allocation:
- Current cost: {{current_cost}} CNY/day
- Cost by service: {{cost_by_service}}
- Cost by region: {{cost_by_region}}
- Cost by operation: {{cost_by_operation}}
- Budget vs actual: {{budget_vs_actual}} CNY
- Anomaly detected: {{anomaly_detected}}
Allocate costs to projects.

CTS cost: storage, API operations, OBS archival, cross-region transfer
```

### 5.5 Configuration Drift Detection
```
Detect CTS configuration drift:
- Tracker: {{tracker_id}}
- Expected status: {{expected_status}}
- Actual status: {{actual_status}}
- Expected retention: {{expected_retention}} days
- Actual retention: {{actual_retention}} days
- Expected services: {{expected_services}}
- Actual services: {{actual_services}}
Recommend reconciliation.

CTS drift: tracker enabled/disabled, retention changes, service filter changes
```

---

## Appendix: CTS-Specific Placeholders

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{{resource_id}}` | CTS tracker ID | `system-abc123` |
| `{{tracker_type}}` | Tracker type | `system` |
| `{{event_type}}` | Event type | `create` |
| `{{user_id}}` | User ID | `user-12345` |
| `{{service_name}}` | Cloud service name | `ECS` |

---

*Prompts version 1.0.0 — for AIOps L4 compliance*
