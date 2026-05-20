# AIOps Best Practices — Huawei Cloud Skill Generator

> **Purpose:** Mandatory specification for all `huaweicloud-[product]-ops` skills with monitoring, alerting, or diagnostic capabilities. Defines patterns, templates, and compliance standards for FinOps-optimized, SecOps-secured, and AIOps-intelligent operations.
> **Version:** 1.0.0
> **Last Updated:** 2026-05-20
> **Status:** MANDATORY — all monitoring/diagnostic skills MUST implement relevant patterns

---

## 1. Core Principles

### 1.1 AIOps Maturity Model

| Level | Name | Characteristics | Target |
|-------|------|-----------------|--------|
| L1 | 基础监控 | Single-metric query, static threshold alerting | All Ops Skills |
| L2 | 关联分析 | Multi-metric joint inspection, composite anomaly patterns | Skills with monitoring.md |
| L3 | 智能诊断 | Cross-skill delegation, AI diagnosis, decision trees | Monitoring + product协同 |
| L4 | 主动预防 | Proactive inspection, trend prediction, knowledge base | Core P0 product skills |
| L5 | 自治修复 | Auto-remediation, self-learning, closed-loop | Future target |

### 1.2 AIOps Five-Step Cycle

```
[异常发现] → [验证确认] → [关联分析] → [根因定位] → [修复建议]
     ↑                                                    |
     └────────────── 反馈优化 ←────────────────────────────┘
```

### 1.3 Cross-Skill Principles

1. **Single Responsibility:** Each skill handles its product's diagnosis only
2. **Clear Delegation:** Define cross-skill calling relationship via delegation matrix
3. **Standardized Output:** All skills output unified diagnosis schema
4. **Knowledge Sharing:** Fault patterns visible across skills
5. **Fault Tolerance:** Delegated skill unavailable → clear degradation path

### 1.4 SLO/SLI 体系 (Service Level Objectives)

Every monitoring/diagnostic skill MUST define SLO/SLI as the foundation for AIOps alerting.

#### SLO 定义模板

```markdown
## SLO/SLI Definition — [Product]

### SLI (Service Level Indicator) 指标选择
| SLI 名称 | 计算公式 | 数据来源 | 采集频率 |
|---------|---------|---------|---------|
| 可用性 | 成功请求数 / 总请求数 × 100% | CES + ELB | 1min |
| 延迟P99 | 第99百分位响应时间(ms) | AOM Trace | 1min |
| 错误率 | 5xx响应数 / 总请求数 × 100% | ELB + AOM | 1min |
| 饱和度 | CPU利用率 / 连接利用率 / 存储利用率 | CES | 5min |

### SLO 目标
| SLI | SLO 目标 | Error Budget (月度) | 告警阈值 |
|-----|---------|--------------------| ---------|
| 可用性 | ≥ 99.9% | 43.2min/月 | < 99.95% 触发Warning |
| 延迟P99 | ≤ 200ms | — | > 300ms 触发Warning |
| 错误率 | ≤ 0.1% | — | > 0.5% 触发Critical |
| 饱和度 | ≤ 80% | — | > 85% 触发Warning |

### Error Budget 燃烧率告警
| 燃烧率 | 消耗速度 | 告警等级 | 含义 |
|--------|---------|---------|------|
| 1× | 正常消耗 (43.2min/月) | — | 正常 |
| 2× | 21.6min耗尽 | Info | 需关注 |
| 5× | 8.6min耗尽 | Warning | 需介入 |
| 14.4× | 3h耗尽 | Critical | 立即行动 |
```

#### SLO 与 AIOps 的集成

- AIOps告警应基于 SLO Error Budget 燃烧率，而非单一静态阈值
- 多SLI联合违规 → 提升告警等级 (如: 可用性↓ + 延迟↑ → Critical)
- SLO合规率纳入巡检报告，形成趋势分析

---

## 2. Multi-Metric Correlation Specs

### 2.1 Required Anomaly Patterns

Any skill with monitoring capabilities MUST define ≥ 4 anomaly patterns:

| Pattern Category | Minimum | Example |
|-----------------|---------|---------|
| Resource Pressure | ≥ 2 | CPU-Memory dual-high, Disk-IO bottleneck |
| Trend Anomaly | ≥ 1 | Memory leak trend, metric monotonic increase |
| Sudden Change | ≥ 1 | CPU spike, traffic drop |
| Correlation-Anomaly | ≥ 1 | Load-CPU mismatch, connection-CPU divergence |

