---
name: huaweicloud-skill-generator
description: >-
  Use when the user needs to create or update a Huawei Cloud Agent Skill
  (`huaweicloud-*-ops`) in this repository — even if they don't explicitly ask for
  scaffolding or generation. Triggers include: user wants to "add a skill for
  product X", "regenerate from OpenAPI", or "fix gaps found during review".
  Also use when an existing skill needs realignment after API doc changes or
  fails a governance/adversarial review. Not for executing live changes against
  cloud accounts or for one-off debugging with no intent to maintain.
license: MIT
compatibility: >-
  Access to Huawei Cloud official documentation, OpenAPI/Swagger for the product,
  `huaweicloud-skill-generator/references/huaweicloud-skill-template.md`,
  `references/evaluation-driven-workflow.md`,
  `references/governance-and-adversarial-review.md` (when present),
  `references/prompt-library.md` (structured prompt repository),
  `references/gcl-prompt-backbone.md` (shared GCL prompt backbone),
  `references/optimization-analysis.md` (three-dimensional optimization framework),
  `references/user-experience-spec.md` (mandatory UX requirements for generated skills),
  `references/execution-environment.md` (CLI + Go SDK setup details),
  `references/cli-behavior.md` (verified huawei CLI behavioral notes),
  and agentskills.io frontmatter conventions.
metadata:
  author: huaweicloud
  version: "1.0.0"
  last_updated: "2026-05-20"
  runtime: Harness AI Agent, Claude Code, Cursor, or compatible Agent runtimes
  type: meta-skill
  guidance_freedom_level: medium
  go_version_minimum: "1.21"
  go_version_jit: "1.25+"
---

# Huawei Cloud Skill Generator (Meta-Skill)

## Quick Start

### What This Skill Does
Scaffolds new or updates existing `huaweicloud-[product]-ops` skills in this repository, based on official Huawei Cloud OpenAPI specs. This is a **meta-skill** — it generates runbooks for agents, not operational execution against cloud accounts.

### Prerequisites
- [ ] Access to OpenAPI/Swagger spec for the target Huawei Cloud product
- [ ] Read access to this repository's template files
- [ ] Network access to Huawei Cloud documentation URLs

### Your First Generation
```
Input: "Generate huaweicloud-ecs-ops for ECS instances, disks, and snapshots"
Output: huaweicloud-ecs-ops/ directory with SKILL.md and references/
```

