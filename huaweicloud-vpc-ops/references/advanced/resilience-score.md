# Resilience Score — VPC

> **Purpose**: VPC-specific resilience scoring model.
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

## 2. VPC-Specific Metrics

| Dimension | VPC-Specific Indicators |
|-----------|------------------------|
| Fault Detection | CES bandwidth/connection, Peering connection status, NAT gateway health |
| Fault Isolation | Route table isolation, security group rules, network ACL boundaries |
| Recovery Automation | Route table update, peering reconnection, EIP rebinding |
| Degradation Quality | Secondary route activation, fallback peering, SNAT fallback |
| Data Consistency | Route table sync, peering state, ACL rule consistency |

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
  product: "VPC"
  weights:
    fault_detection: 0.20
    fault_isolation: 0.30  # Network isolation is paramount for VPC
    recovery_automation: 0.20
    degradation_quality: 0.15
    data_consistency: 0.15
```

## 5. VPC Fault Scenarios

| Scenario | Detection | Isolation | Recovery | Degradation | Data |
|----------|-----------|-----------|----------|-------------|------|
| Peering failure | CES conn_status | Route table switch | Re-establish peering | Secondary route | N/A |
| NAT gateway down | CES natgw_status | Security group | EIP rebind | Bastion host fallback | N/A |
| Route table corruption | VPC route sync | N/A | Route table restore | Static route fallback | Route backup |
| Bandwidth exhausted | CES bandwidth_usage | N/A | Quota increase | Rate limiting | N/A |
| Subnet exhaustion | CES subnet_ip | N/A | Secondary CIDR | IP allocation policy | N/A |

## 6. Compliance Checklist

- [x] All 5 dimensions have scoring criteria
- [x] Product-specific weights defined
- [x] Grade thresholds documented
- [x] Improvement recommendations provided
- [x] VPC-specific fault scenarios documented
