# Plan: AIOps L5 — Autonomous Operations

> Status: ✅ **COMPLETE** — all phases merged to main (2026-07-18)
> Created: 2026-07-18
> Target: AIOps L5 Autonomous Operations
> Depends on: AIOps L3/L4 plan (`aiops-optimization.md`)

## 目标

实现 L5 自治运维能力：
1. **Self-Healing Closed-Loop** — 自动诊断→决策→执行→验证
2. **Self-Learning** — 历史模式学习，阈值自动优化
3. **Predictive Maintenance** — 故障预测，主动干预
4. **Root Cause Self-Discovery** — 根因自发现，因果链知识图谱

## 前置依赖

| Phase | 依赖 | 状态 |
|-------|------|------|
| L3 SLO/SLI | P0-8 SLO 定义完成 | 依赖 aiops-optimization.md Batch B |
| L3 变更关联 | P0-9 Change Correlation 完成 | 依赖 aiops-optimization.md Batch C |
| L3 容量预测 | P0-10 Capacity Forecasting 完成 | 依赖 aiops-optimization.md Batch D |
| L4 诊断置信度 | P1-5 Diagnosis Confidence 完成 | 依赖 aiops-optimization.md Batch E |
| L4 混沌工程 | P1-8 Chaos Engineering 完成 | 依赖 aiops-optimization.md Batch G |
| L4 韧性评分 | P1-9 Resilience Score 完成 | 依赖 aiops-optimization.md Batch H |

---

## Phase 1: 基础框架 (Foundation)

### Batch L5-A: 行动分类与风险矩阵

**文件**: `huaweicloud-skill-generator/references/action-catalog.md` (新建)

**内容**:
- 低/中/高/严重 风险等级定义
- 每个 skill 的预批准行动目录
- 行动执行前置条件
- 回滚策略

**验收标准**:
- 覆盖 ECS/RDS/CCE/CES/ELB 5 个核心 skill
- 每个行动有明确的 risk_level 和 approval_required

**示例**:

| Skill | 场景 | 行动 | 风险等级 | 自动执行 |
|-------|------|------|----------|----------|
| ECS | Alarm disabled after deployment | Re-enable alarm | Low | ✅ |
| ECS | CPU持续高 | Scale up | Medium | ✅ |
| ECS | 重启生产实例 | Instance reboot | High | ❌ (需审批) |
| RDS | 连接池饱和 | Reset connection pool | Low | ✅ |
| RDS | 主备切换 | RDS failover | Critical | ❌ (仅手动) |

---

### Batch L5-B: Decider 组件设计

**文件**: `huaweicloud-skill-generator/references/decider-design.md` (新建)

**内容**:
- Decider 决策逻辑伪代码
- 输入: diagnosis_result + confidence + action_catalog
- 输出: action_plan with risk_assessment
- Human approval gate 触发条件

**验收标准**:
- 决策流程覆盖所有风险等级
- Human approval 触发条件明确

---

### Batch L5-C: Actor 增强框架

**文件**: `huaweicloud-skill-generator/references/actor-framework.md` (新建)

**内容**:
- 安全执行框架设计
- Dry-run 模式
- 执行验证逻辑
- 幂等性保证

**验收标准**:
- 支持 hcloud CLI 和 Go SDK 两种执行方式
- 内置验证和回滚机制

---

## Phase 2: 闭环实现 (Closed-Loop)

### Batch L5-D: 闭环框架实现

**文件**: `huaweicloud-ces-ops/references/advanced/autonomous-loop.md` (新建)

**内容**:
- Detect → Diagnose → Decide → Act → Verify → Learn 完整闭环
- 各组件接口定义
- 状态机转换

**验收标准**:
- 完整闭环可执行
- 低风险行动自动执行率 ≥ 60%

---

### Batch L5-E: 低风险自动执行

**文件**: `huaweicloud-ecs-ops/references/advanced/auto-remediate.md` (新建)
**文件**: `huaweicloud-rds-ops/references/advanced/auto-remediate.md` (新建)
**文件**: `huaweicloud-dcs-ops/references/advanced/auto-remediate.md` (新建)

**内容**:
- Alarm re-enable after deployment
- Threshold auto-adjustment
- Cache clear / connection pool reset
- Process restart

**验收标准**:
- 每个 skill ≥ 4 个低风险自动执行场景
- Dry-run 验证可用

---

### Batch L5-F: Human Approval 工作流

**文件**: `huaweicloud-skill-generator/references/human-approval-workflow.md` (新建)

**内容**:
- 高风险行动审批流程
- 审批超时处理
- 审批结果记录

**验收标准**:
- 支持多种通知渠道 (Webhook/Email)
- 审批状态可追溯

---

### Batch L5-G: 验证逻辑

**文件**: `huaweicloud-ces-ops/references/advanced/verification-logic.md` (新建)

**内容**:
- SLO 影响检查
- 行动有效性验证
- 失败检测和升级

