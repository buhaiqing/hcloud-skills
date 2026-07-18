# AIOps Patterns — ECS

> **Purpose**: ECS-specific anomaly detection patterns for compute instances.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `cpu_saturation` | `cpu_util` > 95% 持续 10min (错误码 ECS-001) | Critical | 垂直扩容或更换更大 flavor |
| `disk_exhaustion` | `diskUsage` > 90% 或 `write_iops` 突降 (ECS-002) | Critical | 清理日志/扩容磁盘 |
| `mem_leak` | `mem_usedPercent` 斜率 > 0.5%/min (ECS-003) | Major | 重启实例或排查应用内存泄漏 |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `mem_leak_trend` | 内存使用率线性上升无回落拐点 | Major | 抓取 heap dump，定位泄漏对象 |
| `disk_fill_acceleration` | 磁盘使用率增速 > 前 24h 均值 2× | Major | 定位高速写入进程 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `network_storm` | 网络包速率 `pps` > 10× 基线 | Major | 排查异常流量/限速 |
| `security_group_misconfig` | 关键指标（如 `cpu_util`）上报消失 (ECS-004) | Critical | 检查安全组/agent 连通性 |
| `spot_reclaim` | 实例状态变为 `TERMINATED` (ECS-006) | Critical | 检查竞价策略，重建按需实例 |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `cpu_mem_dual_high` | `cpu_util` 与 `mem_usedPercent` 同时 > 90% | Critical | 资源瓶颈，扩容或优化 |
| `upstream_cascade` | ELB `active_connection`↓ + ECS `cpu_util`↓ + ELB 504↑ + RDS `connections`↓ 同源 | Critical | 定位单根因（ECS 宕机），恢复实例 |

---

## 2. Alarm Storm Handling

告警风暴处理策略、抑制与聚合规则详见 `references/advanced/alarm-storm-handling.md`，本文件不重复。

---

## 3. Root Cause Analysis

1. **CPU 满载 + 指标消失** → 检查实例存活状态 `hcloud ecs list-servers --server-id <id>` → 安全组/agent 是否阻断监控上报（ECS-004）。
2. **磁盘写被拒 + IOPS 突降** → 排查 `write_iops` 与 `diskUsage` → 定位大文件/日志，清理或扩容。
3. **内存持续上升无回落** → 比对 `mem_usedPercent` 斜率 → 抓取进程内存，确认泄漏后重启容器/实例。
4. **竞价实例 TERMINATED** → 检查实例状态与回收事件 → 评估改为按需或调整竞价上限。
5. **ELB/RDS 指标串联异常** → 以 ECS `cpu_util` 消失为根因锚点 → 先恢复 ECS 再观察级联指标回落。
