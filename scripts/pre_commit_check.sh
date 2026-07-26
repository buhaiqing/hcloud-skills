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
#   Python scripts. Each gate is a self-contained subprocess call.
#   Shared logic lives in the hwcloud-skillcheck Go binary and is tested there.
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

# ── 1. ruff lint (no-op when no Python files) ───────────
if ls "$ROOT"/scripts/*.py >/dev/null 2>&1; then
  run_gate "ruff lint" ruff check scripts/
fi

# ── 2. Python 3.10 syntax compatibility (no-op when no Python files) ──
if ls "$ROOT"/scripts/*.py >/dev/null 2>&1; then
  run_gate "py310 compat (py_compile)" python3 -m py_compile "$ROOT"/scripts/*.py
fi

# ── 3. Go: build, fmt, vet ────────────────────────────────
run_gate "hwcloud-skillcheck build" bash -c "cd $ROOT/hwcloud-skillcheck && go build -trimpath -o $SKILLCHECK_BIN ."
run_gate "gofmt" bash -c "cd $ROOT/hwcloud-skillcheck && [ -z \"\$(gofmt -l .)\" ]"
run_gate "go vet" bash -c "cd $ROOT/hwcloud-skillcheck && go vet ./..."

# ── 4. Go: A-class total entry (replaces validate_local.py) ──
run_gate "hwcloud-skillcheck validate" "$SKILLCHECK_BIN" validate --root "$ROOT"

# ── 5. Go: per-check subcommands ───────────────────────────
run_gate "hwcloud-skillcheck check markdown-links"   "$SKILLCHECK_BIN" check markdown-links --root "$ROOT"
run_gate "hwcloud-skillcheck check references-links" "$SKILLCHECK_BIN" check references-links --root "$ROOT"
run_gate "hwcloud-skillcheck check example-config"    "$SKILLCHECK_BIN" check example-config --root "$ROOT"
run_gate "hwcloud-skillcheck check advanced-coverage" "$SKILLCHECK_BIN" check advanced-coverage --root "$ROOT"
run_gate "hwcloud-skillcheck check audit-results"     "$SKILLCHECK_BIN" check audit-results --root "$ROOT"

# ── 6. Go: GCL surface ──
run_gate "hwcloud-skillcheck aggregate trace" "$SKILLCHECK_BIN" aggregate trace --root "$ROOT"

# ── 7. Go: new learning + l4 subcommands (replaces Python counterparts) ──
run_gate "hwcloud-skillcheck learning gen" "$SKILLCHECK_BIN" learning gen --root "$ROOT"
run_gate "hwcloud-skillcheck l4 handle smoke" "$SKILLCHECK_BIN" l4 handle --fault "smoke" --risk low --root "$ROOT"

# ── 8. Go: skill_generator drift guard (sync + check; sync is self-healing) ──
run_gate "skill_generator drift guard" bash -c "\"$SKILLCHECK_BIN\" drift sync --apply --root \"$ROOT\" && \"$SKILLCHECK_BIN\" drift check --root \"$ROOT\""

# ── 9. Unit tests (skipped in pre-commit hook) ──────────
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
