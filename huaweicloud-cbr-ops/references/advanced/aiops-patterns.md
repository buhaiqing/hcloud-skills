# AIOps Patterns — CBR

> **Purpose**: CBR-specific anomaly detection patterns for cloud backup and recovery.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|-----------------|
| `backup_job_queue_buildup` | Pending jobs > 100 | Warning | 增加备份并发 |
| `storage_quota_near` | Backup storage > 80% | Warning | 清理过期备份 |
| `restore_latency_high` | Restore time > expected | Warning | 检查存储后端 |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|-----------------|
| `backup_failure_rate_up` | Failure rate > 5% | Warning | 检查备份策略 |
| `restore_request_spike` | Restore requests > 2x average | Warning | 检查是否有故障 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|-----------------|
| `backup_duration_spike` | Duration > 2x normal | Warning | 检查数据量变化 |
| `checkpoint_failed` | Checkpoint failure | Critical | 检查存储状态 |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|-----------------|
| `storage_backup_correlation` | High storage + high backup load | Warning | 正常运维 |
| `failure_duration_correlation` | Long failures + specific vaults | Warning | 检查特定Vault |
