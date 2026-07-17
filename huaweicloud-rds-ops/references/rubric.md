# GCL Rubric — huaweicloud-rds-ops

> Per-skill rubric instance for the **Generator-Critic-Loop (GCL)** adversarial quality gate.
> Spec: see root `AGENTS.md` §3 (rubric spec) and §8 (per-skill defaults).

> **Version**: v1 (Phase 2, 2026-06-04)
> **max_iter**: 2
> **Scope**: every RDS mutating operation — instance create/delete/resize/restore, database/user
> create, parameter change, backup create/delete. Read-only `describe*` / `list*` are
> GCL-**exempt**.

## 1. Dimensions

Five mandatory dimensions, scored 0 / 0.5 / 1.

| # | Dimension | What it checks | Threshold |
|---|-----------|----------------|-----------|
| 1 | **Correctness** | Instance / database / user / parameter actually matches the request | ≥ 0.5 (1.0 for `delete-instance` / DDL / restore) |
| 2 | **Safety** | Destructive op confirmed; DDL guarded; prePaid balance checked; secret never leaked | **= 1** (any S-rule hit → ABORT) |
| 3 | **Idempotency** | Re-running does not duplicate side-effects (e.g., 2nd `create-database` would fail) | ≥ 0.5 |
| 4 | **Traceability** | Full request/response captured; password never echoed in trace | ≥ 0.5 |
| 5 | **Spec Compliance** | Flavor / engine version / storage size / parameter value all within quota & range | ≥ 0.5 |

## 2. RDS-Specific Safety Rules (binding — `Safety = 0` on any hit)

| # | Rule | Rationale |
|---|------|-----------|
| S1 | `delete-instance` without explicit user confirmation quoting the instance ID | Irreversible data loss (unless backup exists) |
| S2 | `delete-instance` while the most recent automated backup is **missing or failed**, without a fresh manual backup | Unrecoverable |
| S3 | `delete-instance` for **prePaid** instance with > 7 days remaining in subscription, without refund-warning | Wastes paid period |
| S4 | `restore-from-backup` to a **different** instance without two-step confirmation (overwrites target if it exists) | Cross-instance blast radius |
| S5 | `restore-from-backup` to the **same** instance while it is `ACTIVE` (RDS requires stop) | Operation will fail |
| S6 | `resize-instance` DOWN (smaller flavor / less storage) without maintenance window confirmation | Downtime |
| S7 | `create-database` with name that contains SQL injection pattern (`;--`, `/*`, `' OR 1=1`) | Injection via name |
| S8 | `create-user` / `reset-password` echoing the new password in command args, OR trace contains password value | Credential leak |
| S9 | `update-parameter` setting `innodb_flush_log_at_trx_commit=2` or `sync_binlog=0` on a production-tagged instance without confirmation | Durability regression |
| S10 | `update-parameter` with `max_connections` > 100000 without confirmation | Resource exhaustion |
| S11 | `create-account` granting `ALL PRIVILEGES` + `WITH GRANT OPTION` + `*.*` to a non-admin user | Privilege escalation surface |
| S12 | `delete-database` for a system database (`mysql`, `information_schema`, `performance_schema`, `sys`, `postgres`, `template0/1`) | Operational breakage |
| S13 | `delete-manual-backup` where `backup.status != COMPLETED` (mid-backup) or it's the **only** valid backup | Unrecoverable |
| S14 | Any operation that prints `password` / `PASSWORD` / `sk-…` plaintext in command args, response, or log | Credential leak |
| S15 | `create-instance` referencing `region` / `project_id` not in env contract (typo or default substitution) | Cross-tenant deployment |

The Critic prompt MUST include the full S1–S15 list verbatim (see `prompt-templates.md`).

## 3. Correctness Check Matrix (post-state evidence)

