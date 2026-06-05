---
name: huaweicloud-cbr-ops
description: >-
  Use when the user needs to manage backups, backup policies, restore operations, or monitor Huawei Cloud CBR (Cloud Backup and Recovery / 云备份) — backup vault lifecycle, backup policy configuration, on-demand backup execution, cross-region replication, and restore operations. User mentions CBR, cloud backup, 云备份, 备份策略, 存储库, vault, backup policy, restore, or describes backup-related scenarios (e.g., backup failure, recovery point missing, vault capacity full, RPO/RTO concerns) even without naming the product directly. Not for billing, IAM, or related products that have their own ops skills.
license: MIT
compatibility: >-
  Official Huawei Cloud CLI (`hcloud`), Go 1.21+ runtime
  (for JIT SDK fallback via huaweicloud-sdk-go-v3), valid API credentials,
  network access to Huawei Cloud endpoints.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-05-21"
  runtime: Harness AI Agent, Claude Code, Cursor, or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  api_profile: "CBR v3 — https://support.huaweicloud.com/api-cbr/"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    CBR operations available via `hcloud cbr` commands: create-vault, show-vault, list-vaults, delete-vault, create-policy, list-policies, create-backup, list-backups, restore-backup, replicate-backup.
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
        change: "GCL Phase 2 rollout: added references/rubric.md (v1, 5-dim, S1–S15 CBR-specific Safety rules, including restore-target-disk-validation cascade) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
      - version: "1.0.0"
        date: "2026-05-21"
        change: "Initial skill release."
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud CBR (Cloud Backup and Recovery) Operations Skill

## Overview

Huawei Cloud CBR provides fully-managed backup and recovery services for ECS, RDS, DCS, and other cloud resources. This skill is an **operational runbook** for agents: explicit scope, credential rules, pre-flight checks, **dual-path execution** (official CLI and JIT Go SDK fallback), response validation, and failure recovery.

### CLI applicability (repository policy)

- **`cli_applicability: dual-path`:** Official CLI supports CBR. In each execution flow below, document **both** the SDK step **and** the CLI step.

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions with precise triggers and delegation rules |
| 2 | **Structured I/O** | `{{env.*}}` / `{{user.*}}` / `{{output.*}}` placeholders with typed sources |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute (CLI + SDK) → Validate → Recover |
| 4 | **Complete Failure Strategies** | 12 CBR error codes with HALT vs retry per error type |
| 5 | **Absolute Single Responsibility** | One product (CBR), one primary resource (vault); cross-product delegation documented |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **FinOps** | Backup storage cost analysis, billing model comparison, retention policy cost optimization | `references/well-architected-assessment.md` §3 |
| **SecOps** | IAM minimum permissions, KMS encryption, vault policy, backup data isolation | `references/well-architected-assessment.md` §4 |
| **AIOps** | ≥ 4 anomaly patterns (backup failure spike, vault capacity, slow restore), cross-skill diagnosis | `references/well-architected-assessment.md` §5 |

### Well-Architected Framework Integration (卓越架构)

| Pillar | Skill Integration |
|--------|-------------------|
| **安全 (Security)** | IAM permissions, credential masking, KMS encryption, vault-level access policies |
| **稳定 (Stability)** | Automated backup policies, cross-region replication, retention rules, DR runbook |
| **成本 (Cost)** | Billing model comparison (按需 vs 包年包月), retention period optimization, storage tiering |
| **效率 (Efficiency)** | Batch resource association, policy-based automation, CLI pipeline |
| **性能 (Performance)** | Backup window optimization, incremental backup, bandwidth-aware replication |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud CBR" OR "cloud backup" OR "云备份" OR "备份" OR "存储库" OR "恢复"
- Task involves CRUD or lifecycle operations on backup vaults (create, delete, modify, list)
- Task involves backup policy creation, modification, association with resources
- Task involves on-demand backup execution, restore, or cross-region replication
- Task keywords: **backup**, **restore**, **vault**, **recovery point**, **replication**, **CBR**, **云备份**, **RPO**, **RTO**

### SHOULD NOT Use This Skill When

- Task is purely billing/account management → delegate to: billing skill (when present)
- Task is IAM/permission model only → delegate to: `huaweicloud-iam-ops`
- Task is about instance-level backup that CBR uses (ECS/DCS/RDS) → delegate to respective skill for instance operations
- Task is about monitoring/alarm rules → delegate to: `huaweicloud-ces-ops`
- Task is about audit log analysis → delegate to: `huaweicloud-cts-ops`

## Variables

