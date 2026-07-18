---
name: huaweicloud-eip-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei
  Cloud Elastic IP (EIP / 弹性公网IP) and bandwidth — public IP allocation,
  bandwidth sizing, billing-model choice, bind/unbind, shared bandwidth, 95th-percentile
  billing, idle EIP detection, DDoS-friendly exposure. User mentions EIP, 弹性公网IP,
  公网IP, 带宽, 共享带宽, 95计费, 按带宽计费, 按流量计费, or describes scenarios
  (e.g., "实例访问不了公网", "释放未绑定的EIP", "EIP被限速", "从ECS解绑公网IP")
  even without naming the product directly.
  Not for VPC / subnet / NAT / security-group management that has dedicated ops skills
  (delegate to huaweicloud-vpc-ops / huaweicloud-nat-ops).
license: MIT
compatibility: >-
  Official Huawei Cloud CLI (`hcloud` / `openstack`), Go 1.21+ runtime
  (for JIT SDK fallback via huaweicloud-sdk-go-v3), valid API credentials,
  network access to Huawei Cloud endpoints.
metadata:
  author: huaweicloud
  version: "1.1.0"
  last_updated: "2026-06-23"
  runtime: Harness AI Agent, Claude Code, Cursor or compatible Agent runtimes
  go_version_minimum: "1.21"
  go_version_jit: "1.24+"
  api_profile: "EIP API v3 / VPC EIP - https://support.huaweicloud.com/api-eip/eip_api_0001.html"
  cli_applicability: "dual-path"
  cli_support_evidence: >-
    EIP product supported by hcloud CLI. Verify with `hcloud eip --help` and
    `hcloud bandwidth --help`; the bandwidth and EIP commands are co-located with
    the VPC product family. JIT Go SDK fallback covers advanced operations
    (shared-bandwidth move, 95th-percentile subscription).
  gcl:
    enabled: true
    required: true
    rubric_version: "v1.3"
    max_iter: 2
    rubric_ref: "references/rubric.md"
    prompts_ref: "references/prompt-templates.md"
    trace_dir: "./audit-results/"
    changelog:
      - version: "1.1.0"
        date: "2026-06-23"
        change: "Round 2 self-reflection fixes: Op 7/8 (add/remove from shared bandwidth); S-rule [S-N] suffixes; gap fixes (100Mbps cap, cross-region verify, share_type check, cooldown_at, ECS state check); EipInUse/EipHasBandwidth HALT; Pattern 5 CES metric step; CLI unverified markers."
      - version: "1.0.0"
        date: "2026-06-23"
        change: "Initial skill release: 6 operations (allocate / describe / bind / unbind / release / bandwidth adjust), FinOps billing-model matrix, SecOps IAM table, AIOps 4 anomaly patterns, GCL rubric + prompt templates."
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud Elastic IP (EIP) Operations Skill

## Overview

Huawei Cloud Elastic IP (EIP / 弹性公网IP) is a **region-scoped, independently billed
public IPv4 address** that can be bound to ECS, ENI, NAT gateway, ELB, or
virtual IP. It is the **default public ingress for nearly every service**, and its
billing model (per-bandwidth / per-traffic / 95th-percentile / shared-bandwidth)
is the single largest FinOps lever outside compute.

This skill is an **operational runbook** for agents: explicit scope, credential rules,
pre-flight checks, **dual-path execution** (official `hcloud` CLI primary + JIT Go SDK
fallback), response validation, and failure recovery. **Do not use the web console as the
primary agent execution path.**

### CLI applicability (repository policy)

- **`cli_applicability: dual-path`:** Official CLI covers EIP / bandwidth lifecycle.
  This skill ships `references/cli-usage.md` and documents **both** the SDK step and the
  CLI step in every execution flow below. Use SDK fallback only when the CLI does not
  expose the operation (e.g., moving an EIP into a shared bandwidth, advanced 95th-percentile
  subscription).

### What This Skill Owns