| Operation | Required post-state |
|-----------|---------------------|
| `create-instance` | `ShowInstanceDetail` returns `status: ACTIVE` (or `BUILD` accepted) with same name, flavor, engine, version, storage, vpc_id, subnet_id, security_group_id |
| `delete-instance` | `ShowInstanceDetail` returns 404 or `DBS.200404` within poll budget |
| `resize-instance` | `flavor` / `volume.size` matches target; pre-state was `ACTIVE` (or `SHUTOFF` for downsize) |
| `restore-from-backup` | Target instance `status: ACTIVE`; data checksum matches backup (or S4 two-step completed) |
| `create-database` | `ListDatabases(instance_id)` contains db with same name + charset + collation |
| `create-user` | `ListUsers(instance_id, name=…)` returns the user; **password never returned in response** |
| `reset-password` | `ListUsers` returns same name; **old/new password never in trace** |
| `update-parameter` | `ListConfigurations` or `ShowInstanceConfiguration` reflects new value; `apply_type: dynamic|static` honored |
| `create-manual-backup` | `ShowBackup` returns `status: COMPLETED` with size > 0 |

## 4. Idempotency Patterns

| Op | Idempotency mechanism |
|----|----------------------|
| `create-instance` | Pre-check `ListInstances(name=…)`; if exists, return existing id (refuse to recreate) |
| `delete-instance` | Pre-check `DBS.200404`; if already gone, return success |
| `resize-instance` | Read current `flavor`; if already target size, no-op |
| `restore-from-backup` | Use deterministic `restore_target_instance_name` + tag to dedup |
| `create-database` | Pre-check `ListDatabases(name=…)`; if exists, return success (or warn) |
| `create-user` | Pre-check `ListUsers(name=…)`; if exists, refuse to recreate — ask user to reset password instead |
| `update-parameter` | Read current value; if matches, no-op |
| `create-manual-backup` | Use deterministic `backup_name`; if exists with `COMPLETED`, return existing |

## 5. Traceability Checklist

The Critic scores Traceability = 1 only if **all** are present:

- [ ] `command` and resolved `args` (post-substitution)
- [ ] `exit_code` + `stdout` (≤ 4 KB) + `stderr`
- [ ] `region` / `project_id` actually used
- [ ] `job_id` extracted for async ops (resize, restore, delete)
- [ ] **No** `password` / `PASSWORD` / `sk-…` / `SecretAccessKey` value anywhere in trace
- [ ] For `reset-password` / `create-user`: password passed via env / stdin / KMS reference, NOT as CLI arg

## 6. Spec Compliance Anchors

`huaweicloud-rds-ops/references/core-concepts.md` rules the Critic enforces:

- Engine version: MySQL `{5.7, 8.0}`, PostgreSQL `{12, 13, 14, 15}`, SQL Server `{2019, 2022}`
- Flavor pattern `^db\.rds\.(s|m|c)\.(small|medium|large|xlarge|2xlarge)\.[0-9]+$` (refine per region)
- Storage size 5 GB – 4000 GB; step 10 GB (HA) / 5 GB (single)
- Backup retention 1–35 days (autobackup)
- Parameter value range per `core-concepts.md` parameter table (e.g., `max_connections` ≤ 100000, `innodb_buffer_pool_size` ≤ 80% of total memory)
- Database name regex `^[a-zA-Z][a-zA-Z0-9_$]{0,63}$`
- Username regex `^[a-zA-Z][a-zA-Z0-9_]{0,31}$`; reserved names list: `root`, `admin`, `mysql`, `rdsadmin`, `repl`
- Region list matches `core-concepts.md` §1.2

## 7. Scoring Summary

| Op | Correctness | Safety | Idempotency | Traceability | Spec Compliance | Pass Threshold |
|----|-------------|--------|-------------|--------------|-----------------|----------------|
| `create-instance` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S15 |
| `delete-instance` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S1/S2/S3 |
| `resize-instance` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S6 |
| `restore-from-backup` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S4/S5 |
| `create-database` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S7/S12 |
| `delete-database` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S12 |
| `create-user` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S8/S11 |
| `reset-password` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S8 |
| `update-parameter` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S9/S10 |
| `create-manual-backup` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass |
| `delete-manual-backup` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S13 |

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
- `references/core-concepts.md` — Engine/region/parameter anchors
- `references/troubleshooting.md` — `DBS.20xxxx` error → recovery mapping