### 2.2 Huawei Cloud CES Metric Namespaces

| Service | CES Namespace | Key Metrics |
|---------|--------------|-------------|
| ECS | `SYS.ECS` | cpu_usage, mem_usedPercent, diskUsage_percent |
| RDS | `SYS.RDS` | rds001_cpu_usage, rds002_mem_usage, rds045_iops |
| DCS | `SYS.DCS` | cpu_usage, memory_usage, cpu_usage_percent |
| ELB | `SYS.ELB` | l7e_listener_qps, l7e_listener_errors, active_connection_count |
| CCE | `SYS.CCE` | node_cpu_utilization, node_mem_utilization, pod_count |
| EVS | `SYS.EVS` | read_iops, write_iops, read_bytes |

### 2.3 Pattern Definition Template

```markdown
| Pattern | Metrics Involved | Detection Logic | Severity | Interpretation |
|---------|-----------------|-----------------|----------|----------------|
| cpu_mem_dual_high | cpu_usage, mem_usedPercent | cpu>80% AND mem>85% | Critical | 资源双高压,可能OOM |
| disk_io_bottleneck | read_iops, write_iops, diskUtil | IOPS>阈值AND diskUtil>90% | Warning | 磁盘IO瓶颈 |
| mem_leak_trend | mem_usedPercent (30min trend) | slope>0.5%/min continuously | Critical | 内存泄漏趋势 |
| sudden_cpu_spike | cpu_usage | delta(5min)>50% | Warning | 突发性CPU飙升 |
```

---

## 3. Alert-Driven Cross-Skill Diagnosis

### 3.1 Five-Step Decision Tree

```
[告警触发]
    │
    ├── Step 1: 验证告警有效性
    │   确认指标值是否确实超阈值 → 误报则检查告警规则配置
    │
    ├── Step 2: 检查资源状态
    │   委托对应产品Skill获取资源当前状态
    │
    ├── Step 3: 多指标关联分析
    │   查询CES相关指标,识别复合异常模式
    │
    ├── Step 4: 深度诊断(如适用)
    │   委托AOM应用监控/LTS日志服务
    │
    └── Step 5: 生成统一诊断报告
        汇总所有Skill发现,给出根因和修复建议
```

### 3.2 Namespace-to-Skill Routing Matrix

| CES Namespace | Primary Diagnosis Skill | Delegation |
|--------------|------------------------|-----------|
| `SYS.ECS` | `huaweicloud-ecs-ops` | 可委托网络Skill检查ELB/VPC层 |
| `SYS.RDS` / `SYS.GaussDB` | `huaweicloud-rds-ops` | 必须委托DB诊断做慢SQL分析 |
| `SYS.ELB` | `huaweicloud-elb-ops` | 可委托ECS检查后端健康 |
| `SYS.DCS` | `huaweicloud-dcs-ops` | 可委托连接分析 |
| `SYS.CCE` | `huaweicloud-cce-ops` | 可委托ECS检查节点状态 |

### 3.3 AOM/LTS Delegation Triggers

| Trigger Condition | AOM/LTS API | Skill Action |
|------------------|------------|-------------|
| 应用性能告警 | AOM ListAlarms | 调用AOM获取Trace |
| 数据库性能下降 | LTS ListLogs | 查询慢SQL日志 |
| 容器异常 | AOM GetPodEvents | 委托CCE检查Pod状态 |
| 疑似安全事件 | HSS ListEvents | 启动安全隔离流程 |

---

## 4. Cross-Skill Delegation Matrix

### 4.1 Delegation Matrix Format

Each skill with cross-skill capability MUST define in `integration.md`:

| Alarm Type | Metric | Primary Skill | Secondary Skill | HSS/AOM Deleg. |
|-----------|--------|--------------|----------------|---------------|
| CPU高 | cpu_usage | huaweicloud-ecs-ops | huaweicloud-aom-ops | Optional |
| 数据库慢查询 | rds043_slow_queries | huaweicloud-rds-ops | — | Recommended |
| 连接数满 | active_connection_count | huaweicloud-elb-ops | huaweicloud-ecs-ops | — |
| 安全告警 | hss_event_count | huaweicloud-hss-ops | huaweicloud-ecs-ops | Required |

### 4.2 Delegation Protocol

