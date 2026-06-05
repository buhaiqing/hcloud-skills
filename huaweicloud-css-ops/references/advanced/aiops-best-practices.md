# CSS AIOps Patterns

## Anomaly Detection Framework

### Pattern 1: Cluster Health Degradation

**Definition**: Cluster health transitions from green to yellow or red

**Detection Logic**:
```python
def detect_health_degradation(cluster_id, window_minutes=5):
    health_history = query_ces(
        metric='cluster_health',
        cluster_id=cluster_id,
        window=f'{window_minutes}m'
    )
    
    # Health codes: 0=red, 1=yellow, 2=green
    current = health_history[-1]
    previous = health_history[0]
    
    if current < previous:
        return {
            'type': 'health_degradation',
            'severity': 'critical' if current == 0 else 'warning',
            'from': 'green' if previous == 2 else 'yellow',
            'to': 'red' if current == 0 else 'yellow',
            'timestamp': now()
        }
```

**Root Cause Analysis**:
1. Check unassigned shards: `GET /_cluster/health`
2. Check node status: `GET /_cat/nodes`
3. Check disk usage: `GET /_cat/allocation`

**Remediation**:
- Yellow: Wait for shard allocation or add nodes
- Red: Investigate node failure, force reallocation if needed

---

### Pattern 2: Query Latency Anomaly

**Definition**: Search latency exceeds baseline + 3 standard deviations

**Detection Logic**:
```python
def detect_latency_anomaly(cluster_id, window='1h'):
    latency_p99 = query_ces(
        metric='search_latency',
        cluster_id=cluster_id,
        aggregation='p99',
        window=window
    )
    
    baseline = get_historical_baseline(cluster_id, 'search_latency_p99', days=7)
    std_dev = get_historical_stddev(cluster_id, 'search_latency_p99', days=7)
    
    current_latency = latency_p99[-1]
    threshold = baseline + 3 * std_dev
    
    if current_latency > threshold:
        return {
            'type': 'latency_spike',
            'severity': 'critical' if current_latency > 1000 else 'warning',
            'current_value': current_latency,
            'baseline': baseline,
            'threshold': threshold,
            'z_score': (current_latency - baseline) / std_dev
        }
```

**Correlation Analysis**:
- Correlation with `cpu_usage`: CPU saturation?
- Correlation with `jvm_heap_usage`: GC pressure?
- Correlation with `search_rate`: Query volume spike?

**Remediation**:
1. Check slow query log
2. Add client nodes for query load balancing
3. Optimize expensive queries
4. Scale cluster if resource-bound

---

### Pattern 3: Storage Growth Anomaly

**Definition**: Disk usage growth rate exceeds 2x historical baseline

**Detection Logic**:
```python
def detect_storage_growth_anomaly(cluster_id, window='24h'):
    disk_usage = query_ces(
        metric='disk_usage',
        cluster_id=cluster_id,
        window=window
    )
    
    # Calculate growth rate (GB/hour)
    growth_rate = (disk_usage[-1] - disk_usage[0]) / len(disk_usage) * 60
    
    baseline = get_historical_baseline(cluster_id, 'disk_growth_rate_gb_per_hour')
    
    if growth_rate > baseline * 2:
        # Check if approaching capacity
        current_usage = disk_usage[-1]
        days_to_full = (100 - current_usage) / (growth_rate * 24)
        
        return {
            'type': 'storage_growth_anomaly',
            'severity': 'critical' if days_to_full < 7 else 'warning',
            'growth_rate_gb_per_hour': growth_rate,
            'baseline': baseline,
            'current_usage_percent': current_usage,
            'projected_days_to_full': days_to_full
        }
```

**Root Cause Analysis**:
1. Check indexing rate spike
2. Check for new large indices
3. Check snapshot status (may be accumulating)
4. Check for mapping explosions

**Remediation**:
1. Implement index lifecycle management (ILM)
2. Delete old indices per retention policy
3. Extend cluster storage
4. Add cold nodes for archival data

---

### Pattern 4: JVM Memory Pressure

**Definition**: JVM heap usage exceeds 85% for more than 5 minutes

**Detection Logic**:
```python
def detect_jvm_pressure(cluster_id):
    jvm_heap = query_ces(
        metric='jvm_heap_usage',
        cluster_id=cluster_id,
        window='5m'
    )
    
    avg_heap = sum(jvm_heap) / len(jvm_heap)
    max_heap = max(jvm_heap)
    
    if avg_heap > 85:
        gc_count = query_ces(metric='gc_collection_count', window='5m')
        gc_rate = (gc_count[-1] - gc_count[0]) / 5  # per minute
        
        return {
            'type': 'jvm_memory_pressure',
            'severity': 'critical' if max_heap > 90 else 'warning',
            'average_heap_percent': avg_heap,
            'max_heap_percent': max_heap,
            'gc_rate_per_minute': gc_rate,
            'recommendation': 'scale_up' if avg_heap > 90 else 'monitor'
        }
```

**Correlation Analysis**:
- Check `field_data_cache` size
- Check `request_cache` hit rate
- Check aggregation query volume

**Remediation**:
1. Increase node heap size (scale up)
2. Clear field data cache: `POST /_cache/clear`
3. Reduce shard count per node
4. Optimize memory-intensive queries

---

## Self-Healing Workflows

### Auto-Remediation Level 1: Automatic

| Trigger | Action | Verification |
|---------|--------|--------------|
| Node disconnect | Wait 5min, auto-replace if not recovered | Node count restored |
| Shard unassigned | Trigger reallocation | Health = green |
| Yellow health + node added | Auto-balance shards | Shard distribution even |

