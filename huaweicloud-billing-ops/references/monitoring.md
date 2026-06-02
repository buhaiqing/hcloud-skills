# Monitoring — BSS (费用中心)

## 1. Overview

Billing monitoring covers account balance, budget burn rate, cost trends, and cost anomaly alerts. Unlike infrastructure monitoring, billing has lower frequency (daily checks are sufficient) but higher impact (budget overshoots are irreversible).

## 2. Balance Monitoring

Monitor account balance to prevent service suspension.

| Metric | Check Frequency | Warning Threshold | Critical Threshold |
|--------|----------------|-------------------|-------------------|
| Account balance | Daily | < ¥500 | < ¥100 |
| Credit amount | Daily | < ¥1,000 | < ¥500 |
| Overdue amount | Daily | > ¥0 | — |

## 3. Budget Burn Rate

Track monthly budget consumption rate.

| Metric | Formula | Alert |
|--------|---------|-------|
| Burn rate | (spent so far / days elapsed) × days in month | > 80% of budget |
| Projected spend | burn_rate × total_days | > 100% of budget |

## 4. Cost Dashboard

A cost dashboard should include:

- Current month spend vs budget (gauge chart)
- Top 5 services by cost (bar chart)
- Cost trend last 12 months (line chart)
- Budget alert history (table)
- Optimization savings tracker (Op 14)

## 5. Budget Alert Notification Channels

| Channel | Configuration | Latency |
|---------|--------------|---------|
| Email | Auto-configured via budget API | < 5 min |
| SMS | Requires phone verification | < 2 min |
| CES Alarm | Via CES metric alarm | < 1 min |

## 6. Anomaly Detection Setup

Configure periodic cost checks (Op 11) to run:

- Daily: burn rate check (compare budget consumption)
- Weekly: cost comparison (current week vs average of 3 previous)
- Monthly: full anomaly scan (all services, all regions)

## 7. Cost Audit Trail

All cost-related operations should be logged:

| Operation | Audit Source | Retention |
|-----------|-------------|-----------|
| Budget create/update/delete | CTS | 180 days |
| Resource package refund | CTS | 180 days |
| Optimization action (Op 14) | Tracker log | Permanent |
| Maturity assessment (Op 15) | Scorecard file | Permanent |

## 8. Optimization Savings Tracking

Track cumulative savings from optimization actions:

| Period | Savings from Op 14 | % of Total Spend |
|--------|-------------------|-----------------|
| This month | ¥X,XXX | X.X% |
| Last month | ¥X,XXX | X.X% |
| Cumulative (YTD) | ¥X,XXX | X.X% |

---

## 9. Optimization Backlog (持续优化挖掘)

> **NEW.** The optimization backlog is a structured storage of detected optimization opportunities (output of Op 12).

### 9.1 Storage Layout

```
~/.hcloud/optimization_backlog/{cycle_id}/
├── summary.json                    # Cycle summary with total savings
└── patterns/
    ├── P1_idle_ecs.json           # Per-pattern backlog
    ├── P2_rightsize.json
    ├── P3_reserved_opportunity.json
    ├── P4_resource_package.json
    ├── P5_storage_tiering.json
    ├── P6_log_retention.json
    ├── P7_idle_package.json
    └── P8_zombie.json
```

### 9.2 Backlog Entry Schema (JSON)

```json
{
  "id": "OPT-20260603-001",
  "pattern": "P1_idle_ecs",
  "resource_id": "i-abc123",
  "resource_name": "ecs-web-dev-01",
  "service_type": "ECS",
  "region": "cn-north-4",
  "status": "open",
  "detected_at": "2026-06-03T10:00:00Z",
  "metrics": {
    "avg_cpu_utilization_pct": 2.3,
    "idle_days": 15,
    "monthly_cost": 450.00
  },
  "estimated_savings": {
    "monthly_cny": 450.00,
    "annual_cny": 5400.00
  },
  "confidence": "HIGH",
  "recommended_action": "stop_or_delete",
  "delegate_to": "huaweicloud-ecs-ops",
  "notes": ""
}
```

### 9.3 Prioritization Formula

```
priority_score = estimated_savings_annual_cny × confidence_weight × ease_factor

confidence_weight = 1.0 (HIGH) / 0.7 (MEDIUM) / 0.4 (LOW)

ease_factor:
  1.0 = fully automated (stop, refund, delete)
  0.7 = partially automated (resize, tier)
  0.4 = manual (architecture change, migration)
```

Backlog is sorted by `priority_score` descending.

### 9.4 Backlog Aging Policy

| Age | Action |
|-----|--------|
| < 30 days | Active (keep in backlog) |
| 30-60 days | Active but flag for review |
| 60-90 days | Mark as "stale — possible already resolved" |
| > 90 days | Auto-archive (move to history) |

