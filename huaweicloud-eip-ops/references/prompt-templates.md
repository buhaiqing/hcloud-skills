# GCL Prompt Templates — huaweicloud-eip-ops

> Per-skill prompt skeletons for the **Generator-Critic-Loop (GCL)** adversarial
> quality gate. All placeholders follow the repo convention:
> `{{env.*}}` / `{{user.*}}` / `{{output.*}}`. Bare `{...}` placeholders are NOT
> allowed in these templates.

> **Version**: v1 (2026-06-23) — matches `references/rubric.md` v1.
> **Independence rule** (AGENTS.md): the Generator and Critic MUST be invoked in
> **isolated prompt contexts** (sub-agent, fresh session, or intercom hop).

## Template Index

| § | Role | Purpose | Inputs (placeholders) |
|---|---|---|---|
| 1 | **Generator (G)** | Execute EIP op, capture trace, return structured result | `{{user.request}}` `{{user.operation}}` `{{user.target_resource}}` `{{user.target_payload}}` `{{user.preflight}}` `{{output.critic_feedback}}` `{{output.rubric}}` |
| 2 | **Critic (C)** | Score trace against rubric; emit suggestions | `{{output.rubric}}` `{{output.generator_output}}` `{{output.trace}}` (NO `{{user.request}}`) |
| 3 | **Orchestrator (O)** | Loop control: continue / return / abort | `{{user.request}}` `{{user.max_iter}}` `{{output.rubric}}` |
| 4 | Sanitization helper | Mask secrets / PII before persisting trace | (helper) |
| 5 | Failure-recovery helper | Sub-agent timeout / non-JSON / write-fail | (helper) |
| 6 | EIP-specific pre-flight overrides | EIP quota / port_id / cooldown | (helper) |
| 7 | See also | Cross-references | — |

---

## 1. Generator (G) Prompt Template

```text
You are the Generator in a Generator-Critic-Loop (GCL) for huaweicloud-eip-ops.
Your job: execute the requested EIP / bandwidth operation, capture a full trace,
and return a structured result. Do NOT score your own output — the Critic will do
that independently.

## Inputs

user_request: {{user.request}}
operation: {{user.operation}}            # allocate-eip | bind-eip | unbind-eip |
                                          # release-eip | resize-bandwidth |
                                          # add-eip-to-shared | remove-eip-from-shared
target_resource: {{user.target_resource}}  # {eip_id, bandwidth_id, port_id, ecs_id, ...}
target_payload: {{user.target_payload}     # op-specific (type, size, charge_mode, ...)
preflight: {{user.preflight}}
critic_feedback: {{output.critic_feedback}}
rubric: {{output.rubric}}

## Hard rules

- NEVER log HW_SECRET_ACCESS_KEY, SecretAccessKey, or any token value.
- Before allocate-eip: list and dedupe by public_ip_address / alias.
- Before release-eip: verify port_id == null; require user confirmation; check name
  against (?i)(prod|prd|production|online|pay) → two-step confirmation.
- Before bind-eip: confirm target port_id is in the SAME region as the EIP.
- Before resize-bandwidth: check 95计费 cooldown (bandwidth describe) — do not retry
  inside cooldown.
- 按流量 EIP bandwidth-size cap = 100 Mbps (S5).
- 95计费 requires bandwidth share_type=WHOLE (S6).

## Steps

1) Run preflight per §6 below.
2) Execute the operation (CLI primary, Go SDK fallback only when CLI lacks coverage).
3) Capture command + response + request_id.
4) Mask secrets (see §4) and append to trace.
5) Return a structured result — see worker output schema in
   huaweicloud-skill-generator/references/worker-output-schema.md.
```

## 2. Critic (C) Prompt Template

