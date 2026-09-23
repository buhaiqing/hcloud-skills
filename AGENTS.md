# AGENTS.md — hcloud-skills

## Pre-flight Gate (每次执行前必跑)

收到任务后，**先跑以下 checklist，再动手**：

1. **GCL 触发检查（质量门禁）** — 见下方 §GCL Auto-Execution Gate；满足触发条件 A-E 任一 → 启动 GCL 多子 Agent 架构（详见 §GCL）
2. **Orchestrator 触发检查（执行编排方式）** — 任务是否涉及多文件 / 多阶段 / 多 skill / 用户提到「orchestrator」？→ 是则加载 `subagent-orchestrator` skill，在其 `scripts/` 下运行 `python3 decide.py decide <task_type> <complexity> <risk> <count>` 输出决策 JSON，再按决策执行
3. **Skill generator 检查** — 是否在创建 / 更新 `huaweicloud-*-ops`？→ 是则加载 `huaweicloud-skill-generator` skill
4. **直接执行** — 以上均否 → 直接做

> **GCL 与 Orchestrator 的关系（正交，不冲突）**：GCL 是**质量门禁**（决定「是否用多子 Agent 评审」），
> orchestrator 是**执行编排**（决定「怎么拆/并行还是串行」）。两者独立判定：
> - 触发了 GCL → 走 §GCL 多子 Agent 评审
> - 触发了 orchestrator → 用 decide.py 定编排策略（direct_exec 则主 agent 直接做）
> - 两者同时触发（如多文件 Go 重构）→ **GCL 评审优先**，编排策略服从 GCL 的 Generator/Critic 结构

> 违反此 gate = 流程违规，即使结果正确也需复盘。

### GCL Auto-Execution Gate（强制 — 每次编码/配置任务前判定）

> 详细 GCL 规范见用户级 `~/.codebuddy/rules/gcl-rules.md`。
> 本节是 **hcloud-skills 项目专用的最小可执行决策树**，确保 GCL 不会因 Agent 疏忽而跳过。

#### 触发判定（满足任一即触发 GCL）

| # | 条件 | 示例 |
|---|------|------|
| A | 预计代码变更 > 5 行 | 新增功能、重构、bug fix |
| B | 修改运维配置文件 | `.yml`、`.yaml`、`.json`、`.tf`、`.hcl`、`.toml` |
| C | 任务含触发关键词 | 修复/新增/重构/变更/优化/测试、fix/add/refactor/change/optimize/test |
| D | 修改 GCL 核心文件 | `SKILL.md`、`rubric.md`、`prompt-templates.md`、`AGENTS.md` §GCL |
| E | 修改 Go 代码 | `hwcloud-skillcheck/**/*.go`、`scripts/**/*.go` |

**例外（仅代码变更）**：< 5 行的 typo/注释/格式化改动可跳过 GCL，但须执行 2-round self-review。
**运维配置变更无例外**：所有 `.yml`/`.yaml`/`.json`/`.tf` 等变更必须走 GCL。

#### 风险分级（Risk Triage）

在触发判定前，先评估任务风险等级，决定 GCL 执行深度：

| 风险等级 | 判定条件 | GCL 路径 | 预期开销 |
|----------|----------|----------|----------|
| **Low** | <5 行变更；typo/注释/格式化；纯文档；**非核心文件、非配置文件** | 跳过 GCL → 2-round self-review | ~5 calls |
| **Medium** | 单文件修改；配置微调；非核心文档 | Light GCL → 1 Critic（无 Generator） | ~15 calls |
| **High** | 多文件重构；新功能；Go 代码；核心文件；**运维配置文件** | Full GCL → Generator + ≥2 Critics | ~50 calls |

**快速判定规则**：
```
IF is_core_file OR is_config_file THEN risk = "high"
ELSE IF files_changed <= 1 AND lines_changed < 5 AND NOT is_go_code THEN risk = "low"
ELSE IF files_changed <= 3 AND lines_changed < 100 THEN risk = "medium"
ELSE risk = "high"
```

**核心文件白名单**：`SKILL.md`、`rubric.md`、`prompt-templates.md`、`AGENTS.md` §GCL
**配置文件白名单**：`.yml`、`.yaml`、`.json`、`.tf`、`.hcl`、`.toml`（运维配置变更无例外）

