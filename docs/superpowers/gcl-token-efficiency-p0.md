# Plan: GCL Token Efficiency P0 — Backbone Reuse + SKILL.md GCL Compression

## Goal

Reduce token load of GCL artifacts across all `huaweicloud-*-ops` skills by:
1. **TE-7 (#1):** Make every `references/prompt-templates.md` reference the shared
   `huaweicloud-skill-generator/references/gcl-prompt-backbone.md` instead of inlining the
   shared Generator/Critic/Orchestrator skeletons.
2. **TE-6/compression (#2):** Compress the `## Quality Gate (GCL)` section in every `SKILL.md`
   so the generic Runtime Roles / Rubric Thresholds / Trace Requirements tables become a pointer
   to `docs/gcl-spec.md` + `AGENTS.md`, while **product-specific** GCL content is preserved.

## Scope (IN)

- 21 skills with `references/prompt-templates.md`: billing, cbr, cce, cdn, ces, css, cts, dcs,
  dms, ecs, eip, elb, functiongraph, gaussdb, hss, iam, lts, obs, rds, swr, vpc, waf.
  (dns has NO prompt-templates.md — leave it; its SKILL.md GCL section is already minimal.)
- All 22 `huaweicloud-*-ops/SKILL.md` `## Quality Gate (GCL)` sections.

## Scope (OUT)

- `rubric.md` (separate TE-6 opportunity, deferred to P1).
- `well-architected-assessment.md` (separate P1).
- Any non-GCL content.
- `huaweicloud-skill-generator/` itself (the backbone + template are the source of truth).

## Reference Examples (CORRECT — do not change)

- `huaweicloud-eip-ops/references/prompt-templates.md` (196 lines, references backbone in §7).
- `huaweicloud-cdn-ops/references/prompt-templates.md` (145 lines, backbone_ref=1).
- `huaweicloud-eip-ops/SKILL.md` GCL section (40 lines, already compressed).

## Transformation Rules — prompt-templates.md (#1)

For each skill's `references/prompt-templates.md` where `backbone_ref == 0`:

**KEEP (product-specific, never delete):**
- The H1 title + intro note (version, independence rule).
- `## Template Index` table — but change the §1/§2/§3 rows' "Purpose" to say
  "see gcl-prompt-backbone.md §N (product overrides below)".
- Product-specific Generator content: operation list (`create-x | delete-x | ...`),
  product hard-rules (e.g. ECS S1–S10 references, EIP quota/cooldown rules).
- Product-specific pre-flight overrides section (if present).
- Product-specific Safety Rules table IF it lives here (eip has S1–S17 inline — keep it).
- `## See also` — ADD a line:
  `- \`huaweicloud-skill-generator/references/gcl-prompt-backbone.md\` (shared Generator/Critic/Orchestrator skeleton)`.

**REPLACE (generic, duplicated from backbone):**
- The full Generator JSON output schema block → replace with one line:
  `> Shared Generator skeleton + JSON output schema: see gcl-prompt-backbone.md §1.`
- The full Critic JSON output schema block → replace with:
  `> Shared Critic skeleton + JSON output schema: see gcl-prompt-backbone.md §2.`
- The full Orchestrator loop pseudocode block → replace with:
  `> Shared Orchestrator skeleton + decision logic: see gcl-prompt-backbone.md §3.`
- Generic Sanitization steps that duplicate backbone §4 → replace with pointer.
- Generic Failure Recovery table that duplicates backbone §4 anti-patterns → replace with pointer.

**DO NOT:**
- Introduce bare `{...}` placeholders (AGENTS.md ban).
- Remove product-specific Safety Rules or operation lists.
- Change `{{env.*}}` / `{{user.*}}` / `{{output.*}}` usage.

## Transformation Rules — SKILL.md GCL section (#2)

For each `SKILL.md`, locate `## Quality Gate (GCL)` ... next `## ` heading.

**KEEP (product-specific, never delete):**
- Product-specific `max_iter` overrides (e.g. ECS `delete-server` capped at 2).
- Product-specific Safety = 0 triggers list (e.g. ECS S1–S6, the "MUST self-check before" block).
- Product-specific Spec Compliance anchors (e.g. ECS: region, flavor regex, image prefix).
- The `gcl` metadata block (required / default_max_iter / rubric_version / trace_path).
- The bullet pointing to `references/rubric.md` and `references/prompt-templates.md`.

**REPLACE (generic, duplicated across skills):**
- The `### Runtime Roles` table → replace with:
  `> Runtime Roles (Generator / Critic / Orchestrator) and their isolation constraints: see
   \`docs/gcl-spec.md\` §Runtime Roles and root \`AGENTS.md\` §5.`
- The `### Default Rubric Thresholds` table → replace with:
  `> Default rubric thresholds (correctness ≥0.5, safety =1.0, …): see \`docs/gcl-spec.md\`
   §Thresholds. Product overrides stated above.`
- The `### Trace Requirements` numbered list → replace with:
  `> Trace persistence + masking rules: see \`docs/gcl-spec.md\` §Trace and root \`AGENTS.md\`
   (credential masking mandatory).`

**DO NOT:**
- Delete the `## Quality Gate (GCL)` heading.
- Delete the `gcl:` metadata block in frontmatter.
- Alter any product-specific safety rule wording.

## Execution Approach

- Use git worktree per batch (AGENTS.md git-worktree rule): 3 worktrees, each handles ~7 skills.
- Inside each worktree, delegate to a `deep`/`unspecified-high` sub-agent with this plan attached
  as the sole context, plus the eip example as the target shape.
- Sub-agent MUST verify: after edit, `grep -c gcl-prompt-backbone <skill>/references/prompt-templates.md`
  returns ≥ 1 for every edited file; and SKILL.md still contains product-specific safety triggers.

## Verification (DoD)

1. `grep -c 'gcl-prompt-backbone' <skill>/references/prompt-templates.md` ≥ 1 for all 19
   previously-unreferenced skills (cdn/eip already pass).
2. Every edited `SKILL.md` still contains its product-specific `Safety = 0` / S-rule triggers.
3. `python3 scripts/validate_local.py` passes (no regression in GCL conformance check).
4. `bash scripts/pre_commit_check.sh` passes (ruff/py310 only relevant if scripts touched — they are not,
   but run to be safe).
5. Token delta: total `prompt-templates.md` + `SKILL.md` GCL-section lines reduced by ≥ 30%.

## Risks

- **Low.** Markdown-only, no code. Product-specific content explicitly preserved.
- Backbone is the canonical source; if a skill had a *divergence* from backbone (intentional override),
  the sub-agent must KEEP that divergence and only compress the truly-generic parts.
