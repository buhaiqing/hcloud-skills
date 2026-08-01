# hcloud-skills

> **English** | **[中文](README_CN.md)**

Huawei Cloud operational Agent Skills collection for AI agents.

## Table of Contents

- [Overview](#overview)
- [Core Value](#core-value)
- [Project Structure](#project-structure)
- [Three-Pillar Ops](#three-pillar-ops)
  - [FinOps](#finops)
  - [SecOps](#secops)
  - [AIOps](#aiops)
- [Quick Start](#quick-start)
  - [1. Install Huawei Cloud CLI](#1-install-huawei-cloud-cli)
  - [2. Configure Credentials](#2-configure-credentials)
  - [3. Use Existing Skills](#3-use-existing-skills)
  - [4. Generate a New Skill](#4-generate-a-new-skill)
- [Available Skills](#available-skills)
- [Huawei Cloud Service Mapping](#huawei-cloud-service-mapping)
- [Generation Quality Gates](#generation-quality-gates)
- [References](#references)

## Overview

This repository is a **generator and collection** of Huawei Cloud operational Agent Skills. It provides automation for cloud operations, monitoring, cost management, security governance, and intelligent diagnostics.

> **Skills Farm is a Meta Skill system** — operational knowledge is expressed as structured, agent-parseable, executable, and verifiable declarative runbooks.

## Core Value

| Feature | Description |
|---------|-------------|
| **Three-pillar integration** | FinOps + SecOps + AIOps embedded in every skill |
| **Placeholder model** | `{{env.*}}` (runtime), `{{user.*}}` (user input), `{{output.*}}` (API capture) |
| **Delegation** | `SHOULD/SHOULD NOT Use` boundaries with cross-product delegation |
| **Generator** | Scaffold skills from OpenAPI specs; human review completes gaps |
| **CLI-first execution** | Primary path: `hcloud` CLI; JIT Go SDK fallback when CLI lacks coverage |
| **Safety** | Credential isolation (`{{env.*}}` never collected from user); destructive-op gates |
| **Well-Architected** | Five pillars (Security, Stability, Cost, Efficiency, Performance) + FinOps + SecOps + AIOps |

## Project Structure

```
hcloud-skills/
├── README.md
├── README_CN.md                          # Chinese README (linked from README.md)
├── LICENSE
├── huaweicloud-billing-ops/              # Billing (FinOps)
│   ├── SKILL.md
│   ├── references/
│   └── assets/
├── huaweicloud-skill-generator/          # Meta skill (generator)
│   ├── SKILL.md
│   ├── assets/
│   └── references/
├── huaweicloud-ces-ops/                  # Cloud Eye (CES)
├── huaweicloud-vpc-ops/                  # VPC
├── huaweicloud-iam-ops/                  # IAM
├── huaweicloud-dcs-ops/                  # DCS (Redis)
├── huaweicloud-obs-ops/                  # OBS
├── huaweicloud-ecs-ops/                  # ECS
├── huaweicloud-eip-ops/                  # EIP (Elastic IP / Bandwidth)
├── huaweicloud-rds-ops/                  # RDS
├── huaweicloud-elb-ops/                  # ELB
├── huaweicloud-cce-ops/                  # CCE (Kubernetes)
├── huaweicloud-cts-ops/                  # CTS (audit)
├── huaweicloud-dms-ops/                  # DMS (Kafka/RabbitMQ)
├── huaweicloud-dns-ops/                   # DNS (云解析DNS)
├── huaweicloud-cbr-ops/                  # CBR (backup)
├── huaweicloud-cdn-ops/                   # CDN (Content Delivery Network)
├── huaweicloud-swr-ops/                  # SWR (container registry)
├── huaweicloud-gaussdb-ops/              # GaussDB
├── huaweicloud-hss-ops/                  # HSS (host security)
├── huaweicloud-kms-ops/                  # KMS (Key Management)
├── huaweicloud-waf-ops/                  # WAF
├── huaweicloud-lts-ops/                   # LTS (logging)
├── huaweicloud-css-ops/                   # CSS (Cloud Search Service)
└── huaweicloud-functiongraph-ops/        # FunctionGraph (serverless)
```

Each `huaweicloud-*-ops/` skill follows the same layout:

```
huaweicloud-[product]-ops/
├── SKILL.md                    # Entry runbook: triggers, flows, recovery
├── references/                 # Deep reference files
│   ├── core-concepts.md
│   ├── api-sdk-usage.md
│   ├── cli-usage.md
│   ├── troubleshooting.md
│   ├── monitoring.md
│   ├── integration.md
│   ├── well-architected-assessment.md
│   ├── rubric.md               # GCL rubric (Tier-A skills)
│   └── prompt-templates.md     # GCL prompt templates
└── assets/
    ├── eval_queries.json
    └── example-config.yaml
```

See [README_CN.md](README_CN.md) for a **detailed per-skill directory tree** with Chinese annotations.

## Three-Pillar Ops

### FinOps

| Capability | Description |
|------------|-------------|
| Cost visibility | Billing model comparison (pay-as-you-go vs reserved vs spot), tagging, cost-center integration |
| Cost optimization | Idle resource detection, right-sizing matrix, lifecycle cost management |
| Cost accountability | Budget alerts (80% / 90% / 100% thresholds), chargeback |

### SecOps

| Capability | Description |
|------------|-------------|
| Identity | IAM least privilege, AK/SK rotation (90 days), MFA, agency credentials |
| Network | VPC endpoint isolation, security group patterns, DDoS protection |
| Data | KMS encryption, TDE, audit logs (≥180 days), DLP |
| Threat detection | HSS integration, WAF linkage, vulnerability scanning |

### AIOps

| Capability | Description |
|------------|-------------|
| Multi-metric correlation | ≥4 anomaly patterns (pressure, trend, spike, correlation) |
| Cross-skill diagnosis | Namespace → primary/secondary skill routing matrix |
| Knowledge base | ≥3 product fault patterns + ≥2 cross-product cascade patterns |
| Alarm storm handling | Rate detection, aggregation, root-resource identification |
| Proactive inspection | Discover → collect → detect → diagnose → report loop |
| Self-healing | Pre-check → download → install → verify with graceful degradation |

## Quick Start

### 1. Install Huawei Cloud CLI

**One-click install (Linux):**

```bash
curl -sSL https://cn-north-4.myhuaweicloud.com/cli/latest/hcloud_install.sh -o ./hcloud_install.sh && bash ./hcloud_install.sh -y
```

**macOS:**

```bash
curl -sSL https://ap-southeast-3-hwcloudcli.obs.ap-southeast-3.myhuaweicloud.com/cli/latest/hcloud_install.sh -o ./hcloud_install.sh && bash ./hcloud_install.sh -y
```

**Verify:**

```bash
hcloud version
# Current KooCLI version: 4.1.6
```

### 2. Configure Credentials

```bash
export HW_ACCESS_KEY_ID="your-access-key-id"
export HW_SECRET_ACCESS_KEY="your-secret-access-key"
export HW_REGION_ID="cn-north-4"
export HW_PROJECT_ID="your-project-id"
```

### 3. Use Existing Skills

**CES (monitoring)** — load `huaweicloud-ces-ops`:

```
"Create an alarm when ECS CPU exceeds 80% and notify via SMS"
"Query CPU metrics for instance i-abc123 over the last hour"
"List all alarm rules in cn-north-4"
```

**BSS (billing)** — load `huaweicloud-billing-ops`:

```
"Show this month's bill summary grouped by product"
"Find idle ECS instances with CPU < 5% over the last 30 days"
"Create a budget alert at 80% of monthly spend"
"Recommend which pay-as-you-go instances should move to reserved pricing"
```

**VPC (networking)** — load `huaweicloud-vpc-ops`:

```
"Create a VPC 10.0.0.0/16 with a production subnet"
"Add a security group rule allowing 10.0.0.0/8 to port 22"
"Allocate an EIP and bind it to an ECS instance"
"Configure a NAT gateway for private subnet egress"
```

### 4. Generate a New Skill

Load the generator meta-skill in your agent runtime, then prompt:

> "Generate huaweicloud-ecs-ops for instance lifecycle, disks, and snapshots. Include FinOps cost patterns and SecOps IAM guidance."

See [huaweicloud-skill-generator/SKILL.md](huaweicloud-skill-generator/SKILL.md) and [AGENTS.md](AGENTS.md) for quality gates and workflow.

## hwcloud-skillcheck — Skill Repository Validator

`hwcloud-skillcheck` is a **standalone CLI binary** that validates a hcloud-skills repository (or any skill collection following the same layout). It runs all A-class checks (schema validation, frontmatter, YAML config, markdown links, secret scanning, etc.) with **zero external dependencies** — no Python, no Go toolchain required.

### Quick Install

```bash
# Latest release tag
VERSION=$(curl -sSL https://api.github.com/repos/buhaiqing/hcloud-skills/releases/latest | grep -m1 '"tag_name"' | sed -E 's/.*"v([^"]+)".*/v\1/')

# Linux (amd64) — replace with your platform
curl -sSLO "https://github.com/buhaiqing/hcloud-skills/releases/download/${VERSION}/hwcloud-skillcheck-linux-amd64"
chmod +x hwcloud-skillcheck-linux-amd64
sudo mv hwcloud-skillcheck-linux-amd64 /usr/local/bin/hwcloud-skillcheck

# Verify
hwcloud-skillcheck --help
```

See [Deployment Guide](docs/deployment-guide.md) for:
- All 6 platform binary URLs
- Docker sandbox deployment
- CI/CD integration
- Release process
- Operations checklist
> See [Deployment Guide](docs/deployment-guide.md) for macOS (Intel/Apple Silicon) and Windows (amd64/arm64) install commands.

### Build From Source

```bash
git clone https://github.com/buhaiqing/hcloud-skills.git
cd hcloud-skills
go install github.com/go-task/task/v3/cmd/task@latest
task build          # writes ./hwcloud-skillcheck/bin/hwcloud-skillcheck
task test           # full Go test suite
task release VERSION=X.Y.Z    # cut a release (pushes tag, triggers CI artifacts)
```

### Verify Installation

```bash
hwcloud-skillcheck --help
# Expected output: hwcloud-skillcheck — cross-platform hcloud-skills validator
```

### Unified Pre-commit Gate (Phase 5)

`hwcloud-skillcheck check --pre-commit` is the **single source of truth** for
repository quality gates. The local git hook (`.githooks/pre-commit`) and CI
(`.github/workflows/validate-skills.yml`) both invoke it — running it locally
is equivalent to pushing. It replaces the deleted shell-based pre-commit
checker (ADR-0014).

```bash
# Run all 13 gates locally (git-hook mode skips go test via --skip-tests)
hwcloud-skillcheck check --pre-commit --root .

# CI mode: build binary first, then run with --check-only (adds 3 CI-only soft
# gates: drift sync --dry-run, gcl alarm-wire, critic-score) + test retry loop
hwcloud-skillcheck check --pre-commit --check-only --test-retries 2 --root .
```

The gates in order: gofmt, go vet, validate (frontmatter + eval-queries +
product-assessment + markdown-links + example-config + advanced-coverage),
audit-results, aggregate trace, learning gen, l4 handle smoke, golden run
(soft), check lanes, ab compare (soft), advanced-coverage, drift guard, go
test, and (CI-only) drift sync --dry-run / gcl alarm-wire / critic-score.

> **Note**: `--check-only` skips the binary rebuild (CI builds it separately)
> and runs `drift sync --apply` + `drift check` to bootstrap the gitignored
> `.agents/skills/` runtime copy on a fresh checkout.

### Quick Usage

```bash
# Validate your skill repository (default --root = current directory)
hwcloud-skillcheck validate

# Point to an external skill repo
hwcloud-skillcheck validate --root ./my-skills

# Run specific checks
hwcloud-skillcheck check markdown-links --root .
hwcloud-skillcheck scan secret trace --self-check

# Parallel-execution collection gate: validate cross-file consistency across
# every file changed by parallel subagents + emit a per-agent log
hwcloud-skillcheck check merge-verify --log <parallel-log.json> --root .
```

### Router Policy & Calibration (P2)

From P2 efficiency spec §4.2, the Skill Router reads its decision parameters from a
versioned registry file at runtime; the binary carries no setter for them
(rubric A2.14, S3).

```bash
# The file ships in the repo at hwcloud-skillcheck/capability-registry.json
export HC_CAPABILITY_REGISTRY="$(pwd)/hwcloud-skillcheck/capability-registry.json"

# Inspect current policy:
hwcloud-skillcheck router info --root .

# Validate the default local sandbox and run one embedding smoke call:
hwcloud-skillcheck router embed-test --root . --text "list ecs servers"

# Cloud is opt-in. Preflight checks endpoint, credential env, dimension and timeout
# before any network request and prints a concrete Fix for every problem:
HC_EMBED_PROVIDER=huaweicloud-modelarts \
  hwcloud-skillcheck router embed-test --root . --text "list ecs servers"

# Calibration (offline only, dry-run by default — rubric A2.13):
hwcloud-skillcheck router calibrate --root .                              # dry-run (default)
hwcloud-skillcheck router calibrate --root . --apply                      # commit a bump (default: patch)
hwcloud-skillcheck router calibrate --root . --apply --bump minor         # structural change
hwcloud-skillcheck router calibrate --root . --apply --rollback-to v1.0.0 # revert to v1.0.0
hwcloud-skillcheck router calibrate --root . --source audit-results/      # trace-derived suggestions (advisory)
```

The default embedding sandbox is `local-fasttext`: pure Go, in-process, no network or
credential setup. `huaweicloud-modelarts` is an explicit cloud switch configured by the
registry or `HC_EMBED_PROVIDER`; every provider runs `Preflight()` before initialization.
ONNX remains an optional future provider blocked on vendored native runtime assets, not a
runtime prerequisite. See `docs/deployment-guide.md` §1.6 for configuration and friendly
preflight troubleshooting, and the P2 design §4.2.4 for provider semantics. The Router supports three provider modes: local (`local-fasttext`, the default), cloud (`huaweicloud-modelarts` for Huawei Cloud ModelArts), and off (`none` for environments that must skip Stage-2 rerank). `fallback_chain` is honoured at runtime; when the primary provider fails, the Router falls back and records `fallback_used=true` plus the active provider in trace metadata.

### B-Class Validation Commands (GCL Contract Checks)

These commands validate Generator-Critic-Loop (GCL) artifacts and runtime contracts:

```bash
# Validate GCL Tier-A conformance (rubric.md, prompt-templates.md, Quality Gate section)
hwcloud-skillcheck validate gcl-conformance --root .

# Validate Generator template contract (skill-generator GCL artifacts)
hwcloud-skillcheck validate generator-contract --root .

# Validate safety_class enum contract across the pipeline
hwcloud-skillcheck validate safety-class --root .

# Validate resource_scope PII masking contract
hwcloud-skillcheck validate resource-scope --root .

# Validate CES alarm wiring contract (thresholds consistency)
hwcloud-skillcheck validate alarm-wire-contract --root .

# Validate audit-results directory protection (gitignore, permissions)
hwcloud-skillcheck check audit-results --root .

# Detect drift between canonical skill-generator and the agent runtime copy
# (fails the gate if they diverge; sync first to repair)
hwcloud-skillcheck drift check --root .

# Repair drift by copying the canonical skill-generator to the runtime location
hwcloud-skillcheck drift sync --apply --root .

# Run the rule-based Critic scorer (5-dimension quality scoring for GCL)
hwcloud-skillcheck critic score --generator /path/to/generator-trace.json
```

### GCL Runtime Commands

```bash
# Execute a GCL cycle (Generator → Critic → loop)
hwcloud-skillcheck gcl run \
  --skill huaweicloud-billing-ops \
  --request "CI smoke test" \
  --operation-intent '{"operation":"smoke","resource_scope":[],"expected_state":"no-op","safety_class":"read-only"}' \
  --command 'printf "{\"Response\":{\"RequestId\":\"ci-smoke\"}}"' \
  --max-iter 1 \
  --structural-critic-only

# Plan CES alarm rules from a GCL quality summary
hwcloud-skillcheck gcl alarm-wire plan \
  --summary scripts/fixtures/gcl-quality-summary-healthy.json \
  --write-plan

# Apply planned CES alarm rules (requires --dry-run first)
hwcloud-skillcheck gcl alarm-wire apply \
  --summary scripts/fixtures/gcl-quality-summary-healthy.json \
  --dry-run
```

See [hwcloud-skillcheck CLI Spec](docs/superpowers/specs/hwcloud-skillcheck-cli.md) for the full command reference.

## hwcloud-skillcheck CLI Reference

Complete reference for all `hwcloud-skillcheck` subcommands added in Phase 2 and Phase 3 of the B-class migration.

### validate

| Subcommand | Description |
|------------|-------------|
| `validate gcl-conformance --root <dir>` | Validates GCL Tier-A conformance: checks `references/rubric.md` (8 sections), `references/prompt-templates.md` (7 sections + `operation_intent` + no bare placeholders), and `## Quality Gate (GCL)` in `SKILL.md`. Exit 0 = all pass, 1 = failures. |
| `validate generator-contract --root <dir>` | Validates the skill-generator GCL template contract: checks `huaweicloud-skill-template.md` (metadata.gcl.required, rubric artifact, prompt templates artifact, operation_intent), `huaweicloud-skill-generator/SKILL.md` (references backbone, rubric, prompt-templates), and `gcl-prompt-backbone.md` (Generator/Critic/Orchestrator sections, hcloud primary, Go SDK fallback, Critic read-only constraint). Supports `--json` for JSON report. |
| `validate safety-class --root <dir>` | Validates `safety_class` enum contract across the pipeline: schema gate (`gcl-trace.schema.json`), code gate (sanitizer rejects illegal values), docs gate (`docs/gcl-spec.md`, `gcl-prompt-backbone.md`), and traces gate (no illegal enum values in `audit-results/gcl-trace-*.json`). Expected values: `read-only`, `mutating`, `destructive`. Exit 0 = all pass, 1 = failures. |
| `validate resource-scope --root <dir>` | Validates `operation_intent.resource_scope` PII masking contract: schema gate (allows `***` / `<masked>` / `prefix-***`), code gate (`maskResourceID()` function), runner gate (`masked_fields` includes `operation_intent`), and traces gate (no bare IDs in traces). Exit 0 = all pass, 1 = failures. |
| `validate alarm-wire-contract --root <dir>` | Validates CES alarm wire threshold wiring consistency: checks `huaweicloud-ces-ops/assets/example-config.yaml` (gcl_quality block), `docs/gcl-spec.md` (threshold descriptions), and `audit-results/gcl-alarm-plan-*.json` (existing alarm plans). Hardcoded threshold constants: `pass_rate_warn=0.5`, `pass_rate_critical=0.3`, `max_iter_warn_count=5`, `safety_fail_alert=true`. Exit 0 = all pass, 1 = inconsistencies found. |

### check

| Subcommand | Description |
|------------|-------------|
| `check audit-results --root <dir>` | Validates `audit-results/` directory protection contract: `.gitignore` must contain 8 required patterns (audit-results/, */audit-results/, gcl-trace-*.json, */gcl-trace-*.json, gcl-quality-summary-*.json, */gcl-quality-summary-*.json, gcl-alarm-plan-*.json, */gcl-alarm-plan-*.json), directory mode must be 0700 when present, no tracked files in git. Supports `--json` for JSON report. Exit 0 = clean, 1 = contract violations. |
| `check merge-verify --log <path> [--root <dir>]` | Parallel-execution collection gate. Parses a log of `[{agent, files[], status, result}]`, validates cross-file markdown-link / backtick-path targets across every changed file, and emits a per-agent report. Fails (exit 1) when any subagent reports `status: failed` or a broken reference is found. |

### drift

| Subcommand | Description |
|------------|-------------|
| `drift check --root <dir> [--json]` | Verifies byte-for-byte equality between `huaweicloud-skill-generator/` (canonical, git-tracked) and `.agents/skills/huaweicloud-skill-generator/` (runtime, gitignored). Catches silent drift that would cause the agent runtime to load stale instructions. Exit 0 = identical, 1 = drift or missing root. |
| `drift sync --root <dir> [--apply] [--dry-run]` | Reconciles the runtime copy from canonical. `--dry-run` (default) prints actions without writing; `--apply` performs copy/overwrite/delete. Bootstraps the runtime directory on demand, so it is self-healing on fresh CI checkouts. Exit 0 = synced. |

### critic

| Subcommand | Description |
|------------|-------------|
| `critic score --generator <path> [--emit] [--critic-out <path>]` | Runs the rule-based 5-dimension Critic (safety, correctness, idempotency, traceability, spec_compliance) against a generator trace JSON. Output schema: `scores` (5 floats ∈ {0, 0.5, 1}), `suggestions` (≤5 strings), `blocking` (bool), `_mode` ("critic-v1"), `finops_estimate_usd`. `--emit` also writes the result to `--critic-out`. Exit 0 = always (validation lives in `gcl run`). |

### gcl

| Subcommand | Description |
|------------|-------------|
| `gcl run --root <dir> [--json] [--quiet]` | Executes GCL quality gate on a skill: runs Generator command, collects stdout/stderr/exit, calls Critic scoring, loops修复 (max `max_iter` iterations), writes trace to `audit-results/gcl-trace-YYYYMMDD-HHMMSS.json`. Exit codes: 0 = PASS, 1 = MAX_ITER/ERROR, 2 = SAFETY_VIOLATION, 124 = TIMEOUT. |
| `gcl alarm-wire --root <dir> [--json] [--quiet] [--plan-file <path>]` | Evaluates GCL trace quality against SLO thresholds and optionally generates/applies CES alarm plan. Loads CES `example-config.yaml` thresholds, finds most recent trace in `audit-results/`, evaluates breaches, renders alarm plan. `--plan-file <path>` writes plan JSON and applies (dry-run by default). Exit 0 = no critical breaches, 1 = alerts/warnings. |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All checks passed / PASS |
| 1 | Failures found / MAX_ITER / ERROR |
| 2 | SAFETY_VIOLATION |
| 124 | TIMEOUT (matches `timeout` command) |

### Output Modes

All commands support `--json` for machine-readable JSON output. The `validate` total-entry also supports `--json` for a combined summary.

## Available Skills

> Load these skills in an agent runtime for Huawei Cloud product operations.

| Skill | Product | Core capabilities | Status |
|-------|---------|-------------------|--------|
| `huaweicloud-billing-ops` | BSS (Billing) | Bills, cost analysis, budgets, optimization, maturity assessment | ✅ Ready |
| `huaweicloud-ces-ops` | CES (Cloud Eye) | Alarms, metrics, dashboards, events | ✅ Ready |
| `huaweicloud-vpc-ops` | VPC | VPC, subnets, security groups, EIP, NAT, peering | ✅ Ready |
| `huaweicloud-ecs-ops` | ECS | Instances, disks, snapshots, CloudShell | ✅ Ready |
| `huaweicloud-eip-ops` | EIP (Elastic IP) | Allocate, bind/unbind, release EIP; bandwidth resize; shared bandwidth; 95th-percentile billing; idle EIP detection | ✅ Ready |
| `huaweicloud-rds-ops` | RDS | Instances, backup/restore, parameters, performance | ✅ Ready |
| `huaweicloud-elb-ops` | ELB | Listeners, pools, health checks | ✅ Ready |
| `huaweicloud-cce-ops` | CCE | Clusters, nodes, node pools, add-ons | ✅ Ready |
| `huaweicloud-dcs-ops` | DCS | Redis lifecycle, backup, resize, whitelist | ✅ Ready |
| `huaweicloud-cts-ops` | CTS | Audit trails, trace query, diagnostics | ✅ Ready |
| `huaweicloud-css-ops` | CSS (Cloud Search Service) | Elasticsearch/OpenSearch clusters, snapshots, dictionaries, config | ✅ Ready |
| `huaweicloud-functiongraph-ops` | FunctionGraph | Functions, triggers, versions, diagnostics | ✅ Ready |
| `huaweicloud-iam-ops` | IAM | Users, groups, policies, agencies, AK/SK, MFA | ✅ Ready |
| `huaweicloud-obs-ops` | OBS | Buckets, objects, ACL, lifecycle, CDN, static site | ✅ Ready |
| `huaweicloud-dms-ops` | DMS | Kafka/RabbitMQ, topics/queues, consumer groups | ✅ Ready |
| `huaweicloud-dns-ops` | DNS (云解析DNS) | Zones, recordsets, DNSSEC, PTR reverse DNS | ✅ Ready |
| `huaweicloud-cbr-ops` | CBR | Vaults, policies, backup/restore, replication | ✅ |
| `huaweicloud-cdn-ops` | CDN | Domain lifecycle, cache rules, refresh/preheat, HTTPS, statistics | ✅ |Ready |
| `huaweicloud-swr-ops` | SWR | Orgs, repos, tags, retention, cross-region sync | ✅ Ready |
| `huaweicloud-hss-ops` | HSS | Hosts, vulnerabilities, baselines, tamper protection | ✅ Ready |
| `huaweicloud-kms-ops` | KMS (Key Management) | CMK lifecycle, grants, BYOK, data keys, key rotation | ✅ Ready |
| `huaweicloud-waf-ops` | WAF | Policies, rules, domains, certificates, events | ✅ Ready |
| `huaweicloud-lts-ops` | LTS | Log groups/streams, search, transfer, alarms | ✅ Ready |
| `huaweicloud-gaussdb-ops` | GaussDB | Instances, backup, templates, DB/users, recycle bin | ✅ Ready |

## Huawei Cloud Service Mapping

| Service | Abbr. | Go SDK package | Primary operations |
|---------|-------|----------------|-------------------|
| Elastic Cloud Server | ECS | `services/ecs/v2` | Create, Delete, Describe, Resize |
| Relational Database Service | RDS | `services/rds/v3` | Instance, Backup, Restore |
| Cloud Eye Service | CES | `services/ces/v1` | Alarm, Metric, Dashboard |
| Virtual Private Cloud | VPC | `services/vpc/v3` | VPC, Subnet, SecurityGroup |
| Elastic Load Balance | ELB | `services/elb/v3` | Listener, Pool, Health |
| Elastic IP | EIP | `services/eip/v2` | Publicip, Bandwidth, Shared Bandwidth |
| Cloud Container Engine | CCE | `services/cce/v3` | Cluster, Node, Addon |
| Distributed Cache Service | DCS | `services/dcs/v2` | Instance, Backup, Resize |
| Host Security Service | HSS | `services/hss/v5` | Host, Vulnerability, Event |
| Web Application Firewall | WAF | `services/waf/v1` | Policy, Rule, Domain |
| Log Tank Service | LTS | `services/lts/v2` | Log Group, Stream, Search |
| Object Storage Service | OBS | `services/obs` | Bucket, Object, ACL |
| Identity and Access Management | IAM | `services/iam/v3` | User, Group, Policy, Agency |
| Key Management Service | KMS | `services/kms/v2` | Key, Grant, KeyMaterial |
| Content Delivery Network | CDN | `services/cdn/v1` | Domain, Cache, Refresh, Preheat |

| Distributed Message Service | DMS | `services/dms/v2` | Instance, Topic, Queue, Consumer Group |
| Domain Name Service | DNS | `services/dns/v2` | Zone, RecordSet, DNSSEC |
| Cloud Backup and Recovery | CBR | `services/cbr/v3` | Vault, Policy, Backup, Restore |
| Cloud Search Service | CSS | `services/css/v1` | Cluster, Snapshot, Dictionary, Config |
| Software Repository for Containers | SWR | `services/swr/v2` | Organization, Repository, Image |
| GaussDB | GaussDB | `services/gaussdb/v3` | Instance, Backup, Template, Database/User |

## Generation Quality Gates

Every generated skill must pass the **P0 checklist**:

### Foundation

- [ ] Complete SHOULD / SHOULD NOT Use triggers
- [ ] Pre-flight → Execute → Validate → Recover for each operation
- [ ] ≥10 product error codes with recovery strategies
- [ ] Safety gates for destructive operations
- [ ] Credential masking (`***`)

### FinOps

- [ ] Billing model comparison table
- [ ] Idle resource detection pattern
- [ ] Right-sizing matrix (utilization → recommendation)
- [ ] Cost tagging strategy

### SecOps

- [ ] IAM least-privilege table
- [ ] VPC / security group isolation guidance
- [ ] Encryption at rest and in transit documented
- [ ] HSS / WAF threat-detection triggers

### AIOps

- [ ] ≥4 anomaly patterns with detection logic
- [ ] Cross-skill delegation matrix
- [ ] Fault pattern knowledge base
- [ ] Alarm storm aggregation workflow

### GCL (runtime quality gate)

Tier-A skills also ship Generator-Critic-Loop artifacts: `references/rubric.md`, `references/prompt-templates.md`, and `## Quality Gate (GCL)` in `SKILL.md`. See [docs/gcl-spec.md](docs/gcl-spec.md) and [AGENTS.md](AGENTS.md).

### Local validation

Before pushing changes:

```bash
hwcloud-skillcheck validate
hwcloud-skillcheck check audit-results --root .
hwcloud-skillcheck check --pre-commit --root .   # unified gate (ADR-0014)
```

**Note**: A-class validation scripts have been migrated to the `hwcloud-skillcheck` Go binary.
Run `cd hwcloud-skillcheck && go test ./...` for the full test suite.

### Build & release (hwcloud-skillcheck)

Build & test from source:

```bash
cd hwcloud-skillcheck
go build ./...
go vet ./...
go test ./...
```

Cross-platform release workflows (via Taskfile.yml):

```bash
# 1. Tag + push → CI builds + publishes (recommended; fully automated)
task release VERSION=0.3.0

# 2. Local cross-platform build → ./dist/ (5 platforms + SHA256SUMS)
task release-build VERSION=0.3.0

# 3. Local build + create GitHub Release via gh CLI (requires `gh auth login`)
task release-local VERSION=0.3.0
```

Manual fallback (single-platform, no Taskfile required):

```bash
cd hwcloud-skillcheck
GOOS=linux   GOARCH=amd64 go build -o hwcloud-skillcheck-linux-amd64   .
GOOS=linux   GOARCH=arm64 go build -o hwcloud-skillcheck-linux-arm64   .
GOOS=darwin  GOARCH=amd64 go build -o hwcloud-skillcheck-darwin-amd64  .
GOOS=darwin  GOARCH=arm64 go build -o hwcloud-skillcheck-darwin-arm64  .
GOOS=windows GOARCH=amd64 go build -o hwcloud-skillcheck-windows-amd64.exe .
```

The CI workflow (`.github/workflows/build-skillcheck.yml`) runs automatically on
`v*` tag push, builds a 6-platform matrix, and uploads the binaries to the
GitHub Release via `softprops/action-gh-release`.

## References

- [Huawei Cloud Go SDK](https://github.com/huaweicloud/huaweicloud-sdk-go-v3)
- [Huawei Cloud API documentation](https://support.huaweicloud.com/api/)
- [Huawei Cloud CLI (hcloud)](https://support.huaweicloud.com/hcli/index.html)
- [Huawei Cloud Well-Architected Framework](https://support.huaweicloud.com/topic/68733-1-I)
- [Agent Skills Open Specification](https://agentskills.io/specification)
- [AGENTS.md](AGENTS.md) — repository conventions for agents and contributors
- [README_CN.md](README_CN.md) — 中文文档（详细目录树与说明）
- [Deployment Guide](docs/deployment-guide.md) — installation, operations, CI/CD, security baseline
- [CLI Manual](docs/manual/hwcloud-skillcheck.md) — full command reference