| Variable | Source | Description | Example |
|----------|--------|-------------|---------|
| `{{env.HW_ACCESS_KEY_ID}}` | Environment | Huawei Cloud AK | `AKIA...` |
| `{{env.HW_SECRET_ACCESS_KEY}}` | Environment | Huawei Cloud SK | `***` (masked) |
| `{{env.HW_REGION_ID}}` | Environment | Region code | `cn-north-4` |
| `{{env.HW_PROJECT_ID}}` | Environment | Project ID | `a1b2c3d4...` |
| `{{user.vault_id}}` | User | Backup vault UUID | `vault-abc123` |
| `{{user.vault_name}}` | User | Vault name | `prod-ecs-backup-vault` |
| `{{user.vault_type}}` | User | Vault type | `server` (ECS), `disk` (EVS), `turbo` (SFS) |
| `{{user.billing_mode}}` | User | Billing mode | `prePaid` or `postPaid` |
| `{{user.storage_size}}` | User | Vault capacity (GB) | `1000` |
| `{{user.policy_id}}` | User | Backup policy UUID | `policy-abc123` |
| `{{user.policy_name}}` | User | Policy name | `daily-backup-00-00` |
| `{{user.backup_id}}` | User | Backup/restore point ID | `backup-abc123` |
| `{{user.resource_id}}` | User | Resource to backup (ECS/RDS ID) | `ecs-abc123` |
| `{{user.resource_type}}` | User | Resource type | `OS::Nova::Server` or `OS::Cinder::Volume` |
| `{{user.retention_days}}` | User | Backup retention in days | `30` |
| `{{user.replication_region}}` | User | Target region for replication | `cn-south-1` |
| `{{output.vault_id}}` | Output | Created vault ID | From API response |
| `{{output.backup_id}}` | Output | Created backup ID | From API response |

## Operations

### 1. Vault Management

#### 1.1 Create Backup Vault

**Pre-flight:**
- [ ] Verify `{{env.HW_ACCESS_KEY_ID}}` and `{{env.HW_SECRET_ACCESS_KEY}}` are set
- [ ] Choose vault type matching the resource type to protect
- [ ] Estimate storage capacity needs

**Execute (CLI):**
```bash
hcloud CBR CreateVault \
  --name="{{user.vault_name}}" \
  --type="{{user.vault_type}}" \
  --billing_mode="{{user.billing_mode}}" \
  --storage_size="{{user.storage_size}}" \
  --region="{{env.HW_REGION_ID}}"
```

**Execute (SDK - JIT Go):**
```go
package main

import (
    "fmt"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cbr/v3"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cbr/v3/model"
)

func main() {
    auth := basic.NewCredentialsBuilder().
        WithAk("{{env.HW_ACCESS_KEY_ID}}").
        WithSk("{{env.HW_SECRET_ACCESS_KEY}}").
        WithProjectId("{{env.HW_PROJECT_ID}}").
        Build()
    client := cbr.NewCbrClient(
        cbr.CbrClientBuilder().WithRegion("{{env.HW_REGION_ID}}").WithCredential(auth).Build(),
    )
    req := &model.CreateVaultReq{
        Name: "{{user.vault_name}}",
        Billing: &model.Billing{
            ObjectType: "{{user.vault_type}}",
            ObjectSize: int64({{user.storage_size}}),
        },
    }
    resp, err := client.CreateVault(req)
    if err != nil {
        panic(err)
    }
    fmt.Println(resp.Vault.Id)
}
```

**Validate:**
- [ ] Verify vault appears in `ListVaults` response
- [ ] Confirm vault status is `available`

**Recovery:**
| Error | Action |
|-------|--------|
| `CBR.0001` (quota exceeded) | Request quota increase or delete unused vaults |
| `CBR.0002` (invalid type) | Verify vault type matches resource type |
| `CBR.0003` (billing error) | Verify billing account has sufficient balance |

#### 1.2 List Vaults

**Execute (CLI):**
```bash
hcloud CBR ListVaults \
  --region="{{env.HW_REGION_ID}}"
```

**Execute (SDK - JIT Go):**
```go
req := &model.ListVaultsReq{}
resp, err := client.ListVaults(req)
for _, vault := range resp.Vaults {
    fmt.Printf("%s (%s) - %s, used: %dGB/%dGB\n",
        *vault.Name, *vault.Id, *vault.Billing.Status,
        *vault.Billing.Used, *vault.Billing.Size)
}
```

#### 1.3 Describe Vault

**Execute (CLI):**
```bash
hcloud CBR ShowVault \
  --vault_id="{{user.vault_id}}" \
  --region="{{env.HW_REGION_ID}}"
```

#### 1.4 Delete Vault

> **⚠️ DESTRUCTIVE OPERATION — All backups in vault will be permanently deleted.**

**Pre-flight:**
- [ ] Confirm vault ID with user
- [ ] Verify no resources are still associated with the vault
- [ ] Confirm user has exported or verified no backups are needed

