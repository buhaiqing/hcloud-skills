# Well-Architected Assessment — BSS (费用中心)

## 1. Five Pillars Overview

| Pillar | Assessment | Key Focus for BSS |
|--------|-----------|-------------------|
| Security | ✅ | AK/SK credential management, BSS API access control, audit logging |
| Stability | ✅ | Read-only by default, budget CRUD confirmation gate, retry logic with backoff |
| Cost | ✅ | PRIMARY — cost visibility, budgets, optimization, closed-loop tracking |
| Efficiency | ✅ | Batch billing queries, paginated resource listing, cached summaries |
| Performance | ✅ | Polling backoff, pagination, end-of-month traffic awareness |

## 2. Detailed Assessments

### 2.1 Security

| Aspect | Status | Implementation |
|--------|--------|----------------|
| Credential isolation | ✅ | `{{env.*}}` never exposed; masked in output |
| IAM minimum permissions | ✅ | Read-only bill view uses `bss:bill:view` only |
| API access control | ✅ | BSS API access controlled via IAM policy |
| Audit trail | ✅ | All budget/package changes logged via CTS |
| Credential rotation | ✅ | AK/SK rotation every 90 days (documented) |

### 2.2 Stability

| Aspect | Status | Implementation |
|--------|--------|----------------|
| Read-only by default | ✅ | All billing query ops are read-only |
| Confirmation gates | ✅ | Budget CRUD, package refund require confirmation |
| Retry strategy | ✅ | Exponential backoff: 2s → 5s → 10s, max 3 retries |
| Error boundaries | ✅ | Bill query errors don't affect other operations |

### 2.3 Cost

| Aspect | Status | Implementation |
|--------|--------|----------------|
| Cost visibility | ✅ | Balance, bill, resource usage, cost analysis (Op 1-5, 8) |
| Budget management | ✅ | Budget alerts with 80/90/100% thresholds (Op 6) |
| Unit economics | ✅ | Cost-per-unit calculation (Op 9) |
| TCO analysis | ✅ | On-demand vs reserved vs spot comparison (Op 10) |
| Optimization mining | 🆕 ✅ | 8-pattern proactive scan (Op 12) |
| Reserved sizing | 🆕 ✅ | Coverage analysis + volatility check (Op 13) |
| Closed-loop tracking | 🆕 ✅ | Apply → measure → validate → regress → improve (Op 14) |

### 2.4 Efficiency

| Aspect | Status | Implementation |
|--------|--------|----------------|
| Batch operations | ✅ | Paginated bill queries via limit/offset |
| Cached summaries | ✅ | Monthly summary pre-computed by BSS |
| Automated reports | ✅ | Scheduled Op 12 scans, Op 14 tracking |

### 2.5 Performance

| Aspect | Status | Implementation |
|--------|--------|----------------|
| Throttle handling | ✅ | Backoff on BSS.0901 (throttled) |
| Pagination | ✅ | 100 records per page by default |
| End-of-month awareness | ✅ | Avoid heavy queries on last 2 days of month (BSS load) |

## 3. FinOps Workflow

### 3.1 FinOps Cycle (Closed-Loop)

```
               ┌─────────────────────┐
               │     DISCOVER        │
               │  (Op 8, 12, 13)     │
               └──────────┬──────────┘
                          │
                          ▼
               ┌─────────────────────┐
               │      MONITOR        │
               │  (Op 6, 11, 14)     │
               └──────────┬──────────┘
                          │
                          ▼
               ┌─────────────────────┐
               │     OPTIMIZE        │
               │   (Op 12 mining)    │
               └──────────┬──────────┘
                          │
                          ▼
               ┌─────────────────────┐
               │       APPLY         │
               │ (delegate to skills) │
               └──────────┬──────────┘
                          │
                          ▼
               ┌─────────────────────┐
               │     MEASURE         │
               │   (Op 14 tracker)   │
               └──────────┬──────────┘
                          │
               ═══════════╪═══════════
               ▲          │           │
               │    ┌─────┴─────┐    │
               │    │ VALIDATED │    │  REGRESSED
               │    │ (≥ 80%)   │    │  (< 50%)
               │    └─────┬─────┘    │
               │          │          │
               │          ▼          ▼
               │     CLOSED    lessons.json
               │                     │
               └─── feedback ──▶ playbook.json
                                      │
                                      ▼
                              DETECTOR UPDATE
```

