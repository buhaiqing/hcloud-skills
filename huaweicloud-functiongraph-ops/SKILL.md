---
name: huaweicloud-functiongraph-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei
  Cloud FunctionGraph — serverless function lifecycle, triggers, versioning,
  and diagnostics. User mentions FunctionGraph, 函数工作流, 函数, 云函数,
  serverless function, or describes scenarios (function execution timeout,
  invocation errors, trigger misconfiguration, performance degradation) even
  without naming FunctionGraph directly. Not for CCE/ECS compute, API Gateway
  configuration, or EventGrid event sources that have their own ops skills.
license: MIT
compatibility: >-
  Go 1.21+ runtime for JIT SDK fallback via huaweicloud-sdk-go-v3, valid AK/SK
  credentials, network access to Huawei Cloud endpoints. FunctionGraph does NOT
  have native `hcloud` CLI support — SDK-only execution path.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-05-20"
  runtime: Harness AI Agent, Claude Code, Cursor, or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  api_profile: "https://support.huaweicloud.com/api-functiongraph/"
  cli_applicability: "sdk-only"
  cli_support_evidence: >-
    FunctionGraph is NOT directly supported by `hcloud` CLI. All operations
    use JIT Go SDK via huaweicloud-sdk-go-v3/services/functiongraph/v2.
    CLI can be used indirectly for supporting tasks (e.g., OBS upload via
    `hcloud obs` for code package staging).
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
  gcl:
    enabled: true
    required: true
    rubric_version: "v1"
    max_iter: 2
    rubric_ref: "references/rubric.md"
    prompts_ref: "references/prompt-templates.md"
    trace_dir: "./audit-results/"
    changelog:
      - version: "1.1.0"
        date: "2026-06-04"
        change: "GCL Phase 2 rollout: added references/rubric.md (v1, 5-dim, S1–S17 FunctionGraph-specific Safety rules, including active-trigger guard / $LATEST deploy / destructive inline code / env var secret / SDK-only path) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
      - version: "1.0.0"
        date: "2026-05-20"
        change: "Initial skill release."
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud FunctionGraph Operations Skill

## Overview

Huawei Cloud FunctionGraph (函数工作流) provides serverless function computing: event-driven, auto-scaling, pay-per-execution. This skill is an **operational runbook** for agents: function lifecycle management, code deployment, trigger configuration, version/alias management, invocation monitoring, response validation, and failure recovery. **SDK-only execution**: JIT Go SDK (`huaweicloud-sdk-go-v3/services/functiongraph/v2`).

> **UX Compliance:** This skill follows the User Experience Specification. All operations include onboarding guidance, minimal prompts, smart defaults, clear feedback, and user-friendly error handling.

### CLI Applicability (repository policy)

- **`cli_applicability: sdk-only`** — Official `hcloud` CLI does NOT expose FunctionGraph commands. All operations use JIT Go SDK. Supporting tasks (OBS code upload) may delegate to `huaweicloud-obs-ops` when available.

### Well-Architected + Three-Pillar Integration

