# AGENTS.md — hcloud-skills

## Pre-flight Gate (每次执行前必跑)

收到任务后，**先跑以下 checklist，再动手**：

1. **Orchestrator 触发检查** — 任务是否涉及多文件 / 多阶段 / 多 skill / 用户提到「orchestrator」？→ 是则加载 `subagent-orchestrator` skill 并输出决策 JSON，再执行
2. **Skill generator 检查** — 是否在创建 / 更新 `huaweicloud-*-ops`？→ 是则加载 `huaweicloud-skill-generator` skill
3. **直接执行** — 以上均否 → 直接做

> 违反此 gate = 流程违规，即使结果正确也需复盘。

### Commit Gate（强制 — 不可跳过）

**任何 `git commit` 之前，必须确认所有单元测试通过。** 执行：

```bash
cd hwcloud-skillcheck && go test ./...
```

- exit code ≠ 0 → **禁止 commit**，先修测试
- 2 个预存已知失败（`TestConfirmationRegistry_ConcurrentSafety`、`TestHandleFault_DecisionAutoProceed`）已在 commit 中标注，后续应修复或标注 `// KNOWN-FLAKY: <reason>`
- `git commit` 本身可正常执行（不自动触发 pre-commit hook 的 go test，因为 go test gate 在 pre_commit_check.sh 中）；但 **Agent 必须自行检查**，不允许在测试 red 状态下 commit

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
hwcloud-skillcheck drift sync --apply --root .
```

The drift guard (`hwcloud-skillcheck drift check --root .`) is wired into
`scripts/pre_commit_check.sh` and the CI workflow (`validate-skills.yml`), so a drifted runtime copy is
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

## 复利资产沉淀机制（Compound-Asset Distillation Loop, CADL）

**这不是一条规范，而是一套工作闭环——任何实质任务完成后，Agent 必须走完「提取 → 判定落点 → 写入 → 门禁」才能结束。目的是让每次踩坑、每次评审、每次跨 skill 协作都变成下一次的可复用资产，形成复利。**

### 为什么是机制而非规范

单条规则（如"记得写 AGENTS.md"）会被忽略，因为无触发、无闭环。CADL 把沉淀变成工作流的**必经出口**：任务不做沉淀 = 任务未完成。

### 触发条件（满足任一即必须走 CADL，不局限 CodeGraph）

- 多步 / 跨文件任务完成
- 跨 Skill 协作（用了 delegation matrix 或并行 agent）
- 评审 / 修复循环（如 Generic Critic Loop、GCL、self-review）
- 发现 repo 缺陷 / 坑（即使不在本次 scope，也记）
- 验证中发现预存 FAIL 并归因
- 用户给出可复用的工作流偏好（如"用双写子命令绕过 CLI bug"）

### 闭环步骤

```
1. 提取   → 从刚完成的任务中抽象出可复用模式：
            踩坑避免 / 评审维度 / 协作模式 / 验证命令 / 复用 helper
            格式："问题 → 反模式 → 正确做法（含代码示例）"
2. 落点判定 → 离开本仓库还有用？ → 用户级 ~/.config/opencode/AGENTS.md
            仅本仓库适用？     → 项目级 AGENTS.md（本文件）
            是某 skill 专属可调用的能力？ → 独立 Skill 文件（经 generator）
3. 写入   → 可执行、有示例、有边界、先 grep 现有 AGENTS.md 确认未覆盖（不重复）
4. **门禁** → 写入前查 wc -l，本文件 ≥500 行先精简再写（见 AGENTS.md 行数门禁）
5. **复用** → 下次同类任务，Agent 读 AGENTS.md 即获得该资产 → 复利生效

