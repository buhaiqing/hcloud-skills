---
name: huaweicloud-cdn-ops
description: >-
  Use when the user needs to deploy, configure, troubleshoot, or monitor Huawei Cloud CDN
  (Content Delivery Network / 内容分发网络) — domain lifecycle, cache rules, HTTPS certificates,
  refresh/preheat, and traffic statistics. User mentions CDN, 内容分发, 加速域名,
  缓存刷新, 预热, 命中率, 带宽峰值, or describes scenarios
  (e.g., "加速域名接入CDN", "刷新CDN缓存", "CDN命中率低", "CDN带宽计费")
  even without naming the product directly.
  Not for EIP / bandwidth management (→ huaweicloud-eip-ops),
  HTTPS certificate provisioning (→ huaweicloud-waf-ops),
  or static website hosting (→ huaweicloud-obs-ops).
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
  api_profile: "CDN API v1 / https://support.huaweicloud.com/api-cdn/cdn_api_0002.html"
  cli_applicability: "cli-first"
  cli_support_evidence: >-
    KooCLI (`hcloud cdn`) supports CDN with subcommands: list-domain,
    create-domain, delete-domain, modify-domain, start-domain, stop-domain,
    refresh-cache, preheat-cache, list-stats, modify-domain-config.
    Verify with `hcloud cdn --help`. JIT Go SDK covers advanced operations
    (batch refresh, detailed statistics, origin configurations).
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

# Huawei Cloud CDN Operations Skill

## Overview

Huawei Cloud CDN (Content Delivery Network / 内容分发网络) accelerates static and dynamic
web content delivery via globally distributed edge nodes. CDN domains (加速域名) are the
primary resource: each domain represents a content source that CDN edge nodes cache and
serve to end users.

This skill is an **operational runbook** for agents: explicit scope, credential rules,
pre-flight checks, CLI-first execution with JIT Go SDK fallback, response validation,
and failure recovery. **Do not use the web console as the primary agent execution path.**

### CLI applicability (repository policy)

- **`cli_applicability: cli-first`:** `hcloud cdn` CLI fully covers CDN domain lifecycle.
  JIT Go SDK is the **fallback** for batch operations, detailed statistics, and advanced
  origin configurations.

### What This Skill Owns

| In scope | Out of scope (delegate) |
|---|---|
| CDN domain CRUD (list / create / configure / start / stop / delete) | EIP / origin IP management → `huaweicloud-eip-ops` |
| Cache refresh (URL / directory) + preheat | HTTPS certificate provisioning → `huaweicloud-waf-ops` |
| Traffic / bandwidth / hit rate statistics | Static object hosting → `huaweicloud-obs-ops` |
| Cache rule configuration (TTL, priority, etc.) | Billing / cost analysis → `huaweicloud-billing-ops` |
| Domain-level access control (referer, IP blacklist/whitelist) | CDN+OBS combined → `huaweicloud-obs-ops` |

## Five Core Standards (Quality Gates)

| # | Standard | How This Skill Fulfills It |
|---|----------|---------------------------|
| 1 | **Clear Boundaries** | SHOULD/SHOULD NOT Use + delegation matrix |
| 2 | **Structured I/O** | `{{env.*}}` / `{{user.*}}` / `{{output.*}}` convention |
| 3 | **Explicit Actionable Steps** | Pre-flight → Execute → Validate → Recover per operation |
| 4 | **Complete Failure Strategies** | ≥10 CDN error codes; HALT vs retry per type |
| 5 | **Absolute Single Responsibility** | CDN domains + cache only; cross-product ops delegate |
| 6 | **GCL Adversarial Rubric** | `## Quality Gate (GCL)`; `references/rubric.md` (8 sections); `references/prompt-templates.md` (7 sections) |

### Three-Pillar Ops Integration (FinOps + SecOps + AIOps)

