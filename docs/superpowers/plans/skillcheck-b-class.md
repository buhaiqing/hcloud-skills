# Plan: skillcheck B 类迁移 — GCL 契约校验 + 运行时 GCL 编排

> Status: ✅ READY (依赖 docs/superpowers/specs/skillcheck-b-class.md ✅ FINAL_SPEC)
> Last updated: 2026-07-18
> Execution model: **并行 sub-agent + 主 Agent 聚合** (per AGENTS.md orchestrator rule + GCL rules §3)

## Execution Approach

- **并行优先**：无依赖的 task 分配到独立 worktree，由 sub-agent 并发执行
- **串行约束**：有代码依赖的 task 分阶段串行（后一阶段依赖前一阶段的合并结果）
- **TDD 门禁**：每个迁移必须 先写测试 → 再实现 → 测试通过 → 提交
- **等价性验证**：每个迁移后对照 Python 原版运行，确认退出码一致
- **文档同步**：每阶段合并后立即更新 README/manual/AGENTS.md
- **经验沉淀**：每阶段完成后复盘，写入 AGENTS.md Cross-Language Migration Lessons

## 并行调度设计

### 依赖图

```
Phase 1 (3 路并行)                    Phase 2 (4 路并行)             Phase 3 (2 路并行)          Phase 4 (串行)
┌─────────────────┐                   ┌─────────────────┐            ┌─────────────────┐         ┌─────────────────┐
│ L1-A sanitizer  │                   │ L2-A gcl-conform│ ← 无依赖   │ L3-A safety-cls │ ← 依赖  │ L5-C~L5-E 清理  │
│ L1-C alarm_wire │ ← 互不依赖        │ L2-B gen-contrac│ ← 无依赖   │ L3-B resource-sc│ L1-A   │ ← 需要前 3 阶段  │
│ L5-A install_hk │ ← 互不依赖        │ L2-C audit-res  │ ← 无依赖   │                 │         │ 全部合并后才做   │
│ L5-B pre-commit │                   │ L2-D alarm-wire │ ← 无依赖   │ L4-A gcl run    │ ← 依赖  │                  │
└────────┬────────┘                   └─────────────────┘            │ L4-B gcl alarm   │ L1-B/C │                  │
         │                                                           └─────────────────┘         └─────────────────┘
         ▼ (L1-A → L1-B 串行)                                              ▲
    L1-B runner ← 依赖 L1-A sanitizer                                     │
         │                                                                │
         └──── 4 路并行（L1-B + L2 的 3 个 = 4 个 worktree）──────────────┘
```

### 执行时序

```
Phase 1 (3 worktrees 并行):
  Worktree 1: L1-A (sanitizer)        [独立，~10min]
  Worktree 2: L1-C (alarm_wire)       [独立，~10min]
  Worktree 3: L5-A + L5-B (infra)     [独立，~10min]
  → 全部完成后合并到 main

Phase 1.5 (1 worktree):
  Worktree 4: L1-B (runner)           [依赖 L1-A，~10min]
  → 合并到 main

Phase 2 (4 worktrees 并行):
  Worktree 5: L2-A (gcl-conformance)  [独立，~10min]
  Worktree 6: L2-B (gen-contract)     [独立，~10min]
  Worktree 7: L2-C (audit-results)    [独立，~10min]
  Worktree 8: L2-D (alarm-wire-contr) [独立，~10min]
  → 全部完成后合并到 main

Phase 3 (2 worktrees 并行):
  Worktree 9: L3-A + L3-B (safety-class + resource-scope)  [依赖 L1-A，~15min]
  Worktree 10: L4-A + L4-B (gcl run + gcl alarm-wire)      [依赖 L1-B/C，~15min]
  → 全部完成后合并到 main

Phase 4 (串行):
  删除 Python 原版 + 更新 validate_local.py + 更新文档 + 经验沉淀
```

## 任务拆分 (Task Breakdown)

### Phase 1: 共享库 + 基础设施 (3 路并行)

| Worktree | Task | 文件 | TDD 验证 | 预估 |
|----------|------|------|---------|------|
| WT-1 | L1-A: `internal/gcl/sanitizer.go` — safety_class 枚举 + sanitize_operation_intent + mask_resource_id | `sanitizer.go`, `sanitizer_test.go` | 合法/非法 safety_class 各 3 用例；ID 掩码 5 格式；空/未知输入 | 10min |
| WT-2 | L1-C: `internal/gcl/alarm_wire.go` — CES 告警计划生成 + 阈值验证 | `alarm_wire.go`, `alarm_wire_test.go` | plan 结构正确；阈值边界值；配置漂移检测 | 10min |
| WT-3a | L5-A: `scripts/install_hook.go` — Go 版 git hook 安装器 | `install_hook.go` | --check/--uninstall 路径 | 8min |
| WT-3b | L5-B: 更新 `.githooks/pre-commit` — 触发条件改为 Go 文件 | `.githooks/pre-commit` | bash -n 验证 | 2min |

### Phase 1.5: 依赖链 (1 worktree)

| Worktree | Task | 文件 | TDD 验证 | 预估 |
|----------|------|------|---------|------|
| WT-4 | L1-B: `internal/gcl/runner.go` — GCL 循环核心 (Generator → Critic → Loop → Trace) | `runner.go`, `runner_test.go` | 单轮 PASS / RETRY / SAFETY_FAIL / MAX_ITER 路径 | 10min |

### Phase 2: 简单校验 (4 路并行)

