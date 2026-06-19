# Worker Output Schema (Product Skill → Orchestrator)

> **Single source of truth** for `{{output.product_assessment}}`. Product
> `references/well-architected-assessment.md` files MUST implement this contract
> in their **Worker Output Contract** section — no alternate field names.

---

## 1. Top-level: `{{output.product_assessment}}`

```json
{
  "skill_id": "huaweicloud-ecs-ops",
  "product": "ecs",
  "region": "cn-north-4",
  "scope": "account-wide",
  "assessment_date": "2026-06-19T10:00:00+08:00",
  "status": "OK",
  "partial": false,
  "resource_count": 12,
  "pillars": {
    "reliability": { "score": 85, "status": "assessed", "findings": [] },
    "security": { "score": 78, "status": "assessed", "findings": [] },
    "cost": { "score": 70, "status": "assessed", "findings": [] },
    "efficiency": { "score": 80, "status": "assessed", "findings": [] }
  },
  "recommendations": [],
  "trace": {
    "commands": ["hcloud ecs list-servers --region cn-north-4 (HW_SECRET_ACCESS_KEY=<masked>)"],
    "request_ids": ["0123456789abcdef0123456789abcdef"]
  },
  "errors": []
}
```

### 1.1 Required fields

| Field | Type | Rules |
|-------|------|-------|
| `skill_id` | string | Worker skill name, e.g. `huaweicloud-ecs-ops` |
| `product` | string | Registry code: `ecs`, `rds`, `vpc`, `obs`, … |
| `region` | string | `{{env.HW_REGION_ID}}` |
| `scope` | string | Echo `{{user.scope}}` |
| `assessment_date` | string | ISO 8601 with timezone |
| `status` | enum | `OK` \| `PARTIAL` \| `ERROR` |
| `partial` | bool | `true` if any pillar `status=not_assessed` |
| `resource_count` | int | ≥ 0; primary resources discovered |
| `pillars` | object | Only keys requested in `{{user.pillars}}` (or all four if `all`) |
| `recommendations` | array | ≥ 0; see §2 |
| `trace` | object | `commands[]` + `request_ids[]`; credentials masked |
| `errors` | array | ≥ 0; see §3 |

### 1.2 `status` (top-level)

| Value | When |
|-------|------|
| `OK` | All requested pillars `assessed` or `skipped` |
| `PARTIAL` | ≥1 pillar `not_assessed` but others succeeded |
| `ERROR` | Discovery failed; no reliable pillar scores |

---

## 2. Pillar object

```json
{
  "score": 85,
  "status": "assessed",
  "findings": []
}
```

| `pillars.*.status` | Meaning |
|--------------------|---------|
| `assessed` | Scored from checklist evidence |
| `not_assessed` | Missing data/API failure — orchestrator must not impute pass |
| `skipped` | Not in `{{user.pillars}}` |

**Scoring:** `score = round(passed_checklist_items / total_applicable_items × 100)`. If `not_assessed`, omit `score` or set `null`.

### 2.1 Finding object (all fields required when present)

| Field | Type | Allowed values |
|-------|------|----------------|
| `id` | string | `{product}-{rel\|sec\|cost\|eff}-NNN` (3-digit seq per pillar) |
| `severity` | string | `Critical` \| `High` \| `Medium` \| `Low` |
| `confidence` | string | `HIGH` \| `MEDIUM` \| `LOW` |
| `title` | string | Short issue label |
| `evidence` | string | Observable fact from Describe* / metrics |
| `recommendation` | string | Actionable fix |
| `effort` | string | `quick` \| `medium` \| `major` |

**ID examples:** `ecs-rel-001`, `rds-sec-002`, `obs-cost-003`

### 2.2 Recommendation object

| Field | Type | Allowed values |
|-------|------|----------------|
| `priority` | string | `Critical` \| `High` \| `Medium` \| `Low` |
| `pillar` | string | `reliability` \| `security` \| `cost` \| `efficiency` |
| `action` | string | Imperative remediation step |
| `effort` | string | `quick` \| `medium` \| `major` |

Include top 1–5 items sorted by priority. May mirror high-severity findings.

---

## 3. Error object (`errors[]`)

| Field | Type | Required |
|-------|------|----------|
| `code` | string | yes — API or worker code |
| `message` | string | yes — sanitized (no credentials) |
| `action` | string | yes — `HALT` \| `RETRY` \| `SKIP` |
| `request_id` | string | no |

---

## 4. Trace object

| Field | Rules |
|-------|-------|
| `commands[]` | Every read-only `hcloud` / SDK call; `HW_SECRET_ACCESS_KEY=<masked>` |
| `request_ids[]` | From `Response.RequestId` (or equivalent) per call |

---

## 5. Changelog

| Version | Date | Change |
|---------|------|--------|
| 1.0.0 | 2026-06-19 | Initial Huawei Cloud worker contract (aligned with qcloud-skills) |
