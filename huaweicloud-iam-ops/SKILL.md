---
name: huaweicloud-iam-ops
description: >-
  Use when the user needs to manage Huawei Cloud IAM (Identity and Access Management) —
  users, groups, policies, roles, agencies, credentials, projects, and federation.
  User mentions IAM, 身份认证, 权限管理, 访问控制, 用户组, 策略, 委托, AK/SK, MFA,
  or describes identity/permission-related scenarios (e.g., cannot access resource,
  permission denied, create access key, configure MFA, assign policies, set up agency)
  even without naming the product directly. Not for billing, VPC, or related products
  that have their own ops skills.
license: MIT
compatibility: >-
  Official Huawei Cloud CLI (`hcloud`), Go 1.21+ runtime
  (for JIT SDK fallback via huaweicloud-sdk-go-v3), valid API credentials,
  network access to Huawei Cloud IAM global endpoint.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-05-20"
  runtime: Harness AI Agent, Claude Code, Cursor, or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  api_profile: "IAM API v3 - https://support.huaweicloud.com/api-iam/"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    hcloud supports IAM operations via `hcloud iam` command group.
    JIT Go SDK fallback available via huaweicloud-sdk-go-v3/services/iam/v3
    for operations not exposed in CLI or for complex batch operations.
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
        change: "GCL Phase 2 rollout: added references/rubric.md (v1, 5-dim, S1–S14 IAM-specific Safety rules) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
      - version: "1.0.0"
        date: "2026-05-20"
        change: "Initial skill release."
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_DOMAIN_ID
  global_service: true
  endpoint: "https://iam.myhuaweicloud.com"
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud IAM Operations Skill

## Overview

Huawei Cloud Identity and Access Management (IAM) provides identity authentication, permission control, and resource access management for Huawei Cloud. IAM is the **security foundation** for all cloud operations — every other skill delegates permission and identity questions to this skill. This skill is an **operational runbook** for agents: explicit scope, credential rules, pre-flight checks, **dual-path execution** (official **SDK/API** and matching **CLI** flows), response validation, and failure recovery.

### Critical IAM Characteristics

| Characteristic | Detail |
|----------------|--------|
| **Global Service** | IAM is NOT region-specific; endpoint is `https://iam.myhuaweicloud.com` |
| **Domain ID** | Uses `HW_DOMAIN_ID` (account ID) instead of `HW_PROJECT_ID` for most operations |
| **Synchronous Operations** | Most IAM operations are synchronous (no async job polling needed) |
| **Security Critical** | IAM misconfig = full account compromise; extra safety gates required |

### CLI applicability (repository policy)

- **`cli_applicability: dual-path`:** Official CLI supports this product. You **MUST** ship **`references/cli-usage.md`** and, in **each** execution flow below, document **both** the SDK step **and** the CLI step for every operation.

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions with precise triggers and delegation rules |
| 2 | **Structured I/O** | Placeholder conventions with type and source documented |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute → Validate → Recover |
| 4 | **Complete Failure Strategies** | Error taxonomy ≥ 10 codes; HALT vs retry per error type |
| 5 | **Absolute Single Responsibility** | One product, one resource model; cross-product delegation to other skills |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

In addition to the Five Core Standards, this skill integrates three operational pillars:

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **FinOps (财务运营)** | IAM is free but misconfig has indirect cost impact; permission creep detection | `references/well-architected-assessment.md` §3 |
| **SecOps (安全运营)** | Zero trust, MFA enforcement, credential rotation, least privilege — CRITICAL for IAM | `references/well-architected-assessment.md` §4 |
| **AIOps (智能运营)** | Permission creep detection, unused credentials, stale account identification | `references/well-architected-assessment.md` §5 |

### Well-Architected Framework Integration (卓越架构)

This skill maps operations to Huawei Cloud's Well-Architected Framework five pillars:

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **安全 (Security)** | MFA, credential rotation, least privilege, zero trust, audit logging | `references/well-architected-assessment.md` §2.1 |
| **稳定 (Stability)** | Credential backup, group-based access, agency delegation | `references/well-architected-assessment.md` §2.2 |
| **成本 (Cost)** | IAM is free; indirect cost from over-provisioned permissions | `references/well-architected-assessment.md` §2.3 |
| **效率 (Efficiency)** | Batch permission assignment, policy templates, automation | `references/well-architected-assessment.md` §2.4 |
| **性能 (Performance)** | API rate limits, pagination for large account operations | `references/well-architected-assessment.md` §2.5 |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud IAM" OR "身份认证" OR "权限管理" OR "访问控制"
- User mentions "用户组" OR "策略" OR "委托" OR "AK/SK" OR "MFA"
- Task involves creating, modifying, or deleting **IAM users, groups, policies, roles, or agencies**
- Task involves **credential management** (AK/SK creation, rotation, deletion)
- Task involves **permission assignment** or **access control configuration**
- Task keywords: create user, create group, assign policy, create agency, create access key, enable MFA, permission denied
- User asks to manage identity, authentication, or authorization **via API, SDK, CLI, or automation**
- User reports permission errors, access denied, credential issues, or identity-related problems

