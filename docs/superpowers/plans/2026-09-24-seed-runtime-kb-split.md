# Plan — Seed / Runtime KB Split (2026-09-24)

Spec: `docs/superpowers/specs/2026-09-24-seed-runtime-kb-split-design.md`
Branch: `feature/seed-runtime-kb-split` (worktree `../hcloud-skills-seed-split`)
Execution: GCL — Generator (economic) + 2 blind Critics, ≤3 rounds.

## S1 — Seed 写入 + 合并读写（垂直切片：生成→合并→保存→测试全绿）

阻塞边：无（本片完成即可 demo：gen 只写 seed，loader 合并，save 只写 overlay）。

改动面：
- `internal/learning/knowledge.go`：`WriteSkillAssets` 改写 `failure_patterns.seed.json` / `remediation-playbooks.seed.json`；seed meta 仅 `total_patterns`；新增 check 函数（marker=`docs/gcl-spec.md`，byte-compare，marker 缺失 vacuous pass）。
- `internal/learning/trace.go`：`LoadFailurePatterns` 按 #T3 合并（seed 缺失=现状 scaffold）；`SaveFailurePatterns` 按 #T4 拆写 overlay + `last_aggregation=now`。
- `internal/learning/playbook.go`：`LoadPlaybooks` 合并；`RecordPlaybookOutcome` 结构保持 + seed-only 追加 `{id,metadata}`。
- `cmd/learning.go`：`learning gen [--check]`；help/注释同步 seed 名。
- `cmd/check_precommit.go`：`gateLearningGen` → `gen --check`（label 同步）。
- `cmd/root.go`：help 行同步。

DoD：
- 新增/更新测试全过：#T1(结构) #T3(合并) #T4(拆写/playbook) #T5(gen 不写 overlay + check 三态)。
- 既有 no-seed 形态测试（`playbook_test`、`playbook_feedback_test`、`trace_test` 大部）不改断言即绿。
- 验证：`cd hwcloud-skillcheck && go build ./... && go test ./internal/learning ./cmd -count=1`

## S2 — 消费方接线 + 迁移 + 文档同步

阻塞边：S1。

改动面：
- `internal/learning/knowledge.go` `GeneratePitfallReport` → `LoadFailurePatterns`（#T6）。
- `internal/l4/persistence.go` `readFailurePatternsForSkill` → `learning.LoadFailurePatterns`（#T6；build 验证无 cycle）。
- 迁移（#T7）：跑 `learning gen` 产出 4 组 seed；4 个 Products 的 tracked `remediation-playbooks.json` 置为 `{schema,skill_id,playbooks:[]}`；`git check-ignore` 验证 `.seed.json` 不被 ignore。
- 文档同步：`AGENTS.md` 状态表两行（原位改，控行数）、`references/self-healing-spec.md` §2/§3/§6 提 seed（禁触 §5.x 与 `source_traces_analyzed` 行）、`docs/manual/hwcloud-skillcheck.md` gen 节、`knowledge.go` 包注释（Python baseline 表述改为 Go-only）。
- 测试：pitfall 走合并（#T6）；l4 pre-risk 在 overlay 缺失时命中 seed（fresh-clone 形态，#T6）。

DoD：`go test ./... -count=1` 绿；`./bin/hwcloud-skillcheck validate --root .` 绿（含 doc-contracts）；`learning gen --check` 绿。

## S3 — 全量门禁 + 收口

阻塞边：S2。

- Pre-commit Gate（强制）：`cd hwcloud-skillcheck && go test -race ./... && go vet ./... && gofmt -l .`
- `hwcloud-skillcheck validate --root .`（docs 变更）
- `./bin/hwcloud-skillcheck learning gen --check --root .` + `aggregate trace --require-traces --root .` 冒烟
- Critic 轮次闭环（≤3 轮）后合并回 main，删除 worktree，回写 `findings.md` P0-3 状态。

## 红线（所有子 Agent）

- 只许改本 plan 列出的路径 + 对应 `_test.go`；禁全仓 `gofmt -w`/`--fix`/`git add -A`。
- 禁触：`audit-results/` 既有文件、`.l4-memory/`、`findings.md`/`task_plan.md`/`progress.md`、gitignore、doc-contract 锚点行。
- 提交仅在 feature 分支；每个子 Agent `[max 10 min]`，超时由主 Agent 接管。
