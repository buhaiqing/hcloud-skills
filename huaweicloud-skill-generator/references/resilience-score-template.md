# Resilience Score — Template

> **Purpose**: Template for product-specific resilience scoring models.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §13.2
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

## 2. Resilience Grades

| Score | Grade | Recommendation |
|-------|-------|---------------|
| 8-10 | A (Excellent) | Regular chaos validation, maintain level |
| 6-8 | B (Good) | Supplement missing fault scenarios |
| 4-6 | C (Fair) | Increase self-healing, improve degradation |
| 0-4 | D (Weak) | Prioritize critical resilience gaps |

## 3. Product-Specific Adjustments

Each product may adjust weights based on priorities:

```yaml
resilience_scoring:
  product: "[Product]"
  weights:
    fault_detection: 0.20
    fault_isolation: 0.20
    recovery_automation: 0.25
    degradation_quality: 0.15
    data_consistency: 0.20
```

## 4. Compliance Checklist

- [ ] All 5 dimensions have scoring criteria
- [ ] Product-specific weights defined
- [ ] Grade thresholds documented
- [ ] Improvement recommendations provided
