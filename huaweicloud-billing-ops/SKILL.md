---
name: huaweicloud-billing-ops
description: >-
  Use when the user needs to manage Huawei Cloud Billing & Cost (费用中心 / BSS) —
  account balance, customer bills, bill details, resource usage, monthly summaries,
  budgets, cost analysis, and resource packages. User mentions 账单, 费用, 成本,
  预算, 充值, 余额, 消费, 月度账单, 账单明细, 资源账单, 成本中心, BSS, Billing,
  Budget, Cost, FinOps, or describes cost-related scenarios (e.g., unexpected high
  cost, set monthly budget, analyze spend by service, refund a resource package,
  view pay-per-use charges) even without naming the product directly. Not for IAM,
  resource provisioning, or other products that have their own ops skills.
license: MIT
compatibility: >-
  Official Huawei Cloud CLI (`hcloud`), Go 1.21+ runtime (for JIT SDK fallback
  via huaweicloud-sdk-go-v3), valid API credentials, network access to Huawei
  Cloud BSS endpoint. International (international.huaweicloud.com) and
  China domestic (huaweicloud.com) endpoints differ — set HW_BSS_ENDPOINT
  explicitly for non-default domains.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-06-03"
  runtime: Harness AI Agent, Claude Code, Cursor, or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
  gcl:
    enabled: true
    required: false
    rubric_version: "v1"
    max_iter: 5
    rubric_ref: "references/rubric.md"
    prompts_ref: "references/prompt-templates.md"
    trace_dir: "./audit-results/"
    changelog:
      - version: "1.1.0"
        date: "2026-06-04"
        change: "GCL Phase 3 rollout: added references/rubric.md (v1, 5-dim, S1–S7 BSS-specific Safety rules, including budget-delete-without-confirmation / refund-without-cost-impact / quota-exceeded-silent-fail / credential-leak guards) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
---

# huaweicloud-billing-ops — 华为云费用中心 (BSS) 运维 Skill

## Five Core Standards (五大核心标准)

| # | Standard | How This Skill Satisfies It |
|---|----------|----------------------------|
| 1 | **Clear Boundaries** (边界明确) | SHOULD Use / SHOULD NOT Use with explicit trigger keywords; cross-skill delegation for physical operations (stop, resize, delete) |
| 2 | **Structured I/O** (输入输出结构化) | `{{env.*}}` for credentials, `{{user.*}}` for interactive params, `{{output.*}}` for API captures |
| 3 | **Explicit Actionable Steps** (步骤明确可执行) | Every operation has numbered steps: pre-flight → execute → validate → recover |
| 4 | **Complete Failure Strategies** (失败策略完备) | Error taxonomy with product-specific BSS error codes (≥ 15), retry vs HALT distinction |
| 5 | **Absolute Single Responsibility** (职责绝对单一) | This skill covers BSS cost/billing visibility ONLY. Physical resource ops delegated to product skills |

## Three-Pillar Ops (三支柱运维)

| Pillar | Integration | Reference |
|--------|------------|-----------|
| **FinOps (财务运营)** | PRIMARY — cost visibility, budgeting, anomaly detection, unit economics, TCO, optimization mining, closed-loop tracking | `references/well-architected-assessment.md` §3 |
| **SecOps (安全运营)** | Cost of security incidents; BSS API access control; AK/SK scoping for cost data | `references/well-architected-assessment.md` §4 |
| **AIOps (智能运营)** | Cost anomaly detection (spike/dip), budget burn rate, optimization mining, closed-loop self-improvement | `references/well-architected-assessment.md` §5 |

## Capabilities at a Glance

