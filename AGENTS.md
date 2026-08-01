# AGENTS.md — hcloud-skills

## Pre-flight Gate (每次执行前必跑)

收到任务后，**先跑以下 checklist，再动手**：

1. **GCL 触发检查** — 见下方 §GCL Auto-Execution Gate；满足触发条件 → 启动 GCL 多子 Agent 架构
2. **Orchestrator 触发检查** — 任务是否涉及多文件 / 多阶段 / 多 skill / 用户提到「orchestrator」？→ 是则加载 `subagent-orchestrator` skill 并输出决策 JSON，再执行
3. **Skill generator 检查** — 是否在创建 / 更新 `huaweicloud-*-ops`？→ 是则加载 `huaweicloud-skill-generator` skill
4. **直接执行** — 以上均否 → 直接做

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

#### 执行决策树

```
收到任务
  ├─ 触发条件 A-E 任一满足？
  │   ├─ YES → 启动 GCL 多子 Agent 架构
  │   │         ├─ 创建 worktree（`git-worktree.md`）
  │   │         ├─ 输出模型配置公示
  │   │         ├─ spawn Generator（后台）
  │   │         ├─ spawn ≥2 Critics（后台，并行，不同厂商模型）
  │   │         ├─ 执行 GCL 循环（最多 3 轮）
  │   │         └─ 汇总结果，写入 memory
  │   └─ NO  → 直接执行 + 2-round self-review
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

#### 收尾检查清单（GCL 通过后、commit 前）

- [ ] `go test -race ./...` 全绿
- [ ] `go vet ./...` 零 warning
- [ ] `gofmt -l .` 空输出
- [ ] 无凭据泄露（`maskSecrets` 覆盖所有 detail 输出）
- [ ] memory 文件已写入（`feedback_gcl_auto_trigger.md`、`feedback_gcl_execution.md`）
- [ ] 如涉及文档变更：`hwcloud-skillcheck validate --root .` 通过

### Commit Gate（强制 — 不可跳过）

**任何 `git commit` 之前，必须确认所有单元测试通过。** 执行：

```bash
cd hwcloud-skillcheck && go test ./...
```

- exit code ≠ 0 → **禁止 commit**，先修测试
- 2 个预存已知失败（`TestConfirmationRegistry_ConcurrentSafety`、`TestHandleFault_DecisionAutoProceed`）已在 commit 中标注，后续应修复或标注 `// KNOWN-FLAKY: <reason>`
- `git commit` 本身可正常执行（不自动触发 pre-commit hook 的 go test，因为 go test gate 在 `hwcloud-skillcheck check --pre-commit` 中）；但 **Agent 必须自行检查**，不允许在测试 red 状态下 commit

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

## 复利资产沉淀机制（Compound-Asset Distillation Loop, CADL）

**目的**：让少量高价值决策规则产生复利——下次同类任务**不读代码、不重复踩坑**也能走对。
**默认不写。** 大多数任务的正确终点是：测试绿、CI 绿、代码/配置即文档——**不是**再抄一遍到 AGENTS.md。

### 价值取向（什么值得沉淀）

复利资产 ≠ 经验日记。写入前必须满足 **「四问全过」**：

| # | 问题 | 过栏 |
|---|------|------|
| 1 | **复用半径** — 未来还有多少任务会碰到？ | ≥3 次同类场景，或跨 skill / 跨模块 |
| 2 | **失败成本** — 如果不写，会怎样？ | silent wrong（看起来绿、实际错）或 ≥30min 排查 |
| 3 | **抽象层级** — 这是决策规则还是操作手册？ | 决策规则（遇到 X → 做 Y）；不是标准库/工具官方文档可查到的事实 |
| 4 | **可执行性** — agent 读完能立刻改变行为吗？ | 一条 Rule 即可约束；不需要再读 200 行上下文 |

**任一不过 → 不写入 AGENTS.md。** 落点降级：

| 情况 | 落点 |
|------|------|
| 已用测试 / CI / ADR / workflow 门禁 | ** nowhere ** — 代码即文档，不写 |
| 仅本仓库、但高价值 | 本节「复利资产」或上方规范章节 |
| 跨仓库通用 | 用户级 `~/.config/opencode/AGENTS.md` |
| 某 skill 专属 | skill 的 `references/`，不经 AGENTS.md |

### 明确不写入（反模式）

- **已修复的一次性 bug** — 测试或 CI 已覆盖，下次 red 即信号
- **标准实践** — gofmt、heredoc 引号、`StdinPipe` 先 Close、optional JSON 设默认值
- **环境小技巧** — `GOCACHE=/tmp/...`、action 版本号；写进 commit/PR 即可
- **与现有条目重复** — 写入前 `grep AGENTS.md`；重复 = 噪音
- **纯叙事** — 「某次 CI 红了然后修了」无 Rule 可提取

