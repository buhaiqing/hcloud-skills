# Alarm Storm Handling — SWR

> **Purpose**: Procedures for handling SWR alarm storms with ≥4 concurrent anomaly patterns.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

### 1.1 Detection Criteria

An alarm storm is triggered when:
- ≥5 alarms within 10 minutes for same SWR resource
- ≥3 different anomaly types simultaneously
- Critical severity alarm + ≥2 warning alarms

### 1.2 Severity Classification

| Level | Criteria | Response Time |
|-------|----------|---------------|
| P1 | Critical alarm + ≥4 warnings | Immediate |
| P2 | ≥3 warnings or 2 critical | < 15 min |
| P3 | 1-2 warnings | < 1 hour |

---

## 2. Response Procedures

### 2.1 P1 — Critical Alarm Storm

```
Immediate Actions:
1. Identify root cause alarm (first critical)
2. Isolate affected repository: hcloud swr disable-repository <repo>
3. Notify affected teams via enterprise IM
4. Escalate to on-call if not resolved in 10 min
```

### 2.2 P2 — Warning Alarm Storm

```
Actions within 15 min:
1. Aggregate alarms by pattern type
2. Check correlation: storage + pull + webhook
3. If correlated → single root cause → fix源头
4. If independent → parallel resolution
```

### 2.3 P3 — Minor Storm

```
Actions within 1 hour:
1. Log all alarms for analysis
2. Schedule root cause analysis
3. Apply preventive measures
```

---

## 3. Pattern-Specific Handling

### 3.1 Storage + Pull Correlation (High Priority)

When storage pressure and pull throttling occur together:

1. **Immediate**: Clear old unused images (`age > 90 days`)
2. **Short-term**: Review image retention policy
3. **Long-term**: Implement image lifecycle automation

```bash
# Emergency cleanup
hcloud swr list-repositories | jq -r '.[] | "\(.namespace)/\(.name)"' | \
  while read repo; do
    hcloud swr list-images "$repo" | jq -r '.[] | select(.created_at < (now - 90*86400)) | .tag' | \
      while read tag; do
        echo "Deleting $repo:$tag"
        hcloud swr delete-image "$repo" "$tag"
      done
  done
```

### 3.2 Webhook + Build Failure Correlation

When webhook failures correlate with build trigger failures:

1. **Check**: SCM webhook endpoint accessibility
2. **Check**: TLS certificate validity
3. **Action**: Refresh webhook token if expired

### 3.3 Multi-Resource Alarm Storm

When ≥3 repositories show same anomaly pattern:

1. **Identify**: Common factor (same VPC, same IAM policy, same region)
2. **Check**: Region-level service status
3. **Escalate**: To infrastructure team if region-level issue

---

## 4. Suppression Rules

### 4.1 Time-Window Suppression

```yaml
suppression_rules:
  - pattern: "storage_quota_near"
    window_minutes: 15
    max_alarms: 3

  - pattern: "pull_throttling"
    window_minutes: 10
    max_alarms: 5

  - pattern: "webhook_failure_high"
    window_minutes: 30
    max_alarms: 2
```

### 4.2 Severity Override

- **Critical**: No suppression, immediate notification
- **Warning**: Aggregated notification (max 1 per 15 min)
- **Info**: Suppressed, logged only

---

## 5. Post-Incident Analysis

### 5.1 Required Analysis

After alarm storm resolution:
1. Root cause identification
2. Time-to-resolution metrics
3. False positive rate
4. Pattern frequency analysis

### 5.2 Preventive Measures

| Pattern | Preventive Action |
|---------|------------------|
| storage_quota_near | Set up automated cleanup (retention policy) |
| pull_throttling | Implement multi-region caching |
| webhook_failure_high | Add webhook monitoring + redundancy |
| image_count_growth | Set quota per namespace |

---

## 6. Delegation Matrix

| Scenario | Delegate To | Escalation Trigger |
|----------|-------------|-------------------|
| VPC endpoint down | `huaweicloud-vpc-ops` | Endpoint unreachable > 5 min |
| IAM policy issue | `huaweicloud-iam-ops` | Permission denied after fix |
| Region outage | `huaweicloud-ces-ops` | Multi-resource affected |
| Security incident | `huaweicloud-hss-ops` | Vulnerability detected |
