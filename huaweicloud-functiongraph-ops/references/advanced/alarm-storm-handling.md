# Alarm Storm Handling — FunctionGraph

> **Purpose**: 定义 FunctionGraph 函数批量异常时的告警风暴识别、聚合、抑制与响应流程,避免告警刷屏淹没真实故障,并支持跨 skill 快速联动。
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Alarm Storm Detection

当以下真实异常信号在多个函数上同时出现时,判定为潜在告警风暴,需进入聚合/抑制流程。

### 1.1 单函数异常信号阈值

| 信号 | 指标/字段 | 严重阈值 | 临界阈值 |
|---|---|---|---|
| 失败率 | `fail_count` | > 1% | > 5%(近 5min) |
| 执行超时 | `max_duration` | > 90% 超时阈值 | 频繁触发超时 |
| 触发限流 | `reject_count` | > 0 | 持续增长 |
| 并发逼近上限 | `concurrent_executions` | > 80% 并发上限 | 触及上限被打散 |
| 内存溢出 | `duration` + `fail_count` 同时抬升 | OOM 型失败 | — |
| 冷启动冲击 | `cold_starts / invocations` | > 0.3 | — |
| 时延突增 | P99 时延 | > 3× 基线 | — |
| 调用量骤降 | `invocations` | 较基线下降 > 50% | — |

### 1.2 风暴判定

满足以下任一条件即判定为 **Alarm Storm**:

- **集群性**:≥ 3 个函数在 **5 分钟内** 同时进入 `Critical` 状态。
- **共性根因特征**:集群性 Critical 伴随共享依赖劣化(共享 OBS 桶、APIG 网关、SMN 主题、Timer 触发器、CTS 审计流)或区域级资源劣化。
- **级联特征**:函数间通过事件流(如 OBS→函数、APIG→函数)形成调用链,上游劣化扩散至下游多个函数。

---

## 2. Aggregation Rules

风暴期间,分散告警必须聚合,避免 per-function 逐条 page。

| 规则 | 行为 |
|---|---|
| 合并范围 | 同一触发源(见 §1.2 触发源)引发的 ≥3 函数 Critical,合并为 **单条 consolidated page** |
| 触发源分组 | 按 `OBS` / `APIG` / `SMN` / `Timer` / `CTS` 分组关联,每组生成一条聚合事件 |
| CES 事件标签 | 聚合事件打标签 `aiops-cluster:functiongraph`,便于 CES 事件检索与去重 |
| 快照 | 聚合时自动快照当前受影响的 function 列表、触发源、指标快照,附于 consolidated page |
| 去重窗口 | 同一触发源分组的聚合事件在 5min 内不重复生成;新函数进入同一分组则更新快照而非新增 page |

---

## 3. Suppression Rules

| 规则 | 行为 |
|---|---|
| 暂停非必要修复 | 风暴确认(集群性 Critical)后,暂停非紧急的单函数修复操作,优先定位共性根因 |
| 关联抑制 | 同一触发源分组内的下游函数告警被上游告警抑制,仅保留上游根因告警 + 聚合事件 |
| 噪声抑制 | `reject_count > 0` 这类限流告警,若已被集群并发上限告警覆盖,则降级为 INFO 不 page |
| 冷启动冲击 | `cold_starts/invocations > 0.3` 单独出现且无失败率抬升时,抑制为性能提示,不进入风暴判定 |
| 抑制期限 | 所有抑制在根因恢复或风暴解除后自动失效;最长抑制窗口 30min,超时强制重新评估 |

---

## 4. Response Procedures

| 步骤 | 动作 | 说明 |
|---|---|---|
| 1 | 判定风暴 | 按 §1.2 确认 ≥3 函数 5min 内 Critical,锁定触发源分组 |
| 2 | 暂停非必要修复 | 冻结单函数优化/部署,避免引入额外变量 |
| 3 | 快照 | 抓取受影响函数列表、触发源、指标快照 |
| 4 | 合并 page | 按触发源分组生成单条 consolidated page,打 `aiops-cluster:functiongraph` |
| 5 | 定位共性根因 | 检查共享依赖(OBS/APIG/SMN/Timer/CTS)与区域资源(并发上限、配额) |
| 6 | 联动处理 | 见 §5 Delegation Matrix,触发对应 skill 处理上游根因 |
| 7 | 解除抑制 | 根因恢复后取消抑制,复位告警策略,复盘风暴 |

---

## 5. Delegation Matrix

| 触发源 / 根因 | 联动 Skill | 处理内容 |
|---|---|---|
| 共享 OBS 桶劣化 | `huaweicloud-obs-ops` | 检查桶可用性、带宽、限流 |
| APIG 网关异常 | `huaweicloud-elb-ops`(APIG 链路) | 检查网关健康、后端连接池 |
| SMN 主题堆积/失败 | `huaweicloud-dms-ops` | 检查消息投递、主题订阅状态 |
| Timer 触发器失效 | `huaweicloud-cts-ops` | 检查定时器调用与审计记录 |
| 区域资源劣化 / 并发上限 | `huaweicloud-billing-ops` | 检查配额、并发上限与费用 |
| 权限 / AK 失效 | `huaweicloud-iam-ops` | 检查委托、AK/SK、策略 |
| 数据库依赖异常 | `huaweicloud-rds-ops` | 检查连接、慢查询、实例状态 |
