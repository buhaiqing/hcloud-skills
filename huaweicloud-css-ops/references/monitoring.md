# CSS Monitoring & Alerts

## CES Metrics

### Cluster-Level Metrics

| Metric Name | Description | Unit | Dimensions |
|-------------|-------------|------|------------|
| `cpu_usage` | CPU utilization | % | cluster_id |
| `mem_usage` | Memory utilization | % | cluster_id |
| `disk_usage` | Disk utilization | % | cluster_id |
| `jvm_heap_usage` | JVM heap usage | % | cluster_id |
| `cluster_health` | Cluster health status | - | cluster_id |
| `search_latency` | Search query latency | ms | cluster_id |
| `indexing_rate` | Documents indexed per second | count/s | cluster_id |
| `search_rate` | Search requests per second | count/s | cluster_id |
| `node_count` | Number of nodes | count | cluster_id |
| `shard_count` | Number of shards | count | cluster_id |

### Node-Level Metrics

| Metric Name | Description | Unit | Dimensions |
|-------------|-------------|------|------------|
| `node_cpu_usage` | Node CPU usage | % | cluster_id, node_id |
| `node_mem_usage` | Node memory usage | % | cluster_id, node_id |
| `node_disk_usage` | Node disk usage | % | cluster_id, node_id |
| `node_jvm_heap` | Node JVM heap | % | cluster_id, node_id |

## Alert Rules

### Critical Alerts

```yaml
# Cluster Health Red
alert: CSSClusterHealthRed
expr: cluster_health{cluster_id="*"} == 0
for: 1m
labels:
  severity: critical
annotations:
  summary: "CSS cluster {{ $labels.cluster_id }} health is RED"
  description: "Cluster has unassigned shards or nodes are down"

# Disk Full
alert: CSSDiskFull
expr: disk_usage{cluster_id="*"} > 90
for: 5m
labels:
  severity: critical
annotations:
  summary: "CSS cluster {{ $labels.cluster_id }} disk > 90%"
  description: "Cluster will reject writes at 95%"

# JVM Heap High
alert: CSSJvmHeapHigh
expr: jvm_heap_usage{cluster_id="*"} > 85
for: 5m
labels:
  severity: critical
annotations:
  summary: "CSS cluster {{ $labels.cluster_id }} JVM heap > 85%"
  description: "Cluster may experience OOM errors"
```

### Warning Alerts

```yaml
# Cluster Health Yellow
alert: CSSClusterHealthYellow
expr: cluster_health{cluster_id="*"} == 1
for: 5m
labels:
  severity: warning
annotations:
  summary: "CSS cluster {{ $labels.cluster_id }} health is YELLOW"
  description: "Cluster has replica shards unassigned"

# High Search Latency
alert: CSSSearchLatencyHigh
expr: search_latency{cluster_id="*"} > 500
for: 10m
labels:
  severity: warning
annotations:
  summary: "CSS cluster {{ $labels.cluster_id }} search latency > 500ms"
  description: "Query performance degraded"

# CPU High
alert: CSSCpuHigh
expr: cpu_usage{cluster_id="*"} > 80
for: 15m
labels:
  severity: warning
annotations:
  summary: "CSS cluster {{ $labels.cluster_id }} CPU > 80%"
  description: "Consider scaling cluster"
```

### Info Alerts

```yaml
# Snapshot Failed
alert: CSSSnapshotFailed
expr: increase(snapshot_failures{cluster_id="*"}[1h]) > 0
labels:
  severity: info
annotations:
  summary: "CSS cluster {{ $labels.cluster_id }} snapshot failed"
  description: "Check OBS permissions and cluster health"

# Node Disconnected
alert: CSSNodeDisconnected
expr: node_count{cluster_id="*"} < expected_node_count
labels:
  severity: info
annotations:
  summary: "CSS cluster {{ $labels.cluster_id }} node disconnected"
  description: "Auto-recovery in progress"
```

## AIOps Anomaly Patterns

### Pattern 1: Storage Growth Anomaly

**Detection**: Unexpected disk usage growth rate

```python
def detect_storage_anomaly(cluster_id, window=3600):
    # Get disk usage history
    metrics = query_ces(
        metric="disk_usage",
        cluster_id=cluster_id,
        start=now() - window,
        end=now()
    )
    
    # Calculate growth rate
    growth_rate = (metrics[-1] - metrics[0]) / len(metrics)
    baseline = get_historical_baseline(cluster_id, "disk_usage")
    
    # Anomaly if growth rate > 2x baseline
    if growth_rate > baseline * 2:
        return {
            "type": "storage_growth_anomaly",
            "severity": "warning",
            "current_rate": growth_rate,
            "baseline": baseline,
            "recommendation": "Check for large index creation or bulk imports"
        }
```

**Root Causes**:
- Large bulk indexing operation
- New index creation without lifecycle policy
- Snapshot retention too long

**Remediation**:
1. Review recent indexing operations
2. Implement index lifecycle management (ILM)
3. Adjust snapshot retention policy

