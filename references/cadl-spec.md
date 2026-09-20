# 复利资产沉淀机制（Compound-Asset Distillation Loop, CADL）

**目的**：让少量高价值决策规则产生复利——下次同类任务**不读代码、不重复踩坑**也能走对。
**默认不写。** 大多数任务的正确终点是：测试绿、CI 绿、代码/配置即文档——**不是**再抄一遍到 AGENTS.md。

## 价值取向（什么值得沉淀）

复利资产 ≠ 经验日记。写入前必须满足 **「四问全过」**：

| # | 问题 | 过栏 |
|---|------|------|
| 1 | **复用半径** — 未来还有多少任务会碰到？ | ≥3 次同类场景，或跨 skill / 跨模块 |
| 2 | **失败成本** — 如果不写，会怎样？ | silent wrong（看起来绿、实际错）或 ≥30min 排查 |
| 3 | **抽象层级** — 这是决策规则还是操作手册？ | 决策规则（遇到 X → 做 Y）；不是标准库/工具官方文档可查到的事实 |
| 4 | **可执行性** — agent 读完能立刻改变行为吗？ | 一条 Rule 即可约束；不需要再读 200 行上下文 |

**任一不过 → 不写入 AGENTS.md。** 落点降级：

| 情况 | 落点 |
|------|------|
| 已用测试 / CI / ADR / workflow 门禁 | ** nowhere ** — 代码即文档，不写 |
| 仅本仓库、但高价值 | 本节「复利资产」或上方规范章节 |
| 跨仓库通用 | 用户级 `~/.config/opencode/AGENTS.md` |
| 某 skill 专属 | skill 的 `references/`，不经 AGENTS.md |

## 明确不写入（反模式）

- **已修复的一次性 bug** — 测试或 CI 已覆盖，下次 red 即信号
- **标准实践** — gofmt、heredoc 引号、`StdinPipe` 先 Close、optional JSON 设默认值
- **环境小技巧** — `GOCACHE=/tmp/...`、action 版本号；写进 commit/PR 即可
- **与现有条目重复** — 写入前 `grep AGENTS.md`；重复 = 噪音
- **纯叙事** — 「某次 CI 红了然后修了」无 Rule 可提取

## 触发条件（任务结束时检查）

满足任一 → 走沉淀**判定**（不是判定 = 必须写）：

- 多步 / 跨文件 / 跨 skill 任务完成
- 评审或修复循环（GCL、self-review、CI auto-recover）
- 发现 silent wrong 或架构级坑
- 用户给出可复用的工作流偏好

## 闭环步骤

```
1. 提取   → 能否写成一条 Rule？不能 → 停止
2. 四问   → 全过？不过 → 停止（或降级到 ADR / commit message）
3. grep   → 已有覆盖？→ 停止
4. 落点   → 复利资产 / 规范章节 / ADR / skill references
5. 门禁   → AGENTS.md ≥500 行时，加一条必须删或合并一条（见下行数预算）
6. 复用   → 下次同类任务读 AGENTS.md 即生效
```

## 行数预算

AGENTS.md 是 **agent 上下文税**，不是 wiki。硬上限意识：

- **规范 + 门禁**（Pre-flight、Dual-Copy、TE、GCL 指针）：保留，这是 repo 的操作系统
- **术语表**：索引 ADR，不复制 API 面（详见 `docs/architecture/`）
- **复利资产**： curated，目标 **≤12 条**；超出则 prune 最弱条目

## Skill 侧钩子

- Agent 任务结束前主动做沉淀**判定**；用户未要求时不批量写条目
- `huaweicloud-skill-generator` 在 SKILL.md 末尾保留一行 CADL 提示即可