```
[告警触发]
    │
    ├── 1. 识别Namespace + Metric
    ├── 2. 查矩阵确定主诊断Skill
    ├── 3. 调用主Skill检查资源状态
    ├── 4. 若资源异常 → 调用次Skill
    ├── 5. 若委派="Recommended" → 始终调用
    └── 6. 汇总所有输出生成统一报告
```

---

## 5. Proactive Inspection Workflow

### 5.1 Five-Step Inspection Loop

```
[资源发现] → [指标采集] → [异常检测] → [跨Skill诊断] → [报告生成]
```

### 5.2 Phase Requirements

| Phase | Requirement | Output |
|-------|------------|--------|
| Discovery | List all resources in monitoring scope | Resource inventory |
| Metric Collection | Batch collect key metrics (Period=300s) | Metric data |
| Anomaly Detection | Static threshold + trend slope + comparison | Anomaly list |
| Cross-Skill Diagnosis | Delegate abnormal resources to respective Skills | Diagnostic findings |
| Report Generation | Generate inspection report | Report document |

### 5.3 Trend Detection Algorithm

```go
func calculateSlope(points []DataPoint) float64 {
    n := float64(len(points))
    if n < 2 { return 0 }
    var sumX, sumY, sumXY, sumX2 float64
    for i, p := range points {
        x := float64(i); y := p.Average
        sumX += x; sumY += y; sumXY += x*y; sumX2 += x*x
    }
    return (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
}
```

---

## 6. Alarm Storm Handling

### 6.1 Storm Detection Criteria

| Criterion | Threshold | Action |
|-----------|-----------|--------|
| Alarm frequency | > 10 alarms / 5 minutes | Enter storm mode |
| Same resource | > 3 alarms on one instance | Aggregate to single event |
| Same namespace | > 50% from same namespace | Focus diagnosis on product |
| Cascade pattern | Alarm A triggers, B triggers within 2min | Mark B as "likely caused by A" |

### 6.2 Storm Processing Flow

1. **Detect:** Monitor CES alarm list with State=ALARM
2. **Aggregate:** Group by resource_id, namespace, time window
3. **Suppress:** After aggregation, retain only primary alarm notification
4. **Root Identification:** Find earliest alarm as likely root cause
5. **Focus Diagnosis:** Delegate root resource diagnosis to corresponding Skill

---

## 7. Knowledge Base

### 7.1 Structure

Each product skill SHOULD maintain `references/knowledge-base.md`:

```markdown
### Pattern: [Product]-[N] — [Fault Name]

| Attribute | Content |
|-----------|---------|
| Trigger Metric | [CES metric name] |
| Threshold | [Value] |
| Symptoms | [Description] |
| Correlated Metrics | [Related metrics & expected behavior] |
| Root Cause | [1. Cause A, 2. Cause B...] |
| Diagnosis Steps | [1. Step A, 2. Step B...] |
| Fix | [1. Temporary, 2. Permanent] |
| Prevention | [1. Measure A, 2. Measure B...] |
```

### 7.2 Cascade Fault Patterns

Knowledge base MUST include cross-product cascade patterns, e.g.:
- ECS overload → ELB drops connections → RDS connection pile-up
- Storage full → DB write failure → application error cascade
- Security breach → CPU spike from crypto-mining → service degradation

---

## 8. Observability Trinity

### 8.1 Three-Layer Architecture

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│ Metrics   │────▶│  Logs    │────▶│ Traces   │
│  (CES)    │     │  (LTS)   │     │  (AOM)   │
└──────────┘     └──────────┘     └──────────┘
                       ▼
              ┌─────────────────┐
              │ Unified Report  │
              └─────────────────┘
