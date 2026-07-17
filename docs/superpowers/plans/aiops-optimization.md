# Plan: AIOps L4 成熟度提升

> Status: ✅ **COMPLETE** — all phases merged to main (2026-07-18)
> Final Update: 2026-07-18 (full gap fill — all L2/L3/L4 files complete)
> Updated: 2026-07-18
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

## Phase 0: L2/L3 缺口补齐 (Gap Filling)

> 基于 `aiops-optimization.md` 完成后的审计，发现 9 个文件缺口分布在 5 个技能。补齐后 L2/L3 可达 100% 完整度。

### 缺口清单

| 缺口 | Level | Skill | 文件 | 状态 |
|------|-------|-------|------|------|
| G-1 | L2 | RDS | `huaweicloud-rds-ops/references/knowledge-base.md` | ❌ 缺失 |
| G-2 | L2 | RDS | `huaweicloud-rds-ops/references/prompts.md` | ❌ 缺失 |
| G-3 | L2 | ELB | `huaweicloud-elb-ops/references/knowledge-base.md` | ❌ 缺失 |
| G-4 | L2 | ELB | `huaweicloud-elb-ops/references/prompts.md` | ❌ 缺失 |
| G-5 | L3 | CCE | `huaweicloud-cce-ops/references/advanced/observability-trinity.md` | ❌ 缺失 |
| G-6 | L3 | CES | `huaweicloud-ces-ops/references/advanced/change-correlation.md` | ❌ 缺失 |
| G-7 | L3 | CES | `huaweicloud-ces-ops/references/advanced/capacity-forecasting.md` | ❌ 缺失 |
| G-8 | L3 | ELB | `huaweicloud-elb-ops/references/advanced/change-correlation.md` | ❌ 缺失 |
| G-9 | L3 | ELB | `huaweicloud-elb-ops/references/advanced/capacity-forecasting.md` | ❌ 缺失 |

### Batch G-1: RDS L2 补齐

**文件**:
- `huaweicloud-rds-ops/references/knowledge-base.md` (新建)
- `huaweicloud-rds-ops/references/prompts.md` (新建)

**knowledge-base.md 内容** (参考 ECS knowledge-base.md 格式):
- ≥3 fault patterns (RDS-001/002/003...)
- 每条 pattern: 症状 → 根因 → 证据 → 恢复步骤
- 华为云 RDS 特定故障模式 (主备切换、连接池耗尽、慢查询、存储满)

**prompts.md 内容** (参考 prompts.md 模板格式):
- ≥20 prompts across 5 categories: 诊断类、根因分析类、容量类、可用性类、巡检类
- RDS 特定的 metric namespace 和告警场景

### Batch G-2: ELB L2 补齐

**文件**:
- `huaweicloud-elb-ops/references/knowledge-base.md` (新建)
- `huaweicloud-elb-ops/references/prompts.md` (新建)

**knowledge-base.md 内容**:
- ≥3 fault patterns (ELB-001/002/003...)
- 华为云 ELB 特定故障模式 (后端实例不健康、证书过期、连接超时、带宽瓶颈)

**prompts.md 内容**:
- ≥20 prompts across 5 categories
- ELB 特定的监听器、后端服务器、SSL证书相关 prompts

### Batch G-3: CCE L3 补齐

**文件**:
- `huaweicloud-cce-ops/references/advanced/observability-trinity.md` (新建)

**内容** (参考 ECS observability-trinity.md 格式):
- CCE Metrics → Logs → Traces 联动规则
- CES 指标 → LTS 日志 → APM 链路追踪映射
- Pod 级别可观测性关联分析工作流

### Batch G-4: CES L3 补齐

**文件**:
- `huaweicloud-ces-ops/references/advanced/change-correlation.md` (新建)
- `huaweicloud-ces-ops/references/advanced/capacity-forecasting.md` (新建)

**change-correlation.md 内容** (参考 RDS change-correlation.md 格式):
- CTS 事件类型 → CES 告警映射 (≥5 条)
- 告警规则变更 → 监控异常关联
- 采样周期变更 → 指标抖动关联

**capacity-forecasting.md 内容** (参考 ECS capacity-forecasting.md 格式):
- CES 指标容量预测 (监控指标数量、告警规则数量)
- 线性外推模型
- 容量阈值告警规则

### Batch G-5: ELB L3 补齐

**文件**:
- `huaweicloud-elb-ops/references/advanced/change-correlation.md` (新建)
- `huaweicloud-elb-ops/references/advanced/capacity-forecasting.md` (新建)

**change-correlation.md 内容**:
- CTS 事件类型 → ELB 告警映射 (≥5 条)
- 后端服务器变更 → 流量异常关联
- SSL证书变更 → 连接失败关联