**Before starting GCL / Harness work, also re-read "Hard-Won Lessons" below**
(L1–L8) — every rule there was paid for in a real failure.
```

### Skill 侧钩子

- `huaweicloud-skill-generator` 生成每个 skill 时，在 SKILL.md 末尾注入一行提示，召唤 CADL 意识。
- Agent 在任意 skill 调用结束前，主动检查 CADL 触发条件，而非等用户提醒。

### 反模式（违反 CADL）

| 反模式 | 正确做法 |
|---|---|
| 任务做完就结束，不沉淀 | 走完 CADL 闭环再交付 |
| 把一次性上下文当资产写进 AGENTS.md | 只沉淀跨任务可复用的模式 |
| 重复已有条目 | 写入前 grep 确认未覆盖 |
| 只在 CodeGraph 相关任务才沉淀 | 评审/修复/协作/验证都触发 |

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

- A single shot gun covers everything: `bash scripts/pre_commit_check.sh`. This is what the git hook and CI both invoke — running it locally is equivalent to pushing.
- The git pre-commit hook is fully covered by `scripts/pre_commit_check.sh`; CI runs the same script. Markdown-only commits stay fast because Go build/test gates skip when `.go` and the `hwcloud-skillcheck/` tree are unchanged.
- New scripts MUST:
  - Start with a module docstring describing purpose.
  - Avoid unused imports / unreachable code / bare `except:`.
  - Prefer `flag` (std lib) with explicit `--help` text for CLIs.
  - Keep functions short; favor pure helpers that are unit-testable.
- Tests live next to source as Go `_test.go` files; `go test ./...` runs them. Subagent-driven-development + race detector are how new functionality is verified before commit.
- CI runs `hwcloud-skillcheck validate --root .` plus `go test ./... -race`; local dev MUST run the same suite before pushing.

> **As of 2026-07-26, zero Python scripts remain in `scripts/`.** All 17 scripts

When asking the user a multiple-choice question — through `AskUserQuestion`, free-form
prose, or any other channel — every option beyond the first MUST be accompanied by:

1. **A recommended option.** Pick the one you (the agent) would choose in the user's
   shoes given the evidence in scope, and label it as the recommendation
   (`(Recommended)` in `AskUserQuestion`; `**推荐:A**` or similar in prose).
2. **The reasoning.** One to three short lines that connect the recommendation to
   the evidence: which finding is the highest priority, which option closes the
   loop fastest, what trade-off the other options accept.

**Why this is a rule, not a default.** Most option-lists defer the call to the user.
That pushes the cognitive cost of every decision onto one person who has less
context than the agent that produced the trade-off list. The user has delegated the
investigation; they hired the agent's judgment, not its menu. The recommendation
should still be defensible — the user can override freely — but it should not be
absent.

**Acceptance test before sending a question:** if you delete the user from the
room, would you still know which option to take? If yes, state it. If no (pure
preference with no evidence), ask without a recommendation and say so.

**Exceptions** (when to ask without a recommendation):
- Personal preference (color scheme, naming, library choice with no measurable
  impact).
- Decisions locked behind missing context the user has but you don't (e.g., team
  conventions, customer SLA).
- Decisions where the user previously declared the answer (re-confirming is noise).

In every other case: lead with the recommendation.

## Test Hermeticity — Runtime-State Tests (P0)

Tests touching the real repo (`Path(__file__).resolve().parents[1]`) are **not hermetic by default** — they require state that doesn't exist on a fresh CI checkout (e.g. `audit-results/`, `.agents/skills/huaweicloud-skill-generator/`). Rules:
1. **CLI-style smoke tests** MUST tolerate *absent* state. A missing `audit-results/` is no longer a failure (runtime scripts create it on demand).
2. **Bootstrap functions** MUST self-heal: `mkdir(parents=True, exist_ok=True)` before copying; don't expect callers to pre-create destinations.
3. **Fixture-style tests** needing runtime state MUST use `tempfile.TemporaryDirectory()` **not** `ROOT`; add a `# REPO-ROOT-DEPENDENT` docstring.
4. **No silent state mutation in CI.** Guard with `unittest.skipUnless(Path("…").exists(), "requires runtime state")` or copy to a tempdir.

When a guard reports "missing" as error: is it something the *runtime* creates on demand? If yes, the guard is wrong — guard what must already be true, not what will be true after the first call. Use gitignore/mode/tracked-files checks as hard gates; "exists and is correct" is a soft expectation.

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
| Inventing API fields/CLI flags | Cross-reference against OpenAPI or verified CLI output |
| Printing/logging real credentials | Mask with `***` / `<masked>` |
| Skipping safety gate on destructive ops | Add explicit confirmation step |


- ECS → VPC (subnet), CES (metrics), ELB (load balancing)
- RDS → ECS (CloudShell), CES (performance metrics)
- All products → IAM (permission issues), CTS (audit trails), BSS (billing)