| Pillar | Skill Integration | Reference |
|---|---|---|
| **FinOps** | Traffic-based billing (egress × unit_price), hit rate optimization, idle domain detection, TTL right-sizing | `references/well-architected-assessment.md` §3 |
| **SecOps** | HTTPS enforcement, referer/IP ACL, origin authentication, WAF chaining | `references/well-architected-assessment.md` §4 |
| **AIOps** | Cache purge storm, hit rate degradation, origin 5xx spike, bandwidth DDoS patterns | `references/advanced/aiops-patterns.md` |

## Trigger & Scope (Agent-Readable)

### SHOULD Use This Skill When

- User mentions "CDN" / "内容分发" / "加速域名" / "缓存刷新" / "预热" / "命中率"
- Task keywords: 接入CDN, 创建加速域名, 刷新缓存, 预热, 命中率, CDN带宽, 域名配置
- User asks to: deploy, configure, troubleshoot, or monitor CDN **via API, SDK, CLI, or automation**
- Anomaly reported: "CDN命中率下降", "CDN带宽突增", "源站被打满", "CDN缓存未生效"

### SHOULD NOT Use This Skill When

- EIP / origin IP management → `huaweicloud-eip-ops`
- HTTPS certificate provisioning / WAF policy → `huaweicloud-waf-ops`
- Static website / OBS object hosting → `huaweicloud-obs-ops`
- Pure billing reconciliation → `huaweicloud-billing-ops`

### Delegation Rules

- Before creating a CDN domain: ensure the origin server (ECS / OBS / IP) exists and is accessible.
- For CDN + EIP combined requests: handle EIP via `huaweicloud-eip-ops` first.
- For CDN cost analysis: use this skill's FinOps section, then delegate to `huaweicloud-billing-ops` for invoice-level detail.

## Variable Convention (Agent-Readable)

| Placeholder | Meaning | Agent Action |
|---|---|---|
| `{{env.HW_ACCESS_KEY_ID}}` | AK from runtime env | NEVER ask user; fail if unset |
| `{{env.HW_SECRET_ACCESS_KEY}}` | SK from runtime env | NEVER ask user; fail if unset |
| `{{env.HW_REGION_ID}}` | Region | Use default if user agrees |
| `{{env.HW_PROJECT_ID}}` | Project ID | Use default if user agrees |
| `{{user.domain_id}}` | CDN domain resource ID | Ask once; reuse |
| `{{user.domain_name}}` | Accelerated domain (e.g. `example.com`) | Ask once; validate DNS |
| `{{user.origin_address}}` | Origin server address | Ask once; validate format |
| `{{user.cache_urls}}` | URLs or directories to refresh | Ask per operation |
| `{{output.domain_id}}` | CDN domain ID from API | Parse from response |
| `{{output.job_id}}` | Async job ID (refresh/preheat) | Poll for completion |

> **`{{env.*}}` MUST NOT** be collected from the user. **`{{user.*}}`** MUST be collected interactively when missing.

> **Security Warning (Credential Masking — MANDATORY):** **NEVER** log, print, or expose
> `HW_SECRET_ACCESS_KEY`, `SecretAccessKey`, or any credential value in console output,
> debug messages, error messages, or GCL traces.

## API and Response Conventions

