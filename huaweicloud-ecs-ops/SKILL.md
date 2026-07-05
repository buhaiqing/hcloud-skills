---
name: huaweicloud-ecs-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei
  Cloud ECS (Elastic Cloud Server) — instance lifecycle, disks, security groups,
  CloudShell remote execution, and diagnostics. User mentions ECS, 弹性云服务器,
  云主机, or describes scenarios (instance unreachable, performance degradation,
  disk full, security group misconfiguration) even without naming ECS directly.
  Not for billing, IAM account management, VPC networking configuration, or ELB
  load balancer setup that have their own ops skills.
license: MIT
compatibility: >-
  KooCLI (official binary, latest **4.1.6**), Go 1.21+ runtime for JIT SDK fallback
  via huaweicloud-sdk-go-v3, valid AK/SK credentials, network access to
  Huawei Cloud endpoints.
metadata:
  author: huaweicloud
  version: "1.1.0"
  last_updated: "2026-06-04"
  cli_applicability: "dual-path"
  cli_version: "4.1.6"
  sdk_version: "v0.1.191"
  go_version_minimum: "1.21"
  go_version_jit: "1.25+"
  api_profile: "https://support.huaweicloud.com/api-ecs/ecs_01_0043.html"
  cli_support_evidence: >-
    Huawei Cloud ECS is supported via `hcloud ecs` CLI commands and
    huaweicloud-sdk-go-v3/services/ecs/v2 Go SDK package.
  gcl:
    enabled: true
    required: true
    rubric_version: "v1"
    max_iter: 2
    rubric_ref: "references/rubric.md"
    prompts_ref: "references/prompt-templates.md"
    trace_dir: "./audit-results/"
    pilot: true
    changelog:
      - version: "1.1.0"
        date: "2026-06-04"
        change: "GCL pilot rollout: added references/rubric.md (v1, 5-dim, S1–S10 safety rules) and references/prompt-templates.md (Generator + Critic + Orchestrator skeletons). SKILL.md gains 'Quality Gate (GCL)' chapter."
      - version: "1.0.0"
        date: "2026-05-20"
        change: "Initial skill release."
    CloudShell remote execution via Cloud-Cell Agent (云主机助手) API and OpenStack remote-exec extension.
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
---

> This template follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud ECS Operations Skill

## Overview

Huawei Cloud ECS (Elastic Cloud Server / 弹性云服务器) provides on-demand scalable computing resources. This skill is an **operational runbook** for agents: ECS instance lifecycle management, disk/volume attachment, security group configuration, CloudShell remote execution and file transfer, Cloud-Cell Agent (云主机助手) installation verification, response validation, and failure recovery. **Dual-path execution**: official **SDK/API** (`huaweicloud-sdk-go-v3/services/ecs/v2`) and **`hcloud` CLI**.

> **UX Compliance:** This skill follows the User Experience Specification. All operations include onboarding guidance, minimal prompts, smart defaults, clear feedback, and user-friendly error handling.

### CLI Applicability (repository policy)

- **`cli_applicability: dual-path`** — Official `hcloud ecs` CLI supports most ECS operations. **MUST** document both SDK and CLI paths for every operation.

### Well-Architected + Three-Pillar Integration

