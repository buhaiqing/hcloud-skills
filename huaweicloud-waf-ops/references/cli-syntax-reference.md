# CLI 语法参考

## 通用参数

| 参数 | 说明 | 示例 |
|------|------|------|
| `--cli-region` | 区域 | `ap-southeast-1` |
| `--project_id` | 项目 ID | 从 `hcloud configure` 获取 |
| `--enterprise_project_id` | 企业项目 ID | `0` 或具体 ID |
| `--body` | JSON 请求体 | `'{"key":"value"}'` |
| `--header` | 自定义 header | `--header="Accept-Language=zh-cn"` |

## CLI 模式选择

```bash
# 在线模式（默认）
hcloud WAF ListPolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID"

# 离线模式（无网络）
hcloud WAF ListPolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --cli-offline

# 调试模式
hcloud WAF ListPolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --debug
```

## 分页参数

所有 List 操作支持以下分页参数：

| 参数 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `--page` | integer | 页码 | `1` |
| `--pagesize` | integer | 每页条数 | `10` |

```bash
# 分页查询
hcloud WAF ListHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --page="1" --pagesize="50"

hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --page="1" --pagesize="100"
```

## 事件查询时间范围

`ListEvents` 支持按时间筛选：

| 参数 | 类型 | 说明 |
|------|------|------|
| `--recent` | boolean | 查询最近一周事件 |
| `--from` | long | 开始时间戳（毫秒） |
| `--to` | long | 结束时间戳（毫秒） |
| `--hosts` | array | 按域名筛选（逗号分隔） |
| `--attacks` | array | 按攻击类型筛选 |

```bash
# 最近一周
hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --recent="true"

# 自定义时间范围
hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --from="1717200000000" --to="1717372800000"

# 按域名和攻击类型
hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --hosts="[\"www.example.com\"]" --attacks="[\"cc\",\"sqli\"]"
```

## 命令别名

列出所有 WAF 相关命令：
```bash
hcloud WAF --help
```

查看特定命令详情：
```bash
hcloud WAF ListPolicy --help
hcloud WAF CreateHost --help
hcloud WAF CreateCcRule --help
```

## JSON 输出处理

```bash
# 直接输出 JSON
hcloud WAF ListPolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq '.items[] | {name: .name, id: .id}'

# 统计策略数量
hcloud WAF ListPolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq '.items | length'

# 获取域名列表
hcloud WAF ListHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq -r '.items[].hostname'
```

## 环境变量

```bash
# 区域（可被 --cli-region 覆盖）
export HW_CLOUD_REGION=ap-southeast-1

# 项目 ID
export HW_PROJECT_ID={{env.HW_PROJECT_ID}}

# AK/SK（hcloud configure 已配置时无需重复设置）
export HW_ACCESS_KEY={{env.HW_ACCESS_KEY}}
export HW_SECRET_KEY={{env.HW_SECRET_KEY}}

# 企业项目 ID
export HW_ENTERPRISE_PROJECT_ID=0

# CLI 输出格式
export HCLOUD_FORMAT=json  # 可选: json, table, tsv
```
