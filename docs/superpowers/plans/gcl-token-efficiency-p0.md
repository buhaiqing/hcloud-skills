# Plan: GCL Token Efficiency P0 — Backbone Reuse + SKILL.md GCL Compression

> Status: ✅ **COMPLETE** — all batches merged to main
> Last updated: 2026-07-18

## Execution Approach

- **git worktree per batch** (AGENTS.md git-worktree rule): 3 worktrees, ~7 skills each.
- Each worktree: delegate to sub-agent with this plan as sole context +
  eip example as target shape. Sub-agent commits **per skill** (read→edit→commit) to avoid
  30-min timeout.
- Verification is done by main flow against real `git status`/`git log`, NOT agent self-report.

## Batch Breakdown

| Batch | Skills | Branch | Status |
|---|---|---|---|
| A | billing, cbr, cce, ces, css, cts | `feature/gcl-token-eff-p0a` | ✅ Merged (`5fde17a`), −749 lines |
| B | dcs, dms, ecs, elb, functiongraph, gaussdb | `feature/gcl-token-eff-p0b` | ✅ Merged (`d9cdfd3`) |
| C | hss, iam, lts, obs, rds, swr, vpc, waf | `feature/gcl-token-eff-p0c` | ✅ Merged (`c7a3784`) |

## Transformation Rules

### prompt-templates.md (#1)

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

### SKILL.md GCL section (#2)

Locate `## Quality Gate (GCL)` … next `## ` heading.

**KEEP (product-specific, never delete):**
- Product-specific `max_iter` overrides (e.g. ECS `delete-server` capped at 2).
- Product-specific `Safety = 0` triggers (ECS S1–S6 "MUST self-check before" block).
- Product-specific Spec Compliance anchors (region, flavor regex, image prefix).
- The `gcl:` metadata block (required / default_max_iter / rubric_version / trace_path).
- Bullet to `references/rubric.md` and `references/prompt-templates.md`.

**REPLACE (generic, duplicated across skills):**
- `### Runtime Roles` table → pointer to `docs/gcl-spec.md` + `AGENTS.md`.
- `### Default Rubric Thresholds` table → pointer to `docs/gcl-spec.md §Thresholds`.
- `### Trace Requirements` list → pointer to `docs/gcl-spec.md §Trace` + `AGENTS.md`.

**DO NOT:** delete `## Quality Gate (GCL)` heading; delete `gcl:` frontmatter;
alter product-specific safety wording.

## Verification (post-merge)

```bash
# 1. backbone refs across all skills
for d in huaweicloud-*-ops; do
  f="$d/references/prompt-templates.md"; [ -f "$f ] || continue
  printf "%-28s %s\n" "$d" "$(grep -c gcl-prompt-backbone "$f")"
done | sort -t' ' -k2 -n   # expect 0 only for dns; all others ≥1

# 2. product safety triggers preserved
grep -l "Safety = 0" huaweicloud-ecs-ops/SKILL.md && echo "ECS safety triggers present"

# 3. GCL conformance + repo validation
python3 scripts/validate_local.py
bash scripts/pre_commit_check.sh

# 4. line-delta proof
git diff --stat main..HEAD -- 'huaweicloud-*-ops/**/prompt-templates.md' 'huaweicloud-*-ops/**/SKILL.md'
```

## Post-merge Checklist

- [x] Merge 3 worktrees → main.
- [x] Run verification; confirm DoD all checked.
- [x] Remove worktrees: `git worktree remove ../hcloud-skills-gcl-p0a` etc.
- [x] Clean up: worktrees removed, no stale branches.