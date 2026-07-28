# hwcloud-skillcheck CLI Manual

`hwcloud-skillcheck` is the hcloud-skills validation tool. This manual covers every shipped subcommand, grouped by command family: `validate`, `check`, `scan`, `aggregate`, `lint`, `gcl`, `learning`, `l4`, `drift`, `critic`, `ab`, `manifest`, `telemetry`, `router`, and `memory`.

## Global Flags

| Flag | Description |
|------|-------------|
| `--root <dir>` | Skill repository root (default `.`) |

## `hwcloud-skillcheck validate` Subcommands

### `hwcloud-skillcheck validate gcl-conformance --root <dir>`

Validates a skill's GCL Tier-A artifact set:
- `references/rubric.md` — 8 required sections
- `references/prompt-templates.md` — 7 required sections
- `## Quality Gate (GCL)` heading in `SKILL.md`
- `{{output.operation_intent}}` placeholder in prompt
- No bare `{placeholder}` tokens (outside code blocks)

Exit codes: `0` pass, `1` fail, `2` safety violation.

### `hwcloud-skillcheck validate generator-contract --root <dir>`

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

### `hwcloud-skillcheck validate safety-class --root <dir>`

Validates `operation_intent.safety_class` enum contract across the pipeline:
- `huaweicloud-ces-ops/assets/gcl-trace.schema.json` has enum `[read-only mutating destructive]`
- `hwcloud-skillcheck/internal/gcl/sanitizer.go` exports `SAFETY_CLASS_VALUES`
- `docs/gcl-spec.md` and `huaweicloud-skill-generator/references/gcl-prompt-backbone.md` document all three values
- All `gcl-trace-*.json` files in `audit-results/` have valid `safety_class` values

Exit codes: `0` pass, `1` fail.

### `hwcloud-skillcheck validate resource-scope --root <dir>`

Validates `operation_intent.resource_scope` PII masking contract:
- `huaweicloud-ces-ops/assets/gcl-trace.schema.json` defines allowed patterns: `^\*+$`, `^<masked>$`, `^[A-Za-z][A-Za-z0-9-]*-\*+$`
- `hwcloud-skillcheck/internal/gcl/runner.go` implements `MaskResourceID`
- `runner.go` lists `resource_id` and `user_id` in `maskedFields`
- All `gcl-trace-*.json` files use masked forms (`***`, `<masked>`, or `prefix-***`)

Exit codes: `0` pass, `1` fail.

### `hwcloud-skillcheck validate alarm-wire-contract --root <dir>`

Validates CES alarm threshold wiring in `assets/example-config.yaml`:
- `gcl_quality` block has `pass_rate_critical <= pass_rate_warn`
- `safety_fail_alert` is documented in `docs/gcl-spec.md`
- `pass_rate_critical` is documented in `docs/gcl-spec.md`
- Default threshold values: `pass_rate_critical=0.70`, `pass_rate_warn=0.85`

Exit codes: `0` pass, `1` fail.

### `hwcloud-skillcheck check audit-results --root <dir>`

Validates the `audit-results/` directory protection contract:
- `.gitignore` contains: `audit-results/`, `**/audit-results/`, `**/gcl-trace-*.json`, `**/gcl-quality-summary-*.json`, `**/gcl-alarm-plan-*.json`
- `audit-results/` directory has mode `0700` (owner-only)
- No tracked git files inside `audit-results/`
- `docs/gcl-spec.md` contains fragments: `audit-results/`, `GCL`, `gitignore`

Exit codes: `0` pass, `1` fail.

### `hwcloud-skillcheck validate schema <kind> --file <path>`

Validates a JSON instance against one of the embedded schemas. `kind` is one of `trace`, `summary`, `alarm-plan`, or `eval-queries`. Use `-` for `--file` to read the instance from stdin. The `eval-queries` kind auto-detects array vs object format and dispatches to the matching `$def`.

| Flag | Description |
|------|-------------|
| `--file <path>` | Instance JSON file path (`-` for stdin, required) |

```
hwcloud-skillcheck validate schema trace       --file audit-results/gcl-trace-20260727-120000.json
hwcloud-skillcheck validate schema eval-queries --file huaweicloud-ecs-ops/assets/eval_queries.json
```

Exit codes: `0` valid, `1` schema errors, `2` parse error.

### `hwcloud-skillcheck validate frontmatter --root <dir>`

Validates the YAML frontmatter of every `SKILL.md` under `--root`. Checks required top-level keys (`name`, `description`, `compatibility`, `license`), `metadata.version`, `metadata.last_updated`, `cli_applicability` enum (`dual-path|cli-first|cli-only|sdk-only`), and `name == <skill-dir>` consistency.

| Flag | Description |
|------|-------------|
| `--root <dir>` | Skill repository root (default `.`) |

Exit codes: `0` pass, `1` fail.

### `hwcloud-skillcheck validate eval-queries --root <dir>`

Validates `assets/eval_queries.json` for each skill under `--root`. Each file is parsed and dispatched to one of four shape-specific `$def`s (`activateArrayEntry`, `matchArrayEntry`, `triggerArrayEntry`, `smokeArrayEntry`, `structuredObject`, `matchObject`, `triggerObject`) based on its top-level keys.