| Worktree | Task | 对应 Python | 文件 | 验证 |
|----------|------|------------|------|------|
| WT-5 | L2-A: `validate gcl-conformance` | `check_gcl_conformance.py` | `cmd/validate_gcl.go` | 退出码 + FAIL 列表一致 |
| WT-6 | L2-B: `validate generator-contract` | `check_generator_contract.py` | `cmd/validate_contract.go` | 退出码一致 |
| WT-7 | L2-C: `check audit-results` | `check_audit_results_guard.py` | `cmd/check.go` (追加) | 退出码一致 |
| WT-8 | L2-D: `validate alarm-wire-contract` | `check_gcl_alarm_wire_contract.py` | `cmd/validate_gcl.go` | 退出码 + 错误信息一致 |

### Phase 3: 依赖共享库的校验 + 运行时 (2 路并行)

| Worktree | Task | 依赖 | 文件 | 验证 |
|----------|------|------|------|------|
| WT-9 | L3-A + L3-B: `validate safety-class` + `validate resource-scope` | L1-A (sanitizer) | `cmd/validate_contract.go` | 退出码一致 |
| WT-10 | L4-A + L4-B: `gcl run` + `gcl alarm-wire` CLI 路由 | L1-B (runner), L1-C (alarm_wire) | `cmd/gcl_run.go`, `cmd/gcl_alarm_wire.go` | smoke test |

### Phase 4: 清理 + 文档 (串行)

| # | Task | 描述 | 输出 |
|---|------|------|------|
| L5-C | 删除 8 个 Python 源脚本 + 测试文件 | `git rm scripts/check_*.py ...` | commit |
| L5-D | 删除 `check_py310_compat.py` + `ruff.toml` | 不再需要 Python 工具链 | commit |
| L5-E | 更新 `validate_local.py` | 移除已删脚本引用 | commit |
| DOC-1 | 更新 `README.md` + `README_CN.md` | 反映新命令体系 | commit |
| DOC-2 | 更新 `docs/manual/*.md` | 用户手册加入新子命令 | commit |
| DOC-3 | 更新 `docs/superpowers/plans/skillcheck-b-class.md` | 标记为 COMPLETE | commit |
| LEARN | 复盘沉淀 → AGENTS.md | 写入迁移经验教训 | commit |

## 文档更新计划 (DOC)

### DOC-1: README 更新

- `README.md` — 更新 "Local validation" 章节移除 Python 引用，更新 skillcheck 命令列表加入新增子命令
- `README_CN.md` — 同上

### DOC-2: 用户手册 (docs/manual/)

创建/更新 `docs/manual/skillcheck.md`，包含：

```
# skillcheck 用户手册

## 安装
## 快速开始
## 命令参考
### validate 子命令
  - schema / frontmatter / eval-queries / product-assessment (已有)
  - gcl-conformance / alarm-wire-contract / safety-class / resource-scope / generator-contract (新增)
### check 子命令
  - example-config / markdown-links / references-links / advanced-coverage (已有)
  - audit-results (新增)
### gcl 子命令 (新增命名空间)
  - run: GCL 执行循环
  - alarm-wire: CES 告警编排
## 等价性测试
## 常见问题
```

### DOC-3: AGENTS.md 经验沉淀

每次 Phase 完成后复盘，记录以下内容到 `AGENTS.md` 的 `Cross-Language Migration Lessons` 章节：

```
### 8. Parallel Migration Strategy
### 9. Phase Dependency Management
### 10. Equivalence Testing with Live Python Scripts
```

## 等价性验证

每个迁移完成后，在 worktree 中运行对照测试：

```bash
# Python 原版输出
python3 scripts/check_gcl_conformance.py --json > /tmp/py_result.json

# Go 版输出 (先构建)
go build -C skillcheck -trimpath -o bin/skillcheck .
./skillcheck/bin/skillcheck validate gcl-conformance --json > /tmp/go_result.json

# 对比退出码 + 关键字段
diff <(echo "exit: $?") <(echo "exit: $?")
```

结构化等价规则（同 skillcheck-cli.md spec）：
- **退出码**：Python fail → Go must also fail (no false negatives)
- **失败项集合**：FAIL 行集合一致（路径归一化后）
- **严格度**：Go 可以比 Python 更严格（接受 false positives）

## 风险与对策

| 风险 | 对策 |
|------|------|
| API 限流 (429) 导致 sub-agent 失败 | 主 Agent 直接接管实现，不依赖 sub-agent |
| Worktree 之间依赖同步 | 每个 Phase 先合并到 main，再从 main 创建新 worktree |
| `gcl run` 依赖 hcloud CLI 环境 | 单元测试用 mock 子进程 |
| `gcl alarm-wire apply` 有破坏性 | `plan` 只读；`apply` 默认 dry-run |
| 等价性测试输出格式不一致 | 结构化等价（退出码 + 关键字段） |
| 迁移期间 Python 和 Go 双轨运行 | validate_local.py 优先用 Go，Python 对照保留到 Phase 4 |

## Definition of Done

- [ ] Phase 1 ~ Phase 4 全部完成并合并 main
- [ ] 8 个 Python 脚本删除，`check_py310_compat.py` 删除，`ruff.toml` 删除
- [ ] `scripts/install_hook.go` + `make install-hook` / `make check-hook` / `make uninstall-hook` 可用
- [ ] `.githooks/pre-commit` 触发条件更新为 Go 文件
- [ ] `validate_local.py` 不再引用已删 Python 脚本
- [ ] `make all` 通过，`make self-check` 通过
- [ ] 等价性测试覆盖所有迁移子命令
- [ ] `docs/manual/skillcheck.md` 用户手册更新
- [ ] README 中文/英文版更新
- [ ] AGENTS.md 新增迁移经验沉淀
- [ ] worktree 全部删除，无残留分支