## Sources of Truth

1. OpenAPI + official docs > forums/chat
2. Verified `hcloud` CLI output > assumed behavior
3. `huaweicloud-sdk-go-v3` for SDK fallback patterns

---

## Runtime Quality Gates: GCL

Detailed runtime-quality specs are externalized. Key reads before modifying GCL-related files:
| Spec / Tool | Read or run before modifying |
|---|---|
| `docs/gcl-spec.md` | any `## Quality Gate (GCL)` section, `references/rubric.md`, `references/prompt-templates.md` |
| `hwcloud-skillcheck gcl run --root .` | runtime Orchestrator loop; external Critic required in production |
| `hwcloud-skillcheck validate --root .` | Go total-entry: frontmatter + eval-queries + product-assessment + advanced-coverage + audit-results |


- **Contexts**: isolated Generator + Critic only; shared-context G+C banned.
- **Critic**: read-only, no hcloud/SDK/mutation/self-score; sees sanitized `{{output.operation_intent}}` only.
- **Safety=0/SAFETY_FAIL**: abort immediately, never partial output.
- **Loops bounded**: every run has `max_iterations` + masked trace to `audit-results/gcl-trace-*.json`.
- **Templates**: placeholders MUST use `{{env.*}}/{{user.*}}/{{output.*}}`; bare `{…}` banned.


```bash
hwcloud-skillcheck validate --root .             # Go total-entry: frontmatter + eval-queries + product-assessment + advanced-coverage + audit-results
hwcloud-skillcheck gcl run --root . --skill huaweicloud-billing-ops --request "smoke" --command 'printf ok' --max-iter 1 --structural-critic-only
hwcloud-skillcheck aggregate trace --root . --since-hours 168
hwcloud-skillcheck gcl alarm-wire --root . --plan-file scripts/fixtures/gcl-quality-summary-healthy.json
```

### Relationship to build-time self-reflection

Build-time 2-round self-reflection and runtime GCL are independent gates. A clean self-reflection does not exempt runtime scoring; a passing GCL rubric does not exempt sloppy skill updates.

Build-time 2-round self-reflection and runtime GCL are independent gates.

### GCL changelog

| Version | Date | Change |
|---|---|---|
| 1.0.0 | 2026-06-04 | Initial GCL specification and ECS pilot |
| 1.6.0 | 2026-06-19 | qcloud-style runtime scripts, sanitized `operation_intent`, Tier-A conformance, and CES summary schema added |


## Self-Healing Loop & Experience Learning (L4)

Full spec: `references/self-healing-spec.md`

### Artifacts per skill

| File | Purpose |
|------|---------|
| `assets/remediation-playbooks.json` | Machine-readable fix playbooks (trigger→diagnose→execute→verify→rollback) |
| `assets/failure_patterns.json` | Learned failure knowledge base (signature→fix→stats) |

### Runtime scripts

```bash
# Aggregate GCL traces → update failure_patterns.json
`hwcloud-skillcheck learning trace aggregate --skill huaweicloud-ecs-ops [--since-hours 168] [--dry-run] --root .`
# Learn from single trace
`hwcloud-skillcheck learning trace learn --skill huaweicloud-ecs-ops --trace audit-results/gcl-trace-*.json --root .`
# Knowledge base report
`hwcloud-skillcheck learning trace report --skill huaweicloud-ecs-ops --root .`
```

### GCL integration

`hwcloud-skillcheck gcl run` executes one Generator command (default smoke `echo ok`; override with `--command` for production work) under a per-iteration timeout and writes the trace to `audit-results/gcl-trace-<UTC>-<rand>.json`. The pre-execution risk check on `failure_patterns.json` happens in the L4 orchestrator step loop (`hwcloud-skillcheck l4 handle`), not in `gcl run` — see the L4 section.

### Hard constraints

- Playbooks with `risk_level: critical` MUST NOT auto-execute; always escalate.
- `failure_patterns.json` is append-only during learning; manual curation required for deletion.
- `hwcloud-skillcheck learning trace aggregate` MUST be run after any GCL campaign to close the learning loop.

## CodeGraph Integration — 代码变动即时同步

