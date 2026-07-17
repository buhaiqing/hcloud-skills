---
name: huaweicloud-kms-ops
description: >-
  Use when the user needs to create, manage, rotate, grant, or troubleshoot Huawei Cloud
  Key Management Service (KMS / 密钥管理) master keys — CMK lifecycle, envelope encryption,
  BYOK import, grant management, key state toggle, and deletion scheduling. User mentions
  KMS, 密钥管理, 密钥, CMK, 密钥轮换, 密钥授权, BYOK, 密钥材料导入, 数据密钥,
  or describes scenarios (e.g., "创建CMK", "启用密钥轮换", "授权其他服务使用密钥",
  "查看密钥状态", "调度删除密钥", "导入密钥材料", "生成数据密钥") even without
  naming the product directly.
  Not for OBS object encryption, RDS TDE, or EVS disk encryption — delegate to the
  respective product skills after key creation.
license: MIT
compatibility: >-
  Official Huawei Cloud CLI (`hcloud` / `openstack`), Go 1.21+ runtime
  (for JIT SDK fallback via huaweicloud-sdk-go-v3), valid API credentials,
  network access to Huawei Cloud endpoints.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-06-24"
  runtime: Harness AI Agent, Claude Code, Cursor or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  api_profile: "KMS API v2 - https://support.huaweicloud.com/api-kms/kms_api_0001.html"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    KMS product supported by hcloud CLI via `hcloud kms`. Verify with `hcloud kms --help`.
    JIT Go SDK fallback covers advanced operations (grant management, key material import,
    data key operations, BYOK).
  gcl:
    enabled: true
    required: true
    rubric_version: "v1"
    max_iter: 2
    rubric_ref: "references/rubric.md"
    prompts_ref: "references/prompt-templates.md"
    trace_dir: "./audit-results/"
    changelog:
      - version: "1.0.0"
        date: "2026-06-24"
        change: "Initial skill release: 10 operations (list/create/describe/enable/disable/schedule-delete/grant/crud/import/datakey), FinOps CMK quota matrix, SecOps IAM least-privilege, AIOps 4 anomaly patterns, GCL rubric + prompt templates."
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud Key Management Service (KMS) Operations Skill

## Overview

Huawei Cloud KMS provides hardware-security-module (HSM) backed key management. A Customer
Master Key (CMK / 主密钥) encrypts data at rest across OBS, RDS, EVS, and other services.
CMKs are region-scoped, never exportable in raw form, and support rotation, grants, and
BYOK import. **KMS keys protect other resources** — a key deletion cascades to all resources
that depend on it.

This skill is an **operational runbook** for agents: explicit scope, credential rules,
pre-flight checks, **dual-path execution** (`hcloud` CLI primary + JIT Go SDK fallback),
response validation, and failure recovery. **Do not use the web console as the primary
agent execution path.**

### CLI applicability (repository policy)

- **`cli_applicability: dual-path`:** Official CLI covers key lifecycle and basic grants.
  Ship `references/cli-usage.md` and document both CLI and SDK steps for every operation.
  Use SDK fallback for: data key generation/decryption, key material import, grant
  listing with principals.

### What This Skill Owns

| In scope | Out of scope (delegate) |
|---|---|
| CMK lifecycle (list/create/describe/enable/disable/delete) | OBS object encryption → `huaweicloud-obs-ops` |
| Key rotation toggle | RDS TDE → `huaweicloud-rds-ops` |
| Schedule deletion (7–1096 day window) | EVS disk encryption → EVS skill (when present) |
| Grant management (create/list/revoke) | IAM policy authoring → `huaweicloud-iam-ops` |
| Key material import (BYOK) | Billing cost tracking → `huaweicloud-billing-ops` |
| Data key create / decrypt | |

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions + delegation matrix |
| 2 | **Structured I/O** | `{{env.*}}` / `{{user.*}}` / `{{output.*}}` placeholder convention |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute → Validate → Recover |
| 4 | **Complete Failure Strategies** | ≥10 KMS error codes; HALT vs retry per error type |
| 5 | **Absolute Single Responsibility** | CMK + grants only; encrypting resources delegate to other skills |
| 6 | **GCL Adversarial Rubric** | `## Quality Gate (GCL)`; `references/rubric.md` 8 sections; `references/prompt-templates.md` 7 sections |