| Flag | Description |
|------|-------------|
| `--root <dir>` | Skill repository root (default `.`) |

Exit codes: `0` pass, `1` fail.

### `hwcloud-skillcheck validate product-assessment --root <dir>`

Validates the Worker Output Contract JSON example embedded in every `references/well-architected-assessment.md` under `--root`. Checks for the `Worker Output Contract` heading, presence of a fenced JSON block, required top-level fields (`skill_id`, `product`, `region`, `scope`, `assessment_date`, `status`, `partial`, `resource_count`, `pillars`, `recommendations`, `trace`, `errors`), `status` enum (`OK|PARTIAL|ERROR`), pillar keys (`reliability|security|cost|efficiency`), per-pillar status enum (`assessed|not_assessed|skipped`), `skill_id == <skill-dir>`, and that `trace.commands` does not carry an unmasked secret reference.

| Flag | Description |
|------|-------------|
| `--root <dir>` | Skill repository root (default `.`) |

Exit codes: `0` pass, `1` fail.

## `hwcloud-skillcheck gcl` Subcommands

### `hwcloud-skillcheck gcl run --root <dir> [--json] [--quiet]`

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

### `hwcloud-skillcheck gcl alarm-wire --root <dir> [--json] [--plan-file <path>]`

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

## `hwcloud-skillcheck check` Subcommands

### `hwcloud-skillcheck check example-config --root <dir>`

Validates `assets/example-config.yaml` for every `huaweicloud-*-ops/` skill under `--root`. Checks each file for plaintext secret literals, well-formed `{{env.*}}` / `{{user.*}}` / `{{output.*}}` placeholders (no bare `{x}` tokens), basic YAML structure, and YAML anchor references that follow their definitions. Emits a soft warning when a key is repeated 3+ times without any anchors.

| Flag | Description |
|------|-------------|
| `--root <dir>` | Skill repository root (default `.`) |
| `--warn-only` | Treat failures as warnings (exit `0`) |
| `--json` | Emit JSON report |

Exit codes: `0` pass or warnings only, `1` any file has errors.

### `hwcloud-skillcheck check markdown-links --root <dir>`

Walks every Markdown file under `--root` and verifies that local relative links resolve to an existing file. Does not check `http(s)://` URLs or fragment-only anchors. This is the gate used by `go test ./...` and CI.

| Flag | Description |
|------|-------------|
| `--root <dir>` | Skill repository root (default `.`) |

Exit codes: `0` pass, `1` broken link(s).

### `hwcloud-skillcheck check references-links --root <dir>`

Validates `references/` anchor health for every skill under `--root`: every Markdown link target whose fragment is non-empty must point to a heading that exists in the linked file. Detects both stale anchors and missing targets.

| Flag | Description |
|------|-------------|
| `--root <dir>` | Skill repository root (default `.`) |
| `--warnings-only` | Demote warnings to non-fatal |
| `--json` | Emit JSON report |

Exit codes: `0` pass, `1` broken anchor(s).

### `hwcloud-skillcheck check advanced-coverage --root <dir>`

Validates TE-7 advanced-section coverage: every `huaweicloud-*-ops/SKILL.md` under `--root` must contain a section matching the advanced-coverage rubric (the `advanced/` directory plus an SKILL.md subsection that links to it). Use `--warn-only` during gradual rollouts when a skill has not yet been migrated.

| Flag | Description |
|------|-------------|
| `--root <dir>` | Skill repository root (default `.`) |
| `--warn-only` | Demote missing `advanced/` to warnings |
| `--json` | Emit JSON report |

Exit codes: `0` pass, `1` any skill missing the advanced section.

## `hwcloud-skillcheck scan` Subcommands

### `hwcloud-skillcheck scan secret <kind> --root <dir>`

Scans GCL artifacts under `--root/audit-results/` for credential leaks (AWS keys, AK/SK, bearer tokens, private-key blocks, etc.). `kind` is one of `trace`, `summary`, or `alarm-plan`, which selects the matching glob (`gcl-trace-*.json`, `gcl-quality-summary-*.json`, `gcl-alarm-plan-*.json`). Pass explicit file paths as positional arguments after the kind to scan a specific artifact instead of the glob. With no artifacts and no explicit input, the command returns ok (allow-empty behavior).

| Flag | Description |
|------|-------------|
| `--root <dir>` | Skill repository root (default `.`) |
| `--latest` | When no explicit input, scan only the lexicographically latest artifact |
| `--include-fixture` | Append the healthy fixture (`scripts/fixtures/gcl-{quality-summary,alarm-plan}-healthy.json`) to the scan set for CI smoke |
| `--self-check` | Scan embedded fixtures instead of the repo; verifies the binary itself is leak-free |
| `--json` | Emit JSON report |

```
hwcloud-skillcheck scan secret trace       --root . --latest
hwcloud-skillcheck scan secret summary    --root . --include-fixture --json
hwcloud-skillcheck scan secret alarm-plan --self-check
```

Exit codes: `0` no leaks (or no artifacts), `1` at least one artifact leaked secrets.

