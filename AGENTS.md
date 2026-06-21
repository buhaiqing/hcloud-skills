# AGENTS.md — hcloud-skills

## What This Repo Is

Huawei Cloud Ops Skill collection — structured agent runbooks (`huaweicloud-[product]-ops`) executed via `hcloud` CLI (primary) with Go SDK JIT fallback. Not application code; no build/test/lint step.

## Skill Directory Layout (Convention)

Every skill follows this structure — do not deviate:

```
huaweicloud-[product]-ops/
├── SKILL.md              # Main runbook: frontmatter, triggers, operations, recovery
├── references/           # Deep reference files (core-concepts, api-sdk-usage, cli-usage, troubleshooting, monitoring, integration, well-architected-assessment, etc.)
└── assets/               # eval_queries.json + example-config.yaml
```

**SKILL.md is the entry point.** References provide depth. No duplication between them.

## Generator / Meta-Skill

`huaweicloud-skill-generator` scaffolds new skills from OpenAPI specs. Load the `huaweicloud-skill-generator` skill when creating or updating any `huaweicloud-*-ops`. It enforces P0/P1 quality gates, the Five Core Standards, and three-pillar integration.

Template: `huaweicloud-skill-generator/references/huaweicloud-skill-template.md`

## ⚠️ Dual-Copy Trap

The generator exists in **two places**:

- `huaweicloud-skill-generator/` (root — canonical, tracked by git)
- `.agents/skills/huaweicloud-skill-generator/` (loaded by agent runtime — gitignored)

When editing the generator, update the **root copy** only. The runtime copy
MUST be brought back in sync via:

```bash
python3 scripts/check_skill_generator_drift.py sync --apply
```

The drift guard (`scripts/check_skill_generator_drift.py check`) is wired into
`scripts/validate_local.py` and the CI workflow, so a drifted runtime copy is
a release-blocker. See also `docs/gcl-spec.md` §Dual-Copy Drift.

## Placeholder Conventions

| Placeholder | Source | Rule |
|-------------|--------|------|
| `{{env.*}}` | Runtime environment | **Never** ask user; fail if unset |
| `{{user.*}}` | User input | Collect interactively |
| `{{output.*}}` | API response capture | Chain into subsequent steps |

## Execution Paths

- **Primary**: `hcloud` CLI — always prefer when CLI supports the operation
- **Fallback**: Go SDK (`github.com/huaweicloud/huaweicloud-sdk-go-v3`) via JIT `go run` — for unsupported CLI operations
- `cli_applicability` field in SKILL.md frontmatter: `cli-first` | `dual-path` | `sdk-only` | `cli-only`

## Three-Pillar Integration (Mandatory)

Every skill MUST embed FinOps + SecOps + AIOps. No exceptions:

- **FinOps**: Billing model comparison, idle resource detection, right-sizing, budget alerts
- **SecOps**: IAM least-privilege table, credential masking (`***`), network isolation, encryption
- **AIOps**: ≥4 anomaly patterns, cross-skill delegation matrix, fault knowledge base, alarm storm handling

## Quality Gates

### P0 (Must Pass)
- SHOULD/SHOULD NOT trigger conditions complete
- Pre-flight → Execute → Validate → Recover flow for each operation
- ≥10 product error codes with recovery strategies
- Destructive operations have safety gates (explicit confirmation)
- `assets/eval_queries.json` with should/should-not trigger queries

### P1 (Should Pass)
- Idempotency documented where automation applies
- Cross-skill delegation matrix in `integration.md`
- Adversarial scenarios considered
- Self-reflection completed

## Token Efficiency Requirements (P0 — 强制)

> 在保持 Agent 可执行性的前提下，最小化每个 Skill 的 Token 消耗。

