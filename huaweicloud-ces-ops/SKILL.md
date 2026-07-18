---
name: huaweicloud-ces-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei
  Cloud Cloud Eye Service (CES / 云监控服务) — alarm rules, metrics, dashboards,
  and events lifecycle. User mentions CES, Cloud Eye, 云监控, 告警规则,
  监控指标, or describes scenarios (e.g., alarm rule creation, metric query,
  dashboard setup, alarm storm) even without naming the product directly.
  Not for billing, IAM, or related products that have their own ops skills.
license: MIT
compatibility: >-
  Official Huawei Cloud CLI (`hcloud` / `openstack`), Go 1.21+ runtime
  (for JIT SDK fallback via huaweicloud-sdk-go-v3), valid API credentials,
  network access to Huawei Cloud endpoints.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-05-20"
  runtime: Harness AI Agent, Claude Code, Cursor, or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  api_profile: "CES API v1.0 - https://support.huaweicloud.com/api-ces/ces_api_0001.html"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    CES product supported by hcloud CLI. Use `hcloud ces --help` to verify
    available commands for alarm, metric, and dashboard operations.
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
  gcl:
    enabled: true
    required: true
    rubric_version: "v1"
    max_iter: 3
    rubric_ref: "references/rubric.md"
    prompts_ref: "references/prompt-templates.md"
    trace_dir: "./audit-results/"
    changelog:
      - version: "1.1.0"
        date: "2026-06-04"
        change: "GCL Phase 3 rollout: added references/rubric.md (v1, 5-dim, S1–S10 CES-specific Safety rules, including alarm-rule-delete-without-confirmation / alarm-without-notification / dashboard-delete-with-metrics / credential-leak guards) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud Cloud Eye Service (CES) Operations Skill

## Overview

Huawei Cloud Cloud Eye Service (CES / 云监控服务) provides comprehensive monitoring, alarm, and dashboard capabilities for cloud and custom resources. This skill is an **operational runbook** for agents: explicit scope, credential rules, pre-flight checks, **dual-path execution** (official **SDK/API** and **`hcloud` CLI**), response validation, and failure recovery.

### CLI applicability (repository policy)

- **`cli_applicability: dual-path`:** Official CLI supports CES product. You **MUST** ship **`references/cli-usage.md`** and, in **each** execution flow, document **both** the SDK step **and** the CLI step.

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions with precise triggers and delegation rules |
| 2 | **Structured I/O** | Placeholder conventions with type and source documented |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute (CLI + SDK) → Validate → Recover |
| 4 | **Complete Failure Strategies** | Error taxonomy ≥ 10 codes; HALT vs retry per error type |
| 5 | **Absolute Single Responsibility** | One product (CES), one resource model; cross-product alerts delegate to respective skills |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **FinOps** | Monitoring cost tracking, metric retention cost, dashboard optimization | `references/well-architected-assessment.md` §3 (Cost Pillar) |
| **SecOps** | IAM minimum permissions, credential masking, monitor data access control | `references/well-architected-assessment.md` §4 (SecOps section) |
| **AIOps** | ≥ 6 anomaly patterns, cross-skill diagnosis, alarm storm suppression | `references/monitoring.md` and `references/knowledge-base.md` |

