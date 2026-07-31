# Plan: L4 Maturity Upgrade (迈向自主级)

> Status: 📋 **DRAFT** — 待用户审批后进入 implementation
> Created: 2026-07-31
> Target: Agentic AI L4 (Autonomous) — 在已达成 L3 基础上补齐 4 项能力缺口
> Depends on: ADR-0007 (Outcome Memory), ADR-0008 (Context Memory),
> ADR-0009 (Trust), ADR-0010 (Real Executor), ADR-0011 (Cross-Skill Delegation),
> ADR-0012 (Maturity Assessment), `docs/gcl-spec.md`, `internal/l4/`, `internal/gcl/`

## 背景与目标

ADR-0012 评估结论:仓库处于 **L3(协作级)已达成,L4(自主级)框架就绪但被安全刹车**。
L4 两大引擎(L4 orchestrator + GCL)已是**纯 Go 实现**,语言层面无障碍。阻碍 L4 达标的
是 4 项**能力缺口**,全部可用 Go 补完:

1. **持久化确认存储** — 解除 `confirmation.go:11` 的 in-process 限制
2. **非 critical playbook 安全自动执行** — 基于信任分阈值放开高信任/非破坏性操作
3. **信任冷启动策略** — 新 skill/新操作无历史时的先验基线 + 探索期递减监督
4. **端到端无人值守实测** — 至少一个非 trivial 故障的全自动闭环跑通作为证据

**配套(工具链统一,非 L4 能力增量)**:将残留 5 个 shell 脚本中属于工具逻辑的部分
(如 `scripts/pre_commit_check.sh`)并入 `hwcloud-skillcheck` 子命令,落实"工具尽可能用 Go"。

## 前置依赖

| 项 | 状态 | 说明 |
|----|------|------|
| L3 编排/记忆/自愈 | ✅ 已完成 | `internal/l4/` 已落地 |
| GCL 外部 Critic | ✅ 已完成 | `internal/gcl/critic.go:18-29` |
| Trust Phase 4 单一来源 | ✅ 已完成 | ADR-0009 |
| 破坏性操作 HITL 护栏 | ✅ 保留(不可移除) | 安全设计,非 L4 障碍 |

---

## Phase 1: 持久化确认存储 (Durable Confirmation)

### Batch L4-A: ConfirmationRegistry 后端抽象

**文件**:
- `hwcloud-skillcheck/internal/gcl/confirmation.go` (改)
- `hwcloud-skillcheck/internal/gcl/confirmation_store.go` (新 — 定义 `ConfirmationStore` 接口)
- `hwcloud-skillcheck/internal/gcl/confirmation_file.go` (新 — 基于 `.l4-memory/confirmations.json` 的耐久实现)

**内容**:
- 抽出 `ConfirmationStore` 接口(Issue/Verify/VerifyBound/Prune),当前 in-process map 作为 `MemoryStore` 实现
- 新增 `FileStore`:nonce 持久化到 `.l4-memory/`,atomic write,支持跨进程/重启复用
- `Runner` 按配置选择 store(默认仍 Memory,CI 兼容)

**验收标准**:
- `MemoryStore` 行为与原实现 100% 等价(现有 `confirmation_spec_test.go` 全绿)
- `FileStore` 单测覆盖:Issue→重启进程→Verify 仍成功;TTL 过期清除;一次性消费防重放
- 不破坏 `SAFETY_FAIL` 中止语义

---

## Phase 2: 非 critical playbook 安全自动执行

### Batch L4-B: 信任门控的自动执行策略

**文件**:
- `hwcloud-skillcheck/internal/l4/self_healing.go` (改 — 执行决策)
- `hwcloud-skillcheck/internal/l4/rbac.go` (改 — 风险分级接入)
- `hwcloud-skillcheck/internal/l4/trust.go` (读 — 信任分查询)

**内容**:
- 定义自动执行决策:`risk_level ∈ {low, medium}` **且** `TrustScore ≥ threshold` → 允许 auto-execute
- `critical` 仍强制 `RequiresHumanApproval`(保持 ADR-0012 的安全刹车)
- 决策结果写入 outcome memory,作为后续信任更新的输入

**验收标准**:
- 单元/集成测试:高信任+低风险 playbook 在无人工干预下自动执行并自愈
- 高风险/critical 仍触发 `halt` 或确认 nonce(回归测试覆盖)
- 信任阈值可配置,默认保守

---

## Phase 3: 信任冷启动策略

### Batch L4-C: 先验信任基线与探索期

**文件**:
- `hwcloud-skillcheck/internal/l4/trust.go` (改)
- `hwcloud-skillcheck/internal/l4/orchestrator.go` (改 — 探索期监督递减)

**内容**:
- 新 skill/新操作无 outcome 历史时,赋予**保守先验基线**(非 0,但低于成熟 skill)
- 探索期:前 N 次执行强制 HITL 或降级为 dry-run,随成功次数线性放宽监督
- 信任跨 skill 传播:同 product 下相关操作的信任可部分继承

**验收标准**:
- 冷启动 skill 首次执行不自动放行(走确认/监督)
- 连续 K 次成功后信任提升至可自动执行区间(有测试断言)
- 先验基线值有文档出处(不凭空设定)

