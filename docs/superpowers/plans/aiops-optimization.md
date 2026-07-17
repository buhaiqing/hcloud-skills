# Plan: AIOps L4 成熟度提升

> Status: 📋 **PLANNED** — not started
> Created: 2026-07-18
> Target: AIOps L4 (80% L4 criteria, ~70% overall)
> Parent: `docs/superpowers/plans/gcl-token-efficiency-p0.md`

## 目标

将 AIOps 成熟度从当前约 45% (L2) 提升至 70%+ (L4 标准)，覆盖 P0 + P1 合规检查项。

## AIOps L4 成熟度模型

| Level | 特征 | 目标 |
|-------|------|------|
| L1 | ≥4 异常模式 | 当前: 部分 skill |
| L2 | 委托矩阵+知识库 | 当前: 部分 skill |
| **L3** | **SLO+变更关联+容量预测** | **本计划 P0 项** |
| **L4** | **混沌工程+韧性评分** | **本计划 P1 项** |
| L5 | 自治修复 | 未来目标 |

## 合规检查项 (AIOps Best Practices §16)

### P0 — Must Pass (L3 基础)

| # | 检查项 | 当前状态 | 需要工作 |
|---|--------|---------|---------|
| P0-1 | Multi-Metric Inspection (≥4 patterns) | 部分 skill 有 | 补齐 L1 skill |
| P0-2 | Cross-Skill Decision Tree | 部分 skill 有 | 统一格式 |
| P0-3 | Delegation Matrix | 部分 skill 有 | 补齐 L1 skill + 创建模板 |
| P0-4 | Alarm Storm Handling | CES/ECS/CCE 有 | 推广到其他 |
| P0-5 | Diagnosis Schema | 部分 skill 有 | 统一 schema |
| P0-6 | AOM/LTS Integration | 部分 skill 有 | 补齐 |
| P0-7 | Knowledge Base (≥3 patterns) | 部分 skill 有 | 补齐 L1 skill |
| P0-8 | SLO/SLI Definition | 仅 ECS 有 | 推广到 10 skill |
| P0-9 | Change Correlation (CTS) | 无 | 3 skill 试点 |
| P0-10 | Capacity Forecasting | 仅 ECS/CSS 有 | 推广到核心 skill |

### P1 — Should Pass (L4 能力)

| # | 检查项 | 当前状态 | 需要工作 |
|---|--------|---------|---------|
| P1-1 | Cascade Patterns (≥2) | 部分 skill 有 | 补齐并标准化 |
| P1-2 | Observability Trinity (Metrics-Logs-Traces) | 无 | 新建模板 |
| P1-3 | Prompt Handbook (≥20 prompts) | 无 | 各 skill 新建 |
| P1-4 | Trend Detection | 部分 skill 有 | 标准化 |
| P1-5 | Diagnosis Confidence | 无 | 模板+CES 落地 |
| P1-6 | Change Correlation | 无 | P0-9 已覆盖 |
| P1-7 | Capacity Forecasting | 仅 ECS/CSS 有 | P0-10 已覆盖 |
| P1-8 | **Chaos Engineering** | **无** | **L4 核心要求** |
| P1-9 | **Resilience Score** | **无** | **L4 核心要求** |

---

## Phase 1: L3 基础建设 (P0 items)

### Batch A: 全局模板 (P0-3 委托矩阵)

**文件**: `huaweicloud-skill-generator/references/cross-skill-delegation-matrix-template.md` (新建)

**内容**:
- 告警类型 → skill 映射表模板
- 委托优先级和依赖关系
- 各 skill integration.md 引用格式

### Batch B: SLO/SLI 标准化 (P0-8)