### Well-Architected Framework Integration (卓越架构)

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **安全 (Security)** | IAM permissions for alarm/metric access, credential isolation | `references/well-architected-assessment.md` §1 |
| **稳定 (Stability)** | Multi-region alarm redundancy, critical alarm escalation | `references/well-architected-assessment.md` §2 |
| **成本 (Cost)** | Metric retention billing, dashboard cost optimization | `references/well-architected-assessment.md` §3 |
| **效率 (Efficiency)** | Batch metric queries, alarm template reuse | `references/well-architected-assessment.md` §4 |
| **性能 (Performance)** | Metric data query performance, alarm evaluation tuning | `references/well-architected-assessment.md` §5 |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud CES", "Cloud Eye", "云监控", "云监控服务"
- Task involves alarm rule lifecycle: create, list, enable, disable, delete, modify
- Task involves metric data query: single metric, batch query, custom metrics
- Task involves dashboard management: create, list, view, delete dashboards
- Task involves event management: list events, add event data
- Task keywords: 告警规则, 监控指标, 仪表盘, 告警风暴, CPU使用率告警, 内存告警
- User asks to configure, troubleshoot, or monitor CES resources via API, SDK, CLI, or automation
- Task involves GCL (Generator-Critic-Loop) pass-rate monitoring: trace parsing, custom metric push to `CUSTOM.GCL`, or creating CES alarm rules for GCL health (`gcl-overall-pass-rate-critical`, `gcl-safety-fail-detected`, etc.) → refer to `references/gcl-monitoring.md`

### SHOULD NOT Use This Skill When

- Task is purely billing / account management → delegate to: `huaweicloud-billing-ops`
- Task is IAM / permission model only → delegate to: `huaweicloud-iam-ops`
- Task is creating/deleting the **monitored resource** itself (e.g., ECS instance) → delegate to: `huaweicloud-ecs-ops`
- Task is VPC network configuration → delegate to: `huaweicloud-vpc-ops`

### Delegation Rules

- If creating alarms for a resource, verify the target resource exists first using the respective product skill.
- Multi-product alarm requests: handle CES alarm creation with this skill; resource creation with the respective product skill.
- For FinOps monitoring cost questions: use this skill's cost section.
- For SecOps credential issues: use this skill's security section, delegate account-level IAM to IAM skill.

## Variable Convention (Agent-Readable)

| Placeholder | Meaning | Agent Action |
|-------------|---------|--------------|
| `{{env.HW_ACCESS_KEY_ID}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_REGION_ID}}` | From runtime environment | Use documented default only if skill explicitly allows |
| `{{env.HW_PROJECT_ID}}` | From runtime environment | Use documented default only if skill explicitly allows |
| `{{user.region}}` | User-supplied region | Ask once; reuse |
| `{{user.alarm_name}}` | User-supplied alarm name | Ask once; reuse |
| `{{user.metric_namespace}}` | CES metric namespace (e.g., SYS.ECS) | Ask if unknown, provide common namespace list |
| `{{user.resource_id}}` | User-supplied resource ID | Ask once; reuse |
| `{{output.alarm_id}}` | From alarm API response | Parse per OpenAPI: `$.alarm_id` |
| `{{output.metric_data}}` | From metric query response | Parse per OpenAPI: `$.datapoints` |

> **`{{env.*}}` MUST NOT** be collected from the user. **`{{user.*}}`** MUST be collected interactively when missing.

> **Security Warning (Credential Masking — MANDATORY):** **NEVER** log, print, or expose `HW_SECRET_ACCESS_KEY` or any credential field value.

## API and Response Conventions

- **OpenAPI is canonical** for path, query, body fields, enums, and response shapes.
- **Errors:** Map SDK/HTTP errors to `error_code` / `error_msg` fields per spec.
- **Timestamps:** ISO 8601 (epoch milliseconds for CES metric data).
- **Idempotency:** Alarm names must be unique per project; duplicate names return `CES.0012`.

## Quick Start

### What This Skill Does
Manages Huawei Cloud CES (Cloud Eye / 云监控服务) alarm rules, metric queries, dashboards, and event monitoring.

### Prerequisites

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|-----------|-------------|------------|------------|
| CreateAlarm | Create alarm rule with metric, threshold, notification | Medium | Low |
| ListAlarms | List alarm rules with filters | Low | None |
| EnableAlarm / DisableAlarm | Toggle alarm rule state | Low | Low |
| DeleteAlarm | Remove alarm rule | Low | **High** — irreversible |
| QueryMetric | Query metric data for a resource | Medium | None |
| BatchQueryMetrics | Batch query multiple metric data series | Medium | None |
| CreateDashboard | Create monitoring dashboard | Low | Low |
| ListDashboards | List dashboards | Low | None |
| ListEvents | Query cloud service events | Low | None |
| ShowQuotas | Query CES resource quotas | Low | None |