| 规则 | 要点 | 节省 |
|------|------|------|
| **TE-1** API 查询 > 静态表格 | 用 `hcloud` 命令获取版本/配额，不硬编码 | ~200-500/文件 |
| **TE-2** 省略不必要的 docstring | Go SDK 用 `#` 注释代替函数级 docstring | ~100-200/函数 |
| **TE-3** 紧凑错误表 | 每行 1 个错误码，≤3 列 | ~300-500/文件 |
| **TE-4** JSON paths 集中声明 | 文件顶部统一声明，不重复 | ~50-100/文件 |
| **TE-5** YAML anchors | `example-config.yaml` 用 `&anchor` 消除重复 | ~200-400/文件 |
| **TE-6** 消除跨文件重复 | SKILL.md 已有完整流程，references 不重复 | 因 Skill 而异 |
| **TE-7** 专业内容分层 | AIOps/FinOps 等深度分析放 `references/advanced/`；安全敏感操作单独标注并要求显式确认 | ~3,000-8,000/文件 |

**不可压缩的内容**：Agent 可执行命令本身（参数、JSON paths）、错误恢复逻辑、安全门、Credential 规则、跨技能编排链。

## Skill Update Rule: 2-Round Self-Reflection

**After every skill update or creation, execute 2 mandatory self-reflection rounds and auto-fix all discovered issues before finishing.**

### Round 1 — Foundation Check
1. **FinOps**: Are cost patterns actionable? Billing model comparison present? Idle detection documented?
2. **SecOps**: IAM permissions minimum documented? Credential masking enforced? Network isolation?
3. **AIOps**: Multi-metric correlation defined? Delegation matrix present? Knowledge base populated?

#### Round 1, Item 4 — Token Efficiency (C6 — MUST PASS)

**必检项**：TE-1~TE-7 是否全部满足（见上一节 Token Efficiency Requirements）？未满足则 **BLOCK**。

| TE 规则 | 检查方法 | 不通过则 |
|---------|---------|---------|
| TE-1 | 检查 references/ 中是否有硬编码的版本号/配额数字 | 替换为 `hcloud` 查询命令 |
| TE-2 | 检查 Go SDK 代码块是否有函数级 docstring | 删除 docstring，改用 `#` 行注释 |
| TE-3 | 检查错误表是否超过 3 列 | 合并列，每行 1 个错误码 |
| TE-4 | 检查 JSON path 是否在文件顶部集中声明 | 移至文件顶部统一声明 |
| TE-5 | 检查 example-config.yaml 是否有重复字段 | 用 YAML anchors 消除 |
| TE-6 | 检查 SKILL.md 与 references/ 是否有内容重复 | 删除 references 中的重复 |
| TE-7 | 检查 AIOps/FinOps 是否在 `references/advanced/`；安全敏感操作是否标注 Security-Sensitive | 移至 `advanced/` + 添加 Security-Sensitive 标注 |

**发现任一违规 → 立即修复 → 重新检查直到全部通过。**

### Round 2 — Critical Analysis
4. **Gap Analysis**: What would break in production if a user follows this skill?
5. **Alternative Coverage**: Is there a better way that reduces agent confusion?
6. **Escalation Paths**: Are HALT conditions clear? Enough non-retryable error patterns?
7. **Cross-Pillar Synergy**: Do FinOps recommendations conflict with reliability? SecOps create performance bottlenecks?

**For any issue found: fix immediately, then re-verify.** Do not report and stop — fix and verify the fix passes.

## Python Style & Lint (P0)

- Repository linter is **ruff** (`ruff check .`, pinned to `0.11.8` in CI). Config lives in `ruff.toml`.
- **After every Python script change, run `bash scripts/run_ruff.sh .` locally before declaring the task complete.** Do not batch — fix lint findings introduced by the change immediately. Apply `ruff format .` for any formatting drift introduced by the change.
- A single shot gun covers everything: `bash scripts/pre_commit_check.sh`. This is what the git hook and CI both invoke — running it locally is equivalent to pushing.
- The git pre-commit hook lives at `.githooks/pre-commit` and is installed by `python3 scripts/install_git_hook.py`. It auto-runs only when a `scripts/*.py` file is staged or modified, so markdown-only commits stay fast. Use `--check` to see if the hook is installed, `--uninstall` to remove it.
- New scripts MUST:
  - Start with a module docstring describing purpose.
  - Avoid unused imports / unreachable code / bare `except:`.
  - Prefer `argparse` with explicit `--help` text for CLIs.
  - Keep functions short; favor pure helpers that are unit-testable.