#### Phase 3 决策记录 (Decision Log)

> 用户授权:方案设计与 N 取值由 agent 自主决定(A/B 测试选优),仅要求决策过程与最终方案存档。

**Scope 取舍 (A vs B)**
- A — 仅探索期门控(在 `trust.go` 内对单 (skill,action) 做连续成功计数封顶风险)
- B — 探索期 + 跨 skill 信任传播(同 product 下相关操作部分继承信任)
- **选定 A**。理由:信任传播(B)引入跨 skill 状态耦合,与 ADR-0009「trust 单一来源 = outcome memory」冲突,且冷启动窗口内传播高风险会放大首因错误;本阶段先交付可验证、可回滚的保守基线,B 留待 Phase 4 实测后评估。

**Design (A 落地)**
- `EvaluateOperationWithHistory(score, skill, action, opRisk, opType, mem)` 包裹 `EvaluateOperation`,叠加线性探索期封顶:
  - `consecutiveSuccessCount(mem, skill, action)` 取最近 `ExplorationWindow` 条记录的尾部连续成功数 `k`
  - `coldStartMaxRisk(k, window)` 阶梯:`k<2→none`, `2≤k<3→low`, `3≤k<window→medium`, `k≥window→""`(空=探索完成,回落到 tier 门控)
  - 仅收紧、绝不放松安全:`EvaluateOperation` 内的 critical/destructive 硬覆盖仍优先
- 调用点 `orchestrator.go:292` 传入真实 `trustSkill`/`trustAction`(修复了初版误传 `score.Level` 导致 `k` 恒为 0 的回归)

**N 取值与出处**
- `ExplorationWindow = 5`,出处 = `HealingPolicy.MinSamples = 5`(`self_healing.go`),作为「样本足以形成可信先验」的可文档化下限,避免凭空设定。

**先验基线**
- `zeroTrustScore() = 0.245 → L0_new`(always-confirm),已满足验收 #1「首次执行不自动放行」,无需新增非零先验常量;探索期门控在上层进一步收紧风险敞口。

**测试断言** (`trust_phase3_test.go`):Ramp 表 (k=0,1→block; k=2 low→allow; k=3 medium→allow; k=5 high→allow)、FirstExecutionBlocked、MatureSkillUnaffected (≥window 不受限)、ConfigOverride。全量 `go test ./...` 绿。

---

## Phase 4: 端到端无人值守实测

### Batch L4-D: 全自动闭环演示/测试

**文件**:
- `hwcloud-skillcheck/cmd/mockhcloud/` (复用现有 mock,注入一个可自愈故障)
- `hwcloud-skillcheck/internal/l4/*_test.go` (新集成测试)

**内容**:
- 构造一个非破坏性故障场景(如 mock 返回告警→playbook 自动诊断→执行→验证→学习)
- 全程零人工干预跑通,作为 L4 达标的**客观证据**
- 输出结构化 trace 到 `audit-results/`,可被 GCL 聚合

**验收标准**:
- 集成测试 `TestL4AutonomousClosedLoop` 在 CI 通过
- trace 含完整 Detect→Diagnose→Execute→Verify→Learn 五段
- 该测试成为 L4 验收的回归护栏

---

## Phase 5: 工具链 Go 化收尾(配套,非 L4 能力增量)

### Batch L4-E: shell 脚本并入 Go

**文件**:
- `hwcloud-skillcheck/cmd/check.go` 或新 `cmd/precommit.go` (承载 `pre_commit_check.sh` 逻辑)
- `scripts/pre_commit_check.sh` (保留为 thin wrapper 或直接弃用)

**内容**:
- 将 pre-commit 校验(go build/test/lint/drift check)封装为 `hwcloud-skillcheck check --pre-commit`
- 落实用户偏好"工具尽可能用 Go"

**验收标准**:
- `hwcloud-skillcheck check --pre-commit` 覆盖原 shell 全部检查项
- 现有 CI/git hook 可无缝切换(或保留 shell 作兼容层)

---

## 验收总览 (L4 Definition of Done)

| # | L4 能力 | 落地标志 | 验证 |
|---|---------|----------|------|
| 1 | 耐久确认 | `FileStore` 跨进程复用 | `confirmation_file_test.go` |
| 2 | 安全自动执行 | 高信任+非破坏 playbook 自动闭环 | `self_healing` 集成测试 |
| 3 | 信任冷启动 | 新 skill 探索期递减监督 | `trust` 单测 |
| 4 | 端到端无人值守 | `TestL4AutonomousClosedLoop` 绿 | CI 集成测试 |
| 5 | 工具 Go 化 | pre-commit 入 Go | `cmd/precommit` 测试 |

## 风险与边界

- **不移除破坏性操作 HITL** — 这是安全护栏,不是 L4 障碍;L4 自主区间限定在
  非破坏性 + 高信任。
- **不为了"更自主"放松 Safety=0 中止** — GCL 安全门禁不可降级。
- 所有改动走 GCL(Generator-Critic-Loop)多子 Agent 评审(>5 行代码变更强制)。

## 下一步

待用户审批本 plan 后,按 Phase 1→5 顺序进入 implementation;每个 Batch 独立 commit +
独立 Critic 评审(遵循 AGENTS.md §16.1 每 CR 独立 commit)。