### Auto-Remediation Level 2: Assisted

| Trigger | Action | Operator Decision |
|---------|--------|-------------------|
| Red health > 10min | Alert + suggest force reroute | Approve reroute |
| Disk > 90% | Alert + suggest cleanup or extend | Choose action |
| JVM > 90% | Alert + suggest scale up | Approve scaling |

### Auto-Remediation Level 3: Manual

| Trigger | Action |
|---------|--------|
| Data corruption suspected | Escalate to support |
| Security incident | Isolate cluster + forensics |
| Cascading failure | Manual recovery per runbook |

## Knowledge Base

### Symptom-to-Cause Mapping

```yaml
symptoms:
  cluster_health_red:
    likely_causes:
      - primary_shard_unassigned
      - node_failure
      - disk_full
    diagnostic_commands:
      - GET /_cluster/health?level=shards
      - GET /_cat/shards?v
      - GET /_cat/nodes?v

  high_search_latency:
    likely_causes:
      - hot_shards
      - cpu_saturation
      - large_aggregations
    diagnostic_commands:
      - GET /_nodes/stats
      - GET /_search/slowlog
      - GET /_cat/thread_pool/search?v

  indexing_slow:
    likely_causes:
      - bulk_queue_full
      - disk_io_wait
      - too_many_shards
    diagnostic_commands:
      - GET /_cat/thread_pool/bulk?v
      - GET /_nodes/stats/fs
      - GET /_cluster/settings
```

### Historical Patterns

| Pattern ID | Description | Frequency | Resolution Time |
|------------|-------------|-----------|-----------------|
| P001 | Yellow health after node addition | Common | 5-10 min |
| P002 | Red health during rolling restart | Common | 2-5 min |
| P003 | Latency spike during bulk indexing | Common | Duration of bulk |
| P004 | JVM pressure with field data sort | Uncommon | 10-30 min |
| P005 | Snapshot failure (OBS throttling) | Rare | Retry succeeds |

## Cross-Skill Correlation

### CSS ↔ CES Correlation

```python
def correlate_css_ces_anomaly(cluster_id):
    """Correlate CSS metrics with CES alarms"""
    css_metrics = query_css_metrics(cluster_id)
    ces_alarms = query_ces_alarms(resource_id=cluster_id)
    
    correlations = []
    
    # High latency + CES CPU alarm
    if css_metrics['search_latency'] > threshold:
        cpu_alarm = find_alarm(ces_alarms, 'cpu_usage')
        if cpu_alarm:
            correlations.append({
                'type': 'resource_saturation',
                'cause': 'CPU saturation causing high latency',
                'recommendation': 'Scale cluster or optimize queries'
            })
    
    return correlations
```

### CSS ↔ OBS Correlation

```python
def correlate_snapshot_issues(cluster_id, bucket_name):
    """Correlate snapshot failures with OBS metrics"""
    snapshot_failures = query_snapshot_failures(cluster_id)
    obs_metrics = query_obs_metrics(bucket_name)
    
    if snapshot_failures and obs_metrics['4xx_errors'] > 0:
        return {
            'type': 'obs_permission_issue',
            'cause': 'OBS returning 4xx errors',
            'recommendation': 'Check IAM permissions and OBS bucket policy'
        }
```

## Proactive Inspection

### Daily Inspection Checklist

```bash
#!/bin/bash
# daily-css-inspection.sh

echo "=== CSS Daily Health Check ==="

# 1. Cluster health
hcloud CSS ListClusters -o json | jq '.clusters[] | {name: .name, status: .status}'

# 2. Disk usage
hcloud CES ShowMetricData --namespace SYS.CSS --metric_name disk_usage --period 3600

# 3. Snapshot status
hcloud CSS ListSnapshots --cluster_id $CLUSTER_ID -o json | jq '.snapshots[] | select(.status == "FAILED")'

# 4. Recent alarms
hcloud CES ShowAlarms --namespace SYS.CSS --start_time $(date -d '1 day ago' +%s)000
```

### Weekly Trend Analysis

| Metric | Trend | Action if Degrading |
|--------|-------|---------------------|
| Search latency | Increasing | Capacity planning |
| Storage growth | Accelerating | Extend storage |
| Error rate | Increasing | Investigate |
| Node failures | Any | Root cause analysis |

## Capacity Forecasting

### Linear Projection

```python
def forecast_capacity(cluster_id, days_ahead=30):
    """Simple linear capacity forecast"""
    history = query_ces(
        metric='disk_usage',
        cluster_id=cluster_id,
        window='30d'
    )
    
    # Linear regression
    slope, intercept = linear_refit(history)
    
    # Project forward
    future_date = now() + timedelta(days=days_ahead)
    projected_usage = slope * days_ahead + intercept
    
    return {
        'current_usage_percent': history[-1],
        'projected_usage_percent': projected_usage,
        'projected_full_date': now() + timedelta(days=(100 - history[-1]) / slope),
        'recommendation': 'extend_storage' if projected_usage > 80 else 'monitor'
    }
```

## SLO/SLI Definition

| SLI | SLO | Measurement Window |
|-----|-----|-------------------|
| Search availability | 99.9% | 30 days |
| Search latency (p99) | < 200ms | 7 days |
| Indexing latency (p99) | < 5s | 7 days |
| Snapshot success rate | > 95% | 30 days |
| Cluster health (green) | > 99% | 30 days |

**Error Budget**: 0.1% of requests can fail per month
