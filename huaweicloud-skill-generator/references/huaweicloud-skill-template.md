---
name: huaweicloud-[product-name]-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei
  Cloud [Product Name] — [Resource Type] lifecycle, configuration, and
  diagnostics. User mentions [Product Name], [Product Chinese Name],
  or describes product-specific scenarios (e.g., connection
  drops, performance degradation, resource creation failures) even without
  naming the product directly. Not for billing, IAM, or related products that
  have their own ops skills.
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
  api_profile: "[Paste OpenAPI title/version or doc link]"
  cli_applicability: "dual-path"  # Choose: cli-first / dual-path / sdk-only / cli-only
  cli_support_evidence: >-
    [If CLI covers this product: cite confirmation via `hcloud help`.
    If CLI does NOT cover: note JIT Go SDK fallback required.]
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
---

> This template follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud [Product Name] Operations Skill

## Overview

[Product Name] on Huawei Cloud provides [brief description]. This skill is an **operational runbook** for agents: explicit scope, credential rules, pre-flight checks, **dual-path execution** (official **SDK/API** and, when the product is supported by official **`hcloud` CLI**, the matching **CLI** flows), response validation, and failure recovery. **Do not use the web console as the primary agent execution path** in `SKILL.md` or [Huawei Cloud Console](https://console.huaweicloud.com).

### CLI applicability (repository policy)

- **`cli_applicability: cli-first`:** Official CLI fully supports this product. CLI is the **primary** execution path. JIT Go SDK is the **fallback** only for edge-case operations CLI doesn't expose.
- **`cli_applicability: dual-path`:** Official CLI supports this product. You **MUST** ship **`references/cli-usage.md`** and, in **each** execution flow below, document **both** the SDK step **and** the CLI step for every operation.
- **`cli_applicability: sdk-only`:** Official CLI does **not** expose this product. **Omit** `references/cli-usage.md`. SDK/API remains mandatory for all operations.
- **`cli_applicability: cli-only`:** Read-only/discovery skills that ONLY query cloud resources. No write operations.

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions with precise triggers and delegation rules |
| 2 | **Structured I/O** | Placeholder conventions with type and source documented |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute → Validate → Recover |
| 4 | **Complete Failure Strategies** | Error taxonomy ≥ 10 codes; HALT vs retry per error type |
| 5 | **Absolute Single Responsibility** | One product, one resource model; cross-product delegation to other skills |
| 6 | **GCL Adversarial Rubric** | `## Quality Gate (GCL)` chapter with ≥5-dimension rubric; `references/prompt-templates.md` with G + C prompt skeletons (required for `huaweicloud-{ecs,evs,eip,vpc,rds,gaussdb,dcs,dms,css,cce,cbr,iam,obs,swr,functiongraph,waf,hss}-ops`; recommended for `{elb,ces,lts,cts}-ops`). See root `AGENTS.md` §3, §7, §8. |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

In addition to the Five Core Standards, every generated skill MUST integrate three operational pillars:

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **FinOps (财务运营)** | Cost visibility, right-sizing, billing model comparison, waste detection | `references/well-architected-assessment.md` §3 |
| **SecOps (安全运营)** | IAM minimum permissions, network isolation, encryption, threat detection | `references/well-architected-assessment.md` §4 |
| **AIOps (智能运营)** | Multi-metric correlation, cross-skill diagnosis, knowledge base, self-healing | `references/aiops-best-practices.md` |

### Well-Architected Framework Integration (卓越架构)

Every generated skill MUST map operations to Huawei Cloud's Well-Architected Framework five pillars:

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **安全 (Security)** | IAM permissions, credential masking, network isolation | `references/well-architected-assessment.md` §2.1 |
| **稳定 (Stability)** | Backup/restore, multi-AZ, DR runbook, failure-oriented design | `references/well-architected-assessment.md` §2.2 |
| **成本 (Cost)** | Billing model comparison, waste detection, right-sizing | `references/well-architected-assessment.md` §2.3 |
| **效率 (Efficiency)** | Batch operations, CI/CD integration, automation patterns | `references/well-architected-assessment.md` §2.4 |
| **性能 (Performance)** | Metrics, auto-scaling, performance baselines | `references/well-architected-assessment.md` §2.5 |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud [Product Name]" OR "[Product Chinese Name]" OR "[Product Alias]"
- Task involves CRUD or lifecycle operations on **[Resource Type]**
- Task keywords: [keyword1], [keyword2], [keyword3], …
- User asks to deploy, configure, troubleshoot, or monitor [Product Name] **via API, SDK, CLI, or automation**

### SHOULD NOT Use This Skill When

- Task is purely billing / account management → delegate to: `huaweicloud-billing-ops` (when present)
- Task is IAM / permission model only → delegate to: `huaweicloud-iam-ops` (when present)
- Task is about **[related product]** → delegate to: `huaweicloud-[other]-ops`

### Delegation Rules

- If resource B depends on resource A, complete or verify A before B's SDK or CLI steps.
- Multi-product requests: handle each product with its skill; do not merge unrelated APIs into one ambiguous flow.
- For FinOps questions involving this resource: use this skill's cost section, delegate cross-resource cost to billing skill.
- For SecOps questions: use this skill's security section, delegate account-level IAM to IAM skill.

## Variable Convention (Agent-Readable)

| Placeholder | Meaning | Agent Action |
|-------------|---------|--------------|
| `{{env.HW_ACCESS_KEY_ID}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_REGION_ID}}` | From runtime environment | Use documented default only if skill explicitly allows |
| `{{env.HW_PROJECT_ID}}` | From runtime environment | Use documented default only if skill explicitly allows |
| `{{user.region}}` | User-supplied region | Ask once; reuse |
| `{{user.resource_name}}` | User-supplied name | Ask once; reuse |
| `{{output.resource_id}}` | From last API or CLI JSON response | Parse per **OpenAPI** path for this operation |

> **`{{env.*}}` MUST NOT** be collected from the user. **`{{user.*}}`** MUST be collected interactively when missing.

> **Security Warning (Credential Masking — MANDATORY):** **NEVER** log, print, or expose `HW_SECRET_ACCESS_KEY`, `SecretAccessKey`, or any credential field value in console output, debug messages, error messages, or logs.

## API and Response Conventions

- **OpenAPI is canonical** for path, query, body fields, enums, and response shapes.
- **Errors:** Map SDK/HTTP errors to `code` / `status` / message fields per spec.
- **Timestamps:** ISO 8601 with timezone when the API returns strings.
- **Idempotency:** Document client request tokens, duplicate names, and `ResourceAlreadyExists` behavior per API.

## Quick Start

### What This Skill Does
This skill enables deployment, configuration, troubleshooting, and monitoring of Huawei Cloud [Product Name] resources using the `hcloud` CLI (primary) or JIT Go SDK (fallback).

### Prerequisites
- [ ] Huawei Cloud CLI installed (or Go runtime for JIT fallback)
- [ ] Credentials configured: `HW_ACCESS_KEY_ID`, `HW_SECRET_ACCESS_KEY`
- [ ] Region and Project ID set: `HW_REGION_ID`, `HW_PROJECT_ID`

### Verify Setup
```bash
# Check CLI and credentials
hcloud ecs describe-instances --region {{env.HW_REGION_ID}}
```

### Your First Command
```bash
# Example: List resources
hcloud [product] list --region {{env.HW_REGION_ID}}
```

### Next Steps
- [Core Concepts](references/core-concepts.md) — Understand [Product Name] architecture
- [Common Operations](#execution-flows) — Create, manage, and delete resources
- [Troubleshooting](references/troubleshooting.md) — Fix common issues

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|-----------|-------------|------------|------------|
| Create | Create a new [Resource] | Medium | Low |
| Describe | View [Resource] details | Low | None |
| Modify | Change [Resource] configuration | Medium | Medium |
| Delete | Remove a [Resource] | Low | **High** — irreversible |
| List | View all [Resources] | Low | None |

## Execution Flows

### Operation: Create [Resource]

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| CLI / deps | `hcloud --version` | Exit code 0 | Document CLI install |
| Credentials | Construct credential from env | Non-empty keys | HALT; user configures env |
| Region | Call **ListRegions** or equivalent | `{{user.region}}` supported | Suggest valid region |
| Quota | Call quota/describe API per OpenAPI | Sufficient quota | HALT; user raises quota |

#### Execution — CLI (Primary Path)

```bash
# CLI invocation
hcloud [product] create [resource] \
  --region "{{user.region}}" \
  --name "{{user.resource_name}}"
  # Add parameters per official documentation
```

#### Execution — JIT Go SDK (Fallback Path)

When CLI does not support a specific operation, **JIT build a Go SDK script**:

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "[product]" "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/[product]/v2"
    "[product]_model" "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/[product]/v2/model"
)

func main() {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    region := os.Getenv("HW_REGION_ID")
    
    cfg := config.DefaultHttpConfig()
    client := [product].[product]ClientBuilder().
        WithEndpoint(fmt.Sprintf("[product].%s.myhuaweicloud.com", region)).
        WithCredential(basic.NewCredentialsBuilder().
            WithAk(ak).WithSk(sk).Build()).
        WithHttpConfig(cfg).Build()
    
    request := &[product]_model.Create[Resource]Request{
        // Add fields per OpenAPI request schema
    }
    
    response, err := client.Create[Resource](request)
    if err != nil {
        panic(err)
    }
    fmt.Printf("%+v\n", response)
}
```

#### Post-execution Validation

1. Read `{{output.resource_id}}` from the documented response path.
2. Poll **Describe** until terminal success state or timeout.
3. On success, report `{{output.resource_id}}` and key fields.
4. On terminal failure, go to **Failure Recovery**.

#### Failure Recovery

| Error | Max retries | Backoff | Agent Action | UX Feedback |
|-------|-------------|---------|--------------|-------------|
| `InvalidParameter` | 0–1 | — | Fix args from OpenAPI | `[ERROR] InvalidParameter: Check parameters against OpenAPI docs.` |
| `QuotaExceeded` | 0 | — | HALT | `[ERROR] Quota exceeded. Request quota increase or delete unused resources.` |
| `InsufficientBalance` | 0 | — | HALT | `[ERROR] Insufficient balance. Recharge your Huawei Cloud account.` |
| `ResourceAlreadyExists` | 0 | — | Ask reuse vs new name | `[ERROR] Resource already exists. Use different name or reuse existing.` |
| Throttling / 429 | 3 | exponential | Back off; respect Retry-After | `⚠️ Rate limited. Retrying in {backoff}s...` |
| `InternalError` / 5xx | 3 | 2s, 4s, 8s | Retry; then HALT | `[ERROR] Server-side error. Retry or escalate with RequestId.` |

### Operation: Describe [Resource]

#### Execution

```bash
# CLI — list resources
hcloud [product] describe [resource] \
  --region "{{user.region}}" \
  --resource-id "{{user.resource_id}}"
```

### Operation: Delete [Resource]

#### Pre-flight (Safety Gate)

- **MUST** obtain explicit confirmation: irreversible delete of `{{user.resource_name}}` (`{{user.resource_id}}`).
- **MUST NOT** proceed without clear user assent.
- **MUST** remind user to backup before delete if backup is available.

#### Execution

Call delete API per OpenAPI. Capture response indicating success or error per verified output shape.

#### Post-execution Validation

Poll describe/get until **404** or **NotFound** status — per API semantics — within **max wait**.

### Operation: Backup [Resource]

#### When to Use
- Before any destructive operation
- Scheduled per organizational RPO requirements
- Migration or region transfer prerequisites

#### Execution

```bash
# Create backup/snapshot
hcloud [product] create-backup \
  --resource-id "{{user.resource_id}}" \
  --name "auto-backup-$(date +%Y%m%d-%H%M%S)"
```

### Operation: Restore from Backup

#### Pre-flight (Safety Gate)
- **MUST** warn user: restore overwrites current data; suggest pre-restore backup
- **MUST** confirm: target resource, backup source, expected data loss window

## Prerequisites

1. **Install KooCLI** (official binary):

    ```bash
    # Linux one-click install
    curl -sSL https://cn-north-4.myhuaweicloud.com/cli/latest/hcloud_install.sh -o ./hcloud_install.sh && bash ./hcloud_install.sh -y

    # Verify
    hcloud version
    ```

2. **Bootstrap Go runtime** (JIT SDK fallback — required only if CLI doesn't support operation):

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
    ```

4. **Verify Configuration**:
    ```bash
    hcloud ecs describe-instances --region {{env.HW_REGION_ID}}
    ```

## Reference Directory

- [Core Concepts](references/core-concepts.md)
- [API & SDK Usage](references/api-sdk-usage.md)
- [CLI Usage](references/cli-usage.md)
- [Troubleshooting Guide](references/troubleshooting.md)
- [Monitoring & Alerts](references/monitoring.md)
- [Integration](references/integration.md)
- [Knowledge Base](references/knowledge-base.md)
- [Observability Integration](references/observability.md)
- [User Experience Specification](references/user-experience-spec.md)
- [AIOps Best Practices](references/aiops-best-practices.md)
- [FinOps Cost Optimization](references/well-architected-assessment.md#3-finops-)
- [SecOps Security Operations](references/well-architected-assessment.md#4-secops-)
- [Well-Architected Assessment](references/well-architected-assessment.md)

## Well-Architected + Three-Pillar Assessment

This skill's operations are evaluated against Huawei Cloud's Well-Architected Framework (卓越架构) five pillars plus FinOps, SecOps, and AIOps integration:
- [Security Assessment](references/well-architected-assessment.md#21-安全支柱-security)
- [Stability Assessment](references/well-architected-assessment.md#22-稳定支柱-stability)
- [Cost Assessment](references/well-architected-assessment.md#23-成本支柱-cost)
- [Efficiency Assessment](references/well-architected-assessment.md#24-效率支柱-efficiency)
- [Performance Assessment](references/well-architected-assessment.md#25-性能支柱-performance)
- [FinOps Integration](references/well-architected-assessment.md#3-finops-)
- [SecOps Integration](references/well-architected-assessment.md#4-secops-)
- [AIOps Integration](references/aiops-best-practices.md)
