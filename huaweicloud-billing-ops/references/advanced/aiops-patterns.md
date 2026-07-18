# AIOps Patterns — Billing

> **Purpose**: 账单/费用专属异常检测模式，基于账户余额、预算消耗与周期性成本信号（真实字段）。
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `balance_low` | 账户余额/信用 < ¥500 | Warning | 提醒充值，避免停服 |
| `balance_critical` | 账户余额/信用 < ¥100 或逾期金额 > ¥0 | Critical | 立即充值，否则资源停服 |
| `budget_overrun` | 预算消耗率 > 80% | Warning | 审视高成本资源，设配额 |
| `budget_forecast_exceed` | 预测月末消耗 > 预算 100% | Critical | 冻结非必要资源，告警责任人 |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `daily_burn_anomaly` | 当日 burn-rate 偏离近 7 日均值的 2σ | Warning | 定位当日新增/突增资源 |
| `weekly_cost_anomaly` | 本周成本 > 上周及前 3 周均值 + 2σ | Major | 排查周期性业务或异常计费 |
| `monthly_cost_scan` | 月度全量扫描：单 cost center 占比异常跳变 | Major | 下钻 cost center 资源明细 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `alert_storm_root` | `create-budget` 阈值设为 0% 触发一次性告警风暴 | Critical | 文档点名根因：阈值勿设 0%，改用 >80% 梯度 |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `cost_center_aggregation` | 按 cost center 聚合异常而非逐条资源轰炸 | Warning | 单 cost center 汇总一条通知，降噪 |

---

## 2. Alarm Storm Handling

仅交叉引用，不重复内容：`详见 references/advanced/alarm-storm-handling.md`

---

## 3. Root Cause Analysis

1. **余额不足** → 查余额/信用与逾期金额 → 区分 Warning/Critical 阈值 → 触发充值提醒或停服预警。
2. **预算超支** → 比消耗率与预测曲线 → 定位超额 cost center → 配额限制或资源缩容。
3. **周期性异常** → 日/周/月三粒度对比均值与 2σ → 区分业务增长与计费错误。
4. **告警风暴** → 查 `create-budget` 阈值配置 → 根因：阈值设 0% 立即全量触发 → 改为梯度阈值。
5. **噪声聚合** → 按 cost center 归并 → 单中心单条通知，避免资源级轰炸。
