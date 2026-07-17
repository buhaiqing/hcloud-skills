# GCL Rubric — huaweicloud-lts-ops

> Per-skill rubric instance for the **Generator-Critic-Loop (GCL)** adversarial quality gate.
> Spec: see root `AGENTS.md` §3 (rubric spec) and §8 (per-skill defaults).

> **Version**: v1 (Phase 3, 2026-06-04)
> **max_iter**: 3
> **Scope**: every LTS (Log Tank Service) mutating operation — log group create / delete, log stream create, log transfer create / delete, retention (TTL) update. Read-only list / search are GCL-**exempt**.

## 1. Dimensions

| # | Dimension | What it checks | Threshold |
|---|-----------|----------------|-----------|
| 1 | **Correctness** | Log group / stream / transfer / retention state matches request | ≥ 0.5 |
| 2 | **Safety** | Confirmation; backup before delete; transfer target accessible; credential hygiene | **= 1** (any S-rule hit → ABORT) |
| 3 | **Idempotency** | Re-running does not duplicate log groups / streams / transfers | ≥ 0.5 |
| 4 | **Traceability** | Full request/response; credential never in trace; no log content leaked | ≥ 0.5 |
| 5 | **Spec Compliance** | Retention period (1–365 days), OBS bucket format, quota limits | ≥ 0.5 |

## 2. LTS-Specific Safety Rules (binding — `Safety = 0` on any hit)

| # | Rule | Rationale |
|---|------|-----------|
| S1 | `delete-log-group` without explicit user confirmation quoting the group ID | **CRITICAL** — permanent log loss |
| S2 | `delete-log-group` that still contains active log streams | Each stream holds log data; deletion cascades |
| S3 | `delete-log-group` without offering to transfer logs to OBS first | Irreversible data loss without backup path |
| S4 | `create-log-transfer` targeting a non-existent or inaccessible OBS bucket | Transfer silently fails |
| S5 | `delete-log-transfer` while log retention is set to "never expire" | Logs become inaccessible with no export path |
| S6 | `update-retention` (TTL) shorter than existing log age without warning about data loss | Logs older than new TTL are permanently deleted |
| S7 | `create-log-group` without checking account quota (max groups) | API returns QuotaExceeded; silent UX failure |
| S8 | Any trace contains `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / `sk-…` / password plaintext | Credential leak |
| S9 | `create-log-stream` under a group that has already reached max stream quota | Silent failure; no error surfaced to user |

The Critic prompt MUST include the full S1–S9 list verbatim (see `prompt-templates.md`).

## 3. Correctness Check Matrix (post-state evidence)

| Operation | Required post-state |
|-----------|---------------------|
| `create-log-group` | `ShowLogGroup` returns same `log_group_name` + `ttl_in_days` |
| `delete-log-group` | `ShowLogGroup` returns 404 |
| `create-log-stream` | `ShowLogStream` returns same `log_stream_name`; `ListLogStreams` count increments |
| `create-transfer` | `ShowTransfer` returns `status: ENABLED` with matching `obs_bucket_name` |
| `delete-transfer` | `ShowTransfer` returns 404 |
| `update-retention` | `ShowLogGroup` `ttl_in_days` matches new value |
| `search-logs` | `ListLogs` returns non-empty result within time range |

## 4. Idempotency Patterns

| Op | Idempotency mechanism |
|----|----------------------|
| `create-log-group` | Pre-check `ListLogGroups(name=…)`; if exact group exists, return existing id |
| `delete-log-group` | Pre-check 404; if already gone, return success |
| `create-log-stream` | Pre-check `ListLogStreams(group_id=…)` for same name; skip if exists |
| `create-transfer` | Pre-check `ListTransfers(group_id=…)` for same OBS bucket+prefix; skip if exists |
| `delete-transfer` | Pre-check 404 |
| `update-retention` | Read current `ttl_in_days`; if already target, no-op |

## 5. Traceability Checklist

The Critic scores Traceability = 1 only if **all** are present:

- [ ] `command` and resolved `args` (post-substitution)
- [ ] `exit_code` + `stdout` (≤ 4 KB) + `stderr`
- [ ] `region` / `project_id` actually used
- [ ] For `create-transfer`: the target OBS bucket name is captured
- [ ] For `update-retention`: old and new TTL values are captured
- [ ] **No** `password` / `PASSWORD` / `sk-…` / `SecretAccessKey` value in trace
- [ ] No log content in trace (privacy)

## 6. Spec Compliance Anchors

`huaweicloud-lts-ops/references/core-concepts.md` rules the Critic enforces:

- Retention in days: 1–365 (some older APIs use hours)
- Log group count quota: 100 per account (default; can be raised via ticket)
- Log stream count quota: 200 per log group
- Transfer targets: OBS bucket (must exist in same region), DMS queue
- Log search time range: max 30 consecutive days for single query
- Service endpoints: `lts.cn-north-4.myhuaweicloud.com` (varies by region)

## 7. Scoring Summary

| Op | Correctness | Safety | Idempotency | Traceability | Spec Compliance | Pass Threshold |
|----|-------------|--------|-------------|--------------|-----------------|----------------|
| `create-log-group` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S7 |
| `delete-log-group` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S1/S2/S3 |
| `create-log-stream` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S9 |
| `create-transfer` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S4 |
| `delete-transfer` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S5 |
| `update-retention` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S6 |

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
- `references/core-concepts.md` — Log group / stream quotas, retention limits
- `references/troubleshooting.md` — LTS error code mapping