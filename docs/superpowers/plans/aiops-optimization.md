# Plan: AIOps 集成优化

> Status: 📋 **PLANNED** — not started
> Created: 2026-07-18
> Parent analysis: `docs/superpowers/plans/gcl-token-efficiency-p0.md`

## 目标

将 hcloud-skills AIOps 成熟度从当前约 45% 提升至 65%+，重点解决 4 个高优先级 Gap 和 L1 skill 优化。

## 背景

当前 AIOps 集成存在以下问题：
- 无全局跨技能委托矩阵
- SLO/SLI 定义几乎缺失
- 变更关联分析 (CTS) 空白
- 诊断置信度未实现
- 5 个 L1 skill (OBS/LTS/CBR/SWR/CTS) AIOps 能力不足

## 高优先级改进项

### P1: 全局跨技能委托矩阵

**目标**: 创建统一的全 skill 告警路由矩阵

**文件**: `huaweicloud-skills-shared/references/cross-skill-delegation-matrix.md` (新建)

**内容**:
- 告警类型 → 目标 skill 映射
- 跨产品级联故障路由规则
- 委托优先级和依赖关系

**验收标准**:
- 覆盖全部 20+ skill
- 每个告警类型有明确的处理 skill
- 可被各 skill integration.md 引用

### P2: SLO/SLI 定义标准化

**目标**: 每个 P0 skill 定义至少 1 个 SLO

**涉及 skill**:
- ECS, CCE, RDS, CES, ELB (优先级 1)
- DCS, DMS, CSS, GaussDB, VPC (优先级 2)

**文件**: 各 skill 的 `references/advanced/slo-definition.md` (新建)

**内容**:
- SLI: 可用性、延迟、错误率定义
- SLO: 目标值 (如可用性 ≥ 99.9%)
- 测量方法和告警阈值

### P3: 变更关联分析 (CTS)

**目标**: 实现故障与 CTS 变更事件的关联

**涉及**: ECS, CCE, RDS (优先级 1)

**文件**: 各 skill 的 `references/advanced/change-correlation.md` (新建)

**内容**:
- CTS 事件类型 → 故障类型映射
- 时间窗口关联分析 (故障前后 30 分钟)
- 常见变更触发告警模式

### P4: 诊断置信度模型

**目标**: 实现标准化的诊断置信度评分

**模板**: `huaweicloud-skill-generator/references/diagnosis-confidence-template.md` (新建)

**内容**:
- 置信度分级 (High/Medium/Low)
- 多证据融合评分算法
- 不确定时的升级路径

## L1 Skill 优化

### OBS-ops AIOps 增强

| 文件 | 当前状态 | 目标 |
|------|----------|------|
| `references/knowledge-base.md` | 太简单 | 扩展至 5+ 故障模式 |
| `references/advanced/aiops-patterns.md` | 缺失 | 新建，4+ patterns |
| `integration.md` | 无 delegation | 添加 cross-skill matrix |

### LTS-ops AIOps 增强

| 文件 | 当前状态 | 目标 |
|------|----------|------|
| `references/advanced/aiops-patterns.md` | 缺失 | 新建，4+ patterns |
| `integration.md` | 无 delegation | 添加 cross-skill matrix |

### CBR-ops AIOps 增强

| 文件 | 当前状态 | 目标 |
|------|----------|------|
| `references/advanced/aiops-patterns.md` | 缺失 | 新建，4+ patterns |
| `references/knowledge-base.md` | 缺失 | 新建，5+ 故障模式 |

### SWR-ops AIOps 增强

| 文件 | 当前状态 | 目标 |
|------|----------|------|
| `references/advanced/aiops-patterns.md` | 缺失 | 新建，4+ patterns |
| `references/knowledge-base.md` | 缺失 | 新建，5+ 故障模式 |

### CTS-ops AIOps 增强

| 文件 | 当前状态 | 目标 |
|------|----------|------|
| `references/advanced/aiops-patterns.md` | 太简单 | 扩展至 4+ patterns |
| `references/knowledge-base.md` | 太简单 | 扩展至 5+ 故障模式 |

## 执行方案

### git worktree per batch

| Batch | 内容 | 分支 |
|-------|------|------|
| A | P1 (全局委托矩阵) + CTS-ops 优化 | `feature/aiops-p1-matrix-cts` |
| B | P2 (SLO/SLI) + OBS-ops 优化 | `feature/aiops-p2-slo-obs` |
| C | P3 (变更关联) + LTS-ops 优化 | `feature/aiops-p3-change-lts` |
| D | P4 (置信度模板) + CBR/SWR-ops 优化 | `feature/aiops-p4-confidence-cbr-swr` |

### 验证

```bash
# 1. 全局矩阵存在性检查
ls huaweicloud-skills-shared/references/cross-skill-delegation-matrix.md

# 2. SLO 文件数量
find huaweicloud-*-ops -name "slo-definition.md" | wc -l  # expect ≥5

# 3. change-correlation 文件数量
find huaweicloud-*-ops -name "change-correlation.md" | wc -l  # expect ≥3

# 4. L1 skill advanced 文件
for s in obs lts cbr swr cts; do
  f="huaweicloud-${s}-ops/references/advanced/aiops-patterns.md"
  [ -f "$f" ] && echo "$s: ✅" || echo "$s: ❌"
done

# 5. repo validation
python3 scripts/validate_local.py
bash scripts/pre_commit_check.sh
```

## 依赖

- Batch A 无前置依赖
- Batch B/C/D 可并行于 A 执行

## DoD (Done on Merge)

- [ ] 4 个全局改进项完成
- [ ] 5 个 L1 skill 优化完成
- [ ] 所有新增文件通过 linter
- [ ] `python3 scripts/validate_local.py` 通过
- [ ] worktree 合并后删除

## 预期结果

- AIOps 成熟度: 45% → 65%+
- 新增文件: ~15 个
- SLO 定义: 0 → 10+ skill
- 变更关联: 0 → 3+ skill
