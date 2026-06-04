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

---

## Generator-Critic-Loop (GCL) — Adversarial Quality Gate

> Inspired by GAN's Generator/Discriminator idea, but deliberately **not** a real GAN.
> Naming: **GCL (Generator-Critic-Loop)** to avoid misleading reviewers and LLM trainees.

### 1. Purpose

Apply an adversarial **Generator ↔ Critic** loop with a quantitative rubric to every skill execution.
Most valuable in **high-side-effect cloud operations** (delete, stop, restore, IAM/KMS/DDL) where a single
mistake is unrecoverable. GCL sits **on top of** the existing P0/P1 quality gates and the
**2-round self-reflection** rule — it is the third and highest quality tier in this repo.

| GAN (real) | GCL (this spec) |
|---|---|
| Discriminator learns sample distribution | Critic scores an **explicit rubric** |
| No termination condition | Must terminate: **PASS / MAX_ITER / SAFETY_FAIL** |
| G and D train in parallel | G and C run **sequentially** |
| Goal: "fool the D" | Goal: "pass the rubric threshold" |

### 2. Roles

| Role | Job | Input | Output | Forbidden |
|---|---|---|---|---|
| **Generator (G)** | Execute the cloud operation | user request + previous Critic feedback | result + execution trace | modifying the rubric; self-scoring |
| **Critic (C)** | Independently audit G's output | G's result + trace + rubric | scores + suggestions | calling `hcloud` / SDK / mutating anything |
| **Orchestrator (O)** | Loop control, termination, final return | context + C scores + budget | continue / final result | executing or scoring on its own |

**Hard constraint:** G and C MUST live in **isolated prompt contexts** (preferably isolated sessions
or sub-agents). A shared context is a "pseudo-GCL" and is explicitly banned — see §9.

### 3. Rubric (mandatory per skill)

Each `SKILL.md` MUST declare its skill-specific rubric under `## Quality Gate (GCL)`. Minimum 5 dimensions:

| Dimension | Meaning | Scale | Default threshold |
|---|---|---|---|
| **Correctness** | Resource id / state / config actually matches the request | 0 / 0.5 / 1 | ≥ 0.5 (1.0 required for `delete` / `stop` / IAM / KMS / DDL) |
| **Safety** | Destructive op (`delete` / `stop` / `restore` / IAM / KMS / DDL) was confirmed or guarded | 0 / 1 | = 1 |
| **Idempotency** | Retrying the same call will not cause duplicate side-effects | 0 / 0.5 / 1 | ≥ 0.5 |
| **Traceability** | Output is auditable: command, params, raw response, errors all captured | 0 / 0.5 / 1 | ≥ 0.5 |
| **Spec Compliance** | Conforms to the skill's `core-concepts.md` constraints (region, quota, billing model) | 0 / 0.5 / 1 | ≥ 0.5 |

**Safety = 0 → ABORT immediately, regardless of total score.**

### 4. Loop Flow

```
User Request
     │
     ▼
[0] Pre-flight (Orchestrator)
    - resolve env.* and user.* variables
    - pick skill, load its rubric from SKILL.md
    - verify P0/P1 gates already passed
     │
     ▼
[1] Generate (G) ───────────────────────┐
    - run hcloud (or SDK fallback)       │
    - capture trace                      │
     │                                   │
     ▼                                   │
[2] Critique (C)                        │
    - isolated prompt context            │
    - score every rubric dimension       │
    - emit actionable suggestions        │
     │                                   │
     ▼                                   │
[3] Decide (Orchestrator)               │
    - Safety=0  → ABORT (no partial)    │
    - all pass  → RETURN                 │
    - else & iter<max → inject          │
       suggestions into G                │
    - else → RETURN best + unresolved    │
       rubric items                      │
     └───────────────────────────────────┘
```

### 5. Termination (first match wins)