This skill integrates Huawei Cloud Well-Architected five pillars plus FinOps, SecOps, and AIOps:
- [Security Assessment](references/well-architected-assessment.md#21)
- [FinOps Cost Optimization](references/well-architected-assessment.md#3)
- [SecOps Security Operations](references/well-architected-assessment.md#4)
- [AIOps Integration](references/advanced/aiops-best-practices.md)

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT triggers with precise keywords, delegation rules to VPC/CES skills |
| 2 | **Structured I/O** | `{{env.*}}` for credentials, `{{user.*}}` for instance specs, `{{output.*}}` for API responses |
| 3 | **Explicit Steps** | Every operation: Pre-flight → Execute → Validate → Recover with numbered imperative steps |
| 4 | **Failure Strategies** | 15+ ECS-specific error codes with HALT vs retry distinction |
| 5 | **Single Responsibility** | ECS instances/disks/security groups only; delegates VPC to `huaweicloud-vpc-ops`, monitoring to `huaweicloud-ces-ops` |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud ECS", "弹性云服务器", "云主机", "ECS instance", "云服务器"
- Task involves ECS lifecycle: create, start, stop, restart, resize, delete, list, describe
- Task involves disk management: create EVS, attach, detach, resize, snapshot
- Task involves security group: create, modify rules, bind to instance
- Task involves CloudShell remote execution on ECS: run commands, upload/download files
- Task involves Cloud-Cell Agent (云主机助手): verify installation, collect host diagnostics
- Task keywords: `instance`, `server`, `flavor`, `image`, `disk`, `volume`, `security group`, `CloudShell`, `云助手`

### SHOULD NOT Use This Skill When

- Task is purely billing / cost analysis → delegate to: `huaweicloud-billing-ops`
- Task is IAM permission model only → delegate to: `huaweicloud-iam-ops` (when present)
- Task is VPC/subnet creation → delegate to: `huaweicloud-vpc-ops` (when present)
- Task is ELB listener/backend setup → delegate to: `huaweicloud-elb-ops` (when present)
- Task is CES alarm/dashboard setup → delegate to: `huaweicloud-ces-ops` (when present)

### Delegation Rules

- ECS creation requires VPC/subnet exist → verify with VPC skill first
- Cloud-Cell Agent requires security group allows outbound HTTPS → check with security group skill
- Performance diagnosis for running instances → delegate to CES skill for metrics
- Cost optimization questions → use this skill's FinOps section, delegate cross-resource billing to billing skill

## Variable Convention

| Placeholder | Meaning | Agent Action |
|-------------|---------|--------------|
| `{{env.HW_ACCESS_KEY_ID}}` | From runtime environment | NEVER ask user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | From runtime environment | NEVER ask user; fail if unset |
| `{{env.HW_REGION_ID}}` | Default region (e.g., `cn-north-4`) | Use if skill allows |
| `{{env.HW_PROJECT_ID}}` | Project ID | Use for scoped operations |
| `{{user.instance_name}}` | User-supplied instance name | Ask once; reuse |
| `{{user.flavor}}` | Instance flavor (e.g., `s3.medium.4`) | Suggest defaults |
| `{{user.image_id}}` | Image ID | Ask with `list-images` if unknown |
| `{{user.vpc_id}}` | VPC ID | Use VPC skill to list |
| `{{user.security_group_id}}` | Security group ID | Default to VPC security group |
| `{{output.instance_id}}` | From create response | Parse per OpenAPI path |

> **`{{env.*}}` MUST NOT** be collected from user. **Credential masking is MANDATORY** — never echo `HW_SECRET_ACCESS_KEY`.

## Quick Start

### What This Skill Does
Manage Huawei Cloud ECS instances: create, start/stop, resize, delete, monitor, and execute remote commands via CloudShell or Cloud-Cell Agent.

### Prerequisites

## API and Response Conventions

- **OpenAPI canonical**: `https://support.huaweicloud.com/api-ecs/`
- **Async pattern**: Most ECS operations return `job_id` — poll via `ShowJob` API until `SUCCESS`/`FAIL`
- **Status transitions**: `BUILD` → `ACTIVE` → `STOPPED` → `DELETED`
- **Pagination**: Use `limit` + `offset`, max 1000 per request
- **Idempotency**: Use `client_token` to prevent duplicate creation

## Expected State Transitions

| Operation | Initial State | Target State | Poll API | Max Wait |
|-----------|--------------|--------------|----------|----------|
| Create | — | `ACTIVE` | `ShowJob(job_id)` | 600s |
| Start | `SHUTOFF`/`STOPPED` | `ACTIVE` | `ListServersDetail` | 300s |
| Stop | `ACTIVE` | `SHUTOFF` | `ListServersDetail` | 300s |
| Resize | `ACTIVE` | `ACTIVE` | `ShowJob(job_id)` | 600s |
| Delete | any | absent | `ShowServer(job_id)` 404 | 300s |
| Attach Volume | `ACTIVE` | `ACTIVE` | `ShowServer` | 120s |
| Detach Volume | `ACTIVE` | `ACTIVE` | `ShowServer` | 120s |

## Execution Flows

### Operation: Create ECS Instance

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| VPC/subnet exists | `hcloud vpc list-vpcs` | VPC and subnet IDs valid | Create via VPC skill first |
| Flavor available | `hcloud ecs list-flavors` | Flavor exists in region | Suggest alternative flavor |
| Image available | `hcloud ims list-images` | Image ID valid | List available images |
| Security group exists | `hcloud vpc list-security-groups` | SG ID valid | Create default SG |
| Quota sufficient | `ecs API: ShowQuotas` | Quota > 0 | HALT — request quota increase |
| Credentials valid | Describe server attempt | Non-401 response | HALT — user configures credentials |

#### Execution — CLI (Primary Path)

```bash
# Create a single ECS instance
hcloud ecs create-server \
  --region {{user.region}} \
  --name "{{user.instance_name}}" \
  --flavor-id "{{user.flavor}}" \
  --image-id "{{user.image_id}}" \
  --vpc-id "{{user.vpc_id}}" \
  --nics.[0].subnet-id "{{user.subnet_id}}" \
  --security-groups.[0].id "{{user.security_group_id}}" \
  --root-volume.volumetype SSD \
  --availability-zone "{{user.az}}"
```

#### Execution — JIT Go SDK (Fallback Path)

> Full Go SDK implementation: see [`references/api-sdk-usage.md`](references/api-sdk-usage.md#jit-go-sdk-fallback)

#### Post-execution Validation

1. Extract `{{output.job_id}}` from create response.
2. Poll job status via `ShowJob(job_id)` until `SUCCESS` (max 600s, 5s interval).
3. After job completes, verify instance via `ShowServerDetail(server_id)`.
4. Confirm status is `ACTIVE`.
5. Report `{{output.instance_id}}`, IP addresses, and flavor to user.

#### Failure Recovery

| Error | Max Retries | Agent Action | UX Feedback |
|-------|-------------|--------------|-------------|
| `Ecs.0801` InsufficientResource | 0 | HALT | `[ERROR] Resource quota insufficient. Resize flavor or request quota increase.` |
| `Ecs.0804` ImageNotFound | 0 | HALT | `[ERROR] Image not found. Verify image ID with `hcloud ims list-images`.` |
| `Ecs.0805` FlavorNotFound | 0 | HALT | `[ERROR] Flavor not found. Check available flavors with `hcloud ecs list-flavors`.` |
| `Ecs.0820` SecurityGroupNotFound | 0 | HALT | `[ERROR] Security group not found. Verify or create via VPC skill.` |
| `Ecs.4600` VPC/SubnetNotFound | 0 | HALT | `[ERROR] VPC/subnet not found. Create via `huaweicloud-vpc-ops` first.` |
| `Ecs.4601` IPAddressConflict | 0 | HALT | `[ERROR] IP address conflict. Use different subnet or let system auto-allocate.` |
| `Ecs.4603` InstanceNameDuplicate | 0 | HALT | `[ERROR] Instance name already exists. Choose unique name.` |
| `Ecs.4610` InsufficientQuota | 0 | HALT | `[ERROR] ECS quota exceeded. Submit quota increase request.` |
| `Ecs.4615` AZResourceInsufficient | 0 | HALT | `[ERROR] AZ has insufficient resources. Try different AZ or flavor.` |
| `Ecs.4620` InvalidParameter | 0 | HALT | `[ERROR] Invalid parameter. Verify all parameters against API docs.` |
| Throttling 429 | 3 | Exponential backoff | `⚠️ Rate limited. Retrying in {backoff}s...` |
| InternalError 500 | 3 | Backoff 2s→4s→8s | `[ERROR] Server error. Retry or escalate with RequestId.` |

### Operation: Start/Stop ECS Instance

#### Pre-flight (Safety Gate for Stop)

- Confirm instance status is `ACTIVE` before stop
- Confirm `SHUTOFF` before start
- For stop: warn about service interruption

#### Execution

```bash
# Start instance
hcloud ecs start-server --region {{user.region}} --server-id {{user.instance_id}}

# Stop instance (soft stop)
hcloud ecs stop-server --region {{user.region}} --server-id {{user.instance_id}} --type SOFT

# Force stop
hcloud ecs stop-server --region {{user.region}} --server-id {{user.instance_id}} --type OS-STOP
```

#### Validation

Poll `ListServersDetail(server_id)` until state matches target (`ACTIVE` or `SHUTOFF`). Max 300s.

### Operation: Resize (Flavor Change)

#### Pre-flight

- Instance must be `ACTIVE` or `SHUTOFF`
- Check if resize requires reboot
- Notify user of potential downtime
- Check target flavor compatibility (CPU arch, generation)

#### Execution

```bash
# Resize instance
hcloud ecs resize-server \
  --region {{user.region}} \
  --server-id {{user.instance_id}} \
  --new-flavor-id {{user.target_flavor}}
```

#### Validation

Poll `ShowJob(job_id)` until `SUCCESS`. Verify new flavor in `ShowServerDetail`.

### Operation: Delete ECS Instance

#### Pre-flight (Safety Gate)

- **MUST** obtain explicit confirmation with instance ID: `Delete {{user.instance_name}} ({{user.instance_id}})?`
- **MUST NOT** proceed without clear user assent
- **MUST** remind: this operation releases the instance and may not be recoverable
- **SHOULD** check for attached volumes — detach first or flag for user
- **SHOULD** check for un-released EIPs associated with instance

#### Execution

```bash
# Delete instance
hcloud ecs delete-server --region {{user.region}} --server-id {{user.instance_id}}
```

#### Validation

Poll `ShowServerDetail(server_id)` until 404/Not Found. Max 300s.

### Operation: Attach/Detach EVS Volume

#### Attach

```bash
# Attach EVS volume
hcloud evs attach-volume \
  --region {{user.region}} \
  --volume-id {{user.volume_id}} \
  --server-id {{user.instance_id}} \
  --device /dev/vdb
```

#### Post-attach: Mount on OS (via CloudShell/SSH)

```bash
# Via CloudShell remote execution
format_disk.sh /dev/vdb ext4
mount /dev/vdb /data
echo "/dev/vdb /data ext4 defaults 0 0" >> /etc/fstab
```

#### Detach

```bash
hcloud evs detach-volume --region {{user.region}} --volume-id {{user.volume_id}}
```

## CloudShell Remote Execution

Huawei Cloud CloudShell and Cloud-Cell Agent (云主机助手) provide remote command execution on ECS instances. This is critical for post-provisioning and operational tasks.

### CloudShell Overview

CloudShell provides a browser-based terminal with pre-configured Huawei Cloud CLI access to run commands against cloud resources.

### Cloud-Cell Agent (云主机助手)

The Cloud-Cell Agent is installed on ECS instances and enables remote command execution, file upload/download, and diagnostics without exposing SSH ports.

#### Installation Verification

```bash
# Check Cloud-Cell Agent status via API
hcloud ecs show-server-cloud-cell-detail --server-id {{user.instance_id}}

# Or via Go SDK (check if agent is installed and running)
```

Expected response fields:
| Field | Path | Meaning |
|-------|------|---------|
| `agent_installed` | `serverCloudCellDetail.is_install` | `true` if agent installed |
| `agent_version` | `serverCloudCellDetail.version` | Agent version |
| `status` | `serverCloudCellDetail.status` | `RUNNING` / `STOPPED` / `ERROR` |

#### Remote Command Execution

```bash
# Execute single command via Cloud-Cell Agent
hcloud ecs execute-cloud-cell-command \
  --server-id {{user.instance_id}} \
  --command "df -h" \
  --timeout 60

# Execute script file
hcloud ecs execute-cloud-cell-command \
  --server-id {{user.instance_id}} \
  --command-file /tmp/install-script.sh \
  --timeout 300
```

#### File Upload to ECS

```bash
# Upload file to ECS instance (via Cloud-Cell Agent)
hcloud ecs cloud-cell-upload \
  --server-id {{user.instance_id}} \
  --local-path /tmp/config.yaml \
  --remote-path /etc/myapp/config.yaml

# Alternative: via SCP through CloudShell (if SSH enabled)
scp -i {{user.ssh_key}} /tmp/config.yaml root@{{output.instance_ip}}:/etc/myapp/
```

#### File Download from ECS

```bash
# Download file from ECS (via Cloud-Cell Agent)
hcloud ecs cloud-cell-download \
  --server-id {{user.instance_id}} \
  --remote-path /var/log/myapp/app.log \
  --local-path /tmp/app-logs/
```

#### Typical CloudShell Workflows

| Workflow | Purpose | Commands |
|----------|---------|----------|
| Disk initialization | Format and mount new volume | `mkfs.ext4`, `mount`, `fstab` |
| Software installation | Deploy application | `apt-get install`, `yum install`, Docker setup |
| Log collection | Gather diagnostic logs | `cat /var/log/*`, `jctl`, `systemctl status` |
| System update | OS patching | `apt-get upgrade`, `reboot` |
| Performance check | Diagnose resource usage | `top`, `free -m`, `iostat`, `sar` |
| Network check | Diagnose connectivity | `ping`, `curl`, `ss -tlnp`, `nc` |

### Pre-flight for Remote Execution

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| Cloud-Cell Agent installed | `ShowServerCloudCellDetail` | `is_install: true` | Install agent via CloudShell or security group |
| Agent status running | Status check | `status: RUNNING` | Restart agent or troubleshoot |
| Network reachable | Ping via CloudShell | Response received | Check security group + VPC routing |
| Permission sufficient | IAM agent execution permission | No 403 error | Add ECS Agent execution IAM policy |

## Prerequisites

> Full installation scripts (KooCLI + Go runtime + Credentials): see [references/common-prerequisites.md](../references/common-prerequisites.md)

## Quality Gate (GCL)

This skill is **GCL-required** (per `AGENTS.md` §8). Every mutating operation — create / start /
stop / reboot / resize / delete / attach / detach / CloudShell `run-command` /
`install-cloudcell-agent` — runs through the **Generator-Critic-Loop** before its result is
returned to the user. Read-only `describe*` / `list*` operations are GCL-**exempt**.

| Field | Value |
|-------|-------|
| Rubric version | v1 (pilot, 2026-06-04) |
| `max_iter` | **2** (overridable per-op; do not raise above 2 for `delete-server`) |
| Rubric instance | [`references/rubric.md`](references/rubric.md) |
| Prompt templates | [`references/prompt-templates.md`](references/prompt-templates.md) |
| Trace path | `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` |
| Independence | Generator and Critic run in **isolated** sub-agent / session contexts (no shared prompt) |

### Five-Dimension Rubric (summary; full version in `references/rubric.md`)

| # | Dimension | Threshold | Notes |
|---|-----------|-----------|-------|
| 1 | Correctness | ≥ 0.5 (1.0 for `delete` / `stop` / `reboot` / `resize`) | Verified against `ShowServerDetail` post-state |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | See S1–S10 in rubric §2 |
| 3 | Idempotency | ≥ 0.5 | Pre-check state machine; deterministic names |
| 4 | Traceability | ≥ 0.5 | Full command + args + response + request_id; no credential leak |
| 5 | Spec Compliance | ≥ 0.5 | Region, flavor regex, image prefix, SG rules, name regex, quota |

### Per-Operation Safety Anchors (binding — `references/rubric.md` §2)

These are the most common Safety = 0 triggers for ECS. Agents MUST self-check before
submitting to the Critic:

- **S1** — `delete-server` without explicit user confirmation quoting the instance ID
- **S2** — `stop` / `reboot` / `delete` on prod-named instance (regex: `(?i)(prod|prd|production|online|pay)`) without **two-step** confirmation
- **S3** — `delete-server` while EIP still attached (orphan EIP keeps billing)
- **S4** — `delete-server` while EVS volumes attached (co-delete or block)
- **S5** — Trace contains `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / `sk-…` value
- **S6** — `resize` DOWN on a running instance (Huawei requires stop first)
- **S7** — `run-command` payload contains `rm -rf /`, `mkfs`, `dd if=`, or destructive shell
- **S8** — `resize` to flavor with less local disk than current EVS count, no detach first
- **S9** — `region` / `project_id` not in env contract (typo or default substitution)
- **S10** — `delete-server` on `prePaid` instance with > 7 days remaining, no refund-warning

### Termination Contract (per `AGENTS.md` §5)

| Condition | Status | Returned |
|-----------|--------|----------|
| All dimensions pass | **PASS** | Generator result + scores + trace path |
| `iter == max_iter` (2) and any dim < threshold | **MAX_ITER** | best-so-far + unresolved rubric items |
| `Safety == 0` | **SAFETY_FAIL** | violated S-rule id; **never** return partial |

### Trace Persistence (mandatory)

Every GCL run writes `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` with the schema in
`references/prompt-templates.md` §3. The trace is **append-only**; sanitize secrets before
write (see `prompt-templates.md` §4). The path `./audit-results/` MUST be in `.gitignore`.

### Failure Recovery (Orchestrator-level)

| Failure | Action |
|---------|--------|
| Generator sub-agent timeout (> 120s) | Re-invoke once with validation skipped; if still fails → MAX_ITER |
| Critic sub-agent timeout | Treated as `blocking=true` → MAX_ITER with `unresolved=["all"]` |
| Sub-agent returns non-JSON | Re-prompt with "JSON object only"; MAX_ITER if still bad |
| Trace file write fails | Retry once; surface a warning but do not silently continue |

### See also

- [`references/rubric.md`](references/rubric.md) — full rubric, S1–S10 rules, per-op thresholds
- [`references/prompt-templates.md`](references/prompt-templates.md) — Generator / Critic / Orchestrator prompt skeletons
- Repository root [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8 — GCL specification

## Reference Directory

- [Core Concepts](references/core-concepts.md) — ECS architecture, limits, regions, flavors
- [API & SDK Usage](references/api-sdk-usage.md) — ECS operations map, Go SDK patterns
- [CLI Usage](references/cli-usage.md) — `hcloud ecs` commands, coverage gaps
- [Troubleshooting Guide](references/troubleshooting.md) — 15+ error codes, diagnostics
- [Monitoring & Alerts](references/monitoring.md) — CES metrics, alarm patterns
- [Integration](references/integration.md) — JIT SDK setup, cross-skill delegation
- [Knowledge Base](references/knowledge-base.md) — ECS fault patterns
- [Observability](references/observability.md) — CES→LTS→AOM linkage
- [Well-Architected Assessment](references/well-architected-assessment.md) — Five pillars + FinOps + SecOps + AIOps
- [User Experience Specification](references/user-experience-spec.md) — UX compliance
- [GCL Rubric](references/rubric.md) — Adversarial quality gate (v1, 2026-06-04)
- [GCL Prompt Templates](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons

## Well-Architected + Three-Pillar Assessment

This skill's operations are evaluated against:
- **Well-Architected five pillars**: [Security](references/well-architected-assessment.md#21), [Stability](references/well-architected-assessment.md#22), [Cost](references/well-architected-assessment.md#23), [Efficiency](references/well-architected-assessment.md#24), [Performance](references/well-architected-assessment.md#25)
- **FinOps**: [Cost Optimization](references/well-architected-assessment.md#3) — billing models, right-sizing, idle detection
- **SecOps**: [Security Operations](references/well-architected-assessment.md#4) — IAM, network, encryption, HSS
- **AIOps**: [Intelligent Operations](references/advanced/aiops-best-practices.md) multi-metric correlation, cross-skill diagnosis

## FinOps — ECS Cost Optimization

> Full FinOps guidance (billing models, idle detection, right-sizing, Spot strategy, cost tagging): see [`references/well-architected-assessment.md`](references/well-architected-assessment.md#3)

Key actions:
- **Idle detection**: `cpu_util` < 10% for 7+ days → stop or delete (30-100% savings)
- **Right-sizing**: 7-day avg CPU < 20% → downgrade flavor (30-60% savings)
- **Spot instances**: Up to 90% savings for batch/stateless workloads (5-15% reclaim risk)
- **⚠️** Stopping ECS does NOT stop EVS billing

For cross-resource cost analysis, delegate to `huaweicloud-billing-ops`.