**Execute (CLI):**
```bash
hcloud CBR DeleteVault \
  --vault_id="{{user.vault_id}}" \
  --region="{{env.HW_REGION_ID}}"
```

**Validate:**
- [ ] Vault no longer appears in `ListVaults`

### 2. Backup Policy Management

#### 2.1 Create Backup Policy

**Pre-flight:**
- [ ] Define backup schedule (frequency, time window)
- [ ] Define retention rules (max backups, retention days)

**Execute (CLI):**
```bash
hcloud CBR CreatePolicy \
  --name="{{user.policy_name}}" \
  --type="backup" \
  --schedule="FREQ=DAILY;INTERVAL=1;BYHOUR=0;BYMINUTE=0" \
  --retention_days="{{user.retention_days}}" \
  --region="{{env.HW_REGION_ID}}"
```

**Validate:**
- [ ] Policy appears in `ListPolicies`
- [ ] Verify schedule and retention parameters match input

#### 2.2 Associate Policy with Vault

**Execute (CLI):**
```bash
hcloud CBR AssociateVaultPolicy \
  --vault_id="{{user.vault_id}}" \
  --policy_id="{{user.policy_id}}" \
  --region="{{env.HW_REGION_ID}}"
```

#### 2.3 List Policies

**Execute (CLI):**
```bash
hcloud CBR ListPolicies \
  --region="{{env.HW_REGION_ID}}"
```

### 3. Backup Execution & Restore

#### 3.1 Create On-Demand Backup

**Pre-flight:**
- [ ] Identify vault ID and resource ID to backup
- [ ] Verify resource is in active/running state

**Execute (CLI):**
```bash
hcloud CBR CreateBackup \
  --vault_id="{{user.vault_id}}" \
  --resource_id="{{user.resource_id}}" \
  --resource_type="{{user.resource_type}}" \
  --name="manual-backup-$(date +%Y%m%d)" \
  --region="{{env.HW_REGION_ID}}"
```

**Validate:**
- [ ] Backup appears in `ListBackups` within 60 seconds
- [ ] Backup status transitions to `available`

#### 3.2 List Backups

**Execute (CLI):**
```bash
hcloud CBR ListBackups \
  --vault_id="{{user.vault_id}}" \
  --region="{{env.HW_REGION_ID}}"
```

#### 3.3 Restore from Backup

> **⚠️ DESTRUCTIVE OPERATION — Existing resource state will be overwritten.**

**Pre-flight:**
- [ ] Confirm restore with user — this overwrites current resource data
- [ ] Identify backup ID from `ListBackups`
- [ ] Verify the target resource is not in production critical usage

**Execute (CLI):**
```bash
hcloud CBR RestoreBackup \
  --backup_id="{{user.backup_id}}" \
  --resource_id="{{user.resource_id}}" \
  --region="{{env.HW_REGION_ID}}"
```

**Validate:**
- [ ] Resource status transitions to `RESTORING` then back to `ACTIVE`
- [ ] Verify data integrity post-restore

### 4. Cross-Region Replication

**Pre-flight:**
- [ ] Ensure destination region has CBR service enabled
- [ ] Verify vault exists in destination region (or create one)

**Execute (CLI):**
```bash
hcloud CBR ReplicateBackup \
  --backup_id="{{user.backup_id}}" \
  --destination_region="{{user.replication_region}}" \
  --destination_vault_id="{{user.vault_id}}" \
  --region="{{env.HW_REGION_ID}}"
```

**Validate:**
- [ ] Replication task completes with status `success`
- [ ] Backup appears in destination region's `ListBackups`

### 5. Monitoring

**Execute (CLI):**
```bash
# Query CBR metrics via CES
hcloud CES ListMetrics \
  --namespace="SYS.CBR" \
  --region="{{env.HW_REGION_ID}}"

# Query vault storage usage
hcloud CES ShowMetricData \
  --namespace="SYS.CBR" \
  --metric_name="vault_used_percent" \
  --dim="vault_id={{user.vault_id}}" \
  --period="3600" \
  --from="{{user.start_time}}" \
  --to="{{user.end_time}}" \
  --region="{{env.HW_REGION_ID}}"
```

## Failure Recovery

### Error Code Taxonomy

