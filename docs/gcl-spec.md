# Generator-Critic-Loop (GCL) — Adversarial Quality Gate

> Inspired by GAN's Generator/Discriminator idea, but deliberately **not** a real GAN.
> Naming: **GCL (Generator-Critic-Loop)** to avoid misleading reviewers and LLM trainees.
> This document is the detailed runtime spec for Huawei Cloud (`hcloud` CLI / Go SDK fallback).

## 1. Purpose

Apply an adversarial **Generator ↔ Critic** loop with a quantitative rubric to every skill execution.
Most valuable in high-side-effect cloud operations (`delete`, `stop`, `restore`, IAM/KMS/DDL) where a single mistake is unrecoverable.

| GAN (real) | GCL (this spec) |
|---|---|
| Discriminator learns sample distribution | Critic scores an **explicit rubric** |
| No termination condition | Must terminate: **PASS / MAX_ITER / SAFETY_FAIL** |
| G and D train in parallel | G and C run **sequentially** |
| Goal: "fool the D" | Goal: "pass the rubric threshold" |

## 2. Roles

| Role | Job | Input | Output | Forbidden |
|---|---|---|---|---|
| **Generator (G)** | Execute the cloud operation | user request + previous Critic feedback | result + execution trace | modifying rubric; self-scoring |
| **Critic (C)** | Independently audit output | generator result + trace + rubric + sanitized operation intent | scores + suggestions | calling `hcloud`, SDK clients, or mutating resources |
| **Orchestrator (O)** | Loop control | context + Critic scores + budget | continue / final result | executing or scoring on its own |

**Hard constraint:** Generator and Critic MUST run in isolated prompt contexts. Shared-context G+C is banned.

## 3. Rubric

Each required/recommended skill keeps:

- `## Quality Gate (GCL)` in `SKILL.md`
- `references/rubric.md`
- `references/prompt-templates.md`

Minimum dimensions:

| Dimension | Meaning | Scale | Default threshold |
|---|---|---|---|
| **Correctness** | Resource id / state / config actually matches the request | 0 / 0.5 / 1 | ≥ 0.5; 1.0 for destructive/IAM/KMS/DDL |
| **Safety** | Destructive op was confirmed or guarded | 0 / 1 | = 1 |
| **Idempotency** | Retry does not duplicate side-effects | 0 / 0.5 / 1 | ≥ 0.5 |
| **Traceability** | Command, params, raw response, errors captured | 0 / 0.5 / 1 | ≥ 0.5 |
| **Spec Compliance** | Conforms to `core-concepts.md` / `cli-usage.md` constraints | 0 / 0.5 / 1 | ≥ 0.5 |

**Safety = 0 → ABORT immediately**, regardless of total score.

## 4. Loop Flow

```text
User Request
  ↓
[0] Pre-flight (Orchestrator)
  - resolve env.* and user.* variables
  - pick skill, load rubric
  - derive sanitized operation_intent: operation, expected_state, resource_scope, safety_class
  - omit raw user wording, credentials, and unmasked sensitive identifiers
  ↓
[1] Generate (G)
  - run hcloud or JIT Go SDK fallback
  - capture command/args/exit/raw response/request_id/job_id
  ↓
[2] Critique (C)
  - isolated prompt context
  - score rubric dimensions
  - emit ≤3 concrete suggestions
  ↓
[3] Decide (O)
  - Safety=0 → SAFETY_FAIL
  - all thresholds pass → PASS
  - else and iter<max_iter → retry with critic_feedback
  - else → MAX_ITER
```

The Orchestrator owns `operation_intent` generation. Critic MUST NOT see raw user wording; it may use `{{output.operation_intent}}`.

## 5. Termination

| Condition | Behavior |
|---|---|
| **PASS** | Every rubric dimension meets threshold → return result |
| **MAX_ITER** | Max iterations reached → return best-so-far + unresolved rubric items |
| **SAFETY_FAIL** | Safety = 0 → abort; never return partial or best-effort output |

## 6. Trace & Audit Schema

Every GCL run MUST persist a masked JSON trace under `audit-results/gcl-trace-YYYYMMDD-HHMMSS.json`.