### SHOULD NOT Use This Skill When

- Task is purely billing / account management → delegate to: `huaweicloud-billing-ops`
- Task is VPC / network only → delegate to: `huaweicloud-vpc-ops`
- Task is about specific resource operations (ECS, RDS, etc.) → delegate to the respective skill
- Task is about monitoring/alerting only → delegate to: `huaweicloud-ces-ops`

### Delegation Rules

- IAM is the **security foundation** — other skills delegate IAM permission questions here.
- If a resource operation fails with `403 Permission Denied`, delegate to this skill for permission diagnosis.
- Multi-product requests: handle each product with its skill; do not merge unrelated APIs into one ambiguous flow.
- For SecOps questions involving IAM: use this skill as primary, delegate non-IAM security to HSS/WAF skills.
- For credential-related incidents: this skill handles AK/SK and password; delegate key management to KMS skill.

## Variable Convention (Agent-Readable)

| Placeholder | Meaning | Agent Action |
|-------------|---------|--------------|
| `{{env.HW_ACCESS_KEY_ID}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_DOMAIN_ID}}` | Account domain ID (global) | NEVER ask the user; fail if unset |
| `{{user.user_name}}` | User-supplied IAM user name | Ask once; reuse |
| `{{user.group_name}}` | User-supplied IAM group name | Ask once; reuse |
| `{{user.policy_name}}` | User-supplied policy name | Ask once; reuse |
| `{{user.agency_name}}` | User-supplied agency name | Ask once; reuse |
| `{{user.project_name}}` | User-supplied project name | Ask once; reuse |
| `{{output.user_id}}` | From last API or CLI JSON response | Parse per OpenAPI path for this operation |
| `{{output.group_id}}` | From last API or CLI JSON response | Parse per OpenAPI path |
| `{{output.policy_id}}` | From last API or CLI JSON response | Parse per OpenAPI path |
| `{{output.access_key_id}}` | From AK/SK creation response | **MASK** in all output; show only last 4 chars |

> **`{{env.*}}` MUST NOT** be collected from the user. **`{{user.*}}`** MUST be collected interactively when missing.

> **Security Warning (Credential Masking — MANDATORY):** **NEVER** log, print, or expose `HW_SECRET_ACCESS_KEY`, user passwords, AK/SK secret keys, or any credential field value in console output, debug messages, error messages, or logs. AK/SK secrets must be masked as `***`; show only the Access Key ID prefix.

## API and Response Conventions

- **OpenAPI is canonical** for path, query, body fields, enums, and response shapes.
- **Global Endpoint:** `https://iam.myhuaweicloud.com` — NOT region-specific.
- **Authentication:** Uses Domain-level AK/SK for most operations; `HW_DOMAIN_ID` required.
- **Errors:** Map SDK/HTTP errors to `code` / `status` / message fields per spec.
- **Timestamps:** ISO 8601 with timezone when the API returns strings.
- **Idempotency:** User names must be unique within the domain; policy names must be unique; duplicate creation returns error.
- **Synchronous Operations:** Most IAM operations return results immediately; no job polling required (unlike RDS/ECS).

## Quick Start

### What This Skill Does
This skill enables management of Huawei Cloud IAM identity and access control using the `hcloud` CLI (primary) or JIT Go SDK (fallback).

### Prerequisites

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|-----------|-------------|------------|------------|
| Create User | Create a new IAM user | Low | Low |
| List Users | View all IAM users | Low | None |
| Create Group | Create a user group | Low | Low |
| Create Policy | Create a custom policy | Medium | Medium |
| Attach Policy | Assign policy to user/group | Medium | **High** — over-permission risk |
| Create Agency | Create cross-account delegation | High | **High** — trust relationship |
| Create Access Key | Create AK/SK for a user | Low | **High** — credential exposure |
| Delete User | Remove an IAM user | Low | **High** — irreversible, access loss |
| Delete Policy | Remove a custom policy | Low | **High** — may break access |
| Enable MFA | Configure MFA for a user | Medium | Medium |

