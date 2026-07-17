# GCL Rubric — huaweicloud-ces-ops

> Per-skill rubric instance for the **Generator-Critic-Loop (GCL)** adversarial quality gate.
> Spec: see root `AGENTS.md` §3 (rubric spec) and §8 (per-skill defaults).

> **Version**: v1 (Phase 3, 2026-06-04)
> **max_iter**: 3
> **Scope**: every CES (Cloud Eye Service) mutating operation — alarm rule create / delete /
> enable / disable, dashboard create / delete. Read-only `list*` / `query-metric` are exempt.

## 1. Dimensions

| # | Dimension | What it checks | Threshold |
|---|-----------|----------------|-----------|
| 1 | **Correctness** | Alarm / dashboard state matches request | ≥ 0.5 |
| 2 | **Safety** | Confirmation; notification configured; no silent failures | **= 1** (any S-rule hit → ABORT) |
| 3 | **Idempotency** | Re-running does not duplicate alarms/dashboards | ≥ 0.5 |
| 4 | **Traceability** | Full request/response; credentials masked | ≥ 0.5 |
| 5 | **Spec Compliance** | Alarm type, metric namespace, evaluation period, threshold | ≥ 0.5 |

## 2. CES-Specific Safety Rules (binding — `Safety = 0` on any hit)

| # | Rule | Rationale |
|---|------|-----------|
| S1 | `delete-alarm-rule` without explicit user confirmation quoting the rule ID | **CRITICAL** — monitoring gap |
| S2 | `delete-alarm-rule` that is currently firing (alerting) without acknowledgement | Escalating issue goes undetected |
| S3 | `disable-alarm` when it is the only alerting rule for an important metric | Monitoring blind spot |
| S4 | `create-alarm-rule` with empty `alarm_actions` (no notification) for critical metrics | Alert fires but no one is notified |
| S5 | `create-alarm-rule` with evaluation period < 1 minute for non-critical metrics | Alarm storm / unnecessary cost |
| S6 | `delete-dashboard` without checking if metrics/widgets reference it | Users lose monitoring visibility |
| S7 | `create-alarm-rule` referencing a non-existent resource ID (silent failure) | Rule created but never fires |
| S8 | Any trace contains `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / password plaintext | Credential leak |
| S9 | `create-alarm-rule` with threshold = 0 (trigger-immediately) for CPU/memory | False alarm storm |
| S10 | `delete-dashboard` that is shared with other users/teams without confirmation | Multi-tenant visibility loss |

## 3. Correctness Check Matrix

| Operation | Required post-state |
|-----------|---------------------|
| `create-alarm-rule` | `ShowAlarmRule` returns `alarm_enabled: true` with matching metric_name + threshold |
| `delete-alarm-rule` | `ShowAlarmRule` returns 404 |
| `enable-alarm` | `ShowAlarmRule(alarm_id)` returns `alarm_enabled: true` |
| `disable-alarm` | `ShowAlarmRule(alarm_id)` returns `alarm_enabled: false` |
| `create-dashboard` | `ShowDashboard` returns `dashboard_id`; widgets are queryable |
| `delete-dashboard` | `ShowDashboard` returns 404 |

## 4. Idempotency Patterns

| Op | Idempotency mechanism |
|----|----------------------|
| `create-alarm-rule` | Pre-check `ListAlarmRules(name=…)`; if same metric+threshold exists, return existing id |
| `delete-alarm-rule` | Pre-check 404 |
| `enable/disable-alarm` | Read current `alarm_enabled`; if already target state, no-op |
| `create-dashboard` | Pre-check `ListDashboards(name=…)`; if exists, return existing |
| `delete-dashboard` | Pre-check 404 |

## 5. Traceability Checklist

- [ ] `command` and resolved `args` (post-substitution)
- [ ] `exit_code` + `stdout` (≤ 4 KB) + `stderr`
- [ ] `region` / `project_id` actually used
- [ ] For `create-alarm-rule`: the full alarm_actions list is captured (redacted if contains PII)
- [ ] **No** `password` / `PASSWORD` / `sk-…` / `SecretAccessKey` in trace

## 6. Spec Compliance Anchors

- Alarm types: `EVENT.SYS`, `EVENT.CUSTOM`, `METRIC`
- Metric namespaces: `SYS.ECS`, `SYS.RDS`, `SYS.DCS`, `SYS.ELB`, `SYS.VPC`, etc.
- Evaluation period: 1 min, 5 min, 15 min, 1 hour, 24 hours
- Threshold: 0–100 for percentage metrics; type-dependent for others
- Alarm actions: SMS, Email, Webhook, AutoScaling, ECStopped, etc.
- Dashboard sharing: can be shared across IAM users within same account

## 7. Scoring Summary

| Op | Correctness | Safety | Idempotency | Traceability | Spec Compliance |
|----|-------------|--------|-------------|--------------|-----------------|
| `create-alarm-rule` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 |
| `delete-alarm-rule` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 |
| `enable-alarm` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 |
| `disable-alarm` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 |
| `create-dashboard` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 |
| `delete-dashboard` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 |

## 8. Termination Mapping (per AGENTS.md §5)

| Result | Decision |
|--------|----------|
| All dims pass AND Safety = 1 | **PASS** |
| Safety = 0 | **SAFETY_FAIL** → ABORT |
| Any non-Safety < threshold, iter < max_iter | **RETRY** |
| iter == max_iter | **MAX_ITER** |

## 8.2 Changelog

| Version | Date | Change |
|---|---|---|
| v1 | 2026-06-04 | Initial rubric. |

## 9. See also

- `AGENTS.md` §3, §5, §7, §8 — repo-wide GCL spec
- `references/prompt-templates.md` — Generator + Critic + Orchestrator skeletons
- `references/core-concepts.md` — CES metric namespaces, alarm types