| 子任务 | 文件 |
|--------|------|
| B.1 | `huaweicloud-ecs-ops/references/well-architected-assessment.md` |
| B.2 | `huaweicloud-cce-ops/references/well-architected-assessment.md` |
| B.3 | `huaweicloud-rds-ops/references/well-architected-assessment.md` |
| B.4 | `huaweicloud-ces-ops/references/well-architected-assessment.md` |
| B.5 | `huaweicloud-elb-ops/references/well-architected-assessment.md` |
| B.6 | `huaweicloud-dcs-ops/references/well-architected-assessment.md` |
| B.7 | `huaweicloud-dms-ops/references/well-architected-assessment.md` |
| B.8 | `huaweicloud-css-ops/references/well-architected-assessment.md` |
| B.9 | `huaweicloud-gaussdb-ops/references/well-architected-assessment.md` |
| B.10 | `huaweicloud-vpc-ops/references/well-architected-assessment.md` |

**每个文件内容** (参考 aiops-best-practices.md §1.4):
- SLI 指标定义 (可用性、延迟P99、错误率、饱和度)
- SLO 目标值 (如可用性 ≥ 99.9%)
- Error Budget 燃烧率告警表

### Batch C: 变更关联分析 (P0-9, P1-6)

| 子任务 | 文件 |
|--------|------|
| C.1 | `huaweicloud-ecs-ops/references/advanced/change-correlation.md` (新建) |
| C.2 | `huaweicloud-cce-ops/references/advanced/change-correlation.md` (新建) |
| C.3 | `huaweicloud-rds-ops/references/advanced/change-correlation.md` (新建) |

**每个文件内容**:
- CTS 事件类型 → 故障类型映射 (≥5 条)
- 时间窗口关联分析 (故障前后 30 分钟)
- 常见变更触发告警模式

### Batch D: 容量预测 (P0-10, P1-7)

| 子任务 | 文件 |
|--------|------|
| D.1 | `huaweicloud-ecs-ops/references/advanced/capacity-forecasting.md` (新建/扩展) |
| D.2 | `huaweicloud-cce-ops/references/advanced/capacity-forecasting.md` (新建) |
| D.3 | `huaweicloud-rds-ops/references/advanced/capacity-forecasting.md` (新建) |
| D.4 | `huaweicloud-dcs-ops/references/advanced/capacity-forecasting.md` (新建) |
| D.5 | `huaweicloud-css-ops/references/advanced/capacity-forecasting.md` (新建/扩展) |

**每个文件内容** (参考 aiops-best-practices.md §14):
- 预测模型选择 (线性外推/季节性分解/指数平滑)
- 容量规划工作流 (5-step)
- 容量告警规则 (Warning/Critical)

### Batch E: 诊断置信度 (P1-5)

| 子任务 | 文件 |
|--------|------|
| E.1 | `huaweicloud-skill-generator/references/diagnosis-confidence-template.md` (新建) |
| E.2 | `huaweicloud-ces-ops/references/advanced/diagnosis-confidence.md` (新建) |

**模板内容**:
- 置信度计算模型 (Evidence × Weight)
- 置信度等级与动作 (High/Medium/Low/Very Low)
- 不确定性声明规范

### Batch F: L1 Skill AIOps 增强 (P0-1, P0-3, P0-6, P0-7)

#### Batch F.1: OBS 增强

| 子任务 | 文件 | 操作 |
|--------|------|------|
| F.1.1 | `huaweicloud-obs-ops/references/advanced/aiops-patterns.md` | 新建 (≥4 patterns) |
| F.1.2 | `huaweicloud-obs-ops/references/knowledge-base.md` | 扩展 (≥5 patterns) |
| F.1.3 | `huaweicloud-obs-ops/references/integration.md` | 更新 delegation |
| F.1.4 | `huaweicloud-obs-ops/references/advanced/alarm-storm-handling.md` | 新建 |

#### Batch F.2: LTS 增强

| 子任务 | 文件 | 操作 |
|--------|------|------|
| F.2.1 | `huaweicloud-lts-ops/references/advanced/aiops-patterns.md` | 新建 (≥4 patterns) |
| F.2.2 | `huaweicloud-lts-ops/references/knowledge-base.md` | 扩展 (≥5 patterns) |
| F.2.3 | `huaweicloud-lts-ops/references/integration.md` | 更新 delegation |
| F.2.4 | `huaweicloud-lts-ops/references/advanced/alarm-storm-handling.md` | 新建 |