```

### 8.2 Metrics → Logs Linkage

| CES Anomaly | LTS Query Target | Purpose |
|-------------|-----------------|---------|
| CPU spike | Application error logs | Confirm error burst causing CPU surge |
| Memory leak | Application memory logs | Confirm allocation pattern |
| Max connections | Database access logs | Confirm connection leak source |
| Dropped connections | Nginx/Access logs | Confirm dropped request details |

### 8.3 Metrics → Traces Linkage

| CES Anomaly | AOM Trace Target | Purpose |
|-------------|-----------------|---------|
| CPU spike | Application Trace | Locate hot methods |
| Latency increase | RPC/HTTP Trace | Locate bottleneck service |
| Error rate increase | Error Trace | Locate error root cause |

### 8.4 Degradation Strategy

If AOM/LTS skills unavailable:
1. Use CLI directly (`hcloud ces list-metrics`, `hcloud lts list-logs`)
2. Use OpenAPI SDK directly
3. Provide console link for manual troubleshooting

---

## 9. Unified Diagnosis Report Schema

| Field | Source | Description |
|-------|--------|-------------|
| `report_id` | Generated | UUID v4 tracking ID |
| `timestamp` | CES | Alarm trigger time |
| `alarm_source` | CES | Original alarm rule name |
| `resource_id` | CES | Instance ID |
| `resource_status` | Product Skill | Current resource state |
| `metric_value` | CES | Alarm metric value |
| `metric_trend` | CES | 1h trend analysis |
| `anomaly_patterns` | Multi-Metric Inspection | Detected anomaly patterns |
| `deep_diagnosis` | AOM/LTS/HSS | Deep diagnosis findings |
| `correlated_alarms` | CES | Other alarms on same resource |
| `root_cause` | Comprehensive | Primary root cause |
| `recommendation` | Comprehensive | Actionable fix suggestions |
| `delegated_skills` | Agent | List of Skills invoked |

---

## 10. Prompt Engineering

### 10.1 Prompt Categories

| Category | Minimum | Description |
|----------|---------|-------------|
| Metric Query | ≥ 3 | Single metric, trend, multi-metric batch |
| Alert Management | ≥ 3 | Create, query, check, delete alarm rules |
| Multi-Metric Inspection | ≥ 2 | Execute inspection, analyze correlation |
| Alert-Driven Diagnosis | ≥ 3 | Root cause, cross-skill orchestration, cascade |
| Proactive Inspection | ≥ 2 | Scheduled inspection, report generation |
| Alarm Storm Handling | ≥ 2 | Storm detection, aggregation |
| Knowledge Base Application | ≥ 2 | Match fault pattern, update knowledge base |
| Observability Linkage | ≥ 2 | Metrics→Logs, Metrics→Traces |
| Report Generation | ≥ 3 | Diagnosis report, inspection report, post-mortem |

---

## 11. Multi-Round Self-Reflection

### 11.1 Three-Round Review Flow

```
[Round 1: Initial Diagnosis]
    │
    ├── Collect all Skill outputs
    ├── Execute standard diagnosis per decision tree
    ├── Output initial root cause hypothesis
    │
    ├── Dissatisfied? → [Round 2: Critical Reflection]
    │   ├── Challenge Round 1 assumptions
    │   ├── Check missing correlated metrics
    │   ├── Check missing dependent resources
    │   ├── Compare with similar Knowledge Base patterns
    │   ├── Re-examine timeline (causality inversion?)
    │   └── Output revised root cause
    │
    └── Still dissatisfied? → [Round 3: Deep Review]
        ├── Execute Metrics→Logs→Traces trinity query
        ├── Expand time window
        ├── Check change history (config, deploy, scale)
        ├── Output final root cause with confidence
        └── If uncertain → explicitly mark uncertainty
```

### 11.2 Critical Questions per Round

| # | Question | Purpose |
|---|----------|---------|
| 1 | Is the evidence chain complete? Any weak links? | Verify logic rigor |
| 2 | Are there alternative hypotheses that better explain all anomalies? | Avoid confirmation bias |
| 3 | Any queryable metrics or resources missed? | Fill information gaps |
| 4 | Is the causal relationship on timeline correct? | Verify temporal logic |
| 5 | Knowledge base has similar but different patterns? | Learn from history |
| 6 | Are fix recommendations executable? Any risks? | Ensure actionability |
| 7 | Any findings worth adding as new knowledge patterns? | Knowledge accumulation |

---

## 12. Change Correlation Analysis (变更关联分析)

### 12.1 变更事件收集

```markdown
## Change Correlation — Event Collection

### 变更数据源
| 变更类型 | 数据来源 | 采集方式 |
|---------|---------|---------|
| 配置变更 | CTS (Cloud Trace Service) | CTS ListTraces API |
| 部署变更 | CCE / AOM | CCE ListClusters + AOM ListAlarms |
| 规格变更 | 产品API变更记录 | DescribeInstance 对比 diff |
| 网络变更 | VPC Flow Log | LTS Log Query |
| 安全策略变更 | IAM / WAF | CTS ListTraces (iam/waf filter) |