### Next Steps
- [Generation Workflow](#evaluation-driven-generation-workflow) — Step-by-step generation process
- [Anti-Pattern Checklist](#anti-pattern-checklist) — Common mistakes to avoid
- [P0/P1 Checklist](#p0p1-checklist) — Quality gates for generated skills

---

## Overview

This **meta-skill** defines **how** to author a new **product-scoped** operational skill (e.g. `huaweicloud-ecs-ops`) **inside this repo**. It does **not** perform maintenance against a user's cloud account. Live work uses the generated `huaweicloud-[product]-ops` skills (official **`hcloud` CLI** with **JIT Go SDK fallback**).

### Guidance Freedom Level: Medium (Provide Templates)

This meta-skill operates at **Medium** guidance level: it provides **templates and frameworks** ([huaweicloud-skill-template.md](references/huaweicloud-skill-template.md), prompt library, UX spec) while allowing the agent to adapt based on product-specific context. Low-level scripts (CLI installation, Go runtime JIT download) are detailed in [references/execution-environment.md](references/execution-environment.md).

### Core Principle

Generated skills are **agent-readable runbooks**: triggers, env vs user placeholders, pre-flight → execute → validate → recover, safety gates, and outputs **grounded in OpenAPI and verified CLI behavior**, not guessed.

### Technology Stack
- **CLI:** `hcloud` / `openstack` CLI (primary execution path)
- **SDK:** Huawei Cloud Go SDK (`github.com/huaweicloud/huaweicloud-sdk-go-v3`) — JIT fallback
- **JIT execution:** `go run` (script mode, dynamic generation)

### Repository Scope
All generated layout and policies apply **only** to the `hcloud-skills` monorepo unless explicitly stated elsewhere.

---

## Role Boundary (Agent-Readable)

| This meta-skill **does** | This meta-skill **does not** |
|--------------------------|------------------------------|
| Choose **extend** vs **new** `huaweicloud-[product]-ops` | Replace deep product knowledge already in an existing ops skill |
| Scaffold `SKILL.md`, `references/*`, `assets/*` from the template | Call Huawei Cloud APIs on behalf of the user |
| Enforce naming, frontmatter, P0/P1, delegation, and **governance** hooks | Invent request/response fields or CLI flags without official doc verification |
| Point authors to **adversarial review** before merge (when governance doc exists) | Store or echo real credentials |

If the user wants **operational execution** (e.g. "create a resource"), load the appropriate `huaweicloud-*-ops` skill for that product — not this generator.

---

## When to Use / Not Use

### Use When
- A new Huawei Cloud product needs a **first** ops skill in **this repo**
- An existing skill lacks P0 elements (triggers, placeholders, flows, recovery, destructive gates)
- OpenAPI or official docs changed; the skill should be **realigned** (bump version/changelog)
- A contributor needs the **standard directory layout** for a new `huaweicloud-[product]-ops`

### Do NOT Use When
- One-off debugging with no intent to maintain a reusable skill
- Non–Huawei-Cloud application work
- You only need billing/IAM execution — use dedicated ops skills when they exist

---

## Input / Output Structure

### Input

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product.name` | string | Yes | English product name (e.g., ECS, RDS) |
| `product.slug` | string | Yes | CLI product slug — verify via `hcloud help` or official docs |
| `product.chinese_name` | string | No | Chinese name for trigger matching |
| `primary_resource` | string | Yes | Primary resource type (e.g., Instance, DBInstance) |
| `api_service_id` | string | Yes | API service identifier from OpenAPI tags or SDK package |
| `openapi_url` | string | Recommended | OpenAPI/Swagger URL or path — required for API-accurate fields |
| `operation_list` | string[] | Yes | List of operations (create, describe, modify, delete, list, product-specific) |
| `doc_urls` | string[] | Recommended | Official documentation URLs |
| `cli_support_evidence` | string | Yes | Confirmation that CLI exposes this product (or JIT SDK fallback needed) |

### Output

| Artifact | Description | Required When |
|----------|-------------|---------------|
| `huaweicloud-[product]-ops/SKILL.md` | Main skill runbook — triggers, flows, recovery, safety gates, Well-Architected assessment | Always |
| `references/core-concepts.md` | Architecture, limits, regions, quotas, dependency graph, SPOF analysis | Always |
| `references/api-sdk-usage.md` | Operation map, required fields, pagination, request/response snippets | Always |
| `references/cli-usage.md` | CLI command map, coverage gap table, invocation patterns | `cli_applicability: cli-first` / `dual-path` / `cli-only` |
| `references/troubleshooting.md` | Error codes (≥ 10), ordered diagnostics, multi-round diagnosis | Always |
| `references/monitoring.md` | Metrics, dashboards, alerts, cost & performance metrics | Product has monitoring metrics |
| `references/integration.md` | JIT SDK setup, env vars, cross-skill delegation matrix | Always |
| `references/well-architected-assessment.md` | Five-pillar + FinOps + SecOps + AIOps assessment | Always |
| `references/enhanced-self-healing-framework.md` | Self-healing patterns for installation flows | Always (referenced) |
| `references/knowledge-base.md` | Fault pattern library for diagnostic skills | AIOps/diagnosis skills |
| `references/observability.md` | Metrics→Logs→Traces linkage | Monitoring/AIOps skills |
| `references/idempotency-checklist.md` | Idempotent behavior for retries/automation | Automation-heavy products |
| `references/rubric.md` | GCL rubric with 8 numbered sections and product safety rules | Always |
| `references/prompt-templates.md` | GCL Generator/Critic/Orchestrator templates with sanitized `operation_intent` | Always |
| `assets/example-config.yaml` | Example configuration with UX and optimization settings | Always |
| `assets/eval_queries.json` | Trigger accuracy evaluation queries for the generated skill | Always |

---

## Five Core Standards (Quality Gates)

Every generated skill MUST satisfy these five standards. Reference them throughout the generation workflow.

### Standard 1: Clear Boundaries (边界明确)
- **SHOULD use** conditions: precise, with keywords and intent matching
- **SHOULD NOT use** conditions: explicit negative cases that prevent misfire
- **Delegation rules**: clear pointers to related skills

### Standard 2: Structured I/O (输入输出结构化)
- Input parameters defined with types and sources (`{{env.*}}`, `{{user.*}}`)
- Output fields defined with JSON paths from OpenAPI response schemas
- Placeholder conventions: `{{env.*}}` (from runtime, NEVER ask user), `{{user.*}}` (interactive collect), `{{output.*}}` (from API response)

### Standard 3: Explicit Actionable Steps (步骤明确可执行)
- Every operation: Pre-flight → Execute → Validate → Recover
- Steps are numbered, imperative, specific — not descriptive summaries
- CLI and SDK paths documented separately when both apply

### Standard 4: Complete Failure Strategies (失败策略完备)
- Error taxonomy with product-specific error codes (≥ 10)
- Each error pattern: max retries, backoff strategy, agent action, UX feedback
- HALT vs retry distinction; credential, quota, and business errors clearly separated

### Standard 5: Absolute Single Responsibility (职责绝对单一)
- One skill = one product = one primary resource model
- Cross-product delegation: document in Trigger & Scope, do NOT duplicate full flows
- Naming: `huaweicloud-[product]-ops` (lowercase, hyphenated)

### Standard 6: Asset Distillation Hook (复利资产沉淀钩子)
- Every generated skill MUST inject a trailing line at the end of its `SKILL.md`:
  `> 任务完成后按根 AGENTS.md 的「复利资产沉淀机制 (CADL)」复盘并沉淀可复用资产。`
- This makes the skill self-aware of the Compound-Asset Distillation Loop (CADL) defined in root `AGENTS.md`, so any agent calling the skill sees the trigger signal after task completion.
- The hook covers ALL reusable-asset dimensions — review patterns, fix patterns, cross-skill collaboration, verification findings, pitfall experience — NOT limited to CodeGraph integration.
- Generator itself MUST also follow CADL after each generation run (distill generator-specific patterns into root or user-level AGENTS.md).

---

## Anti-Pattern Checklist

Before and during generation, check against these common anti-patterns:

| # | Anti-Pattern | How It Manifests | Correction |
|---|-------------|-----------------|------------|
| 1 | **Skill = Prompt** | Writing conversational instructions instead of executable steps | Use imperative numbered steps; define I/O; separate triggers from execution |
| 2 | **Skill = Human Doc** | Explaining concepts instead of instructing the agent | Use model-parsable structured language; define behavior boundaries |
| 3 | **Feature Bundling** | One skill tries to do everything (create + monitor + backup + billing) | Split into single-responsibility skills; delegate to existing skills |
| 4 | **API Hallucination** | Inventing field names, JSON paths, or CLI flags not in official docs | Cross-reference every field against OpenAPI or verified CLI output |
| 5 | **Credential Leaking** | Printing, logging, or echoing secret values in any execution path | Mask all credentials with `***` / `<masked>`; check existence only |
| 6 | **No Safety Gate** | Destructive operations (delete, stop, release) without explicit confirmation | Add confirmation step before every destructive path (CLI + SDK) |
| 7 | **Hardcoded Values** | Regions, timeouts, or limits baked into instructions | Use `{{env.*}}` / `{{user.*}}` placeholders; document defaults separately |
| 8 | **Missing Failure Path** | Only documenting the success path; no error handling | Add failure recovery table with error codes, retry logic, HALT conditions |
| 9 | **Over-Engineering** | Adding advanced features before core flow works | Follow evaluation-driven approach: start minimal, expand step by step |
| 10 | **Redundant Redundancy** | Repeating the same info across SKILL.md and references | SKILL.md is entry point; references provide depth — no duplication |

---

## Evaluation-Driven Generation Workflow

This workflow follows the **"fail first, evaluate first"** principle: define what "good" looks like before generating. At each critical node, validate the output and loop back for corrections.

> **Copy the checklist below before starting, and mark each step as you complete it.**

### Workflow Checklist

```
[ ] Step 1: Define Evaluation Targets — What does success look like?
[ ] Step 2: Analyze Sources — Extract operations, fields, errors from OpenAPI
    ↓ [Feedback Loop: Sources complete? If gaps found → research, then return]
[ ] Step 3: Scaffold Layout — Create directory from template
[ ] Step 4: Populate SKILL.md — Fill template with verified data
    ↓ [Feedback Loop: Five core standards satisfied? If not → fix and re-verify]
[ ] Step 5: Fill Reference Files — Complete all references/
    ↓ [Feedback Loop: All files populated? If gaps → fix]
[ ] Step 6: Verify & Review — P0/P1 checklist + adversarial review + 3-pillar assessment
    ↓ [Feedback Loop: Any failures? → return to Step 4 or 5; re-verify after fix]
    ↓ [Self-Reflection 1: FinOps cost patterns adequate? SecOps coverage complete? AIOps maturity?]
    ↓ [Self-Reflection 2: Any gaps in Well-Architected pillars? Missing FinOps/SecOps/AIOps integrations?]
[ ] Step 7: Final Anti-Pattern Check — Run anti-pattern checklist above
```

---

### Step 1: Define Evaluation Targets

Before generating anything, define **3-5 evaluation cases** for the target skill. Each case has a clear PASS/FAIL criterion.

**Template:**
```markdown
| ID | Scenario | Expected Behavior | PASS Condition |
|----|----------|-----------------|----------------|
| E1 | User asks to create a resource with minimal input | Skill prompts for required fields, uses smart defaults for optional | ≤ 2 prompts before execution |
| E2 | User asks to delete a resource | Skill asks for explicit confirmation with resource identifier | Confirmation step present |
| E3 | API returns QuotaExceeded | Skill returns clear error message with remediation steps | Error follows `[ERROR] code → explanation → fix → next step` |
| E4 | User asks about cost optimization | FinOps section provides actionable right-sizing or billing model guidance | Cost assessment section present |
| E5 | Security audit request | SecOps section documents minimum IAM permissions and network isolation | IAM + network security documented |
| E6 | Anomaly detection alert | AIOps section triggers cross-skill diagnosis with delegation matrix | Multi-metric correlation + delegation present |
| E7 | Well-Architected security check | Skill documents minimum RAM/IAM permissions for operations | IAM section in `well-architected-assessment.md` |
| E8 | Cost waste detection | Skill detects idle resource pattern and recommends right-sizing | Cost assessment + waste detection present |
```

**Purpose:** These cases anchor the generation process. Every feature in the generated skill must trace back to at least one evaluation case.

---

### Step 2: Analyze Sources

Extract from OpenAPI and official docs:

- **Operations**: OperationIds grouped by resource tag
- **Parameters**: Required vs optional, types, enums, defaults
- **Response schemas**: JSON paths, terminal states, pagination
- **Error codes**: Product-specific error taxonomy (≥ 10 codes)
- **Async behavior**: Polling intervals, terminal state names
- **CLI coverage**: Which operations CLI supports vs SDK-only
- **API version drift** (updating existing skills): Compare current OpenAPI against `metadata.api_profile`; flag changed signatures, deprecations, new parameters

**Validation checkpoint:** Before proceeding, confirm:
- [ ] All operationIds are real (not invented)
- [ ] JSON paths are from actual response schemas
- [ ] Error codes are documented in OpenAPI or official docs
- [ ] `cli_applicability` is correctly determined
- [ ] API version drift report generated (if updating existing skill)

---

### Step 3: Scaffold Directory Layout

```text
huaweicloud-[product]-ops/
├── SKILL.md
├── references/
│   ├── core-concepts.md
│   ├── api-sdk-usage.md
│   ├── cli-usage.md              # Required when cli_applicability: cli-first or dual-path
│   ├── troubleshooting.md
│   ├── monitoring.md              # When monitoring in scope
│   ├── integration.md
│   ├── well-architected-assessment.md  # MANDATORY: five-pillar + FinOps + SecOps + AIOps
│   ├── rubric.md                 # MANDATORY: GCL 8-section rubric
│   ├── prompt-templates.md       # MANDATORY: GCL 7-section prompt templates
│   └── idempotency-checklist.md  # When retries/automation required
├── assets/
│   ├── example-config.yaml
│   └── eval_queries.json         # MANDATORY: trigger accuracy eval queries
```

---

### Step 4: Populate SKILL.md

Base: [huaweicloud-skill-template.md](references/huaweicloud-skill-template.md).

Replace all `[Placeholder]` with product-specific content derived from Step 2. Every field, JSON path, and CLI command MUST be traceable to OpenAPI or verified CLI output.

**Frontmatter requirements:**
| Field | Rule |
|-------|------|
| `name` | `huaweicloud-[product]-ops` — lowercase, hyphens, ≤ 64 chars |
| `description` | Third person, triggers only (per OpenSpec) |
| `cli_applicability` | `cli-first` / `dual-path` / `sdk-only` / `cli-only` |
| `cli_support_evidence` | Cite confirmation via `hcloud help` or official docs |
| `metadata.gcl` | Include `required`, `default_max_iter`, `rubric_version`, `trace_path` |

**Validation checkpoint (Five Core Standards):**
- [ ] **Boundary**: SHOULD/SHOULD NOT Use conditions complete?
- [ ] **I/O**: All placeholders correctly typed?
- [ ] **Steps**: Every operation has Pre-flight → Execute → Validate → Recover?
- [ ] **Failure**: Error taxonomy ≥ 10 codes, each with recovery action?
- [ ] **Single Responsibility**: One product, one resource model, clear delegation?
- [ ] **GCL**: `## Quality Gate (GCL)` present and references `rubric.md` + `prompt-templates.md`?

---

### Step 5: Fill Reference Files

| File | Content | Source |
|------|---------|--------|
| `core-concepts.md` | Architecture, limits, regions, quotas, resource relationships | Official docs |
| `api-sdk-usage.md` | Operation map, required fields, pagination, request/response snippets | OpenAPI |
| `cli-usage.md` | CLI command map, coverage gap table, JSON output paths | Verified CLI output |
| `troubleshooting.md` | Error code table, ordered diagnostic steps, product-specific patterns | OpenAPI + experience |
| `monitoring.md` | Metrics, dashboards, alarms, anomaly patterns | CES docs |
| `integration.md` | Go bootstrap, JIT SDK setup, dependency config | Execution environment |
| `well-architected-assessment.md` | Five pillars + FinOps + SecOps + AIOps integration | All official pillars |
| `rubric.md` | 8 numbered GCL sections with product-specific safety rules | `docs/gcl-spec.md` + product risk model |
| `prompt-templates.md` | 7 numbered GCL sections; include `{{output.operation_intent}}`; no bare `{...}` placeholders | `references/gcl-prompt-backbone.md` |

---

### Step 6: Verify & Review

Run the [P0/P1 Checklist](#p0p1-checklist) below against the generated skill. Run the [Adversarial Review](references/governance-and-adversarial-review.md) scenarios (when present).

**After initial verification, execute multi-round self-reflection:**

#### Self-Reflection Round 1: Foundation Check
1. **FinOps**: Are cost optimization patterns actionable? Billing model comparison present? Idle detection documented? Unit economics defined? Cost anomaly detection covered?
2. **SecOps**: IAM permissions minimum documented? Credential masking enforced? Network isolation recommended? Zero trust alignment? Security incident response runbook? Supply chain security?
3. **AIOps**: Multi-metric correlation defined? Cross-skill delegation matrix present? Knowledge base populated? SLO/SLI with Error Budget defined? Change correlation analysis? Capacity forecasting?

#### Self-Reflection Round 2: Critical Analysis
4. **Gap Analysis**: What would break if the user follows this skill in production?
5. **Alternative Coverage**: Is there a better way to document this that would reduce agent confusion?
6. **Escalation Paths**: Are HALT conditions clear? Are there enough non-retryable error patterns?
7. **Cross-Pillar Synergy**: Do FinOps recommendations conflict with reliability? Does SecOps create performance bottlenecks? Have trade-off decisions been documented in the Cross-Pillar Trade-off Matrix?
8. **Maturity Assessment**: Has the skill self-assessed against the Maturity Scorecard? Which dimensions are below target?
9. **Sustainability**: Has the resource carbon efficiency been considered? Are there green computing recommendations?

**For any failure:**
1. Identify the gap
2. Return to Step 4 (SKILL.md) or Step 5 (references)
3. Fix the gap
4. Re-verify the full checklist

---

### Step 7: Final Anti-Pattern Check

Run the [Anti-Pattern Checklist](#anti-pattern-checklist) above against the generated skill. Every item must pass.

---

## Description Optimization (Trigger Accuracy)

The `description` field in frontmatter is the sole trigger mechanism for skill activation.

| Principle | Guideline | Example |
|-----------|-----------|---------|
| **Imperative phrasing** | Frame as instruction to agent: "Use when..." | `Use when the user needs to...` |
| **Focus on user intent** | Describe what user is trying to achieve, not skill mechanics | Focus on problems user solves, not CLI/SDK internals |
| **Err on the side of pushy** | Include implicit trigger scenarios explicitly | `even when the user doesn't explicitly mention [product]` |
| **Negative boundaries** | State what the skill is NOT for | `Not for billing, IAM, or related products` |
| **Keep concise** | Under 1024 character hard limit | Aim for 300–700 characters |

---

## Before You Generate: Decisions

### Extend vs New Directory
- **Extend** same product and resource model
- **New** `huaweicloud-[product]-ops` when the **service/API surface** or **primary resource** is distinct

### Naming
- Pattern: `huaweicloud-[product]-ops` (lowercase, hyphenated)
- Search the repo for collisions before creating

### Huawei Cloud Service Mapping

| Huawei Cloud Service | Abbreviation | CLI/SDK Package | Primary Operations |
|----------------------|-------------|-----------------|-------------------|
| Elastic Cloud Server | ECS | `huaweicloud-sdk-go-v3/services/ecs` | Create, Delete, Describe, Resize |
| Cloud Eye Service | CES | `huaweicloud-sdk-go-v3/services/ces` | Alarm, Metric, Dashboard |
| Virtual Private Cloud | VPC | `huaweicloud-sdk-go-v3/services/vpc` | Create, Delete, Describe |
| Elastic Volume Service | EVS | `huaweicloud-sdk-go-v3/services/evs` | Create, Attach, Detach, Snapshot |
| Relational Database Service | RDS | `huaweicloud-sdk-go-v3/services/rds` | Instance, Backup, Restore |
| Cloud Container Engine | CCE | `huaweicloud-sdk-go-v3/services/cce` | Cluster, Node, Addon |
| Distributed Cache Service | DCS | `huaweicloud-sdk-go-v3/services/dcs` | Instance, Backup, Resize |
| Distributed Message Service | DMS | `huaweicloud-sdk-go-v3/services/dms` | Queue, Topic, Group |
| Identity and Access Management | IAM | `huaweicloud-sdk-go-v3/services/iam` | User, Role, Policy |
| Elastic Load Balance | ELB | `huaweicloud-sdk-go-v3/services/elb` | Listener, Pool, Health |
| Object Storage Service | OBS | `huaweicloud-sdk-go-v3/services/obs` | Bucket, Object, ACL |
| GaussDB | GaussDB | `huaweicloud-sdk-go-v3/services/gaussdb` | Instance, Backup, Monitor |
| Host Security Service | HSS | `huaweicloud-sdk-go-v3/services/hss` | Host, Vulnerability, Event |
| Web Application Firewall | WAF | `huaweicloud-sdk-go-v3/services/waf` | Policy, Rule, Domain |
| Log Tank Service | LTS | `huaweicloud-sdk-go-v3/services/lts` | Log Group, Log Stream, Search |

### Sources of Truth
- **OpenAPI + official docs** beat forums and chat logs
- Pin an API/SDK profile in skill `metadata` or `references/integration.md`
- Official docs: https://support.huaweicloud.com/api/

### Secrets
- Only `{{env.*}}` **names** and documentation; never real keys
- Credential masking is MANDATORY

### CLI + JIT Go SDK
- Primary path: `hcloud` / `openstack` CLI
- Fallback path: JIT Go SDK via `github.com/huaweicloud/huaweicloud-sdk-go-v3`
- Execution environment details: [references/execution-environment.md](references/execution-environment.md)

---

## Governance (Expert Recommendation)

**Minimal adversarial review** gives high return for low cost. Treat [governance-and-adversarial-review.md](references/governance-and-adversarial-review.md) as the **reviewer's companion** to this meta-skill.

---

## Three-Pillar Ops Integration

Every generated skill MUST integrate FinOps, SecOps, and AIOps best practices into its operational runbook. This extends beyond the standard Well-Architected pillars.

### FinOps Integration (财务运营)
- **Cost Visibility**: Resource cost attribution, billing model comparison, budget tracking
- **Cost Optimization**: Right-sizing, lifecycle cost management, waste detection, idle resource identification
- **Cost Accountability**: Cost center tagging, chargeback modeling, ROI analysis
- Every skill MUST document: billing model table, right-sizing guidance, waste detection pattern

### SecOps Integration (安全运营)
- **Identity Security**: IAM minimum permissions, credential management, MFA recommendations
- **Network Security**: VPC isolation, security group patterns, DDoS protection
- **Data Security**: Encryption at rest/in transit, backup protection, compliance alignment
- **Threat Detection**: HSS integration, WAF patterns, vulnerability management
- Every skill MUST document: IAM policy table, network isolation guidance, encryption recommendations

### AIOps Integration (智能运营)
- **Multi-Metric Correlation**: ≥ 4 anomaly patterns with detection logic
- **Cross-Skill Diagnosis**: Delegation matrix, decision trees, root cause localization
- **Knowledge Base**: Fault patterns, cascade failures, historical diagnosis
- **Self-Healing**: Automated recovery, graceful degradation, health verification
- Proactive Inspection: Scheduled巡检, trend prediction, capacity forecasting
- Every diagnostic skill MUST document: anomaly patterns, delegation matrix, knowledge base

---

## P0/P1 Checklist

### P0 — MUST PASS

- [ ] **Trigger & Scope** with SHOULD-use / SHOULD-NOT-use and delegation rules
- [ ] **Variables:** `{{env.*}}` vs `{{user.*}}`; no secret literals; `{{env.*}}` never collected from user
- [ ] **Flows:** Pre-flight → Execute → Validate → Recover for **each** critical operation
- [ ] **Primary path** per `cli_applicability` documented
- [ ] **Failure recovery:** HALT vs retry; throttling with exponential backoff; non-retryable business errors
- [ ] **API fidelity:** Fields and paths traceable to OpenAPI/SDK for the stated version
- [ ] **CLI fidelity:** Commands match official docs; JSON paths verified
- [ ] **Safety gates** for destructive operations
- [ ] **Timeouts** for polling and long-running operations (default: 5s interval, 300s max wait)
- [ ] **Self-Healing Framework:** All installation flows follow enhanced-self-healing-framework pattern
- [ ] **UX Onboarding:** Quick Start section present; first-time user can execute first command within 60 seconds
- [ ] **UX Interaction:** Common operations require ≤ 3 prompts; smart defaults documented
- [ ] **UX Error Handling:** Error messages follow [ERROR] format
- [ ] **Description Optimization:** `description` field follows agentskills.io optimization principles
- [ ] **Eval Queries:** `assets/eval_queries.json` created with should/should-not trigger queries

#### Well-Architected + Three-Pillar (P0)
- [ ] **FinOps — Cost Visibility:** Billing model table present; cost attribution guidance documented
- [ ] **FinOps — Cost Optimization:** Idle resource detection pattern; right-sizing guidance present
- [ ] **FinOps — Unit Economics:** At least 1 unit cost metric defined (cost/request or cost/vCPU)
- [ ] **FinOps — Anomaly Detection:** Cost anomaly detection rule documented
- [ ] **SecOps — IAM Security:** Minimum IAM permissions table documented; credential masking enforced
- [ ] **SecOps — Network Security:** VPC/security group isolation guidance; encryption recommendations present
- [ ] **AIOps — Multi-Metric Correlation:** ≥ 4 anomaly patterns with detection logic (monitoring skills)
- [ ] **AIOps — Cross-Skill Delegation:** Delegation matrix defined in `integration.md` (diagnostic skills)
- [ ] **AIOps — Knowledge Base:** Fault pattern library present (diagnostic skills)
- [ ] **AIOps — SLO/SLI:** At least 1 SLO with SLI, Error Budget, and burn rate alerting defined
- [ ] **Five Pillars:** All five Well-Architected pillars addressed per well-architected-assessment.md
- [ ] **Well-Architected Reference:** SKILL.md links to well-architected-assessment.md section
- [ ] **Maturity Scorecard:** Self-assessment scorecard completed
- [ ] **Cross-Pillar Conflicts:** Trade-off matrix reviewed for known pillar conflicts

### P1 — SHOULD PASS

- [ ] **Chaining:** Stable output fields for downstream skills
- [ ] **Naming:** `huaweicloud-[product]-ops` consistent with repo conventions
- [ ] **Pinned** SDK/API baseline in integration.md
- [ ] **Idempotency** documented when automation applies
- [ ] **Adversarial scenarios** considered
- [ ] **FinOps — Right-Sizing:** Resource utilization → recommendation mapping
- [ ] **FinOps — Budget:** Budget alert integration documented
- [ ] **FinOps — Reserved Coverage:** RI/包年包月覆盖率 analysis template
- [ ] **FinOps — TCO Model:** Total Cost of Ownership model documented
- [ ] **SecOps — Threat Detection:** HSS/WAF integration trigger conditions
- [ ] **SecOps — Compliance:** Data protection alignment with industry standards
- [ ] **SecOps — Zero Trust:** Zero Trust Architecture alignment guidance
- [ ] **SecOps — Incident Response:** Security incident response runbook
- [ ] **SecOps — Supply Chain:** SDK dependency security + SBOM guidance
- [ ] **SecOps — Key Lifecycle:** KMS key lifecycle management strategy
- [ ] **AIOps — Proactive Inspection:** Scheduled巡检 workflow defined
- [ ] **AIOps — Alarm Storm:** Aggregation and suppression workflow
- [ ] **AIOps — Change Correlation:** CTS-based change-anomaly correlation
- [ ] **AIOps — Capacity Forecast:** 30-day capacity prediction methodology
- [ ] **AIOps — Diagnosis Confidence:** Confidence score with uncertainty declaration
- [ ] **Five Pillars — Multi-AZ:** Cross-AZ deployment recommendation
- [ ] **Five Pillars — DR Runbook:** Phase 1/2/3 structure
- [ ] **Five Pillars — Auto-Scaling:** Scaling trigger thresholds
- [ ] **Efficiency — IaC:** Terraform/Ansible integration template
- [ ] **Architecture — ADR:** Architecture Decision Records for key decisions
- [ ] **Self-Reflection:** Round 1 + Round 2 self-reflection completed during generation

---

## Example Request

> Add a Huawei Cloud skill for ECS in this repo: instances, disks, snapshots. Docs: `https://support.huaweicloud.com/api-ecs`. Go SDK (JIT fallback).

**Expected output:** `huaweicloud-ecs-ops` tree with **real** operationIds, Go SDK types, response paths, **and** matching CLI commands (primary path), plus JIT Go SDK fallback documentation.

---

## Reference Directory

| File | Purpose |
|------|---------|
| [huaweicloud-skill-template.md](references/huaweicloud-skill-template.md) | Base template for generated SKILL.md |
| [execution-environment.md](references/execution-environment.md) | CLI install, Go JIT download, credential config |
| [cli-behavior.md](references/cli-behavior.md) | Verified CLI behavioral notes |
| [enhanced-self-healing-framework.md](references/enhanced-self-healing-framework.md) | Self-healing patterns for installation flows |
| [governance-and-adversarial-review.md](references/governance-and-adversarial-review.md) | Adversarial review scenarios and governance checklist |
| [prompt-library.md](references/prompt-library.md) | Structured prompts for the generation lifecycle |
| [optimization-analysis.md](references/optimization-analysis.md) | Three-dimensional optimization framework |
| [user-experience-spec.md](references/user-experience-spec.md) | Mandatory UX requirements for all generated skills |
| [aiops-best-practices.md](references/aiops-best-practices.md) | Mandatory AIOps patterns for monitoring/diagnosis skills |
| [well-architected-assessment.md](references/well-architected-assessment.md) | **MANDATORY** Five-pillar + FinOps + SecOps + AIOps integration |

### External References

- [Huawei Cloud Go SDK](https://github.com/huaweicloud/huaweicloud-sdk-go-v3)
- [Huawei Cloud API Docs](https://support.huaweicloud.com/api/)
- [Huawei Cloud CLI (hcloud)](https://support.huaweicloud.com/hcli/index.html)
- [Agent Skills Open Specification](https://agentskills.io/specification)
- [Huawei Cloud Well-Architected Framework](https://support.huaweicloud.com/topic/68733-1-I)
