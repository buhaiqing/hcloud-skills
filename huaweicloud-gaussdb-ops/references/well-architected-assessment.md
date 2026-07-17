# Well-Architected Assessment — huaweicloud-gaussdb-ops

> Product-specific depth: see `huaweicloud-skill-generator/references/well-architected-assessment.md`.

### SLO/SLI Definition — GaussDB

#### SLI (Service Level Indicator) Metrics

| SLI Name | Formula | Data Source | Collection Frequency |
|----------|---------|-------------|---------------------|
| Availability | Successful requests / Total requests × 100% | CES + ELB | 1min |
| Latency P99 | 99th percentile response time (ms) | AOM Trace | 1min |
| Error Rate | 5xx responses / Total requests × 100% | ELB + AOM | 1min |
| Saturation | CPU utilization / Connection utilization / Disk utilization | CES | 5min |

#### SLO Targets

| SLI | SLO Target | Error Budget (Monthly) | Alert Threshold |
|-----|------------|-----------------------|-----------------|
| Availability | ≥ 99.9% | 43.2 min/month | < 99.95% triggers Warning |
| Latency P99 | ≤ 200ms | — | > 300ms triggers Warning |
| Error Rate | ≤ 0.1% | — | > 0.5% triggers Critical |
| Saturation | ≤ 80% | — | > 85% triggers Warning |

#### Error Budget Burn Rate Alerts

| Burn Rate | Consumption Speed | Alert Level | Meaning |
|-----------|------------------|-------------|---------|
| 1× | Normal consumption (43.2 min/month) | — | Normal |
| 2× | 21.6 min exhausted | Info | Attention needed |
| 5× | 8.6 min exhausted | Warning | Intervention needed |
| 14.4× | 3h exhausted | Critical | Immediate action required |


---

## Worker Output Contract (Read-Only Assessment Mode)

> Invoked when Well-Architected review sets `{{user.mode}}=well-architected-readonly`.
> Return **`{{output.product_assessment}}`** — field names MUST match the canonical schema.

**Canonical schema:** [worker-output-schema.md](../../huaweicloud-skill-generator/references/worker-output-schema.md)

| Constant | Value |
|----------|-------|
| `skill_id` | `huaweicloud-gaussdb-ops` |
| `product` | `gaussdb` |
| Finding `id` pattern | `gaussdb-{rel|sec|cost|eff}-NNN` |

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
  "skill_id": "huaweicloud-gaussdb-ops",
  "product": "gaussdb",
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
      "hcloud gaussdb read-only-list --region cn-north-4 (HW_SECRET_ACCESS_KEY=<masked>)"
    ],
    "request_ids": [
      "0123456789abcdef0123456789abcdef"
    ]
  },
  "errors": []
}
```
