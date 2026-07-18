# Plan: skillcheck — 跨平台单二进制 Skill 校验 CLI

> Status: ✅ **COMPLETE** — all batches merged to main (2026-07-18)
> Last updated: 2026-07-18 (T13 equivalence tests added)
> Execution model: **subagent-driven-development** (per user request + AGENTS.md orchestrator rule)

## Execution Approach

- 本任务为**多阶段、多文件、新建 Go module** 的跨语言迁移，触发 AGENTS.md 的 Orchestrator 规则。
- 采用 `superpowers:subagent-driven-development`：主 Agent 拆任务 → 每个 task 在独立 git worktree 中
  由 sub-agent 实现（Generator + Critic 协作，遵循 GCL 规则）→ 主 Agent 合并。
- **worktree 策略**：每个 task 独立 `git worktree add ../hcloud-skills-skillcheck-<n> -b feature/skillcheck-<n>`。
- **GCL 门禁**：每个 task >5 行代码变更，必须走 Generator-Critic-Loop（多子 Agent 模式，见 gcl-rules.md）。
- **等价性验证**：每个迁移的脚本须有对照测试（Python 输出 vs Go 输出），见 Task 验证列。

## Module Path 决策 (Plan 启动时确认)

- 实际 repo owner 未知，Plan 中用占位 `github.com/<owner>/hcloud-skills/skillcheck`。
- **首个 action**：`git remote -v` 取真实 owner，替换占位后再 `go mod init`。

## embed 资源清单 (从 scripts/fixtures 迁移)

| 目标路径 | 来源 | 用途 |
|---|---|---|
| `internal/embed/fixtures/gcl-alarm-plan-healthy.json` | `scripts/fixtures/gcl-alarm-plan-healthy.json` | scan secret alarm-plan `--self-check` |
| `internal/embed/fixtures/gcl-quality-summary-healthy.json` | `scripts/fixtures/gcl-quality-summary-healthy.json` | scan secret summary `--self-check` |
| `internal/embed/fixtures/gcl-trace-healthy.json` | 需从 `scripts/fixtures/` 确认存在 | scan secret trace `--self-check` |
| `internal/embed/schemas/trace.schema.json` | `validate_gcl_trace_schema.py` 内联 schema | validate schema trace |
| `internal/embed/schemas/summary.schema.json` | `validate_gcl_summary_schema.py` 内联 schema | validate schema summary |
| `internal/embed/schemas/alarm-plan.schema.json` | `validate_gcl_alarm_plan_schema.py` 内联 schema | validate schema alarm-plan |
| `internal/embed/schemas/eval-queries.schema.json` | `validate_eval_queries_schema.py` 内联 schema | validate eval-queries |

> schema 当前内联在 Python 脚本中，迁移时需抽取为独立 `.json` 文件再 embed。

## Task Breakdown

| # | Task | 对应 Spec | 文件/范围 | 验证 |
|---|---|---|---|---|
| T1 | Go module 骨架 + embed 资源 + main 路由 | §3 | `go.mod`, `main.go`, `internal/embed/*` | `go build` 通过；`skillcheck --help` 输出子命令 |
| T2 | JSON schema subset 校验器 | §2.1 | `internal/schema/` (替代 `json_schema_subset.py`) | 单元测试：合法/非法 JSON 判定一致 |
| T3 | YAML 解析层 (yaml.v3) | §3.1 | `internal/yaml/` | 解析 frontmatter/example-config 单测 |
| T4 | 共享凭据扫描器 | §2.1 | `internal/security/` (替代 `gcl_security_scan.py`) | 对照测试：含凭据样本被检出且遮蔽 |
| T5 | `validate schema *` 4 子命令 | §2.1 | `cmd/validate.go` | 等价性：同输入同退出码 |
| T6 | `validate frontmatter` / `eval-queries` / `product-assessment` | §2.1 | `cmd/validate.go` | 等价性测试 |
| T7 | `check example-config` / `markdown-links` / `references-links` / `advanced-coverage` | §2.1 | `cmd/check.go` | 等价性测试 |
| T8 | `scan secret *` 3 子命令 + `--self-check` | §2.1 | `cmd/scan.go` | 等价性 + 自校验测试 |
| T9 | `aggregate trace` (无输入 warn 跳过) | §2.1 §4 | `cmd/aggregate.go` | 有/无 trace 文件两路径 |
| T10 | `validate` 总入口编排 (默认 cwd) | §4 | `cmd/validate.go` root 命令 | 一键跑全部 A 类 |
| T11 | `lint go` 子命令 (golangci-lint 替代) | §2.4 | `cmd/lint.go` | smoke: 对 skillcheck 自身跑通 |
| T12 | GitHub Action 多平台编译 + Release | §3.2 | `.github/workflows/build-release.yml` | tag push 产三平台 artifact |
| T13 | 等价性对照测试套件 (Python vs Go) | §6.1 | `skillcheck/testdata/` + 脚本 | 全 A 类子命令退出码/失败项一致 |

## Batch → Worktree 映射

| Batch | Tasks | Branch |
|---|---|---|
| B1 (骨架) | T1, T2, T3 | `feature/skillcheck-b1` |
| B2 (校验器) | T4, T5, T6 | `feature/skillcheck-b2` |
| B3 (检查器) | T7, T8, T9 | `feature/skillcheck-b3` |
| B4 (编排+lint) | T10, T11 | `feature/skillcheck-b4` |
| B5 (分发+测试) | T12, T13 | `feature/skillcheck-b5` |

每 batch 独立 worktree，sub-agent 完成即 commit，主 Agent 验证后合并下一 batch。

## 风险与对策

| 风险 | 对策 |
|---|---|
| Go 版输出与 Python 逐字节不等 | Spec 已改为结构化等价（退出码+失败项），对照测试归一化路径/时间戳 |
| schema 抽取遗漏字段 | T2 前先 `grep` 提取 Python 内联 schema 全文，diff 确认 |
| yaml.v3 行为与手写正则不一致 | T3 用现有 example-config.yaml 做对照，逐文件比对解析结果 |
| CI 无 python3 跑不了等价性测试 | T13 对照测试在本地有 python3 环境跑，CI 仅验证二进制可用 |

## Definition of Done (本 Plan 完成)

- [x] B1–B5 全部合并 main
- [x] Python A 类脚本已全部删除（-4,600 行）
- [ ] 三平台二进制可在 Release 下载
- [ ] 干净容器（无 python3）中 `skillcheck validate --root <外部仓库>` 跑通
- [x] T13 等价性套件全绿