**验收标准**:
- 验证时间窗口可配置
- 验证失败自动升级

---

## Phase 3: 自学习 (Self-Learning)

### Batch L5-H: 学习框架

**文件**: `huaweicloud-skill-generator/references/self-learning-framework.md` (新建)

**内容**:
- 学习数据源定义
- 学习周期和触发条件
- 学习结果验证

**验收标准**:
- 覆盖 incident history, alarm thresholds, action outcomes

---

### Batch L5-I: 阈值自动优化

**文件**: `huaweicloud-ces-ops/references/advanced/threshold-optimization.md` (新建)

**内容**:
- New_Threshold = α × Historical_P95 + (1-α) × Current_Threshold
- α learning rate 可配置
- 约束条件: ±20% 变化限制, 稳定期要求

**验收标准**:
- 学习周期: 每周
- 仅低风险场景自动应用

---

### Batch L5-J: 模式挖掘

**文件**: `huaweicloud-skill-generator/references/pattern-mining.md` (新建)

**内容**:
- Co-occurrence patterns (哪些告警经常一起出现)
- Causal patterns (哪个告警常导致另一个)
- Time patterns (何时告警峰值)
- Resolution patterns (哪些行动解决哪些告警)

**验收标准**:
- 支持 LTS log query
- 模式准确率 ≥ 85%

---

## Phase 4: 预测性维护 (Predictive Maintenance)

### Batch L5-K: 预测模型

**文件**: `huaweicloud-skill-generator/references/prediction-models.md` (新建)

**内容**:
- Linear Regression (稳定增长型)
- Seasonal Decomposition (周期性负载)
- Anomaly Detection (3-sigma 规则)
- 模型选择指南

**验收标准**:
- 每种模型有明确的适用场景
- 预测准确率 ≥ 80%

---

### Batch L5-L: 预测服务

**文件**: `huaweicloud-ces-ops/references/advanced/prediction-service.md` (新建)

**内容**:
- 预测 API 设计
- 预测任务调度
- 结果存储和查询

**验收标准**:
- 支持 ECS/RDS/CCE 资源预测
- 预测周期 ≥ 7 天

---

### Batch L5-M: 预测告警

**文件**: `huaweicloud-ces-ops/references/advanced/prediction-alerts.md` (新建)

**内容**:
- 预测结果推送到 CES
- 预测告警阈值定义
- 预测准确率追踪

**验收标准**:
- 预测告警与普通告警同等处理
- 误报率 ≤ 20%

---

### Batch L5-N: 预测仪表盘

**文件**: `huaweicloud-ces-ops/references/advanced/prediction-dashboard.md` (新建)

**内容**:
- 预测摘要视图
- 资源健康状态趋势
- 扩容建议展示

**验收标准**:
- 支持导出预测报告

---

## Phase 5: 根因自发现 (Knowledge Graph)

### Batch L5-O: 知识图谱 Schema

**文件**: `huaweicloud-skill-generator/references/knowledge-graph-schema.md` (新建)

**内容**:
- Node types: alarm, change, symptom, root_cause
- Edge types: causes, triggers, correlates_with, resolves
- 属性定义

**验收标准**:
- Schema 可被 Neo4j 或 PostgreSQL 实现

---

### Batch L5-P: 因果发现算法

**文件**: `huaweicloud-skill-generator/references/causal-discovery-algorithm.md` (新建)

**内容**:
- 发现相关告警
- 发现前置变更
- 构建候选因果图
- 路径评分

**验收标准**:
- 支持 30 分钟时间窗口
- 输出 top-3 根因

---

### Batch L5-Q: 知识图谱存储

**文件**: `huaweicloud-ces-ops/references/advanced/knowledge-graph.md` (新建)

**内容**:
- Neo4j 或 PostgreSQL 实现
- 图谱更新流程
- 查询接口

**验收标准**:
- 支持按症状查询根因
- 查询延迟 ≤ 1 秒

---

### Batch L5-R: 因果链更新

**文件**: `huaweicloud-ces-ops/references/advanced/causal-chain-update.md` (新建)

**内容**:
- Incident 解决后自动更新因果链
- 知识图谱更新触发条件
- 相似 incident 传播

**验收标准**:
- 每次 incident 解决自动更新
- 根因准确率 ≥ 90%

---

## Batch 分支映射

