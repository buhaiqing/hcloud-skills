---
name: huaweicloud-vpc-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei
  Cloud Virtual Private Cloud (VPC / 虚拟私有云) — VPC lifecycle, subnets, security
  groups, route tables, EIPs, bandwidth, NAT gateways, and VPC peering. User mentions
  VPC, 虚拟私有云, 子网, 安全组, 弹性公网IP, 带宽, NAT网关, or describes scenarios
  (e.g., network isolation, security group rule creation, EIP binding, VPC peering,
  cross-VPC communication) even without naming the product directly.
  Not for billing, IAM, or server/instance provisioning that has dedicated ops skills.
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
  api_profile: "VPC API v3 - https://support.huaweicloud.com/api-vpc/vpc_api_0001.html"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    VPC product fully supported by hcloud CLI. Use `hcloud vpc --help` and
    `hcloud eip --help` to verify available commands.
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
        change: "GCL Phase 2 rollout: added references/rubric.md (v1, 5-dim, S1–S17 VPC/Subnet/SG/EIP/NAT-specific Safety rules) and references/prompt-templates.md (Generator + Critic + Orchestrator). SKILL.md gains 'Quality Gate (GCL)' chapter."
      - version: "1.0.0"
        date: "2026-05-20"
        change: "Initial skill release."
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud Virtual Private Cloud (VPC) Operations Skill

## Overview

Huawei Cloud Virtual Private Cloud (VPC / 虚拟私有云) provides isolated, customizable virtual networks for cloud resources. This skill is an **operational runbook** for agents: explicit scope, credential rules, pre-flight checks, **dual-path execution** (official **SDK/API** and **`hcloud` CLI**), response validation, and failure recovery.

### CLI applicability (repository policy)

- **`cli_applicability: dual-path`:** Official CLI supports VPC and EIP products. You **MUST** ship **`references/cli-usage.md`** and, in **each** execution flow, document **both** the SDK step **and** the CLI step.

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions with precise triggers and cross-product delegation |
| 2 | **Structured I/O** | Placeholder conventions with type and source documented |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute (CLI + SDK) → Validate → Recover |
| 4 | **Complete Failure Strategies** | Error taxonomy ≥ 10 codes; HALT vs retry per error type |
| 5 | **Absolute Single Responsibility** | One product (VPC), one resource model; delegation to ECS/RDS for resource provisioning |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **FinOps** | Bandwidth cost optimization, EIP billing modes, idle IP detection | `references/well-architected-assessment.md` §3 |
| **SecOps** | Security group least-privilege rules, VPC isolation, encryption | `references/well-architected-assessment.md` §4 |
| **AIOps** | ≥ 5 anomaly patterns, cross-skill diagnosis for network issues | `references/advanced/aiops-best-practices.md` and §5 |

### Well-Architected Framework Integration (卓越架构)

| Pillar | Skill Integration | Reference |
|--------|-------------------|-----------|
| **安全 (Security)** | Security groups, NACLs, VPC isolation, encryption | `references/well-architected-assessment.md` §2.1 |
| **稳定 (Stability)** | Multi-AZ subnet distribution, redundant NAT gateways, HA design | `references/well-architected-assessment.md` §2.2 |
| **成本 (Cost)** | EIP billing comparison, bandwidth optimization, idle resource cleanup | `references/well-architected-assessment.md` §2.3 |
| **效率 (Efficiency)** | CIDR planning, automated provisioning via IaC, route table templates | `references/well-architected-assessment.md` §2.4 |
| **性能 (Performance)** | Bandwidth tuning, VPC peering vs VPN selection, NAT throughput | `references/well-architected-assessment.md` §2.5 |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud VPC", "Virtual Private Cloud", "虚拟私有云", "VPC"
- Task involves VPC lifecycle: create, list, view, delete VPCs
- Task involves subnet management: create, list, update, delete subnets
- Task involves security group operations: create, manage rules, delete
- Task involves route table operations: create, add routes, delete
- Task involves EIP lifecycle: allocate, bind, unbind, release
- Task involves bandwidth management: create, update, associate/dissociate
- Task involves VPC peering: create, accept, manage peering connections
- Task involves NAT gateway: create, manage DNAT/SNAT rules
- Task keywords: 子网, 安全组, 路由表, 弹性公网IP, 共享带宽, VPC对等连接, NAT网关, 网络隔离, 安全组规则, 公网IP绑定
- User asks to configure, troubleshoot, or monitor VPC resources via API, SDK, CLI, or automation

