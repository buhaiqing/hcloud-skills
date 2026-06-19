#!/usr/bin/env bash
# Run ruff locally with the version pinned in CI (.github/workflows/validate-skills.yml).
# Usage:
#   scripts/run_ruff.sh           # lint the whole repo
#   scripts/run_ruff.sh scripts/  # lint a subpath (e.g. just edited scripts)
#
# Exits non-zero on any finding so it can be wired into pre-commit / validate_local.py.

set -euo pipefail

RUFF_VERSION="${RUFF_VERSION:-0.11.8}"
TARGET="${1:-.}"

if command -v ruff >/dev/null 2>&1; then
  INSTALLED_VERSION="$(ruff --version 2>/dev/null | awk '{print $2}')"
  if [ "${INSTALLED_VERSION}" != "${RUFF_VERSION}" ]; then
    echo "[run_ruff] WARNING: installed ruff ${INSTALLED_VERSION} differs from CI pinned ${RUFF_VERSION}." >&2
    echo "[run_ruff]          Consider: pipx install --force ruff==${RUFF_VERSION}" >&2
  fi
  exec ruff check "${TARGET}"
fi

if command -v pipx >/dev/null 2>&1; then
  echo "[run_ruff] ruff not on PATH; invoking pipx ruff==${RUFF_VERSION} ..." >&2
  exec pipx run --spec "ruff==${RUFF_VERSION}" ruff check "${TARGET}"
fi

echo "[run_ruff] ruff is not installed." >&2
echo "[run_ruff] Install with: pipx install ruff==${RUFF_VERSION}" >&2
exit 127