---
name: huaweicloud-elb-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei
  Cloud ELB (Elastic Load Balancer) — load balancer lifecycle, listeners, pool
  management, health check configuration, and diagnostics. User mentions ELB,
  弹性负载均衡, 负载均衡器, 负载均衡, 均衡, 流量分发, or describes scenarios
  (backend unhealthy, connection drops, slow response, traffic spike) even
  without naming ELB directly. Not for VPC networking, ECS instance management,
  WAF protection, or DNS resolution that have their own ops skills.
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
  api_profile: "https://support.huaweicloud.com/api-elb/"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    Huawei Cloud ELB is supported via `hcloud elb` CLI commands and
    huaweicloud-sdk-go-v3/services/elb/v3 Go SDK package.
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
        change: "GCL Phase 3 rollout: added references/rubric.md (v1, 5-dim, S1–S13 ELB-specific Safety rules, including delete-without-confirmation / listener-orphan / backend-quorum-loss / cert-expiry-nudging guards) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud ELB Operations Skill

## Overview

Huawei Cloud ELB (Elastic Load Balancer / 弹性负载均衡) provides traffic distribution across backend servers. This skill is an **operational runbook** for agents: load balancer lifecycle, listener configuration, backend pool management, health check setup, SSL/TLS certificate binding, response validation, and failure recovery. **Dual-path execution**: both **SDK/API** (`huaweicloud-sdk-go-v3/services/elb/v3`) and **`hcloud elb` CLI**.

> **UX Compliance:** This skill follows the User Experience Specification. All operations include onboarding guidance, minimal prompts, smart defaults, clear feedback, and user-friendly error handling.

### CLI Applicability (repository policy)

- **`cli_applicability: dual-path`** — Official `hcloud elb` CLI supports most ELB operations. **MUST** document both SDK and CLI paths for every operation.

### Well-Architected + Three-Pillar Integration