```json
{
  "trace_schema_version": "v3",
  "trace_id": "gcl-20260725-153623-a1b2c3d4",
  "skill": "huaweicloud-ecs-ops",
  "skill_version": "1.1.0",
  "request": "<sanitized user request>",
  "operation_intent": {
    "operation": "stop-server",
    "resource_scope": ["ecs-***"],
    "expected_state": "SHUTOFF",
    "safety_class": "destructive"
  },
  "rubric_version": "v1",
  "masked_fields": ["request", "operation_intent.resource_scope"],
  "started_at": "2026-07-25T15:36:23Z",
  "finished_at": "2026-07-25T15:36:24Z",
  "duration_ms": 1234,
  "environment": {
    "runner_version": "2.0.0",
    "python_version": "3.10.x",
    "platform": "Linux",
    "ci": false
  },
  "token_usage": {
    "model": "model-id",
    "input_tokens": 1500,
    "output_tokens": 800,
    "total_tokens": 2300,
    "estimated_cost_usd": 0.0035,
    "cache_hit_tokens": 200,
    "by_phase": {
      "generator": {"input_tokens": 1000, "output_tokens": 500},
      "critic": {"input_tokens": 500, "output_tokens": 300}
    },
    "retry_waste_tokens": 1150,
    "retry_waste_cost_usd": 0.00175,
    "cost_per_iteration_usd": 0.00175
  },
  "resource_context": {
    "resource_id": "ecs-***",
    "resource_type": "ecs",
    "region": "cn-north-4",
    "billing_model": "on-demand",
    "monthly_cost_usd": 150.0,
    "utilization_pct": 35.2
  },
  "cost_attribution": {
    "cloud_api_calls": 2,
    "ai_cost_usd": 0.0035,
    "resource_cost_usd": 0.0000071,
    "total_cost_usd": 0.003507,
    "cost_per_api_call_usd": 0.00175
  },
  "incident": {
    "incident_id": "INC-2026-0042",
    "severity": "P2",
    "alert_fingerprint": "ces-alarm-ecs-cpu-high",
    "triggered_at": "2026-07-25T15:30:00Z",
    "mttr_target_minutes": 30
  },
  "slo_context": {
    "slo_id": "slo-ecs-availability",
    "target": 0.999,
    "current": 0.9995,
    "error_budget_remaining_pct": 85.0,
    "burn_rate_1h": 1.2
  },
  "change_impact": {
    "blast_radius": "single-resource",
    "affected_services": ["web-frontend"],
    "rollback_plan": "start-server",
    "change_window": "maintenance"
  },
  "anomaly_baseline": {
    "metric": "cpu_utilization",
    "normal_range": [10.0, 60.0],
    "current_value": 95.3,
    "deviation_sigma": 3.2,
    "detection_method": "3-sigma"
  },
  "ops_efficiency": {
    "retry_count": 1,
    "wasted_time_ms": 900,
    "first_success_iter": 2,
    "total_api_calls": 2,
    "automation_level": "assisted",
    "total_duration_ms": 1234
  },
  "pre_execution_risk": {
    "pattern_id": "ECS-FP001",
    "category": "resource_state",
    "known_fix": "...",
    "historical_success_rate": 0.85
  },
  "iterations": [
    {
      "iter": 1,
      "generator": { "command": "...", "args": {}, "exit_code": 0, "result_excerpt": "...", "duration_ms": 900 },
      "critic": {
        "scores": { "correctness": 1, "safety": 1, "idempotency": 0.5, "traceability": 1, "spec_compliance": 1 },
        "suggestions": ["..."],
        "blocking": false
      },
      "decision": "PASS"
    }
  ],
  "final": {
    "status": "PASS",
    "iter": 1,
    "output": "...",
    "failure_pattern": null
  }
}
```

### v2 additions (backward-compatible, all new fields optional)

| Field | Purpose |
|---|---|
| `trace_id` | Unique correlation ID (format: `gcl-YYYYMMDD-HHMMSS-<hex8>`) |
| `skill_version` | Extracted from `SKILL.md` frontmatter `metadata.version` |
| `started_at` / `finished_at` / `duration_ms` | Wall-clock timing for SLA analysis |
| `environment` | Runner version, Python version, platform, CI flag |
| `token_usage` | Token cost analysis: model, per-phase breakdown, estimated cost |
| `pre_execution_risk` | Matched failure pattern from knowledge base (see Self-Healing spec) |
| `generator.duration_ms` | Per-command execution time |

