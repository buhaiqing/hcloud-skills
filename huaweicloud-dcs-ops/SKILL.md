---
name: huaweicloud-dcs-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei Cloud DCS (Distributed Cache Service / Redis) — instance lifecycle, backup, restore, resize, password reset, whitelist management, and monitoring. User mentions DCS, Redis, distributed cache, 分布式缓存, or describes cache-related scenarios (e.g., connection drops, high latency, OOM, eviction, cache avalanche, Redis AUTH issues, key expiration) even without naming the product directly. Not for billing, IAM, or related products that have their own ops skills.
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
  api_profile: "DCS v2 — https://support.huaweicloud.com/api-dcs/"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    DCS operations available via `hcloud dcs` commands: create-instance, show-instance, list-instances, delete-instance, reset-password, create-backup, list-backups, show-whitelist, update-whitelist.
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
        change: "GCL Phase 2 rollout: added references/rubric.md (v1, 5-dim, S1–S15 DCS-specific Safety rules, including FLUSHALL/instance-delete/replication-pair guards) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
      - version: "1.0.0"
        date: "2026-05-20"
        change: "Initial skill release."
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud DCS (Distributed Cache Service) Operations Skill

## Overview

Huawei Cloud DCS provides fully-managed Redis (4.0/5.0/6.0) and Memcached instances with automatic scaling, backup, and high-availability. This skill is an **operational runbook** for agents: explicit scope, credential rules, pre-flight checks, **dual-path execution** (official CLI and JIT Go SDK fallback), response validation, and failure recovery.

### CLI applicability (repository policy)

- **`cli_applicability: dual-path`:** Official CLI supports DCS. In each execution flow below, document **both** the SDK step **and** the CLI step.

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions with precise triggers and delegation rules |
| 2 | **Structured I/O** | `{{env.*}}` / `{{user.*}}` / `{{output.*}}` placeholders with typed sources |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute (CLI + SDK) → Validate → Recover |
| 4 | **Complete Failure Strategies** | 12 DCS error codes with HALT vs retry per error type |
| 5 | **Absolute Single Responsibility** | One product (DCS), one primary resource (Instance); cross-product delegation documented |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **FinOps** | Billing model comparison (按需/包年包月/竞价), idle detection, right-sizing matrix | `references/well-architected-assessment.md` §3 |
| **SecOps** | IAM minimum permissions, VPC isolation, Redis AUTH/TLS, whitelist enforcement | `references/well-architected-assessment.md` §4 |
| **AIOps** | ≥ 4 anomaly patterns (OOM, connection storm, cache miss, latency), cross-skill diagnosis | `references/knowledge-base.md` |

### Well-Architected Framework Integration (卓越架构)

| Pillar | Skill Integration |
|--------|-------------------|
| **安全 (Security)** | IAM permissions, credential masking, Redis AUTH/TLS, KMS for backups |
| **稳定 (Stability)** | Backup/restore with RTO/RPO, master-standby auto-failover, multi-AZ |
| **成本 (Cost)** | Billing model comparison (up to 85% savings for 包年包月), idle instance detection |
| **效率 (Efficiency)** | Batch operations, CLI JSON output for jq pipeline, CI/CD integration |
| **性能 (Performance)** | Scaling triggers, hot/big key detection, latency thresholds, per-instance baselines |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud DCS" OR "Redis" OR "分布式缓存" OR "缓存服务" OR "DCS instance"
- Task involves CRUD or lifecycle operations on DCS instances (create, delete, resize, restart, stop, start)
- Task keywords: **Redis**, **Memcached**, **cache**, **DCS**, **connection**, **eviction**, **hit_rate**, **OOM**, **AUTH**, **whitelist**, **backup**, **restore**, **key**, **pipeline**
- User asks to deploy, configure, troubleshoot, or monitor DCS via API, SDK, CLI, or automation

### SHOULD NOT Use This Skill When
- Task is purely billing / cost analysis / 费用 / 预算 → delegate to: `huaweicloud-billing-ops`