| Condition | Behavior |
|---|---|
| **PASS** | Every rubric dimension meets its threshold → return G's result |
| **MAX_ITER** | Reached `max_iterations` (default per skill class — see §8) → return **best-so-far** + unresolved rubric items |
| **SAFETY_FAIL** | Safety = 0 → **ABORT**; never return partial or "best-effort" output |

### 6. Trace & Audit (mandatory)

Every GCL run MUST persist a JSON trace:

```json
{
  "skill": "huaweicloud-ecs-ops",
  "request": "<sanitized user request>",
  "rubric_version": "v1",
  "iterations": [
    {
      "iter": 1,
      "generator": { "command": "hcloud ecs delete", "args": {...}, "exit_code": 0, "result_excerpt": "..." },
      "critic": {
        "scores": {
          "correctness": 1, "safety": 1, "idempotency": 0.5,
          "traceability": 1, "spec_compliance": 1
        },
        "suggestions": ["..."],
        "blocking": false
      },
      "decision": "RETRY"
    }
  ],
  "final": { "status": "PASS", "iter": 2, "output": "..." }
}
```

Path: `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` — must be in `.gitignore`. Trace files are
**append-only**; never overwrite or delete in place.

### 7. Prompt Templates (mandatory per skill)

Each skill's `references/prompt-templates.md` (under the skill directory) MUST contain:

1. **Generator Prompt Template** — placeholders: `{{user.request}}`, `{{output.critic_feedback}}`, `{{output.rubric}}`
2. **Critic Prompt Template** — placeholders: `{{output.generator_output}}`, `{{output.trace}}`, `{{output.rubric}}`

> **Placeholder syntax** MUST follow the repository-wide convention
> (see top-level **Placeholder Conventions**): `{{env.*}}` / `{{user.*}}` / `{{output.*}}`.
> Bare `{...}` placeholders are NOT allowed in skill prompt templates.

**Critic prompt must hide the raw user request** to prevent "answer-aligned" rubber-stamping.
Recommended skeleton:

```text
You are an independent cloud-operation auditor.
You will see one execution result and its trace. Score it STRICTLY against the rubric below.
Do NOT consider the original user request — judge only what was actually done.

rubric: {{output.rubric}}
generator_output: {{output.generator_output}}
trace: {{output.trace}}

Return strict JSON:
{
  "scores": { "correctness": 0|0.5|1, "safety": 0|0.5|1, "idempotency": 0|0.5|1,
              "traceability": 0|0.5|1, "spec_compliance": 0|0.5|1 },
  "suggestions": ["≤ 3 concrete, executable improvements"],
  "blocking": true|false
}
```

### 8. Per-Skill Defaults

| Skill | GCL | Default max_iter | Notes |
|---|---|---|---|
| `huaweicloud-ecs-ops` | **required** | 2 | delete/stop/reboot are destructive |
| `huaweicloud-iam-ops` | **required** | 2 | detach policy / delete user / rotate keys |
| `huaweicloud-rds-ops` | **required** | 2 | instance delete / DDL / restore |
| `huaweicloud-gaussdb-ops` | **required** | 2 | instance delete / DDL / shard rebalance |
| `huaweicloud-dcs-ops` | **required** | 2 | FLUSHALL / instance delete / backup restore |
| `huaweicloud-dms-ops` | **required** | 2 | queue delete / message purge |
| `huaweicloud-css-ops` | **required** | 2 | cluster delete / snapshot restore |
| `huaweicloud-cce-ops` | **required** | 2 | node drain / cluster delete / workload evict |
| `huaweicloud-cbr-ops` | **required** | 2 | restore overwrites source |
| `huaweicloud-vpc-ops` | **required** | 2 | delete VPC / subnet / security group cascades |
| `huaweicloud-obs-ops` | **required** | 2 | bucket delete / lifecycle policy purge |
| `huaweicloud-swr-ops` | **required** | 2 | image delete / tag overwrite |
| `huaweicloud-functiongraph-ops` | **required** | 2 | function delete / version disable |
| `huaweicloud-waf-ops` | **required** | 2 | policy delete / rule disable |
| `huaweicloud-hss-ops` | **required** | 2 | host isolate / policy detach |
| `huaweicloud-elb-ops` | recommended | 3 | listener / backend delete / cert replace |
| `huaweicloud-ces-ops` | recommended | 3 | alarm rule delete |
| `huaweicloud-lts-ops` | recommended | 3 | log group / stream delete |
| `huaweicloud-cts-ops` | recommended | 3 | tracker disable / transfer delete |
| `huaweicloud-billing-ops` | optional | 5 | read-only / report generation |
| `huaweicloud-skill-generator` | optional | 3 | meta operation |