### v3 additions — FinOps + AIOps factual data (all optional, backward-compatible)

| Field | Category | Purpose |
|---|---|---|
| `resource_context` | FinOps | Cloud resource cost context for right-sizing and idle detection |
| `cost_attribution` | FinOps | Operation-level cost breakdown (AI + cloud resource) |
| `token_usage.retry_waste_tokens` | FinOps | Tokens wasted on retry iterations |
| `token_usage.retry_waste_cost_usd` | FinOps | Monetary cost of retry waste |
| `token_usage.cost_per_iteration_usd` | FinOps | Average token cost per GCL iteration |
| `incident` | AIOps | Incident correlation for MTTR tracking |
| `slo_context` | AIOps | SLO error budget and burn rate at execution time |
| `change_impact` | AIOps | Blast radius and rollback plan for change management |
| `anomaly_baseline` | AIOps | Anomaly detection baseline with sigma deviation |
| `ops_efficiency` | AIOps | Derived operational efficiency metrics |

---

## 6A. FinOps Data Contract

FinOps fields enable **operation-level cost visibility** — every GCL run captures the true cost of AI-assisted cloud operations.

### `resource_context` (injected via `--context-json`)

Describes the target cloud resource's cost profile. Enables idle detection, right-sizing analysis, and billing model optimization.

| Key | Type | Required | Description |
|---|---|---|---|
| `resource_id` | string | yes | Masked resource identifier (e.g. `ecs-***`) |
| `resource_type` | string | yes | Huawei Cloud service type (`ecs`, `rds`, `dcs`, etc.) |
| `region` | string | yes | Deployment region (e.g. `cn-north-4`) |
| `billing_model` | string | no | `on-demand` \| `yearly` \| `monthly` \| `spot` |
| `monthly_cost_usd` | float | no | Estimated monthly cost in USD (from BSS API) |
| `utilization_pct` | float | no | Current resource utilization percentage (0–100) |

**Use cases:**
- FinOps idle detection: `utilization_pct < 5` + `billing_model == on-demand` → right-sizing candidate
- Cost trend: aggregate `monthly_cost_usd` across traces per resource_type
- Billing model comparison: on-demand vs yearly savings calculation

### `cost_attribution` (auto-computed)

Derived by the runner at trace finalization. Attributes total operation cost to AI inference and cloud resource time-slice.

| Key | Type | Computed from | Description |
|---|---|---|---|
| `cloud_api_calls` | int | `iterations[].generator.exit_code` count | Number of cloud API invocations |
| `ai_cost_usd` | float | `token_usage.estimated_cost_usd` | Total AI inference cost |
| `resource_cost_usd` | float | `resource_context.monthly_cost_usd / 720 * duration_hours` | Pro-rated resource cost during operation |
| `total_cost_usd` | float | `ai_cost_usd + resource_cost_usd` | Combined operation cost |
| `cost_per_api_call_usd` | float | `ai_cost_usd / cloud_api_calls` | Average AI cost per API call |

**Use cases:**
- Unit economics: cost per operation type / skill
- Waste detection: high `cost_per_api_call_usd` indicates retry-heavy operations
- Budget forecasting: aggregate `total_cost_usd` by skill/region/time window

### `token_usage` enhanced fields (auto-computed)

Extends the base token contract with retry waste analysis:

| Key | Type | Computed from | Description |
|---|---|---|---|
| `retry_waste_tokens` | int | `total_tokens * (iters-1) / iters` | Tokens consumed by failed retry iterations |
| `retry_waste_cost_usd` | float | `estimated_cost_usd * (iters-1) / iters` | Monetary cost of retry waste |
| `cost_per_iteration_usd` | float | `estimated_cost_usd / iters` | Average cost per GCL iteration |

**Use cases:**
- Retry ROI: if `retry_waste_cost_usd` consistently high → improve first-pass success rate
- Model comparison: `cost_per_iteration_usd` across different models
- Budget alert: cumulative retry waste exceeding threshold

---

## 6B. AIOps Data Contract

AIOps fields enable **intelligent operations correlation** — connecting GCL executions to incidents, SLOs, anomalies, and change management.

### `incident` (injected via `--context-json`)

Links the GCL run to an active incident for MTTR tracking and root-cause correlation.