| # | Operation | Description | Triggered When |
|---|-----------|-------------|----------------|
| 1 | Account Balance | Query account balance, currency info | "余额多少", "查余额" |
| 2 | Monthly Bill | Get monthly bill summary | "查看本月账单", "上月消费" |
| 3 | Bill Detail | Resource-level bill line items | "看账单明细", "按资源查询消费" |
| 4 | Resource Usage | Usage by service/resource | "资源使用情况", "用量查询" |
| 5 | Monthly Summary | Summarize by service/region/project | "按产品汇总费用" |
| 6 | Budget Alert | Create/update/delete budget alerts | "设预算", "预算告警" |
| 7 | Resource Package | List/refund resource packages | "资源包", "查看套餐" |
| 8 | Cost Analysis | Multi-dimension cost breakdown | "成本分析", "按标签分摊" |
| 9 | Unit Economics | Cost per unit (user/order/store) | "单位成本", "每用户成本" |
| 10 | TCO Comparison | On-demand vs reserved vs spot | "哪种划算", "比价" |
| 11 | Anomaly Detection | Spike/drop/burn-rate detection | "费用异常", "突然涨价" |
| 12 | 🆕 Optimization Mining | Scan bill detail → identify 8 optimization patterns | "哪些可以优化", "找闲置" |
| 13 | 🆕 Reserved Capacity Sizing | Coverage analysis → buy recommendations | "该不该买包年", "预留建议" |
| 14 | 🆕 Closed-Loop Optimization Tracker | Lifecycle: detect→approve→apply→measure→validate | "跟踪优化", "闭环报告" |
| 15 | 🆕 Maturity Self-Assessment | L1–L5 FinOps maturity evaluation | "FinOps水平", "成熟度评估" |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User asks about **account balance**: "还剩多少钱", "账户余额"
- User asks about **bills**: "本月账单", "上月消费明细", "历史费用"
- User asks about **budgets**: "设置预算", "预算超标了怎么办"
- User asks about **cost analysis**: "按产品分析成本", "成本趋势"
- User asks about **resource packages**: "查看资源包", "续费套餐"
- User asks about **optimization**: "哪些资源可以优化", "帮我省钱"
- User asks about **anomalies**: "费用为什么涨了", "异常检测"
- User asks about **reserved capacity**: "包年包月推荐", "预留实例建议"
- User mentions **FinOps** or **finops** in any context
- User asks "how much did X cost?" even without naming BSS

### SHOULD NOT Use This Skill When

- User asks about **IAM** operations (user/group/policy) → delegate to `huaweicloud-iam-ops`
- User asks to **provision/modify physical resources** (ECS, RDS, OBS) → delegate to product skill
- User asks about **non-billing configurations** like quotas, regions, AZs → delegate to respective product skill
- User asks about **monitoring metrics** (CPU, memory, disk) → delegate to `huaweicloud-ces-ops`
- User asks to **stop/start/delete** resources → delegate to product skill (this skill provides cost analysis only)
- User asks purely technical questions unrelated to cost

## Variables

| Variable | Source | Description | Example |
|----------|--------|-------------|---------|
| `{{env.HW_ACCESS_KEY_ID}}` | Environment | Huawei Cloud AK | `AKID1...` |
| `{{env.HW_SECRET_ACCESS_KEY}}` | Environment | Huawei Cloud SK | `<masked>` |
| `{{env.HW_REGION_ID}}` | Environment | Region | `cn-north-4` |
| `{{env.HW_PROJECT_ID}}` | Environment | Project | `proj1...` |
| `{{env.HW_BSS_ENDPOINT}}` | Environment | BSS endpoint override | `bss.cn-north-4.myhuaweicloud.com` |
| `{{user.cycle}}` | User input | Billing cycle | `2026-05` |
| `{{user.budget_amount}}` | User input | Budget threshold (CNY) | `10000` |
| `{{user.budget_name}}` | User input | Budget name | `"生产环境月度预算"` |
| `{{user.package_id}}` | User input | Resource package ID | `RSCPKG-12345` |
| `{{user.optimization_id}}` | User input | Optimization item ID | `OPT-20260603-001` |
| `{{output.balance}}` | API Response | Current balance amount | `1250.50` |
| `{{output.bill_total}}` | API Response | Total bill amount | `8520.00` |

## Operations

### Operation 1: Account Balance (账户余额查询)

> Query the current account balance, currency, and credit limit.

#### Pre-flight

