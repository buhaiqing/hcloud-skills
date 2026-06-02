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

```
┌──────────────────────────────────────────────────────────────────────┐
│  OPTIMIZATION MINING — 8 PATTERNS                                     │
├──────────────────────────────────────────────────────────────────────┤
│  P1  Idle / Orphaned Resources  (持续 < 5% 利用率 > 7 天)            │
│      → Stop / Release via huaweicloud-{product}-ops                   │
│                                                                      │
│  P2  Right-size (over/under-provisioned)  (CPU < 20% or > 80%)      │
│      → Resize via huaweicloud-{product}-ops                           │
│                                                                      │
│  P3  Reserved Instance Opportunity  (on-demand > 30% for 30d)       │
│      → Buy reserved via Op 13                                         │
│                                                                      │
│  P4  Resource Package Waste  (utilization < 40%)                     │
│      → Downsize / refund package                                      │
│                                                                      │
│  P5  Storage Tiering  (data not accessed > 90 days)                  │
│      → Archive / lifecycle policy via huaweicloud-obs-ops             │
│                                                                      │
│  P6  Log Retention  (logs > 30 days)                                  │
│      → Prune / archive via huaweicloud-lts-ops                        │
│                                                                      │
│  P7  Idle Resource Package  (balance unused > 70%)                   │
│      → Refund via Op 7                                                │
│                                                                      │
│  P8  Zombie Resources  (no billing activity > 30 days)               │
│      → Review / release via huaweicloud-{product}-ops                 │
└──────────────────────────────────────────────────────────────────────┘
```

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

```
DETECTED (Op 12/13) ──▶ PENDING_REVIEW ──▶ APPROVED
                                               │
                      ┌────────────────────────┘
                      ▼
                   APPLIED (via product skill)
                      │
                      ▼
                   MEASURING
                      │
                 ┌────┴────┐
                 ▼         ▼
            VALIDATED  REGRESSED
            (savings   (savings < 50%
             ≥ 80%)     or cost ↑)
                 │         │
                 ▼         ▼
              CLOSED   ROLLED_BACK
                 │         │
                 └──┬──────┘
                    ▼
                ARCHIVED
```

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

### A. Bill & Account (账单/余额)

| # | Query (中文) | Query (English) | Expected Op |
|---|-------------|-----------------|-------------|
| 1 | "查看我的账户余额" | "Check my account balance" | Op 1 |
| 2 | "查一下本月消费了多少" | "How much did I spend this month?" | Op 2 |
| 3 | "上个月的账单明细给我看看" | "Show last month's bill details" | Op 3 |
| 4 | "查一下账户上的欠款情况" | "Check any outstanding debts" | Op 1 |
| 5 | "最近三个月的费用趋势" | "Cost trend for last 3 months" | Op 5 |

### B. Budget & Alerts (预算/告警)

| # | Query | Expected Op |
|---|-------|-------------|
| 1 | "帮我设置本月预算¥10,000" | Op 6 |
| 2 | "当预算超过80%时发短信通知" | Op 6 |
| 3 | "更新之前设定的预算金额" | Op 6 |
| 4 | "查看当前的预算执行情况" | Op 6 |
| 5 | "删除预算'生产环境月度预算'" | Op 6 |
| 6 | "列出所有预算告警规则" | Op 6 |

### C. Cost Analysis (成本分析)

| # | Query | Expected Op |
|---|-------|-------------|
| 1 | "按产品分析最近一个月的成本" | Op 8 |
| 2 | "按企业项目拆分费用" | Op 8 |
| 3 | "按标签分析成本分布" | Op 8 |
| 4 | "对比本月和上月的费用差异" | Op 8 + Op 11 |
| 5 | "分析华东区域的费用构成" | Op 8 |
| 6 | "Top 10 最贵的服务排行" | Op 8 |

### D. Optimization & Savings (优化/省钱)

