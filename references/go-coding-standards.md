# Go 编码规范 (P0 — 强制)

> 适用于 `hwcloud-skillcheck/` 及 hcloud-skills 仓库所有 Go 代码。
> 用户级通用规范见 `~/.codebuddy/rules/`（`coding-standards.md`、`pythonic_refactoring.md` 等）。
> 本文件是 hcloud-skills 项目特定的 Go 编码补充，对齐 AGENTS.md 顶层原则。

## G1. 可测试性

| 规则 | 要点 |
|------|------|
| **G1.1** 依赖注入优先 | 外部依赖（文件系统、网络、CLI）通过 interface/函数参数注入，不硬编码 `os.Stat` / `exec.Command` |
| **G1.2** 纯函数与副作用分离 | 计算逻辑与 I/O 分开；纯函数返回结果，调用方负责输出 |
| **G1.3** 最小导出面 | 测试辅助函数用 `export_test.go` 暴露内部符号，避免污染公开 API |
| **G1.4** Table-driven tests | 多输入/多场景用 `[]struct{name, input, want}` 模式，一行一个 case |
| **G1.5** Fake over mock | 优先用 fake（内存实现）而非 mock（行为验证）；fake 不耦合实现细节 |
| **G1.6** TDD 红-绿-重构 | 新功能先写失败测试 → 最小实现通过 → 重构优化；红测试不交付 |

## G2. 错误处理

| 规则 | 要点 |
|------|------|
| **G2.1** 错误必须处理 | `err` 不能忽略；用 `_ =` 显式标注有意忽略，且必须有注释说明原因 |
| **G2.2** 错误包装 | 用 `fmt.Errorf("context: %w", err)` 保留调用链；不用 `errors.Wrap` |
| **G2.3** Sentinel errors | 包级 `var ErrXxx = errors.New(...)` 供调用方 `errors.Is` 判断 |
| **G2.4** 不吞错误 | `_ = cmd.Run()` 仅在 gate 明确需要 ignore exit code 时允许（如 soft gate），且必须有注释说明 |
| **G2.5** panic 禁区 | 库代码禁止 `panic`；仅 `init()` 和不可恢复的启动错误可用 |

## G3. 并发安全

| 规则 | 要点 |
|------|------|
| **G3.1** goroutine 生命周期明确 | 每个 `go func()` 必须有对应的 `wg.Wait()` / `ctx.Done()` / channel close |
| **G3.2** 共享状态加锁 | 多个 goroutine 写同一变量 → `sync.Mutex`；读多写少 → `sync.RWMutex` |
| **G3.3** channel 所有权 | 创建者负责 close；接收方用 `for range` 或 `v, ok := <-ch` |
| **G3.4** `-race` 必过 | CI 和本地开发都跑 `go test -race ./...`；race 红线 = 不允许合入 |
| **G3.5** 避免 goroutine 泄漏 | context 取消后子 goroutine 必须退出；用 `context.Context` 传播取消信号 |

## G4. 资源管理

| 规则 | 要点 |
|------|------|
| **G4.1** defer close | 文件、连接、子进程 stdout/stderr pipe 必须 `defer Close()` |
| **G4.2** 子进程清理 | `exec.Command` 创建的子进程必须等待（`cmd.Wait()` 或 `cmd.Run()`），不留下僵尸进程 |
| **G4.3** buffer 复用 | 高频路径用 `sync.Pool` 复用 `bytes.Buffer`；低频路径直接用局部变量 |
| **G4.4** 路径安全 | 用户输入的路径参数用 `filepath.Clean` + 验证在预期目录内；防止路径穿越 |

## G5. 性能

| 规则 | 要点 |
|------|------|
| **G5.1** 避免 init-time 重复计算 | 常量/缓存用 `sync.Once` 或包级 `init()` 初始化；不在热路径调 `exec.LookPath` |
| **G5.2** 字符串拼接 | 循环内用 `strings.Builder`；少量拼接用 `+` 或 `fmt.Sprintf` |
| **G5.3** 避免不必要的分配 | `[]byte` 和 `string` 转换只在必要时做；`bytes.Contains` 优于 `strings.Contains(string(b))` |
| **G5.4** map/slice 预分配 | 已知大小的 map 用 `make(map[K]V, hint)`；slice 用 `make([]T, 0, cap)` |
| **G5.5** Benchmark | 性能敏感的代码路径必须有 `BenchmarkXxx` 测试 |
| **G5.6** init-time 缓存 + 测试重置 | `init()` 中缓存的环境变量/路径必须有导出重置函数供测试使用；测试中 `t.Setenv` 不影响 `init()` 已缓存的值 |

