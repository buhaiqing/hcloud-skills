# Monitoring & Alerts — Huawei Cloud DCS (Redis)

## Core Metrics Table

CES Namespace: **SYS.DCS**

| Metric Name | CES Metric ID | Unit | Description | Recommended Alert Threshold |
|-------------|--------------|------|-------------|---------------------------|
| CPU Utilization | cpu_usage | % | Instance CPU usage | Warning: >70%, Critical: >90% |
| Memory Utilization | memory_usage | % | Memory usage percentage | Warning: >80%, Critical: >95% |
| Connected Clients | connected_clients | count | Current Redis client connections | Warning: >80% of max, Critical: >95% |
| Commands Per Second | commands | ops/s | Total commands executed per second | Warning: >80% of capacity, Critical: near max |
| Network In | bytes_in | bytes/s | Inbound network traffic | Monitor for baselining |
| Network Out | bytes_out | bytes/s | Outbound network traffic | Warning: >80% of bandwidth limit |
| Cache Hit Rate | hit_rate | % | Key lookup hit ratio | Warning: <80%, Critical: <60% |
| Evicted Keys | evicted_keys | count | Keys evicted due to maxmemory policy | Warning: >0 for 5min, Critical: >100/min |
| Expired Keys | expired_keys | count | Keys expired due to TTL | Monitor as operational metric |
| Latency | latency | ms | Average command latency | Warning: >10ms, Critical: >100ms |
| Blocked Clients | blocked_clients | count | Clients blocked on blocking commands | Warning: >10, Critical: >100 |
| Keys Count | keys_count | count | Total number of keys in instance | Monitor for growth trends |
| Memory Used | used_memory | MB | Actual memory used in MB | Warning: >80% of capacity |

## Alert Recommendations

| Metric | Warning Condition | Critical Condition | Window | Aggregation | Notification |
|--------|------------------|-------------------|--------|-------------|--------------|
| cpu_usage | > 70% | > 90% | 5 min | Average | SMS + Email |
| memory_usage | > 80% | > 95% | 5 min | Average | SMS + Email + DingTalk |
| connected_clients | > 80% of max | > 95% of max | 5 min | Maximum | SMS + Email |
| hit_rate | < 80% | < 60% | 10 min | Average | Email |
| evicted_keys | > 0 for 5 min | > 100/min | 5 min | Sum | SMS + Email |
| latency | > 10 ms | > 100 ms | 5 min | P95 | SMS + Email |
| bytes_out | > 80% bandwidth | > 95% bandwidth | 5 min | Average | SMS + Email |

## Anomaly Patterns (≥ 4)

### Pattern 1: Memory Pressure → OOM Risk

**Detection**: `memory_usage > 90%` AND (`evicted_keys > 0` OR `hit_rate < 70%`) AND trending upward
**Action**: Alert user to resize or clean up keys; if hit_rate < 50%, investigate cache efficiency
**CES Query**: `SYS.DCS,memory_usage,memory_usage > 90 AND evicted_keys > 0`

### Pattern 2: Connection Exhaustion

**Detection**: `connected_clients > 80% of max` AND `latency spike > 2x baseline`
**Action**: Check for connection leaks in application; consider increasing instance spec or fixing pool config
**CES Query**: `SYS.DCS,connected_clients,connected_clients > {max_clients * 0.8} AND latency > {baseline * 2}`

### Pattern 3: Cache Inefficiency (Miss Storm)

**Detection**: `hit_rate < 70%` AND `expired_keys spike` (3x normal rate) AND `commands count surge`
**Action**: Investigate TTL configuration — possible cache avalanche; check for cache penetration attacks
**CES Query**: `SYS.DCS,hit_rate < 70 AND expired_keys > {normal_rate * 3}`

### Pattern 4: Network Saturation

**Detection**: `bytes_out > 80% bandwidth` sustained for 5 min AND `latency increasing`
**Action**: Identify heavy consumers of data; consider CDN caching for hot data or optimize query patterns
**CES Query**: `SYS.DCS,bytes_out > {bandwidth * 0.8} AND latency > 50`

