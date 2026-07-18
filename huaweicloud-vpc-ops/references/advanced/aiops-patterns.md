# AIOps Patterns — VPC

> **Purpose**: VPC 网络资源的 AIOps 异常检测模式，基于真实 CES 指标与 CTS 事件。
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `bandwidth_saturation` | `bandwidth_out` > 90% 已购带宽 | Major | 扩容共享带宽或升配带宽包 |
| `nat_snat_capacity` | NAT 网关 SNAT 连接数接近网关容量上限 | Major | 扩容 NAT 网关规格或增建 SNAT 规则 |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `ddos_shape` | `packet_out` > 100k pps 持续上升 | Major | 联动 HSS/DDoS 清洗，隔离可疑流量 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `eip_unbind_event` | CTS 出现 EIP 绑定/解绑事件 | Warning | 确认是否预期变更，核查暴露面 |
| `peering_state_change` | 对等连接状态 PENDING→ACTIVE→REJECTED | Major | 检查对端账号审批，重建连接 |
| `secgroup_0_0_0_0_change` | 安全组规则出现 0.0.0.0/0 变更 | Major | 确认是否预期，收紧入向规则 |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `exposure_surface_change` | 解绑 EIP + 安全组 0.0.0.0/0 变更同窗发生 | Critical | 立即核查暴露面，回滚非预期变更 |
| `suspected_attack` | `bandwidth_out` 高 + `packet_out` 高双指标同窗 | Major | 按 DDoS 处置流程联动清洗 |

---

## 2. Alarm Storm Handling

仅交叉引用，避免重复（TE-6）：详见 `references/advanced/alarm-storm-handling.md`。

---

## 3. Root Cause Analysis

1. **带宽打满** → 查 CES `bandwidth_out` 时序 → 区分正常业务高峰与异常流量 → 升配或清洗。
2. **对等连接异常** → 查 CTS peering 事件 → 确认对端审批状态（REJECTED 需对方处理）。
3. **暴露面突变** → 关联 EIP 解绑与 0.0.0.0/0 安全组变更时间窗 → 定位变更来源。
4. **NAT 连接耗尽** → 查 SNAT 连接数指标 → 扩容网关或优化 SNAT 规则。