- [ ] `{{env.HW_ACCESS_KEY_ID}}` and `{{env.HW_SECRET_ACCESS_KEY}}` must be set
- [ ] Network access to BSS endpoint verified

#### Execution

```bash
# Query account balance
hcloud BSS ShowCustomerAccountInfo
```

**Expected output (JSON):**
```json
{
  "account_balances": [
    {
      "amount": "1250.50",
      "currency": "CNY",
      "credit_amount": "0.00",
      "designated_amount": "0.00"
    }
  ]
}
```

#### Validation

- [ ] Response contains `account_balances` array
- [ ] `amount` is a valid number string
- [ ] `currency` matches expected (CNY/USD)

#### Recovery

| Error | Action |
|-------|--------|
| `BSS.0002` Auth failure | Check AK/SK credentials |
| `BSS.0100` Network error | Retry with backoff (2s, 5s, 10s) |
| Empty response | Account may have no balance info; return zero |

### Operation 2: Monthly Bill (月度账单)

> Get the monthly bill summary for a given billing cycle.

#### Pre-flight

- [ ] User provides billing cycle `{{user.cycle}}` (format: YYYY-MM)
- [ ] If not provided, default to current month

#### Execution

```bash
# Query monthly bill summary
hcloud BSS ListCustomerselfResourceRecords --cycle="{{user.cycle}}" --method="SUMMARY"
```

**Expected output (JSON):**
```json
{
  "records": [
    {
      "cycle": "2026-05",
      "bill_type": 1,
      "customer_id": "custo...",
      "currency": "CNY",
      "consumption": "8520.00",
      "balance": "1250.50",
      "cash_amount": "0.00",
      "debt_amount": "0.00",
      "measure_id": 1
    }
  ]
}
```

#### Validation

- [ ] Records array non-empty
- [ ] `cycle` matches requested cycle
- [ ] `consumption` parsed as valid decimal

#### Recovery

| Error | Action |
|-------|--------|
| `BSS.0003` Invalid cycle | Ask user for correct format (YYYY-MM) |
| Timeout > 30s | Retry with pagination (reduce month range) |
| `BSS.0101` No data | Return "No bill data for this cycle" |

### Operation 3: Bill Detail (账单明细)

> Resource-level bill line items with pagination.

#### Pre-flight

- [ ] Billing cycle specified
- [ ] Optional filters: service type, region, enterprise project

#### Execution

```bash
hcloud BSS ListCustomerselfResourceRecords --cycle="{{user.cycle}}" \
  --method="DETAIL" --limit=100
```

#### Validation

- [ ] Records present with resource-level breakdown
- [ ] Each record has: resource_id, resource_name, service_type, amount

#### Recovery

| Error | Action |
|-------|--------|
| Large result set | Paginate with `offset` parameter |
| `BSS.0101` No records | Inform user no detailed records for this cycle |

### Operation 4: Resource Usage (资源使用量)

> Query resource usage by service or resource.

#### Pre-flight

- [ ] User specifies time range or cycle

#### Execution

```bash
hcloud BSS ListCustomerResourceUsage --cycle="{{user.cycle}}"
```

### Operation 5: Monthly Summary (月度汇总)

> Summarize costs by service type, region, or enterprise project.

#### Execution

```bash
hcloud BSS ListMonthlyExpenditures --cycle="{{user.cycle}}"
```

### Operation 6: Budget Alert (预算告警)

> Create, update, list, or delete budget alerts with threshold levels.

#### Pre-flight

- [ ] User specifies budget amount `{{user.budget_amount}}` and name `{{user.budget_name}}`
- [ ] Default thresholds: 80%, 90%, 100%

#### Execution

```bash
# Create budget alert
hcloud BSS CreateBudget --name="{{user.budget_name}}" \
  --amount="{{user.budget_amount}}" \
  --thresholds="80,90,100" \
  --notify-by="email,sms"
```

#### Validation

- [ ] Budget created successfully (check response status)
- [ ] Threshold values are valid percentages

#### Recovery