#### Batch F.3: CBR 增强

| 子任务 | 文件 | 操作 |
|--------|------|------|
| F.3.1 | `huaweicloud-cbr-ops/references/advanced/aiops-patterns.md` | 新建 (≥4 patterns) |
| F.3.2 | `huaweicloud-cbr-ops/references/knowledge-base.md` | 新建 (≥5 patterns) |
| F.3.3 | `huaweicloud-cbr-ops/references/integration.md` | 新建 delegation |
| F.3.4 | `huaweicloud-cbr-ops/references/advanced/alarm-storm-handling.md` | 新建 |

#### Batch F.4: SWR 增强

| 子任务 | 文件 | 操作 |
|--------|------|------|
| F.4.1 | `huaweicloud-swr-ops/references/advanced/aiops-patterns.md` | 新建 (≥4 patterns) |
| F.4.2 | `huaweicloud-swr-ops/references/knowledge-base.md` | 新建 (≥5 patterns) |
| F.4.3 | `huaweicloud-swr-ops/references/integration.md` | 新建 delegation |
| F.4.4 | `huaweicloud-swr-ops/references/advanced/alarm-storm-handling.md` | 新建 |

#### Batch F.5: CTS 增强

| 子任务 | 文件 | 操作 |
|--------|------|------|
| F.5.1 | `huaweicloud-cts-ops/references/advanced/aiops-patterns.md` | 扩展 (≥4 patterns) |
| F.5.2 | `huaweicloud-cts-ops/references/knowledge-base.md` | 扩展 (≥5 patterns) |
| F.5.3 | `huaweicloud-cts-ops/references/advanced/alarm-storm-handling.md` | 新建 |

---

## Phase 2: L4 能力建设 (P1 items)

### Batch G: 混沌工程 (P1-8) — L4 核心要求

**模板文件**: `huaweicloud-skill-generator/references/chaos-engineering-template.md` (新建)

**内容** (参考 aiops-best-practices.md §13):
- 故障注入实验设计表
- 韧性评分模型
- 混沌实验工作流

#### 试点 Skill (每 batch 一个)

| 子任务 | 文件 |
|--------|------|
| G.1 | `huaweicloud-ecs-ops/references/advanced/chaos-engineering.md` (新建) |
| G.2 | `huaweicloud-cce-ops/references/advanced/chaos-engineering.md` (新建) |
| G.3 | `huaweicloud-rds-ops/references/advanced/chaos-engineering.md` (新建) |
| G.4 | `huaweicloud-ces-ops/references/advanced/chaos-engineering.md` (新建) |
| G.5 | `huaweicloud-elb-ops/references/advanced/chaos-engineering.md` (新建) |

**每个文件内容**:
- 故障注入实验设计 (≥5 种场景)
  - 实例故障、AZ 故障、磁盘故障、负载突增、依赖故障
- 韧性评分 (每项 0-10 分)
  - 故障检测速度、故障隔离能力、恢复自动化、降级质量、数据一致性
- 终止条件定义

### Batch H: 韧性评分 (P1-9) — L4 核心要求

**模板文件**: `huaweicloud-skill-generator/references/resilience-score-template.md` (新建)

#### 推广到更多 Skill

| 子任务 | 文件 |
|--------|------|
| H.1 | `huaweicloud-ecs-ops/references/advanced/resilience-score.md` (新建) |
| H.2 | `huaweicloud-cce-ops/references/advanced/resilience-score.md` (新建) |
| H.3 | `huaweicloud-rds-ops/references/advanced/resilience-score.md` (新建) |
| H.4 | `huaweicloud-dcs-ops/references/advanced/resilience-score.md` (新建) |
| H.5 | `huaweicloud-vpc-ops/references/advanced/resilience-score.md` (新建) |

