# L4 闭环优化 Spec + Plan

> **目标**：从 L3+ 推进到完整 L4 运行成熟度
> **设计成熟度**：已是 L4（4 引擎 + 规范 + 验证全部就绪）
> **运行成熟度**：当前 L3（生产闭环未形成，仅 ECS 有完整自愈链）

---

## 1. Spec — 要交付什么

### 1.1 目标

将 4 个独立 L4 引擎（dynamic_orchestration / predictive_ops / topology_graph / progressive_trust）
从"可独立运行的 CLI 工具"升级为"统一调度闭环中的协作模块"，
让任何故障从「检测 → 影响面分析 → 趋势确认 → 编排策略 → GCL 门控 → 信任更新 → 经验沉淀」
一气呵成，无须人工串联。

### 1.2 范围

| 级别 | 工作项 | 交付物 |
|------|--------|--------|
| **P0** | 知识库扩展到 Top-5 高频 skill | RDS/VPC/ELB/CCE 各自 `failure_patterns.json` (≥8 patterns) + `remediation-playbooks.json` (≥4 playbooks) |
| **P0** | Runtime Orchestrator 闭环 | `scripts/runtime_orchestrator.py`（串联 4 引擎 + GCL 的统一调度器） |
| **P1** | Critic v1（生产可用 Critic） | `scripts/critic_v1.py`（5 维规则评分，可独立注入到 GCL Runner） |
| **P1** | Trust 状态跨会话持久化 | 修改 `progressive_trust.py`：trust-state.json 自动累积 |
| **P1** | Topology 动态发现 | 修改 `topology_graph.py`：从 skill assets 解析资源依赖 |

### 1.3 非范围（Out of Scope）

- **CES 告警 → webhook 接入**：基础设施改动大，留 P2
- **剩余 19 个 skill 的 knowledge 扩展**：Phase 1 验证可复用模板后再批量
- **Trust 多用户隔离 / 审计日志**：需要账户体系，P3

### 1.4 验收标准

| 验收项 | 衡量 |
|--------|------|
| Orchestrator 闭环 | `runtime_orchestrator.py handle --fault "RDS connection timeout" --json` 返回包含 `topology / predictive / orchestration / gcl / trust / learning` 6 段的 plan |
| 知识库覆盖 | RDS/VPC/ELB/CCE 各自 ≥8 patterns + ≥4 playbooks；trace_learning.py `report` 不再对这些 skill 返回 "no patterns" |
| Critic v1 | `critic_v1.py --generator <trace.json>` 输出 5 维评分 (Safety/Correctness/Idempotency/SecOps/FinOps) 且可被 `gcl_runner.py --critic-json` 接受 |
| Trust 持久化 | 第一次 evaluate 后 `audit-results/trust-state.json` 自动存在；第二次不传 `--trust-data` 也工作 |
| Topology 动态 | `topology_graph.py impact --resource rds:instance --from-skill-assets` 比默认输出 ≥1 个新发现节点 |
| 全套验证 | `ruff check .` + `python3.10 -m py_compile scripts/*.py` + `python3 scripts/validate_local.py` 全通过 |

---

## 2. Plan — 怎么做

### Phase 1 — 知识库扩展（P0，预计 30 分钟）

**目标**：让 RDS/VPC/ELB/CCE 各拥有可工作的知识库。

**做法**：用一个生成脚本 `scripts/gen_skill_knowledge.py`（用完即删或保留为 generator），
按 OpenAPI 常见错误码 + ECS 模板批量生成。**不手工写 80+ 条 JSON**。

```python
# 伪代码（脚本会落盘）
for skill in [rds, vpc, elb, cce]:
    patterns = generate_patterns(skill, n=10)   # 基于错误码表
    playbooks = generate_playbooks(skill, n=6) # 基于高频 ops
    write(f"huaweicloud-{skill}-ops/assets/failure_patterns.json", patterns)
    write(f"huaweicloud-{skill}-ops/assets/remediation-playbooks.json", playbooks)
```

**关键产物**：
- `scripts/gen_skill_knowledge.py`（生成器）
- `huaweicloud-{rds,vpc,elb,cce}-ops/assets/failure_patterns.json` × 4
- `huaweicloud-{rds,vpc,elb,cce}-ops/assets/remediation-playbooks.json` × 4

### Phase 2 — Runtime Orchestrator（P0，预计 45 分钟）

**目标**：一个 CLI 入口，串联所有引擎。

**核心入口**：`scripts/runtime_orchestrator.py`

```bash
runtime_orchestrator.py handle \
  --fault "RDS connection timeout" \
  --risk medium \
  --skills huaweicloud-rds-ops,huaweicloud-vpc-ops \
  --json
```

**内部流程**（复用已有引擎的内部函数，非 subprocess 调用）：