CodeGraph (`codegraph` CLI) 维护仓库知识图谱。本仓库已配置 MCP Server（`.mcp.json`），Agent 启动时自动获得 `codegraph_explore` 工具。索引数据位于全局 `~/`.omo/codegraph/`（仓库内 `.codegraph` 为软链，已被 `.gitignore` 忽略）。

#### MANDATORY: CodeGraph sync 纪律

1. **读前 sync** — 任何 `codegraph explore/impact/callees` 前先 `codegraph sync --quiet`（过期索引产生假阴性）。例外：`codegraph status` 显示 up-to-date 且距变更 < 几分钟。
2. **写后 sync** — 每次 Go/Python 变更提交前必须 sync。Agent 纪律，非 CI 门禁。
3. **MCP 优先** — 代码理解任务先 `codegraph explore <symbol>`，再 grep/read 补充（AST+调用图覆盖接口实现、动态派送）。纯文本搜索除外。
4. **Fallback 层级** — 代码理解按以下顺序选择工具：
   - **首选**：`codegraph explore <symbol>`（符号定义、调用者、影响面分析）
   - **备选**：`grep` / `read` / `rg`（当 CodeGraph 不可用、索引过期、或仅需文本匹配时）
   - **显式原则**：当 `codegraph explore` 已能回答问题时，禁止跳过他直接用 grep

| 场景 | 命令 |
|------|------|
| 符号定义+调用者 | `codegraph explore <pkg.Symbol>` |
| 影响面 / 调用链 | `codegraph impact` / `codegraph callees <pkg.Symbol>` |
| 同步索引 | `codegraph sync --quiet` |

MCP 配置见 `.mcp.json`（stdio `codegraph serve --mcp`）。前置：`codegraph` 在 PATH 中（`which codegraph` 验证）。
### 版本升级规则

重大功能重构或实现完成后，Git push 成功后必须升级版本：

1. **触发条件**：完成了以下任一工作后
   - 新增了工具子命令（`hwcloud-skillcheck` 新增 `pitfall-report` 等）
   - 新增了 `internal/` 包（新的可复用模块）
   - 重构了核心 GCL / L4 / learning 逻辑
   - 删除了废弃的 Python 脚本或旧逻辑
   - 任何影响 `hwcloud-skills` 对外行为的功能变更

2. **操作步骤**：
   ```bash
   # push 完成后，在仓库根目录执行：
   task release VERSION=X.Y.Z
   ```
   `task release` 会 `git tag` + `git push origin <tag>`，触发 CI 构建和 GitHub Release。

3. **版本号规范**：遵循语义化版本（semver）
   - `X.Y.Z`：主版本.次版本.补丁版本
   - 主版本：破坏性 API 变更
   - 次版本：新功能向后兼容
   - 补丁版本：Bug 修复或小改进

> 日常提交（文档、测试用例、typo 修复等）**不需要**升级版本。

## Hard-Won Lessons (P0 收尾 + P1 规划阶段, 2026-07-27)

### L9. Restricted macOS Go cache

When Go build/test hits a read-only `~/Library/Caches/go-build` error, set `GOCACHE=/tmp/hcloud-go-cache`; this changes only compiler cache location and keeps the source tree untouched.

From the P0 trust-boundary close-out (commits `2b935ea`, `99996a2`, `deaf3b8`)
and the P1+P2 spec/plan cycle. **Each rule is a fix for a real failure that
wasted ≥30 min in this cycle.** Re-read before starting any new GCL / Harness
work.

### L1. "Reuse L4" is a lie when there's an import cycle
**Problem**: spec said "reuse `l4.HighRiskVerbs` in gcl". `l4` imports `gcl`, so `gcl` cannot import `l4` — `import cycle not allowed`.
**Fix**: copy the regex into `gcl` as `gclHighRiskVerbs` / `gclHighRiskCommands`, then add a build-time sync test (`internal/l4/gcl_sync_test.go`) that fails CI on drift. **Cost of drift** = silent trust-boundary leak. **Cost of check** = 30 lines.
**Rule**: when brainstorming says "reuse X", verify the import graph allows it *before* promising reuse.

### L2. `t.TempDir()` is not stable across calls
**Problem**: each call returns a NEW subdirectory. If `writeScript(t,…)` uses `t.TempDir()` and `readFile` also calls `t.TempDir()`, the read path is a different directory and the file is "not found".
**Fix**: cache the path per `*testing.T` via `sync.Map` keyed by the test pointer; `t.Cleanup(func() { scriptPaths.Delete(t) })` to release on test end. See `internal/gcl/critic_schema_test.go:scriptForTest` for the canonical implementation.
**Rule**: any test helper that exposes a path needs to cache per-test, not per-call.

### L3. Shell heredoc trap for JSON test scripts
**Problem**: a script like `printf '%s' {"scores":{…}}` is broken: the shell does brace expansion on the unquoted JSON, mangling it before `printf` runs. Result: `decode-error` from the subprocess with no obvious cause.
**Fix**: heredoc with a single-quoted delimiter escapes both brace and parameter expansion: `cat <<'PAYLOAD_EOF' … PAYLOAD_EOF`.
**Rule**: any test fixture containing `{` `}` `:` `$` in an unquoted context is a bug. Single-quote the heredoc delimiter (`<<'EOF'`) is the safe default.

### L4. `cmd.StdinPipe()` + `cmd.Output()` deadlocks unless you close stdin first
**Problem**: `exec.CommandContext` + `StdinPipe()` + `Output()` blocks forever if the child reads stdin and the parent never closes the pipe. The 60s context eventually fires with empty stdout → `decode-error`.
**Fix**: `Write(genBytes)` then `Close()` then `Output()`. `defer in.Close()` after `Output()` is too late.
**Rule**: any `exec.Cmd` with `StdinPipe()` MUST close the pipe before the call that waits for completion.

### L5. Edit-tool thrash accumulates orphan syntax
**Problem**: 4+ sequential `edit` tool calls on the same file (especially with brace nesting) almost always produce orphan `}` / duplicate `}` / deleted-line ghost. Re-running `go build` shows syntax errors that the latest `edit` did not introduce.
**Fix**: (a) after every 2 edits on the same file, `read` it and grep for `^{` `}$` balance. (b) if structure is broken beyond simple repair, `git checkout HEAD -- <file>` and redo from scratch — the first clean rewrite is faster than untangling. (c) the `edit` tool's "auto-repair" warnings are real — re-read after each.
**Rule**: edit-tool = single-shot per file. Multi-step edits to the same file should be batched into one large `write` instead.

### L6. Test names must match the spec, not the implementation
**Problem**: the P0 spec listed `TestConfirmationRegistryOneTime` (no underscore) as the A8 acceptance test. The shipped test was `TestConfirmationRegistry_OneTimeConsumption` — different name, same behavior. The test passed green, but the spec criterion was not directly observable.
**Fix**: ship a `TestP1Acceptance_AuditsAllCriteria` (P1 plan Task 11) that walks the repo's `_test.go` files and asserts the spec-named test strings exist. CI fails if a criterion lacks a named test.
**Rule**: spec criteria are contracts. The test name is part of the contract, not just a convenience. **Even passing tests do not satisfy a spec criterion — the test name must match.**

### L7. The spec review gate is the user's, not yours
**Problem**: when `brainstorming` is invoked, the HARD-GATE says: do NOT implement, do NOT call writing-plans, until the user approves the spec. In the P1+P2 cycle the agent marked the spec "self-reviewed" and was about to "self-approve" to keep momentum. The user had to say "now approve spec" to break the gate. That's the correct flow.
**Fix**: in todos, mark the user-review task `block(reason: "awaiting user approval")` — explicitly NOT in `done` state. The block label makes it visible to the system that this is a human checkpoint.
**Rule**: when a `block` label exists in the todo, the next phase's work (writing-plans, implementation) MUST wait. Unblocking without user input = process violation.

### L8. Pre-existing flaky tests get conflated with new failures
**Problem**: in the P0 cycle, two tests were red on a clean checkout (before any P0 work): `TestConfirmationRegistry_ConcurrentSafety` (flaky goroutine race) and `TestHandleFault_DecisionAutoProceed` (L4 fixture drift). After P0, both were still red. A naive reader would assume they were P0 regressions. The truth: they pre-date the spec.
**Fix**: in every commit message, **explicitly call out** pre-existing failures (`"TestX failure is pre-existing, not caused by this commit"`). Add a `// KNOWN-FLAKY: <reason>` comment above the test so `go test -run` can filter them.
**Rule**: a "red" test in CI is a signal, not a verdict. Every commit message should distinguish regressions from pre-existing noise.
### L10. P2 EntityMatch heuristic is a proxy, not the spec contract
**Problem**: `router.ConfidenceGate.EntityMatch` (`router.go:computeGate`) is derived from `ManifestScore >= 0.8 → "strong"`, `>= 0.4 → "weak"`, else `"absent"`. Spec §4.2.3 defines the field as "Whether a typed entity from the request matched a skill input/lexicon entry" — semantically different from the overlap-ratio proxy shipped today.
**Fix**: when wiring the real entity recognizer (lexicon.products/actions/resources lookup), update `computeGate` to query it. Until then, mark the field `// heuristic: see L10` near its set-site, and document the gap in the spec changelog ("v0.3.0 partial GREEN").
**Cost of not flagging**: a green-build narrative hides a real semantic gap that will surprise a Critic reviewing a low-overlap request.
**Rule**: any time a contract field is "good enough for GREEN but not for spec", leave a L-numbered lesson near the set site so the next implementer (or Critic) can find it in 10 seconds.
### L11. Sandbox network is sealed — design for offline-mode, not for "I'll just download it"
**Problem**: sandbox cannot reach `github.com`, `goproxy.cn`, `proxy.golang.org`; only local stub hosts resolve.
**Fix**: when a task requires a network-bound dep, do ONE of: (1) vendor into `hwcloud-skillcheck/vendor/`, (2) implement offline-mode, (3) document the blocker with unlock conditions.
**Rule**: before committing to "use SDK X", verify X is vendored, in module cache, or reachable. Otherwise plan for offline-mode first.

