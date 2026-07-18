# AIOps Patterns — CCE

> **Purpose**: CCE-specific anomaly detection patterns for Kubernetes clusters.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `node_not_ready` | `cluster_node_status` = 0 | Critical | 检查节点 kubelet/网络，Cordon 后排水 |
| `node_cpu_pressure` | 节点 `cpu` > 80% | Major | 驱逐低优负载或扩容节点池 |
| `node_disk_pressure` | 节点 `disk` > 90% | Major | 清理容器镜像/日志，扩容数据盘 |
| `oom_kill` | 节点 `oom_killed` > 0 | Critical | 调整 Pod requests/limits |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `pod_restart_loop` | Pod `restart` > 5 次 / 10min | Major | 查看容器退出码，排查 OOM/CrashLoop |
| `api_server_latency_trend` | API server P99 持续上升逼近 1000ms | Major | 检查 etcd 与控制面负载 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `api_server_p99_spike` | API server P99 > 1000ms | Critical | 排查大规模 list/watch 请求 |
| `pod_pending_surge` | 大量 Pod 进入 `Pending` | Critical | 检查资源/调度约束 |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `subnet_ip_exhaustion` | VPC 子网 IP 耗尽 → 节点无法创建 → Pod `Pending` | Critical | 扩容子网或释放闲置 IP |
| `evs_quota_full` | EVS 配额满 → PVC 创建失败 | Critical | 提额或释放闲置磁盘 |
| `control_plane_upgrade` | 控制面升级 → API server 不可用 → `Node NotReady` | Critical | 等待升级完成，验证节点就绪 |

---

## 2. Alarm Storm Handling

告警风暴处理策略、抑制与聚合规则详见 `references/advanced/alarm-storm-handling.md`，本文件不重复。

---

## 3. Root Cause Analysis

1. **节点 NotReady 蔓延** → 检查 `cluster_node_status` 与 kubelet 日志 → 优先排除控制面升级导致的批量 NotReady。
2. **Pod 长时间 Pending** → 区分节点资源不足与网络层阻塞 → 查 VPC 子网 IP 余量确认是否 IP 耗尽。
3. **PVC Pending** → 检查 EVS 配额与存储类 → 配额满则提额或回收旧磁盘。
4. **API server P99 尖刺** → 关联控制面升级时间窗与大量 list/watch → 升级期间限流客户端请求。
5. **频繁 OOM Kill** → 比对 `oom_killed` 与 Pod `restart` → 收紧 limits 前先上调 requests 防驱逐风暴。