## Execution Flows

### Operation: Create Alarm Rule

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| CLI / deps | `hcloud --version` | Exit code 0 | Document CLI install |
| Credentials | Env var existence check | Non-empty AK/SK | HALT; user configures env |
| Region | Verify `{{user.region}}` is valid Huawei Cloud region | Region supported | Suggest valid region |
| Target resource | Verify resource existence via respective product API | Resource ACTIVE | HALT; resource not found |
| Namespace validity | Check metric namespace format (e.g., SYS.ECS) | Valid namespace | List valid namespaces for user |

#### Execution — CLI (Primary Path)

```bash
# Create alarm rule
hcloud ces create-alarm-rule \
  --region "{{user.region}}" \
  --alarm-name "{{user.alarm_name}}" \
  --alarm-enabled true \
  --alarm-action-name "{{user.notification_topic_urn}}" \
  --alarm-resources "{{user.resource_id}}" \
  --metric-namespace "{{user.metric_namespace}}" \
  --metric-name "{{user.metric_name}}" \
  --metric-dimension.0.name "instance_id" \
  --metric-dimension.0.value "{{user.resource_id}}" \
  --comparison-operator "{{user.comparison_operator:GT|LT|GTE|LTE|EQ}}" \
  --threshold "{{user.threshold}}" \
  --evaluation-periods "{{user.evaluation_periods:3}}" \
  --period "{{user.period:300}}" \
  --alarm-level "{{user.alarm_level:2}}"
```

#### Execution — JIT Go SDK (Fallback Path)

```go
package main

import (
    "fmt"
    "os"
    "strconv"

    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ces/v1"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ces/v1/model"
)

func main() {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    region := os.Getenv("HW_REGION_ID")

    cfg := config.DefaultHttpConfig()
    client := v1.CesClientBuilder().
        WithEndpoint(fmt.Sprintf("ces.%s.myhuaweicloud.com", region)).
        WithCredential(basic.NewCredentialsBuilder().
            WithAk(ak).WithSk(sk).Build()).
        WithHttpConfig(cfg).Build()

    thresholdFloat, _ := strconv.ParseFloat(os.Getenv("ALARM_THRESHOLD"), 64)
    evalInt32, _ := strconv.ParseInt(os.Getenv("ALARM_EVAL_PERIODS"), 10, 32)
    periodInt32, _ := strconv.ParseInt(os.Getenv("ALARM_PERIOD"), 10, 32)

    request := &model.CreateAlarmRuleRequest{
        Body: &model.CreateAlarmRuleParam{
            AlarmName:           os.Getenv("ALARM_NAME"),
            AlarmEnabled:        true,
            AlarmActionName:     os.Getenv("ALARM_TOPIC_URN"),
            AlarmResources:      []string{os.Getenv("RESOURCE_ID")},
            MetricNamespace:     os.Getenv("METRIC_NAMESPACE"),
            MetricName:          os.Getenv("METRIC_NAME"),
            ComparisonOperator:  os.Getenv("COMPARISON_OPERATOR"),
            Threshold:           thresholdFloat,
            EvaluationPeriods:   int32(evalInt32),
            Period:              int32(periodInt32),
        },
    }

    response, err := client.CreateAlarmRule(request)
    if err != nil {
        fmt.Fprintf(os.Stderr, "CreateAlarmRule failed: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("%+v\n", response)
}
```

#### Post-execution Validation

1. Read `{{output.alarm_id}}` from response path `$.alarm_id`.
2. Call **DescribeAlarm** with `{{output.alarm_id}}` to confirm exists and enabled.
3. On success, report `{{output.alarm_id}}`, alarm name, and metric details.
4. On terminal failure, go to **Failure Recovery**.

