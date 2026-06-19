# Well-Architected Assessment — Huawei Cloud SWR

## 1. Security (安全)

### IAM Minimum Permissions

| Operation | Required Policy | Action |
|-----------|----------------|--------|
| List organizations/repos | `SWR ReadOnlyAccess` | `swr:*:list` |
| Create organization | `SWR FullAccess` | `swr:organization:create` |
| Create repository | `SWR FullAccess` | `swr:repository:create` |
| Delete organization | `SWR FullAccess` | `swr:organization:delete` |
| Delete repository | `SWR FullAccess` | `swr:repository:delete` |
| Delete image tag | `SWR FullAccess` | `swr:image:delete` |
| Push/pull image | `SWR FullAccess` | `swr:image:push` / `swr:image:pull` |

### Network Security

- Use VPC endpoints for private image pull/push without public internet
- Restrict repository access to specific IAM users/roles
- For public repositories, use with caution — anyone can pull
- Enable vulnerability scanning for all pushed images

### Data Protection

- Images encrypted at rest using platform-managed keys
- TLS 1.2+ for all data in transit (Docker HTTPS)
- Digest-based image verification (SHA256) for integrity
- Retention policies prevent accumulation of outdated images

## 2. Stability (稳定)

### High Availability

- SWR service is region-resilient (multi-AZ by default)
- Image data replicated within region across AZs
- Cross-region sync for DR (active-passive)

### Disaster Recovery

| Phase | Action | RTO | RPO |
|-------|--------|-----|-----|
| Phase 1 | Configure cross-region sync rules | <30 min | <15 min |
| Phase 2 | Failover CCE to DR region SWR | <1 hour | Configurable |
| Phase 3 | Verify image availability in DR | <30 min | N/A |

### Image Immutability

- Use `latest` tag carefully — it's mutable
- For production, use semantic version tags (e.g., `1.25.0`) for immutability
- Image digests provide guaranteed content-addressed references

## 3. Cost (成本)

### Cost Optimization

| Pattern | Detection | Recommendation |
|---------|-----------|---------------|
| Unused images | No pulls for >90 days | Delete image tags |
| Bloated repositories | >100 tags per repo | Set retention policy (keep last 10) |
| Cross-region pull cost | High egress charges | Use local region or sync to target region |
| Large image size | >5GB per image | Optimize Dockerfile (multi-stage, slim base) |

## 4. Efficiency (效率)

- Use retention policies for automatic old image cleanup
- Tag images with CI/CD pipeline metadata (build number, commit SHA)
- Use `ListImages` with sorting for cleanup automation
- Cross-region sync eliminates manual image distribution

## 5. Performance (性能)

| Factor | Limitation | Optimization |
|--------|-----------|-------------|
| Concurrent pulls | 300 per repo default | Request quota increase for large clusters |
| Push throughput | Depends on network bandwidth | Use in-region builds for faster push |
| Image size | 10GB default limit | Optimize Docker images |
| Cross-region pull latency | Higher latency | Sync images to local region |

## 6. FinOps (财务运营)

| Tool | Purpose |
|------|---------|
| CES `repo_storage_usage` | Track storage costs per repo |
| CES `repo_pull_count` | Identify most-used images |
| Retention policy | Automated cost reduction through cleanup |

## 7. SecOps (安全运营)

| Control | Implementation |
|---------|---------------|
| IAM least privilege | Use `SWR ReadOnlyAccess` for developers who only pull |
| Vulnerability scanning | HSS-integrated scanning on push |
| Private repositories | Default — no public access unless explicitly set |
| Image signing | Digest verification ensures tamper detection |
| Audit logging | CTS tracks all image pushes, deletes, and pulls |

## 8. AIOps (智能运营)

### Anomaly Patterns

| Pattern | Detection Logic | Severity |
|---------|----------------|----------|
| Pull failure rate spike | > 5% pull failures in 5min | Critical |
| Storage quota approaching | Total storage > 80% quota | Warning |
| Obsolete image accumulation | > 200 images per repo | Warning |
| Auth failure burst | > 10 auth failures in 5min | Critical |
| Cross-region sync broken | Sync lag > 1 hour | Warning |
| Image push failure | Push error rate > 1% | Critical |

### Cross-Skill Diagnosis

| Symptom | Primary Skill | Supporting Skills |
|---------|--------------|-------------------|
| Pod image pull failure | `huaweicloud-swr-ops` | `huaweicloud-cce-ops` (K8s) |
| Vulnerability found | `huaweicloud-swr-ops` | `huaweicloud-hss-ops` (scanning) |
| Auth/push denied | `huaweicloud-swr-ops` | `huaweicloud-iam-ops` (permissions) |
| Pull timeout (network) | `huaweicloud-swr-ops` | `huaweicloud-vpc-ops` (VPC endpoint) |
| Metric analysis | `huaweicloud-swr-ops` | `huaweicloud-ces-ops` (alarms) |
---

## Worker Output Contract (Read-Only Assessment Mode)

> Invoked when Well-Architected review sets `{{user.mode}}=well-architected-readonly`.
> Return **`{{output.product_assessment}}`** — field names MUST match the canonical schema.

**Canonical schema:** [worker-output-schema.md](../../huaweicloud-skill-generator/references/worker-output-schema.md)

| Constant | Value |
|----------|-------|
| `skill_id` | `huaweicloud-swr-ops` |
| `product` | `swr` |
| Finding `id` pattern | `swr-{rel|sec|cost|eff}-NNN` |

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
  "skill_id": "huaweicloud-swr-ops",
  "product": "swr",
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
      "hcloud swr read-only-list --region cn-north-4 (HW_SECRET_ACCESS_KEY=<masked>)"
    ],
    "request_ids": [
      "0123456789abcdef0123456789abcdef"
    ]
  },
  "errors": []
}
```
