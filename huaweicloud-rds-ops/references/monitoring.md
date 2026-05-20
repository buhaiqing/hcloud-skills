# Monitoring & Alerts — Huawei Cloud RDS

> **Purpose:** CES metrics, alert rules, and AIOps patterns for RDS monitoring.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20

---

## Table of Contents

1. [CES Metric Namespaces](#1-ces-metric-namespaces)
2. [Key Metrics](#2-key-metrics)
3. [Alert Rules](#3-alert-rules)
4. [Anomaly Patterns](#4-anomaly-patterns)
5. [AIOps Multi-Metric Correlation](#5-aiops-multi-metric-correlation)
6. [Proactive Inspection Workflow](#6-proactive-inspection-workflow)

---

## 1. CES Metric Namespaces

### 1.1 RDS Namespace

| Namespace | Description | Metrics Count |
|-----------|-------------|---------------|
| `SYS.RDS` | RDS service metrics | 50+ |

### 1.2 Metric Dimension

| Dimension | Description | Example |
|-----------|-------------|---------|
| `rds_cluster_id` | Instance ID | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| `rds_engine` | Database engine | `MySQL`, `PostgreSQL`, `SQLServer` |
| `rds_project_id` | Project ID | `project-xxx` |

---

## 2. Key Metrics

### 2.1 Resource Metrics

| Metric Name | Description | Unit | Threshold |
|-------------|-------------|------|-----------|
| `rds001_cpu_usage` | CPU utilization | % | > 80%: Warning, > 95%: Critical |
| `rds002_mem_usage` | Memory utilization | % | > 85%: Warning, > 95%: Critical |
| `rds003_connections_usage` | Connection utilization | % | > 90%: Warning |
| `rds004_disk_usage` | Disk utilization | % | > 85%: Warning, > 95%: Critical |
| `rds005_virtual_mem_usage` | Virtual memory utilization | % | > 90%: Warning |

### 2.2 Performance Metrics

| Metric Name | Description | Unit | Threshold |
|-------------|-------------|------|-----------|
| `rds043_slow_queries` | Slow query count (per second) | count/s | > 10: Warning |
| `rds044_qps` | Queries per second | count/s | Varies by instance |
| `rds045_iops` | I/O operations per second | count/s | > threshold: Warning |
| `rds046_disk_throughput` | Disk throughput | MB/s | > threshold: Warning |

### 2.3 Replication Metrics (HA)

| Metric Name | Description | Unit | Threshold |
|-------------|-------------|------|-----------|
| `rds006_replication_lag` | Replication lag | ms | > 1000ms: Warning |
| `rds007_replication_status` | Replication status | — | 1: Normal, 0: Abnormal |

### 2.4 Backup Metrics

| Metric Name | Description | Unit | Threshold |
|-------------|-------------|------|-----------|
| `rds050_backup_failures` | Backup failure count | count | > 0: Critical |
| `rds051_backup_duration` | Backup duration | s | > threshold: Warning |
| `rds052_backup_size` | Backup data size | MB | — |

### 2.5 Transaction Metrics

| Metric Name | Description | Unit | Threshold |
|-------------|-------------|------|-----------|
| `rds010_transaction_count` | Transaction count | count/s | Varies |
| `rds011_commit_count` | Commit count | count/s | — |
| `rds012_rollback_count` | Rollback count | count/s | > threshold: Warning |

---

## 3. Alert Rules

### 3.1 Critical Alerts

| Alert Name | Metric | Condition | Severity | Action |
|------------|--------|-----------|----------|--------|
| CPU Critical | rds001_cpu_usage | > 95% for 5min | Critical | Scale up or optimize |
| Memory Critical | rds002_mem_usage | > 95% for 5min | Critical | Scale up or optimize |
| Disk Full | rds004_disk_usage | > 95% for 2min | Critical | Expand storage immediately |
| Backup Failed | rds050_backup_failures | > 0 in 1h | Critical | Investigate backup issues |
| HA Down | rds007_replication_status | = 0 | Critical | Check standby node |

### 3.2 Warning Alerts

| Alert Name | Metric | Condition | Severity | Action |
|------------|--------|-----------|----------|--------|
| CPU High | rds001_cpu_usage | > 80% for 10min | Warning | Monitor closely |
| Memory High | rds002_mem_usage | > 85% for 10min | Warning | Monitor closely |
| Connections High | rds003_connections_usage | > 80% for 5min | Warning | Check connection pool |
| Disk High | rds004_disk_usage | > 85% for 5min | Warning | Plan storage expansion |
| Slow Queries | rds043_slow_queries | > 10/s for 5min | Warning | Optimize queries |
| Replication Lag | rds006_replication_lag | > 1000ms for 5min | Warning | Check network/load |

### 3.3 Alert Action Matrix

| Alert Type | Primary Action | Secondary Action | Escalation |
|------------|---------------|-----------------|------------|
| CPU Critical | Auto-scale up | Notify ops team | If sustained > 30min |
| Memory Critical | Auto-scale up | Notify ops team | If sustained > 30min |
| Disk Full | Auto-expand storage | Alert DBA | Immediate |
| Backup Failed | Retry backup | Check OBS quota | If > 3 failures |
| HA Down | Trigger failover | Alert DBA | Immediate |

---

## 4. Anomaly Patterns

### 4.1 Pattern Definition Template

| Pattern ID | Pattern Name | Metrics Involved | Detection Logic | Severity |
|------------|-------------|-----------------|-----------------|----------|
| RDS-P001 | CPU-Memory Dual High | rds001_cpu_usage, rds002_mem_usage | cpu > 80% AND mem > 85% | Critical |
| RDS-P002 | Connection Saturation | rds003_connections_usage, rds001_cpu_usage | connections > 90% | Critical |
| RDS-P003 | Storage Pressure | rds004_disk_usage, rds045_iops | disk > 85% OR iops > threshold | Warning |
| RDS-P004 | Slow Query Spike | rds043_slow_queries | delta(10min) > 50% | Warning |
| RDS-P005 | Memory Leak Trend | rds002_mem_usage (30min trend) | slope > 0.5%/min | Critical |

### 4.2 Pattern Detection Examples

```markdown
## Pattern: RDS-P001 — CPU-Memory Dual High

### Detection Logic
1. Query: CES GetMetricData for rds001_cpu_usage
2. Query: CES GetMetricData for rds002_mem_usage
3. Condition: cpu > 80% AND mem > 85% for 5 consecutive minutes

### Interpretation
- Resource exhaustion immininent
- Possible OOM risk
- Application response degradation

### Root Causes
1. Large batch job running
2. Inefficient queries consuming resources
3. Sudden traffic spike
4. Connection pool misconfiguration

### Fix Actions
1. **Immediate**: Identify and kill long-running queries
2. **Short-term**: Scale up instance
3. **Long-term**: Optimize queries, add indexes

### Prevention
- Set up proactive monitoring for >70% thresholds
- Implement query timeout limits
- Use connection pooling properly
```

### 4.3 Trend Anomaly Detection

```go
// Trend detection algorithm
func detectTrendAnomaly(points []DataPoint, threshold float64) bool {
    if len(points) < 6 {
        return false
    }
    
    // Calculate slope
    slope := calculateSlope(points)
    
    // Positive slope indicates potential memory leak
    if slope > threshold {
        return true
    }
    return false
}

func calculateSlope(points []DataPoint) float64 {
    n := float64(len(points))
    var sumX, sumY, sumXY, sumX2 float64
    
    for i, p := range points {
        x := float64(i)
        y := p.Average
        sumX += x
        sumY += y
        sumXY += x * y
        sumX2 += x * x
    }
    
    return (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
}
```

---

## 5. AIOps Multi-Metric Correlation

### 5.1 Cross-Metric Analysis Matrix

| Alert Scenario | Primary Metric | Correlated Metrics | Diagnosis Focus |
|----------------|-----------------|---------------------|------------------|
| CPU spike | rds001_cpu_usage | rds043_slow_queries, rds044_qps | Check for slow queries or high QPS |
| Memory spike | rds002_mem_usage | rds003_connections_usage | Check connection pool, possible leak |
| Connections full | rds003_connections_usage | rds001_cpu_usage, rds010_transaction_count | Check for connection leak, long transactions |
| Disk pressure | rds004_disk_usage | rds045_iops, rds052_backup_size | Check IO patterns, cleanup old data |
| Slow queries | rds043_slow_queries | rds001_cpu_usage, rds006_replication_lag | Check query plans, replication lag |

### 5.2 AIOps Decision Tree

```
[Alert: CPU High]
        │
        ▼
Step 1: Verify alert
  └─► Query CES: rds001_cpu_usage > 80%?
        │
        ▼
Step 2: Check slow queries
  └─► Query: rds043_slow_queries
  └─► If spike → Identify top slow queries
        │
        ▼
Step 3: Check connection count
  └─► Query: rds003_connections_usage
  └─► If high → Connection pool issue
        │
        ▼
Step 4: Generate recommendation
  └─► Based on root cause → specific action
```

### 5.3 Alert-to-Diagnosis Routing Matrix

| CES Alert | Primary Skill | Secondary Skill | Delegation Trigger |
|-----------|---------------|-----------------|-------------------|
| CPU High | huaweicloud-rds-ops | huaweicloud-ces-ops | CPU > 95% for > 10min |
| Memory High | huaweicloud-rds-ops | — | Memory > 90% for > 10min |
| Connections High | huaweicloud-rds-ops | huaweicloud-vpc-ops | Connections > 90% |
| Disk Full | huaweicloud-rds-ops | — | Disk > 95% |
| Backup Failed | huaweicloud-rds-ops | huaweicloud-obs-ops | Any backup failure |
| HA Down | huaweicloud-rds-ops | — | Replication status = 0 |

---

## 6. Proactive Inspection Workflow

### 6.1 Five-Step Inspection Loop

```
[Resource Discovery] → [Metric Collection] → [Anomaly Detection] → [Cross-Skill Diagnosis] → [Report Generation]
```

### 6.2 Step Details

#### Step 1: Resource Discovery
```bash
# List all RDS instances in region
hcloud rds list --region {{user.region}} --limit 100

# Output: Instance inventory with ID, name, status, engine
```

#### Step 2: Metric Collection
```bash
# Batch collect key metrics for all instances
# Use CES batch API: GetMetricData for multiple instances
# Period: 300s (5 min), metrics: cpu, mem, conn, disk, iops
```

#### Step 3: Anomaly Detection
```bash
# Check against thresholds
# Check for trend anomalies (slope > threshold)
# Check for sudden changes (delta > threshold)
# Mark instances with anomalies
```

#### Step 4: Cross-Skill Diagnosis
```bash
# For high-risk instances:
# - Delegate to huaweicloud-rds-ops for detailed diagnosis
# - Query LTS for error logs
# - Query AOM for application traces (if applicable)
```

#### Step 5: Report Generation
```markdown
## RDS Inspection Report

### Summary
- Total instances: N
- Healthy: N
- Warning: N
- Critical: N

### Critical Issues
| Instance | Issue | Metric Value | Recommended Action |
|----------|-------|--------------|-------------------|
| rds-prod-01 | CPU Critical | 98% | Scale up immediately |

### Warning Issues
| Instance | Issue | Metric Value | Recommended Action |
|----------|-------|--------------|-------------------|
| rds-prod-02 | Slow Queries | 15/s | Optimize queries |

### Cost Optimization
| Instance | Utilization | Recommendation |
|----------|-------------|----------------|
| rds-dev-01 | CPU 5%, Mem 10% | Downgrade or delete |
```

---

## 7. Alarm Storm Handling

### 7.1 Storm Detection Criteria

| Criterion | Threshold | Action |
|-----------|-----------|--------|
| Alarm frequency | > 10 alarms / 5 min | Enter storm mode |
| Same resource | > 3 alarms on one instance | Aggregate to single event |
| Same namespace | > 50% from RDS namespace | Focus diagnosis on RDS |
| Cascade pattern | Alarm A triggers B within 2min | Mark B as "caused by A" |

### 7.2 Storm Processing Flow

1. **Detect:** Monitor CES alarm list with State=ALARM
2. **Aggregate:** Group by instance_id, time window (5min)
3. **Suppress:** Retain only primary alarm notification
4. **Root Identification:** Find earliest alarm as likely root cause
5. **Focus Diagnosis:** Delegate root resource diagnosis

---

## 8. Observability Linkage

### 8.1 Metrics → Logs

| CES Anomaly | LTS Query Target | Purpose |
|-------------|-----------------|---------|
| CPU spike | Application error logs | Confirm error burst causing CPU surge |
| Memory leak | Application memory logs | Confirm allocation pattern |
| Slow queries | RDS slow query log | Identify problematic queries |
| Connection max | Database access logs | Confirm connection source |

### 8.2 Metrics → Traces

| CES Anomaly | AOM Trace Target | Purpose |
|-------------|-----------------|---------|
| High latency | Application Trace | Locate slow methods |
| Error rate increase | Error Trace | Locate error root cause |
| Timeout | HTTP/RPC Trace | Locate timeout source |

---

*This document defines monitoring and alerting patterns for RDS. Update with new metrics and patterns as discovered.*