### 触发条件（任务结束时检查）

满足任一 → 走沉淀**判定**（不是判定 = 必须写）：

- 多步 / 跨文件 / 跨 skill 任务完成
- 评审或修复循环（GCL、self-review、CI auto-recover）
- 发现 silent wrong 或架构级坑
- 用户给出可复用的工作流偏好

### 闭环步骤

```
1. 提取   → 能否写成一条 Rule？不能 → 停止
2. 四问   → 全过？不过 → 停止（或降级到 ADR / commit message）
3. grep   → 已有覆盖？→ 停止
4. 落点   → 复利资产 / 规范章节 / ADR / skill references
5. 门禁   → AGENTS.md ≥500 行时，加一条必须删或合并一条（见下行数预算）
6. 复用   → 下次同类任务读 AGENTS.md 即生效
```

### 行数预算

AGENTS.md 是 **agent 上下文税**，不是 wiki。硬上限意识：

- **规范 + 门禁**（Pre-flight、Dual-Copy、TE、GCL 指针）：保留，这是 repo 的操作系统
- **术语表**：索引 ADR，不复制 API 面（详见 `docs/architecture/`）
- **复利资产**： curated，目标 **≤12 条**；超出则 prune 最弱条目

### Skill 侧钩子

- Agent 任务结束前主动做沉淀**判定**；用户未要求时不批量写条目
- `huaweicloud-skill-generator` 在 SKILL.md 末尾保留一行 CADL 提示即可

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

- A single shot gun covers everything: `hwcloud-skillcheck check --pre-commit`. This is what the git hook and CI both invoke — running it locally is equivalent to pushing.
- The git pre-commit hook is fully covered by `hwcloud-skillcheck check --pre-commit`; CI runs the same command. Markdown-only commits stay fast because Go build/test gates skip when `.go` and the `hwcloud-skillcheck/` tree are unchanged.
- New scripts MUST:
  - Start with a module docstring describing purpose.
  - Avoid unused imports / unreachable code / bare `except:`.
  - Prefer `flag` (std lib) with explicit `--help` text for CLIs.
  - Keep functions short; favor pure helpers that are unit-testable.
- Tests live next to source as Go `_test.go` files; `go test ./...` runs them. Subagent-driven-development + race detector are how new functionality is verified before commit.
- CI runs `hwcloud-skillcheck validate --root .` plus `go test ./... -race`; local dev MUST run the same suite before pushing.

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

文档必须放置在以下固定位置，**禁止随意新建顶层 docs/ 子目录**：

| 类型 | 路径 | 说明 |
|------|------|------|
| **ADR（架构决策记录）** | `docs/architecture/NNNN-<slug>.md` | 编号递增，slug 用 kebab-case。任何架构选型（存储/接口/外部依赖/取舍）必写 ADR |
| **Spec（功能规格）** | `docs/superpowers/specs/<slug>.md` | 配合 ADR 写，描述 FR/NFR/数据模型 |
| **Implementation Plan** | `docs/superpowers/plans/YYYY-MM-DD-<slug>.md` | 遵循 `superpowers:writing-plans` 模板 |
| **运行时规范** | `docs/gcl-spec.md`、`docs/deployment-guide.md` 等根级 | 不轻易新建根级 .md，先复用现有 |

**ADR 文件名约束**：

- 4 位数字编号（`0001` ~ `9999`），递增
- 单数主题一个 ADR（如 `0007-outcome-memory-self-healing.md`）
- 状态字段：`Proposed` → `Accepted` → `Superseded`，写入 frontmatter 或正文

**反模式**：

