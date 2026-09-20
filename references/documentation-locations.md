# Documentation Locations (强制)

文档必须放置在以下固定位置，**禁止随意新建顶层 docs/ 子目录**：

| 类型 | 路径 | 说明 |
|------|------|------|
| **ADR（架构决策记录）** | `docs/architecture/NNNN-<slug>.md` | 编号递增，slug 用 kebab-case。任何架构选型（存储/接口/外部依赖/取舍）必写 ADR |
| **Spec（功能规格）** | `docs/superpowers/specs/<slug>.md` | 配合 ADR 写，描述 FR/NFR/数据模型 |
| **Implementation Plan** | `docs/superpowers/plans/YYYY-MM-DD-<slug>.md` | 遵循 `superpowers:writing-plans` 模板 |
| **运行时规范** | `docs/gcl-spec.md`、`docs/deployment-guide.md` 等根级 | 不轻易新建根级 .md，先复用现有 |

## ADR 文件名约束

- 4 位数字编号（`0001` ~ `9999`），递增
- 单数主题一个 ADR（如 `0007-outcome-memory-self-healing.md`）
- 状态字段：`Proposed` → `Accepted` → `Superseded`，写入 frontmatter 或正文

## 反模式

- ❌ 把 ADR 写到 **docs/adr/**、**docs/decisions/**、**docs/adr-NNNN/** 等其他目录
- ❌ 把 Plan 写到 **docs/plans/** 或根级 **docs/<feature>.md**
- ❌ 没有编号的 ADR（如 architecture-decision.md 不允许）

**Why**: 跨仓库协作时（如 GCL 生成新 skill 时引用 ADR），固定路径才能让引用稳定。`docs/architecture/` 是 hcloud-skills 项目的硬约定，所有 skill / generator / docs 工具都必须遵守。
