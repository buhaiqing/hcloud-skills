---
name: huaweicloud-dns-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei Cloud DNS
  (云解析DNS / Cloud DNS) — zone management, DNS record lifecycle, DNSSEC,
  health checks, and traffic routing. User mentions DNS, 云解析, 域名解析,
  A记录, CNAME, TXT记录, 智能DNS, or describes scenarios
  (e.g., "配置域名的DNS解析", "添加A记录指向ECS", "启用DNSSEC", "DNS故障切换")
  even without naming the product directly.
  Not for CDN domain CNAME setup (→ huaweicloud-cdn-ops),
  EIP binding (→ huaweicloud-eip-ops), or domain registration (→ console).
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
  api_profile: "DNS API v3 / https://support.huaweicloud.com/api-dns/"
  cli_applicability: "cli-first"
  cli_support_evidence: >-
    KooCLI (`hcloud dns`) supports DNS with subcommands: list-zones,
    create-zone, delete-zone, list-recordsets, create-recordset, update-recordset,
    delete-recordset, show-quota. Verify with `hcloud dns --help`.
  gcl:
    required: true
    default_max_iter: 2
    rubric_version: "v1"
    trace_path: "./audit-results/gcl-trace-{{timestamp}}.json"
  environment:
    - HW_ACCESS_KEY_ID
    - HW_SECRET_ACCESS_KEY
    - HW_REGION_ID
    - HW_PROJECT_ID
---

> This skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

# Huawei Cloud DNS Operations Skill

## Overview

Huawei Cloud DNS (云解析DNS) provides authoritative DNS services for domain names.
DNS zones are containers for all DNS records of a domain. DNS records (记录集) map
resource names to IP addresses or other data (A, AAAA, CNAME, MX, TXT, etc.).

This skill is an **operational runbook** for agents: explicit scope, credential rules,
pre-flight checks, CLI-first execution with JIT Go SDK fallback, response validation,
and failure recovery. **Do not use the web console as the primary agent execution path.**

### What This Skill Owns

| In scope | Out of scope (delegate) |
|---|---|
| Public zone + record CRUD | Domain registration (→ console / registrar) |
| Private zone (VPC-scoped) | DNS failover with health check (→ VPC / ELB) |
| DNSSEC configuration | CDN CNAME setup → `huaweicloud-cdn-ops` |
| PTR reverse DNS | EIP binding → `huaweicloud-eip-ops` |
| TTL management | TTL + CDN interaction → `huaweicloud-cdn-ops` |

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use + delegation matrix |
| 2 | **Structured I/O** | `{{env.*}}` / `{{user.*}}` / `{{output.*}}` convention |
| 3 | **Explicit Actionable Steps** | Pre-flight → Execute → Validate → Recover |
| 4 | **Complete Failure Strategies** | ≥10 DNS error codes; HALT vs retry per type |
| 5 | **Absolute Single Responsibility** | DNS zones + records only; cross-product delegate |
| 6 | **GCL Adversarial Rubric** | `## Quality Gate (GCL)`; `references/rubric.md`; `references/prompt-templates.md` |

### Three-Pillar Ops Integration

| Pillar | Skill Integration | Reference |
|---|---|---|
| **FinOps** | Zone count vs quota, TTL vs CDN cost, idle zone detection | `references/well-architected-assessment.md` §3 |
| **SecOps** | DNSSEC, IAM minimum permissions, ACL policies | `references/well-architected-assessment.md` §4 |
| **AIOps** | NXDOMAIN spike, resolution latency, TTL storm, delegation chain failure | `references/advanced/aiops-patterns.md` |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "DNS" / "云解析" / "A记录" / "CNAME" / "TXT记录" / "智能DNS" / "DNSSEC"
- Task keywords: 添加DNS解析, 修改A记录, 删除域名解析, 配置DNS, 解析生效
- Anomaly reported: "DNS解析失败", "域名无法访问", "TTL设置", "DNSSEC"

### SHOULD NOT Use This Skill When

- CDN CNAME configuration → `huaweicloud-cdn-ops`
- EIP binding / PTR → `huaweicloud-eip-ops`
- Domain registration → console / registrar (not an API operation)

## Variable Convention (Agent-Readable)

| Placeholder | Meaning | Agent Action |
|---|---|---|
| `{{env.HW_ACCESS_KEY_ID}}` | AK from runtime env | NEVER ask user |
| `{{env.HW_SECRET_ACCESS_KEY}}` | SK from runtime env | NEVER ask user |
| `{{user.zone_id}}` | Zone resource ID | Ask once; reuse |
| `{{user.zone_name}}` | Domain name (e.g., `example.com.`) | Ask once; verify trailing dot |
| `{{user.recordset_id}}` | Record set ID | Ask once; reuse |
| `{{user.record_type}}` | DNS record type | `A` / `AAAA` / `CNAME` / `MX` / `TXT` |
| `{{user.record_value}}` | Record value (IP / hostname) | Validate format |
| `{{output.zone_id}}` | Zone ID from create-list response | Parse from `zone.id` |
| `{{output.recordset_id}}` | Record set ID | Parse from `recordset.id` |

> **`{{env.*}}` MUST NOT** be collected from the user. **`{{user.*}}`** MUST be collected interactively.

> **Security Warning (Credential Masking — MANDATORY):** **NEVER** log or expose
> `HW_SECRET_ACCESS_KEY` in any output, error message, or GCL trace.

## Quick Start

### Prerequisites
- [ ] Huawei Cloud CLI installed (`hcloud dns` available)
- [ ] Credentials: `HW_ACCESS_KEY_ID`, `HW_SECRET_ACCESS_KEY`