| In scope | Out of scope (delegate) |
|---|---|
| EIP allocate / describe / bind / unbind / release | VPC / subnet / route table → `huaweicloud-vpc-ops` |
| Bandwidth create / resize / delete | NAT gateway / SNAT / DNAT → `huaweicloud-nat-ops` |
| `add-eip-to-shared` / `remove-eip-from-shared` | DDoS protection policy → `huaweicloud-ddos-ops` (when present) |
| Idle / unbound EIP detection | Security group / EIP exposure → `huaweicloud-vpc-ops` + `huaweicloud-hss-ops` |
| 95th-percentile subscription | CDN / traffic scheduling → `huaweicloud-cdn-ops` (when present) |
| EIP billing-model comparison | Account-level billing → `huaweicloud-billing-ops` |

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use conditions + delegation matrix below |
| 2 | **Structured I/O** | `{{env.*}}` / `{{user.*}}` / `{{output.*}}` placeholder convention |
| 3 | **Explicit Actionable Steps** | Every operation: Pre-flight → Execute → Validate → Recover |
| 4 | **Complete Failure Strategies** | ≥10 EIP error codes; HALT vs retry per error type |
| 5 | **Absolute Single Responsibility** | EIP + bandwidth only; cross-product ops delegate to other skills |
| 6 | **GCL Adversarial Rubric** | `## Quality Gate (GCL)` chapter; `references/rubric.md` with 8 numbered sections; `references/prompt-templates.md` with 7 numbered sections; shared prompt text from `huaweicloud-skill-generator/references/gcl-prompt-backbone.md` |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

| Pillar | Skill Integration | Reference |
|---|---|---|
| **FinOps** | 4 billing-model matrix (按带宽 / 按流量 / 共享带宽 / 95计费), idle EIP cost ledger, right-sizing by 95th percentile | `references/well-architected-assessment.md` §3 |
| **SecOps** | IAM least privilege (viewer/operator/admin), EIP exposure table, high-risk release gates | `references/well-architected-assessment.md` §4 |
| **AIOps** | 4 anomaly patterns (bandwidth-saturation, burst, idle, billing-shock), cross-skill delegation matrix, fault knowledge base | `references/advanced/aiops-patterns.md` |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "Huawei Cloud EIP" / "弹性公网IP" / "公网IP" / "弹性IP" / "带宽" / "共享带宽"
- Task keywords: 申请公网IP, 绑定EIP, 解绑EIP, 释放EIP, 调整带宽, 共享带宽, 95计费, 95th, 带宽包
- User asks to: deploy, configure, troubleshoot, or monitor EIP / bandwidth **via API, SDK, CLI, or automation**
- Anomaly reported: "EIP 500错误", "公网访问丢包", "EIP 限速", "带宽打满", "未绑定EIP持续扣费"

### SHOULD NOT Use This Skill When

- VPC / subnet / route table / security group only → `huaweicloud-vpc-ops`
- NAT gateway / SNAT / DNAT → `huaweicloud-nat-ops`
- DDoS attack handling → `huaweicloud-ddos-ops` (when present), else `huaweicloud-hss-ops`
- Pure billing reconciliation / 包年包月 invoice → `huaweicloud-billing-ops`
- ECS-level public IP lifecycle on a *dehoused* instance (use ECS lifecycle, not EIP)

### Delegation Rules

- If user needs an EIP **and** a security group, complete the SG first via `huaweicloud-vpc-ops`, then allocate / bind via this skill.
- Multi-product requests: handle each product with its own skill; never merge VPC + EIP into one ambiguous flow.
- For FinOps questions involving EIP cost: use this skill's `references/well-architected-assessment.md` §3, then escalate to `huaweicloud-billing-ops` for cross-resource cost.
- For SecOps questions about EIP exposure: use this skill's security table, then escalate to `huaweicloud-hss-ops` for threat detection.

## Variable Convention (Agent-Readable)

