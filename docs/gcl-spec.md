# GCL — Generator-Critic-Loop: Complete Specification

> Full implementation spec referencing `AGENTS.md §Generator-Critic-Loop`.
> Moved here from AGENTS.md for TE-6/TE-7 compliance — see AGENTS.md `## Token Efficiency Requirements`.

---

## 1. GAN / GCL Comparison

| GAN (real) | GCL (this spec) |
|---|---|
| Discriminator learns sample distribution | Critic scores an **explicit rubric** |
| No termination condition | Must terminate: **PASS / MAX_ITER / SAFETY_FAIL** |
| G and D train in parallel | G and C run **sequentially** |
| Goal: "fool the D" | Goal: "pass the rubric threshold" |

## 2. Loop Flow

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

## 3. Termination (first match wins)

| Condition | Behavior |
|---|---|
| **PASS** | Every rubric dimension meets its threshold → return G's result |
| **MAX_ITER** | Reached `max_iterations` → return **best-so-far** + unresolved rubric items |
| **SAFETY_FAIL** | Safety = 0 → **ABORT**; never return partial or "best-effort" output |

## 4. Trace & Audit Schema

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

Path: `./audit-results/gcl-trace-YYYYMMDD-HHMMSS.json` — must be in `.gitignore`. Trace files are **append-only**; never overwrite or delete in place.

## 5. Prompt Templates

Each skill's `references/prompt-templates.md` MUST contain:

1. **Generator Prompt Template** — placeholders: `{{user.request}}`, `{{output.critic_feedback}}`, `{{output.rubric}}`
2. **Critic Prompt Template** — placeholders: `{{output.generator_output}}`, `{{output.trace}}`, `{{output.rubric}}`

> Placeholder syntax MUST follow `{{env.*}}` / `{{user.*}}` / `{{output.*}}` convention. Bare `{...}` is NOT allowed.

**Critic prompt must hide the raw user request** to prevent rubber-stamping. Recommended skeleton:

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

## 6. Relationship to Existing Quality Gates

```
┌─────────────────────────────────────────────┐
│  GCL  — runtime, per-op                     │  ← NEW
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

- **Skill creation/update** → P0 → P1 → 2-round self-reflection.
- **Skill execution at runtime** → additionally GCL when skill class is `required`/`recommended`.

## 7. Rollout Roadmap

| Phase | Date | Scope |
|-------|------|-------|
| **Phase 1** | 2026-06-04 | GCL spec in `AGENTS.md`; pilot `huaweicloud-ecs-ops` |
| **Phase 2** | 2026-06-04 | 4 required skills: iam, rds, vpc, gaussdb |
| **Phase 3** | 2026-06-04 | Remaining 10 required + 4 recommended + 1 optional skills |
| **Phase 4** | 2026-06-04 | GCL monitoring via CES (`CUSTOM.GCL` namespace, alarms, dashboards) |

## 8. Changelog

| Version | Date | Change |
|---|---|---|
| 1.0.0 | 2026-06-04 | Initial GCL specification |
| 1.1.0 | 2026-06-04 | `huaweicloud-ecs-ops` GCL pilot rollout |
| 1.2.0 | 2026-06-04 | Phase 2 rollout (iam, rds, vpc, gaussdb) |
| 1.3.0 | 2026-06-04 | Phase 3 full-remaining rollout (all 20 skills) |
| 1.4.0 | 2026-06-04 | Phase 4 — GCL monitoring via CES |
| 1.5.0 | 2026-06-05 | Moved detailed GCL spec to `docs/gcl-spec.md` for TE-6/TE-7 compliance |

## 9. See also

- `AGENTS.md §Generator-Critic-Loop` — operational summary (roles, rubric, per-skill defaults, anti-patterns)
- Per-skill `references/prompt-templates.md` — G/C/O prompt skeletons
- `huaweicloud-skill-generator/references/huaweicloud-skill-template.md` — skill template with GCL stub
- `huaweicloud-ces-ops/references/gcl-monitoring.md` — GCL metric collection and alarms