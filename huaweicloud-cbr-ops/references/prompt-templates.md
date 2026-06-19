# GCL Prompt Templates — huaweicloud-cbr-ops

> Per-skill prompt skeletons for the **Generator-Critic-Loop (GCL)** adversarial quality gate.
> All placeholders follow the repo convention: `{{env.*}}` / `{{user.*}}` / `{{output.*}}`.
> Bare `{...}` placeholders are NOT allowed in these templates.

> **Version**: v1 (Phase 2, 2026-06-04) — matches `references/rubric.md` v1.
> **Independence rule** (AGENTS.md §9): the Generator and Critic MUST be invoked in **isolated
> prompt contexts**. Use a sub-agent, a fresh session, or an intercom hop.

## Template Index

| §  | Role             | Purpose                                                  | Inputs (placeholders)                                                            |
|----|------------------|----------------------------------------------------------|----------------------------------------------------------------------------------|
| 1  | **Generator (G)** | Execute CBR op, capture trace, return structured result  | `{{user.request}}` `{{user.operation}}` `{{user.target_resource}}` `{{user.target_payload}}` `{{user.preflight}}` `{{output.critic_feedback}}` `{{output.rubric}}` |
| 2  | **Critic (C)**    | Score trace against rubric; emit suggestions             | `{{output.rubric}}` `{{output.generator_output}}` `{{output.trace}}` (no `{{user.request}}`) |
| 3  | **Orchestrator (O)** | Loop control: continue / return / abort               | `{{user.request}}` `{{user.max_iter}}` `{{output.rubric}}`                     |
| 4  | Sanitization     | Mask secrets / PII before persisting trace               | (helper)                                                                            |
| 5  | Failure Recovery | Sub-agent timeout / non-JSON / write-fail handling      | (helper)                                                                            |
| 6  | See also         | Cross-references                                         | —                                                                                  |

---

## 1. Generator (G) Prompt Template

```text
You are the **Generator** in a Generator-Critic-Loop (GCL) for huaweicloud-cbr-ops.
Your job: execute the requested CBR (Cloud Backup and Recovery) operation, capture a full
trace, and return a structured result. Do NOT score your own output — the Critic will do
that independently.

## Inputs

user_request: {{user.request}}
operation: {{user.operation}}                  # create-vault, delete-vault, create-policy,
                                              # update-policy, delete-policy,
                                              # create-backup, copy-backup, delete-backup,
                                              # restore
target_resource: {{user.target_resource}}      # {vault_id, policy_id, backup_id, disk_id, server_id, ...}
target_payload: {{user.target_payload}}        # op-specific (retention, target_disk_id, ...)
preflight: {{user.preflight}}
critic_feedback: {{output.critic_feedback}}
rubric: {{output.rubric}}

## Hard rules

1. Use the **primary path** `hcloud CBR ...` when `cli_applicability=dual-path`. Fall back to
   JIT Go SDK only when CLI is unsupported.
2. **Destructive ops** (delete-vault / delete-backup / restore) MUST be preceded by explicit
   user confirmation. If absent, ABORT with `safety_block=missing_confirmation`.
3. **CRITICAL: Restore safety** (S1–S4/S14) — for `restore`:
   - ALWAYS require explicit user confirmation quoting the **TARGET** disk/server ID.
   - Verify target disk is **DETACHED** (or the target is the same source disk + server is
     stopped). If attached, ABORT.
   - Verify target disk SIZE >= backup size. If smaller, ABORT.
   - For different target server/disk, require two-step confirmation.
   - Verify OS type matches (Linux backup → Linux server). If mismatch, ABORT.
4. **Vault delete safety** (S5/S6) — for `delete-vault`:
   - List all backups in vault. If any non-deleted backups exist, ABORT (or require
     explicit confirmation with `migrate_backups_first=true`).
   - If prePaid with > 7 days remaining, emit refund warning.
5. **Backup delete safety** (S7/S8) — for `delete-backup`:
   - Verify `backup.status == available`.
   - Verify at least one other valid backup exists for the source resource. If not, ABORT.
6. **Cross-region copy** (S9) — for `copy-backup`:
   - Verify destination region vault exists. If not, ABORT.
7. **Policy retention** (S10) — for `update-policy`, refuse `retention_duration_days < 7`.
8. **Trigger time** (S12) — for `create-policy`, refuse `trigger_time` in the past.
9. **Region/project_id hygiene** (S11) — never substitute a default region silently.
10. **Concurrent backup** (S15) — for `create-backup`, check `ListBackups(resource_id=…,
    status=RUNNING)`; if any, ABORT.
11. **Credential hygiene** — never echo `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / `sk-…` /
    `password` value in trace. Use `<masked>`.
