# Generator-Critic-Loop (GCL) — Adversarial Quality Gate

> Inspired by GAN's Generator/Discriminator idea, but deliberately **not** a real GAN.
> Naming: **GCL (Generator-Critic-Loop)** to avoid misleading reviewers and LLM trainees.
> This document is the detailed runtime spec for Huawei Cloud (`hcloud` CLI / Go SDK fallback).

## 1. Purpose

Apply an adversarial **Generator ↔ Critic** loop with a quantitative rubric to every skill execution.
Most valuable in high-side-effect cloud operations (`delete`, `stop`, `restore`, IAM/KMS/DDL) where a single mistake is unrecoverable.

| GAN (real) | GCL (this spec) |
|---|---|
| Discriminator learns sample distribution | Critic scores an **explicit rubric** |
| No termination condition | Must terminate: **PASS / MAX_ITER / SAFETY_FAIL** |
| G and D train in parallel | G and C run **sequentially** |
| Goal: "fool the D" | Goal: "pass the rubric threshold" |

## 2. Roles

| Role | Job | Input | Output | Forbidden |
|---|---|---|---|---|
| **Generator (G)** | Execute the cloud operation | user request + previous Critic feedback | result + execution trace | modifying rubric; self-scoring |
| **Critic (C)** | Independently audit output | generator result + trace + rubric + sanitized operation intent | scores + suggestions | calling `hcloud`, SDK clients, or mutating resources |
| **Orchestrator (O)** | Loop control | context + Critic scores + budget | continue / final result | executing or scoring on its own |

**Hard constraint:** Generator and Critic MUST run in isolated prompt contexts. Shared-context G+C is banned.

## 3. Rubric

Each required/recommended skill keeps:

- `## Quality Gate (GCL)` in `SKILL.md`
- `references/rubric.md`
- `references/prompt-templates.md`

Minimum dimensions:

| Dimension | Meaning | Scale | Default threshold |
|---|---|---|---|
| **Correctness** | Resource id / state / config actually matches the request | 0 / 0.5 / 1 | ≥ 0.5; 1.0 for destructive/IAM/KMS/DDL |
| **Safety** | Destructive op was confirmed or guarded | 0 / 1 | = 1 |
| **Idempotency** | Retry does not duplicate side-effects | 0 / 0.5 / 1 | ≥ 0.5 |
| **Traceability** | Command, params, raw response, errors captured | 0 / 0.5 / 1 | ≥ 0.5 |
| **Spec Compliance** | Conforms to `core-concepts.md` / `cli-usage.md` constraints | 0 / 0.5 / 1 | ≥ 0.5 |

**Safety = 0 → ABORT immediately**, regardless of total score.

## 4. Loop Flow

```text
User Request
  ↓
[0] Pre-flight (Orchestrator)
  - resolve env.* and user.* variables
  - pick skill, load rubric
  - derive sanitized operation_intent: operation, expected_state, resource_scope, safety_class
  - omit raw user wording, credentials, and unmasked sensitive identifiers
  ↓
[1] Generate (G)
  - run hcloud or JIT Go SDK fallback
  - capture command/args/exit/raw response/request_id/job_id
  ↓
[2] Critique (C)
  - isolated prompt context
  - score rubric dimensions
  - emit ≤3 concrete suggestions
  ↓
[3] Decide (O)
  - Safety=0 → SAFETY_FAIL
  - all thresholds pass → PASS
  - else and iter<max_iter → retry with critic_feedback
  - else → MAX_ITER
```

The Orchestrator owns `operation_intent` generation. Critic MUST NOT see raw user wording; it may use `{{output.operation_intent}}`.

## 5. Termination

| Condition | Behavior |
|---|---|
| **PASS** | Every rubric dimension meets threshold → return result |
| **MAX_ITER** | Max iterations reached → return best-so-far + unresolved rubric items |
| **SAFETY_FAIL** | Safety = 0 → abort; never return partial or best-effort output |

## 6. Trace & Audit Schema

Every GCL run MUST persist a masked JSON trace under `audit-results/gcl-trace-YYYYMMDD-HHMMSS.json`.

```json
{
  "trace_schema_version": "v1",
  "skill": "huaweicloud-ecs-ops",
  "request": "<sanitized user request>",
  "operation_intent": {
    "operation": "stop-server",
    "resource_scope": ["ecs-***"],
    "expected_state": "SHUTOFF",
    "safety_class": "destructive"
  },
  "rubric_version": "v1",
  "masked_fields": ["request", "operation_intent.resource_scope"],
  "iterations": [
    {
      "iter": 1,
      "generator": { "command": "...", "args": {}, "exit_code": 0, "result_excerpt": "..." },
      "critic": {
        "scores": {
          "correctness": 1,
          "safety": 1,
          "idempotency": 0.5,
          "traceability": 1,
          "spec_compliance": 1
        },
        "suggestions": ["..."],
        "blocking": false
      },
      "decision": "PASS"
    }
  ],
  "final": {
    "status": "PASS",
    "iter": 1,
    "output": "...",
    "failure_pattern": null
  }
}
```

Trace files are append-only; do not overwrite/delete in place. `audit-results/` and `gcl-trace-*.json` are gitignored.

