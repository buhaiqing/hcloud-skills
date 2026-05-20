# 安全门

## 高危操作清单

| 操作 | 风险等级 | 影响范围 | 审批要求 |
|------|---------|---------|---------|
| 隔离并终止进程 | 🔴 严重 | 进程被杀可能导致业务中断 | 变更审批 + 双人复核 |
| 恢复隔离文件 | 🟡 高 | 可能将恶意文件恢复到主机 | 变更审批 |
| 解封已封锁 IP | 🟡 高 | 可能放行攻击者 | 变更审批 |
| 切换防护版本 | 🟡 高 | 降级可能降低安全保护 | 变更审批 |
| 处理告警事件 | 🟢 中 | 标记已处理可能掩盖真实攻击 | 记录即可 |
| 创建漏洞扫描 | 🟢 中 | 可能产生额外负载 | 监控即可 |
| 查询类操作 | ⚪ 低 | 无影响 | 无 |

## 审批流程

### 严重级别操作（🔴）

```
┌─────────────────────────────────────────────────┐
│ Step 1: 确认事件类型（确认是恶意而非误报）       │
│   - 检查告警事件详情                             │
│   - 确认文件路径和进程                          │
├─────────────────────────────────────────────────┤
│ Step 2: 评估业务影响                             │
│   - 该进程是否属于关键业务                      │
│   - 确认是否有替代方案                          │
├─────────────────────────────────────────────────┤
│ Step 3: 双人复核                                │
│   - Person A 确认操作参数                       │
│   - Person B 核准执行                          │
├─────────────────────────────────────────────────┤
│ Step 4: 执行隔离                                │
├─────────────────────────────────────────────────┤
│ Step 5: 验证业务恢复                            │
│   - 确认业务未受影响                            │
│   - 记录事件处理结果                            │
└─────────────────────────────────────────────────┘
```

### 高级别操作（🟡）

```
┌─────────────────────────────────────────┐
│ 1. 确认操作目的                          │
│ 2. Lead 审批                            │
│ 3. 执行 + 确认                          │
└─────────────────────────────────────────┘
```

## 操作前安全检查

### 隔离进程前检查

```bash
#!/bin/bash
# 隔离进程前安全检查

EVENT_ID=$1

echo "=== Pre-Isolation Safety Check ==="

# 1. 查看告警事件详情
EVENT_DETAIL=$(hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --event_ids="[\"$EVENT_ID\"]")

FILE_PATH=$(echo "$EVENT_DETAIL" | jq -r '.data_list[0].file_path // "unknown"')
HOST_NAME=$(echo "$EVENT_DETAIL" | jq -r '.data_list[0].host_name // "unknown"')
PROCESS_NAME=$(echo "$EVENT_DETAIL" | jq -r '.data_list[0].process_name // "unknown"')
echo "Event: $EVENT_ID"
echo "Host: $HOST_NAME"
echo "File: $FILE_PATH"
echo "Process: $PROCESS_NAME"

# 2. 检查是否为关键系统进程
CRITICAL_PROCESSES=("sshd" "nginx" "httpd" "mysqld" "java" "docker")
for proc in "${CRITICAL_PROCESSES[@]}"; do
    if [[ "$PROCESS_NAME" == *"$proc"* ]]; then
        echo "[WARN] Process '$PROCESS_NAME' may be a critical service!"
        echo "[REQUIRED] Confirm isolation with service owner"
    fi
done

# 3. 确认操作
echo "[CONFIRM] Isolate and kill '$PROCESS_NAME' on '$HOST_NAME'? (yes/no)"
```

### 解封 IP 前检查

```bash
#!/bin/bash
# 解封 IP 前安全检查

echo "=== Pre-Unblock Safety Check ==="

# 查看当前被封锁的 IP 列表
BLOCKED_IPS=$(hcloud HSS ListBlockedIp --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID")
echo "Currently blocked IPs:"
echo "$BLOCKED_IPS" | jq -r '.data_list[] // [] | "  - \(.block_ip) (blocked at: \(.block_time // "unknown"))"'

echo ""
echo "[CHECK] Has the security incident been resolved?"
echo "[REQUIRED] Verify the source IP is no longer a threat"
```

## 安全门绕过机制

紧急情况下（正在进行的攻击），可简化审批流程：

```bash
# 紧急绕过条件：
# 1. 正在进行的攻击，不立即行动将导致业务受损
# 2. 值班经理授权
# 3. 事后 24h 内补交变更记录

# 紧急隔离（仅需确认事件 ID 正确）
hcloud HSS OperateEvent --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"operate_type\":\"isolate_and_kill\",\"event_id_list\":[\"$EVENT_ID\"],\"operate_detail_list\":[...]}"

# 紧急封禁 IP
hcloud HSS ChangeBlockedIp --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"data_list\":[{\"block_ip\":\"$ATTACKER_IP\",\"host_id\":\"\"}],\"operate_type\":\"block\"}"
```

## 变更记录模板

```json
{
    "timestamp": "2026-01-15T10:30:00+08:00",
    "operator": "{{env.OPERATOR}}",
    "action": "OperateEvent",
    "resource_id": "event-xxxxx",
    "resource_name": "malware-alert-on-host-prod-01",
    "operate_type": "isolate_and_kill",
    "reason": "Confirmed malware detected in /tmp directory",
    "approval": "JIRA-12345"
}
```
