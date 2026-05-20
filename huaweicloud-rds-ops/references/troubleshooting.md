# Troubleshooting Guide — Huawei Cloud RDS

> **Purpose:** Error codes, diagnostics, and recovery strategies for RDS operations.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20

---

## Table of Contents

1. [Error Code Taxonomy](#1-error-code-taxonomy)
2. [Connection Issues](#2-connection-issues)
3. [Performance Issues](#3-performance-issues)
4. [Storage Issues](#4-storage-issues)
5. [Backup & Restore Issues](#5-backup--restore-issues)
6. [Multi-Round Diagnosis Flow](#6-multi-round-diagnosis-flow)

---

## 1. Error Code Taxonomy

### 1.1 RDS Error Codes (≥10 Required)

| Error Code | HTTP Status | Description | Severity | Recovery Action |
|------------|-------------|-------------|----------|-----------------|
| DBS.0001 | 400 | Quota exceeded for instances | Critical | HALT; request quota increase via console |
| DBS.0002 | 400 | Invalid parameter value | High | Fix parameter based on error message; retry |
| DBS.0003 | 400 | Insufficient account balance | Critical | HALT; recharge account |
| DBS.0004 | 409 | Resource already exists | Medium | Use different name or reuse existing |
| DBS.0005 | 404 | Resource not found | Medium | Verify instance ID; check if deleted |
| DBS.0006 | 403 | IAM permission denied | Critical | HALT; check IAM policy |
| DBS.0007 | 409 | Resource in use (can't delete) | High | Wait for operation; check dependent resources |
| DBS.0008 | 500 | Internal server error | Medium | Retry with exponential backoff (2s, 4s, 8s) |
| DBS.0009 | 503 | Service unavailable | Medium | Retry after 60s; escalate if persistent |
| DBS.0010 | 429 | Rate limit exceeded | High | Back off 60s; retry |
| DBS.0011 | 400 | VPC not found | Critical | HALT; create VPC first |
| DBS.0012 | 400 | Subnet not found | Critical | HALT; create subnet in VPC |
| DBS.0013 | 400 | Security group not found | Critical | HALT; create security group |
| DBS.0014 | 400 | Flavor not found | High | List available flavors; choose valid one |
| DBS.0015 | 400 | AZ not available | High | Choose different AZ |
| DBS.0016 | 500 | Backup creation failed | Medium | Check storage quota; retry |
| DBS.0017 | 500 | Restore operation failed | High | Verify backup integrity; retry |
| DBS.0018 | 400 | Parameter modification failed | Medium | Check parameter constraints; retry |

### 1.2 CES Monitoring Error Codes

| Error Code | Description | Detection | Action |
|------------|-------------|-----------|--------|
| ALARM_CPU_HIGH | CPU usage > 80% | CES metric rds001_cpu_usage | Check slow queries; scale up |
| ALARM_MEM_HIGH | Memory usage > 85% | CES metric rds002_mem_usage | Check connection pool; scale up |
| ALARM_CONN_HIGH | Connections > 90% | CES metric rds003_connections_usage | Reduce connections; scale up |
| ALARM_DISK_HIGH | Disk usage > 85% | CES metric rds004_disk_usage | Scale storage; clean logs |
| ALARM_BACKUP_FAIL | Backup failed | CES metric rds050_backup_failures | Check OBS quota; retry |

### 1.3 Network Error Patterns

| Error Pattern | Symptoms | Diagnosis | Fix |
|---------------|----------|-----------|-----|
| Connection refused | Port 3306 blocked | Check SG rules | Open SG for app servers |
| Timeout | Cannot reach endpoint | Check VPC routing | Verify subnet configuration |
| DNS resolution failed | Host not found | Check private DNS | Use private IP instead |
| SSL handshake failed | TLS error | Check SSL settings | Enable/disable SSL based on config |

---

## 2. Connection Issues

### 2.1 "Connection Timeout" Diagnosis Flow

```
[Connection Timeout Error]
        │
        ▼
Step 1: Verify instance status
  ├─► DescribeInstance → status = ACTIVE?
  │   └─► No → Instance not ready; wait or investigate
  │
        ▼
Step 2: Check security group rules
  ├─► ListSecurityGroup → inbound rule for port 3306?
  │   └─► No → Add rule: TCP / 3306 / source: app subnet
  │
        ▼
Step 3: Verify network connectivity
  ├─► Ping private IP → reachable?
  │   └─► No → Check VPC/subnet routing
  │
        ▼
Step 4: Check connection string
  ├─► Format: <host>:<port>/<database>
  │   └─► Verify port from DescribeInstance response
  │
        ▼
Step 5: Check max connections
  ├─► Query: SHOW PROCESSLIST
  │   └─► Full → Scale up instance or optimize queries
  │
[Resolved]
```

### 2.2 "Access Denied" Diagnosis Flow

```
[Access Denied Error]
        │
        ▼
Step 1: Verify username
  ├─► ListUsers → user exists?
  │   └─► No → Create user with correct permissions
  │
        ▼
Step 2: Check password
  ├─► Password correct? (verify with user)
  │   └─► Wrong → Reset password via CLI
  │
        ▼
Step 3: Verify database access
  ├─► Query: SHOW GRANTS FOR 'user'@'%'
  │   └─► No access → Grant database privileges
  │
        ▼
Step 4: Check SSL requirements
  ├─► Instance requires SSL?
  │   └─► Yes → Connect with SSL enabled
  │
[Resolved]
```

---

## 3. Performance Issues

### 3.1 "Slow Query" Diagnosis Flow

```
[Slow Query Report]
        │
        ▼
Step 1: Identify slow queries
  ├─► Query slow log: hcloud rds download-slowlog
  │   └─► Parse query execution time > 1s
  │
        ▼
Step 2: Check query patterns
  ├─► Full table scans? (EXPLAIN)
  │   └─► Yes → Add indexes
  ├─► Missing WHERE clause?
  │   └─► Yes → Optimize query
  ├─► Large result sets?
  │   └─► Yes → Add pagination
  │
        ▼
Step 3: Check resource utilization
  ├─► CPU usage: CES rds001_cpu_usage > 80%?
  ├─► Memory usage: CES rds002_mem_usage > 85%?
  ├─► IOPS: CES rds045_iops > threshold?
  │   └─► Any high → Scale instance or optimize
  │
        ▼
Step 4: Check parameter settings
  ├─► long_query_time < 1?
  ├─► max_connections appropriate?
  ├─► innodb_buffer_pool_size adequate?
  │
[Resolved]
```

### 3.2 "High CPU" Diagnosis Flow

```
[High CPU Alert]
        │
        ▼
Step 1: Confirm metric
  └─► CES: rds001_cpu_usage > 80% for > 5 min?

Step 2: Identify cause
  ├─► Long-running queries?
  │   └─► Query: SHOW FULL PROCESSLIST
  ├─► Lock contention?
  │   └─► Query: SHOW ENGINE INNODB STATUS
  ├─► Full table scans?
  │   └─► Query: EXPLAIN for top queries
  │
        ▼
Step 3: Action
  ├─► Kill long-running query (if safe)
  │   └─► Command: KILL <process_id>
  ├─► Add indexes (if missing)
  ├─► Scale up instance (if sustained)
  │
[Resolved]
```

### 3.3 "Connection Pool Exhausted" Diagnosis Flow

```
[Connection Error: Too Many Connections]
        │
        ▼
Step 1: Check current connections
  └─► Query: SHOW STATUS LIKE 'Threads_connected'
  └─► Query: SHOW STATUS LIKE 'Max_used_connections'

Step 2: Identify connection leak
  ├─► Connection count increasing over time?
  │   └─► Yes → Application connection leak
  │
        ▼
Step 3: Check wait_timeout
  └─► Query: SHOW VARIABLES LIKE 'wait_timeout'
  └─► Value appropriate? (default 28800 = 8 hours)
  └─► If too high → Reduce to 300-600 seconds
  │
        ▼
Step 4: Check max_connections
  └─► Query: SHOW VARIABLES LIKE 'max_connections'
  └─► Value matches instance capacity?
  └─► If low → Increase via parameter modification
  │
        ▼
Step 5: Scale up
  └─► If sustained high → Resize to larger instance
  │
[Resolved]
```

---

## 4. Storage Issues

### 4.1 "Disk Full" Diagnosis Flow

```
[Disk Usage Alert: > 85%]
        │
        ▼
Step 1: Identify space consumption
  ├─► Query: SELECT table_schema, ROUND(SUM(data_length + index_length)/1024/1024,2) AS 'MB'
  │         FROM information_schema.tables GROUP BY table_schema
  │
        ▼
Step 2: Check binlog usage (MySQL)
  └─► Query: SHOW BINARY LOGS
  └─► Space used by binlogs?
  └─► If high → Purge old binlogs or reduce retention
  │
        ▼
Step 3: Check temp tables
  └─► Query: SHOW GLOBAL STATUS LIKE 'Created_tmp%'
  └─► High temp table creation → Optimize queries
  │
        ▼
Step 4: Check slow query log
  └─► Large slow query log files?
  └─► If yes → Truncate or reduce retention
  │
        ▼
Step 5: Scale storage
  └─► If above 90% → Expand volume immediately
  └─► Command: hcloud rds expand-volume --size <new_size>
  │
[Resolved]
```

### 4.2 "IOPS Bottleneck" Diagnosis Flow

```
[High IOPS Alert]
        │
        ▼
Step 1: Confirm IOPS metrics
  └─► CES: rds045_iops > threshold for > 5 min?

Step 2: Identify IO pattern
  ├─► Read vs Write heavy?
  │   └─► Query: SHOW GLOBAL STATUS LIKE 'Innodb_rows%'
  │
        ▼
Step 3: Check table fragmentation
  └─► Query: ANALYZE TABLE <table_name>
  └─► Optimize if fragmentation > 10%
  │
        ▼
Step 4: Check buffer pool
  └─► Query: SHOW GLOBAL STATUS LIKE 'Innodb_buffer_pool%'
  └─► Hit ratio < 95% → Increase buffer pool size
  │
        ▼
Step 5: Consider storage upgrade
  └─► ULTRAHIGH → ULTRAHIGHPRO or ESSD
  │
[Resolved]
```

---

## 5. Backup & Restore Issues

### 5.1 "Backup Failed" Diagnosis Flow

```
[Backup Failed Alert]
        │
        ▼
Step 1: Check backup status
  └─► DescribeInstance → backup_strategy status

Step 2: Check OBS quota
  └─► OBS bucket has sufficient space?
  └─► If no → Clean up old backups or expand quota
  │
        ▼
Step 3: Check instance status during backup
  └─► Is instance in BUILD or RESIZE state?
  │   └─► Yes → Wait for operation to complete
  │
        ▼
Step 4: Check disk space
  └─► Disk usage < 90%?
  │   └─► No → Expand volume first
  │
        ▼
Step 5: Retry backup
  └─► Manual: hcloud rds create-manual-backup
  │
[Resolved]
```

### 5.2 "Restore Failed" Diagnosis Flow

```
[Restore Failed Error]
        │
        ▼
Step 1: Verify backup status
  └─► ListBackups → backup status = COMPLETED?
  └─► If not → Choose different backup
  │
        ▼
Step 2: Check backup integrity
  └─► Backup size > 0?
  └─► If no → Backup corrupted; choose another
  │
        ▼
Step 3: Verify target instance
  └─► Instance status = ACTIVE?
  └─► If not → Wait or choose different instance
  │
        ▼
Step 4: Check disk space on target
  └─► Target has sufficient space for backup size?
  │
        ▼
Step 5: Check for conflicting operations
  └─► Instance in resize/backup operation?
  │   └─► Yes → Wait for operation to complete
  │
[Resolved]
```

---

## 6. Multi-Round Diagnosis Flow

### Round 1: Initial Diagnosis

1. **Collect symptom data**
   - Instance status
   - Recent metrics (CPU, memory, connections, disk)
   - Error logs from last 24 hours

2. **Execute standard checks**
   - Check instance state
   - Check resource utilization
   - Check recent backups

3. **Output initial hypothesis**

### Round 2: Critical Reflection

1. **Challenge assumptions**
   - Is the root cause really what Round 1 suggested?
   - Are there missing correlated metrics?
   - Are there missing dependent resources?

2. **Expand investigation**
   - Check network connectivity (VPC, SG, subnet)
   - Check cross-service dependencies
   - Compare with similar patterns in knowledge base

3. **Output revised hypothesis**

### Round 3: Deep Review (if needed)

1. **Execute Metrics→Logs→Traces trinity**
   - Query CES metrics for anomaly timeline
   - Query LTS for error logs during anomaly
   - Query AOM for application traces (if applicable)

2. **Expand time window**
   - Check from -2h to +1h around anomaly

3. **Check change history**
   - Any recent configuration changes?
   - Any recent deployments?
   - Any recent scaling events?

4. **Output final root cause with confidence**

---

## 7. Emergency Recovery Runbook

### 7.1 Instance Unreachable

```
Step 1: Check instance status
  └─► hcloud rds show --instance-id <id>
  └─► If status = FAILED → Contact support

Step 2: Check HA status (if applicable)
  └─► DescribeInstance → ha.replication_status
  └─► If abnormal → Initiate failover

Step 3: Check security group
  └─► Verify inbound rules allow access

Step 4: Restart instance
  └─► hcloud rds restart --instance-id <id>
  └─► Wait 5 minutes for recovery

Step 5: Escalate if not recovered
  └─► Contact Huawei Cloud support with instance ID and timeline
```

### 7.2 Data Corruption

```
Step 1: Stop writes
  └─► Alert application teams to pause writes

Step 2: Assess corruption scope
  └─► Identify affected databases/tables

Step 3: Determine restoration strategy
  ├─► Option A: Restore from backup (PITR to nearest point)
  ├─► Option B: Rebuild from application state
  └─► Option C: Manual repair (if partial corruption)

Step 4: Execute restoration
  └─► Restore to new instance (safer)
  └─► Validate data integrity

Step 5: Migrate application
  └─► Point application to restored instance
  └─► Resume operations

Step 6: Post-mortem
  └─► Document incident
  └─► Implement preventive measures
```

---

*This document defines troubleshooting patterns for RDS operations. Update with new error codes and patterns as discovered.*