## G6. 可扩展性

| 规则 | 要点 |
|------|------|
| **G6.1** interface 小 | 单一职责，1-3 方法；大 interface 拆分为组合 |
| **G6.2** 配置结构化 | 用 struct 传递选项（functional options pattern），不用 positional bool 参数（≥3 个 bool → struct） |
| **G6.3** 新 gate 零修改注册 | 新增 gate 只需在 `preCommitGates` 追加一行 + 实现 gate 函数；不修改 runner/exit-contract |
| **G6.4** 避免 init 顺序依赖 | 不同文件的 `init()` 不应相互依赖；复杂初始化用显式 `Initialize()` 函数 |

## G7. 代码组织

| 规则 | 要点 |
|------|------|
| **G7.1** 文件行数 | 单文件 ≤ 500 行；超过则按职责拆分 |
| **G7.2** 函数行数 | 单函数 ≤ 50 行（不含测试）；超过则提取辅助函数 |
| **G7.3** 导出最小化 | 仅跨包使用的符号导出；包内共享用小写 |
| **G7.4** 注释规范 | 导出符号必须有 doc comment（`// SymbolName does X.`）；注释写 WHY 不写 WHAT |
| **G7.5** import 分组 | stdlib → 第三方 → 内部包，组间空行分隔 |

## G8. 输入验证

| 规则 | 要点 |
|------|------|
| **G8.1** 公开函数验证参数 | 所有导出函数的参数必须验证（nil check、空字符串、非法值） |
| **G8.2** 防御性编程 | `path.Join` 前验证非空；`slice[i]` 前验证 `i < len` |
| **G8.3** 用户输入净化 | CLI flag 值在传递前验证格式/范围；`filepath.Clean` 防路径穿越 |
| **G8.4** 空字符串即非法 | `--root ""` 等空值参数必须返回明确错误，不得静默 fallback 到 cwd 或其他默认值 |

## G9. TDD 工作流

```
1. RED   — 写一个最小失败测试（验证新行为/边界条件/错误路径）
2. GREEN — 写最小代码让测试通过（不提前抽象、不添加测试未覆盖的逻辑）
3. REFACTOR — 消除重复、改善命名、提取辅助函数（测试必须保持绿）
4. REVIEW — 对照规范自查：G1-G8 是否满足？
```

**门禁**：`go test -race ./...` 全绿 + `go vet ./...` 零 warning 才允许提交。

## G10. Shell→Go 迁移专项

> Phase 5 (pre-commit gate Go migration) 经验沉淀。

| 规则 | 要点 |
|------|------|
| **G10.1** 行为矩阵对照 | 迁移前列出旧脚本每个步骤的完整行为矩阵：命令、参数、exit-code 处理（hard/soft）、retry 逻辑、环境变量依赖。逐项对账，不允许「看起来差不多」。 |
| **G10.2** `os.Exit` 检测 | in-process gate 实现前 grep 目标函数确认无 `os.Exit`；如有则 shell out 到子进程（`exec.Command`），并在注释中标注原因。 |
| **G10.3** `|| true` → `soft: true` | shell 的 `|| true` 语义必须显式映射为 gate 的 `soft: true` 字段；不依赖「exit code 0」或「ignore error」隐含 soft。 |
| **G10.4** Retry 逻辑保持 | shell 中的 retry loop（如 `for i in 1 2; do ...; done`）必须保留为 Go 的显式重试逻辑；不简化为单次调用。 |
| **G10.5** CI-only vs local 分化 | 如果 CI 和本地行为不同，用显式 flag（如 `--check-only`）区分模式，不在代码中硬编码环境检测（如 `if os.Getenv("CI") != ""`）。 |
| **G10.6** 删除旧脚本 | 迁移完成后立即删除旧 shell 脚本（不保留 thin wrapper）；更新所有引用文档；`grep` 验证零残留。 |