## Execution Flows

### Operation: Create IAM User

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| CLI / deps | `hcloud --version` | Exit code 0 | Document CLI install |
| Credentials | Construct credential from env | Non-empty keys | HALT; user configures env |
| Domain ID | Validate `{{env.HW_DOMAIN_ID}}` is set | Non-empty | HALT; user configures env |
| User existence | Call **ListUsers** or check `{{user.user_name}}` | User does NOT exist | Ask: use existing or different name |
| Password policy | Check account password policy | Meets requirements | Warn user of policy constraints |

#### Execution — CLI (Primary Path)

```bash
# CLI invocation for creating IAM user
hcloud iam create-user \
  --domain-id "{{env.HW_DOMAIN_ID}}" \
  --name "{{user.user_name}}" \
  --description "{{user.description}}" \
  --email "{{user.email}}" \
  --phone "{{user.phone}}"
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
    iam "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3"
    iam_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/model"
)

func main() {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    domainId := os.Getenv("HW_DOMAIN_ID")
    
    cfg := config.DefaultHttpConfig()
    client := iam.IamClientBuilder().
        WithEndpoint("https://iam.myhuaweicloud.com").
        WithCredential(basic.NewCredentialsBuilder().
            WithAk(ak).WithSk(sk).WithDomainId(domainId).Build()).
        WithHttpConfig(cfg).Build()
    
    request := &iam_model.CreateUserRequest{
        Body: &iam_model.CreateUserRequestBody{
            User: iam_model.CreateUserOption{
                Name:        "{{user.user_name}}",
                Description: ptrString("{{user.description}}"),
                Email:       ptrString("{{user.email}}"),
                Phone:       ptrString("{{user.phone}}"),
            },
        },
    }
    
    response, err := client.CreateUser(request)
    if err != nil {
        panic(err)
    }
    fmt.Printf("User ID: %s\n", response.User.Id)
}

func ptrString(s string) *string { return &s }
```

#### Post-execution Validation

1. Read `{{output.user_id}}` from the response path `user.id`.
2. Call **ShowUser** to verify user exists and status is `active`.
3. On success, report `{{output.user_id}}` and user name.
4. On failure, go to **Failure Recovery**.

#### Failure Recovery

| Error | Max retries | Backoff | Agent Action | UX Feedback |
|-------|-------------|---------|--------------|-------------|
| `IAM.0001` | 0 | — | Fix args from OpenAPI | `[ERROR] InvalidParameter: Check parameters against OpenAPI docs.` |
| `IAM.0002` | 0 | — | HALT | `[ERROR] User already exists. Use different name or manage existing user.` |
| `IAM.0003` | 0 | — | HALT | `[ERROR] Quota exceeded. Request quota increase or delete unused users.` |
| `IAM.0006` | 0 | — | HALT | `[ERROR] Permission denied. Check IAM permissions for current credentials.` |
| Throttling / 429 | 3 | exponential | Back off; respect Retry-After | `⚠️ Rate limited. Retrying in {backoff}s...` |
| `InternalError` / 5xx | 3 | 2s, 4s, 8s | Retry; then HALT | `[ERROR] Server-side error. Retry or escalate with RequestId.` |

### Operation: Create IAM Group

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| Credentials | Construct from env | Non-empty keys | HALT |
| Domain ID | `{{env.HW_DOMAIN_ID}}` | Set | HALT |
| Group existence | Call **ListGroups** | Group does NOT exist | Ask: use existing or different name |

#### Execution — CLI (Primary Path)

```bash
hcloud iam create-group \
  --domain-id "{{env.HW_DOMAIN_ID}}" \
  --name "{{user.group_name}}" \
  --description "{{user.group_description}}"
```

#### Post-execution Validation

1. Read `{{output.group_id}}` from the response.
2. Call **ShowGroup** to verify group exists.
3. Report `{{output.group_id}}` and group name.

### Operation: Create Custom Policy

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| Credentials | Construct from env | Non-empty keys | HALT |
| Domain ID | `{{env.HW_DOMAIN_ID}}` | Set | HALT |
| Policy name uniqueness | Call **ListPolicies** | Name does NOT exist | Ask: update existing or use different name |
| Policy syntax | Validate JSON against IAM policy grammar | Valid syntax | HALT; fix policy document |