### SHOULD NOT Use This Skill When

- Task is purely billing / account management → delegate to: `huaweicloud-billing-ops`
- Task is IAM / permission model only → delegate to: `huaweicloud-iam-ops`
- Task is creating/deleting VMs or instances → delegate to: `huaweicloud-ecs-ops`
- Task is database RDS operations → delegate to: `huaweicloud-rds-ops`
- Task is load balancer (ELB) → delegate to: `huaweicloud-elb-ops`

### Delegation Rules

- VPC/subnet must exist before provisioning ECS/RDS/ELB resources into them. Complete or verify VPC setup first.
- EIP binding requires target resource (ECS/ELB) to exist — verify in respective product skill.
- Security group rules reference ports/protocols — ensure they match the target service configuration.
- For FinOps VPC bandwidth costs: use this skill's cost section; delegate cross-resource cost to billing skill.
- For SecOps: this skill covers security groups and VPC isolation; delegate account-level IAM to IAM skill.

## Variable Convention (Agent-Readable)

| Placeholder | Meaning | Agent Action |
|-------------|---------|--------------|
| `{{env.HW_ACCESS_KEY_ID}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | From runtime environment | NEVER ask the user; fail if unset |
| `{{env.HW_REGION_ID}}` | From runtime environment | Use documented default only if skill explicitly allows |
| `{{env.HW_PROJECT_ID}}` | From runtime environment | Use documented default only if skill explicitly allows |
| `{{user.region}}` | User-supplied region | Ask once; reuse |
| `{{user.vpc_name}}` | User-supplied VPC name | Ask once; reuse |
| `{{user.vpc_id}}` | User-supplied VPC ID | Ask once; reuse |
| `{{user.cidr}}` | User-supplied CIDR block | Ask if not provided; validate format |
| `{{user.security_group_id}}` | User-supplied SG ID | Ask once; reuse |
| `{{user.eip_id}}` | User-supplied EIP ID | Ask once; reuse |
| `{{user.instance_id}}` | User-supplied target resource ID | Ask once; reuse |
| `{{output.vpc_id}}` | From VPC create response | Parse per OpenAPI: `$.vpc.id` |
| `{{output.subnet_id}}` | From subnet create response | Parse per OpenAPI: `$.subnet.id` |
| `{{output.eip_id}}` | From EIP allocate response | Parse per OpenAPI: `$.publicip.id` |

> **`{{env.*}}` MUST NOT** be collected from the user. **`{{user.*}}`** MUST be collected interactively when missing.

> **Security Warning (Credential Masking — MANDATORY):** **NEVER** log, print, or expose `HW_SECRET_ACCESS_KEY` or any credential field value.

## API and Response Conventions

- **OpenAPI is canonical** for path, query, body fields, enums, and response shapes.
- **Errors:** Map SDK/HTTP errors to `error` / `message` fields per spec.
- **Timestamps:** ISO 8601.
- **Idempotency:** Resource names are not unique per API; use client tokens or describe-before-create for idempotent operations.

## Quick Start

### What This Skill Does
Manages Huawei Cloud VPC (Virtual Private Cloud / 虚拟私有云) including VPC, subnets, security groups, route tables, EIPs, bandwidth, NAT gateways, and VPC peering.

### Prerequisites

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|-----------|-------------|------------|------------|
| CreateVPC | Create VPC with CIDR block | Low | Low |
| ListVPCs | List VPCs | Low | None |
| DescribeVPC | Get VPC details | Low | None |
| DeleteVPC | Delete VPC (must be empty) | Low | **High** — irreversible |
| CreateSubnet | Create subnet in VPC | Low | Low |
| ListSubnets | List subnets | Low | None |
| DeleteSubnet | Delete subnet | Low | **High** |
| CreateSecurityGroup | Create security group | Low | Low |
| AddSecurityGroupRule | Add inbound/outbound rule | Medium | Medium |
| DeleteSecurityGroupRule | Remove rule | Low | Medium |
| CreateRouteTable | Create route table | Low | Low |
| AddRoute | Add route entry | Medium | Medium |
| AllocateEIP | Allocate elastic IP | Low | Low (billing) |
| BindEIP | Bind EIP to resource | Low | Low |
| UnbindEIP | Unbind EIP | Low | Low |
| ReleaseEIP | Release EIP | Low | **High** — irreversible |
| CreateBandwidth | Create shared bandwidth | Low | Low (billing) |
| CreateNATGateway | Create NAT gateway | Medium | Low |
| AddDNATRule | Add DNAT rule | Medium | Medium |
| AddSNATRule | Add SNAT rule | Medium | Medium |
| CreateVpcPeering | Create VPC peering | Medium | Low |

## Execution Flows

### Operation: Create VPC

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|-------|--------|----------|------------|
| CIDR validation | Validate CIDR format (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16) | Valid RFC 1918 range | HALT; provide valid CIDR |
| CIDR overlap | Check against existing VPC CIDRs in project | No overlap | HALT; use non-overlapping CIDR |
| Quota | Check VPC quota (default: 5 per project) | Sufficient quota | HALT; request quota increase |

#### Execution — CLI (Primary Path)

```bash
hcloud vpc create \
  --region "{{user.region}}" \
  --name "{{user.vpc_name}}" \
  --cidr "{{user.cidr:192.168.0.0/16}}"