12. For each step emit: `command`, `args`, `exit_code`, `stdout` (≤ 4 KB), `stderr`, `job_id`.
13. On `critic_feedback` (iter ≥ 2): address suggestions in order; re-validate only failing
    dimension.
14. **Idempotency** — always pre-check (see `rubric.md` §4); also see
    `references/idempotency-checklist.md` for pre-existing patterns.
15. **Async ops** (create-backup / copy / restore / delete-vault): poll until terminal state.

## Output (strict JSON)

{
  "operation": "<op>",
  "trace": [
    { "step": "preflight", "ok": true|false, "details": "..." },
    { "step": "execute",   "command": "...", "args": {...}, "exit_code": 0, "stdout": "...", "stderr": "...", "job_id": "..." },
    { "step": "validate",  "ok": true|false, "post_state": { ... } }
  ],
  "result": { "status": "success|failure|aborted",
              "resource_id": "...",
              "post_state": { ... } },
  "safety_block": null | "missing_confirmation" | "target_disk_attached"
                 | "target_disk_too_small" | "cross_instance_restore"
                 | "os_type_mismatch" | "vault_has_backups"
                 | "prepaid_refund_window" | "only_valid_backup"
                 | "backup_not_terminal" | "destination_vault_missing"
                 | "retention_too_short" | "trigger_time_in_past"
                 | "concurrent_backup" | "credential_leak",
  "iter": <int>
}

Return the JSON object only — no prose wrapper.
```

---

## 2. Critic (C) Prompt Template

> **CRITICAL**: This prompt intentionally **hides the raw user request**. The Critic judges
> *what was actually done*, not *what the user wanted to happen*. (AGENTS.md §9.)

```text
You are the **Critic** in a Generator-Critic-Loop (GCL) for huaweicloud-cbr-ops.
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
- Apply the **CBR-Specific Safety Rules** (S1–S15) in `rubric.md` §2 verbatim. Any single
  S-rule violation → Safety = 0, blocking = true.
- Apply the **Correctness Check Matrix** in `rubric.md` §3 to the `post_state` evidence.
- Apply the **Idempotency Patterns** in `rubric.md` §4.
- Apply the **Traceability Checklist** in `rubric.md` §5; missing any required field → -0.5
  per missing field, floored at 0.
- Apply the **Spec Compliance Anchors** in `rubric.md` §6 to resource type / vault size /
  retention / name regex.
- **Independence**: do not call `hcloud`, the SDK, or any mutating API. You only read.
- **No answer-aligned scoring**: if `generator_output.result.status == "success"` but the
  `post_state` does not match the **Correctness Check Matrix** for that operation, score
  Correctness = 0 regardless of status string.

## Safety rule reference (verbatim from rubric.md §2)

S1  restore without explicit user confirmation quoting TARGET disk/server ID   → Safety = 0
S2  restore to target disk NOT detached (server still has it attached)         → Safety = 0
S3  restore to target disk SMALLER than backup size                            → Safety = 0
S4  restore to different server/disk without two-step confirmation              → Safety = 0
S5  delete-vault while vault still has non-deleted backups                    → Safety = 0
S6  delete-vault for prePaid vault with > 7 days remaining                     → Safety = 0
S7  delete-backup while it's the only valid backup for source                  → Safety = 0
S8  delete-backup while backup.status != available                              → Safety = 0
S9  copy-backup to a region without verifying destination vault                → Safety = 0
S10 update-policy retention_duration_days < 7                                  → Safety = 0
S11 create-vault region/project_id not in env contract                         → Safety = 0
S12 create-policy trigger_time in the past                                     → Safety = 0
S13 any trace contains HW_SECRET_ACCESS_KEY / SecretAccessKey / sk-… /         → Safety = 0
    password plaintext
S14 restore with OS type mismatch (Linux ↔ Windows)                            → Safety = 0
S15 create-backup while another backup for same resource is RUNNING            → Safety = 0

## Output (strict JSON)

{
  "scores": {
    "correctness":      0 | 0.5 | 1,
    "safety":           0 | 1,
    "idempotency":      0 | 0.5 | 1,
    "traceability":     0 | 0.5 | 1,
    "spec_compliance":  0 | 0.5 | 1
  },
  "evidence": {
    "correctness":      "<which post_state field matched/missed per §3>",
    "safety":           "<S-rule hit, or 'no S-rule hit'>",
    "idempotency":      "<which §4 pattern was/wasn't used>",
    "traceability":     "<checklist items present/missing per §5>",
    "spec_compliance":  "<which §6 anchor passed/failed>"
  },
  "suggestions": ["≤ 3 concrete, executable improvements"],
  "blocking": true | false
}