This skill integrates Huawei Cloud Well-Architected five pillars plus FinOps, SecOps, and AIOps:
- [Security Assessment](references/well-architected-assessment.md#21)
- [FinOps Cost Optimization](references/well-architected-assessment.md#3)
- [SecOps Security Operations](references/well-architected-assessment.md#4)
- [AIOps Integration](references/advanced/aiops-best-practices.md)

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT triggers with precise keywords, delegation rules to ECS/CES/WAF skills |
| 2 | **Structured I/O** | `{{env.*}}` for credentials, `{{user.*}}` for LB config, `{{output.*}}` for API responses |
| 3 | **Explicit Steps** | Every operation: Pre-flight → Execute → Validate → Recover with numbered imperative steps |
| 4 | **Failure Strategies** | 20+ ELB-specific error codes with HALT vs retry distinction |
| 5 | **Single Responsibility** | ELB lifecycle only; delegates ECS pool members to ECS skill, monitoring to CES skill |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud ELB", "弹性负载均衡", "负载均衡器", "负载均衡", "均衡", "流量分发"
- Task involves LB lifecycle: create, describe, modify, delete, list
- Task involves listener: create HTTP/HTTPS/TCP/UDP listeners, modify, delete
- Task involves backend pool: add/remove members, configure health checks, manage pools
- Task involves SSL/TLS: bind/unbind certificate, configure HTTPS listener
- Task involves LB type: shared (经典) vs dedicated (独享型), network (NLB) vs application (ALB)
- Task keywords: `load balancer`, `listener`, `backend`, `pool`, `member`, `health check`, `listener`, `均衡`

### SHOULD NOT Use This Skill When

- Task is purely billing / cost analysis → delegate to: `huaweicloud-billing-ops`
- Task is IAM permission model only → delegate to: `huaweicloud-iam-ops` (when present)
- Task is VPC/subnet creation → delegate to: `huaweicloud-vpc-ops` (when present)
- Task is WAF protection configuration → delegate to: `huaweicloud-waf-ops` (when present)
- Task is DNS resolution → delegate to: `huaweicloud-dns-ops` (when present)
- Task is ECS instance management → delegate to: `huaweicloud-ecs-ops`
- Task is ECS security group → delegate to: `huaweicloud-ecs-ops`

### Delegation Rules

- ELB requires VPC/subnet → verify with `huaweicloud-vpc-ops` first
- Backend members are typically ECS → delegate instance verification to `huaweicloud-ecs-ops`
- Health check failing → check CES metrics for backend instances (`huaweicloud-ces-ops`)
- HTTPS listener → SSL certificate management via ELB skill (upload/list certificates)
- WAF integration → delegate WAF policy to `huaweicloud-waf-ops`

## Variable Convention

| Placeholder | Meaning | Agent Action |
|-------------|---------|--------------|
| `{{env.HW_ACCESS_KEY_ID}}` | From runtime environment | NEVER ask user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | From runtime environment | NEVER ask user; fail if unset |
| `{{env.HW_REGION_ID}}` | Default region (e.g., `cn-north-4`) | Use if skill allows |
| `{{env.HW_PROJECT_ID}}` | Project ID | Use for scoped operations |
| `{{user.lb_name}}` | LB name | Ask once; reuse |
| `{{user.lb_type}}` | LB type (`shared`/`dedicated`) | Ask with recommendation |
| `{{user.vpc_id}}` | VPC ID | Use VPC skill to get |
| `{{user.subnet_id}}` | Subnet ID | Use VPC skill to get |
| `{{user.listener_port}}` | Listener port (e.g., 80, 443) | Ask with common defaults |
| `{{user.listener_protocol}}` | Protocol (HTTP/HTTPS/TCP/UDP) | Suggest based on use case |
| `{{output.lb_id}}` | LB ID | Parse from create response |
| `{{output.listener_id}}` | Listener ID | Parse from response |
| `{{output.pool_id}}` | Pool ID | Parse from response |

> **`{{env.*}}` MUST NOT** be collected from user. **Credential masking is MANDATORY** — never echo `HW_SECRET_ACCESS_KEY`.

## Quick Start

### What This Skill Does
Manage Huawei Cloud ELB resources: create load balancers, configure listeners, manage backend pools, set up health checks, and troubleshoot.

### Prerequisites

## API and Response Conventions

- **OpenAPI canonical**: `https://support.huaweicloud.com/api-elb/`
- **API version**: v3 (current), v2 (legacy)
- **LB types**: `shared` (classic, multi-tenant), `dedicated` (独享型, single-tenant)
- **LB categories**: `network` (NLB, L4 TCP/UDP), `application` (ALB, L7 HTTP/HTTPS)
- **Pagination**: `limit` + `marker`, default 2000 per page
- **Async pattern**: Create/delete LB returns `job_id` — poll via ELB job API
- **Idempotency**: Client token for idempotent creation

## Expected State Transitions

| Operation | Initial State | Target State | Poll API | Max Wait |
|-----------|--------------|--------------|----------|----------|
| Create (shared) | — | `ACTIVE` | `ShowLoadBalancer` | 180s |
| Create (dedicated) | — | `ACTIVE` | `ShowLoadBalancer` | 600s |
| Modify | `ACTIVE` | `ACTIVE` | `ShowLoadBalancer` | 120s |
| Delete | `ACTIVE` | absent | `ShowLoadBalancer` 404 | 300s |
| Add member | — | — | `ShowMember` | 60s |
| Health check update | — | — | `ShowHealthMonitor` | 30s |

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|-----------|-------------|------------|------------|
| Create | Create a load balancer | High | Low |
| Describe | View LB details | Low | None |
| Modify | Change LB configuration | Medium | Medium — may affect traffic |
| Delete | Remove a load balancer | High | **High** — traffic down |
| List | View all LBs | Low | None |
| ManageListener | Create/modify/delete listeners | Medium | Medium |
| ManagePool | Create/modify/delete backend pools | Medium | Medium |
| ManageMember | Add/remove backend members | Low | Low |
| ManageHealthCheck | Configure health check rules | Low | Medium — may cause false unhealthy |
| ManageCertificate | Upload/list/delete SSL certs | Low | Low |

## Execution Flows

### Operation: Create Load Balancer

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| VPC/subnet exists | `hcloud vpc list-vpcs` | VPC and subnet IDs valid | Create via VPC skill first |
| LB type available | `hcloud elb list-availability-zones` | Region supports type | Suggest valid LB type |
| Quota sufficient | `hcloud elb show-quota` | Quota > 0 | HALT — request quota increase |
| Credentials valid | `hcloud elb list-loadbalancers` | Non-401 response | HALT — user configures credentials |
| EIP (if public LB) | Check EIP available | EIP exists or can create | Delegate to VPC skill for EIP |
| TLS version compliance | Check listener protocol config | HTTPS uses TLS >= 1.2 (production) | HALT — require TLS 1.2+ for production HTTPS listeners |

#### Execution — CLI (Primary Path)

```bash
# Create a dedicated application load balancer
hcloud elb create-loadbalancer \
  --region {{env.HW_REGION_ID}} \
  --name "{{user.lb_name}}" \
  --vpc-id "{{user.vpc_id}}" \
  --elb-virsubnet-ids "{{user.subnet_id}}" \
  --loadbalancer-type "dedicated" \
  --availability-zone-list "{{user.az}}" \
  --description "Production load balancer" \
  --admin-state-up true
```

#### Execution — JIT Go SDK (Fallback Path)

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    elb "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v3"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v3/model"
    elbregion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v3/region"
)

func main() {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    regionID := os.Getenv("HW_REGION_ID")
    
    client := elb.NewElbClient(
        elb.ElbClientBuilder().
            WithRegion(elbregion.ValueOf(regionID)).
            WithCredential(basic.NewCredentialsBuilder().WithAk(ak).WithSk(sk).Build()).
            Build())
    
    request := &model.CreateLoadBalancerRequest{
        Body: &model.CreateLoadBalancerRequestBody{
            Loadbalancer: &model.CreateLoadBalancerOption{
                Name:             func() *string { v := os.Getenv("LB_NAME"); return &v }(),
                VpcId:            os.Getenv("VPC_ID"),
                ElbVirsubnetIds:  []string{os.Getenv("SUBNET_ID")},
                LoadbalancerType: "dedicated",
                AvailabilityZoneList: []string{os.Getenv("AZ")},
                Description:      func() *string { v := "Production LB"; return &v }(),
                AdminStateUp:     func() *bool { v := true; return &v }(),
            },
        },
    }
    
    response, err := client.CreateLoadBalancer(context.TODO(), request)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Create LB failed: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Printf("LB ID: %s\n", *response.Loadbalancer.Id)
    fmt.Printf("LB Name: %s\n", *response.Loadbalancer.Name)
    fmt.Printf("Provisioning Status: %s\n", *response.Loadbalancer.ProvisioningStatus)
    fmt.Printf("Operating Status: %s\n", *response.Loadbalancer.OperatingStatus)
}
```

#### Post-execution Validation

1. Extract `{{output.lb_id}}` from create response (`loadbalancer.id`).
2. Poll `ShowLoadBalancer(loadbalancer_id)` until `provisioning_status` is `ACTIVE`.
3. Verify `operating_status` is `ONLINE`.
4. Report LB ID, IP address (if EIP attached), and status to user.

#### Failure Recovery

| Error | Max Retries | Agent Action | UX Feedback |
|-------|-------------|--------------|-------------|
| `ELB.1001` InvalidParameter | 0 | HALT | `[ERROR] Invalid parameter. Check LB name, description against API docs.` |
| `ELB.1002` VpcNotFound | 0 | HALT | `[ERROR] VPC not found. Create via `huaweicloud-vpc-ops` first.` |
| `ELB.1003` SubnetNotFound | 0 | HALT | `[ERROR] Subnet not found. Verify subnet ID.` |
| `ELB.1004` QuotaExceeded | 0 | HALT | `[ERROR] ELB quota exceeded. Delete unused LBs or request quota increase.` |
| `ELB.1005` InsufficientBalance | 0 | HALT | `[ERROR] Insufficient balance. Recharge your Huawei Cloud account.` |
| `ELB.1006` AZNotSupported | 0 | HALT | `[ERROR] Availability zone not supported. List available AZs.` |
| `ELB.2001` ListenerPortConflict | 0 | HALT | `[ERROR] Listener port already in use. Choose different port.` |
| `ELB.2002` ListenerProtocolInvalid | 0 | HALT | `[ERROR] Invalid protocol for this LB type. Check supported protocols.` |
| `ELB.3001` PoolNotFound | 0 | HALT | `[ERROR] Backend pool not found. Create pool first.` |
| `ELB.3002` MemberAlreadyExists | 0 | HALT | `[ERROR] Backend member already in pool. Check existing members.` |
| `ELB.3003` HealthCheckInvalid | 0 | HALT | `[ERROR] Health check configuration invalid. Verify delay/timeout/max_retries.` |
| `ELB.4001` CertificateNotFound | 0 | HALT | `[ERROR] SSL certificate not found. Upload certificate first.` |
| `ELB.4002` CertificateExpired | 0 | HALT | `[ERROR] SSL certificate expired. Upload new certificate.` |
| Throttling 429 | 3 | Exponential backoff | `⚠️ Rate limited. Retrying in {backoff}s...` |
| InternalError 500 | 3 | Backoff 2s→4s→8s | `[ERROR] Server error. Retry or escalate with RequestId.` |

### Operation: Manage Listener

#### Create Listener

```bash
# Create HTTP listener on port 80
hcloud elb create-listener \
  --region {{env.HW_REGION_ID}} \
  --protocol-port 80 \
  --protocol HTTP \
  --loadbalancer-id "{{output.lb_id}}" \
  --name "{{user.lb_name}}-http-80" \
  --default-pool-id "{{output.pool_id}}" \
  --admin-state-up true
```

#### JIT Go SDK Path

```go
request := &model.CreateListenerRequest{
    Body: &model.CreateListenerRequestBody{
        Listener: &model.CreateListenerOption{
            ProtocolPort:    func() *int32 { v := int32(80); return &v }(),
            Protocol:        "HTTP",
            LoadbalancerId:  os.Getenv("LB_ID"),
            Name:            func() *string { v := os.Getenv("LISTENER_NAME"); return &v }(),
            DefaultPoolId:   func() *string { v := os.Getenv("POOL_ID"); return &v }(),
            AdminStateUp:    func() *bool { v := true; return &v }(),
        },
    },
}
```

#### Create HTTPS Listener

```bash
hcloud elb create-listener \
  --region {{env.HW_REGION_ID}} \
  --protocol-port 443 \
  --protocol HTTPS \
  --loadbalancer-id "{{output.lb_id}}" \
  --default-tls-container-ref "{{output.certificate_id}}" \
  --name "{{user.lb_name}}-https-443"
```

### Operation: Manage Backend Pool

#### Create Pool

```bash
# Create backend pool
hcloud elb create-pool \
  --region {{env.HW_REGION_ID}} \
  --protocol HTTP \
  --lb-algorithm ROUND_ROBIN \
  --listener-id "{{output.listener_id}}" \
  --name "{{user.pool_name}}"
```

**Supported algorithms**: `ROUND_ROBIN`, `LEAST_CONNECTIONS`, `SOURCE_IP`, `QUIC_CID`

#### Add Backend Member

```bash
# Add ECS instance as backend member
hcloud elb create-member \
  --region {{env.HW_REGION_ID}} \
  --pool-id "{{output.pool_id}}" \
  --address "{{user.member_ip}}" \
  --protocol-port 8080 \
  --subnet-id "{{user.subnet_id}}"
```

### Operation: Configure Health Check

```bash
# Create health monitor for pool
hcloud elb create-healthmonitor \
  --region {{env.HW_REGION_ID}} \
  --pool-id "{{output.pool_id}}" \
  --delay 5 \
  --timeout 3 \
  --max-retries 3 \
  --type HTTP \
  --url-path "/health" \
  --expected-codes "200-399"
```

#### Post-execution Validation

1. Verify health monitor is created: `hcloud elb show-healthmonitor --healthmonitor-id {{output.healthmonitor_id}}`
2. Check members' operating status: `hcloud elb list-members --pool-id {{output.pool_id}}`
3. Expected: members with `operating_status` = `ONLINE`

### Operation: Delete Load Balancer

#### Pre-flight (Safety Gate)

- **MUST** obtain explicit confirmation with LB ID: `Delete load balancer {{user.lb_name}} ({{output.lb_id}})?`
- **MUST NOT** proceed without clear user assent
- **MUST** warn: this operation terminates all traffic through this LB
- **SHOULD** list all listeners and warn they will be deleted
- **SHOULD** check if associated EIP needs separate release
- **SHOULD** recommend draining connections before deletion

#### Execution

```bash
# Delete load balancer (and all associated listeners/members)
hcloud elb delete-loadbalancer \
  --region {{env.HW_REGION_ID}} \
  --loadbalancer-id "{{output.lb_id}}" \
  --cascade true  # Also delete listeners, pools, monitors
```

#### Validation

Poll `ShowLoadBalancer(lb_id)` until 404 Not Found. Max 300s.

### Operation: List/Describe ELB Resources

```bash
# List all load balancers
hcloud elb list-loadbalancers --region {{env.HW_REGION_ID}}

# Describe single load balancer
hcloud elb show-loadbalancer --loadbalancer-id "{{output.lb_id}}"

# List listeners for a LB
hcloud elb list-listeners --loadbalancer-id "{{output.lb_id}}"

# List backend pools
hcloud elb list-pools --loadbalancer-id "{{output.lb_id}}"

# List backend members with health status
hcloud elb list-members --pool-id "{{output.pool_id}}"

# List availability zones for LB creation
hcloud elb list-availability-zones --region {{env.HW_REGION_ID}}
```

## Prerequisites

> Full installation scripts (KooCLI + Go runtime + Credentials): see [references/common-prerequisites.md](../references/common-prerequisites.md)

## Quality Gate (GCL)

This skill is **GCL-recommended** (per `AGENTS.md` §8). Every ELB mutating operation — load balancer create / delete, listener create / update / delete, backend pool create / delete / member add-remove, certificate replace — runs through the **Generator-Critic-Loop** before its result is returned. Read-only `describe*` / `list*` are GCL-**exempt**.

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
| 1 | Correctness | ≥ 0.5 (1.0 for `delete-lb` / `delete-listener` / `delete-pool`) | `ShowLoadBalancer` / `ShowListener` post-state |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | S1–S13 in rubric §2 |
| 3 | Idempotency | ≥ 0.5 | Pre-check before create |
| 4 | Traceability | ≥ 0.5 | Credential MUST be `<masked>` |
| 5 | Spec Compliance | ≥ 0.5 | ELB type / protocol / port / health check params |

### Per-Operation Safety Anchors (binding)

- **S1** — `delete-lb` without explicit user confirmation quoting the LB ID
- **S2** — `delete-lb` while the LB still has active listeners (orphaned)
- **S3** — `delete-lb` while the LB still has active backend pools
- **S4** — `delete-lb` with active EIP bound without warning (EIP orphan)
- **S5** — `create-listener` referencing a non-existent or protocol-mismatched pool
- **S6** — `update-listener` switching to a pool incompatible with listener protocol
- **S7** — `delete-pool` while member servers have active connections (connection drop)
- **S8** — `add-member` with invalid subnet or unreachable IP address
- **S9** — `delete-member` without checking if it is the last healthy member (quorum loss)
- **S10** — `update-certificate` (replace SSL/TLS) on a listener without maintenance window
- **S11** — `create-listener` with `protocol_port` already in use on the same LB
- **S12** — any trace contains `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / password plaintext
- **S13** — `delete-certificate` while it is actively bound to an HTTPS listener

### Termination Contract (per `AGENTS.md` §5)

| Condition | Status | Returned |
|-----------|--------|----------|
| All dimensions pass | **PASS** | Generator result + scores + trace path |
| `iter == max_iter` (3) and any dim < threshold | **MAX_ITER** | best-so-far + unresolved rubric items |
| `Safety == 0` | **SAFETY_FAIL** | violated S-rule id; **never** return partial |

### Trace Persistence (mandatory)

Every GCL run writes `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` (schema in `references/prompt-templates.md` §3). Trace is **append-only**; sanitize secrets before write (see `prompt-templates.md` §4). The path `./audit-results/` is in root `.gitignore`.

### Prompt Backbone

Use `huaweicloud-skill-generator/references/gcl-prompt-backbone.md` as the shared source
for Generator/Critic/Orchestrator wording. Runtime Roles (Generator / Critic /
Orchestrator) and their isolation constraints: see `docs/gcl-spec.md` §Runtime Roles
and root `AGENTS.md` §5. Default rubric thresholds (correctness ≥0.5, safety =1.0,
…): see `docs/gcl-spec.md` §Thresholds. Trace persistence + masking rules: see
`docs/gcl-spec.md` §Trace and root `AGENTS.md` (credential masking mandatory).
This skill's `references/prompt-templates.md` keeps ELB-specific overrides and must
not introduce bare `{...}` placeholders.

### See also

- [`references/rubric.md`](references/rubric.md) — full rubric, S1–S13 rules, per-op thresholds
- [`references/prompt-templates.md`](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons
- Repository root [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8 — GCL specification

## Reference Directory

- [Core Concepts](references/core-concepts.md) — ELB types, architecture, limits
- [API & SDK Usage](references/api-sdk-usage.md) — Operation map, request/response
- [CLI Usage](references/cli-usage.md) — CLI command map, coverage gap table
- [Troubleshooting Guide](references/troubleshooting.md) — Error codes, diagnostic flows
- [Monitoring & Alerts](references/monitoring.md) — CES metrics, dashboards, alarm patterns
- [Integration](references/integration.md) — JIT SDK setup, cross-skill delegation matrix
- [Well-Architected Assessment](references/well-architected-assessment.md) — Five pillars + FinOps + SecOps + AIOps
- [GCL Rubric](references/rubric.md) — Adversarial quality gate (v1, 5-dim, S1–S13 ELB-specific Safety rules)
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
