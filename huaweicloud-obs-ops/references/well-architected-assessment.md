# Well-Architected Assessment — Huawei Cloud OBS

> Maps OBS operations to Well-Architected Framework (卓越架构) five pillars plus FinOps, SecOps, AIOps.

## 1. Framework Overview

| Pillar | OBS Focus | Status |
|--------|----------|--------|
| 安全 (Security) | Bucket policy vs ACL, IAM, encryption, public bucket risks | Required |
| 稳定 (Stability) | CRR, versioning, delete markers, lifecycle for log rotation | Required |
| 成本 (Cost) | Storage class optimization, lifecycle savings, egress cost | Required |
| 效率 (Efficiency) | Multipart upload, CDN, parallel transfer, transfer acceleration | Recommended |
| 性能 (Performance) | First-byte latency, concurrent connections, CDN caching | Required |

## 2. Five Pillar

### 2.1 安全 (Security)

#### Bucket Access Control Comparison

| Method | Scope | Use Case | Risk Level |
|--------|-------|----------|------------|
| **ACL** | Predefined: private/public-read/public-read-write | Simple access control | High if misconfigured |
| **Bucket Policy** | JSON with conditions (IP, user, referrer) | Fine-grained, conditional | Medium |
| **IAM Policy** | User-level, cross-bucket | Multi-bucket management | Low (centralized) |

**Minimum IAM Policy for OBS Operations:**
```json
{
  "Version": "1.1",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["obs:*List*", "obs:*Get*", "obs:*Put*", "obs:*Delete*"],
      "Resource": ["*"]
    }
  ]
}
```

#### Public Bucket Risk

- **CRITICAL**: `public-read-write` ACL = anyone can upload/delete → never use in production
- **HIGH**: `public-read` ACL without CDN protection = data exposure risk
- **Recommended**: `private` ACL + presigned URLs or CDN token auth

#### Encryption at Rest

| Method | Description | When to Use |
|--------|-------------|------------|
| **SSE-OBS** | OBS-managed encryption keys | Default, convenient |
| **SSE-KMS** | KMS-managed keys, customer-controlled | Compliance, key rotation control |
| **SSE-C** | Customer-provided keys | Highest security, manage own keys |

- All data in transit uses TLS — verify endpoint is `https://`
- Recommend bucket policy Deny for insecure transport:
  ```json
  {"Condition": {"Bool": {"obs:SecureTransport": "false"}}}
  ```

### 2.2 稳定 (Stability)

#### Versioning for Data Protection

- **Once enabled, cannot be disabled** — only suspended
- DELETE creates delete marker (not permanent removal)
- Each PUT creates new versionId
- Recovery: GET specific versionId or restore from previous version

#### Cross-Region Replication (CRR)

| Source | Destination | Replication Scope |
|--------|------------|-------------------|
| Bucket in Region-A | Bucket in Region-B | All objects or prefix-filtered |
| Requirements | Versioning enabled on both | Automatic replication of new objects |
| RPO | Minutes (depends on object size/count) | |
| RTO | Instant (destination already has objects) | |

#### DR Runbook

**Phase 1: Backup Verification**
1. Confirm versioning is enabled on source bucket
2. Verify latest objects have recent versionIds
3. Check CRR status if enabled

**Phase 2: Recovery Execution**
1. If CRR active: switch application to destination bucket endpoint
2. If no CRR: restore from last backup copy or previous versions
3. Verify object integrity (ETag comparison)

**Phase 3: Post-Recovery**
1. Verify application connectivity to recovered bucket
2. Monitor for data consistency issues
3. Document recovery duration

### 2.3 成本 (Cost)

#### Billing Model

| Component | Unit | Price Trend | Notes |
|-----------|------|------------|-------|
| Storage | GB/month | Varies by class | Standard > Warm > Cold > Deep Cold |
| Egress Traffic | GB | Highest cost factor | CDN egress is cheaper |
| API Requests | 10,000 calls | Negligible | PUT slightly more than GET |
| Data Retrieval | GB (Warm/Cold) | Varies | Cold has restore fees |