- Task is IAM/permission model only → delegate to: `huaweicloud-iam-ops`
- Task is about ECS hosting the Redis application → delegate to: `huaweicloud-ecs-ops`
- Task is about network/VPC without DCS involvement → delegate to: `huaweicloud-vpc-ops`

### Delegation Rules

- DCS depends on VPC/Subnet/Security Group: verify or delegate to `huaweicloud-vpc-ops` before DCS create
- DCS monitoring/alerts: delegate to `huaweicloud-ces-ops` for alarm rule setup
- For FinOps questions: use this skill's cost section; cross-resource cost to billing skill
- For security audit: use this skill's sec section; account-level IAM to IAM skill

## Variable Convention (Agent-Readable)

| Placeholder | Meaning | Agent Action |
|-------------|---------|--------------|
| `{{env.HW_ACCESS_KEY_ID}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_REGION_ID}}` | From runtime environment | Use documented default |
| `{{env.HW_PROJECT_ID}}` | From runtime environment | Use documented default |
| `{{user.instance_name}}` | User-supplied instance name | Ask once; reuse |
| `{{user.engine_version}}` | Redis version (4.0/5.0/6.0) or Memcached | Ask once with defaults |
| `{{user.capacity_gb}}` | Instance memory capacity (GB) | Ask once; suggest spec sizes |
| `{{user.instance_id}}` | User-supplied DCS instance ID | Parse from prior output or ask |
| `{{output.instance_id}}` | From API/CLI response: instance_id | Parse per OpenAPI path |
| `{{output.status}}` | From API/CLI response: status | Track for polling |

> **`{{env.*}}` MUST NOT** be collected from the user. **`{{user.*}}`** MUST be collected interactively when missing.

> **Security Warning (Credential Masking — MANDATORY):** **NEVER** log, print, or expose `HW_SECRET_ACCESS_KEY` or any credential field value.

## API and Response Conventions

- **OpenAPI is canonical** for path, query, body fields, enums, and response shapes.
- **Errors:** Map to `error_code` / `error_msg` / `request_id` fields per spec.
- **Timestamps:** ISO 8601 with timezone.
- **Idempotency:** Instance names must be unique within a region; `InstanceAlreadyExists` behavior per API.

## Quick Start

### What This Skill Does
Deploy, configure, troubleshoot, and monitor Huawei Cloud DCS (Redis/Memcached) instances using `hcloud` CLI (primary) or JIT Go SDK (fallback).

### Prerequisites

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|-----------|-------------|------------|------------|
| Create Instance | Create a Redis/Memcached instance | Medium | Low |
| Describe Instance | View instance details and status | Low | None |
| List Instances | Enumerate all instances in region | Low | None |
| Resize/Migrate | Change instance spec or capacity | Medium | **Medium** — may cause brief failover |
| Delete Instance | Remove a DCS instance | Low | **High** — irreversible, data loss |
| Backup | Create manual backup (RDB) | Low | None |
| Restore | Restore from backup to new/existing instance | Medium | **Medium** — overwrites data |
| Reset Password | Change Redis AUTH password | Low | Medium — affects existing connections |
| Whitelist Management | Configure IP whitelist for access control | Low | Medium — may block legitimate clients |

## Execution Flows

### Operation: Create DCS Instance

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| CLI / deps | `hcloud --version` | Exit code 0 | Document CLI install from prerequisites |
| Credentials | Construct credential from env | Non-empty keys | HALT; user configures env |
| Region | Call `hcloud dcs list-instances --region` | Region is supported | Suggest valid region |
| VPC | Verify VPC exists: `hcloud vpc list --region` | VPC ID valid | HALT; create VPC via vpc-ops |
| Subnet | Verify subnet in VPC: `hcloud vce list-subnets` | Subnet ID in same VPC | HALT; create subnet via vpc-ops |
| Security Group | Verify SG exists and allows port 6379 | SG ID valid | HALT; create SG with Redis rule |
| Quota | Check DCS instance quota (via API) | Sufficient quota | HALT; user raises quota |

#### Execution — CLI (Primary Path)