| Key | Type | Required | Description |
|---|---|---|---|
| `incident_id` | string | yes | External incident identifier (e.g. `INC-2026-0042`) |
| `severity` | string | yes | `P1` \| `P2` \| `P3` \| `P4` |
| `alert_fingerprint` | string | no | CES alarm rule fingerprint that triggered the incident |
| `triggered_at` | string (ISO 8601) | no | When the incident was first detected |
| `mttr_target_minutes` | int | no | Target Mean-Time-To-Resolve for this severity level |

**Use cases:**
- MTTR calculation: `trace.finished_at - incident.triggered_at` vs `mttr_target_minutes`
- Incident frequency: aggregate by `alert_fingerprint` to identify recurring issues
- Severity-based routing: P1 incidents trigger stricter safety gates

### `slo_context` (injected via `--context-json`)

Captures SLO state at execution time for error-budget-aware decision making.

| Key | Type | Required | Description |
|---|---|---|---|
| `slo_id` | string | yes | SLO identifier (e.g. `slo-ecs-availability`) |
| `target` | float | yes | SLO target (e.g. 0.999 for 99.9% availability) |
| `current` | float | yes | Current SLO achievement value |
| `error_budget_remaining_pct` | float | no | Percentage of error budget remaining (0–100) |
| `burn_rate_1h` | float | no | Error budget burn rate over last hour (1.0 = on-track) |

**Use cases:**
- Change freeze: `error_budget_remaining_pct < 10` → block non-critical mutations
- Burn rate alert: `burn_rate_1h > 5` → escalate to on-call
- Risk-adjusted execution: low budget → prefer read-only diagnostics over mutations

### `change_impact` (injected via `--context-json`)

Documents the expected blast radius and rollback strategy for change management integration.

| Key | Type | Required | Description |
|---|---|---|---|
| `blast_radius` | string | yes | `single-resource` \| `multi-resource` \| `service-wide` \| `region-wide` |
| `affected_services` | list[string] | no | Downstream service names impacted by this change |
| `rollback_plan` | string | no | One-line rollback command or strategy |
| `change_window` | string | no | `maintenance` \| `business-hours` \| `off-peak` |

**Use cases:**
- Approval routing: `blast_radius >= service-wide` → require manual approval
- Rollback automation: `rollback_plan` feeds into auto-recovery playbook
- Change calendar: correlate `change_window` with incident frequency

### `anomaly_baseline` (injected via `--context-json`)

Records the anomaly detection context that triggered (or relates to) this GCL execution.

| Key | Type | Required | Description |
|---|---|---|---|
| `metric` | string | yes | CES metric name (e.g. `cpu_utilization`, `memory_used_percent`) |
| `normal_range` | [float, float] | yes | Baseline [min, max] under normal conditions |
| `current_value` | float | yes | Observed metric value at execution time |
| `deviation_sigma` | float | no | Number of standard deviations from mean |
| `detection_method` | string | no | `3-sigma` \| `ewma` \| `percentile-p99` \| `static-threshold` |

**Use cases:**
- Auto-remediation trigger: `deviation_sigma > 3` → auto-invoke healing playbook
- Baseline drift: track `normal_range` shifts over time for capacity planning
- False-positive analysis: correlate `detection_method` with actual incident outcomes

### `ops_efficiency` (auto-computed)

Derived by the runner at trace finalization. Quantifies operational efficiency for continuous improvement.

| Key | Type | Computed from | Description |
|---|---|---|---|
| `retry_count` | int | `len(iterations) - 1` | Number of retry iterations before terminal state |
| `wasted_time_ms` | int | Sum of non-final `generator.duration_ms` | Time spent on failed attempts |
| `first_success_iter` | int \| null | First iteration with `decision == PASS` | Which iteration first succeeded (null = never passed) |
| `total_api_calls` | int | `len(iterations)` | Total cloud API invocations |
| `automation_level` | string | `full` if 1 iter + PASS, else `assisted` | Degree of automation achieved |
| `total_duration_ms` | int | `trace.duration_ms` | End-to-end wall-clock time |

**Use cases:**
- Automation rate: aggregate `automation_level == full` percentage across all traces
- Efficiency trend: track `retry_count` and `wasted_time_ms` over time per skill
- Skill quality signal: high `retry_count` for a skill → skill runbook needs improvement
- MTTR contribution: `total_duration_ms` feeds into mean-time-to-resolve calculations