| Placeholder | Meaning | Agent Action |
|---|---|---|
| `{{env.HW_ACCESS_KEY_ID}}` | AK from runtime env | NEVER ask the user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | SK from runtime env | NEVER ask the user; fail if unset |
| `{{env.HW_REGION_ID}}` | Region | Use documented default only if user agrees |
| `{{env.HW_PROJECT_ID}}` | Project ID | Use documented default only if user agrees |
| `{{user.region}}` | Cross-region EIP ops | Ask once; reuse |
| `{{user.bandwidth_size}}` | Mbps value | Ask once; validate against quota |
| `{{user.billing_mode}}` | `bandwidth` / `traffic` / `95` / `shared` | Ask once; reuse; **see FinOps matrix** |
| `{{user.resource_id}}` | EIP / bandwidth ID | Ask once; reuse |
| `{{output.public_ip}}` | Allocated EIP address string | Parse from `publicip_address` |
| `{{output.eip_id}}` | EIP resource ID | Parse from `id` / `eip_id` |
| `{{output.bandwidth_id}}` | Bandwidth resource ID | Parse from `bandwidth.id` |

> **`{{env.*}}` MUST NOT** be collected from the user. **`{{user.*}}`** MUST be collected
> interactively when missing, with the FinOps matrix shown alongside the prompt.

> **Security Warning (Credential Masking — MANDATORY):** **NEVER** log, print, or expose
> `HW_SECRET_ACCESS_KEY`, `SecretAccessKey`, or any credential field value in console
> output, debug messages, error messages, or GCL traces.

## API and Response Conventions