```

#### Execution — JIT Go SDK (Fallback Path)

```go
package main

import (
    "fmt"
    "os"

    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v2/model"
)

func main() {
    ak := os.Getenv("HW_ACCESS_KEY_ID")
    sk := os.Getenv("HW_SECRET_ACCESS_KEY")
    region := os.Getenv("HW_REGION_ID")

    cfg := config.DefaultHttpConfig()
    client := vpc.VpcClientBuilder().
        WithEndpoint(fmt.Sprintf("vpc.%s.myhuaweicloud.com", region)).
        WithCredential(basic.NewCredentialsBuilder().WithAk(ak).WithSk(sk).Build()).
        WithHttpConfig(cfg).Build()

    request := &model.CreateVpcRequest{
        Body: &model.Vpc{
            Name: os.Getenv("VPC_NAME"),
            Cidr: os.Getenv("VPC_CIDR"),
        },
    }

    response, err := client.CreateVpc(request)
    if err != nil {
        panic(err)
    }
    fmt.Printf("%+v\n", response)
}
```

#### Post-execution Validation

1. Read `{{output.vpc_id}}` from response path `$.vpc.id`.
2. Call **GetVpc** with `{{output.vpc_id}}` to confirm exists.
3. On success, report VPC ID, name, CIDR, and status.
4. On terminal failure, go to **Failure Recovery**.

#### Failure Recovery

| Error | Max retries | Backoff | Agent Action | UX Feedback |
|-------|-------------|---------|--------------|-------------|
| `VPC.0003` InvalidParameter | 0–1 | — | Fix args | `[ERROR] InvalidParameter: Invalid CIDR or name.` |
| `VPC.0010` CidrConflict | 0 | — | HALT | `[ERROR] CIDR conflicts with existing VPC. Use different range.` |
| `VPC.0020` QuotaExceeded | 0 | — | HALT | `[ERROR] VPC quota exceeded. Delete unused VPCs or request increase.` |
| `InvalidParameter` | 0–1 | — | Fix args | `[ERROR] Invalid parameter. Review input.` |
| `InsufficientBalance` | 0 | — | HALT | `[ERROR] Insufficient balance. Recharge account.` |
| Throttling / 429 | 3 | exponential | Back off | `⚠️ Rate limited. Retrying...` |
| `InternalError` / 5xx | 3 | 2s, 4s, 8s | Retry; then HALT | `[ERROR] Server error. Retry or escalate with RequestId.` |
| `VPC.0016` ProjectNotAuthorized | 0 | — | Verify IAM | `[ERROR] Unauthorized. Check IAM permissions.` |

### Operation: Create Subnet

#### Pre-flight Checks

- Verify VPC exists and is ACTIVE.
- Subnet CIDR must be within parent VPC CIDR.
- Check subnet quota per VPC.

#### Execution — CLI

```bash
hcloud vpc create-subnet \
  --region "{{user.region}}" \
  --vpc-id "{{user.vpc_id}}" \
  --name "{{user.subnet_name}}" \
  --cidr "{{user.subnet_cidr:192.168.1.0/24}}" \
  --gateway-ip "192.168.1.1"
