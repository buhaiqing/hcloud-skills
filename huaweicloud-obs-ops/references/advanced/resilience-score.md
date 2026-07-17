# Resilience Score — OBS

> **Purpose**: OBS-specific resilience scoring model.
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

## 2. OBS-Specific Metrics

| Dimension | OBS-Specific Indicators |
|-----------|------------------------|
| Fault Detection | CES bucket availability, object upload/download success rate, OBS alarm notifications, replication lag |
| Fault Isolation | Bucket-level IAM policies, bucket quotas, OBS multi-AZ placement, versioning |
| Recovery Automation | Versioning restore, cross-region replication, OBS automatic recovery, bucket policy recovery |
| Degradation Quality | Read-only mode, write throttling, CDN fallback, cached content serving |
| Data Consistency | Object checksum validation, versioning, cross-region replication status, MD5 integrity |

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
  product: "OBS"
  weights:
    fault_detection: 0.20
    fault_isolation: 0.20
    recovery_automation: 0.25
    degradation_quality: 0.15
    data_consistency: 0.20
```

## 5. OBS Fault Scenarios

| Scenario | Detection | Isolation | Recovery | Degradation | Data |
|----------|-----------|-----------|----------|-------------|------|
| Bucket policy error | OBS access denied alert | Bucket-level isolation | Policy restore from backup | Read-only mode | Versioning |
| Storage quota full | CES quota_usage alert | Bucket quota enforcement | Lifecycle policy execution | Write throttling | Versioning |
| Access restriction | CES auth failure alert | IAM policy audit | Policy correction | CDN fallback | N/A |
| Replication delay | CES replication_lag alert | Region isolation | Replication retry | Stale content | CRC check |
| Object loss | Object checksum failure | Versioning restore | Previous version restore | N/A | Checksum |

## 6. Compliance Checklist

- [x] All 5 dimensions have scoring criteria
- [x] Product-specific weights defined
- [x] Grade thresholds documented
- [x] Improvement recommendations provided
- [x] OBS-specific fault scenarios documented
