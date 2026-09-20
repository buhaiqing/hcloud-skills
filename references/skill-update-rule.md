# Skill Update Rule: 2-Round Self-Reflection

**After every skill update or creation, execute 2 mandatory self-reflection rounds and auto-fix all discovered issues before finishing.**

## Round 1 — Foundation Check

1. **FinOps**: Are cost patterns actionable? Billing model comparison present? Idle detection documented?
2. **SecOps**: IAM permissions minimum documented? Credential masking enforced? Network isolation?
3. **AIOps**: Multi-metric correlation defined? Delegation matrix present? Knowledge base populated?

### Round 1, Item 4 — Token Efficiency (C6 — MUST PASS)

**必检项**：TE-1~TE-7 是否全部满足（见 Token Efficiency Requirements）？未满足则 **BLOCK**。

| TE 规则 | 检查方法 | 不通过则 |
|---------|---------|---------|
| TE-1 | 检查 references/ 中是否有硬编码的版本号/配额数字 | 替换为 `hcloud` 查询命令 |
| TE-2 | 检查 Go SDK 代码块是否有函数级 docstring | 删除 docstring，改用 `#` 行注释 |
| TE-3 | 检查错误表是否超过 3 列 | 合并列，每行 1 个错误码 |
| TE-4 | 检查 JSON path 是否在文件顶部集中声明 | 移至文件顶部统一声明 |
| TE-5 | 检查 example-config.yaml 是否有重复字段 | 用 YAML anchors 消除 |
| TE-6 | 检查 SKILL.md 与 references/ 是否有内容重复 | 删除 references 中的重复 |
| TE-7 | 检查 AIOps/FinOps 是否在 `references/advanced/`；安全敏感操作是否标注 Security-Sensitive | 移至 `advanced/` + 添加 Security-Sensitive 标注 |

**发现任一违规 → 立即修复 → 重新检查直到全部通过。**

## Round 2 — Critical Analysis

4. **Gap Analysis**: What would break in production if a user follows this skill?
5. **Alternative Coverage**: Is there a better way that reduces agent confusion?
6. **Escalation Paths**: Are HALT conditions clear? Enough non-retryable error patterns?
7. **Cross-Pillar Synergy**: Do FinOps recommendations conflict with reliability? SecOps create performance bottlenecks?

**For any issue found: fix immediately, then re-verify.** Do not report and stop — fix and verify the fix passes.

- A single shot gun covers everything: `hwcloud-skillcheck check --pre-commit`. This is what the git hook and CI both invoke — running it locally is equivalent to pushing.
- The git pre-commit hook is fully covered by `hwcloud-skillcheck check --pre-commit`; CI runs the same command. Markdown-only commits stay fast because Go build/test gates skip when `.go` and the `hwcloud-skillcheck/` tree are unchanged.
- New scripts MUST:
  - Start with a module docstring describing purpose.
  - Avoid unused imports / unreachable code / bare `except:`.
  - Prefer `flag` (std lib) with explicit `--help` text for CLIs.
  - Keep functions short; favor pure helpers that are unit-testable.
- Tests live next to source as Go `_test.go` files; `go test ./...` runs them. Subagent-driven-development + race detector are how new functionality is verified before commit.
- CI runs `hwcloud-skillcheck validate --root .` plus `go test ./... -race`; local dev MUST run the same suite before pushing.
