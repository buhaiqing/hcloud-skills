# Observability Trinity — Huawei Cloud CES

> **Purpose**: Metrics → Logs → Traces linkage rules for CES (monitoring service itself).
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

## 1. Observability Trinity Overview

| Component | Data Source | Purpose |
|-----------|-------------|---------|
| Metrics | CES (self-monitoring) | Alarm evaluation latency, API success rate, notification delivery |
| Logs | LTS (CES operational logs) | Alarm state changes, evaluation errors, quota usage |
| Traces | APM (CES API traces) | API request flow, latency breakdown |

## 2. Linkage Rules

### 2.1 Metric → Log Linkage

| When CES metric alerts | Check LTS logs |
|------------------------|----------------|
| Alarm evaluation latency > 10s | CES operational logs for evaluation errors |
| API error rate > 1% | API gateway logs for 5xx errors |
| Notification delivery failure | LTS: alarm action trigger logs |
| Quota usage > 80% | CES quota utilization logs |

### 2.2 Log → Metric Linkage

| When LTS log pattern detected | Check CES metrics |
|------------------------------|-------------------|
| `alarm_evaluation_error` | Alarm evaluation latency metric |
| `api_5xx_error` | API success rate metric |
| `notification_delivery_failed` | Notification delivery metric |
| `quota_threshold_exceeded` | Quota utilization metrics |

### 2.3 Trace → Metric/Log Linkage

| When APM trace shows | Check metrics + logs |
|---------------------|---------------------|
| CES API latency > 2s | Alarm evaluation metrics |
| CES API errors | Error logs for specific API endpoint |
| Timeout in trace | Downstream service (CES backend) health |

## 3. Data Source Mapping

| Observable | CES Self-Monitoring | LTS Log Group | APM Trace |
|-----------|---------------------|---------------|-----------|
| Alarm evaluation | CES built-in | `ces-operational-alarms` | Yes |
| API performance | CES built-in | `ces-api-gateway` | Yes |
| Notification delivery | CES built-in | `ces-notifications` | No |
| Quota utilization | CES built-in | `ces-quota` | No |

## 4. Correlation Query Examples

### 4.1 Metric Alert → Find Related Logs

```bash
# Alarm evaluation latency spike
REGION="{{env.HW_REGION_ID}}"
LOG_GROUP="{{user.ces_log_group_id}}"

# 1. Confirm metric alert
hcloud ces query-metric-data \
  --namespace "SYS.CES" \
  --metric-name "alarm_evaluation_latency" \
  --dimension "region=$REGION" \
  --period 60 \
  --from "$(( $(date +%s) - 300 ))" \
  --to "$(date +%s)" \
  --output json

# 2. Query LTS for evaluation errors
hcloud lts query-log \
  --log-group-id "$LOG_GROUP" \
  --log-stream-id "ces-operational-alarms" \
  --start-time "$(( $(date +%s) * 1000 - 30 * 60 * 1000 ))" \
  --end-time "$(date +%s)" \
  --keywords "alarm_evaluation_error|timeout" \
  --output json
```

### 4.2 Log Pattern → Find Related Metrics

```bash
# API 5xx errors in logs
REGION="{{env.HW_REGION_ID}}"

# Query API error rate metric
hcloud ces query-metric-data \
  --namespace "SYS.CES" \
  --metric-name "api_error_rate" \
  --dimension "region=$REGION" \
  --period 60 \
  --from "$(( $(date +%s) - 600 ))" \
  --to "$(date +%s)" \
  --output json
```

### 4.3 Trace → Metric Correlation

```bash
# High latency trace on CES API
TRACE_ID="{{user.trace_id}}"

# Get trace details
hcloud apm query-trace \
  --trace-id "$TRACE_ID" \
  --output json

# Query CES API latency metric for the affected endpoint
hcloud ces query-metric-data \
  --namespace "SYS.CES" \
  --metric-name "api_latency_p95" \
  --dimension "region=$REGION" \
  --period 60 \
  --from "$(( $(date +%s) - 300 ))" \
  --to "$(date +%s)" \
  --output json
```

## 5. Trinity-Driven Diagnosis Workflow

```
[CES Metric Alert: alarm_evaluation_latency > 10s]
    │
    ├── 1. Query LTS: ces-operational-alarms logs during alert window
    │   └── hcloud lts query-log --keywords "alarm_evaluation_error"
    │
    ├── 2. Query APM: traces for alarm evaluation API
    │   └── hcloud apm query-trace --service "ces-alarm-evaluator"
    │
    └── 3. Correlate:
        ├── If evaluation error in logs → Check alarm rule validity
        ├── If timeout in trace → Check CES backend health
        └── If quota exceeded → Review alarm count + quotas
```

## 6. CES Self-Monitoring Metrics

| Metric Name | Namespace | Purpose | Alert Threshold |
|-------------|-----------|---------|----------------|
| `alarm_evaluation_latency` | SYS.CES | Time to evaluate alarm | > 10s |
| `api_success_rate` | SYS.CES | CES API availability | < 99% |
| `api_latency_p95` | SYS.CES | CES API latency | > 2s |
| `notification_delivery_rate` | SYS.CES | Alarm action delivery | < 95% |
| `quota_usage_percent` | SYS.CES | Alarm/notification quota | > 80% |

## 7. Compliance Checklist

- [x] Metrics → Logs linkage defined (CES self-monitoring)
- [x] Logs → Metrics linkage defined (evaluation errors, API errors)
- [x] Trace → Metric/Log linkage defined (API latency → metrics + logs)
- [x] Data source mapping documented (CES namespaces → LTS groups → APM)
- [x] Correlation query examples provided (3 CLI examples)
- [x] CES self-monitoring metrics defined (5 core metrics)