### 变更时间窗
- 变更后 30min 内出现异常 → 高度关联
- 变更后 2h 内出现异常 → 中度关联
- 变更后 24h 内出现异常 → 低度关联
```

### 12.2 变更-异常关联决策树

```
[异常发现]
    │
    ├── Step 1: 查询变更历史 (CTS)
    │   ├── 30min内有变更 → 高度怀疑变更导致
    │   │   ├── 配置变更? → 回滚配置, 验证恢复
    │   │   ├── 规格变更? → 评估是否回退规格
    │   │   └── 部署变更? → 回滚版本, 验证恢复
    │   └── 无近期变更 → 排除变更因素
    │
    ├── Step 2: 变更影响范围评估
    │   ├── 受影响资源数 / 总资源数 → 影响面评估
    │   └── 依赖拓扑分析 → 识别级联影响
    │
    └── Step 3: 生成变更关联报告
        ├── 变更时间、类型、操作人
        ├── 异常时间、指标、影响范围
        └── 关联度评分: High / Medium / Low
```

---

## 13. Chaos Engineering Integration (混沌工程集成)

### 13.1 稳定性验证模式

```markdown
## Chaos Engineering — Stability Verification

### 故障注入实验设计
| 实验类型 | 注入方式 | 观测指标 | 预期行为 | 终止条件 |
|---------|---------|---------|---------|---------|
| 实例故障 | 停止主实例 | HA切换时间、服务可用性 | ≤ 60s切换, 可用性≥99.9% | 可用性<99%超5min |
| 网络分区 | 安全组阻断AZ间流量 | 跨AZ请求成功率 | 降级但不断服 | 成功率<50%超2min |
| 磁盘故障 | 填满磁盘至95% | 写入成功率、告警触发 | 告警触发≥告警 | 写入失败超1min |
| 负载突增 | 压测工具打满CPU | AS扩容时间、服务延迟 | 5min内完成扩容 | 延迟>5s超10min |
| 依赖故障 | 模拟下游服务超时 | 熔断触发、降级行为 | 熔断触发+降级响应 | 降级失败超3min |
```

### 13.2 韧性评分模型

```markdown
## Resilience Score

### 评分维度 (每项0-10分)
| 维度 | 评分标准 | 权重 |
|------|---------|------|
| 故障检测速度 | 从故障发生到告警触发的时间 | 20% |
| 故障隔离能力 | 爆炸半径控制、级联防护 | 20% |
| 恢复自动化 | 自愈成功率、MTTR | 25% |
| 降级质量 | 降级后服务可用性 | 15% |
| 数据一致性 | 故障恢复后数据完整性 | 20% |

### 韧性等级
| 分数区间 | 等级 | 建议 |
|---------|------|------|
| 8-10 | A (优秀) | 定期混沌验证, 持续保持 |
| 6-8 | B (良好) | 补充缺失的故障场景验证 |
| 4-6 | C (一般) | 增加自愈能力, 完善降级策略 |
| 0-4 | D (薄弱) | 优先修复关键韧性缺口 |
```

---

## 14. Capacity Forecasting (容量预测)

### 14.1 预测模型选择

| 模型 | 适用场景 | 准确度 | 实现复杂度 | 推荐度 |
|------|---------|--------|----------|--------|
| 线性外推 | 稳定增长型 | 中 | 低 | ★★★ |
| 季节性分解 | 周期性负载 | 高 | 中 | ★★★★ |
| 移动平均 | 平滑趋势 | 中 | 低 | ★★★ |
| 指数平滑 | 短期预测 | 高 | 低 | ★★★★ |
| ML模型 | 复杂模式 | 很高 | 高 | ★★ (高级场景) |

### 14.2 容量规划工作流

```
[指标历史采集] (30-90天)
    │
    ├── Step 1: 识别增长趋势
    │   线性回归 slope → 日均增长率
    │   R² > 0.7 → 趋势可靠
    │
    ├── Step 2: 识别周期性模式
    │   FFT/季节性分解 → 周/月周期
    │   峰谷比 → 扩缩容空间
    │
    ├── Step 3: 计算资源天花板
    │   当前用量 + (增长率 × 预测周期) = 预测用量
    │   配额上限 / 预测用量 = 可用天数
    │
    ├── Step 4: 生成容量建议
    │   可用天数 < 30天 → 立即扩容/申请配额
    │   可用天数 30-90天 → 规划扩容
    │   可用天数 > 90天 → 正常, 下次检查
    │
    └── Step 5: 输出容量报告
        ├── 当前利用率趋势图
        ├── 预测资源耗尽时间点
        ├── 扩容/配额申请建议
        └── 成本影响预估
