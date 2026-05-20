# 常见故障处理

## 1. "policy_id is required" — 策略 ID 为空

**错误信息**：
```
Required request parameter 'policy_id' for method parameter type String is not present
```

**原因**：WAF 规则操作需要关联到具体防护策略，但未提供 `--policy_id`。

**解决方案**：
```bash
# 先获取策略 ID
hcloud WAF ListPolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq -r '.items[].id'

# 然后在后续操作中传入
hcloud WAF ListCcRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>"
```

## 2. "resource_not_found" — 资源不存在

**错误信息**：
```
WAF.00021001 resource_not_found
```

**原因**：操作的资源（策略/域名/规则/证书）不存在或已被删除。

**解决方案**：
- 确认资源 ID 是否正确
- 使用 List 操作确认资源是否存在
- 检查是否传递了正确的 `--enterprise_project_id`

## 3. "WAF.00021005 request_param_error" — 请求参数错误

**错误信息**：
```
WAF.00021005 request_param_error
```

**原因**：请求体参数格式错误或缺少必填字段。

**解决方案**：
- 检查 JSON 语法是否正确（使用 `jq` 验证）
- 确认所有必填字段都已提供
- 检查字段类型是否正确（如 `string` 还是 `integer`）

```bash
# 验证 JSON 格式
echo '{"name":"test","limit_num":100}' | jq .
```

## 4. "Insufficient permissions" — 权限不足

**错误信息**：
```
Insufficient permissions. The request authorized by IAM might fail.
```

**原因**：当前 AK/SK 没有 WAF 相关操作权限。

**解决方案**：
- 确认 IAM 用户具备 `waf:*` 或具体操作权限
- 确认已开启 WAF 服务并完成授权
- 使用 `hcloud WAF ListPolicy` 测试基础权限

## 5. "WAF.00071001" — 证书已被域名绑定

**错误信息**：
```
WAF.00071001 Certificate is used by host
```

**原因**：尝试删除已被域名绑定的证书。

**解决方案**：
```bash
# 查找使用该证书的域名
hcloud WAF ListHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" | jq --arg cert "certId" '.items[] | select(.certificateid == $cert) | .hostname'

# 先解绑或替换证书
hcloud WAF UpdateHost --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" \
  --host_id="<hostId>" --body='{"certificateid":""}'

# 然后再删除证书
hcloud WAF DeleteCertificate --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --certificate_id="<certId>"
```

## 6. "域名接入验证失败" — DNS 解析/回源配置异常

**错误信息**：
```
WAF.00021006 Adding host failed. Please check the DNS configuration.
```

**原因**：添加云模式域名时，WAF 无法验证域名所有权或回源地址不可达。

**解决方案**：
- 确认域名已备案
- 确认 DNS 解析指向 WAF 的 CNAME
- 检查源站地址和端口是否可达
- 确认源站协议（HTTP/HTTPS）配置正确

## 7. CC 规则不生效

**现象**：配置了 CC 防护规则但攻击流量仍然通过。

**原因分析**：
- 规则优先级冲突
- URL 路径匹配模式不正确
- 阈值设置过高

**排查步骤**：
```bash
# 1. 确认规则是否存在
hcloud WAF ListCcRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>"

# 2. 检查策略模式（block 模式才生效）
hcloud WAF ShowPolicy --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>"

# 3. 查看攻击事件确认是否触发
hcloud WAF ListEvents --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --hosts="[\"www.example.com\"]" --attacks="[\"cc\"]"
```

## 8. 证书上传失败

**错误信息**：
```
WAF.00022001 certificate format error
```

**原因**：证书或私钥格式不正确。

**解决方案**：
```bash
# 验证证书格式
openssl x509 -in cert.pem -text -noout

# 验证私钥格式
openssl rsa -in key.pem -check

# 确认证书和私钥匹配
openssl x509 -noout -modulus -in cert.pem | openssl md5
openssl rsa -noout -modulus -in key.pem | openssl md5

# 如果包含换行符，转换为 base64 一行
base64 -w 0 cert.pem
base64 -w 0 key.pem
```

## 9. 引用表被规则引用无法删除

**错误信息**：
```
WAF.00021004 valuelist is used by rule
```

**原因**：引用表（Value List）正被其他规则引用。

**解决方案**：
```bash
# 先确认哪些规则使用了该引用表
hcloud WAF ListCustomRules --cli-region="$HW_CLOUD_REGION" --project_id="$HW_PROJECT_ID" --policy_id="<policyId>" | jq '.items[] | select(.conditions[].value_list_id == "<valueListId>")'

# 删除相关规则后再删除引用表
```

## 10. "WAF.00021006" — 请求达到速率限制

**错误信息**：
```
WAF.00021006 Too many requests. Please try again later.
```

**原因**：API 调用过于频繁，触发了 WAF API 的流控限制。

**解决方案**：
- 减少并发请求数
- 在请求间添加延迟（如 1 秒间隔）
- 使用 List 操作的 `page`/`pagesize` 分页控制，避免一次性拉取大量数据