- Shared helpers (`json_schema_subset`, `gcl_security_scan`) MUST be reused instead of copy-pasted patterns — same rule as TE-6.
- Tests live next to scripts (`scripts/*_test.py`) and are run via `python3 -m unittest discover -s scripts -p "*_test.py"`.
- CI runs the full `validate_local.py` suite; local dev MUST run the same suite before pushing.

## Python 3.10 Syntax Compatibility (P0)

- Agent runtime executes scripts on **Python 3.10**, even though CI lints them with Python 3.11. Any 3.11-only symbol silently breaks the agent.
- **Why two checks instead of one.** `py_compile` only validates parse-time
  syntax; it does NOT execute imports. The original `from datetime import UTC`
  bug shipped through CI because the syntax is valid on 3.10 — only name
  resolution fails at runtime. The gate below now does both checks under
  3.10: `py_compile` for syntax, plus an import dry-run for name resolution.
- Disallowed in `scripts/*.py` (any 3.11+-only stdlib symbol used at runtime):

  | Symbol | Why | 3.10 replacement |
  |--------|-----|------------------|
  | `from datetime import UTC` (and `datetime.UTC`) | 3.11+ alias | `from datetime import timezone; UTC = timezone.utc` with `# noqa: UP017` (see existing usage in `gcl_runner.py`) |
  | `import tomllib` | 3.11+ stdlib module | `import json` (rewrite TOML to JSON/YAML) or `pip install tomli` + `import tomli as tomllib` |
  | `typing.Self` (without `from __future__ import annotations`) | 3.11+ at runtime | `from typing import Self` (works on 3.10) |
  | PEP 695 type aliases (`type Alias = int`) | 3.12+ syntax | `Alias = int` (plain assignment) |
  | PEP 695 type parameters (`class C[T]:`, `def f[T](x: T)`) | 3.12+ syntax | `from typing import TypeVar, Generic` |
  | `datetime.timezone.utc` 3.11+ features (`datetime.GregorianCalendar`, etc.) | varies | 3.10 compatible equivalent |

  The list above is non-exhaustive; the import dry-run in
  `check_py310_compat.py` is the source of truth. Add a new entry here when
  you discover a new trap.
- All scripts MUST start with `from __future__ import annotations` so PEP 604
  / new-style generics remain *string* and are safe across 3.10 / 3.11 / 3.12.
- **Enforcement** (`scripts/check_py310_compat.py`):
  1. `python3.10 -m py_compile` on every `scripts/*.py` — syntax gate.
  2. `python3.10 -c "import importlib.util; …"` per script — **import dry-run**
     that actually loads the module so import-time 3.11+ names are caught.
  3. Both run in fresh subprocesses; module-level state never leaks.
  - Local: `python3 scripts/check_py310_compat.py` (uses the first available
    `python3.10` / `python310`).
  - CI: same command, pinned to `python-version: "3.10"`.
  - The `Python unit tests` workflow step **MUST** be pinned to 3.10 too;
    without `setup-python: "3.10"` it inherits 3.11 and silently misses
    3.10-only import errors.
  - `--no-import-check` is reserved for bisecting a gate failure; it is
    **not** a way to ship a 3.11+ symbol.
- **After every Python script change, the script MUST pass both gates under
  Python 3.10.** A regression is a release-blocker. Add a regression test
  to `check_py310_compat_test.py::ImportTests` whenever you encounter a new
  3.11+ symbol that the import dry-run catches.

