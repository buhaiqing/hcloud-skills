# SPEC: AIOps L4 — Maturity Optimization

> Version: 1.0.0
> Created: 2026-07-26
> Status: **FINAL_SPEC** — 16 Batch + 5 Gap Filling all merged to main (2026-07-18)
> Target: hcloud-skills AIOps L4 maturity (80% criteria, ~70% overall)
> Prerequisite: `docs/superpowers/specs/gcl-token-efficiency-p0.md`

## 1. Overview

### 1.1 Vision

Raise AIOps maturity from ~45% (L2) to 80%+ (L4) across all `huaweicloud-*-ops` skills. This covers P0 (L3 foundation: SLO, change correlation, capacity forecasting) and P1 (L4 capabilities: chaos engineering, resilience scoring, observability trinity).

### 1.2 AIOps Maturity Model

| Level | Characteristic | Target |
|-------|---------------|--------|
| L1 | ≥4 anomaly patterns per skill | ✅ Achieved |
| L2 | Delegation matrix + knowledge base + prompt handbook | ✅ Achieved (100%) |
| L3 | SLO/SLI + change correlation + capacity forecasting | ✅ Achieved (100%) |
| **L4** | **Chaos engineering + resilience scoring + observability trinity** | **✅ Achieved (this spec)** |
| L5 | Autonomous healing | See `aiops-l5-autonomous.md` |

### 1.3 Scope

1. **Gap Filling** (Phase 0): 9 missing L2/L3 files across RDS, ELB, CCE, CES
2. **L3 Foundation** (Phase 1, P0 items): SLO standardization, change correlation, capacity forecasting, diagnosis confidence, L1 skill enhancement
3. **L4 Capabilities** (Phase 2, P1 items): Chaos engineering, resilience scoring, observability trinity, prompt handbook, trend detection
4. **Cascade Patterns** (Phase 3, P1-1): Cross-product cascading failure patterns

### 1.4 Dependencies

- `gcl-token-efficiency-p0.md`: Completed the TE-7 professional content layering that this optimization builds on
- L2 foundations per skill (knowledge-base.md, prompts.md, integration.md) — baseline before L3/L4 work

---

## 2. Implementation Summary

### 2.1 Phase 0 — Gap Filling (9 files)

| Gap | Skill | Level | Files Created | Status |
|-----|-------|-------|--------------|--------|
| G-1 | RDS | L2 | `knowledge-base.md`, `prompts.md` | ✅ |
| G-2 | ELB | L2 | `knowledge-base.md`, `prompts.md` | ✅ |
| G-3 | CCE | L3 | `advanced/observability-trinity.md` | ✅ |
| G-4 | CES | L3 | `advanced/change-correlation.md`, `advanced/capacity-forecasting.md` | ✅ |
| G-5 | ELB | L3 | `advanced/change-correlation.md`, `advanced/capacity-forecasting.md` | ✅ |

### 2.2 Phase 1 — L3 Foundation (P0 Items, 40 files)

| Batch | P0 Item | Files | Count | Status |
|-------|---------|-------|-------|--------|
| A | P0-3 Delegation Matrix Template | `huaweicloud-skill-generator/references/cross-skill-delegation-matrix-template.md` | 1 | ✅ |
| B | P0-8 SLO/SLI (10 skills) | `huaweicloud-*-ops/references/well-architected-assessment.md` | 10 | ✅ |
| C | P0-9 Change Correlation (3 skills) | `huaweicloud-*-ops/references/advanced/change-correlation.md` | 3 | ✅ |
| D | P0-10 Capacity Forecasting (5 skills) | `huaweicloud-*-ops/references/advanced/capacity-forecasting.md` | 5 | ✅ |
| E | P1-5 Diagnosis Confidence | template + CES reference | 2 | ✅ |
| F.1 | OBS AIOps Enhancement | aiops-patterns, knowledge-base, integration, alarm-storm | 4 | ✅ |
| F.2 | LTS AIOps Enhancement | aiops-patterns, knowledge-base, integration, alarm-storm | 4 | ✅ |
| F.3 | CBR AIOps Enhancement | aiops-patterns, knowledge-base, integration, alarm-storm | 4 | ✅ |
| F.4 | SWR AIOps Enhancement | aiops-patterns, knowledge-base, integration, alarm-storm | 4 | ✅ |
| F.5 | CTS AIOps Enhancement | aiops-patterns, knowledge-base, alarm-storm | 3 | ✅ |

### 2.3 Phase 2 — L4 Capabilities (P1 Items, 33 files)

| Batch | P1 Item | Files | Count | Status |
|-------|---------|-------|-------|--------|
| G | P1-8 Chaos Engineering (5 skills) | template + ECS/CCE/RDS/CES/ELB | 6 | ✅ |
| H | P1-9 Resilience Score (5 skills) | template + ECS/CCE/RDS/DCS/VPC | 6 | ✅ |
| I | P1-2 Observability Trinity (3 skills) | template + ECS/CES/RDS | 4 | ✅ |
| J | P1-3 Prompt Handbook (3 skills) | template + ECS/CES/CCE | 4 | ✅ |
| K | P1-4 Trend Detection (4 skills) | ECS/CES/RDS/CCE | 4 | ✅ |

### 2.4 Phase 3 — Cascade Patterns (P1-1, 4 files)

| Batch | P1 Item | Files | Count | Status |
|-------|---------|-------|-------|--------|
| L | P1-1 Cascade Patterns (4 skills) | ECS/CES/CCE/RDS | 4 | ✅ |

### 2.5 Aggregate File Counts

| Phase | Files Created |
|-------|--------------|
| Phase 0 (Gap) | 9 |
| Phase 1 (P0) | 40 |
| Phase 2 (P1) | 33 |
| Phase 3 (P1-1) | 4 |
| **Total** | **86** |