---

## 6C. Runtime Context Injection Contract

All FinOps/AIOps context fields are injected via `--context-json <path>` at runtime. The file is a flat JSON object; the runner extracts known keys and ignores unknown ones.

```bash
hwcloud-skillcheck gcl run --root . \
  --skill huaweicloud-ecs-ops \
  --request "Stop ECS instance" \
  --command 'hcloud ecs stop-server --server-id xxx' \
  --context-json /tmp/ops-context.json \
  --token-json /tmp/token-usage.json \
  --structural-critic-only
```

### `--context-json` schema (all keys optional)

```json
{
  "resource_context": { "resource_id": "...", "resource_type": "ecs", "region": "cn-north-4", "billing_model": "on-demand", "monthly_cost_usd": 150.0, "utilization_pct": 35.2 },
  "incident": { "incident_id": "INC-...", "severity": "P2", "alert_fingerprint": "...", "triggered_at": "...", "mttr_target_minutes": 30 },
  "slo_context": { "slo_id": "...", "target": 0.999, "current": 0.9995, "error_budget_remaining_pct": 85.0, "burn_rate_1h": 1.2 },
  "change_impact": { "blast_radius": "single-resource", "affected_services": ["web"], "rollback_plan": "start-server", "change_window": "maintenance" },
  "anomaly_baseline": { "metric": "cpu_utilization", "normal_range": [10, 60], "current_value": 95.3, "deviation_sigma": 3.2, "detection_method": "3-sigma" }
}
```

### Injection rules

| Rule | Description |
|---|---|
| Optional | Missing `--context-json` → no context fields in trace (backward-compatible) |
| Partial | Only provided keys are injected; absent keys are omitted from trace |
| Read-only | Runner never modifies context values; passes through as-is |
| Masking | `resource_id` values SHOULD be pre-masked by the caller (runner applies no additional masking to context) |
| Unknown keys | Silently ignored (forward-compatible with future schema extensions) |

### Token Usage Contract

`token_usage` is injected via `--token-json <path>` at runtime. Base schema:

| Key | Type | Required | Description |
|---|---|---|---|
| `model` | string | yes | Model identifier used for generation |
| `input_tokens` | int | yes | Total prompt tokens consumed |
| `output_tokens` | int | yes | Total completion tokens produced |
| `total_tokens` | int | yes | `input_tokens + output_tokens` |
| `estimated_cost_usd` | float | no | Estimated monetary cost |
| `cache_hit_tokens` | int | no | Tokens served from prompt cache |
| `by_phase` | object | no | Breakdown by GCL phase (`generator`, `critic`) |

Enhanced fields (`retry_waste_tokens`, `retry_waste_cost_usd`, `cost_per_iteration_usd`) are **auto-computed** by the runner — callers MUST NOT provide them in `--token-json`.

Trace files are append-only; do not overwrite/delete in place. `audit-results/` and `gcl-trace-*.json` are gitignored.

## 7. Prompt Templates

Each skill's `references/prompt-templates.md` MUST contain numbered sections `## 1.` through `## 7.` and include:

1. Generator Prompt Template — placeholders include `{{user.request}}`, `{{output.critic_feedback}}`, `{{output.rubric}}`
2. Critic Prompt Template — placeholders include `{{output.operation_intent}}`, `{{output.generator_output}}`, `{{output.trace}}`, `{{output.rubric}}`
3. Orchestrator Loop Template
4. Sanitization rules
5. Failure recovery
6. Changelog
7. See also

Placeholder syntax MUST follow `{{env.*}}` / `{{user.*}}` / `{{output.*}}`; bare `{...}` placeholders are banned.

## 8. Per-Skill Defaults

