# Resilience Score — DCS

> **Purpose**: DCS-specific resilience scoring model.
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

## 2. DCS-Specific Metrics

| Dimension | DCS-Specific Indicators |
|-----------|------------------------|
| Fault Detection | CES memory/connections/CPU, Redis info metrics, client timeout rate |
| Fault Isolation | Proxy separation, slot distribution, cluster mode node isolation |
| Recovery Automation | Node restart, slot migration, cluster rebalance, backup restore |
| Degradation Quality | Read-only mode, partial availability, proxy fallback |
| Data Consistency | RDB/AOF integrity, slot state, data persistence verification |

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
  product: "DCS"
  weights:
    fault_detection: 0.20
    fault_isolation: 0.20
    recovery_automation: 0.25
    degradation_quality: 0.15
    data_consistency: 0.20
```

## 5. DCS Fault Scenarios

| Scenario | Detection | Isolation | Recovery | Degradation | Data |
|----------|-----------|-----------|----------|-------------|------|
| Node failure | CES node_status | Slot migration | Node restart/replace | Proxy fallback | RDB snapshot |
| Memory exhaustion | CES memory_usage | N/A | Memory eviction policy | Read-only mode | AOF backup |
| Network partition | VPC route table | Proxy isolation | Route recovery | Cluster split-brain | N/A |
| Replication failure | CES repl_status | N/A | Replica re-sync | Read-only replica | Data sync |
| Disk full | CES diskUsage | N/A | Storage expansion | Write rejection | RDB dump |

## 6. Compliance Checklist

- [x] All 5 dimensions have scoring criteria
- [x] Product-specific weights defined
- [x] Grade thresholds documented
- [x] Improvement recommendations provided
- [x] DCS-specific fault scenarios documented
