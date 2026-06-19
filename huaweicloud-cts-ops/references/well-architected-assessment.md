# Huawei Cloud CTS Well-Architected Assessment

## 1. Operational Excellence

- Validate CTS trail lifecycle management with clear create/update/delete flows.
- Provide explicit user confirmation before destructive operations.
- Use strong error messages and remediation guidance for trail delivery issues.

## 2. Security

- Require credentials from environment variables; never prompt for raw secrets.
- Ensure audit trails are delivered to secure storage and access is limited.
- Validate that CTS can only write to approved destinations.
- Support sensitive event filters for compliance and forensic analysis.

## 3. Reliability

- Use post-creation validation and polling for trail activation.
- Retry transient errors, but halt on configuration and authorization failures.
- Monitor trail delivery health and fail fast when destination configuration is invalid.

## 4. Performance Efficiency

- Prefer query filters that narrow the event set and avoid expensive cross-region searches.
- Use `limit` and pagination for large event sets.
- Avoid repeated full-history queries if a narrower time range suffices.

## 5. Cost Optimization

- Recommend retention periods aligned with actual audit requirements.
- Suggest OBS as a cost-efficient storage target for long-term audit archives.
- Avoid creating unnecessary trails; consolidate audit requirements when possible.

## 6. SecOps

- Audit trail existence and query ability are central to security operations.
- Ensure the skill clearly distinguishes between active trails and deleted trails.
- Provide analysis paths for suspicious access and unauthorized operations.

## 7. AIOps

- Use event query results as structured evidence for automated incident analysis.
- Support follow-up investigative actions when query patterns indicate anomalies.
- Integrate with log and metric skills for broader anomaly correlation.
---

## Worker Output Contract (Read-Only Assessment Mode)

> Invoked when Well-Architected review sets `{{user.mode}}=well-architected-readonly`.
> Return **`{{output.product_assessment}}`** — field names MUST match the canonical schema.

**Canonical schema:** [worker-output-schema.md](../../huaweicloud-skill-generator/references/worker-output-schema.md)

| Constant | Value |
|----------|-------|
| `skill_id` | `huaweicloud-cts-ops` |
| `product` | `cts` |
| Finding `id` pattern | `cts-{rel|sec|cost|eff}-NNN` |

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
  "skill_id": "huaweicloud-cts-ops",
  "product": "cts",
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
      "hcloud cts read-only-list --region cn-north-4 (HW_SECRET_ACCESS_KEY=<masked>)"
    ],
    "request_ids": [
      "0123456789abcdef0123456789abcdef"
    ]
  },
  "errors": []
}
```