### Batch I: Observability Trinity (P1-2)

**模板文件**: `huaweicloud-skill-generator/references/observability-trinity-template.md` (新建)

**内容**:
- Metrics → Logs → Traces 联动规则
- 数据源映射 (CES/LTS/AOM)
- 关联分析工作流

#### 试点 Skill

| 子任务 | 文件 |
|--------|------|
| I.1 | `huaweicloud-ecs-ops/references/advanced/observability-trinity.md` (新建) |
| I.2 | `huaweicloud-ces-ops/references/advanced/observability-trinity.md` (新建) |
| I.3 | `huaweicloud-rds-ops/references/advanced/observability-trinity.md` (新建) |

### Batch J: Prompt Handbook (P1-3)

**模板文件**: `huaweicloud-skill-generator/references/prompt-handbook-template.md` (新建)

**内容**:
- 20+ 分类 prompts 模板
- 诊断场景 prompts
- 巡检场景 prompts

#### 试点 Skill

| 子任务 | 文件 |
|--------|------|
| J.1 | `huaweicloud-ecs-ops/references/prompts.md` (新建) |
| J.2 | `huaweicloud-ces-ops/references/prompts.md` (新建) |
| J.3 | `huaweicloud-cce-ops/references/prompts.md` (新建) |

### Batch K: 趋势检测标准化 (P1-4)

| 子任务 | 文件 |
|--------|------|
| K.1 | `huaweicloud-ecs-ops/references/advanced/trend-detection.md` (新建/扩展) |
| K.2 | `huaweicloud-ces-ops/references/advanced/trend-detection.md` (新建) |
| K.3 | `huaweicloud-rds-ops/references/advanced/trend-detection.md` (新建) |
| K.4 | `huaweicloud-cce-ops/references/advanced/trend-detection.md` (新建) |

**每个文件内容**:
- 趋势检测算法 (slope, acceleration, sudden-change)
- 阈值定义
- 告警触发条件

---

## Phase 3: 级联模式标准化 (P1-1)

### Batch L: 级联故障模式

| 子任务 | 文件 |
|--------|------|
| L.1 | `huaweicloud-ecs-ops/references/advanced/cascade-patterns.md` (新建/扩展) |
| L.2 | `huaweicloud-ces-ops/references/advanced/cascade-patterns.md` (新建) |
| L.3 | `huaweicloud-cce-ops/references/advanced/cascade-patterns.md` (新建) |
| L.4 | `huaweicloud-rds-ops/references/advanced/cascade-patterns.md` (新建) |

**每个文件内容**:
- ≥2 跨产品级联故障模式
- 传播路径图
- 隔离/阻断策略

---

## Batch 分支映射

| Batch | 分支 | 任务 | 文件数 |
|-------|------|------|--------|
| A | `feature/aiops-p0a-matrix` | P0-3 委托矩阵模板 | 1 |
| B | `feature/aiops-p0b-slo` | P0-8 SLO (10 skill) | 10 |
| C | `feature/aiops-p0c-change` | P0-9 变更关联 (3 skill) | 3 |
| D | `feature/aiops-p0d-capacity` | P0-10 容量预测 (5 skill) | 5 |
| E | `feature/aiops-p0e-confidence` | P1-5 诊断置信度 | 2 |
| F.1 | `feature/aiops-p0f-obs` | OBS AIOps 增强 | 4 |
| F.2 | `feature/aiops-p0f-lts` | LTS AIOps 增强 | 4 |
| F.3 | `feature/aiops-p0f-cbr` | CBR AIOps 增强 | 4 |
| F.4 | `feature/aiops-p0f-swr` | SWR AIOps 增强 | 4 |
| F.5 | `feature/aiops-p0f-cts` | CTS AIOps 增强 | 3 |
| G | `feature/aiops-p1g-chaos` | P1-8 混沌工程 (5 skill) | 6 |
| H | `feature/aiops-p1h-resilience` | P1-9 韧性评分 (5 skill) | 6 |
| I | `feature/aiops-p1i-observability` | P1-2 Observability Trinity (3 skill) | 4 |
| J | `feature/aiops-p1j-prompts` | P1-3 Prompt Handbook (3 skill) | 4 |
| K | `feature/aiops-p1k-trend` | P1-4 趋势检测 (4 skill) | 4 |
| L | `feature/aiops-p1l-cascade` | P1-1 级联模式 (4 skill) | 4 |