### Pattern 2: Query Latency Spike

**Detection**: P99 search latency > baseline + 3σ

```python
def detect_latency_spike(cluster_id, window=300):
    latency_p99 = query_ces(
        metric="search_latency_p99",
        cluster_id=cluster_id,
        aggregation="p99"
    )
    
    baseline = get_baseline(cluster_id, "search_latency_p99")
    std_dev = get_std_dev(cluster_id, "search_latency_p99")
    
    if latency_p99 > baseline + 3 * std_dev:
        return {
            "type": "latency_spike",
            "severity": "critical" if latency_p99 > 1000 else "warning",
            "current": latency_p99,
            "baseline": baseline,
            "recommendation": "Check for expensive queries or hot shards"
        }
```

**Root Causes**:
- Expensive aggregation queries
- Hot shards (uneven data distribution)
- JVM GC pressure
- Network latency

**Remediation**:
1. Review slow query log
2. Reindex with more shards if needed
3. Add client nodes for query coordination
4. Optimize query structure

### Pattern 3: JVM Heap Pressure

**Detection**: JVM heap usage > 85% for > 5 minutes

```python
def detect_jvm_pressure(cluster_id):
    jvm_heap = query_ces(
        metric="jvm_heap_usage",
        cluster_id=cluster_id
    )
    
    if jvm_heap > 85:
        gc_rate = query_ces(metric="gc_collection_count")
        return {
            "type": "jvm_heap_pressure",
            "severity": "critical" if jvm_heap > 90 else "warning",
            "heap_usage": jvm_heap,
            "gc_rate": gc_rate,
            "recommendation": "Scale cluster or adjust heap settings"
        }
```

**Root Causes**:
- Large aggregation queries
- Too many shards per node
- Memory-intensive sorting
- Field data cache explosion

**Remediation**:
1. Increase node memory (scale up)
2. Reduce shard count per node
3. Enable doc values for fields used in sorting/aggregations
4. Clear field data cache if needed

### Pattern 4: Shard Imbalance

**Detection**: Shard distribution standard deviation > threshold

```python
def detect_shard_imbalance(cluster_id):
    nodes = get_node_stats(cluster_id)
    shard_counts = [n['shards'] for n in nodes]
    
    avg_shards = sum(shard_counts) / len(shard_counts)
    variance = sum((s - avg_shards) ** 2 for s in shard_counts) / len(shard_counts)
    std_dev = variance ** 0.5
    
    cv = std_dev / avg_shards  # Coefficient of variation
    
    if cv > 0.3:  # 30% variation
        return {
            "type": "shard_imbalance",
            "severity": "warning",
            "cv": cv,
            "recommendation": "Enable shard rebalancing or reindex"
        }
```

**Root Causes**:
- Hot spotting (time-based indices)
- Disabled shard allocation
- Node addition/removal

**Remediation**:
1. Enable/adjust shard balancing
2. Use index routing to distribute data
3. Force reroute if needed

## Dashboards

### CSS Overview Dashboard

```json
{
  "dashboard": {
    "title": "CSS Cluster Overview",
    "panels": [
      {
        "title": "Cluster Health",
        "type": "stat",
        "targets": [{
          "metric": "cluster_health",
          "dimensions": {"cluster_id": "$cluster_id"}
        }]
      },
      {
        "title": "Search Rate",
        "type": "graph",
        "targets": [{
          "metric": "search_rate",
          "dimensions": {"cluster_id": "$cluster_id"}
        }]
      },
      {
        "title": "Indexing Rate",
        "type": "graph",
        "targets": [{
          "metric": "indexing_rate",
          "dimensions": {"cluster_id": "$cluster_id"}
        }]
      },
      {
        "title": "Resource Usage",
        "type": "graph",
        "targets": [
          {"metric": "cpu_usage", "dimensions": {"cluster_id": "$cluster_id"}},
          {"metric": "mem_usage", "dimensions": {"cluster_id": "$cluster_id"}},
          {"metric": "disk_usage", "dimensions": {"cluster_id": "$cluster_id"}}
        ]
      }
    ]
  }
}
```

## Multi-Metric Correlation

### Health Degradation Analysis

When cluster health degrades, check correlation between:

1. `cluster_health` + `unassigned_shards`
2. `disk_usage` + `jvm_heap_usage`
3. `search_latency` + `cpu_usage`
4. `node_count` (unexpected drops)

```python
def correlate_health_degradation(cluster_id, timestamp):
    metrics = query_multiple(
        cluster_id=cluster_id,
        metrics=["cluster_health", "unassigned_shards", "disk_usage", "node_count"],
        start=timestamp - 3600,
        end=timestamp
    )
    
    # Find correlation
    if metrics['cluster_health'] == 0:  # Red
        if metrics['unassigned_shards'] > 0:
            return "Unassigned shards causing RED health"
        if metrics['node_count'] < expected:
            return "Node failure causing RED health"
    
    return "Unknown cause"
```
