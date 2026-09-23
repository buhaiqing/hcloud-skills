# CA-Archive — 退役的复利资产条目

> **目的**：AGENTS.md 顶部的「复利资产」节遵循 CADL「≤12 条」目标，超出会触发 prune。
> 本档案保留 **CA-A11 ~ CA-A14** 的完整踩坑叙事（包括 Why + How to apply），便于需要时回查。
> 这些条目之所以被降级：触达半径过窄（只对 hwcloud-skillcheck 子系统有意义），不符合
> AGENTS.md 的高频阅读场景，但完整故事仍有保留价值。
>
> **权威清单（active CA）**：见 `../AGENTS.md` §复利资产。

---

### CA-A11. Git worktree 的 `.git` 是文件，不是目录

**Rule**: worktree 下 `.git` 是一个指向主仓库 `.git/worktrees/<name>` 的文本文件（不是目录）。因此 worktree 中 `git rev-parse --show-toplevel` 返回的是 worktree 自己的根（正确），但 `.git/hooks/` 是主仓库的 hooks（共享）。安装 hook 时写入主仓库的 `.git/hooks/pre-commit`，所有 worktree 共享。**Why**: Phase 5 E2 中误以为每个 worktree 有独立 hooks，实际是共享的——这导致 worktree 内的 hook 行为依赖于主仓库的安装状态。**How to apply**: 在 worktree 中开发 hook 相关功能时，确认 active hook 是主仓库的 `$MAIN_REPO/.git/hooks/pre-commit`，不是 worktree 内的 `.git/hooks/`（worktree 中 `.git` 是文件，不存在 `hooks/` 子目录）。

### CA-A12. `init()` 中缓存的值不能被 `t.Setenv` 覆盖

**Rule**: `init()` 中从环境变量初始化的全局变量（如 `secretReplacer`、`goBinPath`）在测试中不会被 `t.Setenv` 影响——`init()` 在测试运行前已执行完毕。测试需要重新初始化这些变量时，必须提供显式的 reset 函数（如 `initMaskSecrets()`）。**Why**: Phase 5 将 `maskSecrets` 从每次调用 `os.Getenv` 改为 `init()` 时构建 `strings.Replacer`，导致 `TestMaskSecrets` 的 `t.Setenv` 无效——测试通过但实际没有覆盖新逻辑。**How to apply**: 任何 `init()` 中缓存环境变量的包，必须导出 `ResetForTesting()` 或等价函数；测试中先 `t.Setenv` 再调 reset。

### CA-A13. Shell 脚本迁移到 Go 的「影子问题」

**Rule**: 迁移 shell 脚本到 Go 时，必须对照原始脚本逐行审计每个步骤——不是每个 gate 名称，而是每个步骤的完整参数和行为（retry 次数、`|| true` 语义、dry-run 路径、环境变量依赖）。**Why**: Phase 5 E3 中 CI 统一 gate 丢失了 3 个行为：`go test` 的 3 次 retry loop、`critic score` 的 `|| true` 软语义、`drift sync --dry-run` 的独立 smoke 路径。这些被 Critic 发现后才补回。**How to apply**: 迁移清单必须包含「旧行为矩阵」：每个步骤的 exit-code 处理（hard/soft）、参数列表、retry 逻辑、环境变量依赖——逐项对账，不允许「看起来差不多」就通过。

### CA-A14. `os.Exit()` 在 in-process gate 中会杀死测试进程

**Rule**: in-process gate 调用的函数如果内部有 `os.Exit()`，会直接终止测试进程（不是返回 error），导致测试框架无法捕获失败。这类 gate 必须 shell out 到子进程执行（如 `gateGclAlarmWire` 使用 `exec.Command` 而非 `inProcessGate`）。**Why**: Phase 5 E3 中 `gateGclAlarmWire` 最初使用 `inProcessGate(runGCLAlarmWire, ...)`，而 `runGCLAlarmWire` 内部有 `os.Exit(1)`——测试进程被直接杀死，输出为空，排查困难。**How to apply**: 在 in-process gate 实现前，先 grep 目标函数确认无 `os.Exit`；如有则使用子进程方式（`exec.Command`），并在 gate 注释中标注原因。