### Three-Pillar Ops Integration

| Pillar | Integration | Reference |
|---|---|---|
| **FinOps** | CMK quota (100/account default), rotation cost, idle key detection | `references/well-architected-assessment.md` §3 |
| **SecOps** | IAM least privilege (kms:viewer/operator/admin), grant attack surface, deletion cascade | `references/well-architected-assessment.md` §4 |
| **AIOps** | 4 anomaly patterns (state anomaly, grant proliferation, deletion storm, throttling) | `references/advanced/aiops-patterns.md` |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "KMS" / "密钥管理" / "密钥" / "CMK" / "BYOK" / "密钥轮换"
- Task keywords: 创建密钥, 查看密钥, 启用密钥, 禁用密钥, 删除密钥, 授权密钥, 导入密钥材料, 生成数据密钥, 解密数据密钥
- User asks to manage CMK lifecycle **via API, SDK, CLI, or automation**
- Anomaly reported: "密钥意外禁用", "密钥授权过多", "删除窗口快到期", "KMS API 限流"

### SHOULD NOT Use This Skill When

- Encrypting OBS objects → `huaweicloud-obs-ops`
- Enabling RDS TDE → `huaweicloud-rds-ops`
- EVS disk encryption → EVS skill (when present)
- IAM policy authoring → `huaweicloud-iam-ops`
- Pure billing reconciliation → `huaweicloud-billing-ops`

### Delegation Rules

- Create CMK first via this skill, then delegate resource encryption to the target product skill.
- Multi-key requests: handle each CMK operation with this skill; never merge KMS + OBS into one flow.
- For FinOps questions: use this skill's cost section, delegate cross-resource cost to `huaweicloud-billing-ops`.

## Variable Convention

| Placeholder | Meaning | Agent Action |
|---|---|---|
| `{{env.HW_ACCESS_KEY_ID}}` | AK from runtime env | NEVER ask user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | SK from runtime env | NEVER ask user; fail if unset |
| `{{env.HW_REGION_ID}}` | Region | Default if user agrees |
| `{{env.HW_PROJECT_ID}}` | Project ID | Default if user agrees |
| `{{user.key_id}}` | KMS CMK ID (`kmsxxxxxxxx`) | Ask once; reuse |
| `{{user.key_alias}}` | Human-readable alias | Ask once; reuse |
| `{{user.key_state}}` | `ENABLED` / `DISABLED` / `PENDING_DELETION` | From describe-key |
| `{{output.key_id}}` | CMK resource ID | Parse from `key_metadata.key_id` |
| `{{output.key_arn}}` | CMK ARN | Parse from `key_metadata.key_arn` |

> **`{{env.*}}` MUST NOT** be collected from the user. **`{{user.*}}`** MUST be collected interactively when missing.

> **Security Warning (Credential Masking — MANDATORY):** **NEVER** log, print, or expose
> `HW_SECRET_ACCESS_KEY`, SecretAccessKey, or any key material value in console output,
> debug messages, error messages, or GCL traces.

## API Conventions

