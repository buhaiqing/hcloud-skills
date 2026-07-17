# Resilience Score — GaussDB

> **Purpose**: GaussDB-specific resilience scoring model.
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

## 2. GaussDB-Specific Metrics

| Dimension | GaussDB-Specific Indicators |
|-----------|----------------------------|
| Fault Detection | CES CPU/memory/disk alerts, DB connection failures, primary/standby switch events, GaussDB alarm notifications |
| Fault Isolation | AZ placement, DB instance isolation, connection pool limits, GaussDB distributed architecture |
| Recovery Automation | Primary-standby failover, automatic backup restore, GaussDB automatic recovery procedures |
| Degradation Quality | Read-only replica promotion, connection re-routing, query retry mechanisms |
| Data Consistency | WAL integrity, transaction log completeness, backup validation |

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
  product: "GaussDB"
  weights:
    fault_detection: 0.20
    fault_isolation: 0.20
    recovery_automation: 0.25
    degradation_quality: 0.15
    data_consistency: 0.20
```

## 5. GaussDB Fault Scenarios

| Scenario | Detection | Isolation | Recovery | Degradation | Data |
|----------|-----------|-----------|----------|-------------|------|
| Primary failure | CES alert + DB connection error | AZ redistribution | Auto-failover to standby | Read-only mode | WAL integrity |
| Transaction lock contention | CES lock wait timeout alerts | Session isolation | Long-running query kill | Retry with backoff | Transaction rollback |
| Storage space exhaustion | CES disk_usage alert at 85% | Quota enforcement | Auto-snapshot + resize | Write buffering | Snapshot |
| Backup failure | Backup job status alert | Backup queue isolation | Retry with exponential backoff | Point-in-time restore | Backup validation |
| Network partition | VPC route table change | Security group rules | Route update | Read-only mode | N/A |

## 6. Compliance Checklist

- [x] All 5 dimensions have scoring criteria
- [x] Product-specific weights defined
- [x] Grade thresholds documented
- [x] Improvement recommendations provided
- [x] GaussDB-specific fault scenarios documented
