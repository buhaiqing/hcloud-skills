# 错误处理

## 通用错误处理流程

```
[ERROR] 错误码
   ├── 原因：问题根因
   ├── 排查：诊断步骤
   ├── 修复：解决方案
   └── 验证：确认修复的命令
```

## WAF 错误码

| 错误码 | 原因 | 排查 | 修复 |
|--------|------|------|------|
| `WAF.00021001` | 资源不存在 | `hcloud WAF ListPolicy/ListHost` 确认资源 ID | 检查 resource_id 参数，使用正确的 ID |
| `WAF.00021002` | 资源已存在 | 确认是否重复创建 | 使用现有资源或修改名称再创建 |
| `WAF.00021003` | 参数校验失败 | 检查 JSON 格式和必填字段 | 参照 API 文档修正请求参数 |
| `WAF.00021004` | 资源被引用 | 查找引用该资源的所有规则 | 先解除引用再操作 |
| `WAF.00021005` | 请求参数错误 | 检查字段类型和长度限制 | 修正参数类型/长度 |
| `WAF.00021006` | 请求过多/域名验证失败 | 检查请求频率或 DNS 配置 | 降低频率/修正 DNS |
| `WAF.00022001` | 证书格式错误 | 使用 `openssl` 验证证书 | 修正证书格式 |
| `WAF.00022002` | 证书和私钥不匹配 | `openssl` 检查 modulus | 更换匹配的证书/私钥 |
| `WAF.00023001` | 规则冲突 | 检查同类型规则的优先级 | 调整优先级或合并规则 |
| `WAF.00031001` | 域名已接入 | 检查域名是否已在 WAF 中 | 更新已有域名配置 |
| `WAF.00071001` | 证书被域名绑定 | 查询绑定了该证书的域名 | 先解绑或迁移证书 |

## CLI 层错误

| 错误信息 | 原因 | 修复 |
|----------|------|------|
| `Required parameter missing: project_id` | 未传递 project_id | 添加 `--project_id` 参数 |
| `Invalid JSON in body` | JSON 格式错误 | 使用 `jq .` 格式化验证 JSON |
| `Region not found` | 区域代码错误 | 检查 `--cli-region` 或用 `hcloud WAF --help` 查看可用区域 |
| `Network error` | 网络不可达 | 检查网络连接和代理设置 |

## Go SDK 错误处理

```go
import "github.com/huaweicloud/huaweicloud-sdk-go-v3/core"

// 通用错误处理
resp, err := client.ListPolicy(request)
if err != nil {
    // 类型断言获取详细错误
    if requestErr, ok := err.(*core.ServiceResponseError); ok {
        switch requestErr.StatusCode {
        case http.StatusNotFound:
            // 处理 404
        case http.StatusForbidden:
            // 处理 403 权限错误
        default:
            // 其他错误
        }
        fmt.Printf("[ERROR] %s: %s", requestErr.ErrorCode, requestErr.ErrorMsg)
    }
    return
}
```

## 诊断工作流

### 场景 A：无法列出 WAF 资源

```
[ERROR] "Failed to call WAF API"
   ├── 原因：网络不通、认证失败或权限不足
   ├── 排查：
   │   1. hcloud configure list           # 检查 AK/SK 配置
   │   2. hcloud WAF ListPolicy --debug   # 查看详细请求日志
   │   3. curl 测试 API 连通性
   ├── 修复：
   │   1. 重新配置 hcloud configure
   │   2. 检查 IAM 权限策略
   │   3. 确认 WAF 服务已开通
   └── 验证：hcloud WAF ListPolicy
```

### 场景 B：规则添加后不生效

```
[ERROR] "WAF rule configured but not blocking attacks"
   ├── 原因：策略未绑定域名、规则优先级冲突、动作模式非 block
   ├── 排查：
   │   1. hcloud WAF ShowPolicy --policy_id="<policyId>"  # 检查策略配置
   │   2. hcloud WAF ListHost | jq '.items[] | select(.policyid == "<policyId>")'  # 检查域名绑定
   │   3. hcloud WAF ListEvents --recent="true"  # 查看事件确认
   ├── 修复：
   │   1. 绑定额外域名到策略
   │   2. 调整规则优先级（值越小优先级越高）
   │   3. 确认规则 action 为 "block" 而非 "log"
   └── 验证：hcloud WAF ListEvents
```

### 场景 C：证书操作失败

```
[ERROR] "Certificate operation failed"
   ├── 原因：格式错误、已过期或被绑定
   ├── 排查：
   │   1. openssl x509 -in cert.pem -text -noout -dates  # 检查有效期
   │   2. hcloud WAF ListCertificates  # 检查已有证书
   │   3. hcloud WAF ListHost | jq '.[] | select(.certificateid == "<certId>")'  # 检查绑定域名
   ├── 修复：
   │   1. 确保证书未过期
   │   2. 上传前将证书和私钥转为 base64
   │   3. 如需删除证书，先解绑相关域名
   └── 验证：hcloud WAF ListCertificates
```
