# CLI 语法参考

## 通用参数

| 参数 | 说明 | 示例 |
|------|------|------|
| `--cli-region` | 区域 | `ap-southeast-1` |
| `--project_id` | 项目 ID | 从 `hcloud configure` 获取 |
| `--enterprise_project_id` | 企业项目 ID | `all_granted_eps` 或具体 ID |
| `--body` | JSON 请求体 | `'{"key":"value"}'` |

## 企业项目支持

HSS API 广泛支持 `enterprise_project_id` 参数。使用 `all_granted_eps` 查询所有有权限的企业项目：

```bash
hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --enterprise_project_id="all_granted_eps"
```

## 分页参数

所有 List 操作支持以下分页参数：

| 参数 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `--page` | integer | 页码 | `1` |
| `--pagesize` | integer | 每页条数 | `10`（最大 1000） |

```bash
# 分页查询主机
hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --page="1" --pagesize="50"

# 分页查询告警事件
hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --page="1" --pagesize="100"
```

## 告警事件参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `--event_types` | array | 告警事件类型数组 |
| `--handle_status` | string | 处理状态：`unhandled` / `handled` |
| `--host_name` | string | 主机名筛选 |
| `--begin_time` | long | 开始时间戳（毫秒） |
| `--end_time` | long | 结束时间戳（毫秒） |
| `--category` | string | 事件分类：`host` / `container` |

```bash
# 查询未处理的恶意软件事件
hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --event_types="[\"malware\"]" --handle_status="unhandled"

# 按时间范围查询
hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --begin_time="1717200000000" --end_time="1717372800000"

# 查询特定主机的事件
hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --host_name="web-prod-01"
```

## 漏洞管理参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `--type` | string | 漏洞类型：`linux` / `windows` / `web_cms` |
| `--severity` | string | 严重级别：`critical` / `high` / `medium` / `low` |
| `--handle_status` | string | 处理状态：`unhandled` / `handled` |
| `--vul_name` | string | 漏洞名称模糊匹配 |

```bash
# 查询 Linux 严重漏洞
hcloud HSS ListVulnerabilities --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --type="linux" --severity="critical" --handle_status="unhandled"

# 查询 Web-CMS 漏洞
hcloud HSS ListVulnerabilities --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --type="web_cms"
```

## 主机管理参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `--host_name` | string | 主机名称模糊匹配 |
| `--host_id` | string | 主机 ID 精确匹配 |
| `--os_type` | string | 系统类型：`Linux` / `Windows` |
| `--host_status` | string | 主机状态 |
| `--protect_status` | string | 防护状态 |

```bash
# 按系统类型筛选
hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --os_type="Linux"

# 按防护状态筛选
hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --protect_status="not_protected"
```

## JSON 输出处理

```bash
# 列出所有未处理告警数量
hcloud HSS ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --handle_status="unhandled" | jq '.total_num'

# 列出所有主机名称
hcloud HSS ListHosts --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq -r '.data_list[] | "\(.host_name)\t\(.host_id)\t\(.protect_status)"'

# 统计各严重级别漏洞数量
hcloud HSS ListVulnerabilities --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | \
  jq '.data_list | group_by(.severity) | map({severity: .[0].severity, count: length})'
```

## 环境变量

```bash
# 区域
export HW_CLOUD_REGION=ap-southeast-1

# 项目 ID
export HW_PROJECT_ID={{env.HW_PROJECT_ID}}

# AK/SK
export HW_ACCESS_KEY={{env.HW_ACCESS_KEY}}
export HW_SECRET_KEY={{env.HW_SECRET_KEY}}

# 企业项目 ID（影响所有 HSS 查询范围）
export HW_ENTERPRISE_PROJECT_ID=all_granted_eps

# CLI 输出格式
export HCLOUD_FORMAT=json
```