## Test Hermeticity — Runtime-State Tests (P0)

- **Tests that touch the real repo (`Path(__file__).resolve().parents[1]`)
  are NOT hermetic by default.** They depend on state that exists locally
  (e.g. `audit-results/` populated by prior GCL runs,
  `.agents/skills/huaweicloud-skill-generator/` populated by the agent
  runtime) but does **not** exist on a fresh CI checkout. The two
  `test_main_repo_passes` / `test_repo_passes_after_sync` failures in CI run
  #6 are the canonical example.
- Rules for runtime-state tests:
  1. **CLI-style smoke tests** (e.g. `cag.main()`, `csgd.check_drift(ROOT)`)
     MUST tolerate the *absent* state, not just the *wrong* state. The
     audit-results guard was changed: a missing `audit-results/` directory
     is no longer a failure (runtime scripts create it on demand), only
     wrong mode or tracked files fail.
  2. **Bootstrap functions** (e.g. `sync()`) MUST self-heal — if the
     runtime copy is missing, `mkdir(parents=True, exist_ok=True)` before
     copying. Don't expect callers to pre-create the destination.
  3. **Fixture-style tests** that *do* need the runtime state (e.g.
     drift-check end-to-end) MUST use `tempfile.TemporaryDirectory()` with
     a controlled `mkdir` setup, **not** `ROOT`. Mark such tests with a
     `# REPO-ROOT-DEPENDENT` docstring so reviewers can spot them.
  4. **No silent state mutation in CI.** A test that calls
     `csgd.sync(ROOT, dry_run=False)` will leave the runtime copy populated
     in the CI workspace, polluting subsequent runs. Either guard with
     `unittest.skipUnless(Path("…").exists(), "requires runtime state")` or
     copy the populated dir into a tempdir and operate there.
- When a guard's `check_*` function reports "missing" as an error, ask:
  is the missing state something the *runtime* creates on demand? If yes,
  the guard is wrong — the contract is "guard what must already be true",
  not "guard what will be true after the first runtime call". Use the
  gitignore / mode / tracked-files checks as the hard gates; let "exists
  and is correct" be a soft expectation enforced by smoke tests in
  `validate_local.py`, not by `unittest discover` on a fresh checkout.

## Docker Sandbox

```bash
docker-compose build
docker-compose up hcloud-skills
# Inside container:
check-env          # Verify HW_* env vars
skill-list          # List all available skills
skill-read <name>   # Read a skill's SKILL.md
hc <product> <op>   # Alias for hcloud CLI
```

Services: `hcloud-skills` (interactive), `hcloud-worker` (non-interactive), `hcloud-test` (test runner, profile: test), `hcloud-sdk-builder` (Go build, profile: build).

## Environment Variables

| Variable | Required | Default |
|----------|----------|---------|
| `HW_ACCESS_KEY_ID` | Yes | — |
| `HW_SECRET_ACCESS_KEY` | Yes | — |
| `HW_REGION_ID` | No | `cn-north-4` |
| `HW_PROJECT_ID` | Service-specific | — |

## Key Anti-Patterns to Avoid

| Anti-Pattern | What to Do Instead |
|---|---|
| Inventing API fields or CLI flags | Cross-reference every field against OpenAPI or verified CLI output |
| Printing/logging real credentials | Mask with `***` / `<masked>` |
| Skipping safety gate on destructive ops | Add explicit confirmation step |
| Hardcoding regions/timeouts | Use `{{env.*}}` / `{{user.*}}` placeholders |
| One skill does everything | Single product, single resource model; delegate cross-product ops |
| SKILL.md duplicates references/ | SKILL.md = entry point; references = depth; no overlap |

## Delegation Matrix (Common Cross-Product Operations)

- ECS → VPC (subnet), CES (metrics), ELB (load balancing)
- RDS → ECS (CloudShell), CES (performance metrics)
- All products → IAM (permission issues), CTS (audit trails), BSS (billing)

