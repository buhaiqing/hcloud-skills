# Plan: skillcheck B 类迁移 — GCL 契约校验 + 运行时 GCL 编排

> Status: 📝 READY (依赖 docs/superpowers/specs/skillcheck-b-class.md ✅ FINAL_SPEC)
> Last updated: 2026-07-18
> Execution model: **subagent-driven-development** (per AGENTS.md orchestrator rule)

## Execution Approach

- 本任务为多阶段、多文件、跨 Go/Python 迁移，触发 AGENTS.md 的 Orchestrator 规则
- 采用 `superpowers:subagent-driven-development`：主 Agent 拆任务 → 每个 task 在独立 git worktree 中由 sub-agent 实现（TDD + GCL）→ 主 Agent 合并
- **worktree 策略**：每个 batch 独立 `git worktree add ../hcloud-skills-bclass-<n> -b feature/bclass-<n>`
- **TDD 门禁**：每个迁移必须先写测试，再实现，确保测试通过
- **等价性验证**：每个迁移完成后，对照 Python 原版运行，确认退出码一致

## Task Breakdown

### Layer 1: 共享库 (Foundation)

| # | Task | 文件 | TDD 验证 |
|---|------|------|---------|
| L1-A | `internal/gcl/sanitizer.go` — safety_class 枚举 + sanitize_operation_intent | `sanitizer.go`, `sanitizer_test.go` | 合法/非法 safety_class 各 3 用例；ID 掩码 5 格式 |
| L1-B | `internal/gcl/runner.go` — GCL 循环核心 (Generator → Critic → Loop → Trace) | `runner.go`, `runner_test.go` | 单轮 PASS / RETRY / SAFETY_FAIL 路径 |
| L1-C | `internal/gcl/alarm_wire.go` — CES 告警计划/应用 | `alarm_wire.go`, `alarm_wire_test.go` | plan 输出结构正确；阈值验证 |

### Layer 2: 简单校验 (Static Validation — Simple)

| # | Task | 对应 Python | 文件 | TDD 验证 |
|---|--------|------------|------|---------|
| L2-A | `validate gcl-conformance` | `check_gcl_conformance.py` | `cmd/validate_gcl.go` | 通过/不通过 skill 各 1 |
| L2-B | `validate generator-contract` | `check_generator_contract.py` | `cmd/validate_contract.go` | 完整/缺失文件各 1 |
| L2-C | `check audit-results` | `check_audit_results_guard.py` | `cmd/check.go` (追加) | gitignore/权限/tracked 各维度 |
| L2-D | `validate alarm-wire-contract` | `check_gcl_alarm_wire_contract.py` | `cmd/validate_gcl.go` | 配置一致/不一致 |

### Layer 3: 依赖共享库的校验

| # | Task | 对应 Python | 依赖 | 文件 |
|---|--------|------------|------|------|
| L3-A | `validate safety-class` | `check_safety_class_enum.py` | sanitizer.go | `cmd/validate_contract.go` |
| L3-B | `validate resource-scope` | `check_resource_scope_pii.py` | sanitizer.go | `cmd/validate_contract.go` |

### Layer 4: 运行时命令

| # | Task | 对应 Python | 依赖 | 文件 |
|---|--------|------------|------|------|
| L4-A | `gcl run` 子命令 + CLI 路由 | `gcl_runner.py` | runner.go | `cmd/gcl_run.go` |
| L4-B | `gcl alarm-wire` 子命令 + CLI 路由 | `gcl_alarm_wire.py` | alarm_wire.go | `cmd/gcl_alarm_wire.go` |

### Layer 5: 基础设施 + 清理

| # | Task | 描述 |
|---|------|------|
| L5-A | `scripts/install_hook.go` | Go 版 git hook 安装器 + `make install-hook` |
| L5-B | pre-commit hook 更新 | 触发条件改为 Go 文件 |
| L5-C | 删除 Python 原版 | 删除 8 个源脚本 + 测试文件 |
| L5-D | 删除 `check_py310_compat.py` + `ruff.toml` | 不再需要 |
| L5-E | 更新 `validate_local.py` | 移除已删脚本引用，改为全部 skillcheck |

## Batch → Worktree 映射

| Batch | Tasks | Branch | 文件数 | 预估时长 |
|-------|-------|--------|--------|---------|
| B-B1 (共享库) | L1-A, L1-B, L1-C | `feature/bclass-b1` | 6 | 20-30min |
| B-B2 (简单校验) | L2-A, L2-B, L2-C, L2-D | `feature/bclass-b2` | 4 | 20-30min |
| B-B3 (依赖校验) | L3-A, L3-B | `feature/bclass-b3` | 2 | 15-20min |
| B-B4 (运行时) | L4-A, L4-B | `feature/bclass-b4` | 4 | 20-30min |
| B-B5 (清理) | L5-A, L5-B, L5-C, L5-D, L5-E | `feature/bclass-b5` | ~10 | 15-20min |

每 batch 独立 worktree，sub-agent 完成即 commit，主 Agent 验证后合并下一 batch。

## 等价性验证

每个迁移完成后，在 worktree 中运行：

```bash
# 对比 Python 原版 (仍存在时)
python3 scripts/check_gcl_conformance.py --json > /tmp/py_result.json
./skillcheck/bin/skillcheck validate gcl-conformance --json > /tmp/go_result.json
diff <(jq -S . /tmp/py_result.json) <(jq -S . /tmp/go_result.json)
```

## 风险与对策

| 风险 | 对策 |
|------|------|
| `gcl run` 依赖 hcloud CLI 环境 | 单元测试用 mock 子进程，CI 用 `--structural-critic-only` 跳过外部命令 |
| `gcl alarm-wire apply` 有破坏性 | `plan` 子命令只读；`apply` 要求 `--dry-run` 默认开启，非 dry-run 需要确认 |
| 等价性测试中 Python 输出格式与 Go 不完全一致 | 结构化等价（退出码 + 关键字段），不要求逐字节匹配 |
| 迁移期间 Python 脚本和 Go 版同时存在 | validate_local.py 优先用 Go 版，Python 版作为对照保留到 B-B5 删除 |

## Definition of Done

- [ ] B-B1 ~ B-B5 全部合并到 main
- [ ] 8 个 Python 脚本删除，`check_py310_compat.py` 删除
- [ ] `scripts/install_hook.go` + `make install-hook` 可用
- [ ] `validate_local.py` 不再引用已删 Python 脚本
- [ ] `make all` 通过，`make self-check` 通过
- [ ] 等价性测试覆盖所有迁移子命令
- [ ] 用户手册更新
- [ ] AGENTS.md 新增迁移经验沉淀
- [ ] worktree 全部删除，无残留分支