| Error | Action |
|-------|--------|
| `BSS.0201` Budget name exists | Suggest different name or update existing |
| `BSS.0202` Invalid amount | Amount must be > 0; ask for correction |

### Operation 7: Resource Package (资源包)

> List resource packages, check remaining balance, refund if applicable.

#### Execution

```bash
# List resource packages
hcloud BSS ListResourcePackages --status="active"
# Refund a resource package
hcloud BSS RefundResourcePackage --package-id="{{user.package_id}}"
```

#### Safety Gate

- ⚠️ Refund is destructive (package is consumed immediately)
- MUST confirm with user: `"确认退订资源包 {{user.package_id}}？退订后不可恢复。"`
- Only proceed on explicit "yes"

### Operation 8: Cost Analysis (成本分析)

> Multi-dimension cost breakdown by time, service, project, or tag.

#### Execution

```bash
# Cost analysis by service type
hcloud BSS ListCosts --time-range="LAST_30_DAYS" --group-by="service_type"
# By enterprise project
hcloud BSS ListCosts --time-range="THIS_MONTH" --group-by="enterprise_project"
# By tag
hcloud BSS ListCosts --time-range="LAST_MONTH" --group-by="tag:CostCenter"
```

### Operation 9: Unit Economics (单位成本)

> Calculate cost per unit (per user, per order, per store). Requires user-provided denominator.

#### Pre-flight

- [ ] User provides `{{user.denominator}}` (e.g., "5000 users", "200 orders")
- [ ] Total cost fetched via Op 8

#### Calculation

```
Unit Cost = Total Cost / Denominator
```

#### Example

> "The total cost for May 2026 was ¥8,520. With 5,000 active users, the cost per user is ¥1.70."

### Operation 10: TCO Comparison (总拥有成本对比)

> Compare on-demand vs reserved vs spot pricing.

#### Execution

```bash
# Get pricing for on-demand
hcloud BSS ListResourceUsage --cycle="{{user.cycle}}"
# Estimate reserved savings: ~30% discount for 1y, ~50% for 3y
# Estimate spot savings: ~60-80% compared to on-demand
```

#### Output

| Model | Estimated Monthly Cost | Savings vs On-Demand |
|-------|----------------------|---------------------|
| On-demand | ¥8,520 | — |
| Reserved 1y | ¥5,964 | 30% |
| Reserved 3y | ¥4,260 | 50% |
| Spot | ¥1,704 | 80% |

### Operation 11: Anomaly Detection (异常检测)

> Detect cost anomalies: spikes, drops, and burn-rate changes.

#### Detection

- **Spike**: Current period cost > 1.5× previous period average
- **Drop**: Current period cost < 0.5× previous period (may indicate resource removal)
- **Burn rate**: Monthly projected spend > budget × 1.2

#### Execution

```bash
# Get costs for last 3 periods
hcloud BSS ListCosts --time-range="LAST_90_DAYS" --interval="MONTHLY"
# Compare current vs average of previous 2 periods
```

#### Example Output

```
⚠️ Cost Anomaly Detected (May 2026):
- Current: ¥8,520
- Average (Mar-Apr): ¥5,100
- Change: +67% (SPIKE)
- Possible cause: New deployment of ecs-web-cluster (3 instances)
- Action: Review Op 12 (Optimization Mining) for detail
```

---

### Operation 12: Optimization Mining (主动优化机会挖掘)

> 🆕 **NEW — depth-of-insight upgrade.** Scans last 30/60/90 days of billing and usage data to surface optimization opportunities. Outputs a prioritized backlog with estimated savings.

> 📖 **Cross-refs** — backlog storage layout, JSON schema, prioritization formula, and aging policy: `references/monitoring.md` §9. AIOps L4 self-driving detector upgrades: `references/well-architected-assessment.md` §3.1 (FinOps Workflow) and §5.2 (AIOps Maturity).

#### When to Use

- "where can I cut cost?" / "有哪些可以优化的地方？"
- "am I paying for things I don't need?"
- "should I buy reserved capacity?"
- Scheduled weekly FinOps cron run
- Post-deploy verification (3-7 days after major deploy)

