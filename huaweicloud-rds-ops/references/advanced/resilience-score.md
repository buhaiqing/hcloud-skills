# Resilience Score — RDS

> **Purpose**: RDS-specific resilience scoring model.
> **Extends**: `huaweicloud-skill-generator/references/resilience-score-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Scoring Dimensions

| Dimension | Description | Weight | Score Criteria (0-10) |
|-----------|-------------|--------|---------------------|
| Fault Detection Speed | Time from fault occurrence to alert | 20% | 0=never detected, 10=<1min |
| Fault Isolation | Explosion radius control, cascade protection | 20% | 0=no isolation, 10=instant |
| Recovery Automation | Self-healing success rate, MTTR | 25% | 0=no recovery, 10=auto <5min |
| Degradation Quality | Availability during degradation | 15% | 0=total failure, 10=transparent |
| Data Consistency | Data integrity after recovery | 20% | 0=data loss, 10=zero loss |

## 2. RDS-Specific Metrics

| Dimension | RDS-Specific Indicators |
|-----------|------------------------|
| Fault Detection | CES CPU/memory/connections, RDS error logs, primary/replica lag |
| Fault Isolation | AZ redundancy, read replica separation, connection pooling |
| Recovery Automation | Primary replica failover, backup restore, point-in-time recovery |
| Degradation Quality | Read replica promotion, connection draining, read-only mode |
| Data Consistency | Backup success rate, binlog integrity, transaction log replay |

## 3. Resilience Grades

| Score | Grade | Recommendation |
|-------|-------|---------------|
| 8-10 | A (Excellent) | Regular chaos validation, maintain level |
| 6-8 | B (Good) | Supplement missing fault scenarios |
| 4-6 | C (Fair) | Increase self-healing, improve degradation |
| 0-4 | D (Weak) | Prioritize critical resilience gaps |

## 4. Product-Specific Weights

```yaml
resilience_scoring:
  product: "RDS"
  weights:
    fault_detection: 0.15  # DB fault detection is well-instrumented
    fault_isolation: 0.20
    recovery_automation: 0.25
    degradation_quality: 0.15
    data_consistency: 0.25  # Data consistency is critical for databases
```

## 5. RDS Fault Scenarios

| Scenario | Detection | Isolation | Recovery | Degradation | Data |
|----------|-----------|-----------|----------|-------------|------|
| Primary failure | CES instance_status | AZ failover | Auto-failover | Read replica promotion | Binlog replay |
| Connection overflow | CES connections | Connection pool | Max_connections adjustment | Connection queuing | N/A |
| Replication lag | CES repl_lag | N/A | Replica resync | Read-only mode | N/A |
| Storage full | CES diskUsage | N/A | Storage expansion | Write block | Backup |
| Backup failure | CES backup_status | N/A | Retry backup | Risk notification | Data loss risk |

## 6. Compliance Checklist

- [x] All 5 dimensions have scoring criteria
- [x] Product-specific weights defined
- [x] Grade thresholds documented
- [x] Improvement recommendations provided
- [x] RDS-specific fault scenarios documented