#### Failure Recovery

| Error | Max retries | Backoff | Agent Action | UX Feedback |
|-------|-------------|---------|--------------|-------------|
| `CES.0003` InvalidParameter | 0–1 | — | Fix args from OpenAPI | `[ERROR] InvalidParameter: Check parameters against CES API docs.` |
| `CES.0010` InvalidRequestData | 0–1 | — | Fix request body | `[ERROR] InvalidRequestData: Verify request format and field types.` |
| `CES.0012` AlarmAlreadyExists | 0 | — | Ask reuse vs new name | `[ERROR] Alarm rule already exists. Use different name or reuse.` |
| `CES.0013` MetricNotFound | 0 | — | Verify namespace/metric | `[ERROR] Metric not found. Check namespace and metric_name.` |
| `QuotaExceeded` / `CES.0020` | 0 | — | HALT | `[ERROR] Alarm quota exceeded. Delete unused alarms.` |
| `InvalidParameter` | 0–1 | — | Fix args | `[ERROR] Invalid parameter. Review field values.` |
| `InsufficientBalance` | 0 | — | HALT | `[ERROR] Insufficient balance. Recharge your Huawei Cloud account.` |
| Throttling / 429 / `CES.0006` | 3 | exponential | Back off; respect Retry-After | `⚠️ Rate limited. Retrying in {backoff}s...` |
| `CES.0016` ProjectNotAuthorized | 0 | — | Verify project | `[ERROR] Unauthorized project. Check IAM permissions.` |
| `InternalError` / 5xx | 3 | 2s, 4s, 8s | Retry; then HALT | `[ERROR] Server-side error. Retry or escalate with RequestId.` |

### Operation: List / Describe Alarm Rules

#### Execution — CLI

```bash
# List alarm rules
hcloud ces list-alarms \
  --region "{{user.region}}" \
  --alarm-name "{{user.alarm_name}}" \
  --alarm-enabled "{{user.enabled:true|false|all}}"

# Describe specific alarm
hcloud ces describe-alarm \
  --region "{{user.region}}" \
  --alarm-id "{{output.alarm_id}}"
```

#### Execution — SDK

```
GET  /V1.0/{project_id}/alarms                     — List alarms
GET  /V1.0/{project_id}/alarms/{alarm_id}           — Describe alarm
```

#### Post-execution Validation

- Verify alarm state matches expected (enabled/disabled).
- Report alarm details: name, metric namespace, metric name, threshold, comparison operator, evaluation periods, notification topic.

### Operation: Enable / Disable Alarm

#### Execution — CLI

```bash
# Enable alarm
hcloud ces enable-alarm \
  --region "{{user.region}}" \
  --alarm-id "{{user.alarm_id}}"

# Disable alarm
hcloud ces disable-alarm \
  --region "{{user.region}}" \
  --alarm-id "{{user.alarm_id}}"
```

#### Execution — SDK

```
PUT  /V1.0/{project_id}/alarms/{alarm_id}/action  — Enable/Disable (body: {"alarm_enabled": true|false})
```

#### Post-execution Validation

- Call **DescribeAlarm** and confirm `alarm_enabled` field reflects requested state.

### Operation: Delete Alarm Rule

#### Pre-flight (Safety Gate)

- **MUST** obtain explicit confirmation: irreversible delete of alarm `{{user.alarm_name}}` (`{{user.alarm_id}}`).
- **MUST NOT** proceed without clear user assent.

#### Execution

```bash
hcloud ces delete-alarm \
  --region "{{user.region}}" \
  --alarm-id "{{user.alarm_id}}"
```

#### Post-execution Validation

- Call **DescribeAlarm** — expect 404 / `AlarmNotFound`.
- Confirm deletion within 60 seconds.

