---
name: huaweicloud-rds-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei
  Cloud RDS (Relational Database Service) — DB instance lifecycle, backup/restore,
  parameter management, performance monitoring, and diagnostics. User mentions RDS,
  云数据库, MySQL, PostgreSQL, SQL Server, or describes database-related scenarios
  (e.g., connection drops, slow queries, storage full, instance creation failures)
  even without naming the product directly. Not for billing, IAM, or related products
  that have their own ops skills.
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
  api_profile: "RDS API v3 (Recommended) - https://support.huaweicloud.com/api-rds/"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    hcloud supports RDS operations via `hcloud rds` command group.
    JIT Go SDK fallback available via huaweicloud-sdk-go-v3/services/rds/v3
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
        change: "GCL Phase 2 rollout: added references/rubric.md (v1, 5-dim, S1–S15 RDS-specific Safety rules) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
      - version: "1.0.0"
        date: "2026-05-20"
        change: "Initial skill release."
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
  supported_engines:
    - MySQL
    - PostgreSQL
    - SQL Server
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud RDS Operations Skill

## Overview

Huawei Cloud Relational Database Service (RDS) provides a reliable, scalable, and manageable online relational database service running on a cloud computing platform. RDS includes a comprehensive performance monitoring system, multi-level security measures, and a professional database management platform. This skill is an **operational runbook** for agents: explicit scope, credential rules, pre-flight checks, **dual-path execution** (official **SDK/API** and matching **CLI** flows), response validation, and failure recovery.

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
| **FinOps (财务运营)** | Cost visibility, right-sizing, billing model comparison, waste detection | `references/well-architected-assessment.md` §3 |
| **SecOps (安全运营)** | IAM minimum permissions, network isolation, encryption, threat detection | `references/well-architected-assessment.md` §4 |
| **AIOps (智能运营)** | Multi-metric correlation, cross-skill diagnosis, knowledge base, self-healing | `references/advanced/aiops-best-practices.md` |

### Well-Architected Framework Integration (卓越架构)

This skill maps operations to Huawei Cloud's Well-Architected Framework five pillars:

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **安全 (Security)** | IAM permissions, credential masking, network isolation, TDE encryption | `references/well-architected-assessment.md` §2.1 |
| **稳定 (Stability)** | Backup/restore, multi-AZ, DR runbook, failure-oriented design | `references/well-architected-assessment.md` §2.2 |
| **成本 (Cost)** | Billing model comparison, waste detection, right-sizing | `references/well-architected-assessment.md` §2.3 |
| **效率 (Efficiency)** | Batch operations, CI/CD integration, automation patterns | `references/well-architected-assessment.md` §2.4 |
| **性能 (Performance)** | Metrics, auto-scaling, performance baselines | `references/well-architected-assessment.md` §2.5 |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud RDS" OR "云数据库" OR "关系型数据库"
- User mentions database engines: "MySQL", "PostgreSQL", "SQL Server"
- Task involves CRUD or lifecycle operations on **DB instances**
- Task keywords: instance creation, backup, restore, database creation, user management, parameter modification, monitoring, slow query
- User asks to deploy, configure, troubleshoot, or monitor RDS **via API, SDK, CLI, or automation**
- User reports database connection issues, performance degradation, storage alerts

### SHOULD NOT Use This Skill When

- Task is purely billing / account management → delegate to: `huaweicloud-billing-ops`
- Task is IAM / permission model only → delegate to: `huaweicloud-iam-ops` (when present)
- Task is about other database services (GaussDB, DCS) → delegate to: `huaweicloud-gaussdb-ops` or `huaweicloud-dcs-ops`

### Delegation Rules

- If resource B depends on resource A, complete or verify A before B's SDK or CLI steps.
- VPC and Security Group must exist before creating RDS instance → delegate to `huaweicloud-vpc-ops` if needed.
- Multi-product requests: handle each product with its skill; do not merge unrelated APIs into one ambiguous flow.
- For FinOps questions involving RDS resources: use this skill's cost section, delegate cross-resource cost to billing skill.
- For SecOps questions: use this skill's security section, delegate account-level IAM to IAM skill.
- For CES monitoring: delegate metric queries to `huaweicloud-ces-ops`.