| # | Query | Expected Op |
|---|-------|-------------|
| 1 | "帮我看一下有哪些地方可以省钱" | Op 12 |
| 2 | "找出所有闲置资源" | Op 12 |
| 3 | "分析存储成本优化机会" | Op 12 (P5) |
| 4 | "哪些实例利用率太低需要降配" | Op 12 (P2) |
| 5 | "有没有僵尸资源在白白扣费" | Op 12 (P8) |
| 6 | "日志超过30天的能删掉吗" | Op 12 (P6) |

### E. Reserved Capacity (包年包月/预留)

| # | Query | Expected Op |
|---|-------|-------------|
| 1 | "哪些按需实例应该转包年包月？" | Op 13 |
| 2 | "买3年预留实例划算还是1年？" | Op 10 + Op 13 |
| 3 | "预留覆盖率建议" | Op 13 |
| 4 | "按量和包年包月哪个更省钱" | Op 10 |
| 5 | "预留实例的折扣是多少？" | Op 10 |

### F. Resource Package (资源包)

| # | Query | Expected Op |
|---|-------|-------------|
| 1 | "查看剩余资源包" | Op 7 |
| 2 | "资源包快要到期了，帮我续费" | Op 7 |
| 3 | "有哪些资源包可以退订" | Op 7 |
| 4 | "退订未使用的资源包" | Op 7 |
| 5 | "资源包使用率报告" | Op 7 |

### G. Anomaly & Alerts (异常/告警)

| # | Query | Expected Op |
|---|-------|-------------|
| 1 | "为什么这个月的费用突然涨了？" | Op 11 |
| 2 | "帮我检测成本异常波动" | Op 11 |
| 3 | "检查预算执行率是否异常" | Op 6 + Op 11 |
| 4 | "过去30天有没有费用突增？" | Op 11 |
| 5 | "对比上个月的费用变化" | Op 11 |
| 6 | "预警：本月预算已消耗85%" | Op 6 |

### H. FinOps Reporting (报告/对账)

| # | Query | Expected Op |
|---|-------|-------------|
| 1 | "生成上个月的FinOps报告" | Op 5 + Op 8 |
| 2 | "按成本中心的费用对账单" | Op 8 |
| 3 | "各BU的成本分摊报表" | Op 8 |
| 4 | "本月优化收益汇总" | Op 14 |
| 5 | "查看闭环跟踪报告" | Op 14 |

### I. Cross-Skill Cost Ops (跨技能)

| # | Query | Delegate To |
|---|-------|-------------|
| 1 | "帮我看看哪些ECS可以降配" | P2 → huaweicloud-ecs-ops |
| 2 | "分析RDS使用率并优化" | P2 → huaweicloud-rds-ops |
| 3 | "CDN流量包快用完了" | P4 → huaweicloud-cdn-ops |
| 4 | "存储费用太高了，怎么办" | P5 → huaweicloud-obs-ops |
| 5 | "日志太多成本太高" | P6 → huaweicloud-lts-ops |
| 6 | "ECS实例空闲太多" | P1 → huaweicloud-ecs-ops |

### J. Maturity & Guidance (成熟度/指导)

| # | Query | Expected Op |
|---|-------|-------------|
| 1 | "评估我的FinOps成熟度" | Op 15 |
| 2 | "成本管理做到什么水平了？" | Op 15 |
| 3 | "L4成熟度需要什么条件？" | Op 15 |
| 4 | "给一份FinOps改进路线图" | Op 15 |
| 5 | "怎么从L2升到L3？" | Op 15 |

### Prompt Intent Map

| Category | Intent | Example | Expected Op |
|----------|--------|---------|-------------|
| A | Query balance/credit | "查余额" | Op 1 |
| A | Query monthly spending | "本月花了多少" | Op 2 |
| B | Create/modify budget | "设置预算¥10,000" | Op 6 |
| C | Multi-dimension analysis | "按产品分析成本" | Op 8 |
| D | Find savings opportunities | "哪些可以优化" | Op 12 |
| E | Reserved capacity advice | "该不该买包年" | Op 13 |
| F | Manage resource packages | "查看资源包" | Op 7 |
| G | Detect cost anomalies | "费用突然涨了" | Op 11 |
| H | Generate reports | "FinOps报告" | Op 5 + Op 8 |
| I | Cross-skill delegation | "ECS哪些能降配" | Delegate |
| J | Maturity evaluation | "FinOps水平" | Op 15 |