| Code | Category | Action |
|------|----------|--------|
| `CBR.0001` | Quota | HALT — Request quota increase or delete unused vaults |
| `CBR.0002` | Config | Verify vault type matches resource type |
| `CBR.0003` | Config | Check account balance and billing status |
| `CBR.0004` | NotFound | Verify vault ID is correct |
| `CBR.0005` | NotFound | Verify resource ID exists and is active |
| `CBR.0006` | NotFound | Verify backup ID is correct |
| `CBR.0007` | Resource | Resize vault or delete old backups |
| `CBR.0008` | State | Wait for current backup to complete (retry 3×, 60s backoff) |
| `CBR.0009` | Config | Modify policy schedule to avoid overlap |
| `CBR.0010` | Conflict | Disassociate first or use different vault |
| `CBR.0011` | Config | Verify CBR is available in target region |
| `CBR.0012` | Resource | Reduce replication concurrency (retry 3×, 120s backoff) |

### Diagnostic Flow (Backup Failure)

```mermaid
flowchart TD
    A[Backup failed] --> B{Check error code}
    B -->|CBR.0007| C[Check vault capacity]
    C --> D[Resize vault or delete old backups]
    B -->|CBR.0005| E[Verify resource exists]
    E --> F[Check resource status is ACTIVE]
    B -->|Other| G[Check CES backup_failure metric]
    G --> H{Retryable?}
    H -->|Yes| I[Retry with backoff: 30s, 60s, 120s]
    H -->|No| J[HALT - Report error to user]
```

### Diagnostic Flow (Restore Failure)

```mermaid
flowchart TD
    A[Restore failed] --> B{Backup status available?}
    B -->|No| C[Wait for backup to complete]
    B -->|Yes| D{Target resource exists?}
    D -->|No| E[Create target resource first]
    D -->|Yes| F{Insufficient capacity?}
    F -->|Yes| G[Resize target resource]
    F -->|No| H[Contact HW support with backup_id]
```

## Well-Architected Assessment

This skill follows the Huawei Cloud Well-Architected Framework across five pillars plus FinOps, SecOps, and AIOps. See:

- **Full Assessment:** [`references/well-architected-assessment.md`](references/well-architected-assessment.md)
- **Core Concepts:** [`references/core-concepts.md`](references/core-concepts.md)
- **API & SDK Usage:** [`references/api-sdk-usage.md`](references/api-sdk-usage.md)
- **CLI Usage:** [`references/cli-usage.md`](references/cli-usage.md)
- **Troubleshooting:** [`references/troubleshooting.md`](references/troubleshooting.md)
- **Monitoring:** [`references/monitoring.md`](references/monitoring.md)
- **Integration:** [`references/integration.md`](references/integration.md)
- **Idempotency Checklist:** [`references/idempotency-checklist.md`](references/idempotency-checklist.md)

## Quality Gate (GCL)

This skill is **GCL-required** (per `AGENTS.md` §8). Every CBR mutating operation — vault
create / delete, policy create / update / delete, backup create / copy / delete, **restore**
(the most dangerous: overwrites target) — runs through the **Generator-Critic-Loop** before
its result is returned. Read-only are GCL-**exempt**.

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
| 1 | Correctness | ≥ 0.5 (1.0 for `restore` / `delete-vault` / `delete-backup`) | `ShowVault` / `ShowBackup` / `ShowPolicy` post-state |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | S1–S15 in rubric §2 |
| 3 | Idempotency | ≥ 0.5 | Pre-check before create; see also `references/idempotency-checklist.md` |
| 4 | Traceability | ≥ 0.5 | `password` MUST be `<masked>` |
| 5 | Spec Compliance | ≥ 0.5 | Resource type / vault size / retention / name regex |

### Per-Operation Safety Anchors (binding)

- **S1 / S2 / S3** — `restore` confirmation / target disk attached check / size check
- **S4** — `restore` to different server/disk without two-step
- **S5 / S6** — `delete-vault` with backups / prePaid refund
- **S7 / S8** — `delete-backup` only valid / status check
- **S9** — `copy-backup` to a region without destination vault
- **S10** — `update-policy` `retention_duration_days < 7`
- **S14** — `restore` with OS type mismatch (Linux ↔ Windows)
- **S15** — `create-backup` while another backup for same resource is RUNNING

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

- [`references/rubric.md`](references/rubric.md) — full rubric, S1–S15 rules, per-op thresholds
- [`references/prompt-templates.md`](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons
- Repository root [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8 — GCL specification

## Appendices

### A. References

- [Huawei Cloud CBR API Documentation](https://support.huaweicloud.com/api-cbr/)
- [Huawei Cloud Go SDK](https://github.com/huaweicloud/huaweicloud-sdk-go-v3)
- [Huawei Cloud CLI](https://support.huaweicloud.com/hcli/index.html)
- [GCL Rubric](references/rubric.md) — Adversarial quality gate (v1, 5-dim, S1–S15 CBR-specific Safety rules; restore-target-disk-validation cascade)
- [GCL Prompt Templates](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons

### B. Changelog

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-05-21 | Initial CBR ops skill with vault, policy, backup, restore, replication |