#### Execution — CLI (Primary Path)

```bash
hcloud iam create-policy \
  --domain-id "{{env.HW_DOMAIN_ID}}" \
  --name "{{user.policy_name}}" \
  --description "{{user.policy_description}}" \
  --policy-document '{{user.policy_document}}'
```

#### Execution — JIT Go SDK (Fallback Path)

```go
request := &iam_model.CreatePolicyRequest{
    Body: &iam_model.CreatePolicyRequestBody{
        Policy: iam_model.CreatePolicyOption{
            Name:        "{{user.policy_name}}",
            Description: ptrString("{{user.policy_description}}"),
            Policy:      "{{user.policy_document}}",  // JSON policy document
        },
    },
}
response, err := client.CreatePolicy(request)
```

#### Post-execution Validation

1. Read `{{output.policy_id}}` from the response.
2. Call **ShowPolicy** to verify policy exists and content matches.
3. Report `{{output.policy_id}}` and policy name.

### Operation: Attach Policy to User/Group

#### Pre-flight (Safety Gate)

- **MUST** warn about permission scope: "Attaching policy `{{user.policy_name}}` grants the following permissions: [list actions from policy]. Ensure this follows least privilege principle."
- **MUST** verify the policy exists and is in `active` status.
- **MUST NOT** attach administrator-level policies without explicit confirmation.

#### Execution — CLI (Primary Path)

```bash
# Attach policy to user
hcloud iam attach-policy-to-user \
  --domain-id "{{env.HW_DOMAIN_ID}}" \
  --user-id "{{user.user_id}}" \
  --policy-id "{{user.policy_id}}"

# Attach policy to group
hcloud iam attach-policy-to-group \
  --domain-id "{{env.HW_DOMAIN_ID}}" \
  --group-id "{{user.group_id}}" \
  --policy-id "{{user.policy_id}}"
```

#### Post-execution Validation

1. Call **ShowUserPolicies** or **ShowGroupPolicies** to verify policy is attached.
2. Verify the target entity can perform expected actions (optional test).

### Operation: Create Agency (Cross-Account Delegation)

#### Pre-flight (Safety Gate)

- **MUST** warn: "Creating an agency delegates access from your account to another domain. Verify the trusting and trusted domains."
- **MUST** confirm: trusting domain, trusted domain, delegated permissions.
- **MUST** validate the policy document for the agency.

#### Execution — CLI (Primary Path)

```bash
hcloud iam create-agency \
  --domain-id "{{env.HW_DOMAIN_ID}}" \
  --name "{{user.agency_name}}" \
  --description "{{user.agency_description}}" \
  --trust-domain-id "{{user.trusted_domain_id}}" \
  --duration "{{user.duration}}"
```

### Operation: Create Access Key (AK/SK)

#### Pre-flight (Safety Gate — CRITICAL)

- **MUST** warn: "Creating an AK/SK generates permanent credentials. These must be stored securely and rotated every 90 days."
- **MUST** confirm: target user, purpose, and that the user has MFA enabled.
- **MUST NOT** create AK/SK for the account admin without explicit justification.
- **MUST** remind: "The secret key will only be shown once. Save it immediately."

#### Execution — CLI (Primary Path)

```bash
hcloud iam create-access-key \
  --user-id "{{user.user_id}}" \
  --description "{{user.key_description}}"
```

#### Post-execution Validation

1. **MASK** the secret key in all output: `***` (show only last 4 characters of Access Key ID).
2. Record creation timestamp for rotation tracking.
3. Remind user: "Store the secret key securely. It cannot be retrieved again."

#### Failure Recovery

| Error | Max retries | Agent Action | UX Feedback |
|-------|-------------|--------------|-------------|
| `IAM.0003` | 0 | HALT | `[ERROR] AK/SK quota exceeded (max 2 per user). Delete unused keys first.` |
| `IAM.0006` | 0 | HALT | `[ERROR] Permission denied. Only the user themselves or account admin can create AK/SK.` |

### Operation: Delete User

#### Pre-flight (Safety Gate — CRITICAL)

- **MUST** obtain explicit confirmation: irreversible deletion of `{{user.user_name}}` (`{{user.user_id}}`).
- **MUST NOT** proceed without clear user assent.
- **MUST** remind user: "This will permanently delete the IAM user and all associated access keys, policies, and group memberships."
- **MUST** warn about cascading effects: AK/SK will be invalidated, agency relationships may break, applications using this user's credentials will fail.
- **MUST** check: user has no active resources depending on their credentials.

