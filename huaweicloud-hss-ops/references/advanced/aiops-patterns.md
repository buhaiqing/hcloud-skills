# 智能运维（AIOps）

> **JSON paths**：内联 `jq` 调用统一引用以下路径。
> `.total_num` — 列表总数（ListEvents / ListVulnerabilities 顶层）
> `.data_list[]` — HSS 列表结果数组（host / event / vuln）
> `.data_list[].host_id|.host_name|.severity|.event_id|.src_ip|.file_path` — 主机/告警/漏洞字段
> `.data_list | length` — 数组大小 `.host_num|.risk_host_num` — 主机数 / 风险主机数

## 异常检测模式

### 模式 1：安全告警风暴检测
- **指标**：`ListEvents` 当日 vs 昨日告警数（`--begin_time/--end_time` 取 `date +%s*1000` 窗口）。
- **阈值**：变化率 **>200%** → 告警风暴；单日 `$TODAY > 50` 也触发阈值告警。
- **动作**：核查 `ListEvents` 中新攻击模式。

### 模式 2：未处理告警堆积检测
- **指标**：`ListEvents --handle_status=unhandled` 按天（近 7 天）计数。
- **阈值**：未处理告警**连续 3 天增长** → 触发堆积告警。
- **动作**：建自动处理流程或通知安全运维。

### 模式 3：漏洞修复进度监控
- **指标**：`ListVulnerabilities` 总数 / 未处理 / 关键未处理（`--severity=critical --handle_status=unhandled`）。
- **阈值**：修复率 `= (总数-未处理)/总数` **<80%** → 告警；存在关键未处理 → SLA 风险告警。

### 模式 4：主机健康度评分（满分 100）
| 维度 | 条件 | 扣分 |
|------|------|------|
| Agent 状态 | `ShowHost | .agent_status != "online"` | -20 |
| 防护状态 | `.protect_status == "not_protected"` | -20 |
| 未处理告警 | `ListEvents --host_name=… --handle_status=unhandled`：`>10` -15｜`>0` -5 |
| 未修复漏洞 | `ListVulnerabilities --host_id=… --handle_status=unhandled`：`>20` -15｜`>0` -5 |
- **评级**：≥80 Good｜≥50 Needs Attention｜<50 Critical。

## 自动修复模式

### 自动隔离恶意文件
- **触发**：同一主机**连续 3 个**相同 malware 告警（`ListEvents --event_types=["malware"] --handle_status=unhandled`）。
- **动作**：`OperateEvent --body='{"operate_type":"isolate_and_kill","event_id_list":[...],"operate_detail_list":[{"agent_id":..., "file_hash":..., "file_path":..., "process_pid":...}]}'` 隔离并杀进程。

### 自动封锁暴力破解 IP
- **触发**：同一 `src_ip` 的 `login_fail` 告警（`ListEvents --event_types=["login_fail"] --handle_status=unhandled`）**≥10 次**。
- **动作**：`ChangeBlockedIp --body='{"data_list":[{"block_ip":..., "host_id":""}], "operate_type":"block"}'`。
- **检测**：`jq '[.data_list[]|.src_ip//empty]|unique[]'` 分组统计。

## 趋势分析与报告（周报）
- **告警趋势**：遍历 `malware` `ransomware` `login_fail`，分别 `ListEvents --event_types=["..."] | jq '.total_num'`。
- **漏洞统计**：遍历 `critical` `high` `medium` `low`，分别 `ListVulnerabilities --severity=… | .total_num`。
- **Top5 主机**：`ListEvents | jq '.data_list | group_by(.host_name) | map({host:.[0].host_name, count:length}) | sort_by(.count) | reverse | .[0:5]'`。
