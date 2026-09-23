# Runtime Quality Gates: GCL

Detailed runtime-quality specs are externalized. Key reads before modifying GCL-related files:

| Spec / Tool | Read or run before modifying |
|---|---|
| `docs/gcl-spec.md` | any `## Quality Gate (GCL)` section, `references/rubric.md`, `references/prompt-templates.md` |
| `hwcloud-skillcheck gcl run --root .` | runtime Orchestrator loop; external Critic required in production |
| `hwcloud-skillcheck validate --root .` | Go total-entry: frontmatter (incl. dangling `delegates_to` targets) + eval-queries + product-assessment + advanced-coverage + audit-results |

## Hard Constraints

- **Contexts**: isolated Generator + Critic only; shared-context G+C banned.
- **Critic**: read-only, no hcloud/SDK/mutation/self-score; sees sanitized `{{output.operation_intent}}` only.
- **Safety=0/SAFETY_FAIL**: abort immediately, never partial output.
- **Loops bounded**: every run has `max_iterations` + masked trace to `audit-results/gcl-trace-*.json`.
- **Templates**: placeholders MUST use `{{env.*}}/{{user.*}}/{{output.*}}`; bare `{…}` banned.

## CLI Commands

```bash
hwcloud-skillcheck validate --root .             # Go total-entry: frontmatter (incl. dangling `delegates_to` targets) + eval-queries + product-assessment + advanced-coverage + audit-results
hwcloud-skillcheck gcl run --root huaweicloud-billing-ops --request "smoke" --command 'printf ok' --max-iter 1 --structural-critic-only
hwcloud-skillcheck aggregate trace --root . --since-hours 168
hwcloud-skillcheck gcl alarm-wire --root . --plan-file scripts/fixtures/gcl-quality-summary-healthy.json
```

`gcl run` has no `--skill` flag — the skill is selected by `--root <skill-dir>`. The pre-execution risk check on `failure_patterns.json` runs in the L4 orchestrator step loop (`hwcloud-skillcheck l4 handle`), not in `gcl run`. Trace trust rules (schema validation, `invalid_trace` / `skipped_smoke` / `evidence_runs` classification, `--require-evidence`, two trace families) → `docs/gcl-spec.md`.

## Relationship to build-time self-reflection

Build-time 2-round self-reflection and runtime GCL are independent gates. A clean self-reflection does not exempt runtime scoring; a passing GCL rubric does not exempt sloppy skill updates.

## Threshold Calibration

Current GCL gate thresholds lack calibration data. The table below states the present values verbatim and flags the missing evidence future re-tuning must supply. Do **not** change any number without attaching escape-case data or recurrence-rate evidence.

| Gate / knob | Current value | Calibration basis |
|---|---|---|
| Correctness (GCL pass bar) | ≥ 0.5 | 无历史逃逸数据回填，暂定；0.5 意味半分放行（half-pass），待 trace 复发率数据回流后重校准 |
| Safety | = 1.0 | 无历史逃逸数据回填，0 容忍；违反即中止，符合安全门惯例 |
| Idempotency | ≥ 0.5 | 无历史逃逸数据回填，暂定；同上 0.5 半分放行缺口 |
| Traceability | ≥ 0.5 | 无历史逃逸数据回填，暂定；同上 0.5 半分放行缺口 |
| Spec Compliance | ≥ 0.5 | 无历史逃逸数据回填，暂定；同上 0.5 半分放行缺口 |
| confidence (low / mid / high) | 0.70 / 0.85 / 0.95 | 无历史逃逸数据回填，暂定；分档间距 0.15 系经验常数，未经回归校验 |
| auto_execute (low / mid / high) | 0.70 / 0.85 / 0.95 | 同上，未对真实误执行数据回测 |

上表字面值已被 `hwcloud-skillcheck validate doc-contracts` 逐字节 pin，校准前视为**冻结**：不得调整，也不存在 ±0.05 的微调额度（无逃逸样本时的微调等于把臆测写进门禁）。解除冻结需同时满足：① 附逃逸案例或复发率证据；② ≥3 个独立 campaign 的 trace 复发率样本；③ **同一 commit 内**同时更新本表与 `cmd/validate_doc_contracts.go` 的对应锚点。只改其一会让 pre-commit/CI 门禁变红——这是有意的：两处分离的任何改动都视为契约破坏。

## GCL changelog

| Version | Date | Change |
|---|---|---|
| 1.0.0 | 2026-06-04 | Initial GCL specification and ECS pilot |
| 1.6.0 | 2026-06-19 | qcloud-style runtime scripts, sanitized `operation_intent`, Tier-A conformance, and CES summary schema added |
| 1.7.0 | 2026-09-20 | P0 loop closure: `final.critic_type` provenance, two-family trace consumption (L4 traces schema-compatible + real scores), smoke exclusion via `skipped_smoke`, `l2_skipped_no_schema` observability, `--structural-critic-only` implemented (mutually exclusive with `--critic-cmd`), dangling `delegates_to` targets fail `validate` |
| 1.8.0 | 2026-09-20 | Trace trust: canonical-schema validation on the consumption path (`invalid_trace`), `evidence_runs` + `aggregate trace --require-evidence`, `SAFETY_FAIL` never smoke, `by_critic_type` normalization, validated trace-derived patterns (`provenance: trace`, `verified: false`), `hallucination_detection` / `final.failure_pattern` added to `MaskedFields` |