### L12. Every sandbox/provider needs a user-facing preflight
**Problem**: provider configuration failures surface as opaque errors, leading to repeated edit/restart cycles and credential-in-config mistakes.
**Fix**: implement side-effect-free `Preflight(ProviderConfig) PreflightReport` that aggregates all errors before network/native calls, with plain-language Message and actionable Fix.
**Rule**: a new sandbox provider is incomplete until invalid config is covered by preflight tests and the user manual has copy-paste fixes.

### L13. Env-var dependency injection must sync test fixture + injection channel
**Problem**: `TestSafetyClass_UnknownValue` created `sanitizer.go` scaffold but did not set `SKILLCHECK_ROOT` — code under test searched the build cache path instead of the scaffold dir.
**Fix**: when a test creates a fixture file that a production function reads via env var, always `t.Setenv("VAR_NAME", tmp)` in the same test that creates the file.
**Rule**: fixture creation and dependency injection are a contract — both must exist together or neither works.

### L14. Subprocess Mode = exit code, not decode intent
**Problem**: `TestExternalCritic_DecodeError` asserted `Mode=="decode-error"` but the helper exited non-zero (no stdout), producing `subprocess-error` — the test named the intention not the outcome.
**Fix**: subprocess exit ≠ 0 → `out` is empty → `Mode` should reflect subprocess error, not json.Unmarshal failure. Assert the actual code path, not the hoped-for path.
**Rule**: test assertion names must match what the code actually does, not what you intend it to do. "NonZeroExit" not "DecodeError".

### L15. CI Linux `go test` makes `os.Args[0]`/Executable unusable for source-tree walking
**Problem**: `checkSafetyClassCode(skillcheckRoot)` used `filepath.Dir(os.Args[0])` to find the source tree — worked on macOS compiled binary, failed on Linux CI where `go test` puts args[0] in `~/.cache/go-build/...`.
**Fix**: walk up from both `os.Args[0]` AND `os.Getwd()`; stop when `sanitizer.go` is found. `cwd` fallback is the reliable anchor for CI.
**Rule**: `os.Args[0]`/`os.Executable()` are not reliable in `go test` / build-cache environments. Always fall back to `cwd` walking.

### L16. Programmatic Go file edits require gofmt verification before commit
**Problem**: sed/python/`edit` tool inserted code with extra blank lines; macOS gofmt passed, Linux CI gofmt failed → CI red on a clean commit.
**Fix**: after any programmatic edit (sed, python, edit tool), always run `gofmt -l <file>` locally and `gofmt -w <file>` to auto-fix before commit.
**Rule**: `go build` passing locally is insufficient; CI gofmt gate is authoritative. Always verify before push.
