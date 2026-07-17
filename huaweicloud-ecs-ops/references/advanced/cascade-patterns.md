# Cascade Patterns — ECS

> **Purpose**: Cross-product and intra-product cascade fault patterns for Elastic Cloud Server.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Cascade Pattern Overview

Cascade failures occur when a fault in one component triggers faults in dependent components. ECS sits at the core of Huawei Cloud workloads, making cascade patterns critical for overall system reliability.

## 2. Intra-Product Cascade Patterns

### 2.1 ECS → ECS Cascade

| Pattern | Trigger | Propagation Path | Blocking Action |
|---------|---------|-----------------|-----------------|
| 实例故障 → 服务不可用 | Instance DOWN | ECS DOWN → ELB backend removed → Service 5xx | Auto-scale 或快速恢复 |
| 依赖服务 → 性能下降 | Downstream slow | Slow response → Timeout → Circuit open | 降级策略 |
| 宿主机故障 → 虚拟机漂移 | Host failure | VM migration → Network disruption → Service blip | 开启实例维持功能 |
| 磁盘IO瓶颈 → 系统卡顿 | Disk I/O saturation | IO wait → CPU steal → Application timeout | 使用SSD云盘或增强型SSD |

## 3. Cross-Product Cascade Patterns

### 3.1 ECS → RDS Cascade

| Pattern | Trigger | Propagation Path | Blocking Action |
|---------|---------|-----------------|-----------------|
| DB连接耗尽 | Connection pool exhausted | Max connections reached → New connections fail → App errors | RDS 参数调优 + 连接池管理 |
| DB响应慢 | Slow queries | Query timeout → App timeout → ECS CPU high | 优化查询或扩容RDS |
| ECS安全组变更 → DB访问中断 | Security group rule changed | SG denies → Connection refused → App error | 变更前确认依赖关系 |

### 3.2 ECS → ELB Cascade

| Pattern | Trigger | Propagation Path | Blocking Action |
|---------|---------|-----------------|-----------------|
| 后端故障 | ECS instance down | Health check fail → ELB remove → Traffic shift | 自动移除故障实例 |
| 后端超时 | ECS instance slow | Response timeout → ELB retry storm → Cascading timeout | 设置合理超时 + 熔断 |
| 后端异常流量 | DDoS from backend | Outbound traffic spike → ELB throttling → Legitimate traffic blocked | 流量清洗 + 限速 |

### 3.3 ECS → CCE Cascade

| Pattern | Trigger | Propagation Path | Blocking Action |
|---------|---------|-----------------|-----------------|
| 节点压力 | Node CPU/memory high | Pod evictions → Service disruption | 扩容节点池 |
| 节点网络异常 | Node network partition | Pod communication failure → Service degradation | 多AZ部署 |
| 节点磁盘满 | Node disk full | Pod cannot schedule → Service pending | 磁盘容量监控 + 清理 |

### 3.4 ECS → OBS Cascade

| Pattern | Trigger | Propagation Path | Blocking Action |
|---------|---------|-----------------|-----------------|
| OBS带宽瓶颈 | OBS throttling | Upload/download slow → App timeout | 请求分片 + 重试策略 |
| OBS访问凭证过期 | AK/SK expired | Auth failure → Service unable to access data | 自动滚动更新凭证 |

### 3.5 ECS → DCS Cascade

| Pattern | Trigger | Propagation Path | Blocking Action |
|---------|---------|-----------------|-----------------|
| Redis连接耗尽 | Redis max connections | New connections fail → Cache miss → DB overload | 连接池管理 + Redis规格升级 |
| Redis慢查询 | Redis slow command | Blocked operation → App thread exhaustion | 避免大KEY操作 |

## 4. Cascade Detection Rules

| Rule | Condition | Action |
|------|----------|--------|
| Time window | Fault A and Fault B within 30min | Consider cascade |
| Dependency | A affects B (known dependency) | A is likely root cause |
| Correlation | Same resource pool | Common cause likely |
| Latency spike | Response time increase > 3x | Check downstream dependencies |
| Error rate spike | 5xx rate > 10% | Correlate with infrastructure events |

## 5. Blocking/Isolation Strategies

| Strategy | When Applied | Effectiveness |
|----------|-------------|---------------|
| Circuit breaker | Cascade detected | Prevents overload propagation |
| Bulkhead | Service isolation | Limits blast radius |
| Rate limiting | Upstream slow | Protects downstream |
| Graceful degradation | Partial failure | Maintains partial function |
| Autoscaling | Resource pressure | Automatic capacity addition |
| Health check optimization | Frequent false positives | Reduces unnecessary removals |

## 6. Compliance Checklist

- [x] ≥2 cascade patterns documented (ECS intra + cross-product)
- [x] Propagation paths clearly defined
- [x] Blocking actions specified
- [x] Detection rules documented
- [x] Cross-product dependencies (RDS, ELB, CCE, OBS, DCS) covered
