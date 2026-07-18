# WAF Fault Knowledge Base

> 本知识库汇总 WAF 常见故障模式、根因定位速查与跨 skill 联动。WAF 无独立 `monitoring.md`,信号提取自 `SKILL.md`、`common-faults.md` 与 AIOps 模式。命令示例以 `hcloud` CLI 为主,不确定参数以 `{{user.*}}` / `{{env.*}}` 占位。

---

## 常见故障模式

| 故障ID | 症状 | 根因 | 排查步骤 | 恢复动作 |
|---|---|---|---|---|
| WAF-01 | 攻击突增 | 请求变化率 `> 100%` | 查攻击类型分布与来源 | 启用/强化防护规则;封禁 |
| WAF-02 | 规则失效/绕过 | 命中率 `3σ` 偏离或连续 3 天 0 命中 | 查规则命中趋势与 payload | 更新规则;排查绕过手法 |
| WAF-03 | 证书到期 | 证书剩余 `30/7` 天 | 查域名证书有效期 | 更新 SCM 证书并重载 |
| WAF-04 | CC 攻击 | `attacks=["cc"]` 超 500 | 查 CC 频率与源 IP | 启用 CC 限速;封禁 |
| WAF-05 | 源 IP 跨域名攻击 | `group_by .sip` 命中多域名 | 按源 IP 聚合分析 | 封禁源 IP;全域名生效 |
| WAF-06 | 域名健康分低 | 防护域名异常/回源失败 | 查域名状态与回源 | 修复回源;检查源站 |
| WAF-07 | 误拦截 | 正常请求被拦 | 查拦截日志与规则 | 加白名单;调规则精度 |
| WAF-08 | 回源异常 | 源站不可达 | 查回源 IP 与源站健康 | 切换源站;检查安全组 |

---

## 根因定位速查

- **攻击突增 > 100%** → 先看是真实攻击还是业务活动,真实攻击联动 Anti-DDoS。
- **规则 0 命中连续 3 天** → 高度疑似规则绕过或业务变更,必须复核规则有效性。
- **证书 30/7 天告警** → 自动化续期优先,人工介入仅作兜底。
- **跨域名同 SIP** → 说明同一攻击者横向探测,封禁应覆盖全域名而非单域名。

---

## 跨 skill 联动

| 故障场景 | 联动 Skill | 目的 |
|---|---|---|
| 源站为 ECS | `huaweicloud-ecs-ops` | 检查源站实例 |
| 主机入侵排查 | `huaweicloud-hss-ops` | 源站主机安全 |
| 回源负载 | `huaweicloud-elb-ops` | 检查后端负载 |
| 费用异常 | `huaweicloud-billing-ops` | 防护计费 |
| 权限/策略 | `huaweicloud-iam-ops` | 检查 WAF 调用权限 |
| 网络/子网 | `huaweicloud-vpc-ops` | 检查回源网络 |
| 证书管理 | `huaweicloud-scm-ops`(若启用) | 证书续期 |
| 指标监控 | `huaweicloud-ces-ops` | 查看 WAF 指标 |
| DDoS 清洗 | `huaweicloud-antiddos-ops`(若启用) | 大流量攻击清洗 |
