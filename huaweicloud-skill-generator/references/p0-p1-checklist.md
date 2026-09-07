---
name: huaweicloud-skill-generator-p0-p1-checklist
description: P0/P1 quality checklist for generated Huawei Cloud skills — MUST PASS (P0) and SHOULD PASS (P1) criteria
version: "1.0.0"
last_updated: "2026-09-08"
parent_skill: huaweicloud-skill-generator
---

# P0/P1 Checklist for Generated Skills

## P0 — MUST PASS

### Basic Requirements
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

### Well-Architected + Three-Pillar (P0)
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

## P1 — SHOULD PASS

### Quality & Governance
- [ ] **Chaining:** Stable output fields for downstream skills
- [ ] **Naming:** `huaweicloud-[product]-ops` consistent with repo conventions
- [ ] **Pinned** SDK/API baseline in integration.md
- [ ] **Idempotency** documented when automation applies
- [ ] **Adversarial scenarios** considered

### FinOps (P1)
- [ ] **FinOps — Right-Sizing:** Resource utilization → recommendation mapping
- [ ] **FinOps — Budget:** Budget alert integration documented
- [ ] **FinOps — Reserved Coverage:** RI/包年包月覆盖率 analysis template
- [ ] **FinOps — TCO Model:** Total Cost of Ownership model documented

### SecOps (P1)
- [ ] **SecOps — Threat Detection:** HSS/WAF integration trigger conditions
- [ ] **SecOps — Compliance:** Data protection alignment with industry standards
- [ ] **SecOps — Zero Trust:** Zero Trust Architecture alignment guidance
- [ ] **SecOps — Incident Response:** Security incident response runbook
- [ ] **SecOps — Supply Chain:** SDK dependency security + SBOM guidance
- [ ] **SecOps — Key Lifecycle:** KMS key lifecycle management strategy

### AIOps (P1)
- [ ] **AIOps — Proactive Inspection:** Scheduled巡检 workflow defined
- [ ] **AIOps — Alarm Storm:** Aggregation and suppression workflow
- [ ] **AIOps — Change Correlation:** CTS-based change-anomaly correlation
- [ ] **AIOps — Capacity Forecast:** 30-day capacity prediction methodology
- [ ] **AIOps — Diagnosis Confidence:** Confidence score with uncertainty declaration

### Five Pillars (P1)
- [ ] **Five Pillars — Multi-AZ:** Cross-AZ deployment recommendation
- [ ] **Five Pillars — DR Runbook:** Phase 1/2/3 structure
- [ ] **Five Pillars — Auto-Scaling:** Scaling trigger thresholds

### Efficiency & Architecture (P1)
- [ ] **Efficiency — IaC:** Terraform/Ansible integration template
- [ ] **Architecture — ADR:** Architecture Decision Records for key decisions
- [ ] **Self-Reflection:** Round 1 + Round 2 self-reflection completed during generation
