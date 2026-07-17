# GCL Rubric — huaweicloud-obs-ops

> Per-skill rubric instance for the **Generator-Critic-Loop (GCL)** adversarial quality gate.
> Spec: see root `AGENTS.md` §3 (rubric spec) and §8 (per-skill defaults).

> **Version**: v1 (Phase 2, 2026-06-04)
> **max_iter**: 2
> **Scope**: every OBS (Object Storage Service — S3-compatible) mutating operation — bucket
> create / delete / ACL, object upload / download / delete / multi-part copy, lifecycle rules,
> versioning, CORS, bucket policy, static website hosting. Read-only are GCL-**exempt**.

## 1. Dimensions

| # | Dimension | What it checks | Threshold |
|---|-----------|----------------|-----------|
| 1 | **Correctness** | Bucket / object / rule state matches request | ≥ 0.5 (1.0 for `delete-bucket` / `delete-object` / lifecycle purge) |
| 2 | **Safety** | Confirmation; bucket-empty check; public-access guards; CORS safety; secret hygiene | **= 1** (any S-rule hit → ABORT) |
| 3 | **Idempotency** | Re-running does not duplicate side-effects | ≥ 0.5 |
| 4 | **Traceability** | Full request/response; no secret / signature leak | ≥ 0.5 |
| 5 | **Spec Compliance** | Bucket name (DNS-compliant), storage class, region, ACL syntax | ≥ 0.5 |

## 2. OBS-Specific Safety Rules (binding — `Safety = 0` on any hit)

| # | Rule | Rationale |
|---|------|-----------|
| S1 | `delete-bucket` without explicit user confirmation quoting the bucket name | Irreversible data loss |
| S2 | `delete-bucket` while bucket still contains objects (incl. versioned / multipart) | OBS refuses; if forced, data loss |
| S3 | `delete-bucket` with `versioning == Enabled` and versioned objects not yet deleted | Hidden objects remain billable |
| S4 | `delete-object` (single) without confirmation if `size > 1 GB` | Large data loss risk |
| S5 | `delete-objects` (batch) without confirmation if batch size > 100 OR any object > 100 MB | Mass delete risk |
| S6 | `set-bucket-acl` to `public-read` or `public-read-write` without two-step confirmation | Public exposure of all objects |
| S7 | `set-bucket-policy` with `Principal: "*"` and `Action: "s3:*"` Allow on non-website bucket | Anonymous write access |
| S8 | `put-lifecycle-rule` with `Expiration.Days < 1` (immediate deletion) | Accidental mass delete |
| S9 | `put-lifecycle-rule` with `NoncurrentVersionExpiration.NoncurrentDays < 7` on versioned bucket | Aggressive version cleanup |
| S10 | `enable-versioning` on a bucket that previously had versioning suspended (creates orphans) | Data inconsistency |
| S11 | `put-cors` with `AllowedOrigin: "*"` AND `AllowedMethod` includes `PUT` / `POST` / `DELETE` | CSRF / write-from-anywhere surface |
| S12 | `put-bucket-website` redirecting to an `http://` (non-TLS) URL | Mixed-content / downgrade |
| S13 | `delete-bucket` referencing `region` / `project_id` not in env contract (typo) | Cross-tenant |
| S14 | Any trace contains `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / `sk-…` / `password` / `X-OBS-Signature` value | Credential / signature leak |
| S15 | `create-bucket` with name NOT DNS-compliant (uppercase, `_`, `..`, etc.) | Invalid bucket name (S3/OBS will reject) |
| S16 | `put-bucket-tagging` / `put-object-tagging` with `Key > 128 chars` or tag value containing reserved prefixes (`aws:`, `obs:`) | Tagging system corruption |

The Critic prompt MUST include the full S1–S16 list verbatim (see `prompt-templates.md`).

## 3. Correctness Check Matrix (post-state evidence)

| Operation | Required post-state |
|-----------|---------------------|
| `create-bucket` | `HeadBucket` returns 200; same name + region + storage_class |
| `delete-bucket` | `HeadBucket` returns 404 |
| `upload-object` | `HeadObject` returns 200; same size + ETag + content-type |
| `delete-object` | `HeadObject` returns 404 |
| `set-bucket-acl` | `GetBucketAcl` reflects new ACL |
| `put-bucket-policy` | `GetBucketPolicy` returns same policy JSON |
| `put-lifecycle-rule` | `GetLifecycleConfiguration` reflects new rule |
| `enable-versioning` | `GetBucketVersioning` returns `Status: Enabled` |
| `put-cors` | `GetCors` reflects new rules |
| `put-bucket-website` | `GetBucketWebsite` reflects new config |

## 4. Idempotency Patterns

| Op | Idempotency mechanism |
|----|----------------------|
| `create-bucket` | Pre-check `HeadBucket`; if exists with same region, return success; if different region, ABORT (S15 subtype) |
| `delete-bucket` | Pre-check 404; if already gone, return success |
| `upload-object` | Use deterministic `Key`; if exists with same ETag, return success (or warn if overwriting) |
| `delete-object` | Pre-check `HeadObject`; if 404, no-op |
| `set-bucket-acl` | Read current ACL; if matches, no-op |
| `put-bucket-policy` | Read current policy; if matches, no-op |
| `put-lifecycle-rule` | Read current; if rule with same ID + status, no-op |
| `put-cors` | Read current; if matches, no-op |

## 5. Traceability Checklist

The Critic scores Traceability = 1 only if **all** are present:

- [ ] `command` and resolved `args` (post-substitution)
- [ ] `exit_code` + `stdout` (≤ 4 KB) + `stderr`
- [ ] `region` actually used
- [ ] `request_id` / `x-obs-request-id` header extracted
- [ ] **No** `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / `sk-…` / `password` / `X-OBS-Signature` value

## 6. Spec Compliance Anchors

`huaweicloud-obs-ops/references/core-concepts.md` rules the Critic enforces:

- Bucket name: `^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`; lowercase only; no consecutive `.`; not IP-formatted
- Storage class: `STANDARD` / `WARM` / `COLD` (region-dependent)
- Object key: 1–1024 chars; UTF-8; cannot start with `/`
- Max single object: 5 GB (PUT) / 48.8 TB (multipart, max 10000 parts × 5 GB)
- Lifecycle rule ID: `^[a-zA-Z0-9_-]{1,255}$`
- CORS `AllowedOrigin`: `*` or `https://example.com` (no path)
- ACL values: `private` / `public-read` / `public-read-write` / `authenticated-read`

## 7. Scoring Summary

| Op | Correctness | Safety | Idempotency | Traceability | Spec Compliance | Pass Threshold |
|----|-------------|--------|-------------|--------------|-----------------|----------------|
| `create-bucket` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S13/S15 |
| `delete-bucket` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S1/S2/S3 |
| `upload-object` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass |
| `delete-object` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S4 |
| `delete-objects` (batch) | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S5 |
| `set-bucket-acl` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S6 |
| `put-bucket-policy` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S7 |
| `put-lifecycle-rule` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S8/S9 |
| `enable-versioning` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S10 |
| `put-cors` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S11 |
| `put-bucket-website` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S12 |

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
- `references/core-concepts.md` — Bucket name / storage class / ACL anchors
- `references/troubleshooting.md` — OBS error code mapping
