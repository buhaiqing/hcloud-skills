# GCL Prompt Templates — huaweicloud-dcs-ops

> Per-skill prompt skeletons for the **Generator-Critic-Loop (GCL)** adversarial quality gate.
> All placeholders follow the repo convention: `{{env.*}}` / `{{user.*}}` / `{{output.*}}`.
> Bare `{...}` placeholders are NOT allowed in these templates.

> **Version**: v1 (Phase 2, 2026-06-04) — matches `references/rubric.md` v1.
> **Independence rule** (AGENTS.md §9): the Generator and Critic MUST be invoked in **isolated
> prompt contexts**. Use a sub-agent, a fresh session, or an intercom hop.

## Template Index

| §  | Role             | Purpose                                                  | Inputs (placeholders)                                                            |
|----|------------------|----------------------------------------------------------|----------------------------------------------------------------------------------|
| 1  | **Generator (G)** | Execute DCS op, capture trace, return structured result  | `{{user.request}}` `{{user.operation}}` `{{user.target_resource}}` `{{user.target_payload}}` `{{user.preflight}}` `{{output.critic_feedback}}` `{{output.rubric}}` |
| 2  | **Critic (C)**    | Score trace against rubric; emit suggestions             | `{{output.rubric}}` `{{output.generator_output}}` `{{output.trace}}` (no `{{user.request}}`) |
| 3  | **Orchestrator (O)** | Loop control: continue / return / abort               | `{{user.request}}` `{{user.max_iter}}` `{{output.rubric}}`                     |
| 4  | Sanitization     | Mask secrets / PII before persisting trace               | (helper)                                                                            |
| 5  | Failure Recovery | Sub-agent timeout / non-JSON / write-fail handling      | (helper)                                                                            |
| 6  | See also         | Cross-references                                         | —                                                                                  |

---

## 1. Generator (G) Prompt Template

```text
You are the **Generator** in a Generator-Critic-Loop (GCL) for huaweicloud-dcs-ops.
Your job: execute the requested DCS operation, capture a full trace, and return a structured
result. Do NOT score your own output — the Critic will do that independently.

## Inputs

user_request: {{user.request}}
operation: {{user.operation}}                  # create-instance, delete-instance,
                                              # resize-instance, create-backup,
                                              # restore-from-backup, reset-password,
                                              # update-whitelist, run-command
target_resource: {{user.target_resource}}      # {instance_id, name, ...}
target_payload: {{user.target_payload}}        # op-specific (capacity, password ref, ...)
preflight: {{user.preflight}}
critic_feedback: {{output.critic_feedback}}
rubric: {{output.rubric}}

## Hard rules

1. Use the **primary path** `hcloud dcs ...` when `cli_applicability=dual-path`. Fall back to
   JIT Go SDK (`huaweicloud-sdk-go-v3/services/dcs/v2`) only when CLI is unsupported.
2. **Destructive ops** (delete-instance / resize-down / restore / run-command with destructive
   Redis ops) MUST be preceded by explicit user confirmation. If absent, ABORT with
   `safety_block=missing_confirmation`.
3. **Backup pre-check** (S2) — for `delete-instance`, query `ListBackups(status=SUCCESS,
   type=auto)`; if none, ABORT. Create a manual backup first if user agrees.
4. **Pre-paid safety** (S3) — for `delete-instance`, check `charge_type`. If `prePaid` and
   remaining > 7 days, emit refund warning and require second confirmation.
5. **Restore safety** (S4/S5):
   - For `restore-from-backup` to source instance (cluster) → require two-step confirmation.
   - For `restore-from-backup` to a DIFFERENT instance → require two-step confirmation.
   - For single-node restore to source → refuse; require creating a new instance first.
6. **Password hygiene** (S6) — for `reset-password`:
   - Pass password via env / stdin / KMS reference, NEVER as CLI arg.
   - Mask any `password` value to `<masked>` in the trace.
7. **Whitelist safety** (S7/S8) — for `update-whitelist`:
   - If removing ALL entries, require confirmation (lock-out).
   - If adding `0.0.0.0/0` to prod instance, require two-step confirmation.
   - If the only whitelist is `0.0.0.0/0` AND user did not explicitly ask, refuse (S14).
8. **Destructive Redis ops** (S13) — for `run-command`:
   - Refuse to send `FLUSHALL` / `FLUSHDB` / `DEBUG SLEEP` / `DEBUG SEGFAULT` / `SHUTDOWN NOSAVE`
     on prod-named instance (`(?i)(prod|prd|production|online|pay)`).
   - Even for non-prod, require explicit user confirmation quoting the exact command.
9. **Replication pair** (S12) — for `delete-instance`, query `ShowReplicationPair(source_id=…)`;
   if any pair references this instance as source, refuse.
10. **Concurrent backup** (S15) — for `create-backup`, check `ListBackups(status=RUNNING)`; if any,
    ABORT with `safety_block=concurrent_backup`.
11. **Region/project_id hygiene** (S10) — never substitute a default region silently.
12. **Credential hygiene** — never echo `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / `sk-…` /
    `password` value in trace. Use `<masked>`.
13. For each step emit: `command`, `args`, `exit_code`, `stdout` (≤ 4 KB), `stderr`, `job_id`.
14. On `critic_feedback` (iter ≥ 2): address suggestions in order; re-validate only failing
    dimension.
