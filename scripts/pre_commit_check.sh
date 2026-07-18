#!/usr/bin/env bash
# Repository-wide Python pre-commit gate.
#
# Runs (in order):
#   1. ruff check  (style/lint)
#   2. ruff format --check  (formatting drift detection; do NOT auto-fix)
#   3. check_py310_compat.py  (Python 3.10 syntax compatibility)
#   4. Python unit tests (scripts/*_test.py)
#
# Designed to be invoked from `.githooks/pre-commit` and from CI so local
# and remote behavior stay in lockstep. Exits non-zero on the first failure
# so the commit is aborted before any data is written.
#
# Usage:
#   bash scripts/pre_commit_check.sh                # full gate
#   bash scripts/pre_commit_check.sh --skip-tests    # skip unit tests
#   bash scripts/pre_commit_check.sh --staged-only  # only lint staged Python files

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "${REPO_ROOT}"

SKIP_TESTS=0
STAGED_ONLY=0
for arg in "$@"; do
  case "${arg}" in
    --skip-tests) SKIP_TESTS=1 ;;
    --staged-only) STAGED_ONLY=1 ;;
    -h|--help)
      sed -n '2,18p' "${0}"
      exit 0
      ;;
    *)
      echo "[pre_commit_check] unknown argument: ${arg}" >&2
      exit 2
      ;;
  esac
done

log() { printf "\n[pre_commit_check] %s\n" "$1"; }
fail() { printf "\n[pre_commit_check] FAIL: %s\n" "$1" >&2; exit 1; }

TARGETS=("scripts")
if [ "${STAGED_ONLY}" -eq 1 ]; then
  STAGED_PY=()
  while IFS= read -r staged; do
    STAGED_PY+=("${staged}")
  done < <(git diff --cached --name-only --diff-filter=AM -- 'scripts/*.py' || true)
  if [ "${#STAGED_PY[@]}" -eq 0 ]; then
    log "no staged Python files under scripts/; skipping Python gates"
    exit 0
  fi
  TARGETS=("${STAGED_PY[@]}")
fi

log "step 1/3: ruff check ${TARGETS[*]}"
if ! bash scripts/run_ruff.sh "${TARGETS[@]}"; then
  fail "ruff check reported issues; fix them before committing"
fi

log "step 2/3: ruff format --check ${TARGETS[*]}"
if command -v ruff >/dev/null 2>&1; then
  if ! ruff format --check "${TARGETS[@]}"; then
    fail "ruff format drift detected; run 'ruff format ${TARGETS[*]}' locally and re-commit"
  fi
else
  echo "[pre_commit_check] ruff not on PATH; skipping format check"
fi

log "step 3/3: Python 3.10 syntax compatibility"
if [ -f scripts/check_py310_compat.py ]; then
  if ! python3 scripts/check_py310_compat.py; then
    fail "python3.10 compatibility gate failed"
  fi
else
  echo "[pre_commit_check] scripts/check_py310_compat.py not found; skipping"
fi

if [ "${SKIP_TESTS}" -eq 1 ]; then
  log "skipping unit tests (--skip-tests)"
else
  log "step 4/4: Python unit tests"
  if [ "$(find scripts -name '*_test.py' 2>/dev/null | wc -l)" -gt 0 ]; then
    if ! python3 -m unittest discover -s scripts -p "*_test.py"; then
      fail "unit tests failed"
    fi
  else
    echo "[pre_commit_check] no *_test.py files found; skipping"
  fi
fi

log "OK: all Python gates passed"