```

#### Post-execution Validation

- Read `{{output.subnet_id}}` from response `$.subnet.id`.
- Call **GetSubnet** to confirm.
- Report subnet ID, name, CIDR, and gateway.

### Operation: Delete Subnet

#### Pre-flight (Safety Gate)

- **MUST** verify no resources (ECS, RDS, ELB) are deployed in subnet.
- **MUST** obtain explicit confirmation for deletion.
- **MUST** warn: irreversible operation.

#### Execution

```bash
hcloud vpc delete-subnet \
  --region "{{user.region}}" \
  --vpc-id "{{user.vpc_id}}" \
  --subnet-id "{{user.subnet_id}}"
```

### Operation: Create Security Group

#### Pre-flight Checks

- Verify VPC exists.
- Collect security group name and description.

#### Execution — CLI

```bash
hcloud vpc create-security-group \
  --region "{{user.region}}" \
  --vpc-id "{{user.vpc_id}}" \
  --name "{{user.sg_name}}" \
  --description "{{user.sg_description}}"
```

#### Post-execution Validation

- Read security group ID from response `$.security_group.id`.
- Report ID, name, VPC, and default rules.

### Operation: Add Security Group Rule

#### Pre-flight Checks

- Verify security group exists.
- Validate port range (1-65535), protocol (tcp/udp/icmp/any).
- Validate CIDR for source/destination.

#### Execution — CLI

```bash
# Add inbound rule
hcloud vpc create-security-group-rule \
  --region "{{user.region}}" \
  --security-group-id "{{user.security_group_id}}" \
  --direction "ingress" \
  --protocol "{{user.protocol:tcp}}" \
  --port-range "{{user.port_range:22}}" \
  --remote-ip-prefix "{{user.source_cidr:0.0.0.0/0}}" \
  --description "{{user.rule_description:Allow SSH access}}"

# Add outbound rule
hcloud vpc create-security-group-rule \
  --region "{{user.region}}" \
  --security-group-id "{{user.security_group_id}}" \
  --direction "egress" \
  --protocol "{{user.protocol:tcp}}" \
  --port-range "{{user.port_range:443}}" \
  --remote-ip-prefix "{{user.dest_cidr:0.0.0.0/0}}" \
  --description "{{user.rule_description:Allow HTTPS outbound}}"
```

#### Security Best Practices

- **NEVER** use `0.0.0.0/0` for sensitive ports (22, 3389, 3306) — use specific IP ranges
- Follow least-privilege: open only required ports from known sources
- Default deny: ensure no overly permissive rules exist

### Operation: Allocate EIP

#### Pre-flight Checks

- Check EIP quota.
- Determine bandwidth type and size.
- **Warn user**: EIP incurs billing (bandwidth traffic or fixed bandwidth).

#### Execution — CLI

```bash
hcloud eip create \
  --region "{{user.region}}" \
  --type "{{user.ip_type:5_bgp}}" \
  --bandwidth-size "{{user.bandwidth_size:5}}" \
  --bandwidth-share-type "{{user.share_type:PER}}"