```bash
hcloud dcs create-instance \
  --region "{{env.HW_REGION_ID}}" \
  --name "{{user.instance_name}}" \
  --engine "redis" \
  --engine-version "6.0" \
  --capacity 4 \
  --instance-mode "ha" \
  --vpc-id "{{user.vpc_id}}" \
  --subnet-id "{{user.subnet_id}}" \
  --security-group-id "{{user.sg_id}}" \
  --password "{{user.password}}" \
  --availability-zone "{{user.az}}"
```

#### Execution — JIT Go SDK (Fallback Path)

```go
package main

import (
    "fmt"
    "os"
    dcs "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dcs/v2"
    dcs_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/dcs/v2/model"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
)

func main() {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    region := os.Getenv("HW_REGION_ID")

    cfg := config.DefaultHttpConfig()
    client := dcs.NewDcsClient(
        dcs.DcsClientBuilder().
            WithEndpoint(fmt.Sprintf("dcs.%s.myhuaweicloud.com", region)).
            WithCredential(basic.NewCredentialsBuilder().
                WithAk(ak).WithSk(sk).Build()).
            WithHttpConfig(cfg).Build())

    request := &dcs_model.CreateInstanceRequest{}
    request.Body = &dcs_model.CreateInstanceRequestBody{
        Name:            "{{user.instance_name}}",
        Engine:          "redis",
        EngineVersion:   "6.0",
        Capacity:        1, // 4GB = 4 * 1024MB, but API uses GB multiplier
        InstanceMode:    "ha",
        VpcId:           "{{user.vpc_id}}",
        SubnetId:        "{{user.subnet_id}}",
        SecurityGroupId: "{{user.sg_id}}",
        Password:        "{{user.password}}",
        AvailabilityZones: []string{"{{user.az}}"},
    }

    response, err := client.CreateInstance(request)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Instance ID: %s\n", *response.InstanceId)
}
```

#### Post-execution Validation

1. Parse `{{output.instance_id}}` from CLI/SDK response.
2. Poll `hcloud dcs show-instance --instance-id {{output.instance_id}}` every 5s, up to 300s.
3. Terminal success: `status = "RUNNING"`.
4. Terminal failure: `status = "ERROR"` → go to Failure Recovery.

#### Failure Recovery

| Error | Max retries | Backoff | Agent Action | UX Feedback |
|-------|-------------|---------|--------------|-------------|
| `InvalidParameter` | 0–1 | — | Fix args per OpenAPI | `[ERROR] InvalidParameter: Check parameters (engine, capacity, VPC) against DCS docs.` |
| `QuotaExceeded` | 0 | — | HALT | `[ERROR] Quota exceeded. Request quota increase or delete unused instances.` |
| `InsufficientBalance` | 0 | — | HALT | `[ERROR] Insufficient balance. Recharge your Huawei Cloud account.` |
| `InstanceAlreadyExists` | 0 | — | Ask reuse vs new name | `[ERROR] Instance name already exists. Use different name or reuse existing.` |
| `SecurityGroupNotFound` | 0 | — | HALT | `[ERROR] Security group not found. Create SG with Redis port 6379 allowed.` |
| `VPCNotExists` | 0 | — | HALT | `[ERROR] VPC not found. Verify VPC ID exists in region.` |
| `PasswordInvalid` | 0 | — | Fix password | `[ERROR] Password must be 8–64 chars, include letters, digits, special chars.` |
| `EngineNotSupported` | 0 | — | Use supported version | `[ERROR] Unsupported engine version. Use Redis 4.0/5.0/6.0 or Memcached.` |
| Throttling / 429 | 3 | exponential | Back off; respect Retry-After | `⚠️ Rate limited. Retrying in {backoff}s...` |
| `InternalError` / 5xx | 3 | 2s, 4s, 8s | Retry; then HALT | `[ERROR] Server-side error. Retry or escalate with RequestId.` |
| `InvalidInstanceStatus` | 0 | — | Wait for running state | `[ERROR] Instance not in RUNNING state for this operation.` |
| `InstanceNotFound` | 0 | — | Verify instance_id | `[ERROR] Instance not found. Verify the instance_id is correct.` |

---

### Operation: Describe DCS Instance

#### Execution — CLI