---

## 验收检查

```bash
# P0 检查
# A: 模板存在
ls huaweicloud-skill-generator/references/cross-skill-delegation-matrix-template.md

# B: SLO 文件数量 (≥10)
grep -rl "SLO\|SLI" huaweicloud-*-ops/references/well-architected-assessment.md | wc -l

# C: change-correlation 文件 (≥3)
find huaweicloud-*-ops -name "change-correlation.md" | wc -l

# D: capacity-forecasting 文件 (≥5)
find huaweicloud-*-ops -name "capacity-forecasting.md" | wc -l

# E: diagnosis-confidence 文件 (≥2)
find huaweicloud-*-ops -name "diagnosis-confidence.md" | wc -l

# F: L1 skill AIOps 文件
for s in obs lts cbr swr cts; do
  echo "=== $s ==="
  ls huaweicloud-${s}-ops/references/advanced/aiops-patterns.md 2>/dev/null && echo "  aiops-patterns: ✅" || echo "  aiops-patterns: ❌"
  ls huaweicloud-${s}-ops/references/knowledge-base.md 2>/dev/null && echo "  knowledge-base: ✅" || echo "  knowledge-base: ❌"
  ls huaweicloud-${s}-ops/references/advanced/alarm-storm-handling.md 2>/dev/null && echo "  alarm-storm: ✅" || echo "  alarm-storm: ❌"
done

# P1 检查
# G: chaos-engineering 文件 (≥5)
find huaweicloud-*-ops -name "chaos-engineering.md" | wc -l

# H: resilience-score 文件 (≥5)
find huaweicloud-*-ops -name "resilience-score.md" | wc -l

# I: observability-trinity 文件 (≥3)
find huaweicloud-*-ops -name "observability-trinity.md" | wc -l

# J: prompts 文件 (≥3)
find huaweicloud-*-ops -name "prompts.md" | wc -l

# K: trend-detection 文件 (≥4)
find huaweicloud-*-ops -name "trend-detection.md" | wc -l

# L: cascade-patterns 文件 (≥4)
find huaweicloud-*-ops -name "cascade-patterns.md" | wc -l

# 全局验证
python3 scripts/validate_local.py
bash scripts/pre_commit_check.sh
```

---

## DoD

- [ ] Batch A~F 全部完成 (P0 items)
- [ ] Batch G~L 全部完成 (P1 items)
- [ ] P0 合规检查: 10/10 通过
- [ ] P1 合规检查: 8/9 通过 (混沌+韧性已覆盖核心 skill)
- [ ] L1 skill (OBS/LTS/CBR/SWR/CTS) 全部升级到 L2+
- [ ] 所有新增文件通过 linter
- [ ] `python3 scripts/validate_local.py` 通过
- [ ] worktree 合并后删除

## 预期结果

| 指标 | 当前 | 目标 |
|------|------|------|
| AIOps 成熟度 | ~45% | 70%+ |
| P0 合规率 | ~40% | 100% |
| P1 合规率 | ~20% | 80%+ |
| L1→L2+ 升级 | 0 | 5 |
| 新增/修改文件 | — | ~70 个 |

## 风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| 任务量巨大 (16 batch) | 管理复杂度高 | 可按优先级分阶段执行 |
| 部分 skill 难以定义 SLO | 进度延迟 | 先从 ECS/RDS 核心 skill 验证模板 |
| 混沌工程需要实际测试环境 | P1 降级 | 仅做文档化，测试留作后续 |