```

#### Post-execution Validation

- Read `{{output.eip_id}}` from response `$.publicip.id`.
- Read public IP address from `$.publicip.public_ip_address`.
- Report EIP ID, IP address, and bandwidth details.

### Operation: Bind EIP to Resource

#### Pre-flight Checks

- Verify EIP is in "FREE" or "BINDING" state (not already bound to another resource).
- Verify target resource (ECS, ELB, NAT) exists.
- Check resource binding quota/limits.

#### Execution — CLI

```bash
hcloud eip bind \
  --region "{{user.region}}" \
  --publicip-id "{{user.eip_id}}" \
  --port-id "{{user.port_id}}"
```

### Operation: Release EIP

#### Pre-flight (Safety Gate)

- **MUST** unbind first: Verify EIP is not bound to any resource.
- **MUST** obtain explicit confirmation — irreversible.
- **MUST** warn about billing implications.

#### Execution

```bash
hcloud eip delete \
  --region "{{user.region}}" \
  --publicip-id "{{user.eip_id}}"
```

### Operation: Create NAT Gateway

#### Pre-flight Checks

- Verify VPC exists.
- Verify subnet exists in the VPC (NAT gateway must be placed in a subnet).

#### Execution — CLI

```bash
hcloud nat create-gateway \
  --region "{{user.region}}" \
  --name "{{user.nat_name}}" \
  --router-id "{{user.vpc_id}}" \
  --internal-network-id "{{user.subnet_id}}" \
  --spec "{{user.nat_spec:1}}"
```

### Operation: Add SNAT Rule

```bash
hcloud nat create-snat-rule \
  --region "{{user.region}}" \
  --nat-gateway-id "{{user.nat_id}}" \
  --floating-ip-id "{{user.eip_id}}" \
  --cidr "{{user.source_cidr}}"
```

### Operation: Add DNAT Rule

```bash
hcloud nat create-dnat-rule \
  --region "{{user.region}}" \
  --nat-gateway-id "{{user.nat_id}}" \
  --floating-ip-id "{{user.eip_id}}" \
  --protocol "{{user.protocol:tcp}}" \
  --internal-service-port "{{user.internal_port:80}}" \
  --external-service-port "{{user.external_port:8080}}"
```

### Operation: Create VPC Peering

#### Pre-flight Checks

- Both VPCs must be in the same region (or use cross-region peering).
- VPC CIDRs must not overlap.
- Both VPCs must be ACTIVE.

#### Execution — CLI

```bash
hcloud vpc create-peering \
  --region "{{user.region}}" \
  --name "{{user.peering_name}}" \
  --vpc-id "{{user.local_vpc_id}}" \
  --peer-vpc-id "{{user.peer_vpc_id}}"
```

#### Post-execution Validation

- Read peering connection ID from response.
- Report peering ID, status, and both VPC details.
- Note: Peering must be accepted by the peer project (if cross-account).

### Operation: Delete VPC

#### Pre-flight (Safety Gate)

- **MUST** verify VPC has no subnets (delete subnets first).
- **MUST** verify no resources depend on VPC.
- **MUST** obtain explicit confirmation.

#### Execution

```bash
hcloud vpc delete \
  --region "{{user.region}}" \
  --vpc-id "{{user.vpc_id}}"
