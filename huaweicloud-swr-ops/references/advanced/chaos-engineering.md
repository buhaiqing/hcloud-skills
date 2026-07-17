# Chaos Engineering — SWR

> **Purpose**: Document fault injection experiments for SWR resilience verification.
> **Extends**: `huaweicloud-skill-generator/references/chaos-engineering-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Fault Injection Experiment Design

| Experiment Type | Injection Method | Observed Metrics | Expected Behavior | Termination Condition |
|----------------|-----------------|-----------------|------------------|---------------------|
| Image repository deletion | Delete image repository via API | Build/deploy success rate, alert | Image unavailable, build fails | Build failure >50% for >10min |
| Build node failure | Stop build agent instances | Build queue depth, success rate | Build queuing, node replacement | Queue depth >100 for >15min |
| Trigger failure | Disable webhook trigger | Trigger activation rate | CI/CD pipeline blocked | Pipeline failure >30% for >5min |
| Permission change | Revoke SWR access IAM | API call success rate | Permission denied | Failure rate >50% for >3min |
| Organization access restriction | Modify org permissions | User/robot access rate | Access denied | Access failure >20% for >5min |
| Network partition | Block SWR API port | Image push/pull success rate | Retry with backoff | Failure rate >30% for >3min |

## 2. Resilience Score Model

### Scoring Dimensions (0-10 each)

| Dimension | Scoring Criteria | Weight |
|-----------|----------------|--------|
| Fault detection speed | Time from fault to CES/SWR alert | 20% |
| Fault isolation ability | Explosion radius (affected repos) | 20% |
| Recovery automation | Auto-healing success rate, MTTR | 25% |
| Degradation quality | Build availability during degradation | 15% |
| Data consistency | Image integrity after recovery | 20% |

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
  name: "swr-repository-deletion"
  objective: "Verify SWR handles repository deletion gracefully"

  preconditions:
    - "SWR organization with active repositories"
    - "CES alarm configured for repository status"
    - "Image backup to OBS configured"

  steps:
    - inject_fault: "Delete SWR repository via API"
    - observe_metrics: "Monitor build success rate, alert trigger"
    - verify_behavior: "Confirm build fails gracefully, alert fires"
    - rollback_fault: "Restore repository from backup"

  success_criteria:
    - "Alert triggered within 5min"
    - "Build fails with clear error message"
    - "Repository restored without image loss"

  emergency_rollback:
    - "Restore repository from OBS backup"
    - "Re-tag images if needed"
    - "Verify CI/CD pipeline恢复"
```

## 4. SWR-Specific Experiment Details

### 4.1 Image Repository Deletion (Primary Scenario)

**Objective**: Verify repository deletion detection and image restore.

**Injection**:
```bash
# Delete repository
hcloud SWR DeleteRepository --organization <org> --repository <repo>
```

**Metrics to Monitor**:
- `SWR.RepositoryCount` via CES
- Build success rate
- Image push/pull success rate

**Expected**: Build fails with clear error, alert fires.

### 4.2 Build Node Failure

**Objective**: Verify build queue handling and node replacement.

**Injection**:
```bash
# Stop build agent instances
hcloud ECS StopServers --instance_ids <build-agent-ids> --force
```

**Metrics**: Build queue depth, build success rate, node status.

### 4.3 Trigger Failure

**Objective**: Verify webhook trigger failure detection.

**Injection**:
```bash
# Disable webhook trigger
hcloud SWR UpdateTrigger --organization <org> --repository <repo> \
  --trigger <trigger-id> --enable false
```

**Metrics**: Trigger activation rate, pipeline trigger success rate.

## 5. Emergency Rollback Procedures

| Scenario | Rollback Action |
|----------|-----------------|
| Repository deletion | Restore from OBS backup, re-tag images |
| Build node failure | Auto-scaling replacement, drain queue |
| Trigger failure | Re-enable trigger, verify webhook URL |
| Permission change | Re-grant IAM permissions |

## 6. Compliance Checklist

- [x] ≥5 fault injection experiments designed (6 experiments)
- [x] Resilience scoring model defined (5 dimensions, 20% weights)
- [x] Experiment workflow documented (YAML format)
- [x] Emergency rollback procedures defined (4 scenarios)
