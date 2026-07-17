# AIOps Patterns — CTS

> **Purpose**: CTS-specific anomaly detection patterns for audit and traceability.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `trace_volume_near_quota` | Daily trace volume > 80% of quota | Warning | Optimize or scale quota |
| `index_latency_high` | Index latency > 500ms | Warning | Check storage backend |
| `query_timeout_high` | Query timeout rate > 1% | Warning | Optimize query filters |
| `trail_delivery_lag` | Delivery delay > 60s sustained | Warning | Check OBS/SMN/LTS connectivity |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `trace_volume_spike` | Volume > 3x 7-day baseline | Warning | Investigate anomalous activity |
| `error_trace_increase` | Error traces > 2x normal rate | Warning | Check for business exceptions |
| `delete_operation_spike` | Bulk delete events sudden increase | Warning | Verify legitimate cleanup vs attack |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `tracking_group_quota_reach` | New tracking groups near limit (>90%) | Warning | Clean up or request quota increase |
| `trace_data_gap` | Gap in trace data > 5min during business hours | Warning | Check CTS service health |
| `delivery_target_changed` | Trail target changed unexpectedly | Warning | Verify no unauthorized config drift |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `error_trace_correlation` | Error traces + specific resource type | Warning | Inspect affected resource |
| `volume_anomaly_correlation` | Volume spike + delivery latency increase | Warning | Check system load and OBS health |
| `auth_failure_cluster` | Multiple auth failures from same IP in short window | Warning | Check for brute-force or credential stuffing |

---

## 2. Detection Signal Sources

| Signal | Source | Query / Metric |
|--------|--------|----------------|
| Trace volume | CTS query API | `count(*)` over time window |
| Error trace rate | CTS query API | `event_type=system AND status=fail` |
| Delivery lag | CES metric `cts_delivery_latency` | Cloud Eye |
| Quota usage | CTS API `ListTrails` + quota API | `trail_count / max_quota` |
| Auth failures | CTS query | `event_name=*Login* AND status=fail` |

---

## 3. Cross-Skill Delegation

| Anomaly Type | Delegate To | Reason |
|--------------|-------------|--------|
| OBS delivery failure | `huaweicloud-obs-ops` | OBS bucket or permission issue |
| SMN notification failure | `huaweicloud-swr-ops` (SMN) | Topic/subscription misconfig |
| LTS ingestion lag | `huaweicloud-lts-ops` | Log group capacity |
| IAM auth anomaly | `huaweicloud-iam-ops` | Permission or credential issue |
| ECS-related trace spike | `huaweicloud-ecs-ops` | Resource-level investigation |
