# 常见故障处理

## 1. "HSS.00010001" — 认证失败

**错误信息**：
```
HSS.00010001 Authentication failed
```

**原因**：AK/SK 无效、过期或权限不足。

**解决方案**：
- 检查 AK/SK 是否正确配置
- 确认 IAM 用户拥有 `hss:*` 权限
- 重新配置 `hcloud configure`

## 2. "HSS.00020001" — 主机未安装 Agent

**错误信息**：
```
HSS.00020001 Agent not installed on the host
```

**原因**：目标主机未安装 HSS Agent，无法执行入侵检测、漏洞扫描等操作。

**解决方案**：
```bash
# 查询 Agent 安装状态
hcloud HSS ListAgents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"

# 未安装时，使用安装命令
# 需在目标主机上执行安装脚本
```

## 3. "HSS.00020002" — 防护版本不足

**错误信息**：
```
HSS.00020002 Insufficient protection version
```

**原因**：操作需要更高版本（如漏洞扫描需要企业版及以上）。

**解决方案**：
```bash
# 查看当前防护版本
hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq '.data_list[] | "\(.host_name): \(.version)"'

# 升级防护版本
hcloud HSS SwitchHostsProtectStatus --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body='{"version":"hss.version.enterprise","charging_mode":"on_demand","host_id_list":["<hostId>"]}'
```

## 4. "HSS.00030001" — 资源不存在

**错误信息**：
```
HSS.00030001 Resource not found
```

**原因**：查询的资源（主机/漏洞/事件）不存在或已被删除。

**解决方案**：
- 确认资源 ID 是否正确
- 使用 List 操作确认资源是否存在
- 检查 enterprise_project_id 是否正确

## 5. "HSS.00030002" — 操作参数错误

**错误信息**：
```
HSS.00030002 Invalid parameter
```

**原因**：请求参数格式错误或缺少必填字段。

**解决方案**：
- 检查 JSON 语法
- 确认事件 ID 和操作类型匹配
- 检查 operate_detail_list 必须字段是否完整

```bash
# 使用 jq 验证 JSON 格式
echo '{"operate_type":"isolate_and_kill","event_id_list":["evt-xxx"],"operate_detail_list":[{"agent_id":"agent-xxx","file_hash":"hash","file_path":"/path","process_pid":1234}]}' | jq .
```

## 6. 告警事件处理失败 — operate_detail 参数错误

**错误信息**：
```
HSS.00030004 operate_detail validation failed
```

**原因**：不同 `operate_type` 需要的 `operate_detail_list` 字段不同。

**解决方案**：
- `isolate_and_kill` → 需要 `agent_id`, `file_hash`, `file_path`, `process_pid`
- `add_to_alarm_whitelist` → 需要 `keyword`, `hash`
- `add_to_login_whitelist` → 需要 `login_ip`, `private_ip`, `login_user_name`

参考 [SKILL.md](../SKILL.md#处理告警事件) 中不同操作类型的请求示例。

## 7. 漏洞扫描任务创建失败

**错误信息**：
```
HSS.00040001 Scan task already exists
```

**原因**：已有正在执行的扫描任务，不允许重复创建。

**解决方案**：
- 等待当前扫描任务完成（一般 5-30 分钟）
- 或确认上次扫描已完成再进行新的扫描

## 8. "HSS.00050001" — 配额不足

**错误信息**：
```
HSS.00050001 Insufficient quota
```

**原因**：当前 HSS 配额不足，无法切换防护版本或添加更多主机。

**解决方案**：
```bash
# 查看配额
hcloud HSS ListQuotas --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"

# 购买更多配额（通过控制台或 API）
```

## 9. 基线检查结果未更新

**现象**：基线检查结果长时间未变化。

**原因分析**：
- 基线检查未定期执行
- 主机配置未变化

**解决方案**：
```bash
# 手动触发基线检查
hcloud HSS CreateBaselineCheckTask --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --body='{"host_id_list":["<hostId>"]}'

# 等待任务执行（约 5-10 分钟）后查询结果
hcloud HSS ListBaselineCheckResults --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --host_id="<hostId>"
```

## 10. "HSS.00060001" — API 请求频率过高

**错误信息**：
```
HSS.00060001 Too many requests
```

**原因**：触发了 HSS API 的流控限制（600 次/分钟，单个用户/IP 5 次/分钟）。

**解决方案**：
- 减少并发请求数量
- 在请求间添加延迟（建议 ≥ 200ms）
- 使用分页参数 `page`/`pagesize` 减少单次拉取数据量