#### 循环预算（Loop Budget — 硬限制）

每个任务必须设定预算上限，超限则停止并汇报：

| 任务类型 | 最大工具调用 | 最大 Token | 门禁 |
|----------|-------------|-----------|------|
| Typo/注释/格式化 | 10 | 5,000 | Pre-commit only |
| 单函数修改 | 30 | 20,000 | Pre-commit + Post-push |
| 配置变更 | 40 | 30,000 | Pre-commit + Post-push + GCL Light |
| 多文件重构 | 100 | 50,000 | Pre-commit + Post-push + Full GCL |
| 新功能开发 | 150 | 80,000 | Pre-commit + Post-push + Full GCL |

**预算执行规则**：
1. **跟踪**：每 10 个工具调用检查一次预算使用情况
2. **警告**：达到 80% 预算时输出警告
3. **硬停**：达到 100% 预算时立即停止，输出当前进度和阻塞原因
4. **升级**：硬停后等待用户指示，不自动重试

**预算分配建议**：
- 探索阶段：20%（文件读取、代码搜索、上下文收集）
- 实现阶段：50%（代码编写、配置修改、文档更新）
- 验证阶段：25%（测试运行、lint 检查、GCL 评审）
- 收尾阶段：5%（提交、清理、总结）
- **GCL 评审额外预算**：Generator + Critics 调用不计入主任务预算，由 GCL 系统单独管控

**超限处理**：
```
IF tool_calls >= max_tool_calls OR tokens >= max_tokens THEN
  输出 "⚠️ 预算超限" + 当前进度 + 阻塞原因
  等待用户指示（继续/调整范围/终止）
END IF
```

#### 状态管理统一（State Query Layer）

当前状态存储分散在 5 个位置，Agent 需要多次工具调用获取完整状态。目标是引入统一查询层：

| 状态 | 位置 | 当前查询方式 |
|------|------|-------------|
| GCL Trace | `audit-results/gcl-trace-*.json` | `hwcloud-skillcheck aggregate trace --root .` |
| Failure Patterns | `assets/failure_patterns.json` | `hwcloud-skillcheck learning trace report --skill <name> --root .` |
| Remediation Playbooks | `assets/remediation-playbooks.json` | 同上 |
| Context Memory | `.l4-memory/context.json` | 直接读取 |
| Outcome Memory | `.l4-memory/outcomes.jsonl` | 直接读取 |

**当前最佳实践**：任务开始前按需查询相关状态，任务结束后调用 `hwcloud-skillcheck learning trace aggregate` 更新。

**目标架构**（待实现）：
```bash
hwcloud-skillcheck status --skill <name> --root .
# 返回 JSON：{ "gcl_traces": [...], "failure_patterns": {...}, ... }
```

**使用规则**：
1. **任务结束后**：调用 `hwcloud-skillcheck learning trace aggregate` 更新状态
2. **避免**：直接读取多个文件拼接状态（应按需查询）

**状态清理**：
- GCL Trace：保留最近 7 天
- Failure Patterns：append-only，手动 curation 删除

#### 执行决策树

```
收到任务
  ├─ 风险评估（risk_tier）
  │   ├─ LOW → 直接执行 + 2-round self-review
  │   ├─ MEDIUM → Light GCL
  │   │           ├─ 创建 worktree
  │   │           ├─ spawn 1 Critic（后台）
  │   │           ├─ 执行 1 轮评审
  │   │           └─ 汇总结果
  │   └─ HIGH → Full GCL
  │             ├─ 创建 worktree
  │             ├─ 输出模型配置公示
  │             ├─ spawn Generator（后台）
  │             ├─ spawn ≥2 Critics（后台，并行，不同厂商模型）
  │             ├─ 执行 GCL 循环（最多 3 轮）
  │             └─ 汇总结果，写入 memory
  └─ 触发条件 A-E 任一满足？
      ├─ YES → 按风险等级走对应路径
      └─ NO  → 直接执行 + 2-round self-review
```

#### 模型选型（硬约束）

| 角色 | 模型要求 | 厂商要求 |
|------|----------|----------|
| Generator | 中等模型 | 厂商 A |
| Critics (≥2) | 旗舰模型 | 厂商 B（不同厂商）或同厂商更高等级 |