### 9.5 Integration with Op 14

When an optimization is approved (Op 14), the backlog entry state changes from `"status": "open"` to `"status": "in_progress"`. On completion, it moves to `"status": "completed"` or `"status": "rejected"`.

---

## 10. Closed-Loop Workflow (闭环流程)

> **NEW.** The closed-loop workflow connects optimization discovery (Op 12) through application (Op 14) back to detector improvement — closing the FinOps loop.

### 10.1 Loop Diagram

```
                         ┌─────────────────────────────┐
                         │     OPTIMIZATION MINING      │
                         │        (Op 12 / Op 13)       │
                         │       patterns: P1 - P8       │
                         └─────────────┬───────────────┘
                                       │ optimization items
                                       ▼
                         ┌─────────────────────────────┐
                         │    CLOSED-LOOP TRACKER       │
                         │         (Op 14)              │
                         │  DETECTED → APPROVED → ...   │
                         │  VALIDATED / REGRESSED       │
                         └─────────────┬───────────────┘
                                       │
                     ┌─────────────────┼─────────────────┐
                     │                 │                 │
                     ▼                 ▼                 ▼
              ┌───────────┐   ┌───────────────┐   ┌───────────┐
              │ APPLIED   │   │  VALIDATED    │   │ REGRESSED │
              │(executed) │   │(savings ≥80%) │   │(savings   │
              │           │   │               │   │ <50%)     │
              └─────┬─────┘   └───────┬───────┘   └─────┬─────┘
                    │                 │                 │
                    │                 │                 │ LESSONS LEARNED
                    │                 │                 ▼
                    │                 │        ┌─────────────────┐
                    │                 │        │ regressed_lessons│
                    │                 │        │    .json         │
                    │                 │        └────────┬────────┘
                    │                 │                 │
                    │                 │                 ▼
                    │                 │        ┌─────────────────┐
                    │                 │        │   playbook.json  │
                    │                 │        │ (enrichment)     │
                    │                 │        └────────┬────────┘
                    │                 │                 │
                    │                 │                 │ feedback to
                    │                 │                 │ detector
                    │                 │                 ▼
                    │                 │        ┌─────────────────┐
                    └─────────────────┴───────▶│  OPTIMIZATION   │
                                               │  DETECTOR       │
                                               │  (tune patterns) │
                                               └─────────────────┘
                         ═══════  LOOP CLOSED  ═══════
```

### 10.2 regressed_lessons.json

When an optimization REGRESSES (savings < 50% of forecast), a lesson is recorded:

```json
{
  "id": "LESSON-20260603-001",
  "source_opt_id": "OPT-20260603-001",
  "pattern": "P2_rightsize",
  "regression_cause": "workload_increased",
  "forecast_savings_cny": 500.00,
  "actual_savings_cny": 120.00,
  "gap_pct": 76.0,
  "detected_at": "2026-06-17T10:00:00Z",
  "recommendation": "Add workload seasonality factor to P2 pattern"
}
```

### 10.3 playbook.json (Enrichment)

Over time, regressed lessons enrich a playbook that the detector uses:

```json
{
  "patterns": {
    "P2_rightsize": {
      "base_confidence": 0.7,
      "adjustments": [
        {
          "trigger": "workload_increased",
          "adjustment": -0.2,
          "notes": "Reduce confidence for workloads with history of growth"
        }
      ]
    }
  }
}
```

### 10.4 Loop SLOs

| Transition | P50 | P99 | Measurement |
|-----------|-----|-----|-------------|
| DETECTED → APPROVED | < 24h | < 72h | Time from detection to human approval |
| APPROVED → APPLIED | < 4h | < 24h | Time from approval to execution |
| MEASURING duration | 7 days | 14 days | Observation window after apply |
| REGRESSED → ROLLED_BACK | < 2h | < 8h | Recovery time |
| REGRESSED → lesson logged | < 1h | < 4h | Knowledge capture latency |
| Lesson → playbook updated | < 24h | < 72h | Detector improvement cycle |

### 10.5 CI/CD Integration

Example GitLab CI pipeline for weekly optimization scan:

```yaml
optimization-scan:
  stage: finops
  schedule: "0 6 * * 0"  # Every Sunday 6 AM
  script:
    - hcloud BSS ListCosts --time-range="LAST_30_DAYS" --group-by="service_type"
    - hcloud BSS ListCustomerselfResourceRecords --cycle="$(date -d '-1 month' +%Y-%m)" --method="DETAIL" --limit=1000
    # Agent runs Op 12, 14 logic
    - echo "Optimization scan complete. Backlog updated."
    - echo "Scheduled approval window: Mon 9-11 AM"
  artifacts:
    paths:
      - ~/.hcloud/optimization_backlog/
```