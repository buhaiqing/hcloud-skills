# Monitoring — Huawei Cloud LTS

## CES Metrics

LTS reports the following metrics to Cloud Eye Service (CES) for monitoring log service health:

| Metric | Description | Unit | Aggregation |
|--------|-------------|------|-------------|
| `lts_log_volume` | Log ingestion volume | Bytes | Sum |
| `lts_log_request_count` | Log ingestion request count | Count | Sum |
| `lts_log_search_count` | Log search query count | Count | Sum |
| `lts_log_search_latency` | Log search latency | ms | Avg, Max, P95 |
| `lts_transfer_success_count` | Successful transfer count | Count | Sum |
| `lts_transfer_failed_count` | Failed transfer count | Count | Sum |
| `lts_storage_usage` | Current storage usage | Bytes | Max |
| `lts_index_volume` | Index storage volume | Bytes | Sum |

## Recommended Alarm Rules

| Alarm Name | Metric | Threshold | Duration | Severity | Action |
|-----------|--------|-----------|----------|----------|--------|
| Log Ingestion Spike | `lts_log_volume` | > 2x baseline | 5 min | Warning | Check for runaway logging |
| Log Ingestion Drop | `lts_log_volume` | < 0.1x baseline | 10 min | Critical | ICAgent may be down |
| Transfer Failed | `lts_transfer_failed_count` | > 0 | 5 min | Critical | Check OBS/DMS target |
| Search Latency High | `lts_log_search_latency` | > 5000 ms | 5 min | Warning | Optimize index or narrow queries |
| Storage Near Limit | `lts_storage_usage` | > 80% quota | 30 min | Warning | Reduce TTL or increase quota |
| Index Volume Spike | `lts_index_volume` | > 1.5x baseline | 5 min | Warning | Review index field configuration |

## Integration with CES

Create CES alarms that reference LTS metrics:

```bash
# Create alarm for log ingestion drop
hcloud CES CreateAlarm \
  --cli-region="{{env.HW_REGION_ID}}" \
  --alarm_name="lts-ingestion-drop" \
  --metric="lts_log_volume" \
  --namespace="SYS.LTS" \
  --condition="aggregation=average,operator=lt_threshold,value=0.1,period=300,count=2" \
  --alarm_enabled=true
```

## Anomaly Patterns (≥4 patterns)

### Pattern 1: Log Volume Spike
- **Detection**: `lts_log_volume` > 2x baseline for ≥5 minutes
- **Possible Causes**: Application error loop, DDoS attack, misconfigured logging level (DEBUG → ERROR)
- **Action**: Query recent logs for error patterns; check application deployments

### Pattern 2: Log Volume Drop to Zero
- **Detection**: `lts_log_volume` < 0.1x baseline for ≥10 minutes
- **Possible Causes**: ICAgent stopped, network partition, ECS instance down
- **Action**: Check ICAgent status on source hosts; verify network connectivity

### Pattern 3: Log Transfer Failures
- **Detection**: `lts_transfer_failed_count` > 0 for ≥5 minutes
- **Possible Causes**: OBS bucket deleted, bucket policy changed, network issue
- **Action**: Verify OBS bucket exists and permissions; retry transfer

### Pattern 4: Search Latency Degradation
- **Detection**: `lts_log_search_latency` P95 > 5000ms for ≥5 minutes
- **Possible Causes**: Large unstructured data, missing index, complex SQL queries
- **Action**: Review index configuration; simplify queries; consider narrowing time range

## Dashboard Recommendations

Create an LTS operational dashboard with:
- Log ingestion volume (area chart, 24h)
- Transfer success/fail rate (bar chart)
- Search latency P50/P95 (line chart)
- Top 10 log-producing streams (table)
- Storage usage vs quota (gauge)

## Log Audit via CTS

All LTS management operations (CreateLogGroup, DeleteLogGroup, CreateTransfer, etc.) are recorded in CTS.
Query CTS for audit trail:
```
hcloud CTS ListTraces --service_type="LTS" --limit=50
```
