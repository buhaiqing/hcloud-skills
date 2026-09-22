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

## GCL changelog

| Version | Date | Change |
|---|---|---|
| 1.0.0 | 2026-06-04 | Initial GCL specification and ECS pilot |
| 1.6.0 | 2026-06-19 | qcloud-style runtime scripts, sanitized `operation_intent`, Tier-A conformance, and CES summary schema added |
| 1.7.0 | 2026-09-20 | P0 loop closure: `final.critic_type` provenance, two-family trace consumption (L4 traces schema-compatible + real scores), smoke exclusion via `skipped_smoke`, `l2_skipped_no_schema` observability, `--structural-critic-only` implemented (mutually exclusive with `--critic-cmd`), dangling `delegates_to` targets fail `validate` |
| 1.8.0 | 2026-09-20 | Trace trust: canonical-schema validation on the consumption path (`invalid_trace`), `evidence_runs` + `aggregate trace --require-evidence`, `SAFETY_FAIL` never smoke, `by_critic_type` normalization, validated trace-derived patterns (`provenance: trace`, `verified: false`), `hallucination_detection` / `final.failure_pattern` added to `MaskedFields` |
