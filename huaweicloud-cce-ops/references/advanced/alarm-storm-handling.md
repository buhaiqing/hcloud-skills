# Alarm Storm Handling — CCE

> **Purpose**: Handle alarm storms for CCE clusters caused by node/control-plane failures, resource exhaustion, and infrastructure dependency outages.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

| Threshold | Time Window | Severity |
|-----------|-------------|----------|
| Node NotReady ≥3 | 实时 | Critical |
| Node CPU > 80% | 10 min | Warning |
| Node Disk > 90% | 实时 | Critical |
| OOM kills ≥2 | 5 min | Critical |
| Pod restart > 5 | 10 min | Warning |
| API server P99 > 1000ms | 5 min | Critical |

## 2. Aggregation Rules

| Original Alarm | Aggregated To |
|---------------|---------------|
| VPC 子网 IP 耗尽 + 节点无法创建 + Pod Pending | 「子网 IP 耗尽」单根因 |
| EVS 配额满 + PVC 失败 | 「存储配额耗尽」单根因 |
| 控制面升级 + API server 不可用 + Node NotReady | 「控制面升级中断」单根因 |
| 多 Node NotReady + OOM kills | 「节点资源风暴」(按 cluster) |

## 3. Suppression Rules

| Condition | Action |
|-----------|--------|
| 同源 IP 耗尽导致批量 Pending | 抑制下游 Pod 告警，仅报根因 |
| 控制面升级期间 API server 抖动 | 维护窗口内抑制，标注 planned |
| 同 cluster ≥5 Node 告警 | 聚合为单条摘要 |
| OOM kills 已聚合 | 5 min 内抑制重复进程级告警 |

## 4. Response Procedures

### P1 — Critical (节点 NotReady / 磁盘满 / OOM / API server 慢)
```
1. 定位根因：IP 耗尽 / 配额满 / 升级中断 / 资源挤占
2. 扩子网或释放 IP：hcloud vpc ...
3. 扩 EVS 配额或清理 PVC
4. 控制面异常：等待/回滚升级，保护 Node
```

### P2 — Warning (CPU 高 / Pod 重启)
```
1. 驱逐或扩容节点池
2. 排查重启循环（探针/依赖）
```

### P3 — Minor
```
1. 记录，优化调度与 limits
```

```bash
# 查看节点与集群状态（子命令以 hcloud cce --help 为准）
hcloud cce list-clusters
hcloud cce list-nodes --cluster <id>
```

## 5. Delegation Matrix

| Scenario | Delegate To | Escalation Trigger |
|----------|-------------|-------------------|
| 子网 IP 耗尽 | `huaweicloud-vpc-ops` | 无法扩容 |
| 节点 ECS 故障 | `huaweicloud-ecs-ops` | 底层实例异常 |
| EVS 配额/PVC 失败 | `huaweicloud-evs-ops` | 配额申请被拒 |
| 入口流量异常 | `huaweicloud-elb-ops` | 504 上涨 |
| 指标/告警异常 | `huaweicloud-ces-ops` | 监控缺失 |
| 日志采集异常 | `huaweicloud-lts-ops` | 日志断流 |
| 镜像拉取失败 | `huaweicloud-swr-ops` | 仓库不可达 |
| 费用突增 | `huaweicloud-billing-ops` | 预算越界 |
| 权限/Token 失效 | `huaweicloud-iam-ops` | 创建资源 403 |