## Cross-Skill Delegation Patterns

| When User Says | Intent | This Skill Does | Delegates To | Expected Op |
|---------------|--------|----------------|--------------|-------------|
| "停止闲置ECS" | Stop idle resource | Detect idle via Op 12 | huaweicloud-ecs-ops (stop) | P1 |
| "释放无用的磁盘" | Delete orphaned disk | Detect orphaned via Op 12 | huaweicloud-ecs-ops (delete) | P1 |
| "ECS降配" | Resize down | Detect over-provisioned via Op 12 | huaweicloud-ecs-ops (resize) | P2 |
| "RDS升配" | Resize up | Detect under-provisioned via Op 12 | huaweicloud-rds-ops (resize) | P2 |
| "买包年包月" | Buy reserved | Coverage analysis via Op 13 | — (provide recommendation) | P3 |
| "买资源包" | Buy resource package | Needs analysis via Op 13 | — (provide recommendation) | P4 |
| "OBS归档旧数据" | Tier storage | Detect old data via Op 12 | huaweicloud-obs-ops (lifecycle) | P5 |
| "清理日志" | Prune logs | Detect old logs via Op 12 | huaweicloud-lts-ops (delete) | P6 |
| "退订闲置资源包" | Refund package | Detect idle via Op 12 | huaweicloud-billing-ops (refund) | P7 |
| "检查僵尸资源" | Find zombie | Scan billing for inactive via Op 12 | huaweicloud-{product}-ops | P8 |
| "创建ECS实例" | Provision | — (not billing) | huaweicloud-ecs-ops | — |
| "配置安全组" | Network config | — (not billing) | huaweicloud-vpc-ops | — |
| "创建IAM用户" | IAM operation | — (not billing) | huaweicloud-iam-ops | — |
| "查看CPU使用率" | Monitor | — (not billing) | huaweicloud-ces-ops | — |
| "备份RDS" | Backup | — (not billing) | huaweicloud-rds-ops | — |
| "费用异常诊断" | Cross-diagnose | Cost analysis via Op 8+11 | huaweicloud-cts-ops (audit) | G |
| "ECS费用为什么高" | Resource cost drill-down | Cost breakdown via Op 8 | huaweicloud-ecs-ops (config check) | C |
| "CDN带宽费用分析" | Service cost analysis | Bandwidth cost via Op 8 | huaweicloud-cdn-ops (bandwidth) | C |
| "容器集群成本优化" | Container optimization | Cluster cost via Op 8+12 | huaweicloud-cce-ops (node pool) | D |
| "综合FinOps报告" | Full FinOps report | All Ops combined | Cross-skill aggregation | H |

## References

- [references/core-concepts.md](references/core-concepts.md) — BSS architecture, billing models, quotas, endpoints
- [references/api-sdk-usage.md](references/api-sdk-usage.md) — API operation map, pagination, request/response snippets
- [references/cli-usage.md](references/cli-usage.md) — CLI command map, coverage gap table, JSON output paths
- [references/troubleshooting.md](references/troubleshooting.md) — Error code taxonomy (≥ 15 codes), diagnostic flows
- [references/monitoring.md](references/monitoring.md) — BSS observability, budget burn rate, optimization backlog (§9), closed-loop workflow (§10)
- [references/integration.md](references/integration.md) — JIT SDK bootstrap, env vars, cross-skill delegation matrix
- [references/well-architected-assessment.md](references/well-architected-assessment.md) — Five pillars, FinOps workflow (§3.1), AIOps maturity (§5.2), maturity scorecard (§7)
- [references/knowledge-base.md](references/knowledge-base.md) — Common billing fault patterns