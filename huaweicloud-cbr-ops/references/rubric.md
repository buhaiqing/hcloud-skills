# GCL Rubric — huaweicloud-cbr-ops

> Per-skill rubric instance for the **Generator-Critic-Loop (GCL)** adversarial quality gate.
> Spec: see root `AGENTS.md` §3 (rubric spec) and §8 (per-skill defaults).

> **Version**: v1 (Phase 2, 2026-06-04)
> **max_iter**: 2
> **Scope**: every CBR (Cloud Backup and Recovery) mutating operation — vault create / delete,
> policy create / update / delete, backup create / copy / delete, **restore** (the most
> dangerous — overwrites source). Read-only are GCL-**exempt**.

## 1. Dimensions

| # | Dimension | What it checks | Threshold |
|---|-----------|----------------|-----------|
| 1 | **Correctness** | Vault / policy / backup / restore state matches request | ≥ 0.5 (1.0 for `restore` / `delete-vault` / `delete-backup`) |
| 2 | **Safety** | Confirmation; target disk verified; prePaid refund; **restore target** validated | **= 1** (any S-rule hit → ABORT) |
| 3 | **Idempotency** | Re-running does not duplicate side-effects | ≥ 0.5 |
| 4 | **Traceability** | Full request/response; no secret leak | ≥ 0.5 |
| 5 | **Spec Compliance** | Resource type, vault size, retention days, replication region | ≥ 0.5 |

## 2. CBR-Specific Safety Rules (binding — `Safety = 0` on any hit)

| # | Rule | Rationale |
|---|------|-----------|
| S1 | `restore` without explicit user confirmation quoting the **target** disk/server ID | **CRITICAL** — restore overwrites target data |
| S2 | `restore` to a target disk that is **not detached** (server still has it attached) | Restore will fail or cause filesystem inconsistency |
| S3 | `restore` to a target disk whose size is **smaller** than the source backup size | Restore fails / truncates data |
| S4 | `restore` to a different server/disk without two-step confirmation | Cross-instance blast |
| S5 | `delete-vault` while the vault still contains backups, no migration plan | Unrecoverable backups |
| S6 | `delete-vault` for prePaid vault with > 7 days remaining, no refund-warning | Wastes paid period |
| S7 | `delete-backup` while the backup is the only valid one for its source resource | Unrecoverable |
| S8 | `delete-backup` while `backup.status != available` (mid-backup / failed) | Cannot delete non-terminal backup |
| S9 | `copy-backup` (cross-region replication) to a region without first verifying target vault exists | Replication fails |
| S10 | `update-policy` setting `retention_duration_days < 7` (compliance violation) | Compliance regression |
| S11 | `create-vault` referencing `region` / `project_id` not in env contract (typo) | Cross-tenant |
| S12 | `create-policy` with `trigger_time` set to a past timestamp (inadvertent immediate execution) | Unexpected backup run |
| S13 | Any trace contains `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / `password` / `sk-…` plaintext | Credential leak |
| S14 | `restore` to a target disk where the server has a different `os_type` than the backup (Linux ↔ Windows) | OS won't boot |
| S15 | `create-backup` while another backup for the same resource is currently `RUNNING` | Concurrent backup conflict |

The Critic prompt MUST include the full S1–S15 list verbatim (see `prompt-templates.md`).

## 3. Correctness Check Matrix (post-state evidence)

| Operation | Required post-state |
|-----------|---------------------|
| `create-vault` | `ShowVault` returns `status: available`; matches name + resource_type + size |
| `delete-vault` | `ShowVault` returns 404 |
| `create-policy` | `ShowPolicy` returns same name + trigger_time + retention |
| `update-policy` | `ShowPolicy` reflects new value |
| `delete-policy` | `ShowPolicy` returns 404 |
| `create-backup` | `ShowBackup` returns `status: available` with size > 0 |
| `copy-backup` | `ListBackups(destination_region)` contains the replicated backup with `status: available` |
| `delete-backup` | `ShowBackup` returns 404 |
| `restore` | Target disk attached to server (if specified); `restore.status: success`; `validate` step shows server boots and data accessible |

## 4. Idempotency Patterns

| Op | Idempotency mechanism |
|----|----------------------|
| `create-vault` | Pre-check `ListVaults(name=…)`; if exists, return existing id |
| `delete-vault` | Pre-check 404; if already gone, return success |
| `create-policy` | Pre-check `ListPolicies(name=…)`; if exists, refuse (require update) |
| `delete-policy` | Pre-check 404 |
| `create-backup` | Use deterministic `backup_name`; if exists with `available`, return existing |
| `delete-backup` | Pre-check 404 |
| `copy-backup` | Pre-check `ListBackups(destination_region, name=…)`; if exists, return existing |
| `restore` | Use deterministic `restore_job_id`; if already succeeded, no-op |

## 5. Traceability Checklist

The Critic scores Traceability = 1 only if **all** are present:

- [ ] `command` and resolved `args` (post-substitution)
- [ ] `exit_code` + `stdout` (≤ 4 KB) + `stderr`
- [ ] `region` / `project_id` actually used
- [ ] `job_id` for async ops (restore / copy / create-backup)
- [ ] **No** `password` / `PASSWORD` / `sk-…` / `SecretAccessKey` value in trace

## 6. Spec Compliance Anchors

`huaweicloud-cbr-ops/references/core-concepts.md` rules the Critic enforces:

- Resource types: `OS::Nova::Server` (ECS), `OS::Cinder::Volume` (EVS), `OS::Workspace::DesktopV2`
- Vault size: 10 GB – 10485760 GB (10 TB)
- Backup retention: 1–365 days
- Cross-region replication: only available for vault with `availability_zone` not specified
- Vault name regex: `^[a-zA-Z][a-zA-Z0-9._-]{1,63}$`
- Policy name regex: `^[a-zA-Z][a-zA-Z0-9._-]{1,63}$`

## 7. Scoring Summary

| Op | Correctness | Safety | Idempotency | Traceability | Spec Compliance | Pass Threshold |
|----|-------------|--------|-------------|--------------|-----------------|----------------|
| `create-vault` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S11 |
| `delete-vault` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S5/S6 |
| `create-policy` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S12 |
| `update-policy` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S10 |
| `delete-policy` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass |
| `create-backup` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S15 |
| `copy-backup` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S9 |
| `delete-backup` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S7/S8 |
| `restore` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S1/S2/S3/S4/S14 |

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
- `references/core-concepts.md` — Resource type / vault size / retention anchors
- `references/idempotency-checklist.md` — pre-existing idempotency patterns
- `references/troubleshooting.md` — CBR error code mapping