## Sources of Truth

1. OpenAPI + official docs > forums/chat
2. Verified `hcloud` CLI output > assumed behavior
3. `huaweicloud-sdk-go-v3` for SDK fallback patterns
4. API docs: https://support.huaweicloud.com/api/

---

## Runtime Quality Gates: GCL

Detailed runtime-quality specifications are externalized to reduce always-loaded context size:

| Spec / Tool | Read or run before modifying |
|---|---|
| `docs/gcl-spec.md` | any `## Quality Gate (GCL)` section, `references/rubric.md`, `references/prompt-templates.md`, GCL scripts, or CES GCL monitoring wiring |
| `scripts/gcl_runner.py` | runtime Orchestrator loop; external Critic required in production |
| `scripts/gcl_trace_aggregate.py` | trace → quality summary aggregation |
| `scripts/gcl_alarm_wire.py` | CES alarm plan/apply for GCL SLOs |
| `scripts/check_gcl_conformance.py` | Tier-A artifact conformance across all 20 skills |
| `scripts/validate_local.py` | local validation suite for GCL-related gates |

### GCL hard constraints

- Production GCL requires isolated Generator and Critic contexts; shared-context G+C is banned.
- Critic is read-only: it MUST NOT call `hcloud`, use SDK clients, mutate resources, or self-score Generator output.
- Critic MUST NOT see raw user request; it may use sanitized `{{output.operation_intent}}`, Generator output, trace, and rubric.
- Orchestrator owns `operation_intent` generation before Critic scoring; it MUST omit raw user wording, credentials, and unmasked sensitive identifiers.
- `Safety = 0` / `SAFETY_FAIL` MUST abort immediately; never return partial or best-effort output.
- Every GCL loop MUST be bounded by `max_iterations`; unbounded retry loops are banned.
- Every GCL run MUST persist a masked trace under `audit-results/gcl-trace-*.json`.
- Production GCL MUST use externally supplied isolated Critic scores; `--structural-critic-only` is only for CI/local smoke tests and MUST NOT approve production or human acceptance gates.
- GCL prompt templates MUST use `{{env.*}}` / `{{user.*}}` / `{{output.*}}`; bare `{...}` placeholders are banned.
- GCL `required` / `recommended` skills MUST keep `## Quality Gate (GCL)` in `SKILL.md`, plus `references/rubric.md` and `references/prompt-templates.md`.

### Runtime scripts

```bash
python3 scripts/check_gcl_conformance.py
python3 scripts/gcl_runner.py run --skill huaweicloud-billing-ops --request "smoke" --command 'printf ok' --max-iter 1 --structural-critic-only
python3 scripts/gcl_trace_aggregate.py --since-hours 168
python3 scripts/gcl_alarm_wire.py plan --summary scripts/fixtures/gcl-quality-summary-healthy.json
python3 scripts/validate_local.py
```

### Relationship to build-time self-reflection

Build-time 2-round self-reflection and runtime GCL are independent gates. A clean self-reflection does not exempt runtime scoring; a passing GCL rubric does not exempt sloppy skill updates.

### GCL changelog

| Version | Date | Change |
|---|---|---|
| 1.0.0 | 2026-06-04 | Initial GCL specification and ECS pilot |
| 1.3.0 | 2026-06-04 | All 20 skills gained GCL artifacts |
| 1.4.0 | 2026-06-04 | CES monitoring design for GCL pass-rate |
| 1.6.0 | 2026-06-19 | qcloud-style runtime scripts, sanitized `operation_intent`, Tier-A conformance, and CES summary schema added |

### See also

- `docs/gcl-spec.md` — full runtime GCL spec
- `huaweicloud-ces-ops/assets/gcl-quality-summary.schema.json` — quality summary contract
- `huaweicloud-ces-ops/references/gcl-monitoring.md` — CES monitoring design