```bash
hcloud dcs show-instance --region "{{env.HW_REGION_ID}}" --instance-id "{{user.instance_id}}"
```

#### JIT Go SDK

```go
request := &dcs_model.ShowInstanceRequest{InstanceId: "{{user.instance_id}}"}
response, err := client.ShowInstance(request)
```

#### Validation

- Verify status field present: `RUNNING` / `ERROR` / `CREATING` / `RESTARTING`
- Key fields to capture: `name`, `engine`, `engine_version`, `capacity`, `ip`, `port`, `status`, `vpc_id`

---

### Operation: List DCS Instances

#### Execution — CLI

```bash
hcloud dcs list-instances --region "{{env.HW_REGION_ID}}" --limit 50
```

#### Validation

- Parse list of instances with name, id, status, engine, capacity
- Handle pagination with `--offset` / `--limit` if `include_instance_count` > returned count

---

### Operation: Resize DCS Instance

#### Pre-flight

- Instance must be in `RUNNING` state
- Notify user of possible brief failover during resize
- Suggest backup before resize

#### Execution — CLI

```bash
hcloud dcs resize-instance \
  --instance-id "{{user.instance_id}}" \
  --new-spec-code "{{user.new_spec_code}}" \
  --new-capacity {{user.new_capacity_gb}}
```

#### Post-execution Validation

- Poll status until `RUNNING` (may take several minutes for data migration)
- Verify no data loss: connect and run `INFO memory` to check new capacity

---

### Operation: Delete DCS Instance

#### Pre-flight (Safety Gate)

- **MUST** obtain explicit confirmation: "This will delete instance `{{user.instance_name}}` ({{user.instance_id}}) irrevocably. All data will be lost. Proceed?"
- **MUST NOT** proceed without clear assent.
- **MUST** warn: create backup before delete if data retention needed.

#### Execution

```bash
hcloud dcs delete-instance --instance-id "{{user.instance_id}}" --delete-backup=false
```

#### Validation

- Poll `show-instance` until 404 / `InstanceNotFound` within 300s
- If `--delete-backup=false`, verify backup persists in backup list

---

### Operation: Backup DCS Instance

#### Execution — CLI

```bash
hcloud dcs create-backup \
  --instance-id "{{user.instance_id}}" \
  --backup-name "manual-$(date +%Y%m%d-%H%M%S)"
```

#### Validation

- CreateBackup API returns `backup_id`
- Poll `list-backups --instance-id` until backup `status = "SUCCESS"`
- Verify backup file size > 0

---

### Operation: Restore from Backup

#### Pre-flight

- **MUST** warn: restoring overwrites current data; suggest pre-restore backup
- Confirm: target instance ID, backup source

#### Execution

```bash
hcloud dcs restore-instance \
  --instance-id "{{user.instance_id}}" \
  --backup-id "{{user.backup_id}}"
```

#### Validation

- Poll instance status until `RUNNING`
- Verify data integrity: check key count via `INFO keyspace`

---

### Operation: Reset Password

#### Execution — CLI

```bash
hcloud dcs reset-password \
  --instance-id "{{user.instance_id}}" \
  --new-password "{{user.new_password}}"
```

#### Validation

- Instance enters `PASSWORD_RESET` state → must wait for `RUNNING`
- All existing connections will be disconnected; reconnect requires new AUTH

---

### Operation: IP Whitelist Management

#### Show Whitelist

```bash
hcloud dcs show-whitelist --instance-id "{{user.instance_id}}"
```

#### Update Whitelist

```bash
hcloud dcs update-whitelist \
  --instance-id "{{user.instance_id}}" \
  --whitelist-enable true \
  --whitelist "192.168.1.0/24,10.0.0.0/8"
```

#### Validation

- Verify whitelist entry count matches expected
- Test connectivity from whitelisted IP

---

## Prerequisites

> Full installation scripts (KooCLI + Go runtime + Credentials): see [references/common-prerequisites.md](../references/common-prerequisites.md)

## Quality Gate (GCL)

