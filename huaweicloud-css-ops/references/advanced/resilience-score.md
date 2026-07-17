# Resilience Score — CSS

> **Purpose**: CSS-specific resilience scoring model.
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

## 2. CSS-Specific Metrics

| Dimension | CSS-Specific Indicators |
|-----------|----------------------|
| Fault Detection | CES cluster health, shard status, JVM heap usage, disk usage, index health, search latency |
| Fault Isolation | Index-level isolation, shard allocation, cluster routing, CSS anti-affinity |
| Recovery Automation | Auto-snapshot, shard reallocation, index recovery, JVM restart |
| Degradation Quality | Read replica promotion, query timeout, partial results, search fallback |
| Data Consistency | Index replica sync, translog integrity, snapshot restore validation |

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
  product: "CSS"
  weights:
    fault_detection: 0.20
    fault_isolation: 0.20
    recovery_automation: 0.25
    degradation_quality: 0.15
    data_consistency: 0.20
```

## 5. CSS Fault Scenarios

| Scenario | Detection | Isolation | Recovery | Degradation | Data |
|----------|-----------|-----------|----------|-------------|------|
| Shard loss | CES cluster health + shard status | Index isolation | Auto-shard reallocation | Partial search | Replica sync |
| JVM OOM | CES jvm.heap_used alert | Node isolation | JVM restart + recovery | Node exclusion | Translog |
| Index corruption | CES index health alert | Index-level isolation | Snapshot restore | Read-only mode | Snapshot |
| Search timeout | CES search_latency alert | Query timeout + retry | Index refresh | Fallback results | N/A |
| Disk pressure | CES disk_usage alert | Quota enforcement | Snapshot + resize | Write throttling | Snapshot |

## 6. Compliance Checklist

- [x] All 5 dimensions have scoring criteria
- [x] Product-specific weights defined
- [x] Grade thresholds documented
- [x] Improvement recommendations provided
- [x] CSS-specific fault scenarios documented
