# 错误处理

## 通用错误处理流程

```
[ERROR] 错误码
   ├── 原因：问题根因
   ├── 排查：诊断步骤
   ├── 修复：解决方案
   └── 验证：确认修复的命令
```

## HSS 错误码

| 错误码 | 原因 | 排查 | 修复 |
|--------|------|------|------|
| `HSS.00010001` | 认证失败 | 检查 AK/SK 和 IAM 权限 | 重新配置凭证 |
| `HSS.00020001` | Agent 未安装 | `ListAgents` 检查安装状态 | 在主机上安装 Agent |
| `HSS.00020002` | 防护版本不足 | `ListHosts` 查看版本信息 | 升级到企业版/旗舰版 |
| `HSS.00020003` | Agent 离线 | 检查主机网络和 Agent 状态 | 排查 Agent 连接问题 |
| `HSS.00030001` | 资源不存在 | 使用 List 操作确认资源 | 检查资源 ID 正确性 |
| `HSS.00030002` | 参数错误 | 用 `jq` 验证 JSON 格式 | 修正请求参数 |
| `HSS.00030003` | 事件已处理 | 检查事件当前处理状态 | 确认事件尚未被处理 |
| `HSS.00030004` | operate_detail 校验失败 | 检查字段是否完整 | 按操作类型提供正确字段 |
| `HSS.00040001` | 扫描任务已存在 | 检查是否有运行中的任务 | 等待当前任务完成 |
| `HSS.00040002` | 扫描任务不存在 | 检查任务 ID 正确性 | 创建新的扫描任务 |
| `HSS.00050001` | 配额不足 | `ListQuotas` 查看可用配额 | 购买更多 HSS 配额 |
| `HSS.00060001` | 请求频率过高 | 检查 API 调用频率 | 降低并发，增加延时 |
| `HSS.00070001` | 主机组已存在 | 检查主机组名称 | 使用不同的组名 |
| `HSS.00070002` | 主机组非空 | 检查主机组中主机数量 | 先移除组内主机再删除组 |

## CLI 层错误

| 错误信息 | 原因 | 修复 |
|----------|------|------|
| `Required parameter missing: project_id` | 未传递 project_id | 添加 `--project_id` 参数 |
| `Invalid JSON in body` | JSON 格式错误 | 使用 `jq .` 验证 |
| `Region not found` | 区域代码错误 | 检查 `--cli-region` |

## Go SDK 错误处理

```go
import "github.com/huaweicloud/huaweicloud-sdk-go-v3/core"

resp, err := client.ListHosts(request)
if err != nil {
    if requestErr, ok := err.(*core.ServiceResponseError); ok {
        switch requestErr.StatusCode {
        case http.StatusUnauthorized:
            // 401 — 认证失败,检查 AK/SK
        case http.StatusForbidden:
            // 403 — 权限不足
        case http.StatusNotFound:
            // 404 — 资源不存在
        default:
            // 其他错误
        }
        fmt.Printf("[ERROR] %s: %s\n", requestErr.ErrorCode, requestErr.ErrorMsg)
    }
    return
}
```

## 诊断工作流

### 场景 A：无法查询主机列表

```
[ERROR] "Failed to list hosts"
   ├── 原因：认证失败、权限不足或项目 ID 错误
   ├── 排查：
   │   1. hcloud configure list           # 检查配置
   │   2. hcloud HSS ListHosts --debug    # 调试模式查看详细请求
   │   3. 确认 HSS 服务已开通
   ├── 修复：
   │   1. 重新配置 AK/SK
   │   2. 检查 IAM 用户权限（需要 hss:hosts:list）
   │   3. 检查项目 ID 是否正确
   └── 验证：hcloud HSS ListHosts --page="1" --pagesize="5"
```

### 场景 B：告警事件处理失败

```
[ERROR] "Failed to operate event"
   ├── 原因：事件 ID 错误、operate_detail 参数不完整、事件已被处理
   ├── 排查：
   │   1. hcloud HSS ListEvents --handle_status="unhandled"  # 确认事件未处理
   │   2. hcloud HSS ShowEvent --event_id="<eventId>"         # 查看事件详情
   │   3. 确认 operate_type 对应的 operate_detail 字段完整
   ├── 修复：
   │   1. 使用正确的事件 ID
   │   2. 根据 operate_type 补充 operate_detail 必填字段
   │   3. 事件已处理时无需重复操作
   └── 验证：hcloud HSS ListEvents --event_ids="[\"<eventId>\"]"
```

### 场景 C：漏洞扫描失败

```
[ERROR] "Vulnerability scan failed"
   ├── 原因：扫描任务已存在、Agent 离线、主机防护版本不足
   ├── 排查：
   │   1. hcloud HSS ListHosts --host_id="<hostId>"  # 检查防护版本
   │   2. hcloud HSS ListAgents                       # 检查 Agent 状态
   ├── 修复：
   │   1. 如果是企业版以下，先升级防护版本
   │   2. 等待已有扫描任务完成
   │   3. 排查 Agent 连接状态
   └── 验证：hcloud HSS ListVulnerabilities --handle_status="unhandled"
```
