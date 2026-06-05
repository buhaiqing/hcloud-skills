# AIOps Best Practices — Huawei Cloud RDS

> Intelligent operations integration patterns for RDS for MySQL/PostgreSQL/SQL Server:
> slow-query correlation, replica lag detection, storage exhaustion early-warning,
> and self-healing for high-availability failover.
> **Version:** 1.0.0

## AIOps Goals for RDS

RDS is a stateful workload with hard SLOs. AIOps workflows should:

- Correlate slow query, replication lag, and disk pressure metrics
  with application-side traces (when available)
- Detect storage exhaustion before OOM / failover (predict 7-day exhaustion)
- Auto-remediate routine issues (kill long-running idle transactions,
  trigger failover on primary crash)
- Feed audit events to CTS for cross-skill correlation

## Recommended AIOps Patterns

### 1. Slow Query / Lock Wait Correlation

| Pattern | Metrics Correlated | Detection Logic | Remediation |
|---------|-------------------|-----------------|-------------|
| `slow_query_storm` | `slow_queries` (1-min) | rate > 50/min for 5 min | Page on-call, snapshot `SHOW PROCESSLIST` |
| `lock_wait_pileup` | `innodb_row_lock_waits` | delta > 100/min | Kill blocker (manual confirm) |
| `connection_saturation` | `used_connections / max_connections` | ratio > 0.8 | Drain idle connections, scale up |
| `replica_lag_growth` | `replica_lag_seconds` | slope > 10s/min | Pause writes or trigger read-only mode |
| `storage_exhaustion_predicted` | `disk_usage` (24h window) | linear extrapolation hits 95% in 7 days | Auto-ticket for storage expansion |

### 2. Storage Exhaustion Early-Warning

```bash
# Pseudo — pull disk usage, project exhaustion
hcloud rds list-instances -o json | jq '.instances[] | {id, name, disk_usage}'
```

For each instance, compute:

```
days_to_exhaustion = (100 - current_usage) / daily_growth_rate
```

If `days_to_exhaustion ≤ 7` → emit Critical alarm; if `≤ 30` → Warning.

### 3. Failover Correlation

When RDS primary enters `FAILING_OVER` state:

1. Snapshot `show-dr-replication` and `show-instance` outputs
2. Pull `replica_lag` from each replica
3. Cross-reference with CTS `ModifyInstance` events in past 24h
4. After failover completes, verify the new primary via:
   - `rds show-instance` (status = ACTIVE)
   - `rds list-databases` (read-only check via SDK)
5. Emit post-mortem ticket with the captured snapshot

### 4. Anomaly Storm Handling

When ≥ 3 RDS instances trigger Critical alarms within 5 min (e.g., shared
maintenance window, regional degradation):

1. Pause non-essential remediation (avoid cascade failover)
2. Snapshot current state of all 3 instances
3. Emit a single consolidated page
4. Auto-create a CES event tagged `aiops-cluster:rds`

## ML Integration Hooks

RDS AIOps can leverage the following CES metric streams:

| Metric | Aggregation | Use Case |
|--------|-------------|----------|
| `rds001_cpu_util` | 1-min | CPU saturation |
| `rds039_disk_usage` | 1-min | Storage pressure (predictive) |
| `rds044_innodb_row_lock_waits` | 1-min | Lock contention |
| `rds048_replica_lag` | 1-min | Replica health |
| `rds049_slow_queries` | 1-min | Query performance |
| `rds053_connections_usage` | 1-min | Connection pool |

## Cross-Skill Delegation Matrix

| Symptom | Delegate To |
|---------|-------------|
| Slow query from app code | This skill (analyze) + application owner (fix) |
| Disk full | This skill (resize) + `huaweicloud-ces-ops` (alarm) |
| Replica lag from heavy writes | `huaweicloud-ecs-ops` (app server) + this skill (read scaling) |
| Backup failure | `huaweicloud-cbr-ops` (vault) + this skill (RDS backup policy) |
| IAM permission denied | `huaweicloud-iam-ops` |
| Cost spike (over-sized instance) | `huaweicloud-billing-ops` |

## Self-Healing Playbook

| Trigger | Auto Action | Manual Step |
|---------|------------|-------------|
| Single long-running idle txn (>30 min) | Log + emit warning | Investigate after 1h if still running |
| Storage predicted to exhaust in ≤ 7 days | Open Jira ticket | Manual expansion |
| Replica lag > 60s sustained 5 min | Open Sev3 ticket | Manual promotion if needed |
| Primary crashes | Auto-failover (RDS built-in) | Verify post-failover |
| Slow query storm | Snapshot `SHOW PROCESSLIST` | DBA review |

## Reference: jq paths for RDS AIOps

```bash
# Storage pressure: instances with disk_usage > 80%
hcloud rds list-instances -o json | jq '.instances[] | select(.storage_usage > 80) | {id, name, usage: .storage_usage}'

# Replica lag per instance
hcloud rds list-replication-status -o json | jq '.replications[] | {instance_id, lag: .replica_lag}'

# Slow query top 10 (via DAS — Data Admin Service)
hcloud das show-slow-sql -o json | jq '.slow_sql[] | {sql, exec_time, count}'
```

## Knowledge Base Anchors

- RDS ↔ ECS / app server: `references/integration.md` §3
- Slow query analysis: `references/troubleshooting.md`
- Cost anomaly: `references/well-architected-assessment.md` §3 (FinOps)