**启动前必须向用户输出模型配置公示**。

#### GCL 门禁阈值

| 维度 | 阈值 | 不达标处理 |
|------|------|------------|
| Correctness | ≥ 0.5 | 重试（最多 3 轮） |
| Safety | = 1.0 | **立即中止**，不生成部分结果 |
| Idempotency | ≥ 0.5 | 重试 |
| Traceability | ≥ 0.5 | 重试 |
| Spec Compliance | ≥ 0.5 | 重试 |

#### 子 Agent 失败处理

| 失败类型 | 处理 |
|----------|------|
| API 限流 (429) | 主 Agent 直接接管 |
| 上下文超限 | 拆分任务，重新 spawn |
| 连续 2 次失败 | 主 Agent 直接接管 |
| 子 Agent 卡死 (>10min 无输出) | 发送询问 → 30s 无响应 → 强制停止，主 Agent 接管 |

### Pre-commit Gate（本地提交前 — 强制，不可跳过）

**任何 `git commit` 之前，必须通过以下检查。** 一站式执行：

```bash
cd hwcloud-skillcheck && go test -race ./... && go vet ./... && gofmt -l .
```

| 检查项 | 命令 | 失败处理 |
|--------|------|----------|
| 单元测试 | `go test -race ./...` | exit code ≠ 0 → **禁止 commit**，先修测试 |
| 静态分析 | `go vet ./...` | 零 warning |
| 格式化 | `gofmt -l .` | 空输出 |
| 凭据泄露 | `maskSecrets` 覆盖所有 detail 输出 | 禁止明文 |
| 文档校验 | `hwcloud-skillcheck validate --root .`（仅涉及文档变更时） | 通过 |

> **注意**：`git commit` 本身不自动触发 pre-commit hook 的 go test（hook 在 `hwcloud-skillcheck check --pre-commit` 中）；但 **Agent 必须自行检查**，不允许在测试 red 状态下 commit。
> 如涉及 GCL：memory 文件（`feedback_gcl_auto_trigger.md`、`feedback_gcl_execution.md`）需在此步骤写入。

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
`hwcloud-skillcheck check --pre-commit` and the CI workflow (`validate-skills.yml`), so a drifted runtime copy is
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

## 复利资产沉淀机制（CADL）

> 完整规范：[`references/cadl-spec.md`](references/cadl-spec.md)

**核心要点**：
- 默认不写入；写入前过「四问」（复用半径/失败成本/抽象层级/可执行性）
- 复利资产目标 ≤12 条；超出则 prune 最弱条目
- 任务结束时主动做沉淀判定

## Skill Update Rule: 2-Round Self-Reflection

> 完整规范：[`references/skill-update-rule.md`](references/skill-update-rule.md)

**核心要点**：
- 每次 skill 更新/创建后，执行 2 轮 self-reflection
- Round 1: FinOps/SecOps/AIOps + Token Efficiency (TE-1~TE-7)
- Round 2: Gap Analysis / Alternative Coverage / Escalation Paths / Cross-Pillar Synergy
- 发现问题立即修复，不报告即停

## Go 编码规范

详见 [`references/go-coding-standards.md`](references/go-coding-standards.md)（G1-G9：可测试性、错误处理、并发安全、资源管理、性能、可扩展性、代码组织、输入验证、TDD 工作流）。所有 `hwcloud-skillcheck/` Go 代码必须遵守。

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
## Sources of Truth

1. OpenAPI + official docs > forums/chat
2. Verified `hcloud` CLI output > assumed behavior
3. `huaweicloud-sdk-go-v3` for SDK fallback patterns

---

## Documentation Locations (强制)

> 完整规范：[`references/documentation-locations.md`](references/documentation-locations.md)

**核心要点**：
- ADR → `docs/architecture/NNNN-<slug>.md`
- Spec → `docs/superpowers/specs/<slug>.md`
- Plan → `docs/superpowers/plans/YYYY-MM-DD-<slug>.md`
- 禁止随意新建顶层 docs/ 子目录

---

## 术语表 (Glossary)

> 完整规范：[`references/glossary.md`](references/glossary.md)

