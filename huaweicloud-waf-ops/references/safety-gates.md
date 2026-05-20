# 安全门

## 高危操作清单

| 操作 | 风险等级 | 影响范围 | 审批要求 |
|------|---------|---------|---------|
| 删除防护策略 | 🔴 严重 | 该策略下所有域名失去保护 | 变更审批 + 双人复核 |
| 删除防护域名 | 🔴 严重 | 域名失去 WAF 保护 | 变更审批 + 双人复核 |
| 删除证书 | 🟡 高 | 绑定该证书的域名 HTTPS 中断 | 变更审批 |
| 修改防护等级 | 🟡 高 | 防护强度变化可能导致误报或漏报 | 变更审批 |
| 删除规则 | 🟢 中 | 特定规则失效 | 记录即可 |
| 添加白名单 | 🟢 中 | 对应流量绕过 WAF 检测 | 记录即可 |
| 查询类操作 | ⚪ 低 | 无影响 | 无 |

## 审批流程

### 严重级别操作（🔴）

```
┌─────────────────────────────────────────────────┐
│ Step 1: 提交变更申请（Jira/工单系统）             │
│   - 变更内容、影响范围、回滚方案                  │
├─────────────────────────────────────────────────┤
│ Step 2: 技术审核（Lead 审批）                    │
│   - 确认变更必要性                              │
│   - 评估影响范围                                │
├─────────────────────────────────────────────────┤
│ Step 3: 双人复核                                │
│   - Person A 执行命令                           │
│   - Person B 确认命令参数                       │
├─────────────────────────────────────────────────┤
│ Step 4: 执行变更                                │
│   - 窗口期内执行                                │
│   - 保存变更前后状态                            │
├─────────────────────────────────────────────────┤
│ Step 5: 变更验证                                │
│   - 确认业务正常                                │
│   - 保留变更记录                                │
└─────────────────────────────────────────────────┘
```

### 高级别操作（🟡）

```
┌─────────────────────────────────────────┐
│ 1. 变更申请（轻量化）                    │
│ 2. 直接 Lead 审批                       │
│ 3. 执行 + 确认                          │
└─────────────────────────────────────────┘
```

## 操作前安全检查

### 删除策略前检查

```bash
#!/bin/bash
# 删除策略前安全检查

POLICY_ID=$1

echo "=== Pre-deletion Safety Check ==="

# 1. 确认策略绑定域名
BOUND_HOSTS=$(hcloud WAF ListHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq --arg pid "$POLICY_ID" '[.items[] | select(.policyid == $pid) | .hostname]')
BOUND_COUNT=$(echo "$BOUND_HOSTS" | jq 'length')

if [ "$BOUND_COUNT" -gt 0 ]; then
    echo "[BLOCKED] Cannot delete policy — bound to $BOUND_COUNT domains:"
    echo "$BOUND_HOSTS" | jq -r '.[] | "  - \(.)"'
    echo "[REQUIRED] 1. Move domains to another policy first"
    echo "[REQUIRED] 2. Update each domain's policyid"
    exit 1
fi

# 2. 列出该策略下的规则数量
CC_RULES=$(hcloud WAF ListCcRules --policy_id="$POLICY_ID" | jq '.items | length')
CUSTOM_RULES=$(hcloud WAF ListCustomRules --policy_id="$POLICY_ID" | jq '.items | length')
echo "[INFO] Policy has $CC_RULES CC rules and $CUSTOM_RULES custom rules — will be deleted together"

# 3. 确认操作
echo "[CONFIRM] Are you sure you want to delete policy $POLICY_ID? (yes/no)"
# 需要人工确认
```

### 删除域名前检查

```bash
#!/bin/bash
# 删除域名前安全检查

HOST_ID=$1

echo "=== Pre-deletion Safety Check ==="

# 1. 获取域名详情
HOST_INFO=$(hcloud WAF ShowHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --host_id="$HOST_ID")
HOSTNAME=$(echo "$HOST_INFO" | jq -r '.hostname')
echo "Domain: $HOSTNAME"

# 2. 确认 DNS 已切走
echo "[CHECK] Has DNS been switched away from WAF CNAME?"
echo "[REQUIRED] Verify DNS A/CNAME record no longer points to WAF"
# 人工确认

# 3. 确认该域名无其他策略中的例外规则
echo "[INFO] After deletion, $HOSTNAME will lose all WAF protection"
echo "[CONFIRM] Are you sure? (yes/no)"
# 需要人工确认
```

### 修改防护等级前检查

```bash
#!/bin/bash
# 防护等级变更检查

POLICY_ID=$1
NEW_LEVEL=$2

echo "=== Policy Level Change Check ==="

CURRENT=$(hcloud WAF ShowPolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="$POLICY_ID" | jq '.level')
echo "[INFO] Current level: $CURRENT → Target level: $NEW_LEVEL"

if [ "$NEW_LEVEL" -lt "$CURRENT" ]; then
    echo "[WARN] Reducing protection level (from $CURRENT to $NEW_LEVEL)"
    echo "  - Loose protection may allow more attacks through"
    echo "  - Consider temporary monitoring after change"
elif [ "$NEW_LEVEL" -gt "$CURRENT" ]; then
    echo "[WARN] Increasing protection level (from $CURRENT to $NEW_LEVEL)"
    echo "  - May cause false positives"
    echo "  - Monitor error rates after change"
fi
```

## 安全门绕过机制

紧急情况下（如正在进行的攻击），可绕过审批流程直接执行高危操作：

```bash
# 紧急绕过需满足：
# 1. 有正在进行的攻击影响业务
# 2. 有值班经理或以上级别授权
# 3. 事后 24 小时内补交变更记录

# 紧急绕过示例：紧急封禁恶意 IP（无需审批）
hcloud WAF CreateWhiteBlackIpRule --policy_id="<policyId>" \
  --body="{\"name\":\"emergency-block-$(date +%Y%m%d%H%M)\",\"addr\":\"x.x.x.x\",\"white\":0}"
```

## 变更记录模板

```json
{
    "timestamp": "2026-01-15T10:30:00+08:00",
    "operator": "{{env.OPERATOR}}",
    "action": "UpdateHost",
    "resource_id": "host-xxxxx",
    "resource_name": "www.example.com",
    "before": {
        "server": [{"address": "192.168.1.1", "port": 8080}]
    },
    "after": {
        "server": [{"address": "10.0.0.1", "port": 8443}]
    },
    "approval": "JIRA-12345",
    "reason": "Backend server migration"
}
```
