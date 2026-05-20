# CES Observability Integration — Huawei Cloud Cloud Eye Service

## Three-Layer Architecture

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│ Metrics   │────▶│  Logs    │────▶│ Traces   │
│  (CES)    │     │  (LTS)   │     │  (AOM)   │
└──────────┘     └──────────┘     └──────────┘
                       ▼
              ┌─────────────────┐
              │ Unified Report  │
              └─────────────────┘
```

## Metrics → Logs Linkage

| CES Anomaly | LTS Query Target | LogQL Pattern | Purpose |
|-------------|-----------------|---------------|---------|
| CPU spike | Application error logs | `{applog=~"ERROR\|FATAL"}` | Confirm error burst causing CPU surge |
| Memory leak | Application memory logs | `{log=~"OutOfMemoryError\|memory"}` | Confirm allocation pattern |
| Connection pool exhaustion | Database access logs | `{log=~"connection.*timeout\|pool"}` | Confirm connection leak source |
| 5xx error spike | Access/error logs | `{log=~"HTTP [5]\d{2}"}` | Confirm dropped request details |
| Disk I/O high | System logs | `{log=~"iowait\|disk.*error"}` | Confirm I/O bottleneck cause |

## Metrics → Traces Linkage

| CES Anomaly | AOM Trace Target | Purpose |
|-------------|-----------------|---------|
| CPU spike | Application trace | Locate hot methods in application code |
| Latency increase | RPC/HTTP trace | Locate bottleneck service or dependency |
| Error rate increase | Error trace | Locate error root cause |

## Degradation Strategy

If AOM/LTS skills unavailable:

1. Use CLI directly: `hcloud ces list-metrics`, `hcloud lts list-log-groups`
2. Use OpenAPI SDK directly for metric and log queries
3. Provide console link for manual troubleshooting
4. Document the missing correlation in report and recommend enabling AOM/LTS

## Unified Diagnosis Report Fields

| Field | Source | Description |
|-------|--------|-------------|
| `report_id` | Generated | UUID v4 tracking ID |
| `timestamp` | CES | Alarm trigger time |
| `alarm_source` | CES | Original alarm rule name |
| `resource_id` | CES | Instance ID |
| `resource_status` | Product Skill | Current resource state |
| `metric_value` | CES | Alarm metric value |
| `metric_trend` | CES | 1h trend analysis |
| `anomaly_patterns` | This skill | Detected anomaly patterns from monitoring.md |
| `log_findings` | LTS | Relevant log entries |
| `trace_findings` | AOM | Trace-based diagnosis findings |
| `correlated_alarms` | CES | Other alarms on same resource |
| `root_cause` | Comprehensive | Primary root cause |
| `recommendation` | Comprehensive | Actionable fix suggestions |
| `delegated_skills` | Agent | List of Skills invoked |