- ❌ 把 ADR 写到 **docs/adr/**、**docs/decisions/**、**docs/adr-NNNN/** 等其他目录
- ❌ 把 Plan 写到 **docs/plans/** 或根级 **docs/<feature>.md**
- ❌ 没有编号的 ADR（如 architecture-decision.md 不允许）

**Why**: 跨仓库协作时（如 GCL 生成新 skill 时引用 ADR），固定路径才能让引用稳定。`docs/architecture/` 是 hcloud-skills 项目的硬约定，所有 skill / generator / docs 工具都必须遵守。

---

## 术语表 (Glossary)

> **索引，非副本。** 字段级 API → ADR（`docs/architecture/`）与源码（`internal/l4/`）。

| 术语 | 一句话 | 详见 |
|------|--------|------|
| **L3→L4 / L4→L5** | Agent 成熟度跃迁；L4 = outcome memory + healing；L5 = trust 单一来源 | ADR-0007~0009 |
| **Outcome Memory** | 跨任务 step 结果 JSONL（`.l4-memory/outcomes.jsonl`），self-healing 底座 | ADR-0007 |
| **Context Memory** | 跨调用 agent 状态 JSON（`.l4-memory/context.json`），atomic write | ADR-0008 |
| **Trust Score / Phase 1–4** | 历史-derived 可信度；Phase 4 后单一来源 = outcome memory | ADR-0009 |
| **Executor / RealExecutor** | `RunExecutionLoop` 与 subprocess 的 interface seam | ADR-0010 |
| **GCL** | Generator + Critic 双 Agent 闭环质量门控 | `docs/gcl-spec.md` |
| **L4 Orchestrator** | 多 step 执行 + RBAC + GCL + topology + trust + healing | `internal/l4/` |
| **Cross-skill delegation** | Orchestrator 经 `DelegatesTo` 扩计划并同步 pipeline 执行（非 skill 互调） | ADR-0011 |
| **RBAC** | 按 `RBACRisk` 做操作前权限决策 | `internal/l4/rbac.go` |
| **Topology Graph** | skill→resource 静态+动态依赖图 | `internal/l4/topology.go` |
| **CADL** | 复利资产沉淀机制（见上文 §CADL） | 本节 |
| **Dual-Copy Trap** | generator 根副本 vs `.agents/skills/` 运行时副本；见 CA-10 | 上文 §Dual-Copy |

**L4 实现约束**（改 healing/trust/executor 前必读）：

1. `HealingPolicy` 零值安全 → 只以 `p.IsZero()` 判断，不用 sum-based check
2. destructive verb 列表只从 `ExtractHighRiskVerbs()` 取；匹配走 `TaskStep.Verb` 非 `Action` 子串
3. 改 `Executor` interface 或 bypass → 新 ADR

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

#### GoLang 程序集成规范（hwcloud-skillcheck 等 Go 工程）

`codegraph` 的索引基于 AST/调用图，对 Go 的符号命名有固定约定。在 Go 工程中集成或排查 CodeGraph 时必须遵守：

1. **符号记法** — Go 符号用 `pkg.Symbol`（包路径末段 + 导出符号），例如 `internal/l4.TrustScore`、`internal/l4.EvaluateOperationWithHistory`。`codegraph explore` 入参区分大小写，仅索引导出符号（首字母大写）。
2. **编译先行** — 任何 `codegraph explore/impact/callees` 针对 Go 符号前，先确保 `go build ./...` 通过。索引器解析依赖 AST，**编译失败 → 符号缺失 → 假阴性**。
3. **写后 sync 强约束** — 修改 `internal/` 下任何 Go 文件（含 `_test.go`）后、提交前必须 `codegraph sync --quiet`。Go 的接口实现/动态派送（如 `Executor` interface、`HealingPolicy`）只在 sync 后才反映到调用图。
4. **影响面分析优先于 grep** — 改 `internal/l4/` 等核心包前，先 `codegraph impact <pkg.Symbol>` 拿到真实调用方（含间接调用者），再决定是否需 cascade 修改；禁止仅凭 `grep` 判定「无调用方」。
5. **vendor / 离线** — 沙箱无公网时 `codegraph sync` 可能拉取失败；此时回退到 `grep`/`read` 并标注 `// OFFLINE: codegraph unavailable`，不得假设索引存在。

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

## Post-Push CI Monitoring（强制 — 每次 push 后必跑）

与文首 **Commit Gate**（push 前 `go test`）配对。平台：GitHub Actions（`.github/workflows/*.yml`）。

1. **`git push` 成功** → watch CI 到终态（`gh run watch --exit-status` 或 Actions UI / API 等价物）
2. **CI 失败** → 本地 `go test ./...` 先绿 → 最小 fix → `fix(ci): …` commit → 再 push
3. **最多 3 轮** auto-recover；仍失败 → 升级用户
4. **fix commit** body 含 classifier + run id（模板见 `docs/deployment-guide.md` §4.3）
5. **沉淀** — 仅当 fix 提取出通过 CADL 四问的决策规则时写入「复利资产」

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

### CA-11. Git worktree 的 `.git` 是文件，不是目录
**Rule**: worktree 下 `.git` 是一个指向主仓库 `.git/worktrees/<name>` 的文本文件（不是目录）。因此 worktree 中 `git rev-parse --show-toplevel` 返回的是 worktree 自己的根（正确），但 `.git/hooks/` 是主仓库的 hooks（共享）。安装 hook 时写入主仓库的 `.git/hooks/pre-commit`，所有 worktree 共享。**Why**: Phase 5 E2 中误以为每个 worktree 有独立 hooks，实际是共享的——这导致 worktree 内的 hook 行为依赖于主仓库的安装状态。**How to apply**: 在 worktree 中开发 hook 相关功能时，确认 active hook 是主仓库的 `$MAIN_REPO/.git/hooks/pre-commit`，不是 worktree 内的 `.git/hooks/`（worktree 中 `.git` 是文件，不存在 `hooks/` 子目录）。

### CA-12. `init()` 中缓存的值不能被 `t.Setenv` 覆盖
**Rule**: `init()` 中从环境变量初始化的全局变量（如 `secretReplacer`、`goBinPath`）在测试中不会被 `t.Setenv` 影响——`init()` 在测试运行前已执行完毕。测试需要重新初始化这些变量时，必须提供显式的 reset 函数（如 `initMaskSecrets()`）。**Why**: Phase 5 将 `maskSecrets` 从每次调用 `os.Getenv` 改为 `init()` 时构建 `strings.Replacer`，导致 `TestMaskSecrets` 的 `t.Setenv` 无效——测试通过但实际没有覆盖新逻辑。**How to apply**: 任何 `init()` 中缓存环境变量的包，必须导出 `ResetForTesting()` 或等价函数；测试中先 `t.Setenv` 再调 reset。

### CA-13. Shell 脚本迁移到 Go 的「影子问题」
**Rule**: 迁移 shell 脚本到 Go 时，必须对照原始脚本逐行审计每个步骤——不是每个 gate 名称，而是每个步骤的完整参数和行为（retry 次数、`|| true` 语义、dry-run 路径、环境变量依赖）。**Why**: Phase 5 E3 中 CI 统一 gate 丢失了 3 个行为：`go test` 的 3 次 retry loop、`critic score` 的 `|| true` 软语义、`drift sync --dry-run` 的独立 smoke 路径。这些被 Critic 发现后才补回。**How to apply**: 迁移清单必须包含「旧行为矩阵」：每个步骤的 exit-code 处理（hard/soft）、参数列表、retry 逻辑、环境变量依赖——逐项对账，不允许「看起来差不多」就通过。

### CA-14. `os.Exit()` 在 in-process gate 中会杀死测试进程
**Rule**: in-process gate 调用的函数如果内部有 `os.Exit()`，会直接终止测试进程（不是返回 error），导致测试框架无法捕获失败。这类 gate 必须 shell out 到子进程执行（如 `gateGclAlarmWire` 使用 `exec.Command` 而非 `inProcessGate`）。**Why**: Phase 5 E3 中 `gateGclAlarmWire` 最初使用 `inProcessGate(runGCLAlarmWire, ...)`，而 `runGCLAlarmWire` 内部有 `os.Exit(1)`——测试进程被直接杀死，输出为空，排查困难。**How to apply**: 在 in-process gate 实现前，先 grep 目标函数确认无 `os.Exit`；如有则使用子进程方式（`exec.Command`），并在 gate 注释中标注原因。

### CA-15. GCL 是每次编码任务的强制前置步骤
**Rule**: 任何涉及代码/配置变更的任务，在动手前必须先跑 GCL Auto-Execution Gate 决策树（§GCL Auto-Execution Gate）。满足触发条件 A-E 任一 → 必须启动 GCL 多子 Agent 架构；不允许「先写代码再补 GCL」或「这次变更小，跳过 GCL」。**Why**: Phase 5 中 E1-E4 每批次都通过 GCL 多子 Agent 架构评审，Critic 共发现 1 个 BLOCKER（bin/ gitignored）和 4 个 MAJOR（test retry、alarm-wire、soft gates、drift dry-run），这些在自审中均未被发现。如果没有 GCL 门禁，这些问题会直接合入 main。**How to apply**: 收到编码任务后，第一步对照 §GCL Auto-Execution Gate 的 5 个触发条件判定；满足任一则立即启动 GCL（创建 worktree → 公示模型 → spawn Generator + ≥2 Critics → 循环评审）。

### CA-16. Critic 模型必须强于 Generator
**Rule**: Critic 必须使用比 Generator 更强的模型（不同厂商最优，同厂商更高等级次之）。不能用相同模型做 Generator 和 Critic——同构模型会产生同构盲区，漏掉 Generator 的系统性错误。**Why**: Phase 5 中 E1-E3 的 Critics 使用了与 Generator 不同的模型组合，发现了 Generator 自审无法发现的 BLOCKER（bin/ gitignored）和行为丢失（test retry、alarm-wire 等）。**How to apply**: 启动 GCL 前公示模型配置；Critic ≥2 个，必须包含至少一个不同厂商的旗舰模型。
