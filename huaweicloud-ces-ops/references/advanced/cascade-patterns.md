# Cascade Patterns — CES

> **Purpose**: Cross-product and intra-product cascade fault patterns for Cloud Eye Service.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Cascade Pattern Overview

Cascade failures in monitoring can create blind spots during critical incidents. CES monitors other services, so its own failures have cascading effects on observability.

## 2. Intra-Product Cascade Patterns

### 2.1 CES → CES Cascade

| Pattern | Trigger | Propagation Path | Blocking Action |
|---------|---------|-----------------|-----------------|
| CES告警风暴 → 监控通道阻塞 | Alarm storm | Too many alarms → Channel full → Critical alarms delayed | 告警收敛 + 智能聚合 |
| CES指标丢失 → 监控盲区 | Metric gap | Agent failure → Dashboard stale → Anomaly undetected | 多实例部署 + 备份采集 |
| CES阈值过严 → 误告警泛滥 | Threshold misconfigured | False positives → On-call fatigue → Real alarm ignored | 动态阈值调整 |

## 3. Cross-Product Cascade Patterns

### 3.1 CES → ECS Cascade

| Pattern | Trigger | Propagation Path | Blocking Action |
|---------|---------|-----------------|-----------------|
| 监控告警触发 → 自动化脚本执行失败 | Alarm action failure | Script error → No remediation → Problem persists | 脚本测试 + 错误处理 |
| 监控数据丢失 → 扩缩容决策失误 | Metric gap | Bad decision → Over-provisioning or under-provisioning | 多数据源交叉验证 |

### 3.2 CES → CCE Cascade

| Pattern | Trigger | Propagation Path | Blocking Action |
|---------|---------|-----------------|-----------------|
| Pod指标缺失 → 调度失误 | Metric missing | Scheduler wrong decision → Resource imbalance | 补充监控 + 人工确认 |
| 监控告警触发 → K8s API压力 | Alarm action spam | Too many API calls → K8s API throttling → Other controllers affected | 限速 + 批量操作 |

### 3.3 CES → RDS Cascade

| Pattern | Trigger | Propagation Path | Blocking Action |
|---------|---------|-----------------|-----------------|
| CES数据库指标异常 → 监控仪表盘不可用 | CES DB slow/down | Dashboard timeout → No visibility during incident | RDS主备切换 + 监控数据本地缓存 |
| 监控查询超时 → 告警延迟 | Query timeout | Alarm delayed → Incident escalation slow | 查询优化 + 读写分离 |

## 4. Cascade Detection Rules

| Rule | Condition | Action |
|------|----------|--------|
| Alarm correlation | Multiple alarms for same resource within 5min | Group as single incident |
| Missing metrics | No data for > 5min for critical metric | Alert on data gap |
| Alarm loop | Same alarm triggered > 10 times in 1 hour | Suppress + investigate root cause |
| Cross-service correlation | ECS + CCE + RDS alarms simultaneously | Common cause investigation |

## 5. Blocking/Isolation Strategies

| Strategy | When Applied | Effectiveness |
|----------|-------------|---------------|
| Alarm suppression | Known maintenance window | Prevents false alerts |
| Alarm templating | Similar services | Consistent handling |
| Dynamic thresholds | Traffic patterns vary | Reduces false positives |
| Multi-channel routing | Primary channel down | Ensures delivery |
| Metric aggregation | High-frequency metrics | Reduces storage + processing |

## 6. Compliance Checklist

- [x] ≥2 cascade patterns documented (CES intra + cross-product)
- [x] Propagation paths clearly defined
- [x] Blocking actions specified
- [x] Detection rules documented
- [x] Monitoring-specific patterns (alarm storm, metric gap) covered