This skill is **GCL-required** (per `AGENTS.md` §8). Every DCS mutating operation — instance
create / delete / resize, backup create / restore, password reset, whitelist update, and
`FLUSHALL`-style destructive Redis commands — runs through the **Generator-Critic-Loop** before
its result is returned. Read-only `describe*` / `list*` are GCL-**exempt**.

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
| 1 | Correctness | ≥ 0.5 (1.0 for `delete-instance` / `restore` / `flushall`) | `ShowInstance` / `ShowBackup` post-state |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | S1–S15 in rubric §2 |
| 3 | Idempotency | ≥ 0.5 | Pre-check before create |
| 4 | Traceability | ≥ 0.5 | `password` MUST be `<masked>` |
| 5 | Spec Compliance | ≥ 0.5 | Engine version / capacity / whitelist CIDR |

### Per-Operation Safety Anchors (binding)

- **S1** — `delete-instance` without explicit user confirmation quoting the instance ID
- **S2** — `delete-instance` while latest backup missing/failed, no manual backup
- **S3** — `delete-instance` for prePaid instance with > 7 days remaining
- **S4** / **S5** — `restore-from-backup` to source (cluster) or to a different instance
- **S6** — `reset-password` with password in CLI args or in trace
- **S7** — `update-whitelist` removing ALL entries (lock-out)
- **S8** — `update-whitelist` adding `0.0.0.0/0` to prod instance without two-step
- **S9** — `resize-instance` DOWN (smaller memory) without maintenance window
- **S12** — `delete-instance` for Redis source of replication pair (replica orphaned)
- **S13** — `run-command` with `FLUSHALL` / `FLUSHDB` / `DEBUG SLEEP` / `SHUTDOWN NOSAVE` on prod instance

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

### Prompt Backbone

Use `huaweicloud-skill-generator/references/gcl-prompt-backbone.md` as the shared source
for Generator/Critic/Orchestrator wording. Runtime Roles (Generator / Critic /
Orchestrator) and their isolation constraints: see `docs/gcl-spec.md` §Runtime Roles
and root `AGENTS.md` §5. Default rubric thresholds (correctness ≥0.5, safety =1.0,
…): see `docs/gcl-spec.md` §Thresholds. Trace persistence + masking rules: see
`docs/gcl-spec.md` §Trace and root `AGENTS.md` (credential masking mandatory).
This skill's `references/prompt-templates.md` keeps DCS-specific overrides and must
not introduce bare `{...}` placeholders.

### See also

- [`references/rubric.md`](references/rubric.md) — full rubric, S1–S15 rules, per-op thresholds
- [`references/prompt-templates.md`](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons
- Repository root [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8 — GCL specification

## Reference Directory

- [Core Concepts](references/core-concepts.md)
- [API & SDK Usage](references/api-sdk-usage.md)
- [CLI Usage](references/cli-usage.md)
- [Troubleshooting Guide](references/troubleshooting.md)
- [Monitoring & Alerts](references/monitoring.md)
- [Integration & Delegation](references/integration.md)
- [Knowledge Base](references/knowledge-base.md)
- [Well-Architected Assessment](references/well-architected-assessment.md)
- [GCL Rubric](references/rubric.md) — Adversarial quality gate (v1, 5-dim, S1–S15 DCS-specific Safety rules)
- [GCL Prompt Templates](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons

## Well-Architected + Three-Pillar Assessment

This skill's operations are evaluated against:
- [Security](references/well-architected-assessment.md#security): IAM, credential masking, Redis AUTH, TLS
- [Stability](references/well-architected-assessment.md#stability): Backup/recovery RTO/RPO, multi-AZ, DR runbook
- [Cost](references/well-architected-assessment.md#cost): Billing comparison, idle detection, right-sizing
- [Efficiency](references/well-architected-assessment.md#efficiency): Batch operations, CI/CD, automation
- [Performance](references/well-architected-assessment.md#performance): Scaling triggers, hot/big key detection
- [FinOps Integration](references/well-architected-assessment.md#finops): Cost visibility, optimization matrix
- [SecOps Integration](references/well-architected-assessment.md#secops): IAM, VPC, encryption, threat detection
- [AIOps Integration](references/knowledge-base.md): Anomaly patterns, cross-skill delegation, knowledge base
