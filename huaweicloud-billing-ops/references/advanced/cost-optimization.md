# Billing FinOps — Huawei Cloud BSS Deep Dive

> Advanced FinOps patterns layered below the runbook (`references/*.md`).
> Load this file only when the agent needs TCO modeling, reservation
> strategy, or cost anomaly RCA — not for routine bill queries.

## 1. Unit Economics (单位经济学)

| Metric | Formula | Source | Optimization direction |
|--------|---------|--------|------------------------|
| Cost per vCPU-hour | Monthly cost / Σ(vCPU × hours) | BSS + product API | Right-size flavor, switch to 包年包月 |
| Cost per GB-month | Storage cost / Σ(GB-months) | BSS + storage API | Move cold data to 低频 / 归档 |
| Cost per request | Total cost / request count | BSS + CES QPS | Architecture consolidation |
| Cost per active user | Total cost / MAU | BSS + business metrics | Multi-tenant consolidation |

## 2. Reservation Coverage Strategy (预留策略)

- **Break-even formula**: `盈亏月数 = 包年包月总价 / (按需月价 − 包年包月月均价)`
- **Coverage bands**:
  - `> 85%` → audit for over-reservation and idle 包年包月
  - `60-85%` → healthy, continue monitoring
  - `40-60%` → identify steady-state workloads to convert
  - `< 40%` → risk; convert long-running pay-per-use to reservation

## 3. Cost Anomaly Detection

| Pattern | Detection | Severity | Response |
|---------|-----------|----------|----------|
| 成本突增 | day_cost > 7d_avg × 1.5 | Critical | notify + RCA |
| 成本突降 | day_cost < 7d_avg × 0.5 | Warning | check service degradation |
| 预算偏差 | actual > budget × 110% | Warning | adjust budget or optimize |
| 资源突增 | new_resources > 7d_avg × 2 | Warning | confirm planned scaling |
| 闲置浪费反弹 | idle_rate +10% WoW | Info | trigger cleanup wave |

> **Security-Sensitive**: any cost-mutation API (`update-budget`, `delete-budget`,
> `apply-refund`) MUST require explicit operator confirmation before invocation.
> Confirm via Safety Gate (`{{user.confirm_cost_mutation}}`).