## Variable Convention (Agent-Readable)

| Placeholder | Meaning | Agent Action |
|-------------|---------|--------------|
| `{{env.HW_ACCESS_KEY_ID}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_REGION_ID}}` | From runtime environment | Use documented default only if skill explicitly allows |
| `{{env.HW_PROJECT_ID}}` | From runtime environment | Use documented default only if skill explicitly allows |
| `{{user.region}}` | User-supplied region | Ask once; reuse |
| `{{user.instance_name}}` | User-supplied instance name | Ask once; reuse |
| `{{user.engine}}` | Database engine type (MySQL/PostgreSQL/SQL Server) | Ask once; reuse |
| `{{user.engine_version}}` | Database engine version | Ask once; reuse |
| `{{user.flavor_ref}}` | Instance specification code | Ask once; reuse |
| `{{user.vpc_id}}` | VPC ID | Ask once; validate exists |
| `{{user.subnet_id}}` | Subnet ID | Ask once; validate exists |
| `{{user.security_group_id}}` | Security group ID | Ask once; validate exists |
| `{{user.availability_zone}}` | Availability zone ID | Ask once; reuse |
| `{{output.instance_id}}` | From last API or CLI JSON response | Parse per **OpenAPI** path for this operation |
| `{{output.db_port}}` | Database connection port | From instance detail response |
| `{{output.private_ip}}` | Private IP address | From instance detail response |

> **`{{env.*}}` MUST NOT** be collected from the user. **`{{user.*}}`** MUST be collected interactively when missing.

> **Security Warning (Credential Masking — MANDATORY):** **NEVER** log, print, or expose `HW_SECRET_ACCESS_KEY`, database passwords, or any credential field value in console output, debug messages, error messages, or logs.

## API and Response Conventions

- **OpenAPI is canonical** for path, query, body fields, enums, and response shapes.
- **Errors:** Map SDK/HTTP errors to `code` / `status` / message fields per spec.
- **Timestamps:** ISO 8601 with timezone when the API returns strings.
- **Idempotency:** Document client request tokens, duplicate names, and `ResourceAlreadyExists` behavior per API.
- **Async Operations:** RDS CreateInstance, Resize, Restore are async. Poll until status becomes `ACTIVE` or terminal failure.

## Quick Start

### What This Skill Does
This skill enables deployment, configuration, troubleshooting, and monitoring of Huawei Cloud RDS database instances using the `hcloud` CLI (primary) or JIT Go SDK (fallback).

### Prerequisites

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|-----------|-------------|------------|------------|
| Create Instance | Create a new DB instance (single/HA/read replica) | High | Medium |
| Describe Instance | View instance details and status | Low | None |
| Modify Instance | Change configuration, parameters | Medium | Medium |
| Delete Instance | Remove a DB instance | Low | **High** — irreversible, data loss |
| List Instances | View all DB instances | Low | None |
| Create Backup | Manual/automatic backup | Medium | Low |
| Restore Instance | Restore from backup or PITR | High | High — data overwrite |
| Create Database | Create database within instance | Low | Low |
| Manage Users | Create/modify/delete DB users | Medium | Medium |
| Modify Parameters | Update parameter group settings | Medium | Medium |

## Execution Flows

### Operation: Create DB Instance

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| CLI / deps | `hcloud --version` | Exit code 0 | Document CLI install |
| Credentials | Construct credential from env | Non-empty keys | HALT; user configures env |
| Region | Validate `{{user.region}}` is valid Huawei Cloud region | Supported region | Suggest valid regions |
| VPC | Call **ListVpcs** or verify `{{user.vpc_id}}` | VPC exists | HALT; create VPC first |
| Subnet | Verify `{{user.subnet_id}}` in VPC | Subnet exists | HALT; create subnet |
| Security Group | Verify `{{user.security_group_id}}` | SG exists | HALT; create security group |
| Quota | Call **ListQuotas** | Sufficient quota | HALT; user raises quota |
| Flavor | Validate `{{user.flavor_ref}}` against available flavors | Valid flavor | List available flavors |

