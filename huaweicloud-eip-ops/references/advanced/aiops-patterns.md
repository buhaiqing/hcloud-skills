# AIOps Patterns — EIP

> **Purpose**: EIP 弹性公网 IP 的 AIOps 异常检测模式，基于真实 CES 指标与 CTS 事件。
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `bandwidth_saturation_warn` | `outgoing_bandwidth / bandwidth_size` ≥ 0.8 | Warning | 观察流量趋势，准备升配 |
| `bandwidth_saturation_crit` | `outgoing_bandwidth / bandwidth_size` ≥ 0.95 | Critical | 立即扩容带宽（`hcloud eip update` 升配） |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `billing_shock` | 日 `outgoing_bytes` > 3 × 7 天中位数 | Major | 联动 billing-ops 核查费用，确认突发 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `status_abnormal_5m` | `eip_status` ≠ ACTIVE 持续 5 min | Warning | 查绑定状态与配额，尝试恢复 |
| `status_abnormal_15m` | `eip_status` ≠ ACTIVE 持续 15 min | Major | 检查底层资源，必要时重绑 |
| `association_flip` | `eip_association_status` 翻转（绑定↔解绑） | Warning | 确认是否预期变更 |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `same_source_flap` | 同一 EIP `eip_status` 异常 + `eip_association_status` 翻转同源 | Major | 定位根因（实例/网卡），修复后稳定 |
| `region_traffic_peak` | 多 EIP 同时带宽打满 | Major | 判定区域流量高峰，整体上浮配额 |

---

## 2. Alarm Storm Handling

仅交叉引用，避免重复（TE-6）：详见 `references/advanced/alarm-storm-handling.md`。

---

## 3. Root Cause Analysis

1. **带宽打满** → 查 CES `outgoing_bandwidth` 时序 → 区分突发与持续 → 升配或切流量计费。
2. **状态异常** → 查 CTS association 事件 → 确认实例/网卡是否异常。
3. **费用突增** → 关联 `outgoing_bytes` 与 `bandwidth_size` → 核查流量计费样本分布。
4. **区域高峰** → 多 EIP 聚合分析 → 判定业务高峰，统一扩容。