### Pattern 5: Resource Pressure Cascade

**Detection**: `cpu_usage > 80%` AND `memory_usage > 80%` AND `connected_clients > 70%` simultaneously
**Action**: Instance is under comprehensive load — resize to larger spec or scale out to cluster mode
**CES Query**: Multi-metric correlation — all three conditions within same 5 min window

## Dashboards

### Daily Monitoring Dashboard

Create CES dashboard with these panels:

| Panel | Metrics | Time Range | Chart Type |
|-------|---------|------------|------------|
| Resource Health | cpu_usage, memory_usage | Last 24h | Line |
| Throughput | commands/sec, connected_clients | Last 24h | Line |
| Efficiency | hit_rate, evicted_keys, expired_keys | Last 24h | Line + Histogram |
| Latency | latency (avg, P95, max) | Last 24h | Line |
| Network | bytes_in, bytes_out | Last 24h | Area |

### Capacity Planning Dashboard

| Panel | Metrics | Time Range | Purpose |
|-------|---------|------------|---------|
| Memory Growth Trend | memory_usage, used_memory | Last 30d | Predict when resize needed |
| Connection Growth | connected_clients over time | Last 30d | Plan connection limit increases |
| Command Throughput | commands/sec daily max | Last 30d | Identify peak load patterns |

## Cost & Performance Metrics

| Metric | Cost Implication | Detection | Action |
|--------|----------------|-----------|--------|
| memory_usage < 20% for 7 days | Over-provisioned | CES metric + commands < threshold | Right-size to smaller capacity |
| cpu_usage < 5% for 7 days | Under-utilized | CES metric, consider pay-per-use to subscription | Downgrade spec |
| commands ≈ 0 for 7 days | Idle/unused instance | CES command count = 0 | Decommission or repurpose |
| bytes_out >> bytes_in | High egress traffic cost | Network metrics analysis | Optimize query patterns, add CDN |

## Self-Monitoring Workflow

```bash
#!/bin/bash
# Periodic DCS health check
INSTANCE_ID="dcs-xxx"
REGION="{{env.HW_REGION_ID}}"

# Step 1: Check instance status
STATUS=$(hcloud dcs show-instance --instance-id "$INSTANCE_ID" | jq -r '.status')
if [ "$STATUS" != "RUNNING" ]; then
  echo "❌ Instance $INSTANCE_ID is not RUNNING: $STATUS"
  exit 1
fi

# Step 2: Check memory usage (via CES query)
MEM_USAGE=$(hcloud ces describe-metric-data \
  --namespace "SYS.DCS" \
  --metric-name "memory_usage" \
  --dimensions "name=instance_id,value=$INSTANCE_ID" \
  --period 300 | jq -r '.datapoints[-1].average // 0')

if [ "$(echo "$MEM_USAGE > 90" | bc)" -eq 1 ]; then
  echo "⚠️ Memory usage critical: ${MEM_USAGE}%"
fi

# Step 3: Quick Redis check
IP=$(hcloud dcs show-instance --instance-id "$INSTANCE_ID" | jq -r '.ip')
PORT=$(hcloud dcs show-instance --instance-id "$INSTANCE_ID" | jq -r '.port')
redis-cli -h "$IP" -p "$PORT" -a "{{user.password}}" PING > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo "✅ Redis PING successful"
else
  echo "❌ Redis PING failed"
fi
```

## Baseline Establishment

Run the following after deploying a new DCS instance to establish performance baselines:

```bash
# Capture baseline over 24 hours
hcloud ces describe-metric-data \
  --namespace "SYS.DCS" \
  --metric-name "cpu_usage" \
  --dimensions "name=instance_id,value={{user.instance_id}}" \
  --period 300 \
  --from "$(date -d '24 hours ago' +%s)" \
  --to "$(date +%s)"

# Store baseline values for: cpu_usage (p50, p95), memory_usage, latency, commands/sec
# Alert on deviation > 2σ from baseline
```
