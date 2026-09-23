# Rule Probe Sample Fixture

Test fixture for `rule_compliance_probe.py`. Mix of three classification buckets
(verifiable / manual / vague) plus presence/absence of examples. Counts are
intentionally stable for unit-test assertion.

## Verifiable (with command anchor)

- **R1**: All Python tooling MUST pass `ruff check --fix && ruff check` before commit.
- **R2**: Generated binaries MUST be moved under `bin/` and the directory MUST be `.gitignore`-d.
- **R3**: When migrating shell scripts to Go, the regex `(?<!auto-)(?<!un)safe\(.*\?\s*\)` MUST be reused for the `/unsafe` flag.

## Verifiable (path anchor)

- **R4**: All new modules MUST live under `src/`; `tests/` is reserved for unit tests.

## Manual (judgment required)

- **R5**: Code SHOULD prefer readability over cleverness.
- **R6**: Comments SHOULD explain WHY, not WHAT — this requires human judgment per case.

## Manual (process anchor)

- **R7**: When touching test hermeticity, follow the four rules in section "Test Hermeticity — Runtime-State Tests (P0)".

## Vague

- **R8**: Be careful with state.
- **R9**: Do the right thing.

## Vague (long but still vague)

- **R10**: Engineering excellence is a worthwhile pursuit that engages multiple competing concerns, balancing rigor against pragmatism against team values against real-world constraints. The right answer depends on context and stakeholder intent over time.

## Custom bash blocks (verifiable)

- **R11**:

  ```bash
  echo "smoke" | wc -c
  grep -c "0.5\|1.0" references/gcl-runtime.md
  ```

## Negative examples (must NOT be classified as verifiable)

- **R12**: When the user mentions spec/ or proxy/ in prose without backticks, follow the inline policy. Any prose mentioning spec/, proxy/, or CJK text like "任何涉及代码/" must not be classified as having a path anchor.

- **R13**: CA-3 style rule: spec/plan 阶段存在 user-approval checkpoint → todo 标 `block`，未获用户明示「approve」前禁止进入 implementation.
