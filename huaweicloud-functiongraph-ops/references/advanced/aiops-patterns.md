# AIOps Patterns — FunctionGraph

> **Purpose**: Anomaly patterns and root cause analysis for FunctionGraph serverless functions.
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

| # | 模式 | 信号 | 阈值 | 严重度 |
|---|------|------|------|--------|
| 1 | 失败率突增 | fail_count / invocations | > 1% 严重 / > 5% (5min) 临界 | P1 |
| 2 | 执行超时 | max_duration | > 90% 阈值 | P2 |
| 3 | 流控拒绝 | reject_count | > 0 | P2 |
| 4 | 并发饱和 | concurrent_executions | > 80% 上限 | P2 |
| 5 | OOM 崩溃 | duration 异常 + fail_count 上升 | 内存超限 | P1 |
| 6 | 冷启动劣化 | cold_starts / invocations | > 0.3 | P3 |
| 7 | 时延突增 | P99 延迟 | > 3× 基线 | P2 |
| 8 | 调用骤降 | invocations 环比 | > 50% 下降 | P1 |

### 关联模式

- **FG-P001**（失败+OOM）：duration 拉长同时 fail_count 上升 → 内存规格不足
- **FG-P002**（超时+流控）：max_duration 高 + reject_count > 0 → 下游阻塞引发级联
- **FG-P003**（冷启动+时延）：cold_starts 占比高推高 P99 → 预留实例不足

---

## 2. Alarm Storm Handling

告警风暴处理详见 `references/advanced/alarm-storm-handling.md`（本 skill 暂不单独维护告警风暴文档，交叉引用通用准则：>10 告警/5min 进风暴、同函数 >3 告警聚合、因果链 2min 折叠）。

---

## 3. Root Cause Analysis

1. **失败率突增**：先查 `fail_count` 错误类型分布（超时 / 代码异常 / 依赖失败）；结合下游调用指标定位是否为级联。
2. **OOM**：确认内存规格与峰值 `duration`；建议上调内存或优化大对象生命周期。
3. **流控拒绝**：检查 `concurrent_executions` 是否触顶；评估预留并发或拆分函数。
4. **冷启动劣化**：`cold_starts/invocations > 0.3` → 配置预留实例（provisioned concurrency）。
5. **调用骤降**：排除上游触发器故障（APIG/DMS/OBS 事件）；联动 `huaweicloud-iam-ops` 排查权限失效。

### 联动矩阵

| 场景 | 委托 | 触发 |
|------|------|------|
| 下游依赖异常 | `huaweicloud-*-ops`（对应产品） | 依赖错误率上升 |
| 触发器故障 | `huaweicloud-apig-ops` / `huaweicloud-dms-ops` | 调用源中断 |
| 权限失效 | `huaweicloud-iam-ops` | AccessDenied |
| 指标缺失 | `huaweicloud-ces-ops` / AOM | 无数据 > 5 min |