#### Storage Class Cost Savings

| Transition | Days Since Creation | Storage Cost Reduction | Retrieval Cost |
|-----------|-------------------|----------------------|----------------|
| Standard → Warm (IA) | 30 | ~50% | Per-GB retrieval fee |
| Warm → Cold (Archive) | 180 | ~70% vs Standard | Per-GB restore + retrieval |
| Cold → Deep Cold | 365 | ~85% vs Standard | Highest restore cost |

#### Idle Bucket Detection

| Indicator | Threshold | Action |
|-----------|-----------|--------|
| Zero requests for 30 days | request_count = 0 | Verify if still needed |
| Storage growth stalled, objects never accessed | object_count stable, requests = 0 | Consider lifecycle transition |
| Egress traffic spike | bytes_out > 3x average | Investigate, restrict ACL if suspicious |

### 2.4 效率 (Efficiency)

- **Multipart upload**: mandatory for >100MB files, recommended for >50MB
- **Parallel download**: obsutil `-threadNum 10` for fast transfers
- **CDN acceleration**: cache frequently accessed objects at edge
- **Transfer acceleration**: route via fastest network path to OBS
- **Lifecycle automation**: auto-transition, auto-expire, auto-abort incomplete uploads

### 2.5 性能 (Performance)

| Optimization | Technique | Expected Improvement |
|-------------|-----------|-------------------|
| First-byte latency | Enable CDN | 100ms → 10ms for cached objects |
| Throughput | Multipart + 10 concurrent threads | 10MB/s → 100MB/s |
| LIST performance | Hash-prefix key naming | Avoid hot partition bottleneck |
| Cache hit rate | Set appropriate Cache-Control headers | Reduce GET requests to OBS |

## 3. FinOps (财务运营)

### 3.1 成本可见性

| Tool | Use Case |
|------|----------|
| 费用中心 BSS | Monthly OBS billing breakdown |
| Cost Center (CCS) | Tag-based cost attribution per bucket |
| Budget (BUD) | Storage cost budget alerts at 80%/90% |

**Cost Tags per Bucket:**
- `cost_center`: team responsible
- `environment`: prod/staging/dev
- `data_type`: logs/backups/assets
- `retention`: 30d/1y/permanent

### 3.2 成本优化

| Waste Pattern | Detection | Fix |
|--------------|-----------|-----|
| Objects never accessed → Standard storage | request_count = 0 for 30d | Lifecycle transition to Warm/Cold |
| Abandoned multipart uploads | ListMultipartUploads shows stale uploads | Lifecycle abort rule (7 days) |
| Egress via OBS instead of CDN | bytes_out high, no CDN configured | Enable CDN, CDN egress is cheaper |
| Duplicate objects in multiple buckets | ETag comparison across buckets | Deduplicate, use single source |

### 3.3 成本问责

| Alert Threshold | Action |
|----------------|--------|
| Monthly OBS cost > 80% budget | Notify cost owner |
| Egress traffic > expected | Investigate source, restrict if unauthorized |
| Storage class distribution suboptimal | Review lifecycle rules |

## 4. SecOps (安全运营)

### 4.1 Identity Security

- Dedicated IAM user for OBS access (not root AK/SK)
- Minimum policy: only required obs:* actions per bucket
- AK/SK rotation: 90-day cycle
- Use IAM agency for cross-account OBS access

### 4.2 Network Security

- **VPC Endpoint** for private OBS access (no public internet traversal)
- Bucket policy with IP conditions for origin servers only
- No public bucket in production without CDN protection
- TLS enforced via bucket policy (Deny insecure transport)

### 4.3 Data Security

| Aspect | Recommendation |
|--------|---------------|
| At rest | SSE-KMS for production (customer-managed keys) |
| In transit | TLS 1.2+ enforced via policy |
| Access logging | Enable OBS logging → LTS for audit trail |
| Backup | Cross-region replication for critical data |
| Compliance | Align with 等保2.0 requirements for data protection |

