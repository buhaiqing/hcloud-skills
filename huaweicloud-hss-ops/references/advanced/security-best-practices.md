# 安全最佳实践（SecOps）

## 1. IAM 最小权限配置

### 只读运维权限

```json
{
    "Version": "1.1",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "hss:hosts:list",
                "hss:hosts:get",
                "hss:events:list",
                "hss:events:get",
                "hss:vulnerabilities:list",
                "hss:vulnerabilities:get",
                "hss:baselines:list",
                "hss:baselines:get",
                "hss:dashboard:get"
            ]
        }
    ]
}
```

### 完整运维权限（含告警处理）

```json
{
    "Version": "1.1",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "hss:*:*"
            ]
        }
    ]
}
```

## 2. 主机安全基线

### 推荐的基线检查策略

| 检查项 | 检查内容 | 频率 | 建议 |
|--------|---------|------|------|
| 弱口令检查 | 系统账户弱口令 | 每周 | 强制密码复杂度 ≥ 12 位 |
| 系统配置检查 | SSH/Telnet/FTP 等安全配置 | 每周 | 禁用 root 直接 SSH 登录 |
| 风险配置检查 | 文件权限、审核策略 | 每月 | 最小权限原则 |

```bash
# 手动触发基线检查
hcloud HSS CreateBaselineCheckTask --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"host_id_list\":[\"<hostId>\"]}"

# 查看基线检查结果
hcloud HSS ListBaselineCheckResults --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

### Agent 安全

```bash
# 确保所有主机已安装 Agent 且在线
hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq '.data_list[] | select(.agent_status != "online") | "⚠️ \(.host_name) — Agent: \(.agent_status)"'
```

## 3. 入侵检测与事件响应

### 安全告警处理 SOP

```
收到告警 → 确认事件类型 → 评估影响范围 → 采取行动 → 记录处理结果 → 复盘改进
```

```bash
# Step 1: 查看未处理的告警
hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --handle_status="unhandled" | jq '.data_list[] | "\(.event_id): \(.event_type) on \(.host_name)"'

# Step 2: 确认为恶意 → 隔离并终止进程
hcloud HSS OperateEvent --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body='{"operate_type":"isolate_and_kill","event_id_list":["<eventId>"],"operate_detail_list":[{"agent_id":"<agentId>","file_hash":"<hash>","file_path":"<path>","process_pid":1234}]}'

# Step 3: 确认误报 → 加入白名单
hcloud HSS OperateEvent --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body='{"operate_type":"add_to_alarm_whitelist","event_id_list":["<eventId>"],"operate_detail_list":[{"keyword":"<keyword>","hash":"<hash>"}]}'
```

## 4. 漏洞管理策略

### 漏洞修复优先级矩阵

| 严重级别 | Linux 漏洞 | Windows 漏洞 | Web-CMS 漏洞 | 修复 SLA |
|---------|-----------|-------------|-------------|---------|
| Critical | 立即修复 | 立即修复 | 立即修复 | 24h |
| High | 72h 内修复 | 48h 内修复 | 24h 内修复 | 72h |
| Medium | 7天 | 7天 | 3天 | 7天 |
| Low | 30天 | 30天 | 30天 | 30天 |

```bash
# 扫描所有未处理的严重漏洞
hcloud HSS ListVulnerabilities --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --severity="critical" --handle_status="unhandled"

# 创建扫描任务
hcloud HSS CreateScanTask --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"task_name\":\"security-scan-$(date +%Y%m%d)\",\"type\":\"all\"}"
```

### 漏洞修复验证

```bash
# 标记为已修复后，重新扫描确认
hcloud HSS CreateScanTask --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"task_name\":\"verify-fix-$(date +%Y%m%d)\",\"host_id_list\":[\"<hostId>\"],\"type\":\"linux\"}"

# 待扫描完成后检查
hcloud HSS ListVulnerabilities --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --host_id="<hostId>" --handle_status="unhandled"
```

## 5. 网页防篡改

### Web 目录保护配置

```bash
# 为 Web 服务器启用网页防篡改
hcloud HSS CreateWtpProtection --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body="{\"host_id\":\"<hostId>\",\"policy_name\":\"wtp-prod-web\",\"protected_directory\":\"/var/www/html\",\"backup_directory\":\"/var/backup/html\"}"

# 查看保护状态
hcloud HSS ListWtpProtection --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"
```

## 6. 登录安全

### 配置登录白名单

```bash
# 为合法运维 IP 添加登录白名单
# 通过告警事件处理机制添加登录白名单
hcloud HSS OperateEvent --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body='{"operate_type":"add_to_login_whitelist","event_id_list":["<eventId>"],"operate_detail_list":[{"login_ip":"10.0.0.0/8","private_ip":"","login_user_name":"root"}]}'
```

## 7. 日志与审计

关键操作需要记录审计日志：

| 操作类型 | 审计要求 |
|---------|---------|
| 告警事件处理 | 记录操作人、事件 ID、操作类型 |
| 漏洞修复 | 记录修复的漏洞 ID 和时间 |
| 防护版本变更 | 记录变更前后版本 |
| 网页防篡改配置 | 记录保护目录变更 |

确保已启用 CTS 并追踪 HSS 相关操作日志，保存周期 ≥ 180 天。

## 8. 定期安全巡检

```bash
#!/bin/bash
# HSS 安全巡检脚本

echo "=== HSS Security Audit ==="

# 1. 检查 Agent 离线主机
OFFLINE=$(hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq '[.data_list[] | select(.agent_status != "online")] | length')
echo "Offline agents: $OFFLINE"

# 2. 检查未处理告警
UNHANDLED=$(hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --handle_status="unhandled" | jq '.total_num // 0')
echo "Unhandled alerts: $UNHANDLED"

# 3. 检查严重漏洞
CRITICAL_VUL=$(hcloud HSS ListVulnerabilities --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --severity="critical" --handle_status="unhandled" | jq '.total_num // 0')
echo "Critical unpatched vulns: $CRITICAL_VUL"

# 4. 检查未受防护主机
UNPROTECTED=$(hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq '[.data_list[] | select(.protect_status == "not_protected")] | length')
echo "Unprotected hosts: $UNPROTECTED"

echo "=== Audit Complete ==="

# 如果有关键告警，发送通知
if [ "$UNHANDLED" -gt 0 ] || [ "$CRITICAL_VUL" -gt 0 ]; then
    echo "[ALERT] Security issues require immediate attention!"
fi
```