| Skill | GCL | max_iter | Notes |
|---|---|---:|---|
| `huaweicloud-ecs-ops` | required | 2 | delete/stop/reboot |
| `huaweicloud-iam-ops` | required | 2 | detach policy / delete user / rotate keys |
| `huaweicloud-rds-ops` | required | 2 | delete / DDL / restore |
| `huaweicloud-gaussdb-ops` | required | 2 | delete / DDL / shard rebalance |
| `huaweicloud-dcs-ops` | required | 2 | FLUSHALL / delete / restore |
| `huaweicloud-dms-ops` | required | 2 | queue delete / message purge |
| `huaweicloud-css-ops` | required | 2 | cluster delete / snapshot restore |
| `huaweicloud-cce-ops` | required | 2 | node drain / cluster delete |
| `huaweicloud-cbr-ops` | required | 2 | restore overwrites source |
| `huaweicloud-vpc-ops` | required | 2 | delete VPC/subnet/SG cascades |
| `huaweicloud-obs-ops` | required | 2 | bucket delete / lifecycle purge |
| `huaweicloud-swr-ops` | required | 2 | image delete / tag overwrite |
| `huaweicloud-functiongraph-ops` | required | 2 | function delete / version disable |
| `huaweicloud-waf-ops` | required | 2 | policy delete / rule disable |
| `huaweicloud-hss-ops` | required | 2 | host isolate / policy detach |
| `huaweicloud-elb-ops` | recommended | 3 | listener/backend/cert changes |
| `huaweicloud-ces-ops` | recommended | 3 | alarm rule delete |
| `huaweicloud-lts-ops` | recommended | 3 | log group/stream delete |
| `huaweicloud-cts-ops` | recommended | 3 | tracker disable / transfer delete |
| `huaweicloud-billing-ops` | optional | 5 | read-only reports |
| `huaweicloud-skill-generator` | optional | 3 | meta operation |

## 9. Runtime Scripts

| Command | Purpose |
|---|---|
| `hwcloud-skillcheck gcl run --root .` | Orchestrator loop; external Critic required in production |
| `hwcloud-skillcheck aggregate trace --root .` | Aggregate traces into quality summary |
| `hwcloud-skillcheck gcl alarm-wire --root .` | Plan/apply CES alarms from summary |
| `hwcloud-skillcheck validate --root .` | Go total-entry: frontmatter + eval-queries + product-assessment + advanced-coverage + audit-results |

### Phase 4 CES Alarm Wiring Contract

`hwcloud-skillcheck gcl alarm-wire --root .` reads the `gcl_quality:` block from
`huaweicloud-ces-ops/assets/example-config.yaml`. The block MUST define:

| Key | Type | Default | Purpose |
|---|---|---|---|
| `pass_rate_warn` | float (0–1) | 0.85 | `<` triggers WARN alarm |
| `pass_rate_critical` | float (0–1) | 0.70 | `<` triggers CRITICAL alarm |
| `max_iter_warn_count` | int (≥1) | 3 | `>` triggers WARN alarm |
| `safety_fail_alert` | bool | true | `== 0` invariant; CRITICAL on any SAFETY_FAIL |

Wiring contract rules:

- `pass_rate_critical ≤ pass_rate_warn ≤ 1.0` and both `≥ 0.0`.
- Numeric thresholds MUST match `gcl_alarm_wire.DEFAULT_THRESHOLDS` exactly
  (drift is caught by `hwcloud-skillcheck validate --root .` under the product-assessment gate).
- `audit-results/gcl-alarm-plan-*.json` files MUST agree with the wiring
  config; rerun `hwcloud-skillcheck gcl alarm-wire --root . --write-plan` after edits.
- `docs/gcl-spec.md` MUST keep the four threshold names documented.

Production GCL MUST use externally supplied isolated Critic scores via `--critic-json` or stdin. `--structural-critic-only` is only for CI/local smoke tests and cannot approve production or human acceptance gates.

## 10. Anti-Patterns

- Shared context G+C
- Subjective scoring instead of rubric scoring
- Unbounded loop
- Critic sees raw user request
- Safety fail silently downgraded
- Trace not persisted
- Critic mutates resources
- Structural critic used as production quality pass
- Printing/logging credentials

## 11. Monitoring Integration

GCL quality summaries are owned by `huaweicloud-ces-ops`:

- Schema: `huaweicloud-ces-ops/assets/gcl-quality-summary.schema.json`
- Design: `huaweicloud-ces-ops/references/gcl-monitoring.md`
- Namespace: `CUSTOM.GCL`
- Alarm plan: `hwcloud-skillcheck gcl alarm-wire --root . --plan-file <summary.json>`

## 12. Changelog