#### Mining Patterns (8 Categories)

| # | Pattern | Trigger | Action |
|---|---------|---------|--------|
| P1 | Idle/Orphaned | utilization < 5% > 7d | Stop/Release via product skill |
| P2 | Right-size | CPU < 20% or > 80% | Resize via product skill |
| P3 | Reserved Opportunity | on-demand > 30% for 30d | Buy reserved (Op 13) |
| P4 | Package Waste | utilization < 40% | Downsize/refund |
| P5 | Storage Tiering | data untouched > 90d | Archive via obs-ops |
| P6 | Log Retention | logs > 30d | Prune via lts-ops |
| P7 | Idle Package | balance unused > 70% | Refund (Op 7) |
| P8 | Zombie Resources | no billing activity > 30d | Review/release |

#### Prioritization Formula

```
priority_score = estimated_savings × confidence × ease_factor

ease_factor:
  1.0 = fully automated (refund, stop)
  0.7 = partially automated (resize, tier)
  0.4 = manual (architecture change)
```

#### Validation

- [ ] Each pattern yields at least one actionable recommendation
- [ ] Estimated savings include calculation basis
- [ ] Recommendations tagged with confidence (HIGH/MEDIUM/LOW)

---

### Operation 13: Reserved Capacity Sizing (预留容量规划)

> 🆕 **NEW — reserved capacity recommendations.** Analyzes on-demand spend patterns to recommend reserved instance/capacity purchases.

#### Coverage Analysis

```bash
# Get on-demand spend by service for last 30 days
hcloud BSS ListCosts --time-range="LAST_30_DAYS" --group-by="service_type"
```

#### Volatility Check

- Compute coefficient of variation (CV) of monthly spend
- CV > 0.30 → Flag as unpredictable → skip reserved recommendation
- CV ≤ 0.30 → Recommend reserved with confidence

#### Recommendation

| Coverage Gap | Recommended Term | Est. Discount | Payback Period |
|-------------|-----------------|---------------|----------------|
| > 40% on-demand | 1-year reserved | ~30% | 8-10 months |
| > 60% on-demand | 3-year reserved | ~50% | 18-24 months |
| 20-40% on-demand | Partial reserved (50% coverage) | ~30% | 10-12 months |

#### Validation

- [ ] CV calculated from ≥ 3 months of data
- [ ] Recommendation includes projected savings
- [ ] User presented with buy-vs-wait tradeoff

---

### Operation 14: Closed-Loop Optimization Tracker (闭环优化跟踪)

> 🆕 **NEW — closes the loop.** Tracks the full lifecycle of optimization items from detection → approval → application → measurement → validation.

> 📖 **Cross-refs** — full 9-state lifecycle, JSONL log schema, loop SLOs, CI/CD integration: `references/monitoring.md` §10. Lessons → detector feedback arrow: `references/well-architected-assessment.md` §3.1.

#### Lifecycle

| From | To | Trigger |
|------|----|---------|
| DETECTED | PENDING_REVIEW | Auto (Op 12/13) |
| PENDING_REVIEW | APPROVED | User confirms |
| APPROVED | APPLIED | Execute via product skill |
| APPLIED | MEASURING | Auto (7-day window) |
| MEASURING | VALIDATED | savings ≥ 80% |
| MEASURING | REGRESSED | savings < 50% or cost ↑ |
| REGRESSED | ROLLED_BACK | Auto (< 2h) |
| VALIDATED/ROLLED_BACK | ARCHIVED | Auto |

#### Log Format (JSONL)

File: `~/.hcloud/optimization_tracker.jsonl`

```json
{"id":"OPT-20260603-001","pattern":"P1_idle_ecs","state":"DETECTED","resource":"i-abc123","est_savings":1200.00,"confidence":"HIGH","detected_at":"2026-06-03T10:00:00Z"}
{"id":"OPT-20260603-001","pattern":"P1_idle_ecs","state":"APPROVED","approved_at":"2026-06-03T14:30:00Z","approved_by":"user"}
{"id":"OPT-20260603-001","pattern":"P1_idle_ecs","state":"APPLIED","applied_via":"huaweicloud-ecs-ops","actual_savings":1350.00,"applied_at":"2026-06-03T15:00:00Z"}
```