#### Execution — CLI (Primary Path)

```bash
# CLI invocation for MySQL instance
hcloud rds create \
  --region "{{user.region}}" \
  --name "{{user.instance_name}}" \
  --engine "{{user.engine}}" \
  --engine-version "{{user.engine_version}}" \
  --flavor-ref "{{user.flavor_ref}}" \
  --vpc-id "{{user.vpc_id}}" \
  --subnet-id "{{user.subnet_id}}" \
  --security-group-id "{{user.security_group_id}}" \
  --availability-zone "{{user.availability_zone}}" \
  --volume-type "{{user.volume_type}}" \
  --volume-size "{{user.volume_size}}"
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
    "rds" "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3"
    "rds_model" "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/rds/v3/model"
)

func main() {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    region := os.Getenv("HW_REGION_ID")
    projectId := os.Getenv("HW_PROJECT_ID")
    
    cfg := config.DefaultHttpConfig()
    client := rds.RdsClientBuilder().
        WithEndpoint(fmt.Sprintf("rds.%s.myhuaweicloud.com", region)).
        WithCredential(basic.NewCredentialsBuilder().
            WithAk(ak).WithSk(sk).Build()).
        WithHttpConfig(cfg).Build()
    
    request := &rds_model.CreateInstanceRequest{
        Body: &rds_model.CreateInstanceRequestBody{
            Name: "{{user.instance_name}}",
            Datastore: &rds_model.Datastore{
                Type:    "{{user.engine}}",
                Version: "{{user.engine_version}}",
            },
            FlavorRef: "{{user.flavor_ref}}",
            VpcId:     "{{user.vpc_id}}",
            SubnetId:  "{{user.subnet_id}}",
            SecurityGroupId: "{{user.security_group_id}}",
            AvailabilityZone: "{{user.availability_zone}}",
            Volume: &rds_model.Volume{
                Type: "{{user.volume_type}}",
                Size: {{user.volume_size}},
            },
        },
    }
    
    response, err := client.CreateInstance(request)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Instance ID: %s\n", response.Instance.Id)
}
```

#### Post-execution Validation

1. Read `{{output.instance_id}}` from the response path `instance.id`.
2. Poll **DescribeInstance** until terminal state:
   - Success: status = `ACTIVE`
   - Failure: status = `FAILED`
3. Default polling: interval 30s, max wait 1800s (30 minutes).
4. On success, report `{{output.instance_id}}`, private IP, and connection endpoint.
5. On terminal failure, go to **Failure Recovery**.

#### Failure Recovery

| Error | Max retries | Backoff | Agent Action | UX Feedback |
|-------|-------------|---------|--------------|-------------|
| `InvalidParameter` | 0–1 | — | Fix args from OpenAPI | `[ERROR] InvalidParameter: Check parameters against OpenAPI docs.` |
| `QuotaExceeded` | 0 | — | HALT | `[ERROR] Quota exceeded. Request quota increase or delete unused resources.` |
| `InsufficientBalance` | 0 | — | HALT | `[ERROR] Insufficient balance. Recharge your Huawei Cloud account.` |
| `ResourceAlreadyExists` | 0 | — | Ask reuse vs new name | `[ERROR] Resource already exists. Use different name or reuse existing.` |
| `VpcNotFound` | 0 | — | HALT | `[ERROR] VPC not found. Create VPC first using huaweicloud-vpc-ops.` |
| `SubnetNotFound` | 0 | — | HALT | `[ERROR] Subnet not found. Create subnet first using huaweicloud-vpc-ops.` |
| `SecurityGroupNotFound` | 0 | — | HALT | `[ERROR] Security group not found. Create SG first using huaweicloud-vpc-ops.` |
| `FlavorNotFound` | 0 | — | HALT | `[ERROR] Flavor not found. List available flavors first.` |
| Throttling / 429 | 3 | exponential | Back off; respect Retry-After | `⚠️ Rate limited. Retrying in {backoff}s...` |
| `InternalError` / 5xx | 3 | 2s, 4s, 8s | Retry; then HALT | `[ERROR] Server-side error. Retry or escalate with RequestId.` |

