# Resilience Score — ECS

> **Purpose**: ECS-specific resilience scoring model.
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

## 2. ECS-Specific Metrics

| Dimension | ECS-Specific Indicators |
|-----------|------------------------|
| Fault Detection | CES CPU/memory/disk alerts, agent heartbeat loss, host hardware failure |
| Fault Isolation | VPC security groups, ECS placement groups, anti-affinity zones |
| Recovery Automation | Auto-recovery on failure, ASG scaling, snapshot-based restore |
| Degradation Quality | Load balancing failover, instance replacement, degraded mode operation |
| Data Consistency | EVS snapshot integrity, data backup success rate |

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
  product: "ECS"
  weights:
    fault_detection: 0.20
    fault_isolation: 0.20
    recovery_automation: 0.25
    degradation_quality: 0.15
    data_consistency: 0.20
```

## 5. ECS Fault Scenarios

| Scenario | Detection | Isolation | Recovery | Degradation | Data |
|----------|-----------|-----------|----------|-------------|------|
| Instance failure | CES heartbeat | AZ redistribution | Auto-recovery | ASG replacement | EVS snapshot |
| CPU overload | CES cpu_util | ASG scale-out | Auto-scaling | Rate limiting | N/A |
| Memory leak | CES mem_usedPercent | N/A | Process restart | Graceful degradation | N/A |
| Disk full | CES diskUsage | N/A | Snapshot + resize | Write buffering | Snapshot |
| Network partition | VPC route table | Security group rules | Route update | Read-only mode | N/A |

## 6. Compliance Checklist

- [x] All 5 dimensions have scoring criteria
- [x] Product-specific weights defined
- [x] Grade thresholds documented
- [x] Improvement recommendations provided
- [x] ECS-specific fault scenarios documented