Each skill may override `max_iter` in its own `SKILL.md` (under `## Quality Gate (GCL)`).

### 9. Anti-Patterns (banned)

- ❌ **Shared context G+C** — defeats independence → banned
- ❌ **Subjective scoring** — Critic must use the rubric, not "vibes" → banned
- ❌ **Unbounded loop** — always hard-cap iterations → banned
- ❌ **Critic sees the user request** — encourages rubber-stamping → banned
- ❌ **Silently downgrade on Safety fail** — must ABORT visibly → banned
- ❌ **Trace not persisted** — no post-mortem possible → banned
- ❌ **Critic mutates resources** — Critic is read-only by definition → banned
- ❌ **Skip P0/P1 and 2-round self-reflection** — GCL assumes they already passed → banned
- ❌ **Print/log real credentials in trace** — mask `***` / `<masked>` always → banned

### 10. Relationship to Existing Quality Gates

GCL does **not** replace the existing quality layers — it **wraps** them:

```
┌─────────────────────────────────────────────┐
│  GCL (this section)  — runtime, per-op      │  ← NEW
├─────────────────────────────────────────────┤
│  2-Round Self-Reflection  — per-skill-update│  ← existing
├─────────────────────────────────────────────┤
│  P1 Quality Gates  — should-pass           │  ← existing
├─────────────────────────────────────────────┤
│  P0 Quality Gates  — must-pass             │  ← existing
├─────────────────────────────────────────────┤
│  Three-Pillar Integration (FinOps/SecOps/   │  ← existing
│  AIOps)                                     │
└─────────────────────────────────────────────┘
```

- **Skill creation/update** → must pass P0 → P1 → 2-round self-reflection.
- **Skill execution at runtime** → must additionally pass GCL when the skill class is `required` or `recommended`.

### 11. Rollout Roadmap

- **Phase 1 (✅ done 2026-06-04)** — add this section to `AGENTS.md`; pilot on **`huaweicloud-ecs-ops`** (most
  representative destructive workload) with its `references/prompt-templates.md` and a `## Quality Gate (GCL)` chapter
  in `SKILL.md`.
- **Phase 2 (next)** — roll out to the other `required` skills: `huaweicloud-iam-ops`, `huaweicloud-rds-ops`,
  `huaweicloud-vpc-ops`, `huaweicloud-gaussdb-ops` first (highest-blast-radius), then `dcs-ops`, `dms-ops`,
  `cbr-ops`, `css-ops`, `cce-ops`, `obs-ops`, `swr-ops`, `functiongraph-ops`, `waf-ops`, `hss-ops`.
- **Phase 3** — roll out to `recommended` skills (ELB, CES, LTS, CTS).
- **Phase 4 (✅ done 2026-06-04)** — wire `gcl-trace-*.json` pass-rate into CES alarms; refine thresholds from real incident data. Added `huaweicloud-ces-ops/references/gcl-monitoring.md` with trace parser, 4 alarm rules, and SDK custom metric push (namespace `CUSTOM.GCL`). Phase 4 closes the loop: GCL pass-rate is now observable via CES metrics and alarms.

### 12. Changelog