- **OpenAPI is canonical**: see [CDN API v1](https://support.huaweicloud.com/api-cdn/).
- **Errors:** Map SDK/HTTP errors to `error_code` / `error_msg`.
- **Timestamps:** ISO 8601 with timezone.
- **Idempotency:** Domain name is unique; retry on conflict = no-op or update.
- **Async ops:** Refresh cache, preheat cache, create-domain → return `job_id`; poll for completion.

## Quick Start

### Prerequisites
- [ ] Huawei Cloud CLI installed (`hcloud cdn` available)
- [ ] Credentials: `HW_ACCESS_KEY_ID`, `HW_SECRET_ACCESS_KEY`
- [ ] Region and Project ID: `HW_REGION_ID`, `HW_PROJECT_ID`

### Verify Setup
```bash
hcloud cdn list-domain --region {{env.HW_REGION_ID}} --output json | jq '.result[] | {id, domain_name, cname, status}'
```

### Your First Command
```bash
# List all CDN domains
hcloud cdn list-domain --region {{env.HW_REGION_ID}} \
  --output json | jq '.result[] | {id, domain_name, status, service_area}'
```

## Capabilities at a Glance

| Operation | Description | Complexity | Risk Level |
|---|---|---|---|
| List CDN domains | List all accelerated domains | Low | None |
| Create CDN domain | Add new domain for acceleration | Medium | Medium |
| Configure domain | Set cache rules, origin, HTTP headers | Medium | Medium |
| Start / Stop domain | Enable / suspend CDN acceleration | Low | Medium |
| Refresh cache | Purge cached content (URL or directory) | Low | Low |
| Preheat cache | Pre-populate cache for big events | Low | Low |
| Query statistics | Bandwidth, traffic, hit rate, status codes | Low | None |
| Delete CDN domain | Remove domain (irreversible) | Low | **High** |

## Execution Flows

### Operation 1: List CDN Domains

#### Execution — CLI

```bash
hcloud cdn list-domain --region {{env.HW_REGION_ID}} --output json
```

#### Output fields

| Field | JSON path | Meaning |
|---|---|---|
| Domain ID | `result[].id` | `{{output.domain_id}}` |
| Domain name | `result[].domain_name` | `{{user.domain_name}}` |
| Status | `result[].status` | `online` / `offline` / `configuring` |
| CNAME | `result[].cname` | DNS alias to configure |
| Service area | `result[].service_area` | `mainland_china` / `outside_mainland` / `global` |

### Operation 2: Create CDN Domain

#### Pre-flight Checks

| Check | Method | Expected | On Failure |
|---|---|---|---|
| Domain name ownership | Ask user to confirm DNS CNAME ownership | User confirms | HALT; CNAME must point to `{{user.domain_name}}.cdn.{region}.myhwcdn.com` |
| Origin accessibility | `curl -I {{user.origin_address}}` | 2xx response | HALT; origin must be reachable |
| Origin type | Confirm OBS / ECS / IP | OBS bucket / ECS EIP / raw IP | Default to IP if unspecified |
| Quota | `hcloud cdn show-quota` (or SDK) | Domain count within limit | HALT; raise quota |

#### Execution — CLI

```bash
hcloud cdn create-domain \
  --region "{{user.region}}" \
  --domain-name "{{user.domain_name}}" \
  --business-type "web" \
  --service-area "mainland_china" \
  --origin "{{user.origin_address}}" \
  --origin-type "ipaddr"   # or "obs" for OBS buckets
```

#### Execution — JIT Go SDK (Fallback)

```go
//go:build ignore
package main

import (
    "fmt"
    "os"

    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cdn/v1"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cdn/v1/model"
)

func main() {
    ak, sk, region := os.Getenv("HW_ACCESS_KEY_ID"), os.Getenv("HW_SECRET_ACCESS_KEY"), os.Getenv("HW_REGION_ID")
    if ak == "" || sk == "" || region == "" {
        fmt.Fprintln(os.Stderr, "missing required env: HW_ACCESS_KEY_ID / HW_SECRET_ACCESS_KEY / HW_REGION_ID")
        os.Exit(2)
    }
    cfg := config.DefaultHttpConfig()
    client := cdn.CdnClientBuilder().
        WithEndpoint(fmt.Sprintf("cdn.%s.myhuaweicloud.com", region)).
        WithCredential(basic.NewCredentialsBuilder().WithAk(ak).WithSk(sk).Build()).
        WithHttpConfig(cfg).Build()

    req := &model.CreateDomainRequest{
        Body: &model.CreateDomainRequestBody{
            Domain: &model.CreateDomainDetail{
                DomainName:   "example.com",
                BusinessType: "web",
                ServiceArea:  "mainland_china",
                Sources: []model.SourceDomainConfig{
                    {OriginAddr: "1.2.3.4", OriginType: "ipaddr"},
                },
            },
        },
    }
    resp, err := client.CreateDomain(req)
    if err != nil {
        fmt.Fprintln(os.Stderr, "CreateDomain failed:", err)
        os.Exit(1)
    }
    fmt.Printf("domain_id=%s\n", *resp.Domain.Id)
}
```

#### Post-execution Validation
Poll `hcloud cdn list-domain` until `status = online` or `configuring` (takes 1–10 min).

### Operation 3: Configure CDN Domain

#### Execution — CLI

```bash
# Set cache TTL rules
hcloud cdn modify-domain-config \
  --region "{{user.region}}" \
  --domain-id "{{user.domain_id}}" \
  --cache-rules "[{\"match_type\":\"all\",\"ttl\":0,\"priority\":0}]"

# Enable HTTPS
hcloud cdn modify-domain-config \
  --region "{{user.region}}" \
  --domain-id "{{user.domain_id}}" \
  --https-enabled true \
  --https-cert-name "{{user.cert_name}}"
```

### Operation 4: Start / Stop Domain

#### Execution

```bash
hcloud cdn start-domain --region "{{user.region}}" --domain-id "{{user.domain_id}}"
hcloud cdn stop-domain --region "{{user.region}}" --domain-id "{{user.domain_id}}"
```

#### Post-execution Validation
Poll `describe-domain` until `status` reflects target state.

### Operation 5: Refresh Cache

#### Pre-flight
- Confirm user has the correct URLs to purge (wildcard requires confirmation).
- **Warning:** Mass refresh can overload origin. Recommend staged refresh for >100 URLs.

#### Execution

```bash
# Single URL
hcloud cdn refresh-cache \
  --region "{{user.region}}" \
  --type "file" \
  --urls "https://{{user.domain_name}}/path/to/file"

# Directory (trailing slash)
hcloud cdn refresh-cache \
  --region "{{user.region}}" \
  --type "directory" \
  --urls "https://{{user.domain_name}}/static/"
```

#### Post-execution Validation
Async job; poll `hcloud cdn list-tasks --task-type refresh_cache` until `status = finish`.

### Operation 6: Preheat Cache

#### Pre-flight
- Preheat is best-effort; not guaranteed for all edge nodes.
- For major events (product launch, flash sale): preheat 30–60 min before start.

#### Execution

```bash
hcloud cdn preheat-cache \
  --region "{{user.region}}" \
  --urls "https://{{user.domain_name}}/index.html,https://{{user.domain_name}}/assets/"
```

### Operation 7: Query Statistics

#### Execution

```bash
# Bandwidth and traffic
hcloud cdn list-stats \
  --region "{{user.region}}" \
  --domain-id "{{user.domain_id}}" \
  --start-time "$(date -d '24 hours ago' +%Y%m%dT%H%M%SZ)" \
  --end-time "$(date +%Y%m%dT%H%M%SZ)" \
  --stat-type "bandwidth,flux"

# Hit rate
hcloud cdn list-stats \
  --region "{{user.region}}" \
  --domain-id "{{user.domain_id}}" \
  --start-time "$(date -d '24 hours ago' +%Y%m%dT%H%M%SZ)" \
  --end-time "$(date +%Y%m%dT%H%M%SZ)" \
  --stat-type "hit_rate"
```

### Operation 8: Delete CDN Domain

#### Pre-flight (Safety Gate — IRREVERSIBLE)

- **MUST** require explicit confirmation: deleting `{{user.domain_name}}` (`{{user.domain_id}}`) removes it from CDN and all cached content.
- **MUST** warn: any DNS CNAME pointing to this domain will return origin directly.
- **MUST** verify domain is `offline` or `online` (not in `configuring` transitional state).

#### Execution

```bash
hcloud cdn delete-domain \
  --region "{{user.region}}" \
  --domain-id "{{user.domain_id}}"
```

#### Post-execution Validation
Poll `list-domain` until domain disappears (404 equivalent).

## FinOps at a Glance (Details in references/well-architected-assessment.md §3)

| Metric | Formula | Threshold |
|---|---|---|
| CDN traffic cost | `egress_gb × unit_price` | Budget alert at 80% |
| Hit rate | `cache_hit_requests / total_requests` | Warning < 85% |
| Idle domain | Domain `online` + 0 traffic for 7 d | Candidate for stop-domain |
| Cache TTL | TTL vs content freshness | Short TTL = higher origin load |

## SecOps at a Glance (Details in references/well-architected-assessment.md §4)

| Risk | Mitigation |
|---|---|
| Hotlink theft | Referer whitelist + anti-leech config |
| Unauthorized purge | IAM `cdn:cache:refresh` permission scoping |
| HTTPS misconfiguration | Force HTTPS + HTTP/2 + TLS 1.3 |
| Origin exposure | Origin shield / origin authentication |

## AIOps at a Glance (Details in references/advanced/aiops-patterns.md)

| Pattern | Detection Signal | Cross-skill |
|---|---|---|
| Cache purge storm | >100 refresh requests in 1h | → `huaweicloud-billing-ops` (origin cost) |
| Hit rate degradation | hit_rate < 70% for 1h | → `huaweicloud-ces-ops` (origin pull spike) |
| Origin 5xx spike | origin_5xx_rate > 10% for 10 min | → `huaweicloud-ces-ops` + origin skill |
| Bandwidth DDoS | bandwidth p99 > 10× p50 | → `huaweicloud-eip-ops` (rate limit) |

## Quality Gate (GCL)

This skill uses Generator-Critic-Loop runtime validation. Required artifacts:

- `references/rubric.md` — 8 sections: scope, thresholds, evidence, safety rules, scoring guide, examples, escalation, changelog.
- `references/prompt-templates.md` — 7 sections: Generator, Critic, Orchestrator, pre-flight overrides, anti-patterns, changelog, see also.
- `SKILL.md` metadata `gcl` block: `required: true`, `default_max_iter: 2`, `rubric_version: "v1"`.

### Runtime Roles

| Role | Responsibility | Constraint |
|---|---|---|
| Generator | Execute CDN op via `hcloud cdn` or Go SDK | Capture masked trace; never self-score |
| Critic | Score trace against `references/rubric.md` | Read-only; never see raw user request |
| Orchestrator | PASS / RETRY / SAFETY_FAIL / MAX_ITER | safety=0 aborts |

### Default Rubric Thresholds

| Dimension | Threshold | Notes |
|---|---:|---|
| correctness | ≥ 0.5 | 1.0 for delete-domain (irreversible) |
| safety | = 1.0 | Any S-rule hit or credential leak => SAFETY_FAIL |
| idempotency | ≥ 0.5 | Retry must not duplicate side effects |
| traceability | ≥ 0.5 | Command, args, response, request_id captured |
| spec_compliance | ≥ 0.5 | CLI flags and JSON paths verified against OpenAPI |

### Trace Requirements

1. Persist `audit-results/gcl-trace-{{timestamp}}.json` (format: YYYYMMDD-HHMMSS) for PASS, MAX_ITER, and SAFETY_FAIL.
2. Mask `HW_SECRET_ACCESS_KEY`, AK/SK, tokens, and authorization headers.
3. Include sanitized `operation_intent` so Critic can assess expected state without seeing raw user wording.

## Reference Directory

- [Core Concepts](references/core-concepts.md) — CDN model, origin types, cache behavior
- [API & SDK Usage](references/api-sdk-usage.md) — Go SDK JIT patterns
- [CLI Usage](references/cli-usage.md) — `hcloud cdn` command reference
- [Troubleshooting Guide](references/troubleshooting.md) — Top CDN failure patterns
- [Monitoring & Alerts](references/monitoring.md) — Bandwidth, hit rate, origin metrics
- [Integration](references/integration.md) — Cross-skill delegation matrix
- [Well-Architected Assessment](references/well-architected-assessment.md)
- [AIOps Patterns](references/advanced/aiops-patterns.md)
- [GCL Rubric](references/rubric.md)
- [GCL Prompt Templates](references/prompt-templates.md)