- **OpenAPI is canonical**: [KMS API v2](https://support.huaweicloud.com/api-kms/kms_api_0001.html).
- **Errors:** Map SDK/HTTP errors to `error_code` / `error_msg` fields.
- **Timestamps:** ISO 8601 with timezone when the API returns strings.
- **Idempotency:**
  - `create-key` is idempotent by alias — duplicate alias returns existing key.
  - `schedule-key-deletion` is idempotent — calling on already-scheduled key returns success.
  - `create-grant` is idempotent by (key_id, grantee_principal, operations).
  - `enable-key` / `disable-key` are idempotent by state.

## Quick Start

### Verify Setup
```bash
hcloud --version
hcloud kms list-keys --region {{env.HW_REGION_ID}}
```

### Your First Command
```bash
# List all CMKs and their states
hcloud kms list-keys --region {{env.HW_REGION_ID}} \
  --output json | jq '.keys[] | {key_id: .key_id, key_state: .key_state}'
```

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|---|---|---|---|
| List Keys | Enumerate all CMKs and their metadata | Low | None |
| Create Key | Create a CMK with alias, rotation, and tags | Medium | Low |
| Describe Key | View key metadata, rotation status, grants, tags | Low | None |
| Enable Key | Set key state to ENABLED | Low | Low |
| Disable Key | Set key state to DISABLED | Low | Medium — affects all dependent resources |
| Schedule Deletion | Set 7–1096 day deletion window | Medium | **High** — irreversible after window |
| Create Grant | Delegate key usage to another service | Medium | Medium |
| List / Revoke Grants | Manage existing grants | Low | Medium — revoke breaks dependent services |
| Import Key Material | BYOK: import external key material | Medium | Medium |
| Create Data Key | Generate a DEK without leaving cloud | Low | Low |
| Decrypt Data Key | Decrypt ciphertext encrypted by a CMK | Low | Low |

## Execution Flows

### Operation 1: List Keys

#### Execution — CLI
```bash
hcloud kms list-keys --region {{env.HW_REGION_ID}} --output json
```

#### Execution — Go SDK
```go
//go:build ignore
req := &kms_model.ListKeysRequest{
    Limit: ptr.Int32(100),
}
resp, err := client.ListKeys(req)
// resp.Keys[].KeyId / resp.KeyDetails[].KeyState
```

#### Output fields (agent-parse contract)

| Field | JSON path | Meaning |
|---|---|---|
| Key ID | `key_metadata.key_id` | `{{output.key_id}}` |
| Key ARN | `key_metadata.key_arn` | Full ARN |
| Key State | `key_metadata.key_state` | `ENABLED` / `DISABLED` / `PENDING_DELETION` |
| Creation date | `key_metadata.creation_date` | ISO 8601 |
| Key type | `key_metadata.key_type` | `SYMMETRIC_DEFAULT` / `ASYMMETRIC_...` |
| Rotation | `key_metadata.rotation_enabled` | `true` / `false` |

### Operation 2: Create Key

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|---|---|---|---|
| CLI / deps | `hcloud --version` | Exit code 0 | Document CLI install |
| Credentials | Construct from env | Non-empty AK/SK | HALT; configure env |
| Quota | `ShowKeyQuotas` SDK | `used < quota` (default 100 CMK) | HALT; raise quota |
| Alias uniqueness | `list-keys` → alias check | Alias not in use | Use unique alias |

#### Execution — CLI
```bash
hcloud kms create-key \
  --region "{{user.region}}" \
  --alias "{{user.key_alias}}" \
  --key-type "SYMMETRIC_DEFAULT" \
  --rotation-enabled  # enable automatic rotation (365-day interval)
```

#### Execution — Go SDK
```go
//go:build ignore
req := &kms_model.CreateKeyRequest{
    Body: &kms_model.CreateKeyRequestBody{
        Alias:           ptr.String("my-key-alias"),
        KeyUsage:        ptr.String("ENCRYPT_DECRYPT"),
        KeyType:         ptr.String("SYMMETRIC_DEFAULT"),
        RotationEnabled: ptr.Bool(false),
    },
}
resp, err := client.CreateKey(req)
// resp.KeyMetadata.KeyId / resp.KeyMetadata.KeyArn
```

#### Post-execution Validation
Poll `describe-key` until `key_state` = `ENABLED`.

#### Failure Recovery

| Error | Max retries | Agent Action | UX Feedback |
|---|---|---|---|
| `QuotaExceeded` | 0 | HALT | `[ERROR] CMK quota (100) reached. Request increase via Console → KMS → Quota.` |
| `InvalidAliasName` | 0 | Fix alias format | `[ERROR] InvalidAliasName: alias must start with `alias/` prefix.` |
| `DuplicateAlias` | 0 | Return existing key_id | `[WARN] Alias already exists. Returning existing CMK id.` |
| Throttling / 429 | 3 | exponential backoff | `[WARN] Rate limited. Retrying...` |

### Operation 3: Describe Key

#### Execution
```bash
hcloud kms describe-key --region {{env.HW_REGION_ID}} --key-id "{{user.key_id}}"
```

### Operation 4: Enable / Disable Key

#### Pre-flight (Safety Gate)
- **Disable**: warn that all dependent services (OBS, RDS, EVS) will fail to encrypt/decrypt until re-enabled.
- Confirm: if key name matches `(?i)(prod|prd|production|online|encrypt-prod)`, require explicit confirmation.

#### Execution
```bash
# Enable
hcloud kms enable-key --region "{{user.region}}" --key-id "{{user.key_id}}"

# Disable
hcloud kms disable-key --region "{{user.region}}" --key-id "{{user.key_id}}"
```

#### Post-execution Validation
Poll `describe-key` until `key_state` matches target (`ENABLED` / `DISABLED`).

### Operation 5: Schedule Key Deletion

#### Pre-flight (Safety Gate — IRREVERSIBLE AFTER WINDOW)

- **MUST** require explicit confirmation: `schedule-key-deletion` of `{{user.key_id}}` is permanent after the window expires.
- **MUST** warn: key remains **USABLE** during the 7–1096 day deletion window — dependent services keep working until window expires.
- **MUST** warn: any OBS objects, RDS databases, or EVS disks encrypted with this key become **unrecoverable** after window.
- **MUST** verify no active grants remain — revoke all grants first [S3].
- **MUST** verify key is `ENABLED` — cannot schedule deletion of a `DISABLED` key without re-enabling first.

#### Execution
```bash
hcloud kms schedule-key-deletion \
  --region "{{user.region}}" \
  --key-id "{{user.key_id}}" \
  --pending-window-days 30  # 7–1096, default 30
```

#### Post-execution Validation
Poll `describe-key` until `key_state` = `PENDING_DELETION` and `deletion_date` is set.

#### Failure Recovery

| Error | Agent Action |
|---|---|
| `InvalidParameter` (window < 7 or > 1096) | Fix window days |
| `InvalidKeyState` (key is DISABLED) | Re-enable key first, then schedule |
| `KeyExistsException` (already scheduled) | Idempotent — treat as success |

### Operation 6: Create / List / Revoke Grants

#### Create Grant
```bash
hcloud kms create-grant \
  --region "{{user.region}}" \
  --key-id "{{user.key_id}}" \
  --grantee-principal "acs:ram::123456789:role/KMSRole" \
  --operations "Encrypt,Decrypt,GenerateDataKey"
```

#### List Grants
```bash
hcloud kms list-grants --region "{{user.region}}" --key-id "{{user.key_id}}"
```

#### Revoke Grant
```bash
# Get grant_id from list-grants output
hcloud kms revoke-grant \
  --region "{{user.region}}" \
  --key-id "{{user.key_id}}" \
  --grant-id "{{user.grant_id}}"
```

#### Pre-flight (Revoke Safety Gate)
- **MUST** warn: revoking a grant immediately breaks the grantee service's access to data encrypted by this key.

### Operation 7: Import Key Material (BYOK)

#### Pre-flight
- Verify key was created with `key_material_source = EXTERNAL` (import-only key).
- Only symmetric keys support BYOK import.

#### Execution
```bash
# 1) Create an import-only key
hcloud kms create-key \
  --region "{{user.region}}" \
  --alias "byok-key" \
  --key-type "SYMMETRIC_DEFAULT" \
  --import-template "RSAES_OAEP_SHA_256"

# 2) Get import token (valid 24h)
hcloud kms create-import-token \
  --region "{{user.region}}" \
  --key-id "{{user.key_id}}" \
  --output json | jq '{import_token, public_key}'

# 3) Encrypt key material with public_key (do this externally), then import
# agent: wrap key material with the returned public key, then call:
hcloud kms import-key-material \
  --region "{{user.region}}" \
  --key-id "{{user.key_id}}" \
  --import-token "{{user.import_token}}" \
  --encrypted-key-material "{{user.encrypted_material_path}}"
```

### Operation 8: Create / Decrypt Data Key

#### Create Data Key
```bash
# Generate a DEK — plaintext and ciphertext are returned
hcloud kms create-datakey \
  --region "{{user.region}}" \
  --key-id "{{user.key_id}}" \
  --datakey-plain-length 32 \
  --output json | jq '{plaintext, ciphertext}'
```

#### Decrypt Data Key
```bash
hcloud kms decrypt-datakey \
  --region "{{user.region}}" \
  --key-id "{{user.key_id}}" \
  --cipher-text "{{user.cipher_text}}" \
  --output json | jq '.plain_text'
```

## FinOps at a Glance (Details in §3 of well-architected-assessment)

| Item | Value |
|---|---|
| Default CMK quota | 100 per account per region |
| Key rotation interval | 365 days (free for symmetric keys) |
| HSM key creation | Free |
| HSM key usage | Billed per 10,000 API calls |
| BYOK key material | Customer bears key material transport cost |

## SecOps at a Glance (Details in §4 of well-architected-assessment)

| Role | Required Permissions |
|---|---|
| KMS Viewer | `kms:listKeys`, `kms:describeKey`, `kms:listGrants` |
| KMS Operator | Viewer + `kms:createKey`, `kms:enableKey`, `kms:disableKey`, `kms:scheduleKeyDeletion` |
| KMS Admin | Operator + `kms:createGrant`, `kms:revokeGrant`, `kms:importKeyMaterial` |

> **Critical:** `schedule-key-deletion` without revoking grants first leaves dependent
> services with broken encryption. Always check grants before scheduling deletion [S3].

## AIOps at a Glance (Details in references/advanced/aiops-patterns.md)

| Pattern | Detection Signal | Cross-skill |
|---|---|---|
| Key state anomaly | Key flipped to DISABLED without user action | → `huaweicloud-iam-ops` (permission audit) |
| Grant proliferation | `grant_count > 10` on single key | → `huaweicloud-billing-ops` |
| Deletion pending storm | Keys entering `PENDING_DELETION` within 7 d | → `huaweicloud-billing-ops` |
| API throttling | KMS 429 rate limit on routine ops | Back off; reduce polling frequency |

## Quality Gate (GCL)

This skill uses Generator-Critic-Loop runtime validation for cloud operations.

- `references/rubric.md` — 8 sections: scope, dimensions, S-rules, correctness matrix, idempotency, traceability, scoring, escalation.
- `references/prompt-templates.md` — 7 sections: Generator, Critic, Orchestrator, pre-flight, anti-patterns, changelog.
- `SKILL.md` metadata `gcl` block — `required: true`, `default_max_iter: 2`, `rubric_version: "v1"`, `trace_path: "./audit-results/"`.

## Reference Directory

- [Core Concepts](references/core-concepts.md) — CMK model, key types, HSM, rotation
- [API & SDK Usage](references/api-sdk-usage.md) — Go SDK JIT patterns
- [CLI Usage](references/cli-usage.md) — `hcloud kms` command reference
- [Troubleshooting](references/troubleshooting.md) — Top KMS failure patterns
- [Monitoring](references/monitoring.md) — KMS API usage metrics
- [Integration](references/integration.md) — Cross-skill delegation matrix
- [Knowledge Base](references/knowledge-base.md) — Fault patterns
- [Idempotency Checklist](references/idempotency-checklist.md)
- [AIOps Patterns](references/advanced/aiops-patterns.md)
- [Well-Architected Assessment](references/well-architected-assessment.md)
- [GCL Rubric](references/rubric.md)
- [GCL Prompt Templates](references/prompt-templates.md)