**capacity-forecasting.md 内容**:
- ELB 带宽/连接数预测
- 峰值带宽预测模型
- 容量规划工作流

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
| **Phase 0: Gap Filling** |
| G-1 | `feature/aiops-gap-rds` | RDS L2 补齐 (kb + prompts) | 2 |
| G-2 | `feature/aiops-gap-elb` | ELB L2 补齐 (kb + prompts) | 2 |
| G-3 | `feature/aiops-gap-cce` | CCE L3 补齐 (observability-trinity) | 1 |
| G-4 | `feature/aiops-gap-ces` | CES L3 补齐 (change-corr + capacity) | 2 |
| G-5 | `feature/aiops-gap-elb-l3` | ELB L3 补齐 (change-corr + capacity) | 2 |
| **Phase 1: Completed (2026-07-18)** |
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
| **Phase 2: Completed (2026-07-18)** |
| G-A | `feature/aiops-p1g-chaos` | P1-8 混沌工程 (5 skill) | 6 |
| H | `feature/aiops-p1h-resilience` | P1-9 韧性评分 (5 skill) | 6 |
| I | `feature/aiops-p1i-observability` | P1-2 Observability Trinity (3 skill) | 4 |
| J | `feature/aiops-p1j-prompts` | P1-3 Prompt Handbook (3 skill) | 4 |
| K | `feature/aiops-p1k-trend` | P1-4 趋势检测 (4 skill) | 4 |
| L | `feature/aiops-p1l-cascade` | P1-1 级联模式 (4 skill) | 4 |

---

## 验收检查

```bash
# Phase 0: L2/L3 Gap Filling 检查
# G-1: RDS knowledge-base + prompts
ls huaweicloud-rds-ops/references/knowledge-base.md && echo "  RDS kb: ✅" || echo "  RDS kb: ❌"
ls huaweicloud-rds-ops/references/prompts.md && echo "  RDS prompts: ✅" || echo "  RDS prompts: ❌"

# G-2: ELB knowledge-base + prompts
ls huaweicloud-elb-ops/references/knowledge-base.md && echo "  ELB kb: ✅" || echo "  ELB kb: ❌"
ls huaweicloud-elb-ops/references/prompts.md && echo "  ELB prompts: ✅" || echo "  ELB prompts: ❌"

# G-3: CCE observability-trinity
ls huaweicloud-cce-ops/references/advanced/observability-trinity.md && echo "  CCE observability-trinity: ✅" || echo "  CCE observability-trinity: ❌"

# G-4: CES change-correlation + capacity-forecasting
ls huaweicloud-ces-ops/references/advanced/change-correlation.md && echo "  CES change-corr: ✅" || echo "  CES change-corr: ❌"
ls huaweicloud-ces-ops/references/advanced/capacity-forecasting.md && echo "  CES capacity: ✅" || echo "  CES capacity: ❌"

# G-5: ELB change-correlation + capacity-forecasting
ls huaweicloud-elb-ops/references/advanced/change-correlation.md && echo "  ELB change-corr: ✅" || echo "  ELB change-corr: ❌"
ls huaweicloud-elb-ops/references/advanced/capacity-forecasting.md && echo "  ELB capacity: ✅" || echo "  ELB capacity: ❌"

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

- [ ] Phase 0: Batch G-1~G-5 全部完成 (9 个缺口文件)
- [ ] Phase 1: Batch A~F 全部完成 (P0 items)
- [ ] Phase 2: Batch G-A~L 全部完成 (P1 items)
- [ ] L2 完整度: 100% (所有 skill 的 delegation matrix + knowledge-base + prompts)
- [ ] L3 完整度: 100% (所有核心 skill 的 SLO + change-correlation + capacity-forecasting)
- [ ] P0 合规检查: 10/10 通过
- [ ] P1 合规检查: 9/9 通过
- [ ] 所有新增文件通过 linter
- [ ] `python3 scripts/validate_local.py` 通过
- [ ] worktree 合并后删除

## 预期结果

| 指标 | Phase 0 前 | Phase 0 后 | 目标 |
|------|-----------|-----------|------|
| AIOps 成熟度 | ~70% | ~80% | 80%+ |
| L2 完整度 | ~85% | 100% | 100% |
| L3 完整度 | ~75% | 100% | 100% |
| P0 合规率 | 10/10 | 10/10 | 100% |
| P1 合规率 | 9/9 | 9/9 | 100% |
| L1→L2+ 升级 | 5 | 5 | 5 |
| Phase 0 新增文件 | — | 9 个 | 9 个 |

## 风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| 任务量巨大 (16 batch + 5 gap) | 管理复杂度高 | 可按优先级分阶段执行 |
| 部分 skill 难以定义 SLO | 进度延迟 | 先从 ECS/RDS 核心 skill 验证模板 |
| 混沌工程需要实际测试环境 | P1 降级 | 仅做文档化，测试留作后续 |
| Phase 0 发现更多缺口 | 范围蔓延 | 仅补齐已确认的 9 个缺口，不扩展范围 |
