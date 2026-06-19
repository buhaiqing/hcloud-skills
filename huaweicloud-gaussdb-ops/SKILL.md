---
name: huaweicloud-gaussdb-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei
  Cloud GaussDB (for openGauss) — instance lifecycle, backup/restore, parameter
  templates, database/account management, tags, quotas, and recycle bin. User
  mentions GaussDB, gaussdb, openGauss, 高斯数据库, 分布式数据库, or describes
  database scenarios (e.g., connection refused, slow query, disk full, backup
  failure, need to create database/user) even without naming the product
  directly. Not for billing, IAM, or RDS (they have their own ops skills).
license: MIT
compatibility: >-
  Official Huawei Cloud CLI (`hcloud`), Go 1.21+ runtime
  (for JIT SDK fallback via huaweicloud-sdk-go-v3), valid API credentials,
  network access to Huawei Cloud GaussDB endpoints.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-05-21"
  runtime: Harness AI Agent, Claude Code, Cursor, or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  api_profile: "GaussDB for openGauss v3 — https://support.huaweicloud.com/api-gaussdb/"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    GaussDB operations available via `hcloud GaussDB <operation>` where
    operation matches the API Explorer name: ListInstances, ShowInstanceDetail,
    CreateInstance, DeleteInstance, ListBackups, CreateManualBackup,
    ListConfigurations, ApplyConfiguration, ListDatabases, CreateDatabase,
    ListDbUsers, CreateDbUser, etc.
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
        change: "GCL Phase 2 rollout: added references/rubric.md (v1, 5-dim, S1–S17 GaussDB-specific Safety rules, with flavor-gated S12/S13 for DWS) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
      - version: "1.0.0"
        date: "2026-05-21"
        change: "Initial skill release."
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud GaussDB (for openGauss) Operations Skill

## Overview

GaussDB (for openGauss) is Huawei Cloud's enterprise-grade distributed relational database, compatible with openGauss and PostgreSQL ecosystems. It supports deployment modes: **Centralized Standard** (single node), **Distributed Enterprise** (multiple shards with HA), and **Read Replica**. This skill is an **operational runbook** for agents: explicit scope, credential rules, pre-flight checks, **dual-path execution** (official `hcloud` CLI and JIT Go SDK fallback), response validation, and failure recovery.

**API Version**: v3 (openGauss engine) — Go SDK `services/gaussdb/v3`

### CLI applicability (repository policy)

- **`cli_applicability: dual-path`:** Official CLI supports GaussDB. In each execution flow below, document **both** the SDK step **and** the CLI step.

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions below with delegation rules |
| 2 | **Structured I/O** | `{{env.*}}` for credentials, `{{user.*}}` for interactive input, `{{output.*}}` for response capture |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute (CLI + SDK) → Validate → Recover |
| 4 | **Complete Failure Strategies** | 18 service error codes documented; HALT vs retry per error type |
| 5 | **Absolute Single Responsibility** | One product (GaussDB); cross-product delegation to other skills |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **FinOps** | Right-sizing flavors, backup retention optimization, idle instance detection | `references/advanced/cost-optimization.md` |
| **SecOps** | IAM policies, KMS encryption, network isolation, password rotation, SSL | `references/advanced/security-best-practices.md` |
| **AIOps** | 4 anomaly patterns (storage, backup, connections, long-running tasks) | `references/advanced/aiops-patterns.md` |

### Well-Architected Framework Integration (卓越架构)

