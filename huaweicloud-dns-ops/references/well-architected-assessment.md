# DNS Well-Architected + Three-Pillar Assessment

## 1. Security Pillar (安全支柱)

### IAM Minimum Permissions

| Role | Required Permissions |
|---|---|
| DNS Viewer | `DNS ReadOnly` |
| DNS Operator | `DNS FullAccess` — CRUD on zones and recordsets |
| DNS Admin | `DNS FullAccess` + DNSSEC management |

### DNSSEC

| Setting | Recommendation |
|---|---|
| Enable DNSSEC | Enable on all public zones to prevent DNS spoofing |
| DS record | Configure DS record at your registrar for full chain |
| Key rotation | Plan annual DNSSEC key rotation |

### Zone ACL (Private Zones)

| Setting | Recommendation |
|---|---|
| VPC scope | Bind private zone to specific VPCs only |
| IAM policy | Restrict who can modify private zone records |

## 2. Stability Pillar (稳定支柱)

### Multi-NS Design

| Practice | Recommendation |
|---|---|
| NS redundancy | Huawei Cloud DNS provides ≥2 authoritative NS per zone |
| TTL strategy | Short TTL before migration; long TTL after for stability |

### DR Runbook

1. **DNS failover**: Point CNAME / A to backup origin via `update-recordset`.
2. **Accidental delete**: No recovery — DNS delete is immediate; restore from backup record set.
3. **DNSSEC failure**: Disable DNSSEC at Huawei Cloud DNS; remove DS from registrar.

## 3. FinOps (财务运营)

### Billing Model

| Item | Approximate CN Rate |
|---|---|
| Public zone (per zone/month) | ~¥0.1 |
| Private zone (per zone/month) | ~¥2.0 |
| Query count (per 10,000 queries) | ~¥0.05 |

### Cost Optimization

| Pattern | Recommendation |
|---|---|
| Idle private zone | VPC deleted but zone remains → delete zone |
| Excessively low TTL | TTL=1s = 86400 DNS queries/day; increase for stable records |
| Public zone for internal use | Switch to private zone (VPC-scoped, no public exposure) |

## 4. SecOps (安全运营)

| Risk | Mitigation |
|---|---|
| DNS hijacking | Enable DNSSEC; rotate NS keys |
| Zone transfer leak | Huawei Cloud DNS blocks AXFR by default |
| Record spoofing | DNSSEC signing; signed zones |

## 5. Operational Efficiency

- **Batch record operations**: Use SDK batch API for >50 record updates.
- **TTL review**: Quarterly review of TTL values; increase stable record TTLs.
- **Zone audit**: Annual review of all zones; delete unused ones.

## Worker Output Contract

> Read-only assessment mode: `{{user.mode}}=well-architected-readonly` → return `{{output.product_assessment}}`.

**Schema:** [worker-output-schema.md](../../huaweicloud-skill-generator/references/worker-output-schema.md)

| Constant | Value |
|----------|-------|
| `skill_id` | `huaweicloud-dns-ops` |
| `product` | `dns` |
| Finding `id` | `dns-{rel\|sec\|cost\|eff}-NNN` |

| `pillars` key | Source sections |
|----------------|-----------------|
| `reliability` | Stability / multi-NS / TTL strategy |
| `security` | DNSSEC / IAM / zone ACL |
| `cost` | FinOps / idle zone / TTL vs CDN cost |
| `efficiency` | Batch record ops / TTL review |

### Example `{{output.product_assessment}}`

```json
{
  "skill_id": "huaweicloud-dns-ops",
  "product": "dns",
  "region": "cn-north-4",
  "scope": "account-wide",
  "assessment_date": "2026-06-24T10:00:00+08:00",
  "status": "PARTIAL",
  "resource_count": 12,
  "partial": false,
  "pillars": {
    "reliability": {"score": 85, "status": "assessed", "findings": []},
    "security": {
      "score": 70,
      "status": "assessed",
      "findings": [
        {
          "id": "dns-sec-001",
          "severity": "Medium",
          "confidence": "HIGH",
          "title": "Public zone example.com without DNSSEC enabled",
          "evidence": "show-zone returned dnssec_status: OFF",
          "recommendation": "Enable DNSSEC on all public zones",
          "effort": "medium"
        }
      ]
    },
    "cost": {"score": 80, "status": "assessed", "findings": []},
    "efficiency": {"score": 75, "status": "assessed", "findings": []}
  },
  "recommendations": [
    {"pillar": "security", "text": "Enable DNSSEC on all public DNS zones"}
  ],
  "trace": {"commands": ["hcloud dns list-zones ..."], "request_ids": []},
  "errors": []
}
```
