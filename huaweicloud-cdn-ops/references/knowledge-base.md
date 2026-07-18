# CDN Fault Knowledge Base

> 本知识库汇总 CDN 常见故障模式、根因定位速查与跨 skill 联动,基于 `monitoring.md` 中定义的真实指标信号。命令示例以 `hcloud` CLI 为主,不确定的参数以 `{{user.*}}` / `{{env.*}}` 占位。

---

## 常见故障模式

| 故障ID | 症状 | 根因 | 排查步骤 | 恢复动作 |
|---|---|---|---|---|
| CDN-01 | 回源 5xx 比例升高 | 源站返回 5xx | 查 `origin_http_code_5xx_rate > 10%`;检查源站健康检查 | 切换源站/回源策略;联系源站运维 |
| CDN-02 | 命中率下降 | 缓存未命中或缓存被频繁淘汰 | 查 `flux_hit_rate < 70%`;检查缓存规则与 TTL | 调整缓存策略;预热热点资源 |
| CDN-03 | 带宽超阈 | 突发流量或攻击 | 查带宽 `> 100Gbps`;结合 DDoS 信号判断 | 限速/封禁;扩容或升级套餐 |
| CDN-04 | 刷新风暴 | 批量 purge 触发回源压力 | 查 `refresh_cache > 100/h` | 限制刷新频率;合并刷新任务 |
| CDN-05 | DDoS 攻击 | 异常出带宽 | 查 `outgoing_bandwidth` p99 `> 10×` p50 | 启用 Anti-DDoS;封禁异常 IP |
| CDN-06 | 源站不可达 | 源站宕机/网络中断 | 测源站连通性;检查回源 IP 白名单 | 切换源站;检查 EIP/WAF 策略 |
| CDN-07 | 证书过期 | HTTPS 握手失败 | 查域名证书有效期 | 更新 SCM 证书并重载 |
| CDN-08 | 区域劣化 | 单节点异常 | 按区域对比命中率与 5xx | 切流至健康节点;提工单 |

---

## 根因定位速查

- **回源 5xx** → 源站侧问题,优先排查源站应用健康,而非 CDN 节点。
- **命中率骤降** → 多为缓存规则变更或大规模 purge,核对近期配置变更。
- **带宽尖峰 + p99≫p50** → 高度疑似 DDoS,联动 `huaweicloud-antiddos-ops` 与 WAF。
- **源站不可达** → 检查 EIP/WAF 回源白名单与源站安全组。

---

## 跨 skill 联动

| 故障场景 | 联动 Skill | 目的 |
|---|---|---|
| 源站不可达 / 回源异常 | `huaweicloud-eip-ops` / `huaweicloud-waf-ops` | 排查回源链路与白名单 |
| 源站为 OBS | `huaweicloud-obs-ops` | 检查桶可用性与带宽 |
| DDoS / 带宽超阈 | `huaweicloud-antiddos-ops`(若启用) | 攻击清洗 |
| 费用突增 | `huaweicloud-billing-ops` | 带宽计费与预算告警 |
| 指标监控 | `huaweicloud-ces-ops` | 配置/查看 CDN 监控指标 |
| 源站为 ECS | `huaweicloud-ecs-ops` | 检查源站实例状态 |
