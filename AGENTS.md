# AGENTS.md — hcloud-skills

## What This Repo Is

Huawei Cloud Ops Skill collection — structured agent runbooks (`huaweicloud-[product]-ops`) executed via `hcloud` CLI (primary) with Go SDK JIT fallback. Not application code; no build/test/lint step.

## Skill Directory Layout (Convention)

Every skill follows this structure — do not deviate:

```
huaweicloud-[product]-ops/
├── SKILL.md              # Main runbook: frontmatter, triggers, operations, recovery
├── references/           # Deep reference files (core-concepts, api-sdk-usage, cli-usage, troubleshooting, monitoring, integration, well-architected-assessment, etc.)
└── assets/               # eval_queries.json + example-config.yaml
```

**SKILL.md is the entry point.** References provide depth. No duplication between them.

## Generator / Meta-Skill

`huaweicloud-skill-generator` scaffolds new skills from OpenAPI specs. Load the `huaweicloud-skill-generator` skill when creating or updating any `huaweicloud-*-ops`. It enforces P0/P1 quality gates, the Five Core Standards, and three-pillar integration.

Template: `huaweicloud-skill-generator/references/huaweicloud-skill-template.md`

## ⚠️ Dual-Copy Trap

The generator exists in **two places** with diverging content:
- `huaweicloud-skill-generator/` (root — canonical, tracked by git)
- `.agents/skills/huaweicloud-skill-generator/` (loaded by agent runtime — may be stale)

When editing the generator, update the **root copy**. The `.agents/skills/` copy is NOT in git and may drift.

## Placeholder Conventions

| Placeholder | Source | Rule |
|-------------|--------|------|
| `{{env.*}}` | Runtime environment | **Never** ask user; fail if unset |
| `{{user.*}}` | User input | Collect interactively |
| `{{output.*}}` | API response capture | Chain into subsequent steps |

## Execution Paths

- **Primary**: `hcloud` CLI — always prefer when CLI supports the operation
- **Fallback**: Go SDK (`github.com/huaweicloud/huaweicloud-sdk-go-v3`) via JIT `go run` — for unsupported CLI operations
- `cli_applicability` field in SKILL.md frontmatter: `cli-first` | `dual-path` | `sdk-only` | `cli-only`

## Three-Pillar Integration (Mandatory)

Every skill MUST embed FinOps + SecOps + AIOps. No exceptions:

- **FinOps**: Billing model comparison, idle resource detection, right-sizing, budget alerts
- **SecOps**: IAM least-privilege table, credential masking (`***`), network isolation, encryption
- **AIOps**: ≥4 anomaly patterns, cross-skill delegation matrix, fault knowledge base, alarm storm handling

## Quality Gates

### P0 (Must Pass)
- SHOULD/SHOULD NOT trigger conditions complete
- Pre-flight → Execute → Validate → Recover flow for each operation
- ≥10 product error codes with recovery strategies
- Destructive operations have safety gates (explicit confirmation)
- `assets/eval_queries.json` with should/should-not trigger queries

### P1 (Should Pass)
- Idempotency documented where automation applies
- Cross-skill delegation matrix in `integration.md`
- Adversarial scenarios considered
- Self-reflection completed

## Skill Update Rule: 2-Round Self-Reflection

**After every skill update or creation, execute 2 mandatory self-reflection rounds and auto-fix all discovered issues before finishing.**

### Round 1 — Foundation Check
1. **FinOps**: Are cost patterns actionable? Billing model comparison present? Idle detection documented?
2. **SecOps**: IAM permissions minimum documented? Credential masking enforced? Network isolation?
3. **AIOps**: Multi-metric correlation defined? Delegation matrix present? Knowledge base populated?

### Round 2 — Critical Analysis
4. **Gap Analysis**: What would break in production if a user follows this skill?
5. **Alternative Coverage**: Is there a better way that reduces agent confusion?
6. **Escalation Paths**: Are HALT conditions clear? Enough non-retryable error patterns?
7. **Cross-Pillar Synergy**: Do FinOps recommendations conflict with reliability? SecOps create performance bottlenecks?

**For any issue found: fix immediately, then re-verify.** Do not report and stop — fix and verify the fix passes.

## Docker Sandbox

```bash
docker-compose build
docker-compose up hcloud-skills
# Inside container:
check-env          # Verify HW_* env vars
skill-list          # List all available skills
skill-read <name>   # Read a skill's SKILL.md
hc <product> <op>   # Alias for hcloud CLI
```

Services: `hcloud-skills` (interactive), `hcloud-worker` (non-interactive), `hcloud-test` (test runner, profile: test), `hcloud-sdk-builder` (Go build, profile: build).

## Environment Variables

| Variable | Required | Default |
|----------|----------|---------|
| `HW_ACCESS_KEY_ID` | Yes | — |
| `HW_SECRET_ACCESS_KEY` | Yes | — |
| `HW_REGION_ID` | No | `cn-north-4` |
| `HW_PROJECT_ID` | Service-specific | — |

## Key Anti-Patterns to Avoid

| Anti-Pattern | What to Do Instead |
|---|---|
| Inventing API fields or CLI flags | Cross-reference every field against OpenAPI or verified CLI output |
| Printing/logging real credentials | Mask with `***` / `<masked>` |
| Skipping safety gate on destructive ops | Add explicit confirmation step |
| Hardcoding regions/timeouts | Use `{{env.*}}` / `{{user.*}}` placeholders |
| One skill does everything | Single product, single resource model; delegate cross-product ops |
| SKILL.md duplicates references/ | SKILL.md = entry point; references = depth; no overlap |

## Delegation Matrix (Common Cross-Product Operations)

- ECS → VPC (subnet), CES (metrics), ELB (load balancing)
- RDS → ECS (CloudShell), CES (performance metrics)
- All products → IAM (permission issues), CTS (audit trails), BSS (billing)

## Sources of Truth

1. OpenAPI + official docs > forums/chat
2. Verified `hcloud` CLI output > assumed behavior
3. `huaweicloud-sdk-go-v3` for SDK fallback patterns
4. API docs: https://support.huaweicloud.com/api/