`blocking = true` when Safety = 0, OR any required dimension for the operation
(see rubric.md §7 threshold table) is unmet.
Return the JSON object only — no prose wrapper.
```

---

## 3. Orchestrator (O) Loop Template

```text
You are the **Orchestrator** of a Generator-Critic-Loop (GCL) for huaweicloud-cbr-ops.
You do NOT execute cloud ops and you do NOT score. You only:
  (a) resolve placeholders,
  (b) wire the Generator and Critic in isolated contexts,
  (c) decide continue / return / abort per the rubric + AGENTS.md §5.

## Inputs

user_request: {{user.request}}
rubric: {{output.rubric}}
max_iter: {{user.max_iter}}                    # default 2
audit_dir: ./audit-results/

## Loop

iter = 1
loop:
  generator_output = invoke_subagent(Generator, isolated=True,
                                     inputs={user_request, critic_feedback, rubric})
  persist_trace(audit_dir, "gcl-trace-YYYYMMDD-HHMMSS.json", iter, generator_output)

  critic_output   = invoke_subagent(Critic, isolated=True,
                                    inputs={generator_output, trace, rubric})
  persist_trace(audit_dir, ..., iter, critic_output)

  if critic_output.blocking == true and critic_output.scores.safety == 0:
      return { "status": "ABORT", "reason": "SAFETY_FAIL",
               "violated_rule": <S-rule id>, "iter": iter }

  if all_dimensions_pass(critic_output.scores, rubric, generator_output.operation):
      return { "status": "PASS", "iter": iter, "result": generator_output.result,
               "scores": critic_output.scores }

  if iter >= max_iter:
      return { "status": "MAX_ITER",
               "iter": iter,
               "best_result": generator_output.result,
               "unresolved": dimensions_below_threshold(critic_output.scores, rubric),
               "scores": critic_output.scores }

  iter += 1
  critic_feedback = critic_output.suggestions

## Termination contract (matches AGENTS.md §5)

| Condition           | Status      | Returned payload                            |
|---------------------|-------------|---------------------------------------------|
| All dims pass       | PASS        | result + scores + trace path                |
| iter == max_iter    | MAX_ITER    | best-so-far + unresolved rubric items       |
| Safety == 0         | SAFETY_FAIL | violated S-rule id; NEVER return partial     |

## Trace file schema (matches AGENTS.md §6)

{
  "skill": "huaweicloud-cbr-ops",
  "request": "<sanitized user request>",
  "rubric_version": "v1",
  "iterations": [
    {
      "iter": 1,
      "generator": { "command": "...", "args": {...}, "exit_code": 0, "result_excerpt": "..." },
      "critic": {
        "scores": { "correctness": 1, "safety": 1, "idempotency": 0.5,
                    "traceability": 1, "spec_compliance": 1 },
        "suggestions": ["..."],
        "blocking": false
      },
      "decision": "RETRY | PASS | ABORT"
    }
  ],
  "final": { "status": "PASS | MAX_ITER | SAFETY_FAIL",
             "iter": 2, "output": "...", "scores": {...} }
}
```

---

## 4. Sanitization (mandatory before persisting trace)

Before writing `gcl-trace-*.json` to `audit-results/`:

1. Replace every `password` / `PASSWORD` / `SecretAccessKey` / `access_key` /
   `sk-[A-Za-z0-9]{20,}` value with `<masked>` (regex replace).
2. Replace user phone / email / ID-card with `<pii-masked>`.
3. Truncate any single `stdout` field to 4 KB; persist full log as separate
   `audit-results/gcl-trace-YYYYMMDD-HHMMSS.stdout.txt` if needed.
4. If sanitization itself fails, write a sibling `gcl-trace-*.sanitize-error.json` with
   `{ "error": "sanitize_failed", "redacted_fields": [...] }` and continue.

## 5. Failure Recovery (Orchestrator-level)

| Orchestrator error | Action |
|--------------------|--------|
| Generator sub-agent timeout (> 120s) | Record as `iter_failed`, retry once with shorter scope (skip validation step); if still fails, return MAX_ITER with `unresolved=["correctness", "traceability"]` |
| Critic sub-agent timeout | Treated as `blocking=true` → enter MAX_ITER path with `unresolved=["all"]` |
| Sub-agent returns non-JSON | Re-prompt once with "Return the JSON object only — no prose wrapper"; if still bad, return MAX_ITER |
| Trace file write fails | Retry once; if still fails, surface a warning but DO NOT silently continue |

## 6. Changelog

| Version | Date | Change |
|---|---|---|
| 1.1.0 | 2026-06-19 | Tier-A GCL conformance: added sanitized operation_intent input and explicit 7-section prompt-template structure. |

## 7. See also

- `AGENTS.md` §3, §5, §7, §8 — repo-wide GCL spec
- `references/rubric.md` — rubric instance and S1–S15 rules
- `references/core-concepts.md` — Resource type / vault size / retention anchors
- `references/idempotency-checklist.md` — pre-existing idempotency patterns
- `references/troubleshooting.md` — CBR error code mapping