#### Loop SLOs

| Transition | P50 | P99 |
|-----------|-----|-----|
| DETECTED → APPROVED | < 24h | < 72h |
| APPROVED → APPLIED | < 4h | < 24h |
| MEASURING duration | 7 days | 14 days |
| REGRESSED → ROLLED_BACK | < 2h | < 8h |

#### Validation

- [ ] Log file created and append-only
- [ ] Each state transition is timestamped
- [ ] APPLIED entries record `applied_via` product skill
- [ ] State machine prevents invalid transitions (no APPROVED → DETECTED)

---

### Operation 15: Maturity Self-Assessment (FinOps 成熟度自评)

> 🆕 **NEW — FinOps maturity evaluation.** Assesses current FinOps practices against a 5-level maturity model.

> 📖 **Cross-ref** — full scorecard: `references/well-architected-assessment.md` §7.

#### Maturity Levels

| Level | Name | Characteristics |
|-------|------|----------------|
| L1 | Reactive | Manual cost review, no budgets, no alerts |
| L2 | Aware | Basic budget alerts, monthly bill review, some tagging |
| L3 | Managed | Op 11 anomaly detection active, Op 6 budget alerts with escalation, cost-by-tag analysis |
| L4 | Optimized | Op 12 optimization mining + Op 14 tracker active; automated backlog → approval pipeline |
| L5 | Self-driving | Closed loop with self-improving detector; Op 14 REGRESSED feeds back to lessons.json → playbook.json → detector tuning |

#### Self-Assessment Flow

1. For each level, check the criteria against current account configuration
2. Account scores at the highest level where ALL criteria pass
3. Scorecard saved to `~/.hcloud/maturity_scorecard.json`

#### Validation

- [ ] Each level scored independently
- [ ] Gap analysis output: "You are at L{n}; to reach L{n+1}, enable: [...]"
- [ ] Scorecard persisted for trend tracking

---

## Prompt Examples (常见用户提问示例)

| Category | Example Query | Expected Op |
|----------|--------------|-------------|
| A Bill/Account | "查看账户余额" / "本月消费多少" | Op 1, 2, 3 |
| B Budget | "设置预算¥10,000" / "超80%发短信" | Op 6 |
| C Cost Analysis | "按产品分析成本" / "按项目拆分费用" | Op 8 |
| D Optimization | "哪里可以省钱" / "找出闲置资源" | Op 12 |
| E Reserved | "该不该买包年" / "预留覆盖率" | Op 10, 13 |
| F Resource Package | "查看资源包" / "退订未使用的" | Op 7 |
| G Anomaly | "费用突然涨了" / "检测异常波动" | Op 11 |
| H Reporting | "FinOps报告" / "成本对账单" | Op 5, 8 |
| I Cross-Skill | "ECS哪些能降配" / "存储费用太高" | Delegate |
| J Maturity | "FinOps成熟度评估" / "改进路线图" | Op 15 |

## Cross-Skill Delegation Patterns

| Intent | This Skill Detects | Delegates To | Pattern |
|--------|-------------------|--------------|---------|
| Stop idle resource | Idle ECS (Op 12 P1) | huaweicloud-ecs-ops | P1 |
| Delete orphaned disk | Orphaned EVS (Op 12 P1) | huaweicloud-ecs-ops | P1 |
| Resize ECS | Over/under-provisioned (Op 12 P2) | huaweicloud-ecs-ops | P2 |
| Resize RDS | Under-provisioned (Op 12 P2) | huaweicloud-rds-ops | P2 |
| Buy reserved | Coverage gap (Op 13) | — (recommendation only) | P3 |
| Archive storage | Old data (Op 12 P5) | huaweicloud-obs-ops | P5 |
| Prune logs | Old logs (Op 12 P6) | huaweicloud-lts-ops | P6 |
| Refund package | Idle package (Op 12 P7) | self (Op 7) | P7 |
| Find zombies | Inactive resources (Op 12 P8) | huaweicloud-{product}-ops | P8 |
| Cost drill-down | High service cost (Op 8) | huaweicloud-{product}-ops | C |
| Container optimization | Cluster cost (Op 8+12) | huaweicloud-cce-ops | D |
| Full FinOps report | All ops combined | Cross-skill aggregation | H |