| Pillar | Skill Integration |
|--------|-------------------|
| **安全 (Security)** | IAM minimum permissions, KMS disk encryption, SSL/TLS connections, VPC isolation |
| **稳定 (Stability)** | Multi-AZ HA deployment, automated backup policy, restore runbook |
| **成本 (Cost)** | Flavor comparison, idle detection, backup retention tuning |
| **效率 (Efficiency)** | Batch CLI operations, parameter template reuse, CI/CD integration |
| **性能 (Performance)** | Connection pool sizing, query optimization, storage scaling triggers |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud GaussDB" / "GaussDB for openGauss" / "高斯数据库" / "分布式数据库 GaussDB"
- Task involves CRUD or lifecycle management of GaussDB instances (create, list, show, delete, resize, restart)
- Task involves database/account operations (create/list/delete databases, users, schemas, roles)
- Task involves backup/restore (list/create/delete backups, set/show backup policy, restore)
- Task involves parameter templates (list/create/apply/compare/delete)
- Task involves tags, enterprise quotas, or recycle bin management
- Task keywords: **GaussDB**, **openGauss**, **database**, **instance**, **backup**, **restore**, **parameter**, **template**, **高斯**, **分布式数据库**
- User describes symptoms: connection refused, disk full, backup failure, slow query, password reset needed

### SHOULD NOT Use This Skill When
- Task is purely billing / cost analysis / 费用 / 预算 → delegate to: `huaweicloud-billing-ops`

- Task is IAM / permission model only → delegate to: `huaweicloud-iam-ops`
- Task is about RDS MySQL/PostgreSQL → delegate to: `huaweicloud-rds-ops`
- Task is about DCS Redis/Memcached → delegate to: `huaweicloud-dcs-ops`
- Task is about VPC/subnet/security group → delegate to: `huaweicloud-vpc-ops`
- Task is about monitoring/alarm rules → delegate to: `huaweicloud-ces-ops`

### Delegation Rules

- Instance must be ACTIVE before database/user operations → verify with `ShowInstanceDetail` first
- Backup operations depend on instance existence → confirm `instance_id` before backup
- Parameter templates require compatible datastore version → verify version match before `ApplyConfiguration`
- For FinOps questions: use this skill's cost section; delegate cross-resource cost to billing skill
- For SecOps questions: use this skill's security section; delegate account-level IAM to `huaweicloud-iam-ops`

## Variables

| Variable | Source | Description | Example |
|----------|--------|-------------|---------|
| `{{env.HW_ACCESS_KEY_ID}}` | Environment | Huawei Cloud AK | `AKIA...` |
| `{{env.HW_SECRET_ACCESS_KEY}}` | Environment | Huawei Cloud SK | `***` (masked) |
| `{{env.HW_REGION_ID}}` | Environment | Region code | `ap-southeast-1` |
| `{{env.HW_PROJECT_ID}}` | Environment | Project ID | `a1b2c3d4...` |
| `{{user.instance_id}}` | User | GaussDB instance UUID | `ed7cc616...in14` |
| `{{user.instance_name}}` | User | GaussDB instance name | `prod-gauss-01` |
| `{{user.db_name}}` | User | Database name | `appdb` |
| `{{user.db_user}}` | User | Database username | `app_admin` |
| `{{user.backup_name}}` | User | Backup name | `manual-20260521` |
| `{{user.flavor_ref}}` | User | Instance spec code | `gaussdb.opengauss.4xlarge.x864.8` |
| `{{output.instance_id}}` | API Response | Created instance ID | from `CreateInstance` |
| `{{output.backup_id}}` | API Response | Created backup ID | from `CreateManualBackup` |

> **Security Warning:** NEVER log or expose `{{env.HW_SECRET_ACCESS_KEY}}` or any credential values.

---

## Common Operations

### 1. Instance Lifecycle

| Operation | CLI Command (KooCLI) | Equivalent Go SDK |
|-----------|---------------------|-------------------|
| List instances | `hcloud GaussDB ListInstances --cli-region="{{env.REGION}}"` | `gaussdbClient.ListInstances()` |
| Show instance detail | `hcloud GaussDB ShowInstanceDetail --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.ShowInstanceDetail()` |
| Create instance | `hcloud GaussDB CreateInstance` | `gaussdbClient.CreateInstance()` |
| Delete instance | `hcloud GaussDB DeleteInstance --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.DeleteInstance()` |
| Modify name | `hcloud GaussDB UpdateInstanceName --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.UpdateInstanceName()` |
| Scale storage | `hcloud GaussDB ResizeInstanceFlavor` | `gaussdbClient.ResizeInstanceFlavor()` |
| Reboot | `hcloud GaussDB RestartInstance --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.RestartInstance()` |
| Reset password | `hcloud GaussDB ResetPwd --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.ResetPwd()` |
| Add CN | `hcloud GaussDB AddInstanceCN --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.AddInstanceCN()` |
| Add DN | `hcloud GaussDB ExpandInstanceDN --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.ExpandInstanceDN()` |
| Bind EIP | `hcloud GaussDB SetDbUserPwd --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.SetDbUserPwd()` |