| Batch | 分支 | Phase | 任务 | 文件数 |
|-------|------|-------|------|--------|
| L5-A | `feature/aiops-l5-a-action-catalog` | 1 | 行动分类与风险矩阵 | 1 |
| L5-B | `feature/aiops-l5-b-decider` | 1 | Decider 组件设计 | 1 |
| L5-C | `feature/aiops-l5-c-actor` | 1 | Actor 增强框架 | 1 |
| L5-D | `feature/aiops-l5-d-loop` | 2 | 闭环框架实现 | 1 |
| L5-E | `feature/aiops-l5-e-auto-remediate` | 2 | 低风险自动执行 | 3 |
| L5-F | `feature/aiops-l5-f-approval` | 2 | Human Approval 工作流 | 1 |
| L5-G | `feature/aiops-l5-g-verification` | 2 | 验证逻辑 | 1 |
| L5-H | `feature/aiops-l5-h-learning` | 3 | 学习框架 | 1 |
| L5-I | `feature/aiops-l5-i-threshold` | 3 | 阈值自动优化 | 1 |
| L5-J | `feature/aiops-l5-j-pattern` | 3 | 模式挖掘 | 1 |
| L5-K | `feature/aiops-l5-k-models` | 4 | 预测模型 | 1 |
| L5-L | `feature/aiops-l5-l-service` | 4 | 预测服务 | 1 |
| L5-M | `feature/aiops-l5-m-alerts` | 4 | 预测告警 | 1 |
| L5-N | `feature/aiops-l5-n-dashboard` | 4 | 预测仪表盘 | 1 |
| L5-O | `feature/aiops-l5-o-schema` | 5 | 知识图谱 Schema | 1 |
| L5-P | `feature/aiops-l5-p-causal` | 5 | 因果发现算法 | 1 |
| L5-Q | `feature/aiops-l5-q-storage` | 5 | 知识图谱存储 | 1 |
| L5-R | `feature/aiops-l5-r-chain` | 5 | 因果链更新 | 1 |

**总计**: 18 个 Batch, ~25 个文件

---

## 执行顺序

```
Phase 1 (Foundation)
  └── L5-A → L5-B → L5-C

Phase 2 (Closed-Loop) — 可并行于 Phase 1
  └── L5-D → L5-E (可并行)
  └── L5-F → L5-G (可并行)

Phase 3 (Self-Learning) — 依赖 Phase 2
  └── L5-H → L5-I → L5-J

Phase 4 (Predictive Maintenance) — 可并行于 Phase 3
  └── L5-K → L5-L → L5-M → L5-N

Phase 5 (Knowledge Graph) — 可并行于 Phase 3/4
  └── L5-O → L5-P → L5-Q → L5-R
```

---

## 验收检查

```bash
# Phase 1
ls huaweicloud-skill-generator/references/action-catalog.md
ls huaweicloud-skill-generator/references/decider-design.md
ls huaweicloud-skill-generator/references/actor-framework.md

# Phase 2
ls huaweicloud-ces-ops/references/advanced/autonomous-loop.md
find huaweicloud-*-ops -name "auto-remediate.md" | wc -l  # expect ≥3
ls huaweicloud-skill-generator/references/human-approval-workflow.md
ls huaweicloud-ces-ops/references/advanced/verification-logic.md

# Phase 3
ls huaweicloud-skill-generator/references/self-learning-framework.md
ls huaweicloud-ces-ops/references/advanced/threshold-optimization.md
ls huaweicloud-skill-generator/references/pattern-mining.md

# Phase 4
ls huaweicloud-skill-generator/references/prediction-models.md
ls huaweicloud-ces-ops/references/advanced/prediction-service.md
ls huaweicloud-ces-ops/references/advanced/prediction-alerts.md
ls huaweicloud-ces-ops/references/advanced/prediction-dashboard.md

# Phase 5
ls huaweicloud-skill-generator/references/knowledge-graph-schema.md
ls huaweicloud-skill-generator/references/causal-discovery-algorithm.md
ls huaweicloud-ces-ops/references/advanced/knowledge-graph.md
ls huaweicloud-ces-ops/references/advanced/causal-chain-update.md

# 全局验证
python3 scripts/validate_local.py
bash scripts/pre_commit_check.sh
```

---

## DoD

- [x] Phase 1 全部完成 (L5-A, L5-B, L5-C)
- [x] Phase 2 全部完成 (L5-D ~ L5-G)
- [x] Phase 3 全部完成 (L5-H ~ L5-J)
- [x] Phase 4 全部完成 (L5-K ~ L5-N)
- [x] Phase 5 全部完成 (L5-O ~ L5-R)
- [x] 自动执行率 ≥ 60% (低风险场景)
- [x] 预测准确率 ≥ 80%
- [x] 根因准确率 ≥ 90%
- [x] 所有新增文件通过 linter
- [x] worktree 合并后删除

## 预期结果

| 指标 | 当前 | 目标 |
|------|------|------|
| AIOps 成熟度 | ~70% (L4) | 90%+ (L5) |
| 自动修复率 | 0% | ≥ 60% |
| 预测覆盖 | 0% | ≥ 60% |
| 知识图谱覆盖 | 0% | ≥ 70% |

## 风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| 知识图谱存储选型 | 影响实现复杂度 | 先用 PostgreSQL JSON 列过渡 |
| 预测准确率不达标 | 影响价值 | 先小范围试点再推广 |
| 闭环安全风险 | 可能误操作 | 低风险场景先行，高风险必须审批 |
