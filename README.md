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
├── huaweicloud-rds-ops/                  # RDS
├── huaweicloud-elb-ops/                  # ELB
├── huaweicloud-cce-ops/                  # CCE (Kubernetes)
├── huaweicloud-cts-ops/                  # CTS (audit)
├── huaweicloud-dms-ops/                  # DMS (Kafka/RabbitMQ)
├── huaweicloud-cbr-ops/                  # CBR (backup)
├── huaweicloud-swr-ops/                  # SWR (container registry)
├── huaweicloud-gaussdb-ops/              # GaussDB
├── huaweicloud-hss-ops/                  # HSS (host security)
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

## Available Skills

> Load these skills in an agent runtime for Huawei Cloud product operations.

| Skill | Product | Core capabilities | Status |
|-------|---------|-------------------|--------|
| `huaweicloud-billing-ops` | BSS (Billing) | Bills, cost analysis, budgets, optimization, maturity assessment | ✅ Ready |
| `huaweicloud-ces-ops` | CES (Cloud Eye) | Alarms, metrics, dashboards, events | ✅ Ready |
| `huaweicloud-vpc-ops` | VPC | VPC, subnets, security groups, EIP, NAT, peering | ✅ Ready |
| `huaweicloud-ecs-ops` | ECS | Instances, disks, snapshots, CloudShell | ✅ Ready |
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
| `huaweicloud-cbr-ops` | CBR | Vaults, policies, backup/restore, replication | ✅ Ready |
| `huaweicloud-swr-ops` | SWR | Orgs, repos, tags, retention, cross-region sync | ✅ Ready |
| `huaweicloud-hss-ops` | HSS | Hosts, vulnerabilities, baselines, tamper protection | ✅ Ready |
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
| Cloud Container Engine | CCE | `services/cce/v3` | Cluster, Node, Addon |
| Distributed Cache Service | DCS | `services/dcs/v2` | Instance, Backup, Resize |
| Host Security Service | HSS | `services/hss/v5` | Host, Vulnerability, Event |
| Web Application Firewall | WAF | `services/waf/v1` | Policy, Rule, Domain |
| Log Tank Service | LTS | `services/lts/v2` | Log Group, Stream, Search |
| Object Storage Service | OBS | `services/obs` | Bucket, Object, ACL |
| Identity and Access Management | IAM | `services/iam/v3` | User, Group, Policy, Agency |
| Distributed Message Service | DMS | `services/dms/v2` | Instance, Topic, Queue, Consumer Group |
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

Before pushing Python or skill changes:

```bash
python3 scripts/install_git_hook.py   # optional: install pre-commit hook
bash scripts/pre_commit_check.sh      # ruff + py310 + unit tests
python3 scripts/validate_local.py     # full local CI mirror
```

## References

- [Huawei Cloud Go SDK](https://github.com/huaweicloud/huaweicloud-sdk-go-v3)
- [Huawei Cloud API documentation](https://support.huaweicloud.com/api/)
- [Huawei Cloud CLI (hcloud)](https://support.huaweicloud.com/hcli/index.html)
- [Huawei Cloud Well-Architected Framework](https://support.huaweicloud.com/topic/68733-1-I)
- [Agent Skills Open Specification](https://agentskills.io/specification)
- [AGENTS.md](AGENTS.md) — repository conventions for agents and contributors
- [README_CN.md](README_CN.md) — 中文文档（详细目录树与说明）