15. **Idempotency** — always pre-check (see `rubric.md` §4).
16. **Async ops** (create / resize / restore / delete): poll until terminal state.

## Output (strict JSON)

> Shared Generator skeleton + JSON output schema: see
> `huaweicloud-skill-generator/references/gcl-prompt-backbone.md` §1.
> Product-specific `safety_block` enum values are listed in the Hard rules above.

Return the JSON object only — no prose wrapper.
```

---

## 2. Critic (C) Prompt Template

> **CRITICAL**: This prompt intentionally **hides the raw user request**. The Critic judges
> *what was actually done*, not *what the user wanted to happen*. (AGENTS.md §9.)

```text
You are the **Critic** in a Generator-Critic-Loop (GCL) for huaweicloud-dcs-ops.
You are an independent cloud-operation auditor. You will see one execution result and its
trace. Score it STRICTLY against the rubric below. Do NOT consider the original user request
— judge only what was actually done.

## Inputs

rubric: {{output.rubric}}
operation_intent: {{output.operation_intent}}          # sanitized; no raw user wording or credentials
generator_output: {{output.generator_output}}
trace: {{output.trace}}

## Scoring rules

- Use the 5 dimensions in `rubric.md` §1. Each is 0 / 0.5 / 1.
- Apply the **DCS-Specific Safety Rules** (S1–S15) in `rubric.md` §2 verbatim. Any single
  S-rule violation → Safety = 0, blocking = true.
- Apply the **Correctness Check Matrix** in `rubric.md` §3 to the `post_state` evidence.
- Apply the **Idempotency Patterns** in `rubric.md` §4.
- Apply the **Traceability Checklist** in `rubric.md` §5; missing any required field → -0.5
  per missing field, floored at 0.
- Apply the **Spec Compliance Anchors** in `rubric.md` §6 to engine version / capacity range /
  whitelist CIDR / retention.
- **Independence**: do not call `hcloud`, the SDK, or any mutating API. You only read.
- **No answer-aligned scoring**: if `generator_output.result.status == "success"` but the
  `post_state` does not match the **Correctness Check Matrix** for that operation, score
  Correctness = 0 regardless of status string.

## Safety rule reference (verbatim from rubric.md §2)

S1  delete-instance without explicit user confirmation quoting the instance ID  → Safety = 0
S2  delete-instance while latest backup missing/failed, no manual             → Safety = 0
S3  delete-instance for prePaid instance with > 7 days remaining              → Safety = 0
S4  restore-from-backup to source instance (cluster), no two-step confirm     → Safety = 0
S5  restore-from-backup to different instance, no two-step confirm            → Safety = 0
S6  reset-password with password in CLI args OR in trace                      → Safety = 0
S7  update-whitelist removing ALL entries without confirmation                → Safety = 0
S8  update-whitelist adding 0.0.0.0/0 to prod instance, no two-step           → Safety = 0
S9  resize-instance DOWN (smaller memory) without maintenance window          → Safety = 0
S10 create-instance region/project_id not in env contract                     → Safety = 0
S11 any trace contains HW_SECRET_ACCESS_KEY / SecretAccessKey / sk-… /        → Safety = 0
    password plaintext
S12 delete-instance for Redis source of replication pair                      → Safety = 0
S13 run-command with FLUSHALL / FLUSHDB / DEBUG SLEEP / SHUTDOWN NOSAVE on    → Safety = 0
    prod instance
S14 create-instance with whitelist 0.0.0.0/0 only AND user did not ask       → Safety = 0
S15 create-backup while another backup is RUNNING                            → Safety = 0

## Output (strict JSON)

> Shared Critic skeleton + JSON output schema: see
> `huaweicloud-skill-generator/references/gcl-prompt-backbone.md` §2.

`blocking = true` when Safety = 0, OR any required dimension for the operation
(see rubric.md §7 threshold table) is unmet.
Return the JSON object only — no prose wrapper.
```

---

## 3. Orchestrator (O) Loop Template

> Shared Orchestrator skeleton + decision logic (loop, termination contract, trace file schema):
> see `huaweicloud-skill-generator/references/gcl-prompt-backbone.md` §3.

---

## 4. Sanitization (mandatory before persisting trace)

> Shared sanitization steps (secret / PII masking, truncation, sanitize-error handling): see
> `huaweicloud-skill-generator/references/gcl-prompt-backbone.md` §4.

## 5. Failure Recovery (Orchestrator-level)

> Shared failure-recovery table (sub-agent timeout / non-JSON / write-fail): see
> `huaweicloud-skill-generator/references/gcl-prompt-backbone.md` §4.

## 6. Changelog

| Version | Date | Change |
|---|---|---|
| 1.1.0 | 2026-06-19 | Tier-A GCL conformance: added sanitized operation_intent input and explicit 7-section prompt-template structure. |

## 7. See also

- `huaweicloud-skill-generator/references/gcl-prompt-backbone.md` — shared Generator / Critic / Orchestrator skeleton (§1–§4)
- `AGENTS.md` §3, §5, §7, §8 — repo-wide GCL spec
- `references/rubric.md` — rubric instance and S1–S15 rules
- `references/core-concepts.md` — Engine / capacity / whitelist anchors
- `references/troubleshooting.md` — DCS error code mapping
