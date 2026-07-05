---
name: huaweicloud-cts-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei
  Cloud CTS (Cloud Trace Service) — audit trail lifecycle, event collection,
  trace query, and diagnostic analysis. User mentions CTS, 云审计, 云追踪, 审计日志,
  事件追踪, trace, audit trail, or describes scenarios (audit query, access
  history, compliance investigation, event correlation) even without naming CTS.
  Not for IAM policy creation, OBS bucket management, or billing analysis.
license: MIT
compatibility: >-
  KooCLI (official binary, latest **4.1.6**), Go 1.21+ runtime for JIT SDK
  fallback via huaweicloud-sdk-go-v3, valid AK/SK credentials, network access
  to Huawei Cloud endpoints.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-05-20"
  runtime: Harness AI Agent, Claude Code, Cursor, or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  api_profile: "https://support.huaweicloud.com/api-cts/"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    Huawei Cloud CTS is supported via `hcloud cts` CLI commands and
    huaweicloud-sdk-go-v3/services/cts/v3 Go SDK package.
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
  gcl:
    enabled: true
    required: false
    rubric_version: "v1"
    max_iter: 3
    rubric_ref: "references/rubric.md"
    prompts_ref: "references/prompt-templates.md"
    trace_dir: "./audit-results/"
    changelog:
      - version: "1.1.0"
        date: "2026-06-04"
        change: "GCL Phase 3 rollout: added references/rubric.md (v1, 5-dim, S1–S8 CTS-specific Safety rules, including tracker-delete-without-confirmation / tracker-disable-lose-audit / obs-bucket-inaccessible / credential-leak guards) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud CTS Operations Skill

## Overview

Huawei Cloud CTS (Cloud Trace Service / 云审计) captures API calls, user activity, and resource changes across cloud services. This skill is an **operational runbook** for agents: audit trail setup, trace query, event analysis, compliance investigation, and failure recovery. **Dual-path execution**: official **SDK/API** (`huaweicloud-sdk-go-v3/services/cts/v3`) and **`hcloud cts` CLI**.

> **UX Compliance:** This skill follows the User Experience Specification. All operations include onboarding guidance, minimal prompts, smart defaults, clear feedback, and user-friendly error handling.

### CLI Applicability (repository policy)

- **`cli_applicability: dual-path`** — Official `hcloud cts` CLI supports CTS operations. **MUST** document both SDK and CLI paths.

### Well-Architected + Three-Pillar Integration

