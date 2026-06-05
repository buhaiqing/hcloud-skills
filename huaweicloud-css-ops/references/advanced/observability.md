# CSS Observability

## Metrics → Logs → Traces

### Metrics (CES)

**Cluster Metrics**:
- CPU usage
- Memory usage
- Disk usage
- JVM heap usage
- Cluster health status
- Search latency (avg, p99)
- Indexing rate
- Search rate

**Node Metrics**:
- Node CPU
- Node memory
- Node disk
- Node JVM heap
- GC collection count/time

### Logs

**CSS Service Logs**:
- Location: LTS (Log Tank Service)
- Log Group: css-logs
- Log Streams: audit, operation, error

**Elasticsearch Logs**:
- Search slow log
- Indexing slow log
- Deprecation log
- Server log

**Access Logs**:
- HTTP access logs
- Authentication logs
- Audit logs (CTS)

### Traces

Distributed tracing not natively supported by CSS. Use APM tools:
- Jaeger
- Zipkin
- SkyWalking

## Observability Integration

### CES → LTS Pipeline

```yaml
metric_export:
  source: CES
  destination: LTS
  interval: 60s
  metrics:
    - css_cluster_health
    - css_cpu_usage
    - css_disk_usage
```

### Log Analysis Queries

**Slow Query Analysis**:
```sql
-- Find slow queries in last hour
SELECT 
  query_time,
  index_name,
  query_json
FROM css_slowlog
WHERE timestamp > now() - INTERVAL 1 HOUR
  AND query_time > 1000
ORDER BY query_time DESC
LIMIT 100
```

**Error Pattern Analysis**:
```sql
-- Find most common errors
SELECT 
  error_code,
  COUNT(*) as count,
  AVG(response_time) as avg_time
FROM css_errorlog
WHERE timestamp > now() - INTERVAL 24 HOUR
GROUP BY error_code
ORDER BY count DESC
```

## Dashboards

### CSS Overview Dashboard

Panels:
1. Cluster Health (stat)
2. Node Count (stat)
3. Search Rate (graph)
4. Indexing Rate (graph)
5. CPU/Memory/Disk (graph)
6. JVM Heap (graph)
7. Slow Queries (table)
8. Top Indices by Size (table)

### CSS Performance Dashboard

Panels:
1. Search Latency p50/p99 (graph)
2. Indexing Latency (graph)
3. Query Cache Hit Rate (stat)
4. Field Data Cache (graph)
5. Segment Count (graph)
6. Merge Operations (graph)

### CSS Security Dashboard

Panels:
1. Failed Login Attempts (graph)
2. Access by IP (table)
3. Privileged Operations (table)
4. Audit Events (log panel)

## Alerting Integration

### CES Alarm Rules

```yaml
alarms:
  - name: cluster_health_red
    metric: cluster_health
    condition: value == 0
    severity: critical
    
  - name: high_search_latency
    metric: search_latency_p99
    condition: value > 500
    duration: 5m
    severity: warning
```

### LTS Alarm Rules

```yaml
alarms:
  - name: slow_query_spike
    log_filter: query_time > 1000
    threshold: 10 per minute
    severity: warning
    
  - name: authentication_failure
    log_filter: event_type == "auth_failed"
    threshold: 5 per minute
    severity: high
```

## Correlation Analysis

### Metric-to-Log Correlation

When `search_latency` spikes:
1. Query CES for exact timestamp
2. Search LTS for slow queries at that time
3. Correlate with `cpu_usage` and `jvm_heap`

### Alert-to-Metric Correlation

```python
def correlate_alert_to_metrics(alert):
    timestamp = alert.timestamp
    
    # Get related metrics
    metrics = query_ces(
        start=timestamp - 300,
        end=timestamp + 300,
        metrics=['cpu', 'memory', 'disk', 'jvm_heap']
    )
    
    # Find correlation
    for metric in metrics:
        if metric.correlation(alert) > 0.8:
            return metric
```

## Observability Best Practices

1. **Retention**: Keep metrics 90 days, logs 30 days (hot), 1 year (cold)
2. **Sampling**: 100% for errors, 1% for normal traffic
3. **Cardinality**: Limit unique tag values to prevent high cardinality
4. **Correlation**: Always link metrics, logs, and traces for the same event