This skill integrates Huawei Cloud Well-Architected five pillars plus FinOps, SecOps, and AIOps:
- [Security Assessment](references/well-architected-assessment.md#21)
- [FinOps Cost Optimization](references/well-architected-assessment.md#3)
- [SecOps Security Operations](references/well-architected-assessment.md#4)
- [AIOps Integration](references/advanced/aiops-best-practices.md)

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT triggers with precise keywords, delegation rules to OBS/APIG/CES skills |
| 2 | **Structured I/O** | `{{env.*}}` for credentials, `{{user.*}}` for function config, `{{output.*}}` for API responses |
| 3 | **Explicit Steps** | Every operation: Pre-flight → Execute → Validate → Recover with numbered imperative steps |
| 4 | **Failure Strategies** | 12+ FunctionGraph-specific error codes with HALT vs retry distinction |
| 5 | **Single Responsibility** | Function lifecycle only; delegates OBS to `huaweicloud-obs-ops`, API Gateway to `huaweicloud-apig-ops`, monitoring to `huaweicloud-ces-ops` |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud FunctionGraph", "函数工作流", "云函数", "函数计算", "FunctionGraph"
- Task involves function lifecycle: create, deploy, invoke, update, delete, list, describe
- Task involves trigger management: APIG, OBS, SMN, LTS, Timer, CTS, Kafka, DMS triggers
- Task involves version/alias: publish version, create alias, traffic distribution
- Task involves function config: runtime, handler, timeout, memory, environment variables, VPC access
- Task involves monitoring/invocation: execution logs, metrics, async invocation, error tracking
- Task keywords: `function`, `函数`, `trigger`, `触发器`, `handler`, `runtime`, `serverless`, `FaaS`

### SHOULD NOT Use This Skill When

- Task is purely billing / cost analysis → delegate to: `huaweicloud-billing-ops` (when present)
- Task is IAM permission model only → delegate to: `huaweicloud-iam-ops` (when present)
- Task is OBS bucket/object management → delegate to: `huaweicloud-obs-ops` (when present)
- Task is API Gateway (APIG) configuration → delegate to: `huaweicloud-apig-ops` (when present)
- Task is CCE/ECS compute management → delegate to: `huaweicloud-cce-ops` / `huaweicloud-ecs-ops`
- Task is SMN topic/subscription management → delegate to: `huaweicloud-smn-ops` (when present)

### Delegation Rules

- Function code stored in OBS → delegate OBS upload to `huaweicloud-obs-ops` before function create/update
- APIG trigger → create function first, then delegate APIG configuration to `huaweicloud-apig-ops`
- CES alarm on function errors → create function, then delegate alarm to `huaweicloud-ces-ops`
- LTS log query → function execution logs accessible via this skill; advanced analytics delegate to `huaweicloud-lts-ops` (when present)

## Variable Convention

| Placeholder | Meaning | Agent Action |
|-------------|---------|--------------|
| `{{env.HW_ACCESS_KEY_ID}}` | From runtime environment | NEVER ask user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | From runtime environment | NEVER ask user; fail if unset |
| `{{env.HW_REGION_ID}}` | Default region (e.g., `cn-north-4`) | Use if skill allows |
| `{{env.HW_PROJECT_ID}}` | Project ID | Use for scoped operations |
| `{{user.function_name}}` | User-supplied function name | Ask once; reuse |
| `{{user.runtime}}` | Runtime (e.g., `Python3.9`, `Node.js16.17`) | Suggest available runtimes |
| `{{user.handler}}` | Entry point (e.g., `index.handler`) | Ask with runtime-appropriate defaults |
| `{{user.timeout}}` | Timeout in seconds | Default 30s |
| `{{user.memory_size}}` | Memory in MB | Default 256, step 128 |
| `{{user.code_url}}` | OBS code package URL | Ask or delegate upload to OBS skill |
| `{{output.function_urn}}` | From create response | Parse per OpenAPI path |
| `{{output.invocation_id}}` | From invoke response | Parse for async execution tracking |

> **`{{env.*}}` MUST NOT** be collected from user. **Credential masking is MANDATORY** — never echo `HW_SECRET_ACCESS_KEY`.

## Quick Start

### What This Skill Does
Manage Huawei Cloud FunctionGraph serverless functions: create, deploy code, configure triggers, invoke, monitor, and troubleshoot.

### Prerequisites
- [ ] Go 1.21+ runtime (for JIT SDK fallback)
- [ ] Credentials: `HW_ACCESS_KEY_ID`, `HW_SECRET_ACCESS_KEY`
- [ ] Region: `HW_REGION_ID` (e.g., `cn-north-4`)
- [ ] Project ID: `HW_PROJECT_ID`

### Verify Setup
```bash
# SDK verification — list existing functions
go run ./main.go  # ListFunctions query
```

### Your First Command
```bash
# List all functions (JIT Go SDK)
go run -exec "go run /tmp/fg-script/main.go"  # ListFunctions
```

### Next Steps
- [Core Concepts](references/core-concepts.md) — Understand FunctionGraph architecture
- [Function Operations](#execution-flows) — Create, deploy, invoke, manage
- [Trigger Management](#operation-manage-triggers) — Configure event sources
- [Troubleshooting](references/troubleshooting.md) — Fix common function issues

## API and Response Conventions

- **OpenAPI canonical**: `https://support.huaweicloud.com/api-functiongraph/`
- **Async pattern**: Some operations (async invoke) return task_id — poll via `ListAsyncInvocations`
- **Function URN**: Unique Resource Name format `urn:fss:{region}:{project_id}:function:{name}:{version}`
- **Code upload**: Via OBS URL (`code_url`) or direct ZIP upload (≤ 10MB inline, larger via OBS)
- **Pagination**: Use `marker` + `max_items`, default 50 per page
- **Idempotency**: Function name uniqueness within project; duplicate name returns `FSS.0401`

## Expected State Transitions

| Operation | Initial State | Target State | Poll API | Max Wait |
|-----------|--------------|--------------|----------|----------|
| Create | — | `Active` | `ShowFunctionConfig` | 60s |
| UpdateCode | `Active` | `Active` | `ShowFunctionCode` | 30s |
| UpdateConfig | `Active` | `Active` | `ShowFunctionConfig` | 30s |
| Invoke (sync) | `Active` | — | Response body | timeout+10s |
| Invoke (async) | `Active` | `Success`/`Fail` | `ListAsyncInvocations` | 300s |
| Delete | any | absent | `ShowFunctionConfig` 404 | 30s |
| PublishVersion | `Active` | `Active` | `ShowVersion` | 30s |

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|-----------|-------------|------------|------------|
| Create | Create a new function | Medium | Low |
| DeployCode | Update function code | Low | Medium — may break running |
| UpdateConfig | Change function configuration | Low | Medium — may affect behavior |
| Invoke | Execute function (sync/async) | Low | Low |
| List | View all functions | Low | None |
| Describe | View function details | Low | None |
| Delete | Remove a function | Low | **High** — irreversible |
| ManageTriggers | Create/list/delete triggers | Medium | Medium |
| PublishVersion | Create version/alias | Low | Low |

## Execution Flows

### Operation: Create Function

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| Credentials valid | SDK ShowFunctionConfig | Non-401 response | HALT — user configures credentials |
| Function name unique | SDK list + name check | Name not taken | Suggest unique name |
| Runtime valid | Check supported runtimes list | Runtime in list | List available runtimes |
| Code package ready | Check OBS URL exists or ZIP available | File accessible | Upload code package to OBS first |
| VPC config (if needed) | VPC/subnet IDs valid | Valid VPC+subnet | Create via VPC skill first |
| Quota sufficient | SDK ListQuotas | Quota > 0 | HALT — request quota increase |

#### Execution — JIT Go SDK (Primary Path)

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    fg "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/functiongraph/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/functiongraph/v2/model"
    fgregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/functiongraph/v2/region"
)

func main() {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    regionID := os.Getenv("HW_REGION_ID")
    
    client := fg.NewFunctionGraphClient(
        fg.FunctionGraphClientBuilder().
            WithRegion(fgregion.ValueOf(regionID)).
            WithCredential(basic.NewCredentialsBuilder().WithAk(ak).WithSk(sk).Build()).
            Build())
    
    // Create function
    codeType := model.GetCreateFunctionRequestBodyCodeTypeEnum().OBS
    runtime := model.GetCreateFunctionRequestBodyRuntimeEnum().PYTHON_3_9
    
    request := &model.CreateFunctionRequest{
        Body: &model.CreateFunctionRequestBody{
            FunctionName: os.Getenv("FUNCTION_NAME"),
            Runtime:      runtime,
            Handler:      os.Getenv("FUNCTION_HANDLER"),
            CodeType:     codeType,
            CodeUrl:      func() *string { v := os.Getenv("CODE_URL"); return &v }(),
            Timeout:      30,
            MemorySize:   func() *int32 { v := int32(256); return &v }(),
        },
    }
    
    response, err := client.CreateFunction(context.TODO(), request)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Create function failed: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Printf("Function URN: %s\n", *response.FuncUrn)
    fmt.Printf("Function Name: %s\n", *response.FuncName)
    fmt.Printf("Runtime: %s\n", *response.Runtime)
    fmt.Printf("State: %s\n", response.State)
}
```

#### Post-execution Validation

1. Extract `{{output.function_urn}}` from create response (`func_urn` field).
2. Verify function state via `ShowFunctionConfig` — expect `Active`.
3. Test invoke function with simple event to confirm readiness.
4. Report `{{output.function_urn}}`, runtime, and handler to user.

#### Failure Recovery

| Error | Max Retries | Agent Action | UX Feedback |
|-------|-------------|--------------|-------------|
| `FSS.0101` InvalidParameter | 0 | HALT | `[ERROR] Invalid parameter. Verify function name, runtime, handler against API docs.` |
| `FSS.0102` FunctionNameAlreadyExists | 0 | HALT | `[ERROR] Function name already exists. Choose unique name.` |
| `FSS.0103` CodePackageInvalid | 0 | HALT | `[ERROR] Code package invalid. Verify OBS URL or ZIP format.` |
| `FSS.0104` RuntimeNotSupported | 0 | HALT | `[ERROR] Runtime not supported. List available runtimes.` |
| `FSS.0201` QuotaExceeded | 0 | HALT | `[ERROR] Function quota exceeded. Delete unused functions or request increase.` |
| `FSS.0202` InsufficientBalance | 0 | HALT | `[ERROR] Insufficient balance. Recharge Huawei Cloud account.` |
| `FSS.0301` VpcConfigInvalid | 0 | HALT | `[ERROR] VPC configuration invalid. Verify VPC/subnet IDs.` |
| `FSS.0401` ResourceNotFound | 0 | HALT | `[ERROR] Resource not found. Verify function URN or resource ID.` |
| `FSS.0501` TriggerConflict | 0 | HALT | `[ERROR] Trigger already exists for this event source.` |
| `FSS.0502` TriggerNotSupported | 0 | HALT | `[ERROR] Trigger type not supported for this runtime.` |
| Throttling 429 | 3 | Exponential backoff | `⚠️ Rate limited. Retrying in {backoff}s...` |
| InternalError 500 | 3 | Backoff 2s→4s→8s | `[ERROR] Server error. Retry or escalate with RequestId.` |

### Operation: Deploy Function Code

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| Function exists | `ShowFunctionConfig` | Response OK | Create function first |
| Code package ready | Check code URL | File exists & accessible | Upload to OBS first |
| Code size within limit | Check code package size | ≤ 30MB (direct) or ≤ 10GB (OBS) | Optimize or use OBS |

#### Execution — JIT Go SDK

```go
request := &model.UpdateFunctionCodeRequest{
    FunctionUrn: os.Getenv("FUNCTION_URN"),
    Body: &model.UpdateFunctionCodeRequestBody{
        CodeType: model.GetUpdateFunctionCodeRequestBodyCodeTypeEnum().OBS,
        CodeUrl:  func() *string { v := os.Getenv("CODE_URL"); return &v }(),
    },
}
response, err := client.UpdateFunctionCode(context.TODO(), request)
```

#### Validation

1. Verify `state` is `Active`.
2. Check `func_code.size` matches expected code package size.
3. Test invoke to confirm code executes correctly.

### Operation: Invoke Function

#### Sync Invoke

```go
request := &model.InvokeFunctionRequest{
    FunctionUrn: os.Getenv("FUNCTION_URN"),
    Body: func() map[string]interface{} {
        return map[string]interface{}{
            "key": "value", // Event payload per function contract
        }
    }(),
}
response, err := client.InvokeFunction(context.TODO(), request)
```

**Validation**: Check response status code (200 = success). Parse response body for function result.

#### Async Invoke

```go
invokeType := model.GetInvokeFunctionRequestInvocationTypeEnum().ASYNC
request := &model.InvokeFunctionRequest{
    FunctionUrn:     os.Getenv("FUNCTION_URN"),
    X-Cff-Invocation-Type: "Async",
    Body:            map[string]interface{}{"key": "value"},
}
```

**Validation**: Async returns 202 Accepted. Track via `ListAsyncInvocations` with `request_id`.

### Operation: Delete Function

#### Pre-flight (Safety Gate)

- **MUST** obtain explicit confirmation with function URN: `Delete function {{user.function_name}} ({{output.function_urn}})?`
- **MUST NOT** proceed without clear user assent
- **MUST** remind: this operation permanently removes the function and all its versions/aliases
- **SHOULD** check for active triggers — list and notify user before deletion
- **SHOULD** suggest disabling triggers before deletion

#### Execution — JIT Go SDK

```go
request := &model.DeleteFunctionRequest{
    FunctionUrn: os.Getenv("FUNCTION_URN"),
}
response, err := client.DeleteFunction(context.TODO(), request)
```

#### Validation

Verify via `ShowFunctionConfig` — expect 404 NotFound.

### Operation: Manage Triggers

#### List Triggers

```go
request := &model.ListFunctionTriggersRequest{
    FunctionUrn: os.Getenv("FUNCTION_URN"),
}
response, err := client.ListFunctionTriggers(context.TODO(), request)
```

#### Create Trigger (example: Timer trigger)

```go
request := &model.CreateFunctionTriggerRequest{
    FunctionUrn: os.Getenv("FUNCTION_URN"),
    Body: &model.CreateFunctionTriggerRequestBody{
        TriggerTypeCode: model.GetCreateFunctionTriggerRequestBodyTriggerTypeCodeEnum().TIMER,
        EventTypeCode:   "TimerEvent",
        TriggerStatus:   model.GetCreateFunctionTriggerRequestBodyTriggerStatusEnum().ACTIVE,
        EventData: map[string]string{
            "schedule":     "cron(0 0 * * *)",  // Daily at midnight
            "name":         "daily-trigger",
            "schedule_type": "cron",
        },
    },
}
```

**Supported trigger types**: `TIMER`, `APIG`, `OBS`, `SMN`, `LTS`, `CTS`, `DMS`, `KAFKA`, `RABBITMQ`, `DEDICATEDGATEWAY`

#### Validation

Verify trigger list includes new trigger with status `ACTIVE`.

### Operation: Publish Version / Create Alias

#### Publish Version

```go
request := &model.CreateFunctionVersionRequest{
    FunctionUrn: os.Getenv("FUNCTION_URN"),
    Body: &model.CreateFunctionVersionRequestBody{
        Version:      "v1.0.0",
        Description:  func() *string { v := "Production release"; return &v }(),
    },
}
```

#### Create Alias (with traffic distribution)

```go
request := &model.UpdateAliasRequest{
    FunctionUrn: os.Getenv("FUNCTION_URN"),
    AliasName:   "prod",
    Body: &model.UpdateAliasRequestBody{
        Name:         "prod",
        Version:      "v1.0.0",
        AdditionalVersionStrategy: map[string]int32{
            "v2.0.0": 10, // 10% traffic to v2.0.0
        },
    },
}
```

## Prerequisites

1. **Bootstrap Go runtime** (JIT SDK):

    ```bash
    if ! command -v go &> /dev/null; then
        OS=$(uname -s | tr '[:upper:]' '[:lower:]')
        ARCH=$(uname -m)
        [ "$ARCH" = "x86_64" ] && ARCH="amd64"
        [ "$ARCH" = "aarch64" ] && ARCH="arm64"
        mkdir -p /tmp/go-runtime
        curl -fsSL "https://go.dev/dl/go1.25.0.${OS}-${ARCH}.tar.gz" | tar -xz -C /tmp/go-runtime
        export PATH="/tmp/go-runtime/go/bin:$PATH"
        export GOPATH="/tmp/go-workspace"
        export GOPROXY="https://goproxy.cn,direct"
    fi
    ```

2. **JIT Go SDK Workflow**:

    ```bash
    mkdir -p /tmp/fg-sdk-workspace && cd /tmp/fg-sdk-workspace
    go mod init fg-script
    go get github.com/huaweicloud/huaweicloud-sdk-go-v3/services/functiongraph/v2
    ```

3. **Configure Credentials**:

    ```bash
    export HW_ACCESS_KEY_ID="{{env.HW_ACCESS_KEY_ID}}"
    export HW_SECRET_ACCESS_KEY="{{env.HW_SECRET_ACCESS_KEY}}"
    export HW_REGION_ID="{{env.HW_REGION_ID}}"
    export HW_PROJECT_ID="{{env.HW_PROJECT_ID}}"
    test -n "$HW_SECRET_ACCESS_KEY" && echo "✅ Credentials configured"
    ```

## Quality Gate (GCL)

This skill is **GCL-required** (per `AGENTS.md` §8). Every FunctionGraph mutating operation —
function create / delete / code deploy / invoke, version publish / delete, alias create /
delete, trigger create / enable / disable / delete, config update — runs through the
**Generator-Critic-Loop** before its result is returned. Read-only are GCL-**exempt**.

> **Path note**: FunctionGraph is `cli_applicability: sdk-only` — there is no `hcloud
> functiongraph` command group. All Generator operations go through JIT Go SDK
> (`huaweicloud-sdk-go-v3/services/functiongraph/v2`).

| Field | Value |
|-------|-------|
| Rubric version | v1 (Phase 2, 2026-06-04) |
| `max_iter` | **2** |
| Rubric instance | [`references/rubric.md`](references/rubric.md) |
| Prompt templates | [`references/prompt-templates.md`](references/prompt-templates.md) |
| Trace path | `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` |
| Independence | Generator and Critic in **isolated** sub-agent / session contexts |

### Five-Dimension Rubric (summary)

| # | Dimension | Threshold | Notes |
|---|-----------|-----------|-------|
| 1 | Correctness | ≥ 0.5 (1.0 for `delete-function` / `delete-version` / `disable-trigger`) | `ShowFunctionConfig` / `ListTriggers` / `ShowAlias` post-state |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | S1–S17 in rubric §2 |
| 3 | Idempotency | ≥ 0.5 | Pre-check before create |
| 4 | Traceability | ≥ 0.5 | `password` / env var secret MUST be `<masked>`; `code_sha256` only (not content) |
| 5 | Spec Compliance | ≥ 0.5 | Runtime / memory (≤3008MB) / timeout (≤900s) / name regex |

### Per-Operation Safety Anchors (binding)

- **S1 / S2 / S3** — `delete-function` confirmation / active triggers / alias with `additional_version_weights > 0` on `$LATEST`
- **S4** — `delete-version` while version referenced by alias
- **S5 / S6** — `disable-trigger` / `delete-trigger` while `status == ACTIVE` (live traffic cut)
- **S7 / S8** — `deploy-function-code` to `$LATEST` with alias traffic / destructive inline shell
- **S9 / S10** — `memory > 3008` MB / `timeout > 900` s (API limits)
- **S11** — env var with `*SECRET*` / `*PASSWORD*` plaintext (anti-pattern; suggest KMS)
- **S13** — unsupported runtime (Node.js 14.18 / 16.17 / 18.15, Python 3.9-3.11, Java 8/11/17, Go 1.x)
- **S15** — invoke payload > 6 MB (sync) or > 50 MB (async)
- **S16** — TIMER cron more frequent than every 1 minute
- **S17** — memory decrease without warning (cold-start risk)

### Termination Contract (per `AGENTS.md` §5)

| Condition | Status | Returned |
|-----------|--------|----------|
| All dimensions pass | **PASS** | Generator result + scores + trace path |
| `iter == max_iter` (2) and any dim < threshold | **MAX_ITER** | best-so-far + unresolved rubric items |
| `Safety == 0` | **SAFETY_FAIL** | violated S-rule id; **never** return partial |

### Trace Persistence (mandatory)

Every GCL run writes `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` (schema in
`references/prompt-templates.md` §3). Trace is **append-only**; sanitize secrets before write
(see `prompt-templates.md` §4). The path `./audit-results/` is in root `.gitignore`.

### See also

- [`references/rubric.md`](references/rubric.md) — full rubric, S1–S17 rules, per-op thresholds
- [`references/prompt-templates.md`](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons
- [`references/api-sdk-usage.md`](references/api-sdk-usage.md) — SDK patterns (since SDK-only path)
- Repository root [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8 — GCL specification

## Reference Directory

- [Core Concepts](references/core-concepts.md) — FunctionGraph architecture, limits, quotas
- [API & SDK Usage](references/api-sdk-usage.md) — Operation map, request/response snippets
- [Troubleshooting Guide](references/troubleshooting.md) — Error codes, diagnostic flows
- [Monitoring & Alerts](references/monitoring.md) — CES metrics, dashboards, alarm patterns
- [Integration](references/integration.md) — JIT SDK setup, cross-skill delegation matrix
- [Well-Architected Assessment](references/well-architected-assessment.md) — Five pillars + FinOps + SecOps + AIOps
- [GCL Rubric](references/rubric.md) — Adversarial quality gate (v1, 5-dim, S1–S17 FunctionGraph-specific Safety rules; SDK-only path)
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
- [AIOps Integration](references/advanced/aiops-best-practices.md)