### 4.4 Threat Detection

| Threat | Detection Method | Response |
|--------|----------------|----------|
| Public bucket exposure | Regular ACL audit script | Immediate: set to private |
| Unauthorized access | CTS audit log for OBS API calls | Revoke compromised AK |
| Mass deletion | Monitor DeleteObject rate spike | Enable versioning, restore from versions |
| Data exfiltration | Egress traffic anomaly (CES) | Restrict ACL, investigate source IP |

## 5. AIOps Integration

### 5.1 Anomaly Patterns (≥ 4)

| Pattern | Detection Logic | Severity |
|---------|----------------|----------|
| Access pattern anomaly | request_count deviates > 3σ from 7-day baseline | Medium |
| Egress spike | bytes_out > 3x daily average within 1 hour | Critical |
| Error rate surge | error_4xx_count or error_5xx_count > 5x baseline | High |
| Storage anomaly | storage_bytes growth > 2x expected rate | Medium |
| Latency degradation | first_byte_latency P95 > 500ms sustained | Medium |

### 5.2 Delegation Matrix

| Alarm | Primary Skill | Secondary |
|-------|--------------|-----------|
| Egress spike (security) | This skill (OBS) | Security team / HSS |
| Error rate spike | This skill (OBS) | CES for detailed metrics |
| High storage cost | This skill (FinOps section) | Billing skill |
| IAM policy violation | IAM skill | This skill |

### 5.3 Knowledge Base

Refer to `references/knowledge-base.md` for:
- 5+ product-specific fault patterns
- 2+ cross-product cascade scenarios
- Historical diagnosis with resolution times

### 5.4 Proactive Inspection

```
Schedule: Daily
1. List all OBS buckets in account
2. For each bucket: check ACL, versioning status, storage class distribution
3. Detect anomalies: public buckets, no versioning, no lifecycle rules
4. Generate compliance report
5. Alert on: public-read-write ACL, no encryption, versioning disabled on critical buckets
```
---

## Worker Output Contract (Read-Only Assessment Mode)

> Invoked when Well-Architected review sets `{{user.mode}}=well-architected-readonly`.
> Return **`{{output.product_assessment}}`** — field names MUST match the canonical schema.

**Canonical schema:** [worker-output-schema.md](../../huaweicloud-skill-generator/references/worker-output-schema.md)

| Constant | Value |
|----------|-------|
| `skill_id` | `huaweicloud-obs-ops` |
| `product` | `obs` |
| Finding `id` pattern | `obs-{rel|sec|cost|eff}-NNN` |

### Pillar → checklist map

| `pillars` key | Checklist source in this document |
|---------------|-------------------------------------|
| `reliability` | Stability / DR / backup sections |
| `security` | IAM / network / encryption sections |
| `cost` | FinOps / billing / idle detection sections |
| `efficiency` | Automation / batch / CI/CD sections |

### Example `{{output.product_assessment}}`

```json
{
  "skill_id": "huaweicloud-obs-ops",
  "product": "obs",
  "region": "cn-north-4",
  "scope": "account-wide",
  "assessment_date": "2026-06-19T10:00:00+08:00",
  "status": "OK",
  "partial": false,
  "resource_count": 1,
  "pillars": {
    "cost": {
      "score": 80,
      "status": "assessed",
      "findings": []
    },
    "efficiency": {
      "score": 80,
      "status": "assessed",
      "findings": []
    },
    "reliability": {
      "score": 80,
      "status": "assessed",
      "findings": []
    },
    "security": {
      "score": 80,
      "status": "assessed",
      "findings": []
    }
  },
  "recommendations": [],
  "trace": {
    "commands": [
      "hcloud obs read-only-list --region cn-north-4 (HW_SECRET_ACCESS_KEY=<masked>)"
    ],
    "request_ids": [
      "0123456789abcdef0123456789abcdef"
    ]
  },
  "errors": []
}
```