### Verify Setup
```bash
hcloud dns list-zones --region {{env.HW_REGION_ID}} --output json
```

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|---|---|---|---|
| List zones | List all DNS zones | Low | None |
| Create zone | Add a new DNS zone (public or private) | Low | Low |
| Configure DNSSEC | Enable DNSSEC signing | Medium | Medium |
| Add record set | Create A / CNAME / MX / TXT / AAAA record | Low | Low |
| Modify record set | Update record value / TTL | Low | Medium |
| Delete record set | Remove DNS record (irreversible propagation delay) | Low | Medium |
| Delete zone | Remove zone and all records (irreversible) | Low | **High** |
| PTR reverse DNS | Configure reverse DNS for EIP | Low | Medium |

## Execution Flows

### Operation 1: List Zones

#### Execution — CLI

```bash
hcloud dns list-zones --region {{env.HW_REGION_ID}} --output json \
  | jq '.zones[] | {id, name, zone_type, status}'
```

### Operation 2: Create Zone

#### Pre-flight
- Verify domain name ownership (user confirms).
- Check quota: `hcloud dns show-quota`.

#### Execution — CLI

```bash
hcloud dns create-zone \
  --region "{{user.region}}" \
  --name "{{user.zone_name}}" \
  --zone-type "public"   # or "private" with --vpc-id
```

### Operation 3: Add Record Set

#### Execution — CLI

```bash
# A record
hcloud dns create-recordset \
  --zone-id "{{user.zone_id}}" \
  --name "www.{{user.zone_name}}" \
  --type "A" \
  --records "1.2.3.4" \
  --ttl 300

# CNAME record
hcloud dns create-recordset \
  --zone-id "{{user.zone_id}}" \
  --name "cdn.{{user.zone_name}}" \
  --type "CNAME" \
  --records "example.com.cdn.cn-north-4.myhwcdn.com." \
  --ttl 300
```

### Operation 4: Modify Record Set

#### Execution — CLI

```bash
hcloud dns update-recordset \
  --zone-id "{{user.zone_id}}" \
  --recordset-id "{{user.recordset_id}}" \
  --records "5.6.7.8" \
  --ttl 600
```

### Operation 5: Delete Record Set

#### Pre-flight (Safety Gate)
- DNS deletion propagates globally (TTL determines delay, typically 5–60 min).
- Warn: deleted records may still resolve for up to TTL duration.

#### Execution — CLI

```bash
hcloud dns delete-recordset \
  --zone-id "{{user.zone_id}}" \
  --recordset-id "{{user.recordset_id}}"
```

### Operation 6: Delete Zone

#### Pre-flight (Safety Gate — IRREVERSIBLE)

- **MUST** require explicit confirmation: deleting zone `{{user.zone_name}}` removes ALL records.
- **MUST** warn: remaining TTL propagation delay after deletion.

#### Execution — CLI

```bash
hcloud dns delete-zone \
  --zone-id "{{user.zone_id}}"
```

## FinOps at a Glance

| Metric | Action |
|---|---|
| Zone count vs quota | Monitor via `show-quota`; delete unused zones |
| TTL vs CDN cost | Low TTL = more origin DNS lookups = higher CDN CNAME resolution cost |
| Idle private zone | VPC deleted but zone remains → orphan billing |

## SecOps at a Glance

| Rule | Enforcement |
|---|---|
| DNSSEC | Enable for all public zones; protect against spoofing |
| IAM | Use least-privilege: `DNS FullControl` for operators; `DNS ReadOnly` for viewers |
| Zone ACL | Private zones with VPC-scoped IAM policies |

## AIOps at a Glance (Details in references/advanced/aiops-patterns.md)

| Pattern | Detection Signal | Cross-skill |
|---|---|---|
| NXDOMAIN spike | Query success rate drop | → `huaweicloud-ces-ops` |
| Resolution latency | p95 > 100ms for extended period | → `huaweicloud-ces-ops` |
| Delegation chain failure | NS record mismatch | Delegate to DNS registrar |
| TTL storm | >1000 queries/s for same name | Rate limit at edge |

## Quality Gate (GCL)

This skill uses Generator-Critic-Loop runtime validation. Required artifacts:

- `references/rubric.md` — 8 sections: scope, thresholds, safety rules, scoring guide, examples, escalation, changelog.
- `references/prompt-templates.md` — 7 sections: Generator, Critic, Orchestrator, pre-flight, changelog.
- `SKILL.md` metadata `gcl` block: `required: true`, `default_max_iter: 2`, `rubric_version: "v1"`.

### Default Rubric Thresholds

| Dimension | Threshold | Notes |
|---|---:|---|
| correctness | ≥ 0.5 | 1.0 for delete-zone |
| safety | = 1.0 | Any S-rule hit → SAFETY_FAIL |
| idempotency | ≥ 0.5 | Record type + name + value dedupe |
| traceability | ≥ 0.5 | Command, args, request_id captured |
| spec_compliance | ≥ 0.5 | CLI flags verified |

## Reference Directory

- [Core Concepts](references/core-concepts.md) — DNS model, record types, TTL, zone types
- [API & SDK Usage](references/api-sdk-usage.md) — Go SDK JIT patterns
- [CLI Usage](references/cli-usage.md) — `hcloud dns` command reference
- [Troubleshooting Guide](references/troubleshooting.md) — DNS failure patterns
- [Monitoring & Alerts](references/monitoring.md) — Resolution latency, NXDOMAIN metrics
- [Integration](references/integration.md) — Cross-skill delegation
- [Well-Architected Assessment](references/well-architected-assessment.md)
- [AIOps Patterns](references/advanced/aiops-patterns.md)
- [GCL Rubric](references/rubric.md)
- [GCL Prompt Templates](references/prompt-templates.md)

> 任务完成后按根 AGENTS.md 的「复利资产沉淀机制 (CADL)」复盘并沉淀可复用资产。
