# Cascade Patterns — RDS

> **Purpose**: Cross-product and intra-product cascade fault patterns for Relational Database Service.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Cascade Pattern Overview

Database cascade failures often stem from resource exhaustion, slow queries, or connectivity issues. RDS failures impact all dependent services, making detection and isolation critical.

## 2. Intra-Product Cascade Patterns

### 2.1 RDS → RDS Cascade

| Pattern | Trigger | Propagation Path | Blocking Action |
|---------|---------|-----------------|-----------------|
| 主实例故障 → 备实例切换中断 | Failover failure | Primary down → Failover stuck → Service unavailable | 定期切换演练 + 监控切换延迟 |
| 连接数耗尽 → 新连接拒绝 | Max connections reached | Application errors → Connection timeout | 参数调优 + 连接池管理 |
| 慢查询 → 锁等待 → 连接堆积 | Slow query + lock | Query blocks → Connections accumulate → DB unresponsive | 慢查询优化 + 锁超时设置 |
| 存储满 → 写入失败 → 应用报错 | Storage full | INSERT/UPDATE fails → App error cascade | 容量监控 + 自动扩容 |
| 主备延迟 → 读写分离失效 | Replication lag | Read replica stale data → App confusion | 延迟监控 + 降级策略 |

## 3. Cross-Product Cascade Patterns

### 3.1 RDS → ECS Cascade

| Pattern | Trigger | Propagation Path | Blocking Action |
|---------|---------|-----------------|-----------------|
| DB响应慢 → ECS线程阻塞 | Slow queries | Connection busy → Thread pool exhaustion → App slow | 查询优化 + 连接超时设置 |
| DB连接耗尽 → 连接池失效 | Connection exhausted | New connection fails → App error → ECS health check fail | 连接池调优 + 熔断 |
| DB故障 → 应用重试风暴 | DB failover | Temporary disconnect → Retry storm → DB overload | 指数退避重试 + 熔断 |

### 3.2 RDS → CCE Cascade

| Pattern | Trigger | Propagation Path | Blocking Action |
|---------|---------|-----------------|-----------------|
| Pod连接池配置不当 → DB过载 | Connection pool misconfigured | Too many connections → DB CPU high → Service slow | 连接池参数调优 |
| CCE扩缩容 → 连接抖动 | Scaling events | Pods created/destroyed → Connection churn → DB load spike | 连接预热 + 缓冲池 |

### 3.3 RDS → ELB Cascade

| Pattern | Trigger | Propagation Path | Blocking Action |
|---------|---------|-----------------|-----------------|
| DB不可用 → ELB健康检查失败 | RDS down | Backend unhealthy → ELB remove → Traffic routed elsewhere | 快速检测 + 多AZ部署 |
| DB响应慢 → ELB超时 → 请求失败 | Slow DB | ELB timeout → 504 error → App failure | 超时调优 + 降级策略 |

## 4. Cascade Detection Rules

| Rule | Condition | Action |
|------|----------|--------|
| Connection saturation | Connections > 80% max | Alert + auto-scale connection pool |
| Replication lag | Lag > 30 seconds | Alert + check source/replica |
| Slow query | Query time > 5 seconds | Log + optimize |
| Storage usage | Usage > 85% | Alert + schedule cleanup |
| CPU saturation | CPU > 90% for > 5min | Alert + scale up |

## 5. Blocking/Isolation Strategies

| Strategy | When Applied | Effectiveness |
|----------|-------------|---------------|
| Read/Write split | Write pressure | Reduces primary load |
| Connection pooling | Many short connections | Efficient reuse |
| Query timeout | Long-running queries | Prevents resource hogging |
| Rate limiting | Unexpected traffic spike | Protects DB |
| Connection limits | Per-user limits | Prevents single-user exhaustion |
| Backup verification | Before major changes | Ensures recoverability |

## 6. Compliance Checklist

- [x] ≥2 cascade patterns documented (RDS intra + cross-product)
- [x] Propagation paths clearly defined
- [x] Blocking actions specified
- [x] Detection rules documented
- [x] Database-specific patterns (connection exhaustion, replication lag, slow query) covered