#### Execution

```bash
hcloud iam delete-user \
  --domain-id "{{env.HW_DOMAIN_ID}}" \
  --user-id "{{user.user_id}}"
```

#### Post-execution Validation

Poll **ShowUser** until **404** or **NotFound** status — within **max wait 30s** (synchronous operation).

### Operation: Delete Policy

#### Pre-flight (Safety Gate)

- **MUST** check: policy is not attached to any users or groups.
- **MUST** warn: if policy is attached, list affected entities before deletion.
- **MUST** obtain explicit confirmation.

#### Execution

```bash
hcloud iam delete-policy \
  --domain-id "{{env.HW_DOMAIN_ID}}" \
  --policy-id "{{user.policy_id}}"
```

## Prerequisites

> Full installation scripts (KooCLI + Go runtime + Credentials): see [references/common-prerequisites.md](../references/common-prerequisites.md)

## Quality Gate (GCL)

This skill is **GCL-required** (per `AGENTS.md` §8). Every IAM mutating operation — user / group /
policy / agency / access-key / MFA / password / domain — runs through the **Generator-Critic-Loop**
before its result is returned to the user. Read-only `list*` / `get*` / `describe*` are
GCL-**exempt**.

| Field | Value |
|-------|-------|
| Rubric version | v1 (Phase 2, 2026-06-04) |
| `max_iter` | **2** |
| Rubric instance | [`references/rubric.md`](references/rubric.md) |
| Prompt templates | [`references/prompt-templates.md`](references/prompt-templates.md) |
| Trace path | `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` |
| Independence | Generator and Critic run in **isolated** sub-agent / session contexts |

### Five-Dimension Rubric (summary)

| # | Dimension | Threshold | Notes |
|---|-----------|-----------|-------|
| 1 | Correctness | ≥ 0.5 (1.0 for `delete-user` / `delete-policy` / `detach-policy` / `delete-access-key` / `create-access-key`) | Verified via `ShowUser` / `ShowPolicy` / `ListAttachedPolicies` post-state |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | S1–S14 in rubric §2 |
| 3 | Idempotency | ≥ 0.5 | Pre-check before create / attach; S4 cap on AK count |
| 4 | Traceability | ≥ 0.5 | Full request/response; `secret_access_key` MUST be `<masked>` |
| 5 | Spec Compliance | ≥ 0.5 | Policy JSON syntax; principal patterns; password policy; name regex |

### Per-Operation Safety Anchors (binding)

- **S1** — `delete-user` while policies / group memberships / keys still attached
- **S2 / S3** — Detach / attach `AdministratorAccess` / `*:*:*` without two-step confirmation
- **S4** — `create-access-key` when user already has ≥ 2 active keys (Huawei default limit)
- **S5 / S9 / S13** — AK / SK / password plaintext anywhere in trace
- **S6** — `create-policy` with `Action: *` + `Resource: *` + `Effect: Allow` without flag
- **S7** — `create-agency` with `Principal: { IAM: ["*"] }` or service principal `*`
- **S8** — `delete-policy` while `AttachmentCount > 0`
- **S10** — `update-user` / `create-user` with password plaintext
- **S11** — `delete-domain` / `delete-project` from non-`account-level` token
- **S12** — `mfa-disable` for account root or `password_reset`-capable user
- **S14** — `update-password-policy` disabling MFA OR min_length < 8

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