**NEW — Loop feedback (the arrow that closes the loop):**
When Op 14 REGRESSED entries accumulate, they feed into `regressed_lessons.json` → `playbook.json` → detector pattern tuning. This self-improvement mechanism upgrades the FinOps cycle from a one-way pipeline to a true closed loop.

| Step | Operation | Artifact |
|------|-----------|----------|
| DISCOVER | Op 8, Op 12 | Optimization backlog (patterns P1-P8) |
| MONITOR | Op 11, Op 6 | Anomaly detection, budget alerts |
| OPTIMIZE | Op 12 | Prioritized recommendations |
| APPLY | Op 14 delegate | Executed via product skill |
| MEASURE | Op 14 | 7-day observation window |
| LOOP (Op 14) | Apply → measure → validate/regress → feedback to detector | **The arrow that closes the FinOps loop** |
| IMPROVE | Op 14 + monitoring.md §10 | lessons.json → playbook.json → detector |

### 3.2 FinOps Maturity Progression

| Level | Name | Key Capabilities |
|-------|------|-----------------|
| L1 | Reactive | Manual cost review, no tools |
| L2 | Aware | Budget alerts, bill review |
| L3 | Managed | Anomaly detection, cost-by-tag, monthly FinOps review |
| L4 | Optimized | Proactive mining (Op 12), closed-loop tracker (Op 14), automated backlog |
| L5 | Self-driving | Self-improving detector, lessons → playbook feedback, auto-tuning patterns |

## 4. SecOps Assessment

| Aspect | Status | Implementation |
|--------|--------|----------------|
| Credential masking | ✅ | All AK/SK values masked in output |
| API access control | ✅ | IAM policy for billing: read-only vs admin |
| Data masking | ✅ | Bill data includes resource names but not sensitive content |
| Audit trail | ✅ | CTS traces for budget/resource-package changes |

### IAM Minimum Permissions

| Role | Policy | Operations |
|------|--------|------------|
| Billing Viewer | `bss:bill:view` | Op 1-5 (read-only billing queries) |
| Billing Analyst | `bss:bill:view, bss:budget:view, bss:cost:view` | Op 1-8, 11, 15 (view + analysis) |
| Billing Admin | `bss:*:*` | All Op 1-15 (full access) |
| Optimization Operator | `bss:bill:view, bss:budget:create, bss:package:refund` | Op 6, 7, 12-14 (optimization actions) |

## 5. AIOps Assessment

### 5.1 Anomaly Patterns

| Pattern | Detection Logic | Trigger |
|---------|----------------|---------|
| Cost spike | Current period > 1.5× previous average | Op 11 |
| Cost drop | Current period < 0.5× previous average | Op 11 |
| Budget burn rate | Projected spend > budget × 1.2 | Op 6 + Op 11 |
| Spend by new service | Service appears in billing with no 30d history | Op 3 + Op 11 |

### 5.2 AIOps Maturity

| Level | Name | Anomaly Detection | Self-Healing | Cross-Skill | Proactive |
|-------|------|-------------------|-------------|-------------|-----------|
| L1 | Basic | Manual cost review | ❌ | ❌ | ❌ |
| L2 | Monitored | Budget alerts only | ❌ | ❌ | ❌ |
| L3 | Analyzed | Op 11 spike/drop detection | ❌ | Cost delegation | ❌ |
| L4 | **Optimized** | **✅ Closed-loop cost optimization via Op 14 tracker** | ❌ | Full delegation matrix | **✅ Op 12 mining** |
| L5 | **Self-driving** | **✅ Closed-loop cost optimization with self-improving detector** | **✅ Op 14 REGRESSED → lessons → playbook → detector** | **✅ Auto-delegation** | **✅ All patterns active** |

