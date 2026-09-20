# Runtime Quality Gates: GCL

Detailed runtime-quality specs are externalized. Key reads before modifying GCL-related files:

| Spec / Tool | Read or run before modifying |
|---|---|
| `docs/gcl-spec.md` | any `## Quality Gate (GCL)` section, `references/rubric.md`, `references/prompt-templates.md` |
| `hwcloud-skillcheck gcl run --root .` | runtime Orchestrator loop; external Critic required in production |
| `hwcloud-skillcheck validate --root .` | Go total-entry: frontmatter + eval-queries + product-assessment + advanced-coverage + audit-results |

## Hard Constraints

- **Contexts**: isolated Generator + Critic only; shared-context G+C banned.
- **Critic**: read-only, no hcloud/SDK/mutation/self-score; sees sanitized `{{output.operation_intent}}` only.
- **Safety=0/SAFETY_FAIL**: abort immediately, never partial output.
- **Loops bounded**: every run has `max_iterations` + masked trace to `audit-results/gcl-trace-*.json`.
- **Templates**: placeholders MUST use `{{env.*}}/{{user.*}}/{{output.*}}`; bare `{…}` banned.

## CLI Commands

```bash
hwcloud-skillcheck validate --root .             # Go total-entry: frontmatter + eval-queries + product-assessment + advanced-coverage + audit-results
hwcloud-skillcheck gcl run --root . --skill huaweicloud-billing-ops --request "smoke" --command 'printf ok' --max-iter 1 --structural-critic-only
hwcloud-skillcheck aggregate trace --root . --since-hours 168
hwcloud-skillcheck gcl alarm-wire --root . --plan-file scripts/fixtures/gcl-quality-summary-healthy.json
```

## Relationship to build-time self-reflection

Build-time 2-round self-reflection and runtime GCL are independent gates. A clean self-reflection does not exempt runtime scoring; a passing GCL rubric does not exempt sloppy skill updates.

## GCL changelog

| Version | Date | Change |
|---|---|---|
| 1.0.0 | 2026-06-04 | Initial GCL specification and ECS pilot |
| 1.6.0 | 2026-06-19 | qcloud-style runtime scripts, sanitized `operation_intent`, Tier-A conformance, and CES summary schema added |