- **OpenAPI is canonical** for path, query, body fields, enums, and response shapes.
  See [EIP API v3](https://support.huaweicloud.com/api-eip/eip_api_0001.html).
- **Errors:** Map SDK/HTTP errors to `error_code` / `error_msg` fields.
- **Timestamps:** ISO 8601 with timezone when the API returns strings.
- **Idempotency:**
  - `allocate-eip` is **NOT** idempotent on retry — a duplicate call may bill two EIPs.
    Always check existing list first.
  - `release-eip` is **NOT** idempotent on retry — a duplicate call on a released EIP
    returns `ResourceNotFound` and is benign.
  - `bind-eip` / `unbind-eip` accept a `client_token` for safe retry.
  - Bandwidth resize is **idempotent** when the new size equals the current size.

## Quick Start

### What This Skill Does
Enables deployment, configuration, troubleshooting, and monitoring of Huawei Cloud
EIP and bandwidth resources using `hcloud` CLI (primary) or JIT Go SDK (fallback).

### Prerequisites
- [ ] Huawei Cloud CLI installed (or Go runtime for JIT fallback)
- [ ] Credentials configured: `HW_ACCESS_KEY_ID`, `HW_SECRET_ACCESS_KEY`
- [ ] Region and Project ID set: `HW_REGION_ID`, `HW_PROJECT_ID`

### Verify Setup
```bash
hcloud --version
hcloud eip list --region {{env.HW_REGION_ID}}
```

### Your First Command
```bash
# List current EIPs and their binding state
hcloud eip list --region {{env.HW_REGION_ID}} \
  --output json | jq '.publicips[] | {id, public_ip_address, status, bandwidth_size}'
```

### Next Steps
- [Core Concepts](references/core-concepts.md) — EIP model, billing modes, bandwidth types
- [Common Operations](#execution-flows) — Allocate, bind, resize, release
- [Troubleshooting](references/troubleshooting.md) — Top EIP failure patterns

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|---|---|---|---|
| Allocate EIP | Create EIP in target region, pick billing mode | Medium | Low (idempotency caveat) |
| Describe EIP | Get EIP + binding + bandwidth snapshot | Low | None |
| Bind EIP | Attach EIP to ECS / ENI / NAT / ELB / VIP | Medium | Medium |
| Unbind EIP | Detach EIP from current resource | Medium | Medium |
| Release EIP | Permanently delete EIP (irreversible) | Low | **High** — irreversible billing stops |
| Resize Bandwidth | Increase / decrease Mbps on EIP or shared bandwidth | Medium | Medium |
| Add EIP to Shared Bandwidth | Move a PER EIP into a WHOLE shared bandwidth pool | Medium | Medium |
| Remove EIP from Shared Bandwidth | Move an EIP out of shared bandwidth back to PER | Medium | Medium |

## Execution Flows

### Operation 1: Allocate EIP

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|---|---|---|---|
| CLI / deps | `hcloud --version` | Exit code 0 | Document CLI install |
| Credentials | Construct credential from env | Non-empty AK/SK | HALT; user configures env |
| Region | `hcloud eip list --region {{env.HW_REGION_ID}}` | 2xx response | Suggest valid region |
| Quota | `ShowCountQuota` SDK (or `hcloud eip describe-quota` if verified) | Sufficient quota | HALT; user raises quota |
| Billing mode | Ask user with FinOps matrix | User picks `bandwidth` / `traffic` / `shared` | Default to `bandwidth` only with explicit consent |
| Bandwidth-size cap | If `billing-mode=traffic`: reject if `bandwidth-size > 100` [S5] | ≤ 100 Mbps | HALT; suggest `bandwidth` mode or split across EIPs |

#### Execution — CLI (Primary Path)

```bash
# 按带宽计费 (default for stable traffic, predictable cost)
hcloud eip create \
  --region "{{user.region}}" \
  --name "{{user.eip_name}}" \
  --type "5_bgp" \
  --billing-mode "bandwidth" \
  --bandwidth-size "{{user.bandwidth_size}}" \
  --charge-type "postpaid"

# 按流量计费 (spiky traffic, pay-by-byte)
# WARNING: 按流量 bandwidth-size hard cap = 100 Mbps [S5] — pre-flight enforces this.
hcloud eip create \
  --region "{{user.region}}" \
  --name "{{user.eip_name}}" \
  --type "5_bgp" \
  --billing-mode "traffic" \
  --bandwidth-size "{{user.bandwidth_size}}"  # max 100 Mbps
```

#### Execution — JIT Go SDK (Fallback Path)

Use when the user needs operations CLI does not surface (e.g., move EIP into a shared
bandwidth pool, 95th-percentile subscription).

```go
//go:build ignore
// run: go run eip_allocate.go
package main

import (
	"fmt"
	"os"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/eip/v2/model"
)

func main() {
	ak, sk, region := os.Getenv("HW_ACCESS_KEY_ID"), os.Getenv("HW_SECRET_ACCESS_KEY"), os.Getenv("HW_REGION_ID")
	if ak == "" || sk == "" || region == "" {
		// Names only; never print env values.
		fmt.Fprintln(os.Stderr, "missing required env: HW_ACCESS_KEY_ID / HW_SECRET_ACCESS_KEY / HW_REGION_ID")
		os.Exit(2)
	}
	cfg := config.DefaultHttpConfig()
	client := eip.EipClientBuilder().
		WithEndpoint(fmt.Sprintf("eip.%s.myhuaweicloud.com", region)).
		WithCredential(basic.NewCredentialsBuilder().WithAk(ak).WithSk(sk).Build()).
		WithHttpConfig(cfg).Build()

	// 按带宽计费；type=5_bgp 表示全动态 BGP
	req := &model.CreatePublicipRequest{
		Body: &model.CreatePublicipRequestBody{
			Publicip: &model.CreatePublicipOption{
				Type: "5_bgp",
			},
			Bandwidth: &model.CreatePublicipBandwidthOption{
				Name:       "eip-bw",
				Size:       5, // Mbps
				ShareType:  "PER", // PER=独占 / WHOLE=共享带宽
				ChargeMode: "bandwidth",
			},
		},
	}
	resp, err := client.CreatePublicip(req)
	if err != nil {
		// Print error only; never log the request body (may contain PII or sensitive ids).
		fmt.Fprintln(os.Stderr, "CreatePublicip failed:", err)
		os.Exit(1)
	}
	fmt.Printf("eip_id=%s public_ip=%s\n", *resp.Publicip.Id, *resp.Publicip.PublicIpAddress)
}
```

#### Post-execution Validation
1. Read `{{output.eip_id}}` and `{{output.public_ip}}` from `publicip.id` / `publicip.public_ip_address`.
2. Poll `hcloud eip describe --eip-id {{output.eip_id}}` until `status` = `DOWN` (unbound) or `ACTIVE` (bound).
3. On success, report both IDs; on failure, go to **Failure Recovery**.

#### Failure Recovery

| Error | Max retries | Backoff | Agent Action | UX Feedback |
|---|---|---|---|---|
| `InvalidParameter` | 0–1 | — | Fix args from OpenAPI | `[ERROR] InvalidParameter: <msg> — Check parameters against EIP API docs.` |
| `QuotaExceeded` | 0 | — | HALT | `[ERROR] Quota exceeded. Apply for EIP quota raise in Console → Service Quota.` |
| `InsufficientBalance` | 0 | — | HALT | `[ERROR] Insufficient balance. Recharge your Huawei Cloud account.` |
| `EipAllocateFailed` | 0 | — | HALT | `[ERROR] EipAllocateFailed: <msg> — Region may be sold-out; try adjacent region.` |
| Throttling / 429 | 3 | exponential | Back off; respect Retry-After | `[WARN] Rate limited. Retrying after computed backoff seconds...` |
| `InternalError` / 5xx | 3 | 2s, 4s, 8s | Retry; then HALT | `[ERROR] Server-side error. Retry or escalate with RequestId.` |

> **Idempotency caveat:** If a network blip occurs after a successful allocate, the retry
> may produce a second EIP. Always `list` and dedupe by `public_ip_address` before retrying.

### Operation 2: Describe EIP

#### Execution

```bash
# CLI — list all EIPs in region
hcloud eip list --region {{env.HW_REGION_ID}} --output json

# CLI — single EIP
hcloud eip describe --region {{env.HW_REGION_ID}} --eip-id "{{user.eip_id}}"
```

#### Output fields (agent-parse contract)

| Field | JSON path | Meaning |
|---|---|---|
| EIP ID | `publicip.id` | `{{output.eip_id}}` |
| Public IP | `publicip.public_ip_address` | `{{output.public_ip}}` |
| Status | `publicip.status` | `ACTIVE` / `DOWN` / `ERROR` |
| Type | `publicip.type` | `5_bgp` / `5_sbgp` / `5_dualStack` |
| Bound resource | `publicip.port_id` | null = unbound |
| Bandwidth ID | `bandwidth.id` | `{{output.bandwidth_id}}` |
| Bandwidth size | `bandwidth.size` | Mbps |
| Charge mode | `bandwidth.charge_mode` | `bandwidth` / `traffic` |

### Operation 3: Bind EIP

#### Pre-flight (Safety Gate)

- Target resource must be in the **same region** as the EIP [S8].
  → Verify: query `hcloud eip describe --eip-id {{user.eip_id}}` for EIP region, then query target resource (ECS / ENI) for its region — both must match.
- EIP must be in `DOWN` (unbound) state [S13].
- For ECS: target ECS must be `RUNNING` [S13].
  → Verify: `hcloud ecs describe --server-id {{user.ecs_id}} --region {{user.region}}` — `status` must be `RUNNING`. If not, HALT; do not bind to a stopped instance.
- For ENI: target ENI must be `ATTACHED` to a running ECS.

#### Execution

```bash
# CLI — bind to ECS
hcloud eip bind \
  --region "{{user.region}}" \
  --eip-id "{{user.eip_id}}" \
  --port-id "{{user.port_id}}"
```

#### Post-execution Validation
Poll `describe` until `status` = `ACTIVE` and `port_id` matches target. Common SLA: 5–30s.

### Operation 4: Unbind EIP

#### Pre-flight (Safety Gate)

- **MUST** obtain explicit confirmation if the resource name matches
  `(?i)(prod|prd|production|online|pay)` — production blast radius [S4].
- Recommend: unbind during low-traffic window; expect 1–3 connection drops during cutover [S4].

#### Execution

```bash
hcloud eip unbind --region "{{user.region}}" --eip-id "{{user.eip_id}}"
```

#### Post-execution Validation
Poll `describe` until `status` = `DOWN` and `port_id` = null.

### Operation 5: Release EIP

#### Pre-flight (Safety Gate — IRREVERSIBLE)

- **MUST** require explicit confirmation: `release-eip` of `{{user.eip_id}}` (`{{user.public_ip}}`) is permanent [S3].
- **MUST** verify EIP is **unbound** (`port_id` = null) before proceeding; release of a bound EIP orphans the bill [S11].
- **MUST** warn: any DNS A record pointing at this public IP will become unreachable [S16].
- **MUST** verify `bandwidth.share_type`: if EIP is `PER` proceed; if `WHOLE` (in a shared bandwidth pool), first call `huaweicloud-eip-ops` Op 8 to remove from pool [S1/S2].
  → Query `hcloud eip describe --eip-id {{user.eip_id}}`; if `bandwidth.share_type == "WHOLE"`, abort release and delegate to Op 8.

#### Execution

```bash
hcloud eip delete --region "{{user.region}}" --eip-id "{{user.eip_id}}"
```

#### Post-execution Validation
Poll `describe` until 404 / `ResourceNotFound` within max wait (default 30s).

### Operation 6: Resize Bandwidth

#### Pre-flight

- EIP / shared bandwidth must be `RUNNING`.
- New size must be within quota (see `ShowCountQuota` via Go SDK).
- If the bandwidth uses **95th-percentile billing**: query `hcloud bandwidth describe --bandwidth-id {{user.bandwidth_id}}` for `cooldown_at`. If `cooldown_at` is in the future, **do not retry** inside the cooldown window [S7] — wait until cooldown expires or schedule the resize after it.

#### Execution

```bash
# CLI — resize EIP bandwidth
hcloud eip update-bandwidth \
  --region "{{user.region}}" \
  --eip-id "{{user.eip_id}}" \
  --bandwidth-size "{{user.bandwidth_size}}"
```

#### Post-execution Validation
Poll `describe` until `bandwidth.size` matches target. Note: 95th-percentile subscriptions
have a **cooldown window** after each change; see `references/well-architected-assessment.md` §3.

### Operation 7: Add EIP to Shared Bandwidth

#### Pre-flight (Safety Gate)
- EIP must currently be `PER` (not already in a WHOLE pool) — verify `bandwidth.share_type == "PER"`.
- Target WHOLE bandwidth pool must exist in the same region.
- EIP must be `DOWN` (unbound) before moving into a shared pool [S9].

#### Execution — CLI
```bash
# CLI subcommand [unverified] — fall back to Go SDK if unknown-subcommand error
hcloud bandwidth add-eip \
  --region "{{user.region}}" \
  --bandwidth-id "{{user.bandwidth_id}}" \
  --eip-id "{{user.eip_id}}"
```
If CLI returns unknown subcommand error, fall back to Go SDK:
```go
// Move EIP into WHOLE bandwidth pool
req := &model.UpdatePublicipRequest{
    PublicipId: eipID,
    Body: &model.UpdatePublicipRequestBody{
        BandwidthId: &bandwidthID,
    },
}
_, err := client.UpdatePublicip(req)
```

#### Post-execution Validation
Poll `hcloud bandwidth describe --bandwidth-id {{user.bandwidth_id}}` until EIP id appears in `publicip_id` list.

#### Failure Recovery
| Error | Agent Action |
|---|---|
| `EipInSharedBandwidth` | EIP already in a WHOLE pool — move it out first |
| `BandwidthTypeError` | Target bandwidth is PER, not WHOLE |

### Operation 8: Remove EIP from Shared Bandwidth

#### Pre-flight (Safety Gate)
- EIP must currently be in a WHOLE shared bandwidth pool [S9].
- EIP must be `DOWN` (unbound) before removal.

#### Execution — CLI
```bash
# CLI subcommand [unverified] — fall back to Go SDK if unknown-subcommand error
hcloud bandwidth remove-eip \
  --region "{{user.region}}" \
  --bandwidth-id "{{user.bandwidth_id}}" \
  --eip-id "{{user.eip_id}}"
```
If CLI returns unknown subcommand error, fall back to Go SDK:
```go
// Move EIP back to PER (detach from WHOLE bandwidth)
// NOTE: Setting BandwidthId to nil may leave the EIP with no bandwidth attached.
// After this call, the EIP's billing reverts to PER — ensure a default bandwidth
// size is set or the EIP will be unusable.
// TODO (verify): Confirm nil behavior against UpdatePublicip API docs.
req := &model.UpdatePublicipRequest{
    PublicipId: eipID,
    Body: &model.UpdatePublicipRequestBody{
        BandwidthId: nil,
    },
}
_, err := client.UpdatePublicip(req)
```

#### Post-execution Validation
Poll `hcloud bandwidth describe --bandwidth-id {{user.bandwidth_id}}` — EIP id must no longer appear in `publicip_id` list.

#### Failure Recovery
| Error | Agent Action |
|---|---|
| `EipNotInBandwidth` | EIP not in this pool — verify pool id |

## FinOps at a Glance (Details in §3 of well-architected-assessment)

| Billing Mode | Best For | Cost Shape | Risk |
|---|---|---|---|
| `bandwidth` (按带宽) | Stable traffic, predictable cost | Linear: `Mbps × hours × unit_price` | Over-pays during idle hours |
| `traffic` (按流量) | Spiky traffic, mostly idle | `bytes × unit_price` | Surprise bill on burst; **hard cap 100 Mbps** — see Op 1 pre-flight [S5] |
| `shared` (共享带宽) | ≥2 EIPs with complementary patterns | Priced by sum-of-peaks, not sum-of-Mbps | Move-in/out complexity |
| `95` (95th percentile) | Large egress, agreed baseline | Monthly 5-min samples, top 5% discarded | Cooldown after change |

**Default recommendation:**
- Single EIP + 24×7 production load → `bandwidth`
- Single EIP + dev/test, mostly idle → `traffic`
- ≥3 EIPs in same region with disjoint traffic hours → `shared`
- Wholesale / ISP-like traffic shape → `95` (talk to account team first)

## SecOps at a Glance (Details in §4 of well-architected-assessment)

| Role | Required Permissions |
|---|---|
| EIP Viewer | `vpc:eip:list`, `vpc:eip:get` |
| EIP Operator | Viewer + `vpc:eip:create`, `vpc:eip:update`, `vpc:eip:delete` |
| EIP Admin | Operator + `vpc:bandwidth:*`, `vpc:eip:bind`, `vpc:eip:unbind` on prod |

> **High-risk release:** `release-eip` for an EIP in `WHOLE` shared-bandwidth mode may
> leave the shared bandwidth **partially empty**; admin role required. See
> `references/well-architected-assessment.md` §4 S-rules.

## AIOps at a Glance (Details in references/advanced/aiops-patterns.md)

| Pattern | Detection Signal | Cross-skill delegation |
|---|---|---|
| Bandwidth saturation | `outgoing_bytes / bandwidth_size > 0.9` for 5 min | → `huaweicloud-ces-ops` for threshold, `huaweicloud-billing-ops` for overage |
| Burst / DDoS shape | Egress p99 > 10× p50 for 10 min | → `huaweicloud-ddos-ops` (when present), `huaweicloud-hss-ops` |
| Idle EIP | `port_id == null` for 7 d AND bandwidth paid | → `huaweicloud-billing-ops` for cost attribution |
| Billing shock | 24h cost > 3× 7-day median | → `huaweicloud-billing-ops` for invoice audit |

## Quality Gate (GCL)

This skill uses Generator-Critic-Loop runtime validation for cloud operations.
Generated skills MUST ship the following artifacts:

- `references/rubric.md` — 8 numbered sections: scope, thresholds, evidence, product safety rules, scoring guide, examples, escalation, changelog.
- `references/prompt-templates.md` — 7 numbered sections: Generator, Critic, Orchestrator, product pre-flight overrides, product-only anti-patterns, changelog, see also.
- `SKILL.md` metadata `gcl` block — `required: true`, `default_max_iter: 2`, `rubric_version: "v1"`, `trace_path: "./audit-results/gcl-trace-{{timestamp}}.json"`.

### Runtime Roles

| Role | Responsibility | Constraint |
|---|---|---|
| Generator | Execute one requested EIP / bandwidth op via `hcloud` or Go SDK fallback | Capture masked trace; never self-score |
| Critic | Score Generator output against `references/rubric.md` | Read-only; never see raw `{{user.request}}`; no cloud mutations |
| Orchestrator | Decide PASS / RETRY / SAFETY_FAIL / MAX_ITER | Persist trace under `audit-results/`; safety=0 aborts |

### Default Rubric Thresholds

| Dimension | Threshold | Notes |
|---|---:|---|
| correctness | ≥ 0.5 | 1.0 for `release-eip` / `unbind` (irreversible or blast-radius) |
| safety | = 1.0 | Any S-rule hit or credential leak => SAFETY_FAIL |
| idempotency | ≥ 0.5 | `allocate-eip` MUST dedupe by `public_ip_address` |
| traceability | ≥ 0.5 | Command, args, response excerpt, errors, request_id captured |
| spec_compliance | ≥ 0.5 | Uses OpenAPI-verified fields; no invented flags |

### Trace Requirements

1. Persist `audit-results/gcl-trace-{{timestamp}}.json` (format: YYYYMMDD-HHMMSS) for PASS, MAX_ITER, and SAFETY_FAIL.
2. Mask `HW_SECRET_ACCESS_KEY`, AK/SK values, tokens, passwords, and authorization headers.
3. Include sanitized `operation_intent` so the Critic can assess expected state without seeing raw user wording.
4. Use root scripts: `scripts/gcl_runner.py`, `scripts/gcl_trace_aggregate.py`, `scripts/check_gcl_conformance.py`.

### Prompt Backbone

Use `huaweicloud-skill-generator/references/gcl-prompt-backbone.md` as the shared source
for Generator/Critic/Orchestrator wording. This skill's `references/prompt-templates.md`
keeps product-specific overrides and must not introduce bare `{...}` placeholders.

## Reference Directory

- [Core Concepts](references/core-concepts.md) — EIP model, billing modes, bandwidth types
- [API & SDK Usage](references/api-sdk-usage.md) — Go SDK JIT patterns
- [CLI Usage](references/cli-usage.md) — `hcloud eip` / `hcloud bandwidth` reference
- [Troubleshooting Guide](references/troubleshooting.md) — Top EIP failure patterns
- [Monitoring & Alerts](references/monitoring.md) — Bandwidth / egress / idle metrics
- [Integration](references/integration.md) — Cross-skill delegation matrix
- [Knowledge Base](references/knowledge-base.md) — EIP fault patterns
- [Idempotency Checklist](references/idempotency-checklist.md) — Safe-retry contract per op
- [AIOps Best Practices](references/advanced/aiops-patterns.md) — 4 anomaly patterns
- [FinOps + SecOps + Well-Architected](references/well-architected-assessment.md)
- [GCL Rubric](references/rubric.md)
- [GCL Prompt Templates](references/prompt-templates.md)

> 任务完成后按根 AGENTS.md 的「复利资产沉淀机制 (CADL)」复盘并沉淀可复用资产。
