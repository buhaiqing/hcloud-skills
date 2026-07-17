# GCL Rubric — huaweicloud-iam-ops

> Per-skill rubric instance for the **Generator-Critic-Loop (GCL)** adversarial quality gate.
> Spec: see root `AGENTS.md` §3 (rubric spec) and §8 (per-skill defaults).

> **Version**: v1 (Phase 2, 2026-06-04)
> **max_iter**: 2
> **Scope**: every IAM mutating operation — user/group/policy/agency/access-key/MFA/password/domain.
> Read-only `list*` / `get*` / `describe*` operations are GCL-**exempt**.

## 1. Dimensions

Five mandatory dimensions, scored 0 / 0.5 / 1.

| # | Dimension | What it checks | Threshold |
|---|-----------|----------------|-----------|
| 1 | **Correctness** | User / group / policy / agency / AK id actually exists with requested attributes | ≥ 0.5 (1.0 for `delete-user` / `delete-policy` / `detach-policy` / `delete-access-key` / `create-access-key`) |
| 2 | **Safety** | Destructive op confirmed; privilege-escalation path guarded; secret never leaked | **= 1** (any S-rule hit → ABORT) |
| 3 | **Idempotency** | Re-running does not duplicate side-effects (e.g., 2nd `create-access-key` would still mint new key — guarded) | ≥ 0.5 |
| 4 | **Traceability** | Full request/response captured; AK secret value NEVER logged | ≥ 0.5 |
| 5 | **Spec Compliance** | Policy JSON conforms to IAM action/resource syntax; principal is valid | ≥ 0.5 |

## 2. IAM-Specific Safety Rules (binding — `Safety = 0` on any hit)

| # | Rule | Rationale |
|---|------|-----------|
| S1 | `delete-user` while user has attached policies / group memberships / access keys without prior detach | Cascade failure or orphaned policies |
| S2 | `detach-policy` of `AdministratorAccess` / `SecurityAdministrator` / `TenantAdministrator` / `*:*:*` without two-step confirmation + dry-run | Privilege escalation surface |
| S3 | `attach-policy` of `AdministratorAccess` / `*:*:*` to a user / group / agency without explicit two-step confirmation | Same |
| S4 | `create-access-key` for a user that already has ≥ 2 active keys (Huawei default limit) without prior deactivation | Quota + security: rotate not stack |
| S5 | `delete-access-key` without specifying the `access_key_id` to delete, OR with the request body's `secret_access_key` value present in trace | Secret leak risk |
| S6 | `create-policy` whose policy document contains `"Resource": "*"` combined with `"Effect": "Allow"` and `"Action": "*"` (i.e. admin-by-policy) without flagging | Privilege escalation surface |
| S7 | `create-agency` whose `trust_policy` contains `"Principal": { "IAM": [ "*" ] }` OR service principal is `"*"` | Open delegation |
| S8 | `delete-policy` where `policy.AttachmentCount > 0` (still attached to entities) | Detach-first required |
| S9 | Any operation that prints `SecretAccessKey` value, AK/SK value, or `sk-…` / `password` value | Credential leak |
| S10 | `update-user` / `create-user` setting `password` field in plaintext, or in trace | Same |
| S11 | `delete-domain` / `delete-project` from a non-`account-level` token without `--domain-id` matching `{{env.HW_DOMAIN_ID}}` | Cross-tenant destruction |
| S12 | `mfa-disable` for the account root / for a user with `password_reset` capability | Account takeover surface |
| S13 | `create-access-key` returned response in trace, but `secret_access_key` value not masked to `<masked>` | **S9 subtype, but auto-fail** |
| S14 | `update-login-policy` / `update-password-policy` that disables MFA requirement OR sets minimum password length < 8 | Compliance regression |

The Critic prompt MUST include the full S1–S14 list verbatim (see `prompt-templates.md`).

## 3. Correctness Check Matrix (post-state evidence)

| Operation | Required post-state |
|-----------|---------------------|
| `create-user` | `ShowUser(user_id)` returns matching name + email + phone + enabled status |
| `create-group` | `ListGroups` contains group with same path + name |
| `create-policy` | `ShowPolicy` returns policy with same name + document hash unchanged + `AttachmentCount == 0` |
| `attach-policy` | `ListAttachedPolicies(user_or_group)` contains the policy id |
| `detach-policy` | `ListAttachedPolicies` no longer contains it |
| `create-agency` | `ShowAgency(agency_id)` returns matching name + trust_policy + duration |
| `create-access-key` | `ListAccessKeys(user_id)` count incremented; **only `access_key_id` echoed, secret never returned in trace** |
| `delete-access-key` | `ListAccessKeys` no longer contains the deleted key id |
| `delete-user` | `ShowUser(user_id)` returns 404 or `IAM.0004` |
| `update-password-policy` | `GetPasswordPolicy` reflects new minimum_length / require_symbols etc. |
| `mfa-enable` / `mfa-disable` | `ListMFADevices(user_id)` reflects new device list |

## 4. Idempotency Patterns

The Generator should prefer these patterns. Critic scores 1.0 if present, 0.5 if absent but operation is naturally retry-safe, 0 if the call duplicates side-effects on retry.

| Op | Idempotency mechanism |
|----|----------------------|
| `create-user` | Pre-check `ListUsers(name=…)`; if exists, return existing id |
| `create-group` | Pre-check `ListGroups(path=…, name=…)`; if exists, return existing id |
| `create-policy` | Pre-check `ListPolicies(name=…)`; refuse to overwrite; ask user |
| `attach-policy` | Pre-check `ListAttachedPolicies`; if already attached, no-op |
| `detach-policy` | Pre-check; if not attached, no-op |
| `create-access-key` | **CRITICAL** — pre-check count; if ≥ 2 keys, **ABORT** with S4 (do NOT mint a 3rd) |
| `delete-access-key` | Pre-check key id exists; if not, no-op (return success) |
| `update-password-policy` | Read current policy; if matches target, no-op |
| `create-agency` | Pre-check by name; if exists, return existing |

## 5. Traceability Checklist

The Critic scores Traceability = 1 only if **all** are present:

- [ ] `command` and resolved `args` (post-substitution)
- [ ] `exit_code` + `stdout` (≤ 4 KB) + `stderr`
- [ ] `domain_id` actually used (proves no env-leak)
- [ ] `request_id` for async ops (agency create)
- [ ] **No** `SecretAccessKey` / `secret_access_key` / `sk-…` / `password` value anywhere in trace (regex `(?i)(secret[_-]?access[_-]?key|access[_-]?key[_-]?id|sk-[A-Za-z0-9]{20,}|password[\"']?\s*[:=]\s*[\"'][^\"']{6,})` must return zero hits)
- [ ] For `create-access-key`: response field `secret_access_key` is replaced with `<masked>` (or omitted) before persisting

## 6. Spec Compliance Anchors

`huaweicloud-iam-ops/references/core-concepts.md` rules the Critic enforces:

- Policy `Action` MUST be one of `<service>:<resource>:<action>` pattern; rejects bare `*` unless paired with `Allow` + `Resource: *` AND user has explicitly approved S6
- Policy `Resource` MUST be a URN `acs:<service>::<account>:<resource-path>` or `*`
- Principal in trust policy MUST be one of `{ "IAM": ["<user-or-agency-arn>"] }` or service-principal `<service>.<region>.<domain>`; `*` is S7
- Password policy: `minimum_length ≥ 8`, `require_uppercase / lowercase / digits / symbols` defaults
- User name regex `^[a-zA-Z][a-zA-Z0-9._-]{0,31}$` (per `core-concepts.md` limits)
- Group name regex similar; no path traversal (`..`, `/`)
- AK rotation period ≤ 90 days (recommendation, not enforced)

## 7. Scoring Summary

| Op | Correctness | Safety | Idempotency | Traceability | Spec Compliance | Pass Threshold |
|----|-------------|--------|-------------|--------------|-----------------|----------------|
| `create-user` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass |
| `create-group` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass |
| `create-policy` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S6 guard |
| `attach-policy` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S2/S3 guards |
| `detach-policy` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S2 guard |
| `create-agency` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S7 guard |
| `create-access-key` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S4/S5/S13 guards |
| `delete-access-key` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S5/S9 guards |
| `delete-user` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S1 guard |
| `delete-policy` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S8 guard |
| `mfa-enable / mfa-disable` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S12 guard |
| `update-password-policy` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S14 guard |

## 8. Termination Mapping (per AGENTS.md §5)

| Local result | Decision |
|--------------|----------|
| All dims meet per-op threshold AND Safety = 1 | **PASS** |
| `Safety = 0` | **SAFETY_FAIL** → ABORT |
| Any non-Safety dim < threshold AND `iter < max_iter` | **RETRY** |
| `iter == max_iter` | **MAX_ITER** → best-so-far + unresolved rubric items |

## 8.2 Changelog

| Version | Date | Change |
|---|---|---|
| v1 | 2026-06-04 | Initial rubric. |

## 9. See also

- `AGENTS.md` §3, §5, §7, §8 — repo-wide GCL spec
- `references/prompt-templates.md` — Generator + Critic + Orchestrator prompt skeletons
- `references/core-concepts.md` — Policy syntax, principal patterns, password policy defaults
- `references/troubleshooting.md` — `IAM.0001`–`IAM.0015` error → recovery mapping
