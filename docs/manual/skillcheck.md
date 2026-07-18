# skillcheck CLI Manual

`skillcheck` is the hcloud-skills validation tool. This manual covers B-Class commands (GCL contracts, audit-results guard, and the GCL runtime runner).

## Global Flags

| Flag | Description |
|------|-------------|
| `--root <dir>` | Skill repository root (default `.`) |

## `skillcheck validate` Subcommands

### `skillcheck validate gcl-conformance --root <dir>`

Validates a skill's GCL Tier-A artifact set:
- `references/rubric.md` — 8 required sections
- `references/prompt-templates.md` — 7 required sections
- `## Quality Gate (GCL)` heading in `SKILL.md`
- `{{output.operation_intent}}` placeholder in prompt
- No bare `{placeholder}` tokens (outside code blocks)

Exit codes: `0` pass, `1` fail, `2` safety violation.

### `skillcheck validate generator-contract --root <dir>`

Validates the `huaweicloud-skill-generator` template contract:
- `template.metadata.gcl.required: true`
- `template.metadata.gcl.rubric_version: "v1"`
- `template.quality_gate_heading: ## Quality Gate (GCL)`
- `template.rubric_artifact: references/rubric.md`
- `backbone.generator_section: ## 1. Generator prompt template`
- `backbone.critic_section: ## 2. Critic prompt template`
- `backbone.orchestrator_section: ## 3. Orchestrator prompt template`
- `backbone.hcloud_primary: PRIMARY: hcloud`
- `backbone.operation_intent: {{output.operation_intent}}`
- `backbone.trace_persistence: audit-results/gcl-trace-*.json`
- No bare `{placeholder}` in any template section

Exit codes: `0` pass, `1` fail.

### `skillcheck validate safety-class --root <dir>`

Validates `operation_intent.safety_class` enum contract across the pipeline:
- `huaweicloud-ces-ops/assets/gcl-trace.schema.json` has enum `[read-only mutating destructive]`
- `skillcheck/internal/gcl/sanitizer.go` exports `SAFETY_CLASS_VALUES`
- `docs/gcl-spec.md` and `huaweicloud-skill-generator/references/gcl-prompt-backbone.md` document all three values
- All `gcl-trace-*.json` files in `audit-results/` have valid `safety_class` values

Exit codes: `0` pass, `1` fail.

### `skillcheck validate resource-scope --root <dir>`

Validates `operation_intent.resource_scope` PII masking contract:
- `huaweicloud-ces-ops/assets/gcl-trace.schema.json` defines allowed patterns: `^\*+$`, `^<masked>$`, `^[A-Za-z][A-Za-z0-9-]*-\*+$`
- `skillcheck/internal/gcl/runner.go` implements `MaskResourceID`
- `runner.go` lists `resource_id` and `user_id` in `maskedFields`
- All `gcl-trace-*.json` files use masked forms (`***`, `<masked>`, or `prefix-***`)

Exit codes: `0` pass, `1` fail.

### `skillcheck validate alarm-wire-contract --root <dir>`

Validates CES alarm threshold wiring in `assets/example-config.yaml`:
- `gcl_quality` block has `pass_rate_critical <= pass_rate_warn`
- `safety_fail_alert` is documented in `docs/gcl-spec.md`
- `pass_rate_critical` is documented in `docs/gcl-spec.md`
- Default threshold values: `pass_rate_critical=0.70`, `pass_rate_warn=0.85`

Exit codes: `0` pass, `1` fail.

### `skillcheck check audit-results --root <dir>`

Validates the `audit-results/` directory protection contract:
- `.gitignore` contains: `audit-results/`, `**/audit-results/`, `**/gcl-trace-*.json`, `**/gcl-quality-summary-*.json`, `**/gcl-alarm-plan-*.json`
- `audit-results/` directory has mode `0700` (owner-only)
- No tracked git files inside `audit-results/`
- `docs/gcl-spec.md` contains fragments: `audit-results/`, `GCL`, `gitignore`

Exit codes: `0` pass, `1` fail.

## `skillcheck gcl` Subcommands

### `skillcheck gcl run --root <dir> [--json] [--quiet]`

Runs a GCL structural critic loop against a skill:
1. Loads `SKILL.md`, `references/rubric.md`, `references/prompt-templates.md`
2. Executes smoke command (`echo ok`) through the generator-critic loop
3. Writes trace to `audit-results/gcl-trace-YYYYMMDD-HHMMSS.json`

| Flag | Description |
|------|-------------|
| `--root` | Skill directory (required) |
| `--json` | Emit JSON report |
| `--quiet` | Suppress stdout, print only trace path |

Exit codes: `0` pass, `1` error, `2` safety violation.

### `skillcheck gcl alarm-wire --root <dir> [--json] [--plan-file <path>]`

Evaluates GCL trace quality against CES SLO thresholds and generates an alarm plan:
- Loads `assets/example-config.yaml` for threshold defaults
- Finds most recent `gcl-trace-*.json` in `audit-results/`
- Writes plan to `audit-results/gcl-alarm-plan-YYYYMMDD-HHMMSS-plan.json`

| Flag | Description |
|------|-------------|
| `--root` | Repository root (required) |
| `--json` | Emit JSON report |
| `--plan-file <path>` | Write alarm plan to specific path |

Exit codes: `0` no breaches, `1` threshold breach.