### Operation: Query Metric Data

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| Namespace | Verify namespace pattern (SYS.xxx or SERVICE.xxx) | Valid prefix | HALT; provide namespace list |
| Resource existence | Verify via respective product skill | Resource exists | HALT; resource not found |

#### Execution — CLI

```bash
# Query single metric
hcloud ces query-metric-data \
  --region "{{user.region}}" \
  --metric-namespace "{{user.metric_namespace}}" \
  --metric-name "{{user.metric_name}}" \
  --metric-dimension.0.name "instance_id" \
  --metric-dimension.0.value "{{user.resource_id}}" \
  --from "from-{{user.from_time}}" \
  --to "to-{{user.to_time}}" \
  --filter "{{user.filter:average}}" \
  --period "{{user.period:1}}"
```

#### Execution — SDK

```
GET  /V1.0/{project_id}/metric-data?namespace=NAMESPACE&metric_name=METRIC&dim.0=key1,value1&filter=average&period=1&from=FROM&to=TO
POST /V1.0/{project_id}/metric-data/batch-query  — Batch query
```

#### Post-execution Validation

- Verify `datapoints` array is non-empty when resource exists and has data.
- Report: datapoint count, min/max/avg values, time range.
- If empty: confirm resource exists, metric name is correct, time range has data.

#### Failure Recovery

| Error | Max retries | Backoff | Agent Action | UX Feedback |
|-------|-------------|---------|--------------|-------------|
| `CES.0013` MetricNotFound | 0 | — | Verify namespace/metric | `[ERROR] Metric not found. Check namespace and metric_name.` |
| `CES.0003` InvalidParameter | 0–1 | — | Fix time range/format | `[ERROR] Invalid parameter. Check from/to timestamps.` |
| Throttling / 429 | 3 | exponential | Back off | `⚠️ Rate limited. Retrying...` |

### Operation: Create Dashboard

#### Pre-flight Checks

- Verify user has dashboard creation permission.
- Collect dashboard name.

#### Execution — CLI

```bash
hcloud ces create-dashboard \
  --region "{{user.region}}" \
  --title "{{user.dashboard_name}}"
```

#### Post-execution Validation

- Read dashboard ID from response `$.id`.
- Call **ShowDashboard** to confirm it exists.

### Operation: Delete Dashboard

#### Pre-flight (Safety Gate)

- **MUST** obtain explicit confirmation before deletion.

#### Execution

```bash
hcloud ces delete-dashboard \
  --region "{{user.region}}" \
  --id "{{user.dashboard_id}}"
```

#### Post-execution Validation

- Call **ShowDashboard** with `{{user.dashboard_id}}` — expect `DashboardNotFound` (404).
- Confirm deletion within 60 seconds.

### Operation: List Events

#### Execution — CLI

```bash
hcloud ces list-events \
  --region "{{user.region}}" \
  --namespace "{{user.event_namespace:CES}}" \
  --from "from-{{user.from_time}}" \
  --to "to-{{user.to_time}}"
```

### Operation: Show Quotas

#### Execution — CLI

```bash
hcloud ces show-quotas \
  --region "{{user.region}}"
```

### Operation: Self-Healing — Advanced Alarm Management

Two self-healing patterns for production alarm management:

- **Auto Re-enable After Deployment**: Re-enables alarms disabled during deployment windows. Detects disabled alarms, validates deployment completion, re-enables with confirmation, and escalates failures.
- **Auto-adjust Thresholds**: Analyzes 30-day historical baselines (P95 + 10% buffer) to tune thresholds and reduce false positives without decreasing sensitivity.

> Full implementation (CLI + Go SDK, pre-flight checks, failure recovery, idempotency): see [`references/advanced/self-healing.md`](references/advanced/self-healing.md)

## Prerequisites

> Full installation scripts (KooCLI + Go runtime + Credentials): see [references/common-prerequisites.md](../references/common-prerequisites.md)