| Version | Date | Change |
|---|---|---|
| 1.0.0 | 2026-06-04 | Initial GCL specification |
| 1.1.0 | 2026-06-04 | ECS pilot rollout |
| 1.2.0 | 2026-06-04 | Phase 2 rollout to high-blast-radius skills |
| 1.3.0 | 2026-06-04 | All 20 skills gained GCL artifacts |
| 1.4.0 | 2026-06-04 | CES monitoring design |
| 1.5.0 | 2026-06-05 | Moved detailed spec to `docs/gcl-spec.md` |
| 1.6.0 | 2026-06-19 | Added qcloud-style runtime scripts, sanitized `operation_intent`, Tier-A conformance, and CES quality-summary contract |
| 1.7.0 | 2026-07-25 | Trace schema v2: added `trace_id`, `skill_version`, timing, `environment`, `token_usage` contract, `pre_execution_risk`, per-iteration `duration_ms` |
| 1.8.0 | 2026-07-25 | Trace schema v3: FinOps (`resource_context`, `cost_attribution`, token retry-waste) + AIOps (`incident`, `slo_context`, `change_impact`, `anomaly_baseline`, `ops_efficiency`) full data contract; `--context-json` injection |
| 1.9.0 | 2026-07-27 | Trust boundary contract (§14): `SanitizeRequest` enforces fail-closed sanitization of user request; `applyMaskFields` enforces `MaskedFields` declarations on persist; embedded JSON-schema validators for `operation_intent` (`embed.OperationIntentSchema`) and `critic_output` (`embed.CriticOutputSchema`) gate Critic input/output at the wire format |

## 13. See also

- `AGENTS.md` — always-loaded GCL hard constraints and validation pointers
- `huaweicloud-*-ops/references/rubric.md` — per-skill scoring rubrics
- `huaweicloud-*-ops/references/prompt-templates.md` — G/C/O templates
- `huaweicloud-ces-ops/references/gcl-monitoring.md` — CES monitoring design

## 14. Trust Boundary Contract

The GCL pipeline crosses three trust boundaries. Each boundary is enforced by a
specific code path; every cross-boundary payload MUST be sanitized, validated,
or masked at the boundary — never trusted downstream.

```text
┌──────────────────────────────────────────────────────────────────────────┐
│ User / CLI                                                               │
│   raw request (may contain resource IDs, ARNs, credentials)             │
└────────────┬─────────────────────────────────────────────────────────────┘
             │ (1) SanitizeRequest  — fail-closed on unrecognized tokens
             ▼
┌──────────────────────────────────────────────────────────────────────────┐
│ Orchestrator                                                             │
│   trace.SanitizedRequest  → surfaced to Critic                          │
│   trace.Request           → kept raw for audit only                     │
└────────────┬─────────────────────────────────────────────────────────────┘
             │ (2) Critic output must validate against critic_output.schema
             ▼
┌──────────────────────────────────────────────────────────────────────────┐
│ Critic (isolated prompt context)                                         │
│   receives sanitized_request + operation_intent + generator_output      │
│   emits scores {correctness, safety, idempotency,                        │
│                 traceability, spec_compliance} ∈ [0, 1]                  │
└────────────┬─────────────────────────────────────────────────────────────┘
             │ (3) applyMaskFields — wipe fields listed in masked_fields
             ▼
┌──────────────────────────────────────────────────────────────────────────┐
│ audit-results/gcl-trace-<ts>-<hex>.json  (mode 0600)                     │
```

### 14.1 Boundary 1 — Request sanitization (Orchestrator input)

The `SanitizeRequest` function in
`hwcloud-skillcheck/internal/gcl/sanitizer.go` is the single point that
transforms a raw user request into a Critic-safe form. Rules, in priority
order:

| Pattern | Replacement | Example |
|---|---|---|
| Huawei Cloud resource IDs (`ecs-…`, `rds-…`, `vpc-…`, `dcs-…`, `elb-…`, `cce-…`, `kms-…`, `iam-…`, `sg-…`, 4+ char suffix) | `<id>` | `ecs-abc12345` → `<id>` |
| ARNs (`acs:<region>:<svc>:<type>:<id>`) | `<arn>` | `acs:cn-north-4:project:ecs:instance:abc` → `<arn>` |
| Embedded credentials (`HW_SECRET_ACCESS_KEY=…`, `SecretAccessKey=…`, `AK=…`, `SK=…`) | `<redacted>` | `AK=ABCDEFGHIJ1234567890` → `<redacted>` |
| Already-masked markers (`***`, `<masked>`, `<id>`, `<arn>`, `<redacted>`) | pass-through (idempotent) | n/a |

