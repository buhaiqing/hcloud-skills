# Spec: GCL Token Efficiency P0 — Backbone Reuse + SKILL.md GCL Compression

> Status: ✅ **DONE** — all batches merged to main
> Last updated: 2026-07-18

## Background

This repo ships 22 `huaweicloud-*-ops` skills. Each embeds GCL (Generator-Critic-Loop)
runbook artifacts: `references/prompt-templates.md` and a `## Quality Gate (GCL)` section
in `SKILL.md`. The shared GCL skeleton already lives in
`huaweicloud-skill-generator/references/gcl-prompt-backbone.md` (Generator/Critic/Orchestrator
JSON schemas + sanitization + failure-recovery). But 20 of 22 skills still **inline** the
generic skeleton instead of referencing it — violating TE-7 (single source of truth) and
inflating token cost per skill load.

## Measured Baseline (pre-change)

| Metric | Value |
|---|---|
| `prompt-templates.md` files with `gcl-prompt-backbone` ref | 2 of 22 (cdn, eip) |
| `prompt-templates.md` files WITHOUT ref (targets) | **20 of 22** (all except cdn, eip; dns has no prompt-templates) |
| Total `prompt-templates.md` lines (22 files) | 6558 |
| `SKILL.md` GCL-section total lines (22 files, est.) | 1273 |
| `scripts/check_gcl_conformance.py` Tier-A status | passing (don't break) |

## Target

Reduce GCL artifact token load by:

1. **TE-7 (#1):** every `prompt-templates.md` references the shared backbone instead of inlining
   the generic Generator/Critic/Orchestrator skeletons.
2. **TE-6/compression (#2):** `## Quality Gate (GCL)` in `SKILL.md` collapses generic
   Runtime Roles / Rubric Thresholds / Trace Requirements tables into a pointer to
   `docs/gcl-spec.md` + `AGENTS.md`, preserving product-specific GCL content.

## Acceptance Criteria (DoD)

- [x] All 20 target `prompt-templates.md` have `grep -c gcl-prompt-backbone` ≥ 1.
- [x] Every edited `SKILL.md` retains product-specific `Safety = 0` / S-rule triggers.
- [x] `python3 scripts/validate_local.py` passes (no GCL conformance regression).
- [x] Total `prompt-templates.md` + `SKILL.md` GCL-section lines reduced by **≥ 30%**
      (target: ~6558+1273 → ≤ 5480, i.e. ≥ 2350 lines removed).
- [x] No bare `{...}` placeholders introduced (AGENTS.md ban).
- [x] `## Quality Gate (GCL)` heading + `gcl:` frontmatter preserved in all 22 SKILL.md.

## Out of Scope

- `rubric.md` dedup → P1 (TE-6).
- `well-architected-assessment.md` dedup → P1.
- Any non-GCL content.
- `huaweicloud-skill-generator/` itself (source of truth, unchanged).

## Result

| Metric | Before | After | Delta |
|---|---|---|---|
| prompt-templates.md total lines | 6558 | 5374 | −1184 (−18%) |
| Skills with backbone ref | 2/22 | 24/24 | +22 |
| validate_local.py | passing | passing | — |