## 7. Prompt Templates

Each skill's `references/prompt-templates.md` MUST contain numbered sections `## 1.` through `## 7.` and include:

1. Generator Prompt Template — placeholders include `{{user.request}}`, `{{output.critic_feedback}}`, `{{output.rubric}}`
2. Critic Prompt Template — placeholders include `{{output.operation_intent}}`, `{{output.generator_output}}`, `{{output.trace}}`, `{{output.rubric}}`
3. Orchestrator Loop Template
4. Sanitization rules
5. Failure recovery
6. Changelog
7. See also

Placeholder syntax MUST follow `{{env.*}}` / `{{user.*}}` / `{{output.*}}`; bare `{...}` placeholders are banned.

## 8. Per-Skill Defaults

| Skill | GCL | max_iter | Notes |
|---|---|---:|---|
| `huaweicloud-ecs-ops` | required | 2 | delete/stop/reboot |
| `huaweicloud-iam-ops` | required | 2 | detach policy / delete user / rotate keys |
| `huaweicloud-rds-ops` | required | 2 | delete / DDL / restore |
| `huaweicloud-gaussdb-ops` | required | 2 | delete / DDL / shard rebalance |
| `huaweicloud-dcs-ops` | required | 2 | FLUSHALL / delete / restore |
| `huaweicloud-dms-ops` | required | 2 | queue delete / message purge |
| `huaweicloud-css-ops` | required | 2 | cluster delete / snapshot restore |
| `huaweicloud-cce-ops` | required | 2 | node drain / cluster delete |
| `huaweicloud-cbr-ops` | required | 2 | restore overwrites source |
| `huaweicloud-vpc-ops` | required | 2 | delete VPC/subnet/SG cascades |
| `huaweicloud-obs-ops` | required | 2 | bucket delete / lifecycle purge |
| `huaweicloud-swr-ops` | required | 2 | image delete / tag overwrite |
| `huaweicloud-functiongraph-ops` | required | 2 | function delete / version disable |
| `huaweicloud-waf-ops` | required | 2 | policy delete / rule disable |
| `huaweicloud-hss-ops` | required | 2 | host isolate / policy detach |
| `huaweicloud-elb-ops` | recommended | 3 | listener/backend/cert changes |
| `huaweicloud-ces-ops` | recommended | 3 | alarm rule delete |
| `huaweicloud-lts-ops` | recommended | 3 | log group/stream delete |
| `huaweicloud-cts-ops` | recommended | 3 | tracker disable / transfer delete |
| `huaweicloud-billing-ops` | optional | 5 | read-only reports |
| `huaweicloud-skill-generator` | optional | 3 | meta operation |

## 9. Runtime Scripts

| Script | Purpose |
|---|---|
| `scripts/gcl_runner.py` | Orchestrator loop; external Critic required in production |
| `scripts/gcl_trace_aggregate.py` | Aggregate traces into quality summary |
| `scripts/gcl_alarm_wire.py` | Plan/apply CES alarms from summary |
| `scripts/check_gcl_conformance.py` | Verify per-skill GCL artifacts |
| `scripts/check_markdown_links.py` | Validate top-level local path references |
| `scripts/validate_local.py` | One-command local validation suite |

Production GCL MUST use externally supplied isolated Critic scores via `--critic-json` or stdin. `--structural-critic-only` is only for CI/local smoke tests and cannot approve production or human acceptance gates.

## 10. Anti-Patterns

- Shared context G+C
- Subjective scoring instead of rubric scoring
- Unbounded loop
- Critic sees raw user request
- Safety fail silently downgraded
- Trace not persisted
- Critic mutates resources
- Structural critic used as production quality pass
- Printing/logging credentials

## 11. Monitoring Integration

GCL quality summaries are owned by `huaweicloud-ces-ops`:

- Schema: `huaweicloud-ces-ops/assets/gcl-quality-summary.schema.json`
- Design: `huaweicloud-ces-ops/references/gcl-monitoring.md`
- Namespace: `CUSTOM.GCL`
- Alarm plan: `scripts/gcl_alarm_wire.py plan --summary <summary.json>`

## 12. Changelog

| Version | Date | Change |
|---|---|---|
| 1.0.0 | 2026-06-04 | Initial GCL specification |
| 1.1.0 | 2026-06-04 | ECS pilot rollout |
| 1.2.0 | 2026-06-04 | Phase 2 rollout to high-blast-radius skills |
| 1.3.0 | 2026-06-04 | All 20 skills gained GCL artifacts |
| 1.4.0 | 2026-06-04 | CES monitoring design |
| 1.5.0 | 2026-06-05 | Moved detailed spec to `docs/gcl-spec.md` |
| 1.6.0 | 2026-06-19 | Added qcloud-style runtime scripts, sanitized `operation_intent`, Tier-A conformance, and CES quality-summary contract |

## 13. See also

- `AGENTS.md` — always-loaded GCL hard constraints and validation pointers
- `huaweicloud-*-ops/references/rubric.md` — per-skill scoring rubrics
- `huaweicloud-*-ops/references/prompt-templates.md` — G/C/O templates
- `huaweicloud-ces-ops/references/gcl-monitoring.md` — CES monitoring design
