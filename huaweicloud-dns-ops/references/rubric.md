# GCL Rubric — huaweicloud-dns-ops

> Per-skill rubric instance for the **Generator-Critic-Loop (GCL)** adversarial quality gate.
> Spec: see root `AGENTS.md` and `docs/gcl-spec.md`.

> **Version**: v1 (initial, 2026-06-24)
> **max_iter**: 2
> **Scope**: every DNS mutating operation (`create-zone`, `delete-zone`,
> `create-recordset`, `update-recordset`, `delete-recordset`, `set-dnssec-status`).
> Read-only `list-zones`, `list-recordsets`, `show-zone` are GCL-**exempt**.

## 1. Dimensions

| # | Dimension | What it checks | Threshold |
|---|---|---|---:|
| 1 | **Correctness** | Zone / record matches request | ≥ 0.5 (1.0 for `delete-zone`) |
| 2 | **Safety** | Cascade impact, credential hygiene | **= 1** (any S-rule hit → ABORT) |
| 3 | **Idempotency** | Re-running does not duplicate | ≥ 0.5 |
| 4 | **Traceability** | Full request/response captured | ≥ 0.5 |
| 5 | **Spec Compliance** | CLI flags / JSON paths from OpenAPI | ≥ 0.5 |

## 2. DNS-Specific Safety Rules (S-rules — `Safety = 0` on any hit)

| # | Rule | Rationale |
|---|---|---|
| S1 | `delete-zone` without confirming all records removed | Accidental zone delete loses all DNS records |
| S2 | `delete-zone` matching production domain without two-step confirmation | Production blast radius |
| S3 | `delete-recordset` without confirming TTL propagation delay | Deleted records may resolve for up to TTL duration |
| S4 | `create-zone` without verifying domain ownership | Domain not delegated → zone unusable |
| S5 | MX record pointing to CNAME (RFC 1034 violation) | Agent must catch MX→CNAME and reject |
| S6 | `update-recordset` removing all records without confirmation | Blank record = DNS resolution failure |
| S7 | Any op printing `HW_SECRET_ACCESS_KEY` / `sk-…` | Credential leak |
| S8 | CNAME + A record conflict: creating CNAME when A already exists | CNAME replaces A at same name |

## 3. Correctness Check Matrix (post-state evidence)

| Operation | Required post-state |
|---|---|
| `create-zone` | `show-zone` returns `status = ACTIVE` |
| `delete-zone` | `list-zones` returns 404 / zone absent |
| `create-recordset` | `list-recordsets` includes new recordset |
| `update-recordset` | `list-recordsets` shows updated value / TTL |
| `delete-recordset` | `list-recordsets` excludes deleted recordset |
| `set-dnssec-status` | `show-zone` shows updated `dnssec_status` |

## 4. Idempotency Check

| Operation | Idempotent? | Pre-retry step |
|---|---|---|
| `CreateZone` | No | `list-zones`; skip if exists |
| `CreateRecordSet` | No | `list-recordsets`; update if exists |
| `DeleteZone` | Yes (404 = success) | n/a |
| `DeleteRecordSet` | Yes | n/a |
| `UpdateRecordSet` | Yes | n/a |
| `SetDnssecStatus` | Yes | n/a |

## 5. Traceability Requirements

1. Command, args, response excerpt, `request_id` captured.
2. `HW_SECRET_ACCESS_KEY` / AK masked.
3. `operation_intent` sanitized (no raw user request).
4. Persist to `audit-results/gcl-trace-YYYYMMDD-HHMMSS.json`.

## 6. Scoring Guide

`composite` is the **weighted geometric mean** of dims 1, 3, 4, 5 (weight: 0.4 / 0.2 / 0.2 / 0.2). **dim 2 (Safety) is binary and does not participate in composite** — safety=0 always produces SAFETY_FAIL.

| Condition | Result |
|---|---|
| safety = 0 (any S-rule hit) | `SAFETY_FAIL` — abort immediately |
| safety = 1, all dims ≥ threshold | `PASS` |
| safety = 1, any dim < threshold, iter < max_iter | `RETRY` |
| safety = 1, iter == max_iter | `MAX_ITER` (best-so-far with `uncertain: true`) |

## 7. Examples

### Example 1 — Create Zone PASS

- Trace: domain ownership verified → `create-zone` → poll → `status = ACTIVE`
- Scores: correctness=1.0, safety=1.0, idempotency=1.0, traceability=1.0, spec_compliance=1.0
- Verdict: `PASS`

### Example 2 — MX→CNAME Violation (S5) → SAFETY_FAIL

- Trace: agent created MX record pointing to `mail.example.com` (CNAME)
- Verdict: `SAFETY_FAIL`. RFC 1034 forbids MX→CNAME; agent must reject this.

## 8. Escalation & Changelog

### 8.1 Escalation Path

1. `SAFETY_FAIL` → abort; do not retry.
2. `MAX_ITER` → best-so-far with `uncertain` flag; user decides.
3. IAM `Unauthorized` → HALT; delegate to `huaweicloud-iam-ops`.
4. Zone locked (DNSSEC transition) → HALT; wait for completion.

### 8.2 Changelog

| Version | Date | Change |
|---|---|---|
| v1 | 2026-06-24 | Initial rubric: 8 DNS-specific safety rules, 5 dimensions, 2 examples. |
