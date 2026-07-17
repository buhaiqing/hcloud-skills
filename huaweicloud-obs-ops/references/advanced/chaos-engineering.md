# Chaos Engineering — OBS

> **Purpose**: Document fault injection experiments for OBS resilience verification.
> **Extends**: `huaweicloud-skill-generator/references/chaos-engineering-template.md`
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Fault Injection Experiment Design

| Experiment Type | Injection Method | Observed Metrics | Expected Behavior | Termination Condition |
|----------------|-----------------|-----------------|------------------|---------------------|
| Bucket policy error | Replace bucket policy with deny-all | Object access success rate | Access denied, monitoring alert | Access failure >50% for >3min |
| Storage quota full | Upload until quota exhausted | Write success rate, quota usage | Write failure, lifecycle trigger | Write failure >1min |
| Access restriction | Revoke IAM user permissions | Object operation success rate | Permission denied | Failure rate >20% for >2min |
| Replication delay | Slow cross-region replication | Replication lag, object consistency | Stale content served from replica | Lag >1h for >10min |
| Object deletion | Delete objects via API | Versioning restore time | Version restore | Data loss if no versioning |
| Network partition | Block OBS port via SG | Upload/download success rate | Retry with backoff | Failure rate >30% for >3min |

## 2. Resilience Score Model

### Scoring Dimensions (0-10 each)

| Dimension | Scoring Criteria | Weight |
|-----------|----------------|--------|
| Fault detection speed | Time from fault to CES/OBS alert | 20% |
| Fault isolation ability | Explosion radius (affected buckets) | 20% |
| Recovery automation | Auto-healing success rate, MTTR | 25% |
| Degradation quality | Read access during degradation | 15% |
| Data consistency | Object integrity after recovery | 20% |

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
  name: "obs-bucket-policy-error"
  objective: "Verify OBS handles bucket policy misconfiguration"

  preconditions:
    - "OBS bucket with versioning enabled"
    - "CES alarm configured for bucket access"
    - "IAM policy backup available"

  steps:
    - inject_fault: "Replace bucket policy with deny-all"
    - observe_metrics: "Monitor access denied rate via CES"
    - verify_behavior: "Confirm monitoring alert, check CDN fallback"
    - rollback_fault: "Restore original bucket policy"

  success_criteria:
    - "Alert triggered within 5min"
    - "CDN serves cached content"
    - "Policy restored without data loss"

  emergency_rollback:
    - "Restore bucket policy from backup"
    - "Clear CDN cache if needed"
    - "Verify object integrity via MD5"
```

## 4. OBS-Specific Experiment Details

### 4.1 Bucket Policy Error (Primary Scenario)

**Objective**: Verify bucket policy error detection and CDN fallback.

**Injection**:
```bash
# Replace bucket policy with deny-all
hcloud OBS PutBucketPolicy --bucket <bucket-name> --policy '{
  "Statement": [{"Effect": "Deny", "Principal": "*", "Action": "obs:*"}]
}'
```

**Metrics to Monitor**:
- `OBS.AccessDeniedCount` via CES
- Object upload/download success rate
- CDN cache hit rate

**Expected**: Access denied alert, CDN serves cached content.

### 4.2 Storage Quota Full

**Objective**: Verify quota enforcement and lifecycle policy execution.

**Injection**:
```bash
# Upload large objects until quota exhausted
for i in {1..100}; do
  hcloud OBS PutObject --bucket <bucket> --key "test/large_$i.dat" --body /dev/urandom
done
```

**Metrics**: Quota usage, write success rate, lifecycle trigger.

### 4.3 Replication Delay

**Objective**: Verify cross-region replication lag handling.

**Injection**: (Simulate slow replication by adding network delay on replica link - requires network configuration)

**Metrics**: Replication lag, object consistency check, staleness duration.

## 5. Emergency Rollback Procedures

| Scenario | Rollback Action |
|----------|-----------------|
| Policy lockout | Restore from policy backup, use OBS console emergency override |
| Quota overflow | Enable lifecycle policy, delete old versions |
| Replication failure | Retry replication, verify destination bucket |
| Data corruption | Restore from versioning, re-upload from source |

## 6. Compliance Checklist

- [x] ≥5 fault injection experiments designed (6 experiments)
- [x] Resilience scoring model defined (5 dimensions, 20% weights)
- [x] Experiment workflow documented (YAML format)
- [x] Emergency rollback procedures defined (4 scenarios)
