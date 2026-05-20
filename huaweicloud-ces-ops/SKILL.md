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
- [ ] Huawei Cloud CLI installed (or Go runtime for JIT fallback)
- [ ] Credentials configured: `HW_ACCESS_KEY_ID`, `HW_SECRET_ACCESS_KEY`
- [ ] Region and Project ID set: `HW_REGION_ID`, `HW_PROJECT_ID`

### Verify Setup
```bash
hcloud ces list-alarms --region {{env.HW_REGION_ID}}
```

### Your First Command
```bash
# List all alarm rules in region
hcloud ces list-alarms --region {{env.HW_REGION_ID}}
```

### Next Steps
- [Core Concepts](references/core-concepts.md) — Understand CES architecture and metrics
- [Execution Flows](#execution-flows) — Alarm, metric, dashboard operations
- [Troubleshooting](references/troubleshooting.md) — Fix common CES issues

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

## Prerequisites

1. **Install KooCLI** (official binary):

    ```bash
    curl -sSL https://cn-north-4.myhuaweicloud.com/cli/latest/hcloud_install.sh -o ./hcloud_install.sh && bash ./hcloud_install.sh -y
    hcloud version
    ```

2. **Bootstrap Go runtime** (JIT SDK fallback):

    ```bash
    if ! command -v go &> /dev/null; then
        OS=$(uname -s | tr '[:upper:]' '[:lower:]')
        ARCH=$(uname -m)
        [ "$ARCH" = "x86_64" ] && ARCH="amd64"
        [ "$ARCH" = "aarch64" ] && ARCH="arm64"
        mkdir -p /tmp/go-runtime
        curl -fsSL "https://go.dev/dl/go1.25.0.${OS}-${ARCH}.tar.gz" | tar -xz -C /tmp/go-runtime
        export PATH="/tmp/go-runtime/go/bin:$PATH"
        export GOPROXY="https://goproxy.cn,direct"
    fi
    ```

3. **Configure Credentials**:

    ```bash
    export HW_ACCESS_KEY_ID="{{env.HW_ACCESS_KEY_ID}}"
    export HW_SECRET_ACCESS_KEY="{{env.HW_SECRET_ACCESS_KEY}}"
    export HW_REGION_ID="{{env.HW_REGION_ID}}"
    export HW_PROJECT_ID="{{env.HW_PROJECT_ID}}"
    test -n "$HW_SECRET_ACCESS_KEY" && echo "✅ Credentials configured"
    ```

4. **Verify Configuration**: `hcloud ces list-alarms --region {{env.HW_REGION_ID}}`

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