### Operation: Describe DB Instance

#### Execution

```bash
# CLI — describe specific instance
hcloud rds show \
  --region "{{user.region}}" \
  --instance-id "{{user.instance_id}}"

# List all instances
hcloud rds list --region "{{user.region}}"
```

### Operation: Delete DB Instance

#### Pre-flight (Safety Gate)

- **MUST** obtain explicit confirmation: irreversible delete of `{{user.instance_name}}` (`{{user.instance_id}}`).
- **MUST NOT** proceed without clear user assent.
- **MUST** remind user: "This will permanently delete the database instance and all associated data. Ensure you have a backup."
- **MUST** warn about cascading effects: automated backups will be deleted, read replicas will be affected.

#### Execution

```bash
# Delete with backup retention option
hcloud rds delete \
  --region "{{user.region}}" \
  --instance-id "{{user.instance_id}}" \
  --keep-backup-days 7
```

#### Post-execution Validation

Poll describe/get until **404** or **NotFound** status — per API semantics — within **max wait 300s**.

### Operation: Create Manual Backup

#### Execution

```bash
hcloud rds create-manual-backup \
  --region "{{user.region}}" \
  --instance-id "{{user.instance_id}}" \
  --name "manual-backup-$(date +%Y%m%d-%H%M%S)"
```

### Operation: Restore from Backup

#### Pre-flight (Safety Gate)
- **MUST** warn user: restore overwrites current data; suggest pre-restore backup
- **MUST** confirm: target instance, backup source, expected data loss window
- **MUST** verify: backup status is `COMPLETED` before restore

#### Execution

```bash
# Restore to existing instance (overwrites data)
hcloud rds restore \
  --region "{{user.region}}" \
  --instance-id "{{user.instance_id}}" \
  --backup-id "{{user.backup_id}}"

# Restore to new instance (safer)
hcloud rds restore-to-new \
  --region "{{user.region}}" \
  --backup-id "{{user.backup_id}}" \
  --name "{{user.new_instance_name}}"
```

### Operation: Modify Instance Specifications

#### Execution

```bash
# Resize flavor
hcloud rds resize \
  --region "{{user.region}}" \
  --instance-id "{{user.instance_id}}" \
  --flavor-ref "{{user.new_flavor_ref}}"

# Scale storage
hcloud rds expand-volume \
  --region "{{user.region}}" \
  --instance-id "{{user.instance_id}}" \
  --size "{{user.new_size_gb}}"
```

### Operation: Create Database (Internal)

#### Execution — JIT Go SDK

```go
request := &rds_model.CreateDatabaseRequest{
    InstanceId: "{{user.instance_id}}",
    Body: &rds_model.CreateDatabaseRequestBody{
        Name: "{{user.database_name}}",
        CharacterSet: "utf8mb4",
    },
}
response, err := client.CreateDatabase(request)
```

### Operation: Create DB User

#### Execution — JIT Go SDK

```go
request := &rds_model.CreateDbUserRequest{
    InstanceId: "{{user.instance_id}}",
    Body: &rds_model.CreateDbUserRequestBody{
        Name: "{{user.username}}",
        Password: "{{user.password}}", // Masked in logs
        Databases: []rds_model.DatabaseForCreation{
            {Name: "{{user.database_name}}", Readonly: false},
        },
    },
}
```

### Operation: Modify Parameters

#### Execution

```bash
# Update parameter group
hcloud rds modify-parameter \
  --region "{{user.region}}" \
  --instance-id "{{user.instance_id}}" \
  --parameter-name "max_connections" \
  --parameter-value "500"
```

## Prerequisites

