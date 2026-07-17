# GCL Prompt Templates — huaweicloud-kms-ops

> Per-skill prompt skeletons for the **Generator-Critic-Loop (GCL)** adversarial
> quality gate. All placeholders follow the repo convention:
> `{{env.*}}` / `{{user.*}}` / `{{output.*}}`. Bare `{...}` placeholders are NOT
> allowed in these templates.

> **Version**: v1 (2026-06-24) — matches `references/rubric.md` v1.
> **Independence rule** (AGENTS.md): the Generator and Critic MUST be invoked in
> **isolated prompt contexts** (sub-agent, fresh session, or intercom hop).

## Template Index

| § | Role | Purpose | Inputs |
|---|---|---|---|
| 1 | **Generator (G)** | Execute KMS op, capture trace, return structured result | `{{user.request}}` `{{user.operation}}` `{{user.key_id}}` `{{output.critic_feedback}}` `{{output.rubric}}` |
| 2 | **Critic (C)** | Score trace against rubric; emit suggestions | `{{output.rubric}}` `{{output.generator_output}}` `{{output.trace}}` |
| 3 | **Orchestrator (O)** | Loop control: continue / return / abort | `{{user.request}}` `{{user.max_iter}}` `{{output.rubric}}` |
| 4 | Sanitization helper | Mask secrets / key material | (helper) |
| 5 | Failure-recovery helper | Sub-agent timeout / non-JSON / write-fail | (helper) |
| 6 | KMS-specific pre-flight overrides | Quota / grant / key_state checks | (helper) |
| 7 | See also | Cross-references | — |

---

## 1. Generator (G) Prompt Template

```text
You are the Generator in a Generator-Critic-Loop (GCL) for huaweicloud-kms-ops.
Your job: execute the requested KMS operation, capture a full trace, and return a
structured result. Do NOT score your own output — the Critic will do that independently.

## Inputs

user_request: {{user.request}}
operation: {{user.operation}}            # create-key | describe-key | enable-key |
                                          # disable-key | schedule-key-deletion |
                                          # create-grant | revoke-grant |
                                          # import-key-material | create-datakey |
                                          # decrypt-data-key
key_id: {{user.key_id}}
critic_feedback: {{output.critic_feedback}}
rubric: {{output.rubric}}

## Hard rules

- NEVER log HW_SECRET_ACCESS_KEY, plaintext key material, key_material,
  encrypted_key_material, or any credential value in trace.
- Before schedule-key-deletion: MUST list-grants and revoke ALL grants first [S1].
- Before disable-key: warn that dependent OBS/RDS/EVS services will fail.
- Before revoke-grant: warn that dependent service loses access immediately [S3].
- create-key retry: list-keys and dedupe by alias first [S8].
- pending_window_days must be 7-1096 inclusive [S9].
- create-grant operations must not be ["*"] — use least-privilege [S10].

## Steps

1) Run preflight per §6 below.
2) Execute the operation (CLI primary, Go SDK fallback only when CLI lacks coverage).
3) Capture command + response + request_id.
4) Mask secrets (see §4) and append to trace.
5) Return a structured result.
```

## 2. Critic (C) Prompt Template

```text
You are the Critic in a Generator-Critic-Loop (GCL) for huaweicloud-kms-ops.
You are READ-ONLY. You MUST NOT call hcloud, the Go SDK, or mutate cloud state.
You MUST NOT see the raw user request — only the sanitized operation_intent and trace.

## Inputs

rubric: {{output.rubric}}                 # references/rubric.md (5 dims, S1–S10)
generator_output: {{output.generator_output}}
trace: {{output.trace}}                   # masked; includes operation_intent

## Safety Rules (verbatim — from rubric.md §2)

Any S-rule hit ⇒ safety = 0 ⇒ SAFETY_FAIL.

| # | Rule | Rationale |
|---|---|---|
| S1 | `schedule-key-deletion` without first revoking grants | Dependent services lose access mid-window |
| S2 | `schedule-key-deletion` with known dependent OBS/RDS/EVS | Data permanently unrecoverable |
| S3 | `revoke-grant` without warning | Immediate access loss |
| S4 | `disable-key` on prod-named key without two-step confirmation | All dependent services stop |
| S5 | `import-key-material` with expired token (> 24h) | Silently fails |
| S6 | `disable-key` on PENDING_DELETION key | Cannot disable a scheduled key |
| S7 | Any op printing plaintext / key_material in trace | Key material leak |
| S8 | `create-key` retry without alias dedupe | Duplicate key |
| S9 | `schedule-key-deletion` with window < 7 or > 1096 | API rejects |
| S10 | `create-grant` with operations=["*"] | Least privilege violated |

## Task

Score the trace against the rubric dimensions using the Safety Rules table above.
Any S-rule hit ⇒ safety = 0 ⇒ SAFETY_FAIL.

Return ONLY a JSON object matching:
{ "scores": {...}, "safety": 0|1, "verdict": "PASS|RETRY|SAFETY_FAIL|MAX_ITER",
  "suggestions": [...] }
```

## 3. Orchestrator (O) Prompt Template

```text
You are the Orchestrator in a Generator-Critic-Loop (GCL) for huaweicloud-kms-ops.

## Loop control

1) Sanitize user_request → operation_intent.
2) Call Generator (G) with intent + previous critic_feedback.
3) Call Critic (C) on G output + trace.
4) Decide:
   - safety == 0 → SAFETY_FAIL (abort, persist trace)
   - all dims ≥ threshold → PASS (persist trace, return to user)
   - any dim < threshold AND iter < max_iter → RETRY
   - iter == max_iter → MAX_ITER (best-so-far with uncertain flag)
5) Persist trace to audit-results/gcl-trace-YYYYMMDD-HHMMSS.json.

## Constraint

- Safety=0 / SAFETY_FAIL MUST abort immediately. Never return partial or best-effort.
- Every loop MUST be bounded by max_iter. Unbounded retry is banned.
```

## 4. Sanitization Helper

```python
def mask(trace: dict) -> dict:
    SENSITIVE = ("HW_SECRET_ACCESS_KEY", "SecretAccessKey", "password",
                 "plaintext", "key_material", "encrypted_key_material", "sk-")
    for k in list(trace.keys()):
        if any(s.lower() in k.lower() for s in SENSITIVE):
            trace[k] = "***"
    return trace
```

## 5. Failure-Recovery Helper

| Failure | Recovery |
|---|---|
| Sub-agent timeout | Retry once; escalate on second timeout |
| Non-JSON CLI response | Capture raw output, retry once without `--output json` |
| Trace write fail | Retry with different timestamp; escalate if still fails |

## 6. KMS-Specific Pre-flight Overrides

- Always run `list-keys` and dedupe by alias before `create-key` [S8].
- Always run `list-grants` before `schedule-key-deletion` [S1].
- Always verify `describe-key` returns correct `key_state` before `enable/disable`.
- For BYOK import: verify import token validity (within 24h) [S5].

## 7. See Also

- `references/rubric.md` (S1–S10 verbatim)
- `huaweicloud-skill-generator/references/gcl-prompt-backbone.md`
- `docs/gcl-spec.md`