**核心术语**：
- **GCL** — Generator + Critic 双 Agent 闭环质量门控
- **L4 Orchestrator** — 多 step 执行 + RBAC + GCL + topology + trust + healing
- **Outcome Memory** — 跨任务 step 结果 JSONL，self-healing 底座
- **CADL** — 复利资产沉淀机制（见 `references/cadl-spec.md`）
- **Dual-Copy Trap** — generator 根副本 vs `.agents/skills/` 运行时副本

> L4 实现约束（改 healing/trust/executor 前必读）→ `references/glossary.md`

---

## Runtime Quality Gates: GCL

> 完整规范：[`references/gcl-runtime.md`](references/gcl-runtime.md)

**核心约束**：
- Contexts: isolated Generator + Critic only; shared-context G+C banned
- Safety=0/SAFETY_FAIL: abort immediately, never partial output
- Loops bounded: every run has `max_iterations` + masked trace

```bash
hwcloud-skillcheck validate --root .             # Go total-entry: frontmatter (incl. dangling `delegates_to` targets) + eval-queries + product-assessment + advanced-coverage + audit-results
hwcloud-skillcheck gcl run --root huaweicloud-billing-ops --request "smoke" --command 'printf ok' --max-iter 1 --structural-critic-only
hwcloud-skillcheck aggregate trace --root . --since-hours 168
hwcloud-skillcheck gcl alarm-wire --root . --plan-file scripts/fixtures/gcl-quality-summary-healthy.json
```

## Self-Healing Loop & Experience Learning (L4)

> 完整规范：[`references/self-healing-spec.md`](references/self-healing-spec.md)

**核心产物**：
- `assets/remediation-playbooks.json` — 修复 playbooks
- `assets/failure_patterns.json` — 失败知识库

**关键命令**：
```bash
hwcloud-skillcheck learning trace aggregate --skill <name> --root .
hwcloud-skillcheck learning trace report --skill <name> --root .
```

**硬约束**：`risk_level: critical` 的 playbooks 禁止自动执行；`failure_patterns.json` append-only；每次 GCL campaign 后必须跑 `hwcloud-skillcheck learning trace aggregate` 闭环。

## CodeGraph Integration — 代码变动即时同步

> 完整规范：[`references/codegraph-integration.md`](references/codegraph-integration.md)

**核心纪律**：
- 读前 sync（`codegraph sync --quiet`）
- 写后 sync（Go/Python 变更提交前）
- MCP 优先（`codegraph explore` 优于 grep）
- Go 符号用 `pkg.Symbol`；编译先行

### 版本升级规则

> 完整规范：[`references/version-upgrade.md`](references/version-upgrade.md)

**核心要点**：
- 触发条件：新增工具子命令 / 新增 internal 包 / 重构核心逻辑 / 功能变更
- 操作：`task release VERSION=X.Y.Z`
- 版本号：语义化版本（主版本.次版本.补丁版本）

### Post-push Gate（推送后 — CI 验证）

与 **Pre-commit Gate**（本地提交前）配对。平台：GitHub Actions（`.github/workflows/*.yml`）。

| 步骤 | 动作 | 失败处理 |
|------|------|----------|
| 1. watch CI | `gh run watch --exit-status` 或 Actions UI | — |
| 2. CI 失败 | 本地 `go test ./...` 先绿 → 最小 fix → `fix(ci): …` commit → 再 push | 最多 3 轮 auto-recover |
| 3. 升级 | 3 轮仍失败 → 升级用户 | — |
| 4. fix commit | body 含 classifier + run id（模板见 `docs/deployment-guide.md` §4.3） | — |
| 5. 沉淀 | 仅当 fix 提取出通过 CADL 四问的决策规则时写入「复利资产」 | — |

> 操作细节（API、log 拉取、escalation 条件、commit 模板）→ **`docs/deployment-guide.md` §4.3**
## 复利资产（Curated — 开始 GCL / Harness / L4 工作前速读）

> 每条均通过 CADL 四问。完整踩坑叙事在 ADR / commit / PR，此处只留 **Rule**。
> 开始 GCL / Harness / L4 工作前速读本节即可。

