# 智能运维（AIOps）

> **JSON paths**：内联 `jq` 调用统一引用以下路径。
> `.total` — 事件总数（ListEvents 顶层） `.items[]` — 事件数组
> `.items[].rule|.sip|.attack|.host` — 规则ID / 源IP / 攻击类型 / 防护域名
> `.server[0].front_protocol` — 域名前端协议 `.certificateid` — 绑定证书ID
> `.policyid` — 防护策略ID `.name|.expire_time` — 证书名 / 到期时间

## 异常检测模式

### 模式 1：攻击流量突增检测
- **指标**：`ListEvents` 事件总数在当前窗口 vs 上周同窗口的**变化率**。
- **阈值**：变化率 **>100%** → 告警，建议收紧 CC 规则或提高防护等级。
- **检测**：`hcloud WAF ListEvents --project_id=… --recent=true | jq '.total'` 对比前两周窗口（`--from/--to` 取 `date +%s*1000 -1209600000 / -604800000`）。

### 模式 2：规则命中率异常（过低或过高）
- **指标**：按规则分组命中次数 vs 基线偏差 **>3σ**（原文按均值 3 倍近似）。
- **动作**：
  - 连续 **3 天命中为 0** 的规则 → 疑似失效/被绕过。
  - 命中超过 `均值×3` 的规则 → 标记为异常（可能误报或攻击集中）。
- **检测**：`hcloud WAF ListEvents --recent=true | jq '.items | group_by(.rule) | map({rule:.[0].rule, count:length}) | sort_by(.count) | reverse'`。

### 模式 3：证书到期预测
- **指标**：证书剩余有效期天数。
- **阈值**：到期前 **30 天** 预警；前 **7 天** 紧急告警。
- **检测**：`hcloud WAF ListCertificates | jq -r '.items[] | "\(.name): \(.expire_time)"'`，对 `expire_time` 用 `strptime("%Y-%m-%d")|mktime` 与 `now` 比对。

### 模式 4：防护域名健康度评分（满分 100）
| 维度 | 扣分项 | 扣分 |
|------|--------|------|
| HTTPS | `front_protocol != "HTTPS"` | -10 |
| 证书 | `certificateid` 为空/null | -15 |
| CC 规则 | `ListCcRules --policy_id=… | jq '.items|length' == 0` | -10 |
| 黑白名单 | `ListWhiteBlackIpRules --policy_id=… | length == 0` | -10 |
| 攻击趋势 | `ListEvents --hosts=[...] --recent | .total > 10000` | -15 |
- **评级**：≥80 Good｜≥50 Needs Improvement｜<50 Critical。
- **查询**：`hcloud WAF ListHost | jq '.items[] | select(.hostname==$d)'`。

## 自动修复模式

### 自动缓解 CC 攻击
- **触发**：`ListEvents --attacks=["cc"] --recent | jq '.total > 500'`。
- **动作**：临时收紧 CC 规则——`ListCcRules --policy_id=… --page=1 --pagesize=50` 取 `.items[].limit_num`，`UpdateCcRule --rule_id=… --body='{"limit_num":新值}'`（阈值降 50%，`NEW_LIMIT=CURRENT/2`）。

### 自动清理僵尸规则
- **触发**：自定义规则创建超过 **90 天**（`--timestamp` 推算 `AGE_DAYS>90`）且 `ListEvents --recent` 中 `select(.rule==$rid)` 命中数为 0。
- **动作**：`ListCustomRules --policy_id=…` 扫描；确认零命中后可 `DeleteCustomRule --rule_id=…`（默认注释，需显式开启）。

## 异常关联分析

### 攻击模式识别（同 IP 跨域名）
- **检测**：`hcloud WAF ListEvents --recent | jq '.items | group_by(.sip) | map({ip:.[0].sip, attacks:[.[].attack]|unique, hosts:[.[].host]|unique, count:length}) | sort_by(.count) | reverse | .[0:10]'`。
- **输出**：按攻击数排序的 Top 10 源 IP——覆盖域数列、攻击类型列表。

### AIOps 基线数据收集（每日）
- **指标快照**：域名数 `ListHost | .items|length`、策略数 `ListPolicy | .items|length`、7天事件数 `ListEvents --recent | .total`、证书数 `ListCertificates | .items|length`。
- 每日落盘一份 JSON 基线报告供趋势对比。