```
1. topology_graph.impact(fault_resource)         → 影响面
2. orchestration.discover_transitive_skills()    → 协同 skill 集
3. predictive_ops.detect_trend(metrics)          → 趋势（可选）
4. orchestration.build_plan()                   → 编排 plan
5. gcl_runner.run_plan()  ← Critic v1 评分       → 通过/拒绝
6. progressive_trust.update(history, outcome)    → 信任累积
7. trace_learning.record(trace)                  → 经验沉淀
```

输出 6 段 JSON：
```json
{
  "fault_id": "uuid",
  "topology": {...},
  "orchestration": {...},
  "predictive": {...},
  "gcl": {...},
  "trust": {...},
  "learning": {...}
}
```

### Phase 3 — Critic v1（P1，预计 30 分钟）

**目标**：把 structural-critic-only 升级为可独立部署的规则引擎。

**新文件**：`scripts/critic_v1.py`

**5 维评分规则**：

| 维度 | 检测内容 | 评分逻辑 |
|------|---------|---------|
| Safety | 破坏性命令 + 是否有 dry-run | 0/1 二值 |
| Correctness | 命令结构 + JSON paths 是否在文件顶部声明 | 0/1 二值 |
| Idempotency | GET 1.0 / POST 0.5 / DELETE 0.0 | 0-1 连续 |
| SecOps | `mask_secrets` 检测 + IAM 权限提示 | 0/1 二值 |
| FinOps | 按产品+操作类型查成本表 | 0-1 连续 |

**CLI**：
```bash
critic_v1.py --generator <generator.json>           # 输出 critic.json
critic_v1.py --generator <g> --emit-critic-json     # 与 gcl_runner 兼容
```

### Phase 4 — Trust 持久化（P1，预计 20 分钟）

**修改 `scripts/progressive_trust.py`**：

- 新增 subcommand `state-path`（打印默认 trust-state.json 路径）
- `evaluate` 默认从 `audit-results/trust-state.json` 加载历史
- 每次 evaluate 完成后 append 到 trust-state.json（去重 by operation_id）
- `--trust-data` 显式传入时优先
- 兼容：文件不存在时 graceful 降级（视为首次运行）

### Phase 5 — Topology 动态发现（P1，预计 25 分钟）

**修改 `scripts/topology_graph.py`**：

- 新增 `--from-skill-assets` flag
- 解析逻辑：读 `huaweicloud-*-ops/SKILL.md` frontmatter 中的 `delegates_to` + `references/integration.md` 中的依赖声明
- 与现有 static model 合并，标记来源（`source: static | dynamic | both`）
- 动态发现额外边时输出 warning 提示人工 review

### Phase 6 — 验证 + CADL（P1，预计 15 分钟）

**验证套件**：
1. `bash scripts/run_ruff.sh .`
2. `python3 scripts/check_py310_compat.py`
3. `python3 scripts/validate_local.py`
4. smoke test：每个新 CLI 至少 1 次成功调用
5. E2E：orchestrator handle 一条 fault 走完全闭环

**CADL 沉淀**（AGENTS.md Lesson #10）：
- 多引擎串联 = "函数调用而非 subprocess"
- 知识库批量生成 = "模板驱动而非手工编写"
- 信任持久化 = "显式优于隐式，但默认要 dumb 友好"

---

## 3. 实施顺序与依赖

```
Phase 1 (kb_ext) ──→ Phase 2 (orchestrator)
                         ↓
              ┌──────────┴──────────┐
              ↓                     ↓
       Phase 3 (critic v1)    Phase 4 (trust)
                         ↓
                   Phase 5 (topology)
                         ↓
                   Phase 6 (verify + CADL)
```

依赖说明：
- Phase 2 依赖 Phase 1（orchestrator 调用 trace_learning 需要 knowledge base 存在）
- Phase 3 不依赖其他（独立 critic 引擎）
- Phase 4/5 独立模块改动
- Phase 6 必须最后跑全套验证

---

## 4. 风险与回滚

| 风险 | 概率 | 影响 | 缓解 |
|------|------|------|------|
| 生成脚本产出低质量 knowledge | 中 | 中 | 复用 ECS 模板，trace_learning 验证 schema 合法 |
| Orchestrator 串联出错 | 中 | 高 | 每个步骤独立 try/except，单步失败不影响整体 |
| Critic v1 误判 | 中 | 中 | 默认仍可走 structural-critic-only；v1 仅 `--use-critic-v1` 启用 |
| Trust 状态污染 | 低 | 中 | 每次更新加 timestamp；首次运行自动创建空文件 |
| Topology 动态边错误 | 中 | 低 | 只标记边而不删除静态边；warn 而非 error |

---

## 5. Self-Reflection 节点

按用户偏好，每个 Phase 结束后做轻量自审：
- **效率**：是否有更简单实现？
- **质量**：是否符合 AGENTS.md 规范（ruff / py310 / 验证）？
- **复利**：是否产出可复用模式？是否值得沉淀到 AGENTS.md？

每个 Phase 完成时同步更新此文档的 ✅/⏳ 状态。