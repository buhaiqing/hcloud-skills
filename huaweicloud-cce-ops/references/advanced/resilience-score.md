# Resilience Score — CCE

> **Purpose**: CCE-specific resilience scoring model.
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

## 2. CCE-Specific Metrics

| Dimension | CCE-Specific Indicators |
|-----------|------------------------|
| Fault Detection | CES node/pod metrics, LTS workload logs, cluster health status |
| Fault Isolation | Namespace isolation, network policies, taints/tolerations |
| Recovery Automation | Auto-healing node, pod disruption budget, state'set rolling update |
| Degradation Quality | Pod disruption budget, graceful shutdown, PDB-protected services |
| Data Consistency | PVC snapshot, ETCD backup, persistent storage replication |

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
  product: "CCE"
  weights:
    fault_detection: 0.20
    fault_isolation: 0.25  # Kubernetes namespace/pod isolation is critical
    recovery_automation: 0.25
    degradation_quality: 0.15
    data_consistency: 0.15
```

## 5. CCE Fault Scenarios

| Scenario | Detection | Isolation | Recovery | Degradation | Data |
|----------|-----------|-----------|----------|-------------|------|
| Node failure | CES node_not_ready | PodDisruptionBudget | Auto-healing node | Pod rescheduling | PVC attachment |
| Pod crashloop | LTS log analysis | Namespace quota | Restart policy | Service unavailable | N/A |
| Memory pressure | CES memory usage | N/A | OOMKill handling | Eviction | N/A |
| Disk pressure | CES diskUsage | N/A | Storage expansion | Pod eviction | PVC snapshot |
| Network partition | VPC route table | NetworkPolicy | Route recovery | Read-only mode | N/A |

## 6. Compliance Checklist

- [x] All 5 dimensions have scoring criteria
- [x] Product-specific weights defined
- [x] Grade thresholds documented
- [x] Improvement recommendations provided
- [x] CCE-specific fault scenarios documented
