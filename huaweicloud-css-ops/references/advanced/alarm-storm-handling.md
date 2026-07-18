# Alarm Storm Handling — CSS

> **Purpose**: Handle alarm storms for CSS clusters caused by health degradation, disk/jvm pressure, and node failures.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

| Threshold | Time Window | Severity |
|-----------|-------------|----------|
| cluster_health = Red | 实时 | Critical |
| cluster_health = Yellow | 实时 | Warning |
| disk_usage > 90% | 实时 | Critical（写被拒） |
| jvm_heap > 85% | 5 min | Warning |
| search_latency > 500ms | 5 min | Warning |
| node_count < expected | 实时 | Critical |

## 2. Aggregation Rules

| Original Alarm | Aggregated To |
|---------------|---------------|
| cluster_health + unassigned_shards | 「分片未分配」单根因 |
| disk_usage + jvm_heap | 「存储/JVM 压力」同源事件 |
| search_latency + cpu 同时异常 | 「查询过载」单根因 |
| 单节点故障 → shards unassigned → yellow → red | 按 cluster_id 聚合为「集群降级风暴」 |

## 3. Suppression Rules

| Condition | Action |
|-----------|--------|
| 同 cluster_id yellow→red 级联 | 抑制中间态，仅报最终 red |
| 单节点故障引发多 shard 告警 | 聚合为单条「节点故障」 |
| disk_usage 已 Critical | 抑制 jvm_heap 重复，标注根因 |
| search_latency 抖动 | 5 min 内聚合，计数不轰炸 |

## 4. Response Procedures

### P1 — Critical (Red / 磁盘满写拒 / 节点缺失)
```
1. 确认根因：磁盘满 / 节点掉线 / 分片未分配
2. 磁盘满：清理或扩容，解除 write block
3. 节点缺失：等待重加入或补充节点
4. 分片未分配：手动 reroute 或恢复快照
```

### P2 — Warning (Yellow / JVM 高 / 查询慢)
```
1. 降低写入速率，优化查询
2. 调整 JVM 堆或扩容数据节点
```

### P3 — Minor
```
1. 记录趋势，规划容量
```

```bash
# 查看集群与节点（子命令以 hcloud css --help 为准）
hcloud css list-clusters
hcloud css list-nodes --cluster <id>
```

## 5. Delegation Matrix

| Scenario | Delegate To | Escalation Trigger |
|----------|-------------|-------------------|
| 网络隔离/不通 | `huaweicloud-vpc-ops` | 节点不可达 |
| 加密密钥失效 | `huaweicloud-kms-ops` | 解密失败 |
| 快照存储异常 | `huaweicloud-obs-ops` | 快照写失败 |
| 指标/告警异常 | `huaweicloud-ces-ops` | 监控缺失 |
| 权限/账号问题 | `huaweicloud-iam-ops` | 403 创建资源 |
| 日志采集异常 | `huaweicloud-lts-ops` | 日志断流 |