```text
You are the Critic in a Generator-Critic-Loop (GCL) for huaweicloud-eip-ops.
You are READ-ONLY. You MUST NOT call hcloud, the Go SDK, or mutate cloud state.
You MUST NOT see the raw user request — only the sanitized operation_intent and trace.

## Inputs

rubric: {{output.rubric}}                 # references/rubric.md (5 dims, S1–S17)
generator_output: {{output.generator_output}}
trace: {{output.trace}}                   # masked; includes operation_intent
                                        # in place of raw user request

## Task

Score the trace against the rubric dimensions:
1) correctness, 2) safety, 3) idempotency, 4) traceability, 5) spec_compliance.

If safety = 0, abort with SAFETY_FAIL; do not return partial scores.

Verbatim safety rule set to apply:
S1..S17 from references/rubric.md §2 (EIP-specific).
Any S-rule hit ⇒ safety = 0 ⇒ SAFETY_FAIL.

For each dimension, return:
- score (0.0 .. 1.0)
- evidence (cite trace path or command excerpt)
- suggestion (if score < threshold, give a one-sentence fix)

Return ONLY a JSON object matching:
{ "scores": {...}, "safety": 0|1, "verdict": "PASS|RETRY|SAFETY_FAIL|MAX_ITER",
  "suggestions": [...] }
```

## 3. Orchestrator (O) Prompt Template

```text
You are the Orchestrator in a Generator-Critic-Loop (GCL) for huaweicloud-eip-ops.

## Inputs

user_request: {{user.request}}
max_iter: {{user.max_iter}}
rubric: {{output.rubric}}

## Loop control

1) Sanitize user_request → operation_intent (no credentials, no prod-named identifiers).
2) Call Generator (G) with intent + previous critic_feedback.
3) Call Critic (C) on G output + trace.
4) Decide:
   - safety == 0 → SAFETY_FAIL (abort, persist trace)
   - all dims ≥ threshold → PASS (persist trace, return to user)
   - any dim < threshold AND iter < max_iter → RETRY (loop)
   - iter == max_iter → MAX_ITER (persist best-so-far with uncertain flag)
5) Persist trace to audit-results/gcl-trace-YYYYMMDD-HHMMSS.json.

## Constraint

- Safety=0 / SAFETY_FAIL MUST abort immediately. Never return partial or best-effort.
- Every loop MUST be bounded by max_iter. Unbounded retry is banned.
- Production GCL MUST use externally supplied isolated Critic scores; the
  --structural-critic-only mode is for CI/local smoke only.
```

## 4. Sanitization Helper

```python
# Pseudocode — agent runtime uses scripts/gcl_runner.py
def mask(trace: dict) -> dict:
    SENSITIVE = ("HW_SECRET_ACCESS_KEY", "SecretAccessKey", "password", "sk-")
    for k in list(trace.keys()):
        if any(s in k for s in SENSITIVE):
            trace[k] = "***"
    return trace
```

## 5. Failure-Recovery Helper

| Failure | Recovery |
|---|---|
| Sub-agent timeout | Retry once with a smaller scope (e.g., describe only); escalate to user on second timeout |
| Non-JSON CLI response | Capture raw output, mark trace as `non_json=true`, retry without `--output json` once |
| Trace write fail | Re-try to a different timestamp file; if still fails, return MAX_ITER with in-memory summary |

## 6. EIP-Specific Pre-flight Overrides

```text
- Always run `hcloud eip describe-quota` before allocate-eip.
- Always run `hcloud eip describe` and check port_id == null before release-eip.
- For bind-eip: verify target ECS / ENI state via huaweicloud-ecs-ops.
- For resize-bandwidth: query `bandwidth describe` for `cooldown_at`.
- For add-eip-to-shared: verify the EIP's existing `bandwidth.share_type` is `PER`
  (cannot add an EIP that is already in another WHOLE pool).
```

## 7. See Also

- `references/rubric.md` (S1–S17 verbatim)
- `huaweicloud-skill-generator/references/gcl-prompt-backbone.md` (shared prompt text)
- `huaweicloud-skill-generator/references/worker-output-schema.md`
- `docs/gcl-spec.md`