**Fail-closed rule.** If `SanitizeRequest` encounters a long token (>32 chars,
alphanumeric) that matches no pattern above, it returns an error. The caller
(`Run`) MUST propagate that error as `ExitUsage` — it MUST NOT persist a
trace containing the unrecognized token. This is the "Critic request leak"
defense specified in §10 Anti-Patterns.

### 14.2 Boundary 2 — Critic output schema validation

Critic output is validated against
`hwcloud-skillcheck/internal/embed/schemas/critic_output.schema.json` (exposed
via `embed.CriticOutputSchema`). The schema is closed for `scores`
(`additionalProperties: false`) — every required dimension is fixed:
`correctness`, `safety`, `idempotency`, `traceability`, `spec_compliance`,
each `number` in `[0, 1]`. Optional fields: `suggestions` (string array),
`blocking` (boolean), `mode` (string).

A Critic output that fails validation MUST be rejected before it can drive a
`Decide(...)` call. The rejection surfaces as `ExitUsage` and is logged with
the underlying `schema.ValidateFile` errors.

### 14.3 Boundary 3 — operation_intent schema validation

The Orchestrator-derived `operation_intent` is validated against
`hwcloud-skillcheck/internal/embed/schemas/operation_intent.schema.json`
(exposed via `embed.OperationIntentSchema`). Required: `operation`,
`expected_state`, `safety_class ∈ {"read-only", "mutating", "destructive"}`.
Optional: `resource_scope` (string array, pre-masked).

This is in addition to the runtime enum check in `gcl.enforceSafetyClass`
(`internal/gcl/sanitizer.go`) — the schema is the wire-format guard, the
enum check is the runtime guard. Both MUST pass before the intent reaches
the Critic.

### 14.4 Boundary 4 — Trace masking on persist

`PersistTrace` in `hwcloud-skillcheck/internal/gcl/runner.go` invokes
`applyMaskFields(trace)` immediately before `json.MarshalIndent`. The helper
walks `trace.MaskedFields` (the *list* of fields the run promised to mask)
and replaces each with the `<masked>` sentinel:

| `MaskedFields` entry | Action |
|---|---|
| `"request"` | `trace.Request = "<masked>"` (raw request stays only in memory) |
| `"operation_intent"` | passed through; resource_scope was already masked by `SanitizeOperationIntent` |
| `"generator.command"` | each `iterations[].generator.command = "<masked>"` |
| `"generator.result_excerpt"` | each `iterations[].generator.result_excerpt = "<masked>"` |

Anything NOT in `MaskedFields` is persisted verbatim. The masking is the
*enforcement* side of the contract declared by `MaskedFields`; the field is
a promise, the helper makes it true.

### 14.5 Summary — the three guarantees

| Guarantee | Enforced by | Failure mode |
|---|---|---|
| Critic never sees raw user wording, ARNs, or credentials | `SanitizeRequest` → `trace.SanitizedRequest` | `ExitUsage` (fail-closed) |
| Critic scores are well-formed and bounded `[0, 1]` | `embed.CriticOutputSchema` + `schema.ValidateFile` | `ExitUsage` (reject + log) |
| Persisted trace never leaks masked fields | `applyMaskFields` in `PersistTrace` | `PersistTrace` returns error; trace not written |

### 14.6 Reference — files defining this contract

| File | Role |
|---|---|
| `hwcloud-skillcheck/internal/gcl/sanitizer.go` (`SanitizeRequest`, `SanitizeOperationIntent`, `MaskResourceID`, `MaskSecrets`) | Boundary 1 |
| `hwcloud-skillcheck/internal/gcl/runner.go` (`Run`, `PersistTrace`, `applyMaskFields`) | Boundaries 1 + 4 |
| `hwcloud-skillcheck/internal/embed/schemas/operation_intent.schema.json` | Boundary 3 |
| `hwcloud-skillcheck/internal/embed/schemas/critic_output.schema.json` | Boundary 2 |
| `hwcloud-skillcheck/internal/embed/embed.go` (`OperationIntentSchema`, `CriticOutputSchema`) | Boundary 2 + 3 — schema accessors |
| `hwcloud-skillcheck/internal/schema/schema.go` (`ValidateFile`) | generic validator reused at Boundary 2 |
