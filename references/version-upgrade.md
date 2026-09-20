# 版本升级规则

重大功能重构或实现完成后，Git push 成功后必须升级版本：

## 触发条件

完成了以下任一工作后：

- 新增了工具子命令（`hwcloud-skillcheck` 新增 `pitfall-report` 等）
- 新增了 `internal/` 包（新的可复用模块）
- 重构了核心 GCL / L4 / learning 逻辑
- 删除了废弃的 Python 脚本或旧逻辑
- 任何影响 `hwcloud-skills` 对外行为的功能变更

## 操作步骤

```bash
# push 完成后，在仓库根目录执行：
task release VERSION=X.Y.Z
```

`task release` 会 `git tag` + `git push origin <tag>`，触发 CI 构建和 GitHub Release。

## 版本号规范

遵循语义化版本（semver）：

- `X.Y.Z`：主版本.次版本.补丁版本
- 主版本：破坏性 API 变更
- 次版本：新功能向后兼容
- 补丁版本：Bug 修复或小改进

> 日常提交（文档、测试用例、typo 修复等）**不需要**升级版本。