### CA-1. 承诺 reuse 前先验 import graph
**Rule**: spec 说「reuse X」→ 先确认无 import cycle；有 cycle → 复制 + sync test 门禁，不硬 import。

### CA-2. Edit-tool 多轮 patch = 语法孤儿
**Rule**: 同一文件 ≥2 次 edit → 必须 read 验结构；嵌套乱 → `git checkout -- <file>` 整文件 rewrite，比逐行修快。

### CA-3. Spec 审批门在人类，不在 agent
**Rule**: spec/plan 阶段存在 user-approval checkpoint → todo 标 `block`，未获用户明示「approve」前禁止进入 implementation。

### CA-4. 「GREEN 但不符合 spec」= 语义债
**Rule**: 契约字段用 proxy/heuristic 凑绿 → 在 set-site 标 `// heuristic: see CA-4` + spec changelog 记 gap；否则 Critic 会被假绿误导。

### CA-5. Sandbox 无公网 — 设计 offline-first
**Rule**: 新依赖先查 vendor / module cache / 可达性；不可达 → offline-mode 或显式 blocker，不假设 `go get` 能跑。

### CA-6. `go test` / CI 里 `os.Args[0]` 不可信
**Rule**: 找源码树 → `os.Args[0]` 与 `os.Getwd()` 双路径向上 walk；Linux CI build cache 下 args[0] 不在 repo 内。

### CA-7. CLI `--root` 必须 cwd-tolerant
**Rule**: 「repo root」类 flag → walk up 找标志文件（generator、`SKILL.md` 等），不能只 `filepath.Abs(".")`。

### CA-8. CLI 产出物默认不进 git
**Rule**: 文件是 subcommand **输出**且非手改 → `.gitignore` + loader 自 seed；timestamp-only diff = 不该 track 的信号。

### CA-9. 多 workflow 重叠 → 按 trigger 职责拆分
**Rule**: 审计 step 重叠；「每次 commit」与「release artifact」用不同 trigger 事件拆分。`paths:` 只减文档噪音，不治架构重复。

### CA-10. Dual-Copy Trap（generator 双副本）
**Rule**: 只改 `huaweicloud-skill-generator/` 根副本；改后 `hwcloud-skillcheck drift sync --apply --root .`，CI `drift check` 会拦漂移。

> 原 CA-11 ~ CA-14（档案记作 CA-A11 ~ CA-A14）已降级到 [`references/ca-archive.md`](references/ca-archive.md)，完整 Why + How to apply 在档案中保留。
> 退役原因：触达半径窄，踩坑叙事仍有用，但不属高频阅读场景。

### CA-11. GCL 是每次编码任务的强制前置步骤
**Rule**: 任何涉及代码/配置变更的任务，在动手前必须先跑 GCL Auto-Execution Gate 决策树（§GCL Auto-Execution Gate）。满足触发条件 A-E 任一 → 必须启动 GCL 多子 Agent 架构；不允许「先写代码再补 GCL」或「这次变更小，跳过 GCL」。**Why**: Phase 5 中 E1-E4 每批次都通过 GCL 多子 Agent 架构评审，Critic 共发现 1 个 BLOCKER（bin/ gitignored）和 4 个 MAJOR（test retry、alarm-wire、soft gates、drift dry-run），这些在自审中均未被发现。如果没有 GCL 门禁，这些问题会直接合入 main。**How to apply**: 收到编码任务后，第一步对照 §GCL Auto-Execution Gate 的 5 个触发条件判定；满足任一则立即启动 GCL（创建 worktree → 公示模型 → spawn Generator + ≥2 Critics → 循环评审）。

### CA-12. Critic 模型必须强于 Generator
**Rule**: Critic 必须使用比 Generator 更强的模型（不同厂商最优，同厂商更高等级次之）。不能用相同模型做 Generator 和 Critic——同构模型会产生同构盲区，漏掉 Generator 的系统性错误。**Why**: Phase 5 中 E1-E3 的 Critics 使用了与 Generator 不同的模型组合，发现了 Generator 自审无法发现的 BLOCKER（bin/ gitignored）和行为丢失（test retry、alarm-wire 等）。**How to apply**: 启动 GCL 前公示模型配置；Critic ≥2 个，必须包含至少一个不同厂商的旗舰模型。