## Quality Gate (GCL)

This skill is **GCL-recommended** (per `AGENTS.md` §8). Every CES mutating operation — alarm rule create / delete / enable / disable, dashboard create / delete — runs through the **Generator-Critic-Loop** before its result is returned. Read-only metric queries and list operations are GCL-**exempt**.

| Field | Value |
|-------|-------|
| Rubric version | v1 (Phase 3, 2026-06-04) |
| `max_iter` | **3** |
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
| 1 | Correctness | ≥ 0.5 | `ShowAlarm` / `ShowDashboard` post-state |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | S1–S10 in rubric §2 |
| 3 | Idempotency | ≥ 0.5 | Pre-check before create |
| 4 | Traceability | ≥ 0.5 | Credential MUST be `<masked>` |
| 5 | Spec Compliance | ≥ 0.5 | Alarm type / metric namespace / evaluation period |

### Per-Operation Safety Anchors (binding)

- **S1** — `delete-alarm-rule` without explicit user confirmation quoting the rule ID
- **S2** — `delete-alarm-rule` that is currently firing (alerting) without acknowledgement
- **S3** — `disable-alarm` when it is the only alerting rule for an important metric
- **S4** — `create-alarm-rule` with empty `alarm_actions` (no notification) for critical metrics
- **S5** — `create-alarm-rule` with evaluation period < 1 minute for non-critical metrics
- **S6** — `delete-dashboard` without checking if metrics/widgets reference it
- **S7** — `create-alarm-rule` referencing a non-existent resource ID (silent failure)
- **S8** — any trace contains `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / password plaintext
- **S9** — `create-alarm-rule` with threshold = 0 (trigger-immediately) for CPU / memory
- **S10** — `delete-dashboard` that is shared with other users/teams without confirmation

### See also

- [`references/rubric.md`](references/rubric.md) — full rubric, S1–S10 rules, per-op thresholds
- [`references/prompt-templates.md`](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons
- Repository root [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8 — GCL specification

## Reference Directory

- [Core Concepts](references/core-concepts.md)
- [API & SDK Usage](references/api-sdk-usage.md)
- [CLI Usage](references/cli-usage.md)
- [Troubleshooting Guide](references/troubleshooting.md)
- [Monitoring & Alerts](references/monitoring.md) — CES self-monitoring patterns
- [Integration](references/integration.md)
- [Knowledge Base](references/knowledge-base.md)
- [Observability Integration](references/observability.md)
- [FinOps Cost Optimization](references/well-architected-assessment.md#3-finops-)
- [SecOps Security Operations](references/well-architected-assessment.md#4-secops-)
- [Well-Architected Assessment](references/well-architected-assessment.md)
- [GCL Rubric](references/rubric.md) — Adversarial quality gate (v1, 5-dim, S1–S10 CES-specific Safety rules)
- [GCL Prompt Templates](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons
- [GCL Monitoring](references/gcl-monitoring.md) — GCL pass-rate monitoring design: trace parser script, CES alarm rules, custom metric push via SDK, dashboards, and threshold optimization (Phase 4)

## Well-Architected + Three-Pillar Assessment

This skill's operations are evaluated against Huawei Cloud's Well-Architected Framework (卓越架构) five pillars plus FinOps, SecOps, and AIOps integration:
- [Security Assessment](references/well-architected-assessment.md)
- [Stability Assessment](references/well-architected-assessment.md)
- [Cost Assessment](references/well-architected-assessment.md)
- [Efficiency Assessment](references/well-architected-assessment.md)
- [Performance Assessment](references/well-architected-assessment.md)
- [FinOps Integration](references/well-architected-assessment.md)
- [SecOps Integration](references/well-architected-assessment.md)
- [AIOps Integration](references/knowledge-base.md)

> 任务完成后按根 AGENTS.md 的「复利资产沉淀机制 (CADL)」复盘并沉淀可复用资产。