### 2. Backup & Restoration

| Operation | CLI Command | Equivalent Go SDK |
|-----------|-------------|-------------------|
| List backups | `hcloud GaussDB ListBackups` | `gaussdbClient.ListBackups()` |
| Create manual backup | `hcloud GaussDB CreateManualBackup` | `gaussdbClient.CreateManualBackup()` |
| Delete manual backup | `hcloud GaussDB DeleteManualBackup --backup_id="{{env.GAUSSDB_BACKUP_ID}}"` | `gaussdbClient.DeleteManualBackup()` |
| Set backup policy | `hcloud GaussDB SetBackupPolicy` | `gaussdbClient.SetBackupPolicy()` |
| Show backup policy | `hcloud GaussDB ShowBackupPolicy` | `gaussdbClient.ShowBackupPolicy()` |
| Restore to new instance | `hcloud GaussDB RestoreInstance` | `gaussdbClient.RestoreInstance()` |

### 3. Parameter Templates

| Operation | CLI Command | Equivalent Go SDK |
|-----------|-------------|-------------------|
| List templates | `hcloud GaussDB ListConfigurations` | `gaussdbClient.ListConfigurations()` |
| Show template detail | `hcloud GaussDB ShowConfigurationSetting` | `gaussdbClient.ShowConfigurationSetting()` |
| Create template | `hcloud GaussDB CreateConfigurationTemplate` | `gaussdbClient.CreateConfigurationTemplate()` |
| Apply template | `hcloud GaussDB ApplyConfiguration --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.ApplyConfiguration()` |
| Compare templates | `hcloud GaussDB ListDiffDetails` | `gaussdbClient.ListDiffDetails()` |

### 4. Database & Account Administration

| Operation | CLI Command | Equivalent Go SDK |
|-----------|-------------|-------------------|
| List databases | `hcloud GaussDB ListDatabases --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.ListDatabases()` |
| Create database | `hcloud GaussDB CreateDatabase --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.CreateDatabase()` |
| List database users | `hcloud GaussDB ListDbUsers --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.ListDbUsers()` |
| Create database user | `hcloud GaussDB CreateDbUser --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.CreateDbUser()` |
| List schemas | `hcloud GaussDB ListSchemas --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.ListSchemas()` |
| Create schema | `hcloud GaussDB CreateSchema --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.CreateSchema()` |

### 5. Monitoring & Tasks

| Operation | CLI Command | Equivalent Go SDK |
|-----------|-------------|-------------------|
| List tasks | `hcloud GaussDB ListTasks` | `gaussdbClient.ListTasks()` |
| List instance tags | `hcloud GaussDB ShowInstanceConfiguration --instance_id="{{env.GAUSSDB_INSTANCE_ID}}"` | `gaussdbClient.ShowInstanceConfiguration()` |

---

## Cost Optimization (FinOps)

1. **Instance Sizing**: Use `ListFlavors()` (CLI: `hcloud GaussDB ListFlavors`) to compare available instance specifications before provisioning. Match workload to `flavor_ref` tiers.
2. **Storage Scaling**: Start with `ULTRAHIGH` SSD and scale up as needed. Use `ShowInstanceDetail()` to monitor `disk_usage`.
3. **Automated Backup Retention**: Keep backups per compliance. Shorter retention reduces storage costs. Use `SetBackupPolicy()` to adjust `keep_days`.
4. **Idle Instance Detection**: Query instances with low `disk_usage` and no recent `backup_used_space` growth via `ListInstances()`.
5. **Read Replica Cleanup**: Remove unused read replicas. List all replicas with `ListInstances(type="Readonly")` and delete stale ones.