## `hwcloud-skillcheck aggregate` Subcommands

### `hwcloud-skillcheck aggregate trace --root <dir>`

Aggregates every `audit-results/gcl-trace-*.json` under `--root` into a quality summary JSON with totals (per `final.status` bucket and overall `pass_rate`), per-dimension average rubric scores (`correctness`, `safety`, `idempotency`, `traceability`, `spec_compliance`), per-skill buckets, and the list of source trace paths. Parsing is fan-out across `runtime.NumCPU()` workers. With no traces found, exits `0` and emits a WARN by default; pass `--require-traces` to fail the gate (and fall back to the embedded healthy fixture so the parsing path still runs end-to-end). `--self-check` aggregates the embedded healthy trace fixture without needing repo files.

| Flag | Description |
|------|-------------|
| `--root <dir>` | Skill repository root (default `.`) |
| `--since-hours <N>` | Only include traces modified within the last `N` hours (`-1` disables, default) |
| `--output <file>` | Write summary JSON to `file` instead of stdout (relative paths resolve against `--root`) |
| `--require-traces` | Fail (exit `1`) instead of WARN when no traces exist; also falls back to the embedded fixture |
| `--self-check` | Aggregate the embedded trace fixture instead of the repo |

Exit codes: `0` summary produced (or WARN with no traces), `1` `--require-traces` failed or self-check sanity check failed.

## `hwcloud-skillcheck lint` Subcommands

### `hwcloud-skillcheck lint go --root <dir>`

Runs `gofmt -l` and `go vet ./...` against the Go module at `--root`. Pass `--fix` to also invoke `gofmt -w` and rewrite any files that need formatting. The `go` and `gofmt` binaries must be on `PATH`; this is an opt-in command that never blocks the A-class checks.

| Flag | Description |
|------|-------------|
| `--root <dir>` | Go module root (default `.`) |
| `--fix` | Rewrite files with `gofmt -w` after listing offenders |
| `--quiet` | Suppress the per-run banner; only report failures |

Exit codes: `0` clean, `1` one or more `gofmt` / `go vet` issues.

## `hwcloud-skillcheck learning` Subcommands

The `learning` family rebuilds per-skill `failure_patterns.json` and the generator `common-pitfalls.md` from GCL trace history. The default target skills are the top-frequency ones (RDS / VPC / ELB / CCE).

### `hwcloud-skillcheck learning gen --root <dir>`

Regenerates `failure_patterns.json` plus `remediation-playbooks.json` for every top-frequency skill under `--root`. Idempotent: re-running produces the same artifact bytes when no input traces have changed.

| Flag | Description |
|------|-------------|
| `--root <dir>` | Repo root (default `.`) |

Exit codes: `0` artifacts written, `1` generation error.

### `hwcloud-skillcheck learning trace aggregate --root <dir> --skill <id>`

Scans `audit-results/gcl-trace-*.json` for `--skill`, extracts failure patterns, deduplicates against the existing `failure_patterns.json`, and writes the merged result. Use `--dry-run` to preview without writing.

| Flag | Description |
|------|-------------|
| `--root <dir>` | Repo root (default `.`) |
| `--skill <id>` | Skill id (e.g. `huaweicloud-ecs-ops`, required) |
| `--since-hours <N>` | Only include traces newer than `N` hours (`0` = all, default) |
| `--dry-run` | Print counts and would-be writes; do not modify files |

### `hwcloud-skillcheck learning trace learn --root <dir> --skill <id> --trace <path>`

Loads a single `gcl-trace-*.json`, extracts its failure pattern (no-op for PASS traces), and merges it into the per-skill `failure_patterns.json`. If the `(category, error_message_regex, command_pattern)` triple already exists, increments counters and timestamps; otherwise appends a new entry with the next sequential `id`. Use `--dry-run` to preview.

| Flag | Description |
|------|-------------|
| `--root <dir>` | Repo root (default `.`) |
| `--skill <id>` | Skill id (required) |
| `--trace <path>` | Path to `gcl-trace-*.json` (relative paths resolve against `--root`, required) |
| `--dry-run` | Print action without writing |

### `hwcloud-skillcheck learning trace report --root <dir> --skill <id>`

Prints a human-readable summary of the failure knowledge base for `--skill`: total pattern count, last aggregation timestamp, source-trace count, and a per-category breakdown. Pass `--json` to emit the full structured record (skill + patterns + meta) on stdout.

| Flag | Description |
|------|-------------|
| `--root <dir>` | Repo root (default `.`) |
| `--skill <id>` | Skill id (required) |
| `--json` | Emit JSON instead of human-readable text |

```
hwcloud-skillcheck learning gen                                   --root .
hwcloud-skillcheck learning trace aggregate --root . --skill huaweicloud-rds-ops --since-hours 24
hwcloud-skillcheck learning trace learn     --root . --skill huaweicloud-rds-ops --trace audit-results/gcl-trace-20260727-120000.json
hwcloud-skillcheck learning trace report    --root . --skill huaweicloud-rds-ops --json
```

Exit codes: `0` ok, `1` missing required flags or file errors.
