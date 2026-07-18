# Alarm Storm Handling — RDS

> **Purpose**: Procedures for handling RDS alarm storms with ≥4 concurrent anomaly patterns (primary/standby switch, connection pool exhaustion, slow queries, storage full, read-only, backup failure).
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

### 1.1 Detection Criteria (rds monitoring.md §7)

Alarm storm trigger thresholds:

- **>10 告警 / 5min** 进入风暴状态
- 同实例 >3 告警 → 聚合为单事件
- 同 namespace >50% 告警聚焦 RDS → 标记为 RDS 域风暴
- 告警 A 触发后 2min 内触发 B → 标记 B「由 A 引起」

### 1.2 RDS Anomaly Signals

| 信号 | 指标 | 阈值 |
|------|------|------|
| 主备切换 | ha_switch / rds_ha_lag | rds_ha_lag > 5s |
| 连接池耗尽 | connections_used / max | > 90% |
| 慢查询突增 | slow_query_count / rds043_slow_queries | p99 > 5000ms |
| 存储满 | rds004_disk | disk_usage > 85% → 只读 |
| 数据库只读 | read_only_state | 非预期只读 |
| 备份失败 | backup_status | failed |

### 1.3 Severity Classification

| Level | Criteria | Response |
|-------|----------|----------|
| P1 | 主备切换 + 只读/存储满 | 立即 |
| P2 | 连接饱和 + CPU 高 (RDS-P002) | < 15 min |
| P3 | 1-2 慢查询/备份告警 | < 1 hour |

---

## 2. Aggregation Rules

- 同实例 5min 内 >3 告警 → 聚合为单事件，附关联指标快照
- 同 namespace >50% 告警指向 RDS → 升级为域级风暴，统一指挥
- 关联模式自动归并：
  - **RDS-P001**: rds001_cpu_usage + rds002_mem_usage 双高 → 单根因「资源瓶颈」
  - **RDS-P002**: rds003_connections 饱和 + rds001_cpu_usage 高 → 单根因「连接风暴」
- 因果链：A 触发后 2min 内 B 触发 → B 折叠进 A 事件，不单独计告警

---

## 3. Suppression Rules

```yaml
suppression_rules:
  - pattern: "slow_query_spike"
    window_minutes: 10
    max_alarms: 3
  - pattern: "backup_failed"
    window_minutes: 60
    max_alarms: 1
  - pattern: "disk_usage_high"
    window_minutes: 30
    max_alarms: 2
```

- Critical（主备切换/只读/存储满）：不抑制，立即通知
- Warning（慢查询/连接>90%）：聚合通知（≤1 / 15min）
- Info：仅记录日志

---

## 4. Response Procedures

### 4.1 P1 — 主备切换/只读/存储满

```
立即动作:
1. 确认主备切换状态与 RDS-P001 关联
2. 存储满 → 扩容或清理: hcloud rds expand-volume <instance_id> --size <gb>
3. 非预期只读 → 排查只读副本权重与磁盘水位
4. 10min 未恢复 → 升级 on-call
```

```bash
# 查看实例指标快照
hcloud rds show-metrics <instance_id> --metrics rds001_cpu_usage,rds002_mem_usage,rds004_disk
# 查看主备延迟
hcloud rds show-replication <instance_id>
```

### 4.2 P2 — 连接饱和 + CPU 高 (RDS-P002)

```bash
# 查看连接来源 Top
hcloud rds list-connections <instance_id> --top 20
# 杀除空闲长连接
hcloud rds kill-session <instance_id> --idle-minutes 30
```

### 4.3 P3 — 慢查询/备份失败

```bash
# 慢查询明细
hcloud rds show-slow-log <instance_id> --threshold 5000
# 备份重试
hcloud rds retry-backup <instance_id>
```

---

## 5. Delegation Matrix

| Scenario | Delegate To | Escalation Trigger |
|----------|-------------|-------------------|
| 监控指标缺失/阈值调优 | `huaweicloud-ces-ops` | 指标无数据 > 5 min |
| 网络/子网/安全组 | `huaweicloud-vpc-ops` | 连接超时且非 RDS 侧 |
| 备份存储 (OBS) | `huaweicloud-obs-ops` | 备份落盘失败持续 |
| 权限/密钥问题 | `huaweicloud-iam-ops` | AccessDenied 反复出现 |
| 日志分析/审计 | `huaweicloud-lts-ops` / AOM | 需溯源慢查询 SQL |
