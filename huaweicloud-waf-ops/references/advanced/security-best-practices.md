# 安全最佳实践（SecOps）

## 1. 最小权限原则

### IAM 权限配置

```json
{
    "Version": "1.1",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "waf:policy:list",
                "waf:policy:get",
                "waf:host:list",
                "waf:host:get",
                "waf:event:list",
                "waf:event:get"
            ]
        }
    ]
}
```

只读运维权限（日常巡检），避免误操作：
```bash
# 确认当前权限可读
hcloud WAF ListPolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"

# 如需要规则变更，临时申请更高权限
```

### 写操作权限分离

```json
{
    "Version": "1.1",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "waf:*:*"
            ]
        }
    ]
}
```

规则变更、证书更新等写操作由专人执行，遵循变更管理流程。

## 2. 防护策略基线

### 推荐基线配置

```bash
# 创建标准安全基线策略
hcloud WAF CreatePolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"name\":\"security-baseline-prod\",\"level\":2,\"full_detection\":true}"

BASELINE_POLICY_ID=$(hcloud WAF ListPolicy | jq -r '.items[] | select(.name=="security-baseline-prod") | .id')

# 全局基础规则（CC 防护）
hcloud WAF CreateCcRule --policy_id="$BASELINE_POLICY_ID" \
  --body='{"url":"/*","limit_num":1000,"limit_period":60,"lock_time":600,"tag_type":"ip","action":{"category":"block"}}'

# 拦截高风险国家
hcloud WAF CreateGeoIpRule --policy_id="$BASELINE_POLICY_ID" \
  --body='{"name":"block-high-risk-regions","geoip":"RU|KP|IR|SY|AF","white":0}'
```

### 策略分层

| 层级 | 策略名称 | 防护等级 | 适用场景 |
|------|---------|---------|---------|
| L1 | security-baseline-prod | 严格 (3) | 核心生产业务 |
| L2 | security-standard | 中等 (2) | 一般生产业务 |
| L3 | security-loose | 宽松 (1) | 测试/开发环境 |

## 3. 规则变更安全

### 灰度发布规则变更

```bash
# 1. 先在测试策略中创建规则
TEST_POLICY_ID=$(hcloud WAF CreatePolicy --body='{"name":"test-rule-validation"}' | jq -r '.id')
hcloud WAF CreateCustomRule --policy_id="$TEST_POLICY_ID" \
  --body='{"name":"test-rule","conditions":[{"category":"url","logic_operation":"contain","contents":["/test"]}],"action":{"category":"log"},"priority":100}'

# 2. 观察日志模式
hcloud WAF ListEvents --recent="true" --attacks="[\"custom_rule\"]"

# 3. 确认无误后改为 block
hcloud WAF UpdateCustomRule --policy_id="$PROD_POLICY_ID" --rule_id="<ruleId>" \
  --body='{"action":{"category":"block"}}'
```

### 变更回滚流程

```bash
# 变更前：备份现有规则列表
hcloud WAF ListCcRules --policy_id="$POLICY_ID" > /tmp/cc-rules-backup-$(date +%Y%m%d).json

# 变更后如发现问题：删除新规则
hcloud WAF DeleteCcRule --policy_id="$POLICY_ID" --rule_id="<newRuleId>"

# 或重新应用备份规则
cat /tmp/cc-rules-backup-*.json | jq -c '.items[]' | while read rule; do
    hcloud WAF CreateCcRule --policy_id="$POLICY_ID" --body="$rule"
done
```

## 4. 证书安全

### 证书轮换周期

```bash
# 列出所有证书及其到期时间
hcloud WAF ListCertificates --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq -r '.items[] | "\(.name): expires=\(.expire_time)"'

# 到期前 30 天自动提醒
hcloud WAF ListCertificates --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq -r '.items[] | select(.expire_time | strptime("%Y-%m-%d") | mktime < now + 2592000) | .name'
```

### 证书安全基线

- 使用 RSA 2048+ 或 ECC 256+ 位密钥
- 证书有效期不超过 1 年（建议 90 天）
- 私钥妥善保管，不上传到版本控制系统
- 使用华为云 SCM 统一管理证书

## 5. 攻击事件响应

### 实时告警配置

```bash
# 配置定期检查攻击事件脚本（配合监控系统）
# 阈值：5 分钟内超过 100 次攻击触发告警
```

### 攻击事件分析

```bash
# 按攻击类型分组统计
hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --recent="true" | \
  jq '.items | group_by(.attack) | map({attack: .[0].attack, count: length}) | sort_by(.count) | reverse'

# 按来源 IP 统计
hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --recent="true" | \
  jq '.items | group_by(.sip) | map({ip: .[0].sip, count: length}) | sort_by(.count) | reverse | .[0:10]'
```

### 紧急封禁流程

```bash
# 发现恶意 IP 后立即封禁
ATTACKER_IP="x.x.x.x"
hcloud WAF CreateWhiteBlackIpRule --policy_id="$POLICY_ID" \
  --body="{\"name\":\"emergency-block-$(date +%Y%m%d)\",\"addr\":\"$ATTACKER_IP\",\"white\":0}"
```

## 6. 日志与审计

### 操作审计

关键操作需记录审计日志：

| 操作类型 | 审计要求 | 记录方式 |
|---------|---------|---------|
| 策略创建/删除 | 记录操作人和时间 | CTS + CLI 日志 |
| 域名接入/删除 | 记录变更前后配置 | CTS |
| 证书上传/删除 | 记录证书指纹 | CTS |
| 规则创建/删除 | 记录规则详情 | CTS + 备份文件 |

### CTS 配置

确保已启用云审计服务（CTS）并配置 WAF 相关追踪器：
- 追踪 WAF 所有写操作
- 日志保存周期 ≥ 180 天
- 配置转储到 OBS 长期保存

## 7. 定期安全巡检

```bash
#!/bin/bash
# WAF 安全巡检脚本

echo "=== WAF Security Audit ==="

# 1. 检查所有域名是否配置了 HTTPS
hcloud WAF ListHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq '.items[] | select(.server[].front_protocol != "HTTPS") | "⚠️ Domain \(.hostname) not using HTTPS"'

# 2. 检查证书即将过期
hcloud WAF ListCertificates --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq -r '.items[] | select(.expire_time // "9999-12-31" | strptime("%Y-%m-%d") | mktime < now + 2592000) | "⚠️ Cert \(.name) expires on \(.expire_time)"'

# 3. 检查是否有策略未绑定域名
hcloud WAF ListPolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq '.items[] | .id as $pid | .name as $pname | if (.hosts // []) | length == 0 then "⚠️ Policy \($pname) has no hosts" else empty end'

# 4. 检查最近 24 小时攻击趋势
hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --recent="true" | \
  jq '.total as $total | if $total > 1000 then "⚠️ High attack volume: \($total) events (7d)" else "✓ Attack volume normal: \($total) events (7d)" end'

echo "=== Audit Complete ==="
```
