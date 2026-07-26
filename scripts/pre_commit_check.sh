#!/usr/bin/env bash
# ============================================================
# hcloud-skills pre-commit gate — "single shot gun"
#
# Runs all checks that CI performs locally (except Go tests).
# Any non-zero exit aborts the hook / CI step.
#
# Flags:
#   --skip-tests   Skip slow unit-test / GCL integration steps
#                  (useful for git hook; CI calls without this flag
#                  to get full coverage)
#
# Design principle:
#   - git hook  → --skip-tests (fast, non-destructive)
#   - CI        → no flag (full suite)
#
# Token-efficiency note (TE-6):
#   This script intentionally does NOT import shared helpers from
#   any external file. Each gate is a self-contained subprocess call
#   into the hwcloud-skillcheck Go binary (which is tested in its own
#   test suite). As of 2026-07-26 there are zero Python scripts in
#   scripts/, so no Python-toolchain gates (ruff, py_compile) are
#   needed — the binary is the single source of truth.
# ============================================================

set -euo pipefail

SKIP_TESTS=0
if [[ "${1:-}" == "--skip-tests" ]]; then
  SKIP_TESTS=1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Build hwcloud-skillcheck if missing.
SKILLCHECK_BIN="${SKILLCHECK_BIN:-$ROOT/bin/hwcloud-skillcheck}"
if [[ ! -x "$SKILLCHECK_BIN" ]]; then
  echo "==> building hwcloud-skillcheck"
  (cd "$ROOT/hwcloud-skillcheck" && go build -trimpath -o "$SKILLCHECK_BIN" .)
fi

# Track overall result; exit 1 on any gate failure.
FAILED=0

run_gate() {
  local label="$1"
  shift
  echo "==> [gate] $label"
  if ! "$@"; then
    echo "FAIL: $label" >&2
    FAILED=1
  fi
}

# ── 1. Go: fmt, vet (binary is built conditionally above; running `go build`
#    here again would just rebuild the same artifact and double wall-clock cost) ──
run_gate "gofmt" bash -c "cd $ROOT/hwcloud-skillcheck && [ -z \"\$(gofmt -l .)\" ]"
run_gate "go vet" bash -c "cd $ROOT/hwcloud-skillcheck && go vet ./..."

# ── 2. Go: A-class total entry (already runs validate frontmatter, eval-queries,
#    product-assessment, example-config, markdown-links, references-links, and
#    advanced-coverage). The four `check X` calls below used to repeat that
#    work and double subprocess startup cost. Keep `audit-results` as a
#    dedicated gate because the validate total-entry does NOT cover it. ──
run_gate "hwcloud-skillcheck validate" "$SKILLCHECK_BIN" validate --root "$ROOT"
run_gate "hwcloud-skillcheck check audit-results" "$SKILLCHECK_BIN" check audit-results --root "$ROOT"

# ── 4. Go: GCL surface ──
run_gate "hwcloud-skillcheck aggregate trace" "$SKILLCHECK_BIN" aggregate trace --require-traces --root "$ROOT"

# ── 5. Go: learning + l4 subcommands ──
run_gate "hwcloud-skillcheck learning gen" "$SKILLCHECK_BIN" learning gen --root "$ROOT"
run_gate "hwcloud-skillcheck l4 handle smoke" "$SKILLCHECK_BIN" l4 handle --fault "smoke" --risk low --root "$ROOT"

# ── 6. Go: skill_generator drift guard (sync + check; sync is self-healing) ──
run_gate "skill_generator drift guard" bash -c "\"$SKILLCHECK_BIN\" drift sync --apply --root \"$ROOT\" && \"$SKILLCHECK_BIN\" drift check --root \"$ROOT\""

# ── 6. Unit tests (skipped in pre-commit hook) ──────────
if (( SKIP_TESTS == 0 )); then
  run_gate "Go test" bash -c "cd $ROOT/hwcloud-skillcheck && go test ./... -count=1"
fi

# ── Summary ───────────────────────────────────────────────
echo ""
if (( FAILED == 0 )); then
  echo "All pre-commit gates passed."
else
  echo "One or more pre-commit gates FAILED."
fi

exit $FAILED
