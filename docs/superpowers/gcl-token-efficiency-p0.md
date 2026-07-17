# Spec + Plan: GCL Token Efficiency P0 — Backbone Reuse + SKILL.md GCL Compression

> Status: spec ✅ | plan ✅ | implement 🔄 (B/C running in worktrees)
> Last updated: 2026-07-18

---

## 0. Spec

### 0.1 Background

This repo ships 22 `huaweicloud-*-ops` skills. Each embeds GCL (Generator-Critic-Loop)
runbook artifacts: `references/prompt-templates.md` and a `## Quality Gate (GCL)` section
in `SKILL.md`. The shared GCL skeleton already lives in
`huaweicloud-skill-generator/references/gcl-prompt-backbone.md` (Generator/Critic/Orchestrator
JSON schemas + sanitization + failure-recovery). But 20 of 22 skills still **inline** the
generic skeleton instead of referencing it — violating TE-7 (single source of truth) and
inflating token cost per skill load.

### 0.2 Measured Baseline (from main repo, pre-change)

| Metric | Value |
|---|---|
| `prompt-templates.md` files with `gcl-prompt-backbone` ref | 2 of 22 (cdn, eip) |
| `prompt-templates.md` files WITHOUT ref (targets) | **20 of 22** (all except cdn, eip; dns has no prompt-templates) |
| Total `prompt-templates.md` lines (22 files) | 6558 |
| `SKILL.md` GCL-section total lines (22 files, est.) | 1273 |
| `scripts/check_gcl_conformance.py` Tier-A status | passing (don't break) |

### 0.3 Target

Reduce GCL artifact token load by:
1. **TE-7 (#1):** every `prompt-templates.md` references the shared backbone instead of inlining
   the generic Generator/Critic/Orchestrator skeletons.
2. **TE-6/compression (#2):** `## Quality Gate (GCL)` in `SKILL.md` collapses generic
   Runtime Roles / Rubric Thresholds / Trace Requirements tables into a pointer to
   `docs/gcl-spec.md` + `AGENTS.md`, preserving product-specific GCL content.

### 0.4 Acceptance Criteria (DoD)

- [ ] All 20 target `prompt-templates.md` have `grep -c gcl-prompt-backbone` ≥ 1.
- [ ] Every edited `SKILL.md` retains product-specific `Safety = 0` / S-rule triggers.
- [ ] `python3 scripts/validate_local.py` passes (no GCL conformance regression).
- [ ] Total `prompt-templates.md` + `SKILL.md` GCL-section lines reduced by **≥ 30%**
      (target: ~6558+1273 → ≤ 5480, i.e. ≥ 2350 lines removed).
- [ ] No bare `{...}` placeholders introduced (AGENTS.md ban).
- [ ] `## Quality Gate (GCL)` heading + `gcl:` frontmatter preserved in all 22 SKILL.md.

### 0.5 Out of Scope

- `rubric.md` dedup → P1 (TE-6).
- `well-architected-assessment.md` dedup → P1.
- Any non-GCL content.
- `huaweicloud-skill-generator/` itself (source of truth, unchanged).

---

## 1. Plan

### 1.1 Reference Examples (CORRECT — do not modify)

- `huaweicloud-eip-ops/references/prompt-templates.md` (196 lines, refs backbone in §7).
- `huaweicloud-cdn-ops/references/prompt-templates.md` (145 lines, backbone_ref=1).
- `huaweicloud-eip-ops/SKILL.md` GCL section (~40 lines, already compressed).

### 1.2 Transformation Rules — prompt-templates.md (#1)

For each skill where `backbone_ref == 0`:

**KEEP (product-specific, never delete):**
- H1 title + intro note (version, independence rule).
- `## Template Index` table — change §1/§2/§3 "Purpose" cells to
  "see gcl-prompt-backbone.md §N (product overrides below)".
- Product-specific Generator content: operation list (`create-x | delete-x | …`),
  product hard-rules (ECS S1–S10, EIP quota/cooldown, etc.).
- Product-specific pre-flight overrides section (if present).
- Product-specific Safety Rules table IF inline here (eip S1–S17 — keep).
- `## See also` — ADD:
  `- \`huaweicloud-skill-generator/references/gcl-prompt-backbone.md\` (shared Generator/Critic/Orchestrator skeleton)`.

**REPLACE (generic, duplicated from backbone):**
- Full Generator JSON schema block → `> Shared Generator skeleton + JSON schema: see gcl-prompt-backbone.md §1.`
- Full Critic JSON schema block → `> Shared Critic skeleton + JSON schema: see gcl-prompt-backbone.md §2.`
- Orchestrator loop pseudocode → `> Shared Orchestrator skeleton + decision logic: see gcl-prompt-backbone.md §3.`
- Generic Sanitization steps duplicating backbone §4 → pointer.
- Generic Failure Recovery table duplicating backbone §4 → pointer.

**DO NOT:** introduce bare `{...}`; remove product Safety Rules / operation lists;
change `{{env.*}}` / `{{user.*}}` / `{{output.*}}` usage.

### 1.3 Transformation Rules — SKILL.md GCL section (#2)

Locate `## Quality Gate (GCL)` … next `## ` heading.

**KEEP (product-specific, never delete):**
- Product-specific `max_iter` overrides (e.g. ECS `delete-server` capped at 2).
- Product-specific `Safety = 0` triggers (ECS S1–S6 "MUST self-check before" block).
- Product-specific Spec Compliance anchors (region, flavor regex, image prefix).
- The `gcl:` metadata block (required / default_max_iter / rubric_version / trace_path).
- Bullet to `references/rubric.md` and `references/prompt-templates.md`.

**REPLACE (generic, duplicated across skills):**
- `### Runtime Roles` table →
  `> Runtime Roles (Generator/Critic/Orchestrator) + isolation: see \`docs/gcl-spec.md\` §Runtime Roles and root \`AGENTS.md\` §5.`
- `### Default Rubric Thresholds` table →
  `> Default rubric thresholds (correctness ≥0.5, safety =1.0, …): see \`docs/gcl-spec.md\` §Thresholds. Product overrides stated above.`
- `### Trace Requirements` list →
  `> Trace persistence + masking: see \`docs/gcl-spec.md\` §Trace and root \`AGENTS.md\` (credential masking mandatory).`

**DO NOT:** delete `## Quality Gate (GCL)` heading; delete `gcl:` frontmatter;
alter product-specific safety wording.

### 1.4 Execution Approach

- **git worktree per batch** (AGENTS.md git-worktree rule): 3 worktrees, ~7 skills each.
  - `../hcloud-skills-gcl-p0a` → branch `feature/gcl-token-eff-p0a` (billing/cbr/cce/ces/css/cts) — **DONE, committed `93dfd41`**.
  - `../hcloud-skills-gcl-p0b` → branch `feature/gcl-token-eff-p0b` (dcs/dms/ecs/elb/functiongraph/gaussdb) — **IN PROGRESS**.
  - `../hcloud-skills-gcl-p0c` → branch `feature/gcl-token-eff-p0c` (hss/iam/lts/obs/rds/swr/vpc/waf) — **IN PROGRESS (redo)**.
- Each worktree: delegate to `unspecified-high` sub-agent with this plan as sole context +
  eip example as target shape. Sub-agent commits **per skill** (read→edit→commit) to avoid
  30-min timeout (learned: a prior C-agent returned empty "completed" with no commits).
- Verification is done by main flow against real `git status`/`git log`, NOT agent self-report.

### 1.5 Verification Commands (run by main flow post-merge)

```bash
# 1. backbone refs across all skills
for d in huaweicloud-*-ops; do
  f="$d/references/prompt-templates.md"; [ -f "$f ] || continue
  printf "%-28s %s\n" "$d" "$(grep -c gcl-prompt-backbone "$f")"
done | sort -t' ' -k2 -n   # expect 0 only for dns; all others ≥1

# 2. product safety triggers preserved (spot-check ECS as densest)
grep -l "Safety = 0" huaweicloud-ecs-ops/SKILL.md && echo "ECS safety triggers present"

# 3. GCL conformance + repo validation
python3 scripts/validate_local.py
bash scripts/pre_commit_check.sh

# 4. line-delta proof
git diff --stat main..HEAD -- 'huaweicloud-*-ops/**/prompt-templates.md' 'huaweicloud-*-ops/**/SKILL.md'
```

---

## 2. Implement (tracking)

| Batch | Skills | Worktree | Branch | Status |
|---|---|---|---|---|
| A | billing, cbr, cce, ces, css, cts | gcl-p0a | feature/gcl-token-eff-p0a | ✅ committed `93dfd41`, −749 lines |
| B | dcs, dms, ecs, elb, functiongraph, gaussdb | gcl-p0b | feature/gcl-token-eff-p0b | 🔄 dcs/dms/ecs done; elb/functiongraph/gaussdb in flight |
| C | hss, iam, lts, obs, rds, swr, vpc, waf | gcl-p0c | feature/gcl-token-eff-p0c | 🔄 redo agent running (prior empty run) |

### 2.1 Post-merge Checklist

- [ ] Merge 3 worktrees → main.
- [ ] Run §1.5 verification; confirm DoD §0.4 all checked.
- [ ] Remove worktrees: `git worktree remove ../hcloud-skills-gcl-p0a` etc.
- [ ] Commit this plan file to main as spec record (or delete if treated as temp).
- [ ] Report final line-delta to user.
