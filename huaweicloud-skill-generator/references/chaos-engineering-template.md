# Chaos Engineering — Template

> **Purpose**: Template for documenting fault injection experiments and resilience verification.
> **Extends**: `huaweicloud-skill-generator/references/aiops-best-practices.md` §13
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Fault Injection Experiment Design

| Experiment Type | Injection Method | Observed Metrics | Expected Behavior | Termination Condition |
|----------------|-----------------|-----------------|------------------|---------------------|
| Instance failure | Stop primary instance | HA failover time, availability | ≤ 60s failover, ≥99.9% availability | Availability <99% for >5min |
| AZ failure | Block cross-AZ traffic | Cross-AZ request success rate | Degraded but not failed | Success rate <50% for >2min |
| Disk failure | Fill disk to 95% | Write success rate, alert trigger | Alert triggered | Write failure >1min |
| Load spike | Stress test CPU | AS scaling time, latency | 5min scaling, latency <500ms | Latency >5s for >10min |
| Dependency failure | Simulate downstream timeout | Circuit breaker, degradation | Circuit breaker triggered | Degradation failure >3min |

## 2. Resilience Score Model

### Scoring Dimensions (0-10 each)

| Dimension | Scoring Criteria | Weight |
|-----------|----------------|--------|
| Fault detection speed | Time from fault to alert | 20% |
| Fault isolation ability | Explosion radius control | 20% |
| Recovery automation | Self-healing success rate, MTTR | 25% |
| Degradation quality | Availability during degradation | 15% |
| Data consistency | Data integrity after recovery | 20% |

### Resilience Grades

| Score | Grade | Recommendation |
|-------|-------|---------------|
| 8-10 | A (Excellent) | Regular chaos validation, maintain |
| 6-8 | B (Good) | Supplement missing fault scenarios |
| 4-6 | C (Fair) | Increase self-healing, improve degradation |
| 0-4 | D (Weak) | Prioritize critical resilience gaps |

## 3. Chaos Experiment Workflow

```yaml
chaos_experiment:
  name: "[Experiment Name]"
  objective: "[What to verify]"

  preconditions:
    - "[Prerequisites]"

  steps:
    - inject_fault
    - observe_metrics
    - verify_behavior
    - rollback_fault

  success_criteria:
    - "[Expected outcomes]"

  emergency_rollback:
    - "[Rollback steps if things go wrong]"
```

## 4. Compliance Checklist

- [ ] ≥5 fault injection experiments designed
- [ ] Resilience scoring model defined
- [ ] Experiment workflow documented
- [ ] Emergency rollback procedures defined