> Full installation scripts (KooCLI + Go runtime + Credentials): see [references/common-prerequisites.md](../references/common-prerequisites.md)

## Quality Gate (GCL)

This skill is **GCL-required** (per `AGENTS.md` §8). Every RDS mutating operation — instance
create / delete / resize / restore, database / user create, parameter change, backup create /
delete — runs through the **Generator-Critic-Loop** before its result is returned. Read-only
`describe*` / `list*` are GCL-**exempt**.

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
| 1 | Correctness | ≥ 0.5 (1.0 for `delete-instance` / DDL / restore) | `ShowInstanceDetail` post-state |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | S1–S15 in rubric §2 |
| 3 | Idempotency | ≥ 0.5 | Pre-check before create |
| 4 | Traceability | ≥ 0.5 | `password` MUST be `<masked>`; never in CLI args |
| 5 | Spec Compliance | ≥ 0.5 | Engine version / flavor regex / storage range / parameter value / name regex |

### Per-Operation Safety Anchors (binding)

- **S1** — `delete-instance` without explicit user confirmation quoting the instance ID
- **S2** — `delete-instance` while latest automated backup is missing / failed, no manual
- **S3** — `delete-instance` for prePaid instance with > 7 days remaining, no refund-warning
- **S4 / S5** — `restore-from-backup` to different instance (cross-instance blast) or to same ACTIVE instance
- **S6** — `resize-instance` DOWN (smaller flavor / less storage) without maintenance window
- **S7** — `create-database` name with SQL injection chars
- **S8 / S14** — `create-user` / `reset-password` with password in CLI args or in trace
- **S9** — `update-parameter` weakening durability (`innodb_flush_log_at_trx_commit=2`, `sync_binlog=0`) on prod-tagged instance
- **S10** — `update-parameter` with `max_connections > 100000` without confirmation
- **S11** — `create-account` with `ALL PRIVILEGES + GRANT + *.*` to non-admin
- **S12** — `delete-database` for system DB (`mysql`, `information_schema`, `performance_schema`, `sys`, `postgres`, `template0/1`)
- **S13** — `delete-manual-backup` while `status != COMPLETED` or it's the only valid backup
- **S15** — `create-instance` with `region` / `project_id` not in env contract

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

## Reference Directory