---

## Security Best Practices (SecOps)

1. **IAM Fine-Grained Control**: Use `gaussdb:*` action-level policies. Restrict `DeleteInstance`, `ResetPwd`, and `CreateDbUser` to admin roles only.
2. **Network Isolation**: Deploy GaussDB in private VPC subnets. Bind EIP only for necessary data migration, then unbind immediately using `SetDbUserPwd()` (EIP binding operations).
3. **Encryption**: Enable disk encryption during instance creation (`disk_encryption_id` parameter). Use KMS-managed keys.
4. **SSL Connections**: Download SSL certificate via `ShowSslCertDownloadLink()` and enforce ssl=on for client connections.
5. **Account Password Policy**: Rotate passwords regularly via `ResetPwd()`. Enforce password complexity at the application level (minimum 8 chars, mixed case, digits, special chars).

---

## Intelligent Operations (AIOps)

### Anomaly Pattern 1: Storage Exhaustion
```bash
# Check disk usage across all instances
hcloud GaussDB ListInstances --cli-region="{{env.REGION}}" --query "instances[?disk_usage>='90']"
```
**Remediation**: Scale storage via `ResizeInstanceFlavor()` or archive old data.

### Anomaly Pattern 2: Backup Failure
```bash
# Find failed backups in last 7 days
hcloud GaussDB ListBackups --backup_type=manual --query "backups[?status=='FAILED']"
```
**Remediation**: Check disk space, then retry: `hcloud GaussDB CreateManualBackup`.

### Anomaly Pattern 3: High Connection Saturation
Monitor via CloudEye or check `ListTasks()` for connection-related tasks. Scale instance flavor or adjust `max_connections` parameter.

### Anomaly Pattern 4: Long-Running Tasks
```bash
# Find tasks running longer than expected
hcloud GaussDB ListTasks --query "tasks[?status=='Running']"
```
**Remediation**: Review task type. Cancel via support ticket if stuck.

---

## Safety Gates (High-Risk Operations)

| Operation | Risk Level | Pre-flight Check |
|-----------|-----------|------------------|
| `DeleteInstance` | **CRITICAL** | Verify backup exists, confirm instance ID, require `--confirm` flag |
| `ResetPwd` | **HIGH** | Notify app team, schedule maintenance window |
| `ResizeInstanceFlavor` | **HIGH** | Verify instance status is `ACTIVE`, estimate downtime window |
| `DeleteManualBackup` | **MEDIUM** | Ensure at least one valid backup remains |
| `ApplyConfiguration` | **MEDIUM** | Review parameter diff, plan for potential restart |

**Gate Pattern**:
```bash
# Before destructive operations: dry-run check
hcloud GaussDB ShowInstanceDetail \
  --instance_id="{{env.GAUSSDB_INSTANCE_ID}}" \
  --cli-region="{{env.REGION}}" \
  > /tmp/gaussdb-precheck.json
grep "ACTIVE" /tmp/gaussdb-precheck.json || { echo "[ERROR] Instance not ACTIVE — abort"; exit 1; }
echo "CONFIRM: Type 'yes' to proceed"; read -r ans; [ "$ans" = "yes" ] || exit 1
```

---

## Error Troubleshooting

| Code | HTTP | Fix |
|------|------|-----|
| `DBS.200001` | 400 | Check request parameters against API docs |
| `DBS.200010` | 400 | Ensure instance is ACTIVE |
| `DBS.200012` | 400 | Perform operation on primary instance |
| `DBS.200301` | 403 | Apply for higher instance/storage quota |
| `DBS.200404` | 404 | Verify instance_id exists and region |
| `DBS.200409` | 409 | Wait for current task to complete |
| `DBS.200500` | 500 | Retry with exponential backoff |

