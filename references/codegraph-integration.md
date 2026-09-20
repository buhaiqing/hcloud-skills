# CodeGraph Integration — 代码变动即时同步

CodeGraph (`codegraph` CLI) 维护仓库知识图谱。本仓库已配置 MCP Server（`.mcp.json`），Agent 启动时自动获得 `codegraph_explore` 工具。索引数据位于全局 `~/`.omo/codegraph/`（仓库内 `.codegraph` 为软链，已被 `.gitignore` 忽略）。

## MANDATORY: CodeGraph sync 纪律

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

## GoLang 程序集成规范（hwcloud-skillcheck 等 Go 工程）

`codegraph` 的索引基于 AST/调用图，对 Go 的符号命名有固定约定。在 Go 工程中集成或排查 CodeGraph 时必须遵守：

1. **符号记法** — Go 符号用 `pkg.Symbol`（包路径末段 + 导出符号），例如 `internal/l4.TrustScore`、`internal/l4.EvaluateOperationWithHistory`。`codegraph explore` 入参区分大小写，仅索引导出符号（首字母大写）。
2. **编译先行** — 任何 `codegraph explore/impact/callees` 针对 Go 符号前，先确保 `go build ./...` 通过。索引器解析依赖 AST，**编译失败 → 符号缺失 → 假阴性**。
3. **写后 sync 强约束** — 修改 `internal/` 下任何 Go 文件（含 `_test.go`）后、提交前必须 `codegraph sync --quiet`。Go 的接口实现/动态派送（如 `Executor` interface、`HealingPolicy`）只在 sync 后才反映到调用图。
4. **影响面分析优先于 grep** — 改 `internal/l4/` 等核心包前，先 `codegraph impact <pkg.Symbol>` 拿到真实调用方（含间接调用者），再决定是否需 cascade 修改；禁止仅凭 `grep` 判定「无调用方」。
5. **vendor / 离线** — 沙箱无公网时 `codegraph sync` 可能拉取失败；此时回退到 `grep`/`read` 并标注 `// OFFLINE: codegraph unavailable`，不得假设索引存在。
