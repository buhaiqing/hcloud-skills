# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**hcloud-skills** is a Huawei Cloud Ops Skills collection — structured operational runbooks for AI agents to execute cloud operations via CLI/SDK. Each skill follows the [Agent Skill Open Specification](https://agentskills.io/specification).

## Core Architecture

### Three-Pillar Integration
Every skill embeds:
- **FinOps**: Cost visibility, waste detection, right-sizing, budget alerts
- **SecOps**: IAM least-privilege, encryption, threat detection integration
- **AIOps**: Anomaly patterns, multi-metric correlation, self-healing, alarm storm handling

### Skill Structure (Standard Layout)
```
huaweicloud-[product]-ops/
├── SKILL.md              # Main runbook: triggers, operations, error codes
├── references/
│   ├── core-concepts.md       # Product architecture
│   ├── api-sdk-usage.md       # API/SDK patterns
│   ├── cli-usage.md           # CLI command mappings
│   ├── troubleshooting.md     # Error recovery
│   ├── monitoring.md          # CES metrics
│   ├── well-architected-assessment.md  # Five-pillar + FinOps/SecOps/AIOps
│   └── aiops-best-practices.md  # (optional) ML-driven patterns
├── assets/
│   └── eval_queries.json      # Trigger test queries
```

### Key Conventions

**Placeholders**:
- `{{env.*}}` — Runtime environment (never ask user, fail if unset)
- `{{user.*}}` — User-provided inputs (instance IDs, names, specs)
- `{{output.*}}` — Captured API responses for subsequent operations

**Execution Flow**:
```
Pre-flight Checks → Execute → Validate → Recover
```

**Delegation Matrix**: Skills delegate cross-product operations:
- ECS → VPC (subnet creation), CES (metrics), ELB (load balancing)
- RDS → ECS (CloudShell), CES (performance metrics)
- All → IAM (permission issues), CTS (audit trails)

## Dual-Path Execution

**Primary**: `hcloud` CLI (official Huawei Cloud CLI)
```bash
hcloud ecs list-servers --region cn-north-4
hcloud ces create-alarm-rule --name "cpu-high" ...
```

**Fallback**: Go SDK (`huaweicloud-sdk-go-v3`) for unsupported CLI operations
```go
import "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2"
```

## Docker Sandbox

Run skills in isolated environment:
```bash
docker-compose build
docker-compose up hcloud-skills
# Inside container: check-env, skill-list, skill-read <name>
```

## Environment Variables

Required for execution:
- `HW_ACCESS_KEY_ID`
- `HW_SECRET_ACCESS_KEY`
- `HW_REGION_ID` (default: cn-north-4)
- `HW_PROJECT_ID` (optional, service-specific)

## Quality Gates (P0 Checklist)

Before any skill generation/update:
- [ ] SHOULD/SHOULD NOT triggers complete
- [ ] Pre-flight → Execute → Validate → Recover flow
- [ ] ≥10 product error codes with recovery strategies
- [ ] Destructive operations have safety gates
- [ ] IAM least-privilege permissions table
- [ ] FinOps cost optimization patterns
- [ ] AIOps anomaly detection patterns (≥4 types)

## Product-to-Skill Mapping

| Service | Skill | SDK Package |
|---------|-------|-------------|
| ECS | huaweicloud-ecs-ops | services/ecs/v2 |
| CES | huaweicloud-ces-ops | services/ces/v1 |
| VPC | huaweicloud-vpc-ops | services/vpc/v3 |
| RDS | huaweicloud-rds-ops | services/rds/v3 |
| ELB | huaweicloud-elb-ops | services/elb/v3 |
| CCE | huaweicloud-cce-ops | services/cce/v3 |
| IAM | huaweicloud-iam-ops | services/iam/v3 |
| OBS | huaweicloud-obs-ops | services/obs |
| DCS | huaweicloud-dcs-ops | services/dcs/v2 |
| LTS | huaweicloud-lts-ops | services/lts/v2 |

## Meta-Skill: Generator

`huaweicloud-skill-generator` scaffolds new skills from OpenAPI specs. Use when:
- Creating new `huaweicloud-[product]-ops`
- Updating existing skill after API changes
- Filling P0 gaps in existing skills

Reference template: `huaweicloud-skill-generator/references/huaweicloud-skill-template.md`