- [`references/rubric.md`](references/rubric.md) — full rubric, S1–S14 rules, per-op thresholds
- [`references/prompt-templates.md`](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons
- Repository root [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8 — GCL specification

## Reference Directory

- [Core Concepts](references/core-concepts.md) — IAM architecture, identity model, permission model
- [API & SDK Usage](references/api-sdk-usage.md) — API mapping, Go SDK patterns
- [CLI Usage](references/cli-usage.md) — hcloud command reference
- [Troubleshooting Guide](references/troubleshooting.md) — Error codes, diagnostics
- [Monitoring & Alerts](references/monitoring.md) — CTS events, AIOps patterns
- [Integration](references/integration.md) — Cross-skill delegation, SDK setup
- [Well-Architected Assessment](references/well-architected-assessment.md) — Five pillars + FinOps + SecOps + AIOps
- [GCL Rubric](references/rubric.md) — Adversarial quality gate (v1, 5-dim, S1–S14 IAM-specific Safety rules)
- [GCL Prompt Templates](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons

## FinOps Integration (财务运营)

### Cost Impact

IAM service itself is **free** — no direct billing. However, IAM misconfigurations have **indirect cost impact**:

| Misconfiguration | Cost Impact | Detection |
|------------------|-------------|-----------|
| Over-provisioned permissions | Unintended resource creation | Policy audit (quarterly) |
| Orphaned AK/SK | Potential unauthorized usage | Unused key detection (90 days) |
| Unrestricted agency | Cross-account resource consumption | Agency permission review |

### Cost Governance

| Pattern | Description | Action |
|---------|-------------|--------|
| Permission creep | Users accumulate permissions over time | Quarterly permission audit |
| Unused credentials | AK/SK not used for > 90 days | Disable and notify |
| Stale accounts | Users not logged in for > 180 days | Disable and notify |

## SecOps Integration (安全运营)

### IAM as Security Foundation

IAM is the **most critical** service for security. Every other skill depends on IAM for access control.

#### Zero Trust Principles

| Principle | IAM Implementation | Enforcement |
|-----------|-------------------|-------------|
| Never trust, always verify | MFA for all human users | Policy enforcement |
| Least privilege | Fine-grained policies, no wildcard actions | Policy audit |
| Assume breach | Credential rotation, short-lived tokens | 90-day AK/SK rotation |
| Explicit deny | Deny overrides allow in policy evaluation | Policy design |

#### Credential Rotation Schedule

| Credential Type | Rotation Period | Method |
|----------------|-----------------|--------|
| AK/SK | 90 days | Create new → Update apps → Delete old |
| Password | 90 days (policy-enforced) | Password policy configuration |
| MFA Device | On compromise suspicion | Re-register MFA device |
| Agency | Annual review | Review and re-authorize |

#### Incident Response

| Incident | IAM Response | Severity |
|----------|--------------|----------|
| Compromised AK/SK | Immediately disable → Create new → Audit CTS logs | Critical |
| Unauthorized access | Revoke permissions → Enable MFA → Audit | Critical |
| Permission escalation | Remove excessive permissions → Audit policy assignments | High |
| Account lockout | Check login attempts → Reset if legitimate | Medium |

## AIOps Integration (智能运营)

### Permission Analytics

| Pattern | Detection Logic | Severity | Action |
|---------|-----------------|----------|--------|
| Permission creep | User has > N policies over time | Warning | Review and trim |
| Unused credentials | AK/SK not used for > 90 days | Warning | Disable and notify |
| Stale accounts | No login for > 180 days | Warning | Disable and notify |
| Orphaned policies | Policy not attached to any entity | Info | Clean up |
| Cross-account anomaly | Unusual agency activity | Critical | Investigate immediately |

### Cross-Skill Delegation Matrix

| Alert Type | Primary Skill | Secondary Skill | Notes |
|-----------|---------------|-----------------|-------|
| Permission Denied | huaweicloud-iam-ops | Respective resource skill | Check IAM policy first |
| Credential Compromise | huaweicloud-iam-ops | huaweicloud-cts-ops | Audit CTS for unauthorized actions |
| MFA Bypass Attempt | huaweicloud-iam-ops | huaweicloud-hss-ops | Security incident response |

### Proactive Inspection Workflow

1. **Discovery**: List all IAM users, groups, and policies
2. **Analysis**: Identify permission creep, unused credentials, stale accounts
3. **Cross-Skill Check**: Verify resource access patterns match permissions
4. **Report Generation**: Generate security posture report with findings
5. **Remediation**: Recommend and execute permission cleanup

## Well-Architected + Three-Pillar Assessment

This skill's operations are evaluated against Huawei Cloud's Well-Architected Framework (卓越架构) five pillars plus FinOps, SecOps, and AIOps integration:
- [Security Assessment](references/well-architected-assessment.md#21-安全支柱-security)
- [Stability Assessment](references/well-architected-assessment.md#22-稳定支柱-stability)
- [Cost Assessment](references/well-architected-assessment.md#23-成本支柱-cost)
- [Efficiency Assessment](references/well-architected-assessment.md#24-效率支柱-efficiency)
- [Performance Assessment](references/well-architected-assessment.md#25-性能支柱-performance)
- [FinOps Integration](references/well-architected-assessment.md#3-finops-财务运营)
- [SecOps Integration](references/well-architected-assessment.md#4-secops-安全运营)
- [AIOps Integration](references/well-architected-assessment.md#5-aiops-integration)