This skill integrates Huawei Cloud Well-Architected five pillars plus FinOps, SecOps, and AIOps:
- [Security Assessment](references/well-architected-assessment.md#21)
- [FinOps Cost Optimization](references/well-architected-assessment.md#3)
- [SecOps Security Operations](references/well-architected-assessment.md#4)
- [AIOps Integration](references/aiops-best-practices.md)

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT triggers with precise keywords, delegation rules to IAM/APIG/OBS skills |
| 2 | **Structured I/O** | `{{env.*}}` for credentials, `{{user.*}}` for trace query inputs, `{{output.*}}` for API responses |
| 3 | **Explicit Steps** | Every operation: Pre-flight → Execute → Validate → Recover with numbered imperative steps |
| 4 | **Failure Strategies** | 12+ CTS-specific error codes with HALT vs retry distinction |
| 5 | **Single Responsibility** | CTS audit/trace only; delegates IAM to `huaweicloud-iam-ops`, log storage to `huaweicloud-obs-ops`, alarm to `huaweicloud-ces-ops` |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud CTS", "云审计", "云追踪", "审计日志", "Trace Service", "审计轨迹"
- Task involves trail lifecycle: create, describe, modify, delete, list
- Task involves trace query: search audit events, filter by user/resource/action/time
- Task involves trail delivery: configure event storage to OBS/SMN/Log Tank Service
- Task involves compliance or security investigation: access history, resource changes, policy audit
- Task keywords: `audit`, `trace`, `trail`, `event`, `compliance`, `forensic`, `change history`

### SHOULD NOT Use This Skill When

- Task is purely billing / cost analysis → delegate to: `huaweicloud-billing-ops`
- Task is IAM permission model only → delegate to: `huaweicloud-iam-ops` (when present)
- Task is OBS bucket/object management → delegate to: `huaweicloud-obs-ops` (when present)
- Task is API Gateway configuration → delegate to: `huaweicloud-apig-ops` (when present)
- Task is real-time application performance monitoring → delegate to: `huaweicloud-ces-ops`

### Delegation Rules

- CTS trails deliver events to OBS/SMN/LTS → delegate destination creation to respective skills
- Compliance investigation often needs IAM audit context → delegate IAM policy lookups to `huaweicloud-iam-ops`
- Event analysis with performance impact → delegate metrics to `huaweicloud-ces-ops`
- Resource delete/change correlation → use `huaweicloud-rds-ops` / `huaweicloud-ecs-ops` for affected resources

## Variable Convention (Agent-Readable)

| Placeholder | Meaning | Agent Action |
|-------------|---------|--------------|
| `{{env.HW_ACCESS_KEY_ID}}` | From runtime environment | NEVER ask user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | From runtime environment | NEVER ask user; fail if unset |
| `{{env.HW_REGION_ID}}` | Default region (e.g., `cn-north-4`) | Use if skill allows |
| `{{env.HW_PROJECT_ID}}` | Project ID | Use for scoped operations |
| `{{user.trail_name}}` | User-supplied trail name | Ask once; reuse |
| `{{user.query_start_time}}` | Query start time | Ask once in ISO format |
| `{{user.query_end_time}}` | Query end time | Ask once in ISO format |
| `{{user.query_filter}}` | Query filter expression | Ask once; suggest common filters |
| `{{user.destination_type}}` | Audit destination type | Ask with options OBS/SMN/LTS |
| `{{output.trail_id}}` | From create response | Parse per API response |
| `{{output.query_result_count}}` | From query response | Parse per API response |

> **`{{env.*}}` MUST NOT** be collected from user. **Credential masking is MANDATORY** — never echo `HW_SECRET_ACCESS_KEY`.

## Quick Start

### What This Skill Does
Manage Huawei Cloud CTS audit trails and trace queries: create trails, query audit events, inspect access history, and troubleshoot audit delivery.

### Prerequisites

## API and Response Conventions

- **OpenAPI canonical**: `https://support.huaweicloud.com/api-cts/`
- **Trail resource**: audit trail configuration with delivery destination, status, and retention
- **Query resource**: search audit events by time range, user, action, resource, and result
- **Pagination**: `limit` + `marker` or `offset` + `limit` depending on API version
- **Idempotency**: Trail name unique within project; duplicate names return `Cts.0401`

## Expected State Transitions

| Operation | Initial State | Target State | Poll API | Max Wait |
|-----------|--------------|--------------|----------|----------|
| Create trail | — | `ACTIVE` | `ShowTrail` | 60s |
| Update trail | `ACTIVE` | `ACTIVE` | `ShowTrail` | 30s |
| Delete trail | `ACTIVE` | absent | `ShowTrail` 404 | 30s |
| Query events | — | results returned | — | 30s |

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|-----------|-------------|------------|------------|
| CreateTrail | Create an audit trail | Medium | Low |
| DescribeTrail | View trail details | Low | None |
| ListTrails | List all trails | Low | None |
| UpdateTrail | Modify trail delivery | Medium | Medium |
| DeleteTrail | Delete a trail | Low | High |
| QueryEvents | Search audit events | Medium | None |
| ShowEvent | View single event detail | Low | None |
| ManageDestination | Configure OBS/SMN/LTS destination | Medium | Medium |
| LoggingHealth | Validate delivery health | Medium | Medium |

## Execution Flows

### Operation: Create Audit Trail

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| Credentials valid | `hcloud cts list-trails` | Non-401 response | HALT — configure credentials |
| Delivery destination exists | Verify OBS/SMN/LTS target | Destination reachable | Create destination first |
| Trail name unique | `ListTrails` filter | Name not found | Ask for unique trail name |
| Quota sufficient | `ShowQuota` or API quota | Quota available | HALT — request quota increase |

#### Execution — CLI (Primary Path)

```bash
hcloud cts create-trail \
  --region {{env.HW_REGION_ID}} \
  --name "{{user.trail_name}}" \
  --delivery-to "{{user.destination_type}}" \
  --delivery-config "{{user.destination_config}}" \
  --log-file-prefix "cts/{{user.trail_name}}" \
  --retention-days 365
```

#### Execution — JIT Go SDK (Fallback Path)

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    cts "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cts/v3"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cts/v3/model"
    ctsregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cts/v3/region"
)

func main() {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    regionID := os.Getenv("HW_REGION_ID")

    client := cts.NewCtsClient(
        cts.CtsClientBuilder().
            WithRegion(ctsregion.ValueOf(regionID)).
            WithCredential(basic.NewCredentialsBuilder().WithAk(ak).WithSk(sk).Build()).
            Build())

    deliveryConfig := &model.TrailDeliveryConfig{
        // Fill destination-specific config
    }

    request := &model.CreateTrailRequest{
        Body: &model.CreateTrailRequestBody{
            Name:           os.Getenv("TRAIL_NAME"),
            DeliveryConfig: deliveryConfig,
            RetentionDays:  func() *int32 { v := int32(365); return &v }(),
        },
    }

    response, err := client.CreateTrail(context.TODO(), request)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Create trail failed: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("Trail ID: %s\n", *response.TrailId)
    fmt.Printf("Trail Name: %s\n", *response.Name)
    fmt.Printf("Status: %s\n", *response.Status)
}
```

#### Post-execution Validation

1. Extract `{{output.trail_id}}` from create response.
2. Poll `ShowTrail(trail_id)` until `status` is `ACTIVE`.
3. Validate destination delivery health if supported.
4. Report trail ID, destination type, and status.

#### Failure Recovery

| Error | Max Retries | Agent Action | UX Feedback |
|-------|-------------|--------------|-------------|
| `Cts.0401` | 0 | HALT | `[ERROR] Trail name already exists. Choose a different name.` |
| `Cts.0402` | 0 | HALT | `[ERROR] Invalid destination configuration. Verify OBS/SMN/LTS settings.` |
| `Cts.0403` | 0 | HALT | `[ERROR] Region not supported for CTS. Use a supported region.` |
| `Cts.0404` | 0 | HALT | `[ERROR] Delivery destination not reachable. Validate target service access.` |
| `Cts.0405` | 0 | HALT | `[ERROR] Quota exceeded. Delete unused trails or request quota increase.` |
| `Cts.0406` | 0 | HALT | `[ERROR] Insufficient balance. Recharge your Huawei Cloud account.` |
| `Cts.0407` | 0 | HALT | `[ERROR] Invalid audit filter. Check the filter expression syntax.` |
| `Cts.0420` | 0 | HALT | `[ERROR] Trail status invalid. Retry after verifying trail configuration.` |
| Throttling 429 | 3 | Exponential backoff | `⚠️ Rate limited. Retrying in {backoff}s...` |
| InternalError 500 | 3 | Backoff 2s→4s→8s | `[ERROR] Server error. Retry or escalate with RequestId.` |

### Operation: Query Audit Events

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| Trail exists | `ShowTrail` | Trail found | Create trail first |
| Time range valid | `{{user.query_start_time}}` / `{{user.query_end_time}}` | Start < End | Ask corrected time range |
| Filter syntax valid | Query parse | No error | Ask corrected filter |

#### Execution — CLI

```bash
hcloud cts query-events \
  --region {{env.HW_REGION_ID}} \
  --trail-id "{{output.trail_id}}" \
  --start-time "{{user.query_start_time}}" \
  --end-time "{{user.query_end_time}}" \
  --filter "{{user.query_filter}}" \
  --limit 100
```

#### Execution — JIT Go SDK

```go
request := &model.QueryEventsRequest{
    TrailId:   os.Getenv("TRAIL_ID"),
    StartTime: os.Getenv("QUERY_START_TIME"),
    EndTime:   os.Getenv("QUERY_END_TIME"),
    Filter:    func() *string { v := os.Getenv("QUERY_FILTER"); return &v }(),
    Limit:     func() *int32 { v := int32(100); return &v }(),
}
response, err := client.QueryEvents(context.TODO(), request)
```

#### Validation

1. Check response code is success.
2. Verify `{{output.query_result_count}}` is parsed from result count.
3. If zero results, suggest broader filter or longer time range.

### Operation: Delete Trail

#### Pre-flight (Safety Gate)

- **MUST** obtain explicit confirmation: `Delete CTS trail {{user.trail_name}} ({{output.trail_id}})?`
- **MUST NOT** proceed without clear user assent
- **MUST** warn: deleting a trail may remove audit delivery configuration and stop future event collection
- **SHOULD** check whether downstream OBS/SMN/LTS delivery needs to be archived

#### Execution

```bash
hcloud cts delete-trail \
  --region {{env.HW_REGION_ID}} \
  --trail-id "{{output.trail_id}}"
```

#### Validation

Verify `ShowTrail(trail_id)` returns 404 Not Found.

## Prerequisites

> Full installation scripts (KooCLI + Go runtime + Credentials): see [references/common-prerequisites.md](../references/common-prerequisites.md)

## Quality Gate (GCL)

This skill is **GCL-recommended** (per `AGENTS.md` §8). Every CTS mutating operation — audit trail (tracker) create / delete / update — runs through the **Generator-Critic-Loop** before its result is returned. Read-only event query and list operations are GCL-**exempt**.

| Field | Value |
|-------|-------|
| Rubric version | v1 (Phase 3, 2026-06-04) |
| `max_iter` | **3** |
| Rubric instance | [`references/rubric.md`](references/rubric.md) |
| Prompt templates | [`references/prompt-templates.md`](references/prompt-templates.md) |
| Trace path | `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` |
| Independence | Generator and Critic in **isolated** sub-agent / session contexts |

### Five-Dimension Rubric (summary)

| # | Dimension | Threshold | Notes |
|---|-----------|-----------|-------|
| 1 | Correctness | ≥ 0.5 | `ShowTracker` / `ListTraces` post-state |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | S1–S8 in rubric §2 |
| 3 | Idempotency | ≥ 0.5 | Pre-check before create |
| 4 | Traceability | ≥ 0.5 | Credential MUST be `<masked>` |
| 5 | Spec Compliance | ≥ 0.5 | Tracker type / OBS bucket / retention / log file validation |

### Per-Operation Safety Anchors (binding)

- **S1** — `delete-tracker` without explicit user confirmation quoting the tracker name
- **S2** — `delete-tracker` when it is the ONLY active tracker for the project (audit loss)
- **S3** — `update-tracker` (disable/stop) the only tracker for a compliance-mandated project
- **S4** — `create-tracker` / `update-tracker` pointing to a non-existent or inaccessible OBS bucket
- **S5** — `update-tracker` with log file validation disabled (tampering risk)
- **S6** — `update-tracker` reducing retention period below compliance minimum (< 180 days)
- **S7** — any trace contains `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / password plaintext
- **S8** — `delete-tracker` while it is actively used by CTS-dependent compliance workflows

### Termination Contract (per `AGENTS.md` §5)

| Condition | Status | Returned |
|-----------|--------|----------|
| All dimensions pass | **PASS** | Generator result + scores + trace path |
| `iter == max_iter` (3) and any dim < threshold | **MAX_ITER** | best-so-far + unresolved rubric items |
| `Safety == 0` | **SAFETY_FAIL** | violated S-rule id; **never** return partial |

### Trace Persistence (mandatory)

Every GCL run writes `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` (schema in `references/prompt-templates.md` §3). Trace is **append-only**; sanitize secrets before write. The path `./audit-results/` is in root `.gitignore`.

### See also

- [`references/rubric.md`](references/rubric.md) — full rubric, S1–S8 rules, per-op thresholds
- [`references/prompt-templates.md`](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons
- Repository root [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8 — GCL specification

## Reference Directory

- [Core Concepts](references/core-concepts.md) — CTS architecture, limits, delivery targets
- [API & SDK Usage](references/api-sdk-usage.md) — Operation map, request/response snippets
- [CLI Usage](references/cli-usage.md) — CLI command map, coverage gap table
- [Troubleshooting Guide](references/troubleshooting.md) — Error codes, diagnostic flows
- [Monitoring & Alerts](references/monitoring.md) — trace health metrics, log review patterns
- [Integration](references/integration.md) — cross-skill delegation, IAM requirements
- [Well-Architected Assessment](references/well-architected-assessment.md) — Five pillars + FinOps + SecOps + AIOps
- [GCL Rubric](references/rubric.md) — Adversarial quality gate (v1, 5-dim, S1–S8 CTS-specific Safety rules)
- [GCL Prompt Templates](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons

## Well-Architected + Three-Pillar Assessment

This skill's operations are evaluated against Huawei Cloud's Well-Architected Framework five pillars plus FinOps, SecOps, and AIOps integration:
- [Security Assessment](references/well-architected-assessment.md#21)
- [Stability Assessment](references/well-architected-assessment.md#22)
- [Cost Assessment](references/well-architected-assessment.md#23)
- [Efficiency Assessment](references/well-architected-assessment.md#24)
- [Performance Assessment](references/well-architected-assessment.md#25)
- [FinOps Integration](references/well-architected-assessment.md#3)
- [SecOps Integration](references/well-architected-assessment.md#4)
- [AIOps Integration](references/aiops-best-practices.md)
