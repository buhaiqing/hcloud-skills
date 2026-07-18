# Superpowers — Plans & Specs 索引

> 维护者: 自动追踪
> 更新: 2026-07-18

## 图例

| 标记 | 含义 |
|------|------|
| ✅ | 全部完成，已合并 main |
| 🔶 | 部分完成，存在未解决的 DoD 项 |
| 🔄 | 进行中 / 待执行 |
| ❌ | 未开始 / 存在阻塞问题 |

---

## Plans

| 文件 | 状态 | 说明 |
|------|------|------|
| `plans/aiops-l5-autonomous.md` | ✅ | 18 个 Batch，全部合并 main (2026-07-18) |
| `plans/aiops-optimization.md` | ✅ | L4 成熟度提升，全部合并 main (2026-07-18) |
| `plans/gcl-token-efficiency-p0.md` | ✅ | 3 个 Batch，全部合并 main (−749 行) |
| `plans/skillcheck-b-class.md` | 🔶 | Spec ✅ FINAL_SPEC；DoD 存在未完成项 (Phase 4 清理) |
| `plans/skillcheck-cli.md` | 🔶 | 声称 COMPLETE；但 DoD 有 2 项未完成 |

---

## Specs

| 文件 | 状态 | 对应 Plan | 说明 |
|------|------|-----------|------|
| `specs/aiops-l5-autonomous.md` | 🔄 | `plans/aiops-l5-autonomous.md` | Status: DRAFT — plan 已完成但 spec 仅为草稿，供 review |
| `specs/aiops-optimization.md` | ❌ | `plans/aiops-optimization.md` | **无对应 spec 文件** |
| `specs/gcl-token-efficiency-p0.md` | ✅ | `plans/gcl-token-efficiency-p0.md` | 完成，行数从 6558 降至 5374 |
| `specs/skillcheck-b-class.md` | ✅ | `plans/skillcheck-b-class.md` | FINAL_SPEC，3 轮 self-critique 完成 |
| `specs/skillcheck-cli.md` | ✅ | `plans/skillcheck-cli.md` | FINAL_SPEC；但 DoD 有 2 项未解决 (见下) |

---

## 未完成项详情

### 🔶 skillcheck-cli.md — DoD 未完成

| # | DoD 项 | 状态 | 说明 |
|---|--------|------|------|
| 1 | 三平台二进制可在 Release 下载 | ❌ | B1–B5 合并完成，但未验证 Release 发布 |
| 2 | 干净容器（无 python3）中 `skillcheck validate --root <外部仓库>` 跑通 | ❌ | 未在干净容器环境中验证 |

**影响**: `plans/skillcheck-cli.md` 标记为 COMPLETE 存在虚报。需在真实 Release 后重新验证。

### 🔶 skillcheck-b-class.md — DoD 未完成

**2026-07-18 更新**: Phase 1 部分完成（L1-A ✅ L1-B ✅ L1-C ✅ L5-A ✅ L5-B ✅），其余 Phase 未开始。

| # | DoD 项 | 状态 | 说明 |
|---|--------|------|------|
| 1 | Phase 1~4 全部完成并合并 main | 🔄 | Phase 1 已完成（L1-A ✅ L1-B ✅ L1-C ✅ L5-A ✅ L5-B ✅）；Phase 1.5/2/3/4 未开始 |
| 2 | 8 个 Python 脚本删除 | ❌ | Phase 4（L5-C）待执行 |
| 3 | `check_py310_compat.py` 删除，`ruff.toml` 删除 | ❌ | Phase 4（L5-D）待执行 |
| 4 | `scripts/install_hook.go` + make targets | ✅ | 已完成 (b425927) |
| 5 | `.githooks/pre-commit` 触发条件更新 | ✅ | 已完成 (b425927) |
| 6 | `validate_local.py` 更新 | ❌ | Phase 4（L5-E）待执行 |
| 7 | `docs/manual/skillcheck.md` 用户手册 | ❌ | Phase 4（DOC-1）待执行 |
| 8 | README 中文/英文版更新 | ❌ | Phase 4（DOC-2）待执行 |
| 9 | worktree 全部删除 | 🔄 | feature/bclass-p1a/p1c/p5 已删除；剩余分支待清理 |

### ❌ aiops-optimization — 无对应 Spec

| Plan 文件 | 状态 | Spec 文件 |
|-----------|------|-----------|
| `plans/aiops-optimization.md` | ✅ COMPLETE | **不存在** `specs/aiops-optimization.md` |

---

## 缺失文件

```
docs/superpowers/specs/aiops-optimization.md  ← 不存在（应该有对应 plan）
```

---

## 执行建议

1. **继续**: `skillcheck-b-class.md` — Phase 1 完成！下一个任务：Phase 2 (L2-A~D: 4 路并行校验命令)
2. **继续**: Phase 2 (L2-A~D: 4 路并行校验命令)
3. **继续**: Phase 3 (L3-A~B: safety-class + resource-scope，依赖 L1-A；L4-A~B: gcl run + alarm-wire，依赖 L1-B)
4. **收尾**: Phase 4 清理（L5-C~L5-E: 删除 Python 脚本、更新文档）
5. **验证**: `skillcheck-cli.md` DoD — Release 发布后验证三平台 artifact + 干净容器
6. **可选**: `aiops-l5-autonomous.md` spec 从 DRAFT 推进到 FINAL_SPEC
7. **可选**: `aiops-optimization.md` 是否补 spec（计划已 COMPLETE，价值有限）
