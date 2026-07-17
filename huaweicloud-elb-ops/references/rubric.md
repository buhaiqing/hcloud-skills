# GCL Rubric — huaweicloud-elb-ops

> Per-skill rubric instance for the **Generator-Critic-Loop (GCL)** adversarial quality gate.
> Spec: see root `AGENTS.md` §3 (rubric spec) and §8 (per-skill defaults).

> **Version**: v1 (Phase 3, 2026-06-04)
> **max_iter**: 3
> **Scope**: every ELB (Elastic Load Balancer) mutating operation — load balancer create / delete,
> listener create / update / delete, pool create / delete, member add / remove, certificate bind /
> replace / delete. **CRITICAL**: includes `delete-lb` / `delete-listener` / `delete-pool` which
> are the highest-frequency traffic-disruption paths.
> Read-only `describe*` / `list*` are GCL-**exempt**.

## 1. Dimensions

| # | Dimension | What it checks | Threshold |
|---|-----------|----------------|-----------|
| 1 | **Correctness** | LB / listener / pool / member state matches request | ≥ 0.5 (1.0 for `delete-lb` / `delete-listener` / `delete-pool`) |
| 2 | **Safety** | Confirmation; orphan prevention; quorum preservation; credential hygiene | **= 1** (any S-rule hit → ABORT) |
| 3 | **Idempotency** | Re-running does not duplicate side-effects (especially create operations) | ≥ 0.5 |
| 4 | **Traceability** | Full request/response; credential never in trace | ≥ 0.5 |
| 5 | **Spec Compliance** | LB type (shared/dedicated), listener protocol, port range, health check params | ≥ 0.5 |

## 2. ELB-Specific Safety Rules (binding — `Safety = 0` on any hit)

| # | Rule | Rationale |
|---|------|-----------|
| S1 | `delete-lb` without explicit user confirmation quoting the LB ID | **CRITICAL** — traffic disruption for all bound listeners |
| S2 | `delete-lb` while the LB still has active listeners (orphaned) | Listeners become orphaned; user may not realize scope |
| S3 | `delete-lb` while the LB still has active backend pools | Active connections dropped mid-flight |
| S4 | `delete-lb` with EIP bound, no warning about EIP orphan | EIP continues billing; user loses LB-to-EIP linkage |
| S5 | `create-listener` referencing a non-existent or protocol-mismatched pool | Connection break / silent failure |
| S6 | `update-listener` switching to a pool incompatible with listener protocol | Protocol mismatch → all backends unhealthy |
| S7 | `delete-pool` while member backend servers have active connections | Connection drops without graceful drain |
| S8 | `add-member` with invalid subnet or unreachable IP address | Useless member; wastes health check resources |
| S9 | `delete-member` without checking if it is the last healthy member | Quorum loss → all backends marked unhealthy |
| S10 | `update-certificate` (replace SSL/TLS) on a listener without maintenance window | SNI / TLS handshake breakage for active users |
| S11 | `create-listener` with `protocol_port` already in use on the same LB | Port conflict → create failure |
| S12 | Any trace contains `HW_SECRET_ACCESS_KEY` / `SecretAccessKey` / `sk-…` / password plaintext | Credential leak |
| S13 | `delete-certificate` while it is actively bound to an HTTPS listener | HTTPS listeners lose TLS binding |

The Critic prompt MUST include the full S1–S13 list verbatim (see `prompt-templates.md`).

## 3. Correctness Check Matrix (post-state evidence)

| Operation | Required post-state |
|-----------|---------------------|
| `create-lb` | `ShowLoadBalancer` returns `provisioning_status: ACTIVE` with same name + type + vpc_id |
| `delete-lb` | `ShowLoadBalancer` returns 404 within poll budget |
| `create-listener` | `ShowListener` returns `provisioning_status: ACTIVE` with same protocol + port + pool |
| `delete-listener` | `ShowListener` returns 404 |
| `create-pool` | `ShowPool` returns `provisioning_status: ACTIVE` with same protocol + lb_algorithm |
| `delete-pool` | `ShowPool` returns 404 |
| `add-member` | `ShowMember` returns `operating_status: ONLINE` (or `NO_MONITOR` accepted) |
| `remove-member` | `ShowMember` returns 404 |
| `update-certificate` | `ShowCertificate` returns new `expiration` / `subject` matching the certificate |
| `delete-certificate` | `ShowCertificate` returns 404; verify no listener references it |

## 4. Idempotency Patterns

| Op | Idempotency mechanism |
|----|----------------------|
| `create-lb` | Pre-check `ListLoadBalancers(name=…)`; if exists with same params, return existing id |
| `delete-lb` | Pre-check 404; if already gone, return success |
| `create-listener` | Pre-check `ListListeners(lb_id=…)` for same protocol+port; if exists, return existing |
| `delete-listener` | Pre-check 404 |
| `create-pool` | Pre-check `ListPools(lb_id=…)` for same protocol+algorithm; if exists, return existing |
| `delete-pool` | Pre-check 404 |
| `add-member` | Pre-check `ListMembers(pool_id=…)` for same address+port; if exists, skip |
| `remove-member` | Pre-check 404 |
| `update-certificate` | Read current `certificate_id`; if already same, no-op |

## 5. Traceability Checklist

The Critic scores Traceability = 1 only if **all** are present:

- [ ] `command` and resolved `args` (post-substitution)
- [ ] `exit_code` + `stdout` (≤ 4 KB) + `stderr`
- [ ] `region` / `project_id` actually used
- [ ] `job_id` extracted for async ops (LB create/delete, listener create)
- [ ] **No** `password` / `PASSWORD` / `sk-…` / `SecretAccessKey` value in trace
- [ ] For cert operations: certificate content is redacted, only metadata tracked

## 6. Spec Compliance Anchors

`huaweicloud-elb-ops/references/core-concepts.md` rules the Critic enforces:

- LB types: `shared` (classic), `dedicated` (enhanced)
- Listener protocols: `HTTP`, `HTTPS`, `TCP`, `UDP`
- Pool protocols: `HTTP`, `HTTPS`, `TCP`, `UDP` (must match listener protocol)
- Health check: interval 5–300s, timeout 1–300s, max retries 1–10
- Port range: 1–65535
- LB provisioning states: `ACTIVE`, `PENDING_CREATE`, `PENDING_UPDATE`, `PENDING_DELETE`, `ERROR`
- EIP binding: optional; warn on delete-LB with active EIP

## 7. Scoring Summary

| Op | Correctness | Safety | Idempotency | Traceability | Spec Compliance | Pass Threshold |
|----|-------------|--------|-------------|--------------|-----------------|----------------|
| `create-lb` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass |
| `delete-lb` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S1/S2/S3/S4 |
| `create-listener` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S5/S11 |
| `delete-listener` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass |
| `create-pool` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass |
| `delete-pool` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S7 |
| `add-member` | ≥ 0.5 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S8 |
| `remove-member` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S9 |
| `update-certificate` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S10 |
| `delete-certificate` | = 1 | = 1 | ≥ 0.5 | ≥ 0.5 | ≥ 0.5 | all pass + S13 |

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
- `references/core-concepts.md` — ELB type / protocol / port anchors
- `references/troubleshooting.md` — ELB error code mapping