---

## Quality Gate (GCL)

This skill is **GCL-required** (per `AGENTS.md` §8). Every GaussDB mutating operation — instance
create / delete / resize, backup create / delete, parameter change, account / database admin,
shard rebalance (DWS) — runs through the **Generator-Critic-Loop** before its result is returned.
Read-only `show*` / `list*` are GCL-**exempt**.

| Field | Value |
|-------|-------|
| Rubric version | v1 (Phase 2, 2026-06-04) |
| `max_iter` | **2** |
| Rubric instance | [`references/rubric.md`](references/rubric.md) |
| Prompt templates | [`references/prompt-templates.md`](references/prompt-templates.md) |
| Trace path | `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` |
| Independence | Generator and Critic in **isolated** sub-agent / session contexts |
| Flavor gating | Critic applies S12 / S13 ONLY when `deployment == "dws"` |

### Five-Dimension Rubric (summary)

| # | Dimension | Threshold | Notes |
|---|-----------|-----------|-------|
| 1 | Correctness | ≥ 0.5 (1.0 for `DeleteInstance` / DDL / shard rebalance) | `ShowInstanceDetail` / `ShowBackup` / `ShowClusterTopology` post-state |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | S1–S17 in rubric §2; S12/S13 flavor-gated |
| 3 | Idempotency | ≥ 0.5 | Pre-check before create |
| 4 | Traceability | ≥ 0.5 | `password` MUST be `<masked>`; never in CLI args |
| 5 | Spec Compliance | ≥ 0.5 | Engine version / flavor regex / node count / db & user name regex |

### Per-Operation Safety Anchors (binding)

- **S1 / S2** — `DeleteInstance` without explicit confirmation / while `status != ACTIVE`
- **S3** — `DeleteInstance` for prePaid instance with > 7 days remaining
- **S4** — `DeleteInstance` while latest automated backup is missing / failed
- **S5** — `ResizeInstanceFlavor` (downsize) without verifying `ACTIVE` first
- **S6** — `ApplyConfiguration` restart-required on prod-tagged instance, no maintenance window
- **S7** — `DeleteManualBackup` while `status != COMPLETED` or only valid backup
- **S8** — `ResetPwd` with password in CLI args or in trace
- **S9** — `CreateAccount` `ALL PRIVILEGES + GRANT + *.*` to non-admin
- **S10 / S11** — `CreateDatabase` SQL injection in name / `DeleteDatabase` for system DB
- **S12** *(DWS only)* — Shard rebalance without two-step confirmation
- **S13** *(DWS only)* — `UpdateInstance` decreasing DN/CN below `min_replicas` floor
- **S14** — Any trace contains `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / `sk-…` / `password`
- **S15** — `CreateInstance` with `region` / `project_id` not in env contract
- **S16** — `ApplyConfiguration` `wal_level` change on active primary
- **S17** — `DeleteInstance` while read-replica count > 0

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
- Repository root [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8 — GCL specification

## References

- [API Navigation](references/api-navigation.md) — Full API catalog
- [CLI Syntax Reference](references/cli-syntax-reference.md) — KooCLI detailed usage
- [Common Faults](references/common-faults.md) — Troubleshooting guide
- [Error Handling](references/error-handling.md) — Error code matrix
- [Cost Optimization](references/advanced/cost-optimization.md) — FinOps deep dive
- [Security Best Practices](references/advanced/security-best-practices.md) — SecOps hardening
- [AIOps Patterns](references/advanced/aiops-patterns.md) — Anomaly detection + remediation
- [Safety Gates](references/advanced/safety-gates.md) — High-risk operation controls
- [Example Config](assets/example-config.yaml) — Reference configuration
- [Example Output](assets/example-output.json) — Sample API responses
- [GCL Rubric](references/rubric.md) — Adversarial quality gate (v1, 5-dim, S1–S17 GaussDB-specific Safety rules; S12/S13 DWS-gated)
- [GCL Prompt Templates](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons
