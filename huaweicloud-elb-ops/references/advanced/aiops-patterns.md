# AIOps Patterns — ELB

> **Purpose**: ELB 弹性负载均衡的 AIOps 异常检测模式，基于真实 CES 监控指标。
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

### 1.1 Resource Pressure Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `backend_degrade` | `m9_unhealthy_host` > 0 持续 | Major | 检查后端健康检查与实例状态 |
| `latency_spike` | `m10` P99 > 3s | Major | 定位慢后端，优化或扩容 |

### 1.2 Trend Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `traffic_surge` | `m1_cps` > 3 × 基线 | Warning | 确认业务高峰，预扩容 |

### 1.3 Sudden Change Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `5xx_rate_warn` | `m7_req_5xx` > 1% | Critical | 查后端错误码，隔离故障节点 |
| `5xx_rate_crit` | `m7_req_5xx` > 5% 持续 5 min | Critical | 紧急回滚或切流，抑制错误扩散 |
| `drop_rate` | `m5_drop_rate` > 0% | Major | 查连接数与带宽上限，扩容 |

### 1.4 Correlation Anomaly Patterns

| Pattern | Detection Logic | Severity | Expected Action |
|---------|---------------|----------|----------------|
| `backend_degradation` | `m9_unhealthy_host` + `m7_req_5xx` + `m8` 时延同源 | Critical | 判定后端整体降级，扩容/重启后端 |
| `connection_storm` | `m5_drop_rate` + `m1_cps` 双高同窗 | Critical | 连接风暴，限流并扩容 ELB/后端 |

---

## 2. Alarm Storm Handling

仅交叉引用，避免重复（TE-6）：详见 `references/advanced/alarm-storm-handling.md`。

---

## 3. Root Cause Analysis

1. **5xx 突增** → 查 `m7_req_5xx` 与 `m9_unhealthy_host` → 定位故障后端实例。
2. **连接丢弃** → 关联 `m5_drop_rate` 与 `m1_cps` → 判定连接风暴，查配额/带宽。
3. **时延升高** → 查 `m10` P99 与 `m8` 时延 → 定位慢后端或链路问题。
4. **后端不健康** → 查健康检查配置与后端实例状态 → 修复或剔除故障节点。