---

## 3. Architectural Decisions

### 3.1 Template-First Approach

Every cross-skill deliverable (chaos engineering, resilience score, observability trinity, prompt handbook, diagnosis confidence) starts with a **template** in `huaweicloud-skill-generator/references/`, then rolls out to individual skills. This ensures consistency and reduces per-skill authoring cost.

```
huaweicloud-skill-generator/references/
├── cross-skill-delegation-matrix-template.md   (Batch A)
├── diagnosis-confidence-template.md             (Batch E)
├── chaos-engineering-template.md                (Batch G)
├── resilience-score-template.md                 (Batch H)
├── observability-trinity-template.md            (Batch I)
├── prompt-handbook-template.md                  (Batch J)
```

### 3.2 Priority Ordering

P0 items (L3 foundation) are mandatory gates before P1 (L4 capabilities):
1. SLO/SLI definitions → establish measurable targets
2. Change correlation → connect CTS events to alarms
3. Capacity forecasting → predict resource exhaustion
4. Diagnosis confidence → quantify certainty
5. Chaos engineering + resilience scoring → L4 core

### 3.3 Skill Selection Criteria

- **Core skills** (ECS, CCE, RDS, CES, ELB): receive all L3 and L4 deliverables
- **L1 skills** (OBS, LTS, CBR, SWR, CTS): focused on L2 gap filling + anomaly patterns + knowledge base
- **Infrastructure skills** (VPC, DCS, DMS, CSS, GaussDB): SLO/SLI only (Batch B)

---

## 4. Key Design Patterns

### 4.1 SLO/SLI Definition Pattern

Each skill's `well-architected-assessment.md` includes:
- **SLI dimensions**: availability, latency (P99), error rate, saturation
- **SLO targets**: e.g., availability ≥ 99.9%, P99 latency < 200ms
- **Error budget**: monthly budget with burn rate alerts at 10%/20%/50%

### 4.2 Change Correlation Pattern

CTS events → alarm mapping with 30-minute correlation window:
```yaml
change_correlation:
  window: 30min
  mappings:
    - cts_event: "ecs reboot"
      triggered_alarms: ["ecs_instance_down", "connection_timeout"]
    - cts_event: "security group rule update"
      triggered_alarms: ["connection_drop", "latency_spike"]
```

### 4.3 Capacity Forecasting Model

Linear extrapolation for monotonic metrics:
```
exhaustion_date = (limit - current_value) / daily_growth_rate
```
With Warning at 80% and Critical at 95% thresholds.

### 4.4 Diagnosis Confidence Model

```
confidence = Σ(evidence_i × weight_i) / Σ(weight_i)
```
Levels: High (≥0.8), Medium (≥0.5), Low (≥0.2), Very Low (<0.2)

### 4.5 Chaos Engineering Experiment Design

Each experiment defines:
- **Fault scenario**: instance failure, AZ failure, disk failure, load spike, dependency failure
- **Blast radius**: single instance, multi-instance, AZ-level
- **Duration**: 5-30 minutes depending on risk
- **Rollback**: automated rollback procedure
- **Resilience scoring**: 5 dimensions × 0-10 scale

### 4.6 Resilience Score Dimensions

| Dimension | Weight | Description |
|-----------|--------|-------------|
| Fault Detection Speed | 20% | Time from fault injection to detection |
| Fault Isolation | 20% | Ability to contain blast radius |
| Recovery Automation | 25% | Degree of automated recovery |
| Degradation Quality | 20% | Quality of degraded service |
| Data Consistency | 15% | Data integrity after recovery |

---

## 5. Relationship to L5 Autonomous Operations

The L4 capabilities established here are prerequisites for the L5 autonomous loop:

| L4 Component | Used By L5 | L5 Implementation |
|-------------|-----------|-------------------|
| Diagnosis Confidence | Diagnoser | `internal/l4/orchestrator.go` — confidence threshold gating |
| Change Correlation | Diagnoser (causal chain) | `internal/l4/topology.go` — topology graph |
| Capacity Forecasting | Predictor | `internal/l4/predictive.go` — breach prediction |
| Chaos Engineering | Resilience Testing | `internal/l4/trust.go` — trust-tier escalation |
| Observability Trinity | Detector + Verifier | CES alarm + LTS log + APM trace correlation |
| Cascade Patterns | Diagnoser (cross-skill) | `internal/l4/orchestration.go` — cross-skill delegation |

---

## 6. Acceptance Criteria

| Metric | Target | Actual (2026-07-18) |
|--------|--------|---------------------|
| AIOps Maturity | 80%+ | ~80% |
| L2 Completeness | 100% | 100% (all skills: delegation + kb + prompts) |
| L3 Completeness | 100% | 100% (core skills: SLO + change-corr + capacity) |
| P0 Compliance | 10/10 | 10/10 |
| P1 Compliance | 9/9 | 9/9 |
| Chaos Engineering rollout | 5 skills | 5 (ECS/CCE/RDS/CES/ELB) |
| Resilience Score rollout | 5 skills | 5 (ECS/CCE/RDS/DCS/VPC) |
| Prompt Handbook | 3 skills | 3 (ECS/CES/CCE) |

---

## 7. References

- `docs/superpowers/plans/aiops-optimization.md` — execution plan (COMPLETE)
- `huaweicloud-skill-generator/references/aiops-best-practices.md` — AIOps best practices guide
- `huaweicloud-skill-generator/references/well-architected-assessment.md` §7 — Maturity Model
- `docs/superpowers/specs/gcl-token-efficiency-p0.md` — prerequisite TE-7 layering
- `docs/superpowers/specs/aiops-l5-autonomous.md` — L5 autonomous spec (upstream consumer)