```

### 14.3 容量告警规则

| 指标 | Warning | Critical | 建议动作 |
|------|---------|----------|---------|
| CPU利用率趋势 | 30天内预测>80% | 14天内预测>90% | 扩容/优化 |
| 存储增长率 | 60天内预测>85% | 30天内预测>95% | 扩容/清理 |
| 连接数趋势 | 30天内预测>80% | 14天内预测>90% | 增加最大连接数/扩容 |
| 配额余量 | 剩余<30% | 剩余<15% | 申请配额提升 |

---

## 15. Diagnosis Confidence Scoring (诊断置信度评分)

### 15.1 置信度计算模型

```
Confidence = Σ(Evidence_i × Weight_i) / Σ(Weight_i)

Evidence来源与权重:
| 证据类型 | 权重 | 说明 |
|---------|------|------|
| 直接指标异常 | 0.30 | CES指标超阈值，确认异常存在 |
| 关联指标异常 | 0.20 | 多指标联合异常，增强诊断信心 |
| 变更时间关联 | 0.15 | 异常前有变更，提供因果线索 |
| 知识库匹配 | 0.15 | 历史故障模式匹配，提供经验依据 |
| 依赖资源异常 | 0.10 | 下游/上游异常，提供传播路径 |
| 日志证据 | 0.10 | LTS日志中的错误/异常，提供直接证据 |
```

### 15.2 置信度等级与动作

| 置信度 | 等级 | 诊断行为 | 报告措辞 |
|--------|------|---------|---------|
| 0.8-1.0 | 高 | 直接给出根因+修复建议 | "根因确认: ..." |
| 0.5-0.8 | 中 | 给出最可能根因+备选假设+进一步排查步骤 | "最可能根因: ... (置信度: XX%), 建议进一步验证: ..." |
| 0.2-0.5 | 低 | 列出多个假设+各自排查步骤 | "疑似根因 (需验证): 1)... 2)... 3)..." |
| 0-0.2 | 极低 | 仅描述异常现象+建议人工排查 | "异常已确认但根因未明, 建议人工排查以下方向: ..." |

### 15.3 不确定性声明 (Uncertainty Declaration)

所有诊断报告 MUST 包含不确定性声明:
- 明确标注哪些结论是确定的、哪些是推测的
- 推测性结论标注置信度百分比
- 列出未覆盖的排查方向
- 如果关键数据缺失(如LTS/AOM不可用), 必须声明数据盲区

---

## 16. Compliance Checklists

### 16.1 P0 — Must Pass

- [ ] **Multi-Metric Inspection:** ≥ 4 anomaly patterns with CLI + SDK implementation
- [ ] **Cross-Skill Decision Tree:** Verify → Check → Correlate → Diagnose → Report
- [ ] **Delegation Matrix:** Complete alarm-to-Skill mapping in `integration.md`
- [ ] **Proactive Inspection:** Discovery → Collection → Detection → Diagnosis → Report
- [ ] **Alarm Storm Handling:** Detection criteria + aggregation/suppression workflow
- [ ] **Diagnosis Schema:** Unified report format per Section 9
- [ ] **AOM/LTS Integration:** Delegation triggers for applicable skills
- [ ] **Knowledge Base:** `references/knowledge-base.md` with ≥ 3 fault patterns
- [ ] **Multi-Round Reflection:** 3-round review process defined in troubleshooting
- [ ] **SLO/SLI Definition:** At least 1 SLO with SLI, Error Budget, and burn rate alerting

### 16.2 P1 — Should Pass

- [ ] **Cascade Patterns:** ≥ 2 cross-product cascade fault patterns
- [ ] **Observability Trinity:** Metrics→Logs→Traces linkage rules in `references/observability.md`
- [ ] **Prompt Handbook:** `references/prompts.md` with ≥ 20 categorized prompts
- [ ] **Trend Detection:** Slope, acceleration, sudden-change algorithms implemented
- [ ] **Diagnosis Confidence:** Confidence score for each root cause judgment
- [ ] **Change Correlation:** CTS-based change event correlation with anomaly timeline
- [ ] **Capacity Forecasting:** 30-day capacity prediction with exhaustion date
- [ ] **Chaos Engineering:** At least 1 fault injection experiment design documented
- [ ] **Resilience Score:** Product-specific resilience scoring model defined

---

*This AIOps specification is mandatory. All monitoring, alerting, and diagnostic skills MUST pass compliance checklists.*
