# Monitoring — Huawei Cloud FunctionGraph

## CES Metrics (Cloud Eye Service)

Namespace: `SYS.FUNCTIONGRAPH`

### Key Metrics

| Metric | Name | Unit | Recommended Threshold |
|--------|------|------|---------------------|
| `count` | Invocation count | count | Baseline-dependent |
| `fail_count` | Failed invocations | count | Warning: > 0 in 5min |
| `duration` | Average duration | ms | Warning: > 80% of timeout |
| `max_duration` | Max duration | ms | Warning: > timeout |
| `reject_count` | Throttled invocations | count | Warning: > 0 |
| `concurrent_executions` | Concurrent executions | count | Warning: > 60% of limit |
| `reserved_instance_num` | Reserved instances | count | — |

## Alert Patterns

### Resource Pressure Alerts

| Alert | Metric | Condition | Severity |
|-------|--------|-----------|----------|
| High error rate | `fail_count` | > 1% of total invocations | Critical |
| Function timeout | `max_duration` | > 90% of timeout threshold | Critical |
| Throttling | `reject_count` | > 0 in 5min | Warning |
| Concurrent limit near | `concurrent_executions` | > 80% of limit | Warning |

### Anomaly Patterns

| Pattern | Metrics | Detection Logic | Severity |
|---------|---------|----------------|----------|
| error_rate_spike | `fail_count`, `count` | fail/count > 5% in 5min | Critical |
| duration_degradation | `duration`, `max_duration` | avg duration > 2× baseline | Warning |
| sudden_traffic_drop | `count` | invocation count drop > 50% | Warning |
| cold_start_impact | `duration`, `max_duration` | max >> avg (cold start > 3× avg) | Info |
| memory_pressure | `duration`, `fail_count` | frequent OOM kills | Critical |

## Dashboards

- Recommended CES dashboard: FunctionGraph → grouped by function name
- Metrics: `count`, `fail_count`, `duration(avg/max/P99)`, `reject_count`
- Time range: last 1h for real-time, last 7d for trends

## SLA & Error Budget

| Metric | SLO Target | Error Budget (monthly) |
|--------|-----------|----------------------|
| Function success rate | ≥ 99.9% | 43.2 min of failures |
| P99 latency (sync) | ≤ 2× configured timeout | — |
| Throttle rate | ≤ 0.1% of invocations | — |

## Logs (LTS)

Function execution logs are sent to LTS automatically. Log group format: `/functiongraph/{function_urn}`

Common log queries:
```lql
# Find all errors in last hour
resource.name="/functiongraph/{function_urn}" | where level = "ERROR"

# Find timeout executions
resource.name="/functiongraph/{function_urn}" | where message contains "timeout"

# Trace specific invocation
resource.name="/functiongraph/{function_urn}" | where request_id = "{request_id}"
```

## Cost Monitoring

| Metric | Purpose | Action |
|--------|---------|--------|
| `count` × avg duration | Total compute time (GB-seconds) | Right-size memory |
| Reserved instance count | Fixed cost | Review necessity |
| Monthly cost (BSS) | Budget tracking | Set budget alerts |
