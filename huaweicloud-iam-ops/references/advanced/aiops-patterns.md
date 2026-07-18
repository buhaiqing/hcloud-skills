# AIOps Patterns — IAM

> **Purpose**: Anomaly patterns and root cause analysis for Huawei Cloud IAM (identity, access, policy, credential).
> **Version**: 1.0.0
> **Last Updated**: 2026-07-18

---

## 1. Anomaly Patterns

| # | 模式 | 信号 | 阈值 | 严重度 |
|---|------|------|------|--------|
| 1 | 权限拒绝激增 | AccessDenied 事件数 | 突增 / 批量账号出现 | P1 |
| 2 | AK/SK 泄露或失效 | 异常 IP 调用 / 密钥报错 | 新地域登录 / 密钥无效 | P1 |
| 3 | 策略配置错误 | 非预期授权 / 授权丢失 | 权限漂移 | P2 |
| 4 | MFA 失效 | MFA 校验失败率 | > 基线 | P2 |
| 5 | 配额耗尽 | 用户/策略/用户组配额 | 创建失败 | P3 |
| 6 | 跨账号委派异常 | 委托/代理异常拒绝 | 委派链断裂 | P2 |

### 关联模式

- **IAM-P001**（拒绝+泄露）：AccessDenied 突增叠加异常 IP → 疑似密钥泄露，优先冻结
- **IAM-P002**（策略漂移+配额）：授权丢失伴随配额耗尽 → 批量操作失败

---

## 2. Alarm Storm Handling

告警风暴处理待补充（iam 当前无独立 `alarm-storm-handling.md`）。通用准则可参照：>10 告警/5min 进风暴、同账号/同域 >3 告警聚合、因果链 2min 折叠。

---

## 3. Root Cause Analysis

1. **权限拒绝激增**：聚合 AccessDenied 按 principal + action + resource；定位是策略误删还是密钥轮换。
2. **AK/SK 泄露或失效**：拉取 CTS 登录/调用日志，比对历史可信 IP；确认后立即禁用密钥并轮换。
3. **策略配置错误**：对比基线策略版本（git/备份），回滚非预期变更；联动 `huaweicloud-cts-ops` 溯源操作人。
4. **MFA 失效**：检查虚拟 MFA 设备状态与时钟偏移；为高风险账号强制重绑。
5. **配额耗尽**：核对用户/用户组/策略配额上限，清理僵尸实体或提工单扩容。
6. **跨账号委派异常**：校验委托策略（agency）与信任关系；确认被委托方权限未被回收。

### 联动矩阵

| 场景 | 委托 | 触发 |
|------|------|------|
| 操作审计溯源 | `huaweicloud-cts-ops` | 需定位操作人/时间 |
| 计费/配额异常 | `huaweicloud-billing-ops` | 配额与费用关联 |
| 各产品权限问题 | 对应 `huaweicloud-*-ops` | 产品侧 AccessDenied |
| 安全事件 | `huaweicloud-hss-ops` | 疑似泄露/入侵 |