- [Core Concepts](references/core-concepts.md) — RDS architecture, limits, quotas
- [API & SDK Usage](references/api-sdk-usage.md) — API mapping, Go SDK patterns
- [CLI Usage](references/cli-usage.md) — hcloud command reference
- [Troubleshooting Guide](references/troubleshooting.md) — Error codes, diagnostics
- [Monitoring & Alerts](references/monitoring.md) — CES metrics, AIOps patterns
- [Integration](references/integration.md) — Cross-skill delegation, SDK setup
- [Well-Architected Assessment](references/well-architected-assessment.md) — Five pillars + FinOps + SecOps + AIOps
- [GCL Rubric](references/rubric.md) — Adversarial quality gate (v1, 5-dim, S1–S15 RDS-specific Safety rules)
- [GCL Prompt Templates](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons

## FinOps Integration (财务运营)

### Cost Visibility

| Billing Model | Best Scenario | Savings |
|--------------|---------------|---------|
| 按需计费 (Pay-per-use) | 开发测试、短期负载 | N/A |
| 包年包月 (Subscription) | 生产环境、稳定负载 | 最高85% vs 按需 |
| 竞价实例 (Spot) | 不适用RDS | N/A |

### Cost Optimization Patterns

| Utilization Pattern | Recommendation | Expected Savings |
|--------------------|-----------------|------------------|
| CPU < 20% for 7 days | Right-size to smaller flavor | 30-60% |
| Storage > 80% for 3 days | Scale storage or enable auto-scale | Prevent outage |
| Idle instances (no connections) | Consider deletion or downgrade | 100% of compute |

### Cost Tagging Strategy

```yaml
# Recommended tags for cost tracking
tags:
  - key: CostCenter
    value: "{{user.cost_center}}"
  - key: Environment
    value: "{{user.environment}}"  # prod/staging/dev
  - key: Owner
    value: "{{user.owner}}"
  - key: CreatedDate
    value: "{{timestamp}}"
  - key: ExpectedDecommission
    value: "{{user.decommission_date}}"
```

## SecOps Integration (安全运营)

### IAM Minimum Permissions

| Operation | IAM Action | Resource Scope |
|-----------|-----------|----------------|
| CreateInstance | rds:instance:create | acs:rds:*:*:instance/* |
| DeleteInstance | rds:instance:delete | acs:rds:*:*:instance/${instance_id} |
| DescribeInstance | rds:instance:get | acs:rds:*:*:instance/* |
| ListInstances | rds:instance:list | acs:rds:*:*:instance/* |
| CreateBackup | rds:backup:create | acs:rds:*:*:instance/${instance_id} |
| RestoreInstance | rds:instance:restore | acs:rds:*:*:instance/${instance_id} |

### Network Security

- **VPC Isolation**: RDS instances must be created in private subnets
- **Security Groups**: Restrict inbound to application servers only
- **TDE Encryption**: Enable Transparent Data Encryption for sensitive data
- **SSL/TLS**: Enforce encrypted connections for all database traffic

### Credential Security

- Database passwords stored in **KMS** (Key Management Service)
- **Never** log passwords or connection strings
- Use **IAM Agency** for cross-service access
- Rotate AK/SK every **90 days**

## AIOps Integration (智能运营)

### Multi-Metric Correlation Patterns

| Pattern | Metrics | Detection Logic | Severity |
|---------|---------|-----------------|----------|
| CPU-Memory Dual High | rds001_cpu_usage, rds002_mem_usage | cpu>80% AND mem>85% | Critical |
| Connection Exhaustion | rds003_connections_usage, rds001_cpu_usage | connections>90% | Critical |
| Storage Pressure | rds004_disk_usage, rds045_iops | disk>85% OR iops>threshold | Warning |
| Slow Query Spike | rds043_slow_queries | delta(10min)>50% | Warning |

### Cross-Skill Delegation Matrix

| Alert Type | Primary Skill | Secondary Skill | Notes |
|-----------|---------------|-----------------|-------|
| High CPU | huaweicloud-rds-ops | huaweicloud-ces-ops | Check slow queries |
| Connection Issues | huaweicloud-rds-ops | huaweicloud-vpc-ops | Check SG rules |
| Storage Full | huaweicloud-rds-ops | — | Scale storage |
| Backup Failures | huaweicloud-rds-ops | huaweicloud-obs-ops | Check OBS quota |

### Proactive Inspection Workflow

1. **Discovery**: List all RDS instances in region
2. **Metric Collection**: Batch collect CPU, memory, connections, storage
3. **Anomaly Detection**: Check patterns against thresholds
4. **Cross-Skill Diagnosis**: Delegate abnormal instances
5. **Report Generation**: Generate inspection report with findings

## Well-Architected + Three-Pillar Assessment

This skill's operations are evaluated against Huawei Cloud's Well-Architected Framework (卓越架构) five pillars plus FinOps, SecOps, and AIOps integration:
- [Security Assessment](references/well-architected-assessment.md#21-安全支柱-security)
- [Stability Assessment](references/well-architected-assessment.md#22-稳定支柱-stability)
- [Cost Assessment](references/well-architected-assessment.md#23-成本支柱-cost)
- [Efficiency Assessment](references/well-architected-assessment.md#24-效率支柱-efficiency)
- [Performance Assessment](references/well-architected-assessment.md#25-性能支柱-performance)
- [FinOps Integration](references/well-architected-assessment.md#3-finops-财务运营)
- [SecOps Integration](references/well-architected-assessment.md#4-secops-安全运营)
- [AIOps Integration](references/advanced/aiops-best-practices.md)

> 任务完成后按根 AGENTS.md 的「复利资产沉淀机制 (CADL)」复盘并沉淀可复用资产。
