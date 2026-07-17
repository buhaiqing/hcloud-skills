# Knowledge Base — SWR Fault Patterns

> **Purpose**: Common SWR fault patterns and resolution procedures.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## Fault Pattern 1: Image Pull Failure

### Symptom
CCE pods fail to pull images from SWR with `ImagePullBackOff` error.

### Root Causes
| Cause | Probability | Detection |
|-------|-------------|-----------|
|credentials expired | 40% | Check `docker login` expiry |
| network policy blocked | 30% | Check VPC endpoint config |
| image not exist | 20% | Verify image tag exists |
| quota exceeded | 10% | Check SWR quota limits |

### Resolution
1. Verify SWR credentials: `hcloud swr get-auth-config`
2. Check VPC endpoint: `hcloud vpc show endpoint`
3. Confirm image exists: `hcloud swr list-images`
4. Check quota: `hcloud swr show-quota`

### Delegation
- Network issues → `huaweicloud-vpc-ops`
- IAM issues → `huaweicloud-iam-ops`

---

## Fault Pattern 2: Storage Quota Exceeded

### Symptom
Cannot push new images; error: `storage quota exceeded`.

### Root Causes
| Cause | Probability | Detection |
|-------|-------------|-----------|
| old unused images | 60% | Analyze image age |
| large image layers | 30% | Check layer size |
| policy not enforced | 10% | Review cleanup policy |

### Resolution
1. List repositories: `hcloud swr list-repositories`
2. Analyze image size: `hcloud swr show-repository <repo>`
3. Delete old images: `hcloud swr delete-image <repo> <tag>`
4. Set retention policy: `hcloud swr update-repository <repo> --retention-days 30`

---

## Fault Pattern 3: Webhook Delivery Failure

### Symptom
Webhook notifications not delivered; console shows `webhook failure` alert.

### Root Causes
| Cause | Probability | Detection |
|-------|-------------|-----------|
| endpoint unreachable | 50% | Check endpoint health |
| TLS cert expired | 30% | Verify certificate expiry |
| payload too large | 15% | Check payload size |
| retry exhausted | 5% | Review retry config |

### Resolution
1. Test endpoint: `curl -I <webhook-url>`
2. Check cert: `echo | openssl s_client -connect <host:443>`
3. Verify payload size < 64KB
4. Check webhook logs: `hcloud swr list-webhook-logs`

---

## Fault Pattern 4: Pull Rate Throttling

### Symptom
Image pulls return `429 Too Many Requests`.

### Root Causes
| Cause | Probability | Detection |
|-------|-------------|-----------|
| too many concurrent pulls | 70% | Check pull rate metric |
| account rate limit | 20% | Check quota config |
| region outage | 10% | Check region status |

### Resolution
1. Check current rate: `hcloud ces get-metric-data --metric pull_count`
2. Implement pull caching (imagePullPolicy: Always)
3. Use multiple replicas for load distribution
4. Request rate limit increase via ticket

---

## Fault Pattern 5: Repository Access Denied

### Symptom
IAM user cannot access repository; error: `permission denied`.

### Root Causes
| Cause | Probability | Detection |
|-------|-------------|-----------|
| missing IAM permission | 50% | Check IAM policy |
| RAM user not in org | 30% | Verify org membership |
| policy not propagated | 20% | Check policy version |

### Resolution
1. Check IAM policy: `hcloud iam list-policies --name SWRAdmin`
2. Add RAM user to org: `hcloud iam add-user-to-org <user-id>`
3. Update policy version: `hcloud iam update-policy-version <policy-id>`
4. Delegation → `huaweicloud-iam-ops`

---

## Fault Pattern 6: Build Trigger Failure

### Symptom
Automated build not triggered on code push.

### Root Causes
| Cause | Probability | Detection |
|-------|-------------|-----------|
| trigger disabled | 40% | Check trigger status |
| branch mismatch | 30% | Verify branch config |
| token expired | 20% | Check SCM token |
| webhook blocked | 10% | Check SCM webhook |

### Resolution
1. List triggers: `hcloud swr list-triggers <repo>`
2. Enable trigger: `hcloud swr update-trigger <repo> <trigger> --enable`
3. Refresh SCM token: `hcloud swr update-trigger <repo> <trigger> --token <new-token>`
4. Test trigger: `hcloud swr test-trigger <repo> <trigger>`

---

## Cross-Skill Escalation Matrix

| Fault | Escalate To | Trigger Condition |
|-------|-------------|-------------------|
| Network timeout | `huaweicloud-vpc-ops` | VPC endpoint unreachable |
| IAM denied | `huaweicloud-iam-ops` | Permission check failed |
| Vulnerability found | `huaweicloud-hss-ops` | Image scan critical |
| Metric anomaly | `huaweicloud-ces-ops` | CES alarm triggered |
| Audit trail needed | `huaweicloud-cts-ops` | Delete/push event |