> **Note:** L4 is achievable when Op 12 (mining) + Op 14 (tracker) are active with manual approval. L5 adds the self-improving feedback loop (regressed_lessons.json → playbook.json enrichment → detector tuning), which requires sustained operation of the closed-loop over multiple cycles.

### 5.3 Cross-Skill Diagnosis

| Scenario | Lead Skill | Supporting Skill |
|----------|-----------|-----------------|
| Cost spike on ECS | huaweicloud-billing-ops | huaweicloud-ecs-ops, huaweicloud-cts-ops |
| Budget exceeded on RDS | huaweicloud-billing-ops | huaweicloud-rds-ops |
| Unexplained bandwidth cost | huaweicloud-billing-ops | huaweicloud-cdn-ops, huaweicloud-vpc-ops |
| Resource package underutilized | huaweicloud-billing-ops | huaweicloud-{product}-ops |

## 6. Well-Architected Score

| Pillar | Score | Notes |
|--------|-------|-------|
| Security | 4/5 | IAM permission granularity could be improved |
| Stability | 4/5 | Some operations depend on BSS API availability |
| Cost | 5/5 | Primary pillar — full coverage |
| Efficiency | 4/5 | Batch operations could be optimized further |
| Performance | 3/5 | BSS API rate limits constrain throughput |

## 7. Maturity Self-Assessment Scorecard

> Referenced by SKILL.md Op 15.

### Scorecard Template

```json
{
  "account_id": "{{env.HW_ACCESS_KEY_ID}}",
  "assessed_at": "2026-06-03T00:00:00Z",
  "overall_level": "L3",
  "scores": {
    "L1_reactive": {"passed": true, "checks": ["manual_review_exists"]},
    "L2_aware": {"passed": true, "checks": ["budget_alerts_active", "monthly_review_active"]},
    "L3_managed": {"passed": true, "checks": ["anomaly_detection_active", "cost_by_tag_active", "finops_review_active"]},
    "L4_optimized": {"passed": false, "checks": ["op12_mining_inactive", "op14_tracker_inactive"]},
    "L5_self_driving": {"passed": false, "checks": ["self_improving_detector_inactive"]}
  },
  "gaps": [
    "Enable Op 12 (Optimization Mining) to reach L4",
    "Enable Op 14 (Closed-Loop Tracker) to reach L4",
    "Run 3+ closed-loop cycles to reach L5"
  ],
  "recommendations": [
    "Configure weekly optimization scan (Op 12) to identify savings opportunities",
    "Enable closed-loop tracking (Op 14) to measure optimization effectiveness",
    "Set up monthly FinOps review with cost-by-tag analysis"
  ]
}
```

### Scoring Criteria

| Level | Criteria | Evidence |
|-------|----------|----------|
| L1 | User has manually reviewed costs at least once | Ask user: have you reviewed costs manually? |
| L2 | Budget alerts exist + monthly review process | Check via Op 6 (ListBudgets) |
| L3 | Op 11 anomaly detection running + cost tags exist | Check if Op 11 configured + cost tags present |
| L4 | Op 12 mining + Op 14 tracker active | Check backlog directory + tracker log |
| L5 | Regressed lessons → playbook feedback working | Check regressed_lessons.json + playbook.json |
---

## Worker Output Contract (Read-Only Assessment Mode)

> Invoked when Well-Architected review sets `{{user.mode}}=well-architected-readonly`.
> Return **`{{output.product_assessment}}`** — field names MUST match the canonical schema.

**Canonical schema:** [worker-output-schema.md](../../huaweicloud-skill-generator/references/worker-output-schema.md)

| Constant | Value |
|----------|-------|
| `skill_id` | `huaweicloud-billing-ops` |
| `product` | `billing` |
| Finding `id` pattern | `billing-{rel|sec|cost|eff}-NNN` |

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
  "skill_id": "huaweicloud-billing-ops",
  "product": "billing",
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
      "hcloud billing read-only-list --region cn-north-4 (HW_SECRET_ACCESS_KEY=<masked>)"
    ],
    "request_ids": [
      "0123456789abcdef0123456789abcdef"
    ]
  },
  "errors": []
}
```