## Quality Gate (GCL)

This skill is **GCL-optional** (per `AGENTS.md` §8). Most BSS operations are read-only (bill queries, cost analysis) and GCL-**exempt**. Mutating operations — budget alert create / update / delete, resource package refund — run through the **Generator-Critic-Loop** before their result is returned.

| Field | Value |
|-------|-------|
| Rubric version | v1 (Phase 3, 2026-06-04) |
| `max_iter` | **5** |
| Rubric instance | [`references/rubric.md`](references/rubric.md) |
| Prompt templates | [`references/prompt-templates.md`](references/prompt-templates.md) |
| Trace path | `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` |
| Independence | Generator and Critic in **isolated** sub-agent / session contexts |

### Five-Dimension Rubric (summary)

> The five-dimension rubric (Correctness / Safety / Idempotency / Traceability / Spec Compliance),
> default thresholds, termination contract (PASS / MAX_ITER / SAFETY_FAIL), and trace-persistence
> rules are defined in [`docs/gcl-spec.md`](../../docs/gcl-spec.md) and the repo root
> [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8. This skill overrides only the items below.

| # | Dimension | Threshold | Notes |
|---|-----------|-----------|-------|
| 1 | Correctness | ≥ 0.5 | Bill data matches query params / budget state matches request |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | S1–S7 in rubric §2 |
| 3 | Idempotency | ≥ 0.5 | Read queries are trivially idempotent; create check before budget creation |
| 4 | Traceability | ≥ 0.5 | Credential MUST be `<masked>` |
| 5 | Spec Compliance | ≥ 0.5 | Billing model / budget threshold / time range / currency |

### Per-Operation Safety Anchors (binding)

- **S1** — `delete-budget` without explicit user confirmation quoting the budget name
- **S2** — `delete-budget` when it is the only budget alert for a cost center
- **S3** — `update-budget` reducing threshold below current spend without warning (immediate alarm)
- **S4** — `refund-package` without calculating remaining value and cost impact
- **S5** — `create-budget` with threshold = 0% (trigger-immediately, alarm storm)
- **S6** — any trace contains `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / password plaintext
- **S7** — any operation that silently fails due to billing account quota/status without user-visible error

### See also

- [`references/rubric.md`](references/rubric.md) — full rubric, S1–S7 rules, per-op thresholds
- [`references/prompt-templates.md`](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons
- Repository root [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8 — GCL specification

## References

- [references/core-concepts.md](references/core-concepts.md) — BSS architecture, billing models, quotas, endpoints
- [references/api-sdk-usage.md](references/api-sdk-usage.md) — API operation map, pagination, request/response snippets
- [references/cli-usage.md](references/cli-usage.md) — CLI command map, coverage gap table, JSON output paths
- [references/troubleshooting.md](references/troubleshooting.md) — Error code taxonomy (≥ 15 codes), diagnostic flows
- [references/monitoring.md](references/monitoring.md) — BSS observability, budget burn rate, optimization backlog (§9), closed-loop workflow (§10)
- [references/integration.md](references/integration.md) — JIT SDK bootstrap, env vars, cross-skill delegation matrix
- [references/well-architected-assessment.md](references/well-architected-assessment.md) — Five pillars, FinOps workflow (§3.1), AIOps maturity (§5.2), maturity scorecard (§7)
- [references/knowledge-base.md](references/knowledge-base.md) — Common billing fault patterns
- [references/rubric.md](references/rubric.md) — GCL rubric (v1, 5-dim, S1–S7 BSS-specific Safety rules)
- [references/prompt-templates.md](references/prompt-templates.md) — GCL Generator / Critic / Orchestrator skeletons