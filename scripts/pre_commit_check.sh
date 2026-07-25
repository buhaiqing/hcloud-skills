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
#   Shared logic lives in validate_local.py and is tested there.
# ============================================================

set -euo pipefail

SKIP_TESTS=0
if [[ "${1:-}" == "--skip-tests" ]]; then
  SKIP_TESTS=1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

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

# ── 1. ruff lint (all Python files) ────────────────────────
run_gate "ruff lint" ruff check .

# ── 2. Python 3.10 syntax compatibility ───────────────────
# Inline check (check_py310_compat.py may not exist in all branches).
# Covers both parse-level syntax (py_compile) and import-level 3.11+ names.
run_gate "py310 compat (py_compile)" python3 -m py_compile "$ROOT"/scripts/*.py

# ── 3. validate_local.py — GCL Tier-A, skillcheck, markdown links, drift guard ─
run_gate "validate_local.py" python3 "$ROOT"/scripts/validate_local.py --root "$ROOT"

# ── 4. Generator GCL contract ────────────────────────────
if [[ -f "$ROOT/scripts/check_generator_contract.py" ]]; then
  run_gate "generator GCL contract" python3 "$ROOT"/scripts/check_generator_contract.py
fi

# ── 5. references/ deep link health ──────────────────────
if [[ -f "$ROOT/scripts/check_references_link_health.py" ]]; then
  run_gate "references/ deep link health" python3 "$ROOT"/scripts/check_references_link_health.py
fi

# ── 6. eval_queries.json schema ───────────────────────────
if [[ -f "$ROOT/scripts/validate_eval_queries_schema.py" ]]; then
  run_gate "eval_queries.json schema" python3 "$ROOT"/scripts/validate_eval_queries_schema.py
fi

# ── 7. SKILL.md frontmatter ───────────────────────────────
if [[ -f "$ROOT/scripts/validate_skills_frontmatter.py" ]]; then
  run_gate "SKILL.md frontmatter" python3 "$ROOT"/scripts/validate_skills_frontmatter.py
fi

# ── 8. Well-Architected worker JSON ───────────────────────
if [[ -f "$ROOT/scripts/validate_product_assessment.py" ]]; then
  run_gate "Well-Architected worker JSON" python3 "$ROOT"/scripts/validate_product_assessment.py
fi

# ── 9. example-config.yaml anchors ─────────────────────────
if [[ -f "$ROOT/scripts/check_example_config.py" ]]; then
  run_gate "example-config.yaml anchors" python3 "$ROOT"/scripts/check_example_config.py --warn-only
fi

# ── 10. references/advanced coverage (TE-7) ─────────────
if [[ -f "$ROOT/scripts/check_advanced_coverage.py" ]]; then
  run_gate "references/advanced coverage (TE-7)" python3 "$ROOT"/scripts/check_advanced_coverage.py
fi

# ── 11. audit-results gitignore guard ─────────────────────
if [[ -f "$ROOT/scripts/check_audit_results_guard.py" ]]; then
  run_gate "audit-results gitignore guard" python3 "$ROOT"/scripts/check_audit_results_guard.py
fi

# ── 12. gcl_quality wiring contract ───────────────────────
if [[ -f "$ROOT/scripts/check_gcl_alarm_wire_contract.py" ]]; then
  run_gate "gcl_quality wiring contract" python3 "$ROOT"/scripts/check_gcl_alarm_wire_contract.py
fi

# ── 13. GCL Tier-A conformance ───────────────────────────
if [[ -f "$ROOT/scripts/check_gcl_conformance.py" ]]; then
  run_gate "GCL Tier-A conformance" python3 "$ROOT"/scripts/check_gcl_conformance.py
fi

# ── 14. GCL trace schema ──────────────────────────────────
if [[ -f "$ROOT/scripts/validate_gcl_trace_schema.py" ]]; then
  run_gate "GCL trace schema" python3 "$ROOT"/scripts/validate_gcl_trace_schema.py --latest
fi

# ── 15. GCL trace security ────────────────────────────────
if [[ -f "$ROOT/scripts/check_gcl_trace_security.py" ]]; then
  run_gate "GCL trace security" python3 "$ROOT"/scripts/check_gcl_trace_security.py --latest
fi

# ── 16. GCL quality summary schema ────────────────────────
if [[ -f "$ROOT/scripts/validate_gcl_summary_schema.py" ]]; then
  run_gate "GCL quality summary schema" python3 "$ROOT"/scripts/validate_gcl_summary_schema.py
fi

# ── 17. GCL quality summary security ──────────────────────
if [[ -f "$ROOT/scripts/check_gcl_summary_security.py" ]]; then
  run_gate "GCL quality summary security" python3 "$ROOT"/scripts/check_gcl_summary_security.py --include-fixture
fi

# ── 18. safety_class enum contract ───────────────────────
if [[ -f "$ROOT/scripts/check_safety_class_enum.py" ]]; then
  run_gate "safety_class enum contract" python3 "$ROOT"/scripts/check_safety_class_enum.py
fi

# ── 19. resource_scope PII contract ───────────────────────
if [[ -f "$ROOT/scripts/check_resource_scope_pii.py" ]]; then
  run_gate "resource_scope PII contract" python3 "$ROOT"/scripts/check_resource_scope_pii.py
fi

# ── 20. skill_generator drift guard (standalone) ─────────
if [[ -f "$ROOT/scripts/check_skill_generator_drift.py" ]]; then
  run_gate "skill_generator drift guard" python3 "$ROOT"/scripts/check_skill_generator_drift.py check
fi

# ── 21. GCL alarm plan schema ─────────────────────────────
if [[ -f "$ROOT/scripts/validate_gcl_alarm_plan_schema.py" ]]; then
  run_gate "GCL alarm plan schema" python3 "$ROOT"/scripts/validate_gcl_alarm_plan_schema.py --include-fixture
fi

# ── 22. GCL alarm plan security ───────────────────────────
if [[ -f "$ROOT/scripts/check_gcl_alarm_plan_security.py" ]]; then
  run_gate "GCL alarm plan security" python3 "$ROOT"/scripts/check_gcl_alarm_plan_security.py --include-fixture
fi

# ── 23. Unit tests (skipped in pre-commit hook) ──────────
if (( SKIP_TESTS == 0 )); then
  if [[ -f "$ROOT/scripts/gcl_structural_critic_test.py" ]]; then
    run_gate "GCL structural-critic unit tests" python3 -m unittest "$ROOT"/scripts/gcl_structural_critic_test -v
  fi

  if [[ -f "$ROOT/scripts/check_py310_compat.py" ]]; then
    run_gate "check_py310_compat.py (import dry-run)" python3 "$ROOT"/scripts/check_py310_compat.py
  fi
fi

# ── Summary ───────────────────────────────────────────────
echo ""
if (( FAILED == 0 )); then
  echo "OK: all pre-commit gates passed"
else
  echo "FAIL: one or more gates failed — see above" >&2
fi

exit $FAILED
