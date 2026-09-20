# 术语表 (Glossary)

> **索引，非副本。** 字段级 API → ADR（`docs/architecture/`）与源码（`internal/l4/`）。

| 术语 | 一句话 | 详见 |
|------|--------|------|
| **L3→L4 / L4→L5** | Agent 成熟度跃迁；L4 = outcome memory + healing；L5 = trust 单一来源 | ADR-0007~0009 |
| **Outcome Memory** | 跨任务 step 结果 JSONL（`.l4-memory/outcomes.jsonl`），self-healing 底座 | ADR-0007 |
| **Context Memory** | 跨调用 agent 状态 JSON（`.l4-memory/context.json`），atomic write | ADR-0008 |
| **Trust Score / Phase 1–4** | 历史-derived 可信度；Phase 4 后单一来源 = outcome memory | ADR-0009 |
| **Executor / RealExecutor** | `RunExecutionLoop` 与 subprocess 的 interface seam | ADR-0010 |
| **GCL** | Generator + Critic 双 Agent 闭环质量门控 | `docs/gcl-spec.md` |
| **L4 Orchestrator** | 多 step 执行 + RBAC + GCL + topology + trust + healing | `internal/l4/` |
| **Cross-skill delegation** | Orchestrator 经 `DelegatesTo` 扩计划并同步 pipeline 执行（非 skill 互调） | ADR-0011 |
| **RBAC** | 按 `RBACRisk` 做操作前权限决策 | `internal/l4/rbac.go` |
| **Topology Graph** | skill→resource 静态+动态依赖图 | `internal/l4/topology.go` |
| **CADL** | 复利资产沉淀机制（见 `references/cadl-spec.md`） | `references/cadl-spec.md` |
| **Dual-Copy Trap** | generator 根副本 vs `.agents/skills/` 运行时副本；见 CA-10 | `references/agents-md-rules.md` |

## L4 实现约束（改 healing/trust/executor 前必读）

1. `HealingPolicy` 零值安全 → 只以 `p.IsZero()` 判断，不用 sum-based check
2. destructive verb 列表只从 `ExtractHighRiskVerbs()` 取；匹配走 `TaskStep.Verb` 非 `Action` 子串
3. 改 `Executor` interface 或 bypass → 新 ADR