```

## Prerequisites

> Full installation scripts (KooCLI + Go runtime + Credentials): see [references/common-prerequisites.md](../references/common-prerequisites.md)

## Quality Gate (GCL)

This skill is **GCL-required** (per `AGENTS.md` §8). Every VPC / Subnet / Security-Group / EIP /
NAT mutating operation runs through the **Generator-Critic-Loop** before its result is returned.
Read-only `list*` / `get*` / `describe*` are GCL-**exempt**.

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
| 1 | Correctness | ≥ 0.5 (1.0 for `delete-vpc` / `delete-subnet` / `delete-security-group` / `release-eip` / `disassociate-eip`) | `ShowVpc` / `ShowSubnet` / `ShowSecurityGroup` / `ShowEip` / `ShowNat` post-state |
| 2 | Safety | **= 1** (any S-rule hit → ABORT) | S1–S17 in rubric §2 |
| 3 | Idempotency | ≥ 0.5 | Pre-check before create / add |
| 4 | Traceability | ≥ 0.5 | Full request/response; no credential leak |
| 5 | Spec Compliance | ≥ 0.5 | CIDR / SG rule syntax / EIP type / NAT spec |

### Per-Operation Safety Anchors (binding)

- **S1** — `delete-vpc` while subnets / NAT / VPN / peerings still present
- **S2** — `delete-subnet` while instances / ENIs / ELB / NAT / private-IP still use it
- **S3** — `delete-security-group` while it's the **default** SG or referenced by others
- **S4** — `add-security-group-rule` opening all protocols from `0.0.0.0/0`
- **S5** — `add-security-group-rule` opening 22/3389/3306/5432/1433/6379 from `0.0.0.0/0`
- **S6** — `add-security-group-rule` as the **only** egress rule (lock-out)
- **S7** — `delete-security-group-rule` closing the last ingress 22/3389 for prod instance
- **S8 / S9** — `release-eip` while `status == BOUND` or in shared-bandwidth package with others
- **S10** — `disassociate-eip` on prod-named instance without two-step confirmation
- **S11** — `create-vpc` with `cidr` overlapping existing VPC in same region
- **S12** — `create-subnet` with `cidr` not ⊂ parent VPC
- **S13** — `create-nat-gateway` without private subnet or bound EIP
- **S14** — `delete-nat-gateway` with SNAT/DNAT rules or routing dependencies
- **S15** — Any trace contains `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / `sk-…` / `password`
- **S16** — `add-security-group-rule` with protocol ∉ {tcp,udp,icmp,icmpv6,any} or port > 65535
- **S17** — `create-vpc` with `region` / `project_id` not in env contract

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
This skill's `references/prompt-templates.md` keeps VPC-specific overrides and must
not introduce bare `{...}` placeholders.

### See also

- [`references/rubric.md`](references/rubric.md) — full rubric, S1–S17 rules, per-op thresholds
- [`references/prompt-templates.md`](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons
- Repository root [`AGENTS.md`](../AGENTS.md) §3, §5, §7, §8 — GCL specification

## Reference Directory

- [Core Concepts](references/core-concepts.md)
- [API & SDK Usage](references/api-sdk-usage.md)
- [CLI Usage](references/cli-usage.md)
- [Troubleshooting Guide](references/troubleshooting.md)
- [Monitoring & Alerts](references/monitoring.md)
- [Integration](references/integration.md)
- [Knowledge Base](references/knowledge-base.md)
- [AIOps Best Practices](references/advanced/aiops-best-practices.md)
- [FinOps Cost Optimization](references/well-architected-assessment.md#3-finops-)
- [SecOps Security Operations](references/well-architected-assessment.md#4-secops-)
- [Well-Architected Assessment](references/well-architected-assessment.md)
- [GCL Rubric](references/rubric.md) — Adversarial quality gate (v1, 5-dim, S1–S17 VPC/Subnet/SG/EIP/NAT Safety rules)
- [GCL Prompt Templates](references/prompt-templates.md) — Generator / Critic / Orchestrator skeletons

## Well-Architected + Three-Pillar Assessment

This skill's operations are evaluated against Huawei Cloud's Well-Architected Framework five pillars plus FinOps, SecOps, and AIOps integration:
- [Security Assessment](references/well-architected-assessment.md#21-安全支柱-security)
- [Stability Assessment](references/well-architected-assessment.md#22-稳定支柱-stability)
- [Cost Assessment](references/well-architected-assessment.md#23-成本支柱-cost)
- [Efficiency Assessment](references/well-architected-assessment.md#24-效率支柱-efficiency)
- [Performance Assessment](references/well-architected-assessment.md#25-性能支柱-performance)
- [FinOps Integration](references/well-architected-assessment.md#3-finops-)
- [SecOps Integration](references/well-architected-assessment.md#4-secops-)
- [AIOps Integration](references/advanced/aiops-best-practices.md)
