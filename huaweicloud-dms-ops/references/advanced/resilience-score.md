# Resilience Score — DMS

> **Purpose**: DMS-specific resilience scoring model.
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

## 2. DMS-Specific Metrics

| Dimension | DMS-Specific Indicators |
|-----------|------------------------|
| Fault Detection | CES queue depth alerts, consumer lag metrics, DMS topic partition status, consumer group heartbeat |
| Fault Isolation | Consumer group isolation, topic partitioning, queue-level isolation, DLQ (Dead Letter Queue) |
| Recovery Automation | Consumer group rebalance, partition reassignment, message redelivery, DLQ processing |
| Degradation Quality | Message buffering, producer/consumer retry, partial availability mode |
| Data Consistency | Message durability, offset commit integrity, partition lease management |

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
  product: "DMS"
  weights:
    fault_detection: 0.20
    fault_isolation: 0.20
    recovery_automation: 0.25
    degradation_quality: 0.15
    data_consistency: 0.20
```

## 5. DMS Fault Scenarios

| Scenario | Detection | Isolation | Recovery | Degradation | Data |
|----------|-----------|-----------|----------|-------------|------|
| Queue full | CES queue_depth alert | Producer rate limiting | Consumer scaling | Message buffering | DLQ retention |
| Consumer group failure | CES lag alert, heartbeat loss | Consumer group isolation | Rebalance + reconnect | Message redelivery | Offset commit |
| Partition rebalance | CES partition count change | Topic isolation | Automatic reassignment | Producer retry | Message order |
| Network partition | VPC route change | Security group rules | Route update | Producer buffering | N/A |
| DLQ overflow | CES DLQ depth alert | DLQ isolation | DLQ drain + reprocess | Message loss risk | DLQ retention |

## 6. Compliance Checklist

- [x] All 5 dimensions have scoring criteria
- [x] Product-specific weights defined
- [x] Grade thresholds documented
- [x] Improvement recommendations provided
- [x] DMS-specific fault scenarios documented
