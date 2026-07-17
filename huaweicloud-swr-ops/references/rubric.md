# GCL Rubric — huaweicloud-swr-ops

> Per-skill rubric instance for the **Generator-Critic-Loop (GCL)** adversarial quality gate.
> Spec: see root `AGENTS.md` §3 (rubric spec) and §8 (per-skill defaults).

> **Version**: v1 (Phase 2, 2026-06-04)
> **max_iter**: 2
> **Scope**: every SWR (Software Repository for Container — container image registry) mutating
> operation — organization create / delete, repository create / delete, image / tag delete,
> retention policy create / update. Read-only are GCL-**exempt**.

## 1. Dimensions

| # | Dimension | What it checks | Threshold |
|---|-----------|----------------|-----------|
| 1 | **Correctness** | Org / repo / image / retention state matches request | ≥ 0.5 (1.0 for `delete-org` / `delete-repo` / `delete-image` / `delete-image-tag`) |
| 2 | **Safety** | Confirmation; shared-org guard; image-in-use detection; secret hygiene | **= 1** (any S-rule hit → ABORT) |
| 3 | **Idempotency** | Re-running does not duplicate side-effects | ≥ 0.5 |
| 4 | **Traceability** | Full request/response; no credential leak | ≥ 0.5 |
| 5 | **Spec Compliance** | Org name regex, repo name regex, retention days range | ≥ 0.5 |

## 2. SWR-Specific Safety Rules (binding — `Safety = 0` on any hit)

| # | Rule | Rationale |
|---|------|-----------|
| S1 | `delete-organization` without explicit user confirmation quoting the org name | All repos & images lost |
| S2 | `delete-organization` while the org still contains non-deleted repositories | Cascade refusal |
| S3 | `delete-organization` if it's the user's **default / last** org | Account-level loss |
| S4 | `delete-repository` without explicit user confirmation quoting the repo name + namespace | All image tags lost |
| S5 | `delete-repository` while a CCE / CCI workload is currently using any image tag from this repo | Pods may fail to pull on restart |
| S6 | `delete-image` (all tags) without explicit two-step confirmation | Mass tag delete |
| S7 | `delete-image-tag` for a tag currently in use by a running CCE/CCI workload (check `kubectl get pods` image refs OR CCE deployment spec) | Same as S5, per-tag |
| S8 | `update-retention-policy` with `retention_days < 1` (immediate cleanup) on a prod repo | Aggressive cleanup |
| S9 | `update-retention-policy` with `tag_count < 5` (keeps only 4 most recent tags) on a prod repo | Insufficient rollback headroom |
| S10 | `create-organization` with name conflicting with a built-in `library` (SWR reserved) | Name conflict |
| S11 | `create-repository` referencing `region` / `project_id` not in env contract (typo) | Cross-tenant |
| S12 | Any trace contains `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / `sk-…` / `password` / registry docker login password plaintext | Credential leak |
| S13 | `create-repository` with name `library/*` (only allowed for the system `library` org) | Name conflict |
| S14 | `delete-image-tag` for a tag with `pull_count > 0` in last 30 days (recently used) | Hot image removal risk |
| S15 | `share-repository` (cross-tenant share) without explicit user confirmation of target account_id | Cross-tenant data leak |

The Critic prompt MUST include the full S1–S15 list verbatim (see `prompt-templates.md`).

## 3. Correctness Check Matrix (post-state evidence)

| Operation | Required post-state |
|-----------|---------------------|
| `create-organization` | `ListOrganizations` contains org with same name |
| `delete-organization` | `ListOrganizations` no longer contains it |
| `create-repository` | `ListRepositories(namespace, name=…)` returns same name + category |
| `delete-repository` | `ListRepositories` no longer contains it |
| `delete-image` | `ListImages(repository=…)` no longer contains the digest |
| `delete-image-tag` | `ListImageTags` no longer contains the tag |
| `create-retention-policy` | `ShowRetentionPolicy` returns same name + retention_days + tag_count |
| `update-retention-policy` | `ShowRetentionPolicy` reflects new values |
| `share-repository` | `ShowRepository.shared_to` contains target account_id |

## 4. Idempotency Patterns

| Op | Idempotency mechanism |
|----|----------------------|
| `create-organization` | Pre-check `ListOrganizations(name=…)`; if exists, return existing id |
| `delete-organization` | Pre-check 404; if already gone, return success |
| `create-repository` | Pre-check `ListRepositories(name=…)`; if exists, return existing id |
| `delete-repository` | Pre-check 404; if already gone, return success |
| `delete-image` | Pre-check `ListImages`; if digest absent, no-op |
| `delete-image-tag` | Pre-check `ListImageTags`; if tag absent, no-op |
| `create-retention-policy` | Pre-check; if exists with same values, no-op |
| `update-retention-policy` | Read current; if matches, no-op |

## 5. Traceability Checklist

The Critic scores Traceability = 1 only if **all** are present:

- [ ] `command` and resolved `args` (post-substitution)
- [ ] `exit_code` + `stdout` (≤ 4 KB) + `stderr`
- [ ] `region` / `project_id` actually used
- [ ] **No** `password` / `PASSWORD` / `sk-…` / `SecretAccessKey` / docker registry password in trace

## 6. Spec Compliance Anchors

`huaweicloud-swr-ops/references/core-concepts.md` rules the Critic enforces:

- Organization name regex: `^[a-z][a-z0-9-]{1,63}$`; reserved: `library`
- Repository name regex: `^[a-z0-9][a-z0-9._/-]{1,127}$`; must not start with `/` or contain `..`
- Image tag regex: `^[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}$`
- Retention `retention_days`: 1–365
- Retention `tag_count`: 1–100 (default 30)
- Region list per `core-concepts.md` §1.2

## 7. Scoring Summary

| Op | Correctness | Safety | Idempotency | Traceability | Spec Compliance | Pass Threshold |
|----|-------------|--------|-------------|--------------|-----------------|----------------|
| `create-organization` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S10 |
| `delete-organization` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S1/S2/S3 |
| `create-repository` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S11/S13 |
| `delete-repository` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S4/S5 |
| `delete-image` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S6 |
| `delete-image-tag` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S7/S14 |
| `create-retention-policy` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass |
| `update-retention-policy` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S8/S9 |
| `share-repository` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S15 |

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
- `references/prompt-templates.md` — Generator + Critic + Orchestrator skeletons
- `references/core-concepts.md` — Org / repo / tag name regex; retention range anchors
- `references/idempotency-checklist.md` — pre-existing idempotency patterns
- `references/troubleshooting.md` — SWR error code mapping