| Version | Date | Change |
|---|---|---|
| 1.0.0 | 2026-06-04 | Initial GCL specification added to `AGENTS.md` (Correctness threshold relaxed to ≥0.5; pilot scoped to `huaweicloud-ecs-ops`; per-skill defaults set for all current skills) |
| 1.1.0 | 2026-06-04 | `huaweicloud-ecs-ops` GCL pilot rollout: added `references/rubric.md` (v1, 5-dim, S1–S10 ECS-specific Safety rules) and `references/prompt-templates.md` (Generator + Critic + Orchestrator skeletons); `## Quality Gate (GCL)` chapter inserted in `SKILL.md`; frontmatter `metadata.gcl` block; `.gitignore` excludes `audit-results/` and `gcl-trace-*.json`. Per-skill defaults table corrected (removed non-existent `evs-ops` / `eip-ops`, added real `vpc-ops` / `gaussdb-ops`). Phase 1 closed. |
| 1.2.0 | 2026-06-04 | Phase 2 rollout to 4 highest-blast-radius `required` skills: `huaweicloud-iam-ops` (S1–S14, AK/policy/agency safety), `huaweicloud-rds-ops` (S1–S15, DDL/parameter durability), `huaweicloud-vpc-ops` (S1–S17, SG wide-open / EIP orphan / NAT cascade), `huaweicloud-gaussdb-ops` (S1–S17, with flavor-gated S12/S13 for DWS shard rebalance). Each skill gains `references/rubric.md` + `references/prompt-templates.md` + `## Quality Gate (GCL)` chapter + frontmatter `metadata.gcl` block + GCL refs in `Reference Directory` / `## References`. Phase 2 closed; 5 of 15 required skills now GCL-enabled (ecs + iam + rds + vpc + gaussdb). |
| 1.3.0 | 2026-06-04 | Phase 3 full-remaining rollout. **10 remaining required skills** (`huaweicloud-dcs-ops`, `huaweicloud-dms-ops`, `huaweicloud-css-ops`, `huaweicloud-cce-ops`, `huaweicloud-cbr-ops`, `huaweicloud-obs-ops`, `huaweicloud-swr-ops`, `huaweicloud-functiongraph-ops`, `huaweicloud-waf-ops`, `huaweicloud-hss-ops`) fully GCL-enabled. **4 recommended skills** (`huaweicloud-elb-ops` S1–S13, `huaweicloud-ces-ops` S1–S10, `huaweicloud-lts-ops` S1–S9, `huaweicloud-cts-ops` S1–S8) GCL-enabled with max_iter=3. **1 optional skill** (`huaweicloud-billing-ops` S1–S7) GCL-enabled with max_iter=5. All 20 skills in the repository now have complete GCL artifacts: frontmatter `metadata.gcl` block, `## Quality Gate (GCL)` chapter, `references/rubric.md`, and `references/prompt-templates.md`. |
| 1.4.0 | 2026-06-04 | **Phase 4 launch** — GCL monitoring design. Added `huaweicloud-ces-ops/references/gcl-monitoring.md` with: `gcl-pass-rate-parser.sh` (bash parser for `audit-results/gcl-trace-*.json`), 4 CES alarm rule definitions (`gcl-overall-pass-rate-critical`, `gcl-safety-fail-detected`, `gcl-max-iter-rate-warning`, `gcl-ecs-pass-rate-major`), Go SDK custom metric push script (namespace `CUSTOM.GCL`), dashboard recommendations (5 widgets), and threshold optimization guidance. Updated `huaweicloud-ces-ops/SKILL.md` — Trigger & Scope includes GCL monitoring trigger. Phase 4 closes the loop: GCL pass-rate is now observable via CES metrics and alarms. |

### 13. See also

- Each skill's `SKILL.md` → `## Quality Gate (GCL)` chapter — the per-skill rubric instance
- Each skill's `references/prompt-templates.md` — the G/C/O prompt skeletons
- `huaweicloud-skill-generator/references/huaweicloud-skill-template.md` — template that now ships with a GCL stub section to be